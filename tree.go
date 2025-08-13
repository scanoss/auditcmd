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
	
	// Set initial selected node to first child if available
	if len(app.FileTree.Children) > 0 {
		app.TreeState.selectedNode = app.FileTree.Children[0]
		updateTreeDisplay(app)
	}
}

func updateTreeDisplay(app *AppState) {
	app.TreeState.displayLines = make([]TreeDisplayLine, 0)
	buildTreeDisplay(app.FileTree, 0, app.TreeState)
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

	// Add pending file count for directories
	displayName := node.Name
	if node.IsDir {
		pendingCount := countPendingFilesInDirectory(node.Path)
		if pendingCount > 0 {
			displayName = fmt.Sprintf("%s (%d)", node.Name, pendingCount)
		}
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

func countPendingFilesInDirectory(dirPath string) int {
	if globalApp == nil {
		return 0
	}
	
	pendingCount := 0
	
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
					// Check if file has been audited
					if len(match.AuditCmd) == 0 {
						pendingCount++
					}
					break // Only count first valid match per file
				}
			}
		}
	}
	
	return pendingCount
}