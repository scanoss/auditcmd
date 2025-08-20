// Copyright (c) 2025 SCANOSS
// SPDX-License-Identifier: MIT

package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/awesome-gocui/gocui"
)

func showExportDialog(g *gocui.Gui, app *AppState) error {
	maxX, maxY := g.Size()
	
	// Generate filename
	filename := generateDefaultCSVFilename(app.FilePath)
	
	// Check if file exists to show appropriate warning
	fileExists := false
	if _, err := os.Stat(filename); err == nil {
		fileExists = true
	}
	
	// Main dialog frame - fixed 4-line height like Accept/Ignore dialogs
	if v, err := g.SetView("export_dialog", maxX/4, maxY/3, 3*maxX/4, maxY/3+5, 0); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "EXPORT to CSV"
		v.Frame = true
		v.Editable = false
		v.TitleColor = gocui.ColorYellow
		v.BgColor = gocui.ColorBlack
		v.FgColor = gocui.ColorYellow
		
		if _, err := g.SetCurrentView("export_dialog"); err != nil {
			return err
		}
	}
	
	// Update the dialog display
	updateExportDialog(g, app, filename, fileExists)
	
	// Clear any existing keybindings first
	g.DeleteKeybindings("export_dialog")
	
	// Set up keybindings for the dialog
	g.SetKeybinding("export_dialog", gocui.KeyEnter, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		// Don't close dialog yet - we'll use it for progress updates
		g.DeleteKeybindings("export_dialog")
		
		// Start export in goroutine so GUI remains responsive
		go func() {
			performCSVExportAsync(g, app, filename)
		}()
		
		return nil
	})
	
	g.SetKeybinding("export_dialog", gocui.KeyEsc, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		return closeExportDialog(g, app)
	})
	
	return nil
}

func updateExportDialog(g *gocui.Gui, app *AppState, filename string, fileExists bool) error {
	v, err := g.View("export_dialog")
	if err != nil {
		return err
	}
	
	// Line 1: Filename, Line 2: Warning if exists, Line 3: empty, Line 4: help
	v.Clear()
	fmt.Fprintf(v, " File: %s\n", filename)
	if fileExists {
		fmt.Fprintf(v, " WARNING: File exists and will be overwritten\n")
	} else {
		fmt.Fprintf(v, " File will be created\n")
	}
	fmt.Fprintf(v, "\n")
	fmt.Fprintf(v, " ENTER: Export  ESC: Cancel")
	
	return nil
}

func closeExportDialog(g *gocui.Gui, app *AppState) error {
	g.DeleteKeybindings("export_dialog")
	g.DeleteView("export_dialog")
	
	// Restore current view
	if app.ActivePane == "tree" {
		g.SetCurrentView("tree")
	} else {
		g.SetCurrentView("files")
	}
	
	return nil
}

func generateDefaultCSVFilename(jsonPath string) string {
	// Remove extension and add .csv
	ext := filepath.Ext(jsonPath)
	base := strings.TrimSuffix(jsonPath, ext)
	return base + ".csv"
}

func performCSVExportAsync(g *gocui.Gui, app *AppState, filename string) {
	err := performCSVExport(g, app, filename)
	if err != nil {
		// Handle error in GUI thread
		g.Update(func(g *gocui.Gui) error {
			return showExportError(g, app, fmt.Sprintf("Export failed: %v", err))
		})
	}
}

func performCSVExport(g *gocui.Gui, app *AppState, filename string) error {
	// Check if file exists for the dialog
	fileExists := false
	if _, err := os.Stat(filename); err == nil {
		fileExists = true
	}
	// Create or overwrite the CSV file
	file, err := os.Create(filename)
	if err != nil {
		return showExportError(g, app, fmt.Sprintf("Failed to create file: %v", err))
	}
	defer file.Close()
	
	writer := csv.NewWriter(file)
	defer writer.Flush()
	
	// First, determine max number of line ranges across all data
	maxRanges := getMaxLineRanges(app.ScanData)
	
	// Write CSV header with dynamic deeplink columns
	header := []string{"File Path", "Match Type", "PURL", "License", "Status", "Comment", "Line Ranges"}
	// Add deeplink columns based on max ranges found
	if maxRanges > 1 {
		for i := 1; i <= maxRanges; i++ {
			header = append(header, fmt.Sprintf("Deeplink %d", i))
		}
	} else {
		header = append(header, "Deeplink")
	}
	if err := writer.Write(header); err != nil {
		return showExportError(g, app, fmt.Sprintf("Failed to write header: %v", err))
	}
	
	// Collect all files from the scan data
	allFiles := make(map[string]bool)
	for filePath := range app.ScanData.Files {
		allFiles[filePath] = true
	}
	
	// Export each file with progress tracking
	totalFiles := len(allFiles)
	processedFiles := 0
	
	for filePath := range allFiles {
		processedFiles++
		
		// Update progress in dialog
		updateExportProgress(g, processedFiles, totalFiles, filename, fileExists)
		
		// Small delay to make progress visible
		time.Sleep(10 * time.Millisecond)
		
		matches, exists := app.ScanData.Files[filePath]
		
		if !exists || len(matches) == 0 {
			// File with no matches - fill deeplink columns with empty strings
			record := []string{filePath, "no-match", "", "", "Pending", "", ""}
			for i := 0; i < maxRanges; i++ {
				record = append(record, "")
			}
			if err := writer.Write(record); err != nil {
				return showExportError(g, app, fmt.Sprintf("Failed to write record: %v", err))
			}
			continue
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
			// No valid match found - fill deeplink columns with empty strings  
			record := []string{filePath, "no-match", "", "", "Pending", "", ""}
			for i := 0; i < maxRanges; i++ {
				record = append(record, "")
			}
			if err := writer.Write(record); err != nil {
				return showExportError(g, app, fmt.Sprintf("Failed to write record: %v", err))
			}
			continue
		}
		
		// Extract license information
		licenses := make([]string, 0)
		for _, license := range match.Licenses {
			licenses = append(licenses, license.Name)
		}
		licenseStr := strings.Join(licenses, "; ")
		
		// Extract PURL information
		purlStr := ""
		if len(match.Purl) > 0 {
			purlStr = strings.Join(match.Purl, "; ")
		}
		
		// Determine status and comment
		status := "Pending"
		comment := ""
		
		if len(match.AuditCmd) > 0 {
			latest := match.AuditCmd[len(match.AuditCmd)-1]
			switch strings.ToLower(latest.Decision) {
			case "identified":
				status = "Accepted"
			case "ignored":
				status = "Ignored"
			default:
				status = "Pending"
			}
			comment = latest.Assessment
		}
		
		// Extract line ranges and generate multiple deeplinks
		lineRanges := extractLineRanges(match)
		deeplinks := generateMultipleDeeplinks(g, match, lineRanges, maxRanges)
		
		// Build record with dynamic deeplink columns
		record := []string{filePath, match.ID, purlStr, licenseStr, status, comment, lineRanges}
		record = append(record, deeplinks...)
		if err := writer.Write(record); err != nil {
			return showExportError(g, app, fmt.Sprintf("Failed to write record: %v", err))
		}
	}
	
	// Export completed successfully - close dialog and return to main interface
	g.Update(func(g *gocui.Gui) error {
		g.DeleteView("export_dialog")
		if app.ActivePane == "tree" {
			g.SetCurrentView("tree")
		} else {
			g.SetCurrentView("files")
		}
		return nil
	})
	
	return nil
}

// extractLineRanges extracts line ranges from oss_lines field
func extractLineRanges(match *FileMatch) string {
	if match.OSSLines == nil {
		return ""
	}
	
	// Convert interface{} to string
	switch v := match.OSSLines.(type) {
	case string:
		if v == "all" {
			return "all"
		}
		return v
	default:
		return ""
	}
}

// generateDeeplink creates a GitHub deeplink from PURL information
func generateDeeplink(g *gocui.Gui, match *FileMatch, lineRanges string) string {
	if len(match.Purl) == 0 {
		return ""
	}
	
	// Look for pkg:github PURL
	for _, purl := range match.Purl {
		if strings.HasPrefix(purl, "pkg:github/") {
			// Use match.File instead of scanned file path - this is the path in the matched repo
			return generateGitHubDeeplink(g, purl, match.File, match.ID, lineRanges)
		}
	}
	
	return ""
}

// generateGitHubDeeplink creates GitHub URL with optional line highlighting
func generateGitHubDeeplink(g *gocui.Gui, purl, filePath, matchType, lineRanges string) string {
	// Parse PURL: pkg:github/owner/repo[@commit]
	// First try with commit hash
	re := regexp.MustCompile(`pkg:github/([^/]+)/([^@?]+)@([^?]+)`)
	matches := re.FindStringSubmatch(purl)
	
	var owner, repo, commit string
	if len(matches) == 4 {
		// PURL with commit hash
		owner = matches[1]
		repo = matches[2]
		commit = matches[3]
	} else {
		// Try without commit hash
		re = regexp.MustCompile(`pkg:github/([^/]+)/([^?]+)`)
		matches = re.FindStringSubmatch(purl)
		if len(matches) != 3 {
			return ""
		}
		owner = matches[1]
		repo = matches[2]
		commit = getDefaultBranch(g, owner, repo) // Get actual default branch
	}
	
	baseURL := fmt.Sprintf("https://github.com/%s/%s/blob/%s/%s", owner, repo, commit, filePath)
	
	// For snippet matches, add line highlighting if available
	if matchType == "snippet" {
		// Only add line highlighting if we have specific line ranges
		if lineRanges != "" && lineRanges != "all" {
			// Parse line ranges like "11-14,20-25" and use first range
			ranges := strings.Split(lineRanges, ",")
			if len(ranges) > 0 {
				firstRange := strings.TrimSpace(ranges[0])
				if firstRange != "" {
					// Convert "11-14" to "L11-L14" format
					if strings.Contains(firstRange, "-") {
						parts := strings.Split(firstRange, "-")
						if len(parts) == 2 {
							startLine := strings.TrimSpace(parts[0])
							endLine := strings.TrimSpace(parts[1])
							baseURL += fmt.Sprintf("#L%s-L%s", startLine, endLine)
						}
					} else {
						// Single line
						baseURL += "#L" + firstRange
					}
				}
			}
		}
		// If no specific line ranges, snippet still gets the base URL without highlighting
	}
	
	return baseURL
}

// gitHubRepoInfo represents the GitHub API response for repository info
type gitHubRepoInfo struct {
	DefaultBranch string `json:"default_branch"`
}

// Cache for default branches to avoid repeated API calls
var defaultBranchCache = make(map[string]string)

// getDefaultBranch fetches the default branch name for a GitHub repository
func getDefaultBranch(g *gocui.Gui, owner, repo string) string {
	repoKey := fmt.Sprintf("%s/%s", owner, repo)
	
	// Check cache first
	if branch, exists := defaultBranchCache[repoKey]; exists {
		return branch
	}
	
	
	// Update export dialog to show progress instead of separate modal
	updateExportProgressDialog(g, repoKey)
	defer updateExportProgressDialog(g, "") // Clear progress message
	
	// Small delay to make the branch checking message visible
	time.Sleep(50 * time.Millisecond)
	
	// Try to get default branch from GitHub API with short timeout
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s", owner, repo)
	
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		// Fallback: try master first (older repos), GitHub redirects to main if needed
		defaultBranchCache[repoKey] = "master"
		return "master"
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != 200 {
		// Fallback for private repos or API limits: try master first  
		defaultBranchCache[repoKey] = "master"
		return "master"
	}
	
	var repoInfo gitHubRepoInfo
	if err := json.NewDecoder(resp.Body).Decode(&repoInfo); err != nil {
		defaultBranchCache[repoKey] = "master"
		return "master"
	}
	
	if repoInfo.DefaultBranch == "" {
		defaultBranchCache[repoKey] = "master"
		return "master"
	}
	
	// Cache the result
	defaultBranchCache[repoKey] = repoInfo.DefaultBranch
	return repoInfo.DefaultBranch
}

// getMaxLineRanges determines the maximum number of line ranges in any match
func getMaxLineRanges(scanData ScanResult) int {
	maxRanges := 1 // At least one deeplink column
	
	for _, matches := range scanData.Files {
		for _, match := range matches {
			if match.ID == "snippet" {
				lineRanges := extractLineRanges(&match)
				if lineRanges != "" && lineRanges != "all" {
					ranges := strings.Split(lineRanges, ",")
					if len(ranges) > maxRanges {
						maxRanges = len(ranges)
					}
				}
			}
		}
	}
	
	return maxRanges
}

// generateMultipleDeeplinks creates multiple deeplinks for multiple line ranges
func generateMultipleDeeplinks(g *gocui.Gui, match *FileMatch, lineRanges string, maxRanges int) []string {
	deeplinks := make([]string, maxRanges)
	
	if len(match.Purl) == 0 {
		return deeplinks // All empty strings
	}
	
	// Look for pkg:github PURL
	var githubPurl string
	for _, purl := range match.Purl {
		if strings.HasPrefix(purl, "pkg:github/") {
			githubPurl = purl
			break
		}
	}
	
	if githubPurl == "" {
		return deeplinks // All empty strings
	}
	
	// Parse individual ranges and create deeplinks
	if match.ID == "snippet" && lineRanges != "" && lineRanges != "all" {
		ranges := strings.Split(lineRanges, ",")
		for i, rangeStr := range ranges {
			if i >= maxRanges {
				break
			}
			deeplinks[i] = generateGitHubDeeplinkWithRange(g, githubPurl, match.File, strings.TrimSpace(rangeStr))
		}
	} else {
		// Single deeplink for file matches or snippet without ranges
		deeplinks[0] = generateGitHubDeeplinkWithRange(g, githubPurl, match.File, "")
	}
	
	return deeplinks
}

// generateGitHubDeeplinkWithRange creates GitHub URL with specific line range
func generateGitHubDeeplinkWithRange(g *gocui.Gui, purl, filePath, lineRange string) string {
	// Parse PURL: pkg:github/owner/repo[@commit]
	// First try with commit hash
	re := regexp.MustCompile(`pkg:github/([^/]+)/([^@?]+)@([^?]+)`)
	matches := re.FindStringSubmatch(purl)
	
	var owner, repo, commit string
	if len(matches) == 4 {
		// PURL with commit hash
		owner = matches[1]
		repo = matches[2]
		commit = matches[3]
	} else {
		// Try without commit hash
		re = regexp.MustCompile(`pkg:github/([^/]+)/([^?]+)`)
		matches = re.FindStringSubmatch(purl)
		if len(matches) != 3 {
			return ""
		}
		owner = matches[1]
		repo = matches[2]
		commit = getDefaultBranch(g, owner, repo) // Get actual default branch
	}
	
	baseURL := fmt.Sprintf("https://github.com/%s/%s/blob/%s/%s", owner, repo, commit, filePath)
	
	// Add line highlighting for specific range
	if lineRange != "" {
		// Convert "11-14" to "L11-L14" format
		if strings.Contains(lineRange, "-") {
			parts := strings.Split(lineRange, "-")
			if len(parts) == 2 {
				startLine := strings.TrimSpace(parts[0])
				endLine := strings.TrimSpace(parts[1])
				baseURL += fmt.Sprintf("#L%s-L%s", startLine, endLine)
			}
		} else {
			// Single line
			baseURL += "#L" + lineRange
		}
	}
	
	return baseURL
}

// updateExportProgress shows overall export progress in status line only
func updateExportProgress(g *gocui.Gui, processed, total int, filename string, fileExists bool) {
	g.Update(func(g *gocui.Gui) error {
		updateExportStatusLine(g, fmt.Sprintf("Processing file %d of %d...", processed, total), filename, fileExists)
		return nil
	})
}

// updateExportProgressDialog updates the status line with branch checking info
func updateExportProgressDialog(g *gocui.Gui, repoKey string) {
	if repoKey != "" {
		g.Update(func(g *gocui.Gui) error {
			// We don't have filename context here, so we'll use a simpler approach
			v, err := g.View("export_dialog")
			if err != nil {
				return nil
			}
			
			// Get current content lines to preserve filename info
			lines := strings.Split(v.ViewBuffer(), "\n")
			v.Clear()
			
			// Preserve first 3 lines (filename, warning/status, blank line)
			if len(lines) >= 3 {
				fmt.Fprintf(v, "%s\n", lines[0])
				fmt.Fprintf(v, "%s\n", lines[1]) 
				fmt.Fprintf(v, "\n")
			}
			fmt.Fprintf(v, " Checking default branch for %s...", repoKey)
			return nil
		})
	}
}

// updateExportStatusLine updates just the last line of the export dialog
func updateExportStatusLine(g *gocui.Gui, statusMessage, filename string, fileExists bool) {
	v, err := g.View("export_dialog")
	if err != nil {
		return
	}
	
	v.Clear()
	
	// Reconstruct the dialog with original content but new status line
	fmt.Fprintf(v, " File: %s\n", filename)
	if fileExists {
		fmt.Fprintf(v, " WARNING: File exists and will be overwritten\n")
	} else {
		fmt.Fprintf(v, " File will be created\n")
	}
	fmt.Fprintf(v, "\n")
	fmt.Fprintf(v, " %s", statusMessage)
}

func showExportError(g *gocui.Gui, app *AppState, message string) error {
	maxX, maxY := g.Size()
	if v, err := g.SetView("export_error", maxX/4, maxY/3, 3*maxX/4, maxY/3+4, 0); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "Export Error"
		v.Frame = true
		fmt.Fprintf(v, "%s\nPress ESC to close.", message)
		
		g.SetKeybinding("export_error", gocui.KeyEsc, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
			g.DeleteKeybindings("export_error")
			g.DeleteView("export_error")
			if app.ActivePane == "tree" {
				g.SetCurrentView("tree")
			} else {
				g.SetCurrentView("files")
			}
			return nil
		})
		
		if _, err := g.SetCurrentView("export_error"); err != nil {
			return err
		}
	}
	
	return nil
}

