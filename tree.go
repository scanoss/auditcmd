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
		// In directory mode, intelligently select initial directory
		if len(app.FileTree.Children) > 0 {
			selectedNode := app.FileTree.Children[0] // Default to first child

			// If we're in "matched" or "pending" mode, try to find a directory with matching files
			if app.ViewFilter == "matched" || app.ViewFilter == "pending" {
				for _, child := range app.FileTree.Children {
					if child.IsDir {
						fileCount := countFilesInDirectory(child.Path)
						if fileCount > 0 {
							selectedNode = child
							break
						}
					}
				}
			}

			app.TreeState.selectedNode = selectedNode
		} else {
			app.TreeState.selectedNode = app.FileTree
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

	// Update custom scrollable list with display lines
	treeItems := make([]string, 0, len(app.TreeState.displayLines))
	for _, line := range app.TreeState.displayLines {
		treeItems = append(treeItems, line.Line)
	}
	app.TreeList.SetItems(treeItems)

	// Find current selection index in display lines
	currentIndex := -1
	for i, line := range app.TreeState.displayLines {
		if line.Node == app.TreeState.selectedNode {
			currentIndex = i
			break
		}
	}
	if currentIndex >= 0 {
		app.TreeList.SelectedIndex = currentIndex
		app.TreeList.adjustScroll()
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

	// Skip directories with zero files based on view filter
	if node.IsDir && globalApp != nil && fileCount == 0 {
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
					isProcessed := len(match.AuditCmd) > 0

					switch app.ViewFilter {
					case "matched":
						// Count all files with valid matches
						count++
					case "pending":
						// Count only unprocessed files
						if !isProcessed {
							count++
						}
					case "all":
						// Count all files with valid matches
						count++
					default:
						count++
					}
					break // Only count first valid match per file
				}
			}
		}

		// Skip PURLs with zero files based on view filter
		if count == 0 {
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

	// Use custom scrollable list for rendering
	isActive := (app.ActivePane == "tree")
	app.TreeList.Render(v, isActive)

	return nil
}

func navigateTree(g *gocui.Gui, app *AppState, direction string) error {
	if len(app.TreeState.displayLines) == 0 {
		return nil
	}

	// Use custom scrollable list for navigation
	app.TreeList.Navigate(direction)

	// Update selected node based on new index
	newIndex := app.TreeList.GetSelectedIndex()
	if newIndex >= 0 && newIndex < len(app.TreeState.displayLines) {
		app.TreeState.selectedNode = app.TreeState.displayLines[newIndex].Node
	}

	// Re-render the tree
	if v, err := g.View("tree"); err == nil {
		isActive := (app.ActivePane == "tree")
		app.TreeList.Render(v, isActive)
	}

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
			// Root directory - only count files with no "/" (actual root files)
			isInDirectory = !strings.Contains(filePath, "/")
		} else {
			// Check if file path starts with directory path
			isInDirectory = strings.HasPrefix(filePath, dirPath+"/")
		}

		if isInDirectory {
			if globalApp.ViewFilter == "all" {
				// For "all" view, count all files in directory (not just matched ones)
				count++
			} else {
				// For other views, only count files with valid matches
				for _, match := range matches {
					if match.ID == "file" || match.ID == "snippet" {
						isProcessed := len(match.AuditCmd) > 0

						switch globalApp.ViewFilter {
						case "matched":
							// Count all files with valid matches
							count++
						case "pending":
							// Count only unprocessed files
							if !isProcessed {
								count++
							}
						default:
							count++
						}
						break // Only count first valid match per file
					}
				}
			}
		}
	}

	return count
}
