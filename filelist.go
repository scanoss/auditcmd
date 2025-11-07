// Copyright (c) 2025 SCANOSS
// SPDX-License-Identifier: MIT

package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/awesome-gocui/gocui"
)

func updateFileList(g *gocui.Gui, app *AppState) error {
	v, err := g.View("files")
	if err != nil {
		return err
	}

	// Check if we're in content view mode - don't clear or update
	if app.ViewMode == "content" {
		return nil // Content is managed by displayFileContent
	}

	if app.TreeState == nil || app.TreeState.selectedNode == nil {
		return nil
	}

	node := app.TreeState.selectedNode
	var files []string

	if app.TreeViewType == "purls" {
		// In PURL mode, show files from the selected PURL's file list
		if len(node.Files) > 0 {
			files = node.Files
		}
	} else {
		// In directory mode, show files in the selected directory
		if !node.IsDir {
			return nil
		}
		files = getFilesInDirectory(app, node.Path)
	}
	
	// Filter and format files with status indicators
	displayFiles := make([]string, 0)
	filteredFiles := make([]string, 0) // Track filtered file paths for selection

	for _, filePath := range files {
		matches := app.ScanData.Files[filePath]

		// Apply view filter
		shouldShow := false
		statusIcon := "- "

		if app.ViewFilter == "all" {
			shouldShow = true
		}

		if len(matches) > 0 {
			// Find the first valid match (file or snippet)
			var match *FileMatch
			for j, m := range matches {
				if m.ID == "file" || m.ID == "snippet" {
					match = &matches[j]
					break
				}
			}

			if match != nil {
				if app.ViewFilter == "matched" || app.ViewFilter == "all" {
					shouldShow = true
				}

				// Check if file has been processed
				isProcessed := len(match.AuditCmd) > 0
				if isProcessed {
					latest := match.AuditCmd[len(match.AuditCmd)-1]
					decision := strings.ToLower(strings.TrimSpace(latest.Decision))
					if decision == "identified" {
						statusIcon = "✓ "
					} else {
						statusIcon = "✗ "
					}
				} else {
					statusIcon = "? "
					if app.ViewFilter == "pending" {
						shouldShow = true
					}
				}
			}
		}

		if shouldShow {
			// Apply path highlighting if there are matches
			highlightedPath := filePath
			if len(matches) > 0 {
				highlightedPath = highlightMatchingPath(filePath, matches)
			}
			displayFiles = append(displayFiles, statusIcon+highlightedPath)
			filteredFiles = append(filteredFiles, filePath) // Keep track of filtered file paths
		}
	}

	// Update our custom scrollable list
	app.FileList.SetItems(displayFiles)
	app.CurrentFileList = filteredFiles // Keep filtered file paths for selection

	// Sync the selected index after updating the list
	app.SelectedFileIndex = app.FileList.GetSelectedIndex()

	// Render the custom list
	isActive := (app.ActivePane == "files")
	app.FileList.Render(v, isActive)
	
	return nil
}

func getFilesInDirectory(app *AppState, dirPath string) []string {
	files := make([]string, 0)
	
	// If dirPath is empty (root), show all files
	// Otherwise, show files that are in this directory or subdirectories
	for filePath, matches := range app.ScanData.Files {
		
		if app.ViewFilter == "all" {
			// In "all" mode, include all files regardless of match status
		} else {
			// Filter by match type - only show files with id = "file" or "snippet"
			hasValidMatch := false
			for _, match := range matches {
				if match.ID == "file" || match.ID == "snippet" {
					hasValidMatch = true
					break
				}
			}
			
			if !hasValidMatch {
				continue
			}
		}
		
		// Check if file is in the selected directory or its subdirectories
		if dirPath == "" {
			// Root directory - only show files with no "/" (actual root files)
			if !strings.Contains(filePath, "/") {
				files = append(files, filePath)
			}
		} else {
			// Check if file is in this directory or subdirectories
			if strings.HasPrefix(filePath, dirPath+"/") {
				files = append(files, filePath)
			}
		}
	}
	
	// Sort files by path
	sort.Strings(files)
	return files
}


func displayFileContent(g *gocui.Gui, app *AppState, filePath string) error {
	v, err := g.View("files")
	if err != nil {
		return err
	}

	v.Clear()
	// Reset scroll position to top when opening new file
	v.SetOrigin(0, 0)
	// Title will be set by updatePaneTitles

	matches, exists := app.ScanData.Files[filePath]
	if !exists || len(matches) == 0 {
		fmt.Fprintf(v, "No match data found for this file")
		return nil
	}

	// Find the first valid match (file or snippet)
	var match *FileMatch
	for i, m := range matches {
		if m.ID == "file" || m.ID == "snippet" {
			match = &matches[i]
			break
		}
	}
	
	if match == nil {
		fmt.Fprintf(v, "No valid matches found for this file")
		return nil
	}

	app.CurrentMatch = match

	// Check if file_url is empty or only whitespace
	if strings.TrimSpace(match.FileURL) == "" {
		fmt.Fprintf(v, "No file_url available for this file. This requires scanning with an API key.")
		return nil
	}

	if app.APIKey == "" {
			fmt.Fprintf(v, "File Content Not Available\n")
			fmt.Fprintf(v, "========================\n\n")
			fmt.Fprintf(v, "API key required to fetch file contents from:\n")
			fmt.Fprintf(v, "%s\n\n", match.FileURL)
			fmt.Fprintf(v, "To view file contents:\n")
			fmt.Fprintf(v, "1. Exit the application\n")
			fmt.Fprintf(v, "2. Run: ./auditcmd --reset-api-key\n")
			fmt.Fprintf(v, "3. Restart and provide your API key\n\n")
			fmt.Fprintf(v, "You can still navigate, review, and audit files\n")
			fmt.Fprintf(v, "based on the metadata shown in the status panel.")
		} else {
			content, err := fetchFileContent(match.FileURL, app.APIKey)
			if err != nil {
				fmt.Fprintf(v, "Error fetching file content: %v\n\n", err)
				fmt.Fprintf(v, "This may indicate:\n")
				fmt.Fprintf(v, "• Invalid API key\n")
				fmt.Fprintf(v, "• Network connectivity issues\n")
				fmt.Fprintf(v, "• API service unavailable\n\n")
				fmt.Fprintf(v, "Try running: ./auditcmd --reset-api-key")
				return nil
			}

			lines := strings.Split(content, "\n")
			highlightLines := parseOSSLines(match.OSSLines)

			// Display all content at once and let gocui handle scrolling
			for i, line := range lines {
				lineNum := i + 1
				
				// Highlight logic based on match type
				shouldHighlight := false
				if match.ID == "file" {
					// For "file" type, highlight the entire file
					shouldHighlight = true
				} else if match.ID == "snippet" && highlightLines != nil {
					// For "snippet" type, only highlight specific lines
					shouldHighlight = contains(highlightLines, lineNum)
				}
				
				if shouldHighlight {
					fmt.Fprintf(v, "\033[43m\033[30m%4d: %s\033[0m\n", lineNum, line)
				} else {
					fmt.Fprintf(v, "%4d: %s\n", lineNum, line)
				}
			}
		}

	return nil
}

func fetchFileContent(url string, apiKey string) (string, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}
	
	// Add required headers as per curl example
	req.Header.Set("X-API-Key", apiKey)
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()
	
	// Read response body
	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %v", err)
	}
	
	// Check for API errors
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API error %d: %s", resp.StatusCode, string(content))
	}

	return string(content), nil
}

func parseOSSLines(ossLines interface{}) []int {
	if ossLines == nil {
		return nil
	}

	switch v := ossLines.(type) {
	case string:
		if v == "all" {
			return nil
		}
		
		if strings.Contains(v, "-") {
			parts := strings.Split(v, "-")
			if len(parts) == 2 {
				start, err1 := strconv.Atoi(parts[0])
				end, err2 := strconv.Atoi(parts[1])
				if err1 == nil && err2 == nil {
					lines := make([]int, 0)
					for i := start; i <= end; i++ {
						lines = append(lines, i)
					}
					return lines
				}
			}
		}
		
		if num, err := strconv.Atoi(v); err == nil {
			return []int{num}
		}
	}

	return nil
}

// highlightMatchingPath highlights the parts of filePath that match with the matched file path
func highlightMatchingPath(filePath string, matches []FileMatch) string {
	if len(matches) == 0 {
		return filePath
	}

	// Find the first valid match (file or snippet)
	var matchedPath string
	for _, match := range matches {
		if match.ID == "file" || match.ID == "snippet" {
			matchedPath = match.File
			break
		}
	}

	if matchedPath == "" {
		return filePath
	}

	// Find the longest common suffix between filePath and matchedPath
	commonSuffix := findCommonSuffix(filePath, matchedPath)
	if commonSuffix == "" {
		return filePath
	}

	// Find where the common suffix starts in filePath
	suffixStart := len(filePath) - len(commonSuffix)
	if suffixStart <= 0 {
		// The entire path matches, highlight everything
		return "\033[43m\033[30m" + filePath + "\033[0m"
	}

	// Split the path into non-matching and matching parts
	prefix := filePath[:suffixStart]
	suffix := filePath[suffixStart:]

	// Return with highlighting on the matching suffix
	return prefix + "\033[43m\033[30m" + suffix + "\033[0m"
}

// findCommonSuffix finds the longest common suffix between two paths
func findCommonSuffix(path1, path2 string) string {
	// Split paths into components
	parts1 := strings.Split(path1, "/")
	parts2 := strings.Split(path2, "/")

	// Find common suffix components
	i := len(parts1) - 1
	j := len(parts2) - 1
	var commonParts []string

	for i >= 0 && j >= 0 && parts1[i] == parts2[j] {
		commonParts = append([]string{parts1[i]}, commonParts...)
		i--
		j--
	}

	if len(commonParts) == 0 {
		return ""
	}

	return strings.Join(commonParts, "/")
}

func contains(slice []int, item int) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func navigateFileList(g *gocui.Gui, app *AppState, direction string) error {
	// Use our custom scrollable list for navigation
	app.FileList.Navigate(direction)
	
	// Update selected file index to match
	app.SelectedFileIndex = app.FileList.GetSelectedIndex()
	
	// Re-render the list
	if v, err := g.View("files"); err == nil {
		isActive := (app.ActivePane == "files")
		app.FileList.Render(v, isActive)
	}
	
	return nil
}

func navigateFileListPage(g *gocui.Gui, app *AppState, direction string) error {
	// Use our custom scrollable list for page navigation
	app.FileList.NavigatePage(direction)
	
	// Update selected file index to match
	app.SelectedFileIndex = app.FileList.GetSelectedIndex()
	
	// Re-render the list
	if v, err := g.View("files"); err == nil {
		isActive := (app.ActivePane == "files")
		app.FileList.Render(v, isActive)
	}
	
	return nil
}

func selectFile(g *gocui.Gui, app *AppState) error {
	if len(app.CurrentFileList) == 0 || app.SelectedFileIndex < 0 || app.SelectedFileIndex >= len(app.CurrentFileList) {
		return nil
	}

	selectedFile := app.CurrentFileList[app.SelectedFileIndex]
	app.CurrentFile = selectedFile
	
	return displayFileContent(g, app, selectedFile)
}

func scrollFileContent(g *gocui.Gui, app *AppState, direction string, pageMode bool) error {
	if app.ViewMode != "content" || app.CurrentFile == "" {
		return nil
	}

	v, err := g.View("files")
	if err != nil {
		return err
	}

	// Use gocui's built-in scrolling
	ox, oy := v.Origin()
	_, viewHeight := v.Size()
	if viewHeight <= 0 {
		viewHeight = 20
	}

	scrollAmount := 1
	if pageMode {
		scrollAmount = viewHeight - 2 // Leave some overlap for page scrolling
	}

	switch direction {
	case "up":
		newY := oy - scrollAmount
		if newY < 0 {
			newY = 0
		}
		v.SetOrigin(ox, newY)
	case "down":
		newY := oy + scrollAmount
		v.SetOrigin(ox, newY)
	}

	return nil
}