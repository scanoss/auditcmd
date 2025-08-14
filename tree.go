// Copyright (c) 2025 SCANOSS
// SPDX-License-Identifier: MIT

package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/awesome-gocui/gocui"
)

func initTreeState(app *AppState) {
	app.TreeState = &TreeState{
		selectedNode: app.FileTree,
		expandedDirs: make(map[string]bool),
		displayLines: make([]TreeDisplayLine, 0),
	}
	app.TreeState.expandedDirs[""] = true
	updateTreeDisplay(app)
	
	// Set initial selected node
	if app.TreeViewType == "purls" {
		// In PURL mode, select first PURL if available
		if len(app.PURLRanking) > 0 {
			app.TreeState.selectedNode = &TreeNode{
				Name:  app.PURLRanking[0].PURL,
				Path:  "purl_0",
				IsDir: false,
				Files: app.PURLRanking[0].Files,
			}
		}
	} else {
		// In directory mode, select first child if available
		if len(app.FileTree.Children) > 0 {
			app.TreeState.selectedNode = app.FileTree.Children[0]
		}
	}
	
	updateTreeDisplay(app)
}

func updateTreeDisplay(app *AppState) {
	app.TreeState.displayLines = make([]TreeDisplayLine, 0)
	
	if app.TreeViewType == "purls" {
		buildPURLDisplay(app)
	} else {
		buildTreeDisplay(app.FileTree, 0, app.TreeState)
	}
}

func buildTreeDisplay(node *TreeNode, indent int, state *TreeState) {
	if node.Name == "Root" {
		for _, child := range node.Children {
			buildTreeDisplay(child, indent, state)
		}
		return
	}

	prefix := strings.Repeat("  ", indent)
	symbol := ""
	
	if node.IsDir {
		if state.expandedDirs[node.Path] {
			symbol = "[-] "
		} else {
			symbol = "[+] "
		}
	} else {
		symbol = "    "
	}

	// Add file count for directories based on audited filter setting
	displayName := node.Name
	fileCount := 0
	if node.IsDir {
		fileCount = countFilesInDirectory(node.Path)
		if fileCount > 0 {
			displayName = fmt.Sprintf("%s (%d)", node.Name, fileCount)
		}
	}

	// Skip directories with zero files when hiding audited files
	if node.IsDir && globalApp != nil && globalApp.HideIdentified && fileCount == 0 {
		return
	}

	line := fmt.Sprintf("%s%s%s", prefix, symbol, displayName)
	state.displayLines = append(state.displayLines, TreeDisplayLine{
		Node:   node,
		Indent: indent,
		Line:   line,
	})

	if node.IsDir && state.expandedDirs[node.Path] {
		sortedChildren := make([]*TreeNode, len(node.Children))
		copy(sortedChildren, node.Children)
		sort.Slice(sortedChildren, func(i, j int) bool {
			if sortedChildren[i].IsDir != sortedChildren[j].IsDir {
				return sortedChildren[i].IsDir
			}
			return sortedChildren[i].Name < sortedChildren[j].Name
		})

		for _, child := range sortedChildren {
			buildTreeDisplay(child, indent+1, state)
		}
	}
}

func buildPURLDisplay(app *AppState) {
	for i, purlEntry := range app.PURLRanking {
		// Calculate count based on HideIdentified setting
		count := 0
		for _, filePath := range purlEntry.Files {
			matches, exists := app.ScanData.Files[filePath]
			if !exists {
				continue
			}
			
			// Find the first valid match (file or snippet)
			for _, match := range matches {
				if match.ID == "file" || match.ID == "snippet" {
					if app.HideIdentified {
						// Count only pending files when audited files are hidden
						if len(match.AuditCmd) == 0 {
							count++
						}
					} else {
						// Count all files when audited files are shown
						count++
					}
					break // Only count first valid match per file
				}
			}
		}
		
		// Skip PURLs with zero files when hiding audited files
		if app.HideIdentified && count == 0 {
			continue
		}
		
		displayName := fmt.Sprintf("%s (%d)", purlEntry.PURL, count)
		
		// Create a fake TreeNode for PURL entries
		purlNode := &TreeNode{
			Name:  purlEntry.PURL,
			Path:  fmt.Sprintf("purl_%d", i),
			IsDir: false,
			Files: purlEntry.Files,
		}
		
		line := fmt.Sprintf("    %s", displayName)
		app.TreeState.displayLines = append(app.TreeState.displayLines, TreeDisplayLine{
			Node:   purlNode,
			Indent: 0,
			Line:   line,
		})
	}
}

func displayTree(g *gocui.Gui, app *AppState) error {
	v, err := g.View("tree")
	if err != nil {
		return err
	}

	v.Clear()
	
	for i, line := range app.TreeState.displayLines {
		fmt.Fprintf(v, "%s\n", line.Line)
		
		if i > 100 {
			break
		}
	}

	return nil
}

func navigateTree(g *gocui.Gui, app *AppState, direction string) error {
	if len(app.TreeState.displayLines) == 0 {
		return nil
	}

	currentIndex := -1
	for i, line := range app.TreeState.displayLines {
		if line.Node == app.TreeState.selectedNode {
			currentIndex = i
			break
		}
	}

	switch direction {
	case "up":
		if currentIndex > 0 {
			app.TreeState.selectedNode = app.TreeState.displayLines[currentIndex-1].Node
			currentIndex--
		}
	case "down":
		if currentIndex < len(app.TreeState.displayLines)-1 {
			app.TreeState.selectedNode = app.TreeState.displayLines[currentIndex+1].Node
			currentIndex++
		}
	}

	displayTree(g, app)
	updateFileList(g, app)
	updateStatus(g, app)
	
	return nil
}

func toggleTreeNode(g *gocui.Gui, app *AppState) error {
	if app.TreeState.selectedNode == nil || !app.TreeState.selectedNode.IsDir {
		return nil
	}

	path := app.TreeState.selectedNode.Path
	app.TreeState.expandedDirs[path] = !app.TreeState.expandedDirs[path]
	
	updateTreeDisplay(app)
	displayTree(g, app)
	updateFileList(g, app)
	
	return nil
}

// Access to app state for counting pending files
var globalApp *AppState

func setGlobalApp(app *AppState) {
	globalApp = app
}

func countFilesInDirectory(dirPath string) int {
	if globalApp == nil {
		return 0
	}
	
	count := 0
	
	for filePath, matches := range globalApp.ScanData.Files {
		// Check if file is in this directory or subdirectories
		isInDirectory := false
		if dirPath == "" {
			// Root directory - count all files
			isInDirectory = true
		} else {
			// Check if file path starts with directory path
			isInDirectory = strings.HasPrefix(filePath, dirPath+"/")
		}
		
		if isInDirectory {
			for _, match := range matches {
				// Only count files with id = "file" or "snippet"
				if match.ID == "file" || match.ID == "snippet" {
					if globalApp.HideIdentified {
						// Show pending count when audited files are hidden
						if len(match.AuditCmd) == 0 {
							count++
						}
					} else {
						// Show total matches count when audited files are shown
						count++
					}
					break // Only count first valid match per file
				}
			}
		}
	}
	
	return count
}