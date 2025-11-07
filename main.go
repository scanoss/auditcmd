// Copyright (c) 2025 SCANOSS
// SPDX-License-Identifier: MIT

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/awesome-gocui/gocui"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <scanoss-result.json>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "       %s --reset-api-key   (reset stored API key)\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "       %s --api-key-status  (check API key status)\n", os.Args[0])
		os.Exit(1)
	}

	// Handle special commands
	if os.Args[1] == "--reset-api-key" {
		configPath := getConfigFilePath()
		// Load existing config to preserve other settings
		config, _ := loadConfig()
		config.APIKey = "" // Clear only the API key
		if err := saveConfig(config); err != nil {
			fmt.Printf("Error updating config file: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("API key removed from %s\n", configPath)
		fmt.Println("You will be prompted for a new API key on next run.")
		os.Exit(0)
	}

	if os.Args[1] == "--api-key-status" {
		configPath := getConfigFilePath()
		apiKey, err := loadAPIKey()
		if err != nil {
			fmt.Printf("API Key Status: Not configured\n")
			fmt.Printf("Config file: %s (not found)\n", configPath)
			fmt.Println("Run the application to be prompted for an API key.")
		} else {
			fmt.Printf("API Key Status: Configured\n")
			fmt.Printf("Config file: %s\n", configPath)
			fmt.Printf("API key: %s...%s (%d characters)\n", 
				apiKey[:min(4, len(apiKey))], 
				apiKey[max(0, len(apiKey)-4):], 
				len(apiKey))
		}
		os.Exit(0)
	}

	app := &AppState{
		ActivePane:        "tree",
		FilePath:          os.Args[1],
		CurrentFileList:   make([]string, 0),
		SelectedFileIndex: 0,
		PaneWidth:         loadPaneWidth(),        // Load from config
		ViewFilter:        loadViewFilter(),       // Load from config
		ViewMode:          "list",
		TreeViewType:      "directories",
		FileList:          NewScrollableList([]string{}),
		TreeList:          NewScrollableList([]string{}),
	}

	if err := loadScanData(app); err != nil {
		log.Fatalf("Failed to load scan data: %v", err)
	}

	if err := buildFileTree(app); err != nil {
		log.Fatalf("Failed to build file tree: %v", err)
	}
	

	if err := buildPURLRanking(app); err != nil {
		log.Fatalf("Failed to build PURL ranking: %v", err)
	}

	// Initialize API key (may be empty if user skipped)
	apiKey, err := getOrPromptAPIKey()
	if err != nil {
		log.Fatalf("Failed to get API key: %v", err)
	}
	app.APIKey = apiKey
	
	if app.APIKey == "" {
		fmt.Println("Running in limited mode without API key.")
	}

	setGlobalApp(app) // Set global reference for pending file counting
	initTreeState(app)
	

	g, err := gocui.NewGui(gocui.OutputNormal, true)
	if err != nil {
		fmt.Printf("Error initializing GUI: %v\n", err)
		fmt.Println("This application requires a proper terminal environment.")
		fmt.Println("Data loaded successfully:")
		fmt.Printf("- %d file entries found\n", len(app.ScanData.Files))
		fmt.Printf("- File tree built with %d top-level items\n", len(app.FileTree.Children))
		os.Exit(1)
	}
	defer g.Close()

	g.Highlight = false
	g.Cursor = false
	g.SelFgColor = gocui.ColorDefault
	
	// Don't set initial current view to avoid gocui cursor artifacts
	
	// Initial layout and populate all views
	if err := layoutWithApp(g, app); err != nil {
		log.Fatalf("Failed to create initial layout: %v", err)
	}
	
	// Initialize views that don't depend on the main loop
	updatePaneTitles(g, app)
	displayTree(g, app)
	
	// Render the initial file list (already populated by initTreeState)
	if v, err := g.View("files"); err == nil {
		isActive := (app.ActivePane == "files")
		app.FileList.Render(v, isActive)
	}
	
	
	g.SetManagerFunc(func(g *gocui.Gui) error {
		if err := layoutWithApp(g, app); err != nil {
			return err
		}
		updatePaneTitles(g, app)
		displayTree(g, app)
		
		// Always ensure file list is updated
		updateFileList(g, app)
		
		updateStatus(g, app)
		updateHelpBar(g, app)
		updateCursorPositions(g, app)
		return nil
	})

	if err := keybindings(g, app); err != nil {
		log.Panicln(err)
	}

	// Force initial file list update after everything is set up
	updateFileList(g, app)

	if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
		log.Panicln(err)
	}
}

func loadScanData(app *AppState) error {
	data, err := ioutil.ReadFile(app.FilePath)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, &app.ScanData.Files)
}

func buildFileTree(app *AppState) error {
	root := &TreeNode{
		Name:     "Root",
		Path:     "",
		IsDir:    true,
		Children: make([]*TreeNode, 0),
	}

	// Get file paths from JSON keys and filter by match type
	paths := make([]string, 0)
	for filePath, matches := range app.ScanData.Files {
		// Only include files with valid matches (id = "file" or "snippet")
		hasValidMatch := false
		for _, match := range matches {
			if match.ID == "file" || match.ID == "snippet" {
				hasValidMatch = true
				break
			}
		}
		if hasValidMatch {
			paths = append(paths, filePath)
		}
	}

	sort.Strings(paths)

	// Build directory tree (no files in tree, only directories)
	for _, path := range paths {
		parts := strings.Split(path, "/")
		current := root

		// Only create directory nodes, not file nodes
		for i, part := range parts[:len(parts)-1] { // Exclude the file name
			if part == "" {
				continue
			}

			found := false
			for _, child := range current.Children {
				if child.Name == part {
					current = child
					found = true
					break
				}
			}

			if !found {
				node := &TreeNode{
					Name:     part,
					Path:     strings.Join(parts[:i+1], "/"),
					IsDir:    true,
					Parent:   current,
					Children: make([]*TreeNode, 0),
					Files:    make([]string, 0),
				}

				current.Children = append(current.Children, node)
				current = node
			}
		}
	}

	// If no directories were created, add a virtual "All Files" node
	if len(root.Children) == 0 && len(paths) > 0 {
		allFilesNode := &TreeNode{
			Name:     "All Files",
			Path:     "",
			IsDir:    true,
			Parent:   root,
			Children: make([]*TreeNode, 0),
			Files:    make([]string, 0),
		}
		root.Children = append(root.Children, allFilesNode)
	}

	// Check if there are files in the root directory (no "/" in path)
	rootFiles := make([]string, 0)
	for filePath := range app.ScanData.Files {
		if !strings.Contains(filePath, "/") {
			rootFiles = append(rootFiles, filePath)
		}
	}
	
	// If there are files in root, add a "." directory entry at the beginning
	if len(rootFiles) > 0 {
		rootDirNode := &TreeNode{
			Name:     ".",
			Path:     "",
			IsDir:    true,
			Parent:   root,
			Children: make([]*TreeNode, 0),
			Files:    make([]string, 0),
		}
		
		// Insert at the beginning
		newChildren := make([]*TreeNode, 0, len(root.Children)+1)
		newChildren = append(newChildren, rootDirNode)
		newChildren = append(newChildren, root.Children...)
		root.Children = newChildren
	}

	app.FileTree = root
	
	// Pre-calculate pending counts for all directories
	calculateDirectoryCounts(root, app)
	
	return nil
}

func calculateDirectoryCounts(node *TreeNode, app *AppState) {
	// This function pre-calculates pending file counts for all directories
	// to ensure counts are available immediately when the tree is displayed
	for _, child := range node.Children {
		if child.IsDir {
			calculateDirectoryCounts(child, app)
		}
	}
}

func buildPURLRanking(app *AppState) error {
	purlMap := make(map[string][]string)
	
	// Collect first PURL from each file with valid matches
	for filePath, matches := range app.ScanData.Files {
		for _, match := range matches {
			// Only process files with id = "file" or "snippet"
			if match.ID != "file" && match.ID != "snippet" {
				continue
			}
			
			// Get first PURL from this match
			if len(match.Purl) > 0 {
				firstPURL := match.Purl[0]
				if _, exists := purlMap[firstPURL]; !exists {
					purlMap[firstPURL] = make([]string, 0)
				}
				purlMap[firstPURL] = append(purlMap[firstPURL], filePath)
			}
			break // Only process first valid match per file
		}
	}
	
	// Convert map to sorted slice
	app.PURLRanking = make([]PURLRankEntry, 0, len(purlMap))
	for purl, files := range purlMap {
		app.PURLRanking = append(app.PURLRanking, PURLRankEntry{
			PURL:  purl,
			Files: files,
			Count: len(files),
		})
	}
	
	// Sort by count descending, then by PURL name ascending
	sort.Slice(app.PURLRanking, func(i, j int) bool {
		if app.PURLRanking[i].Count != app.PURLRanking[j].Count {
			return app.PURLRanking[i].Count > app.PURLRanking[j].Count
		}
		return app.PURLRanking[i].PURL < app.PURLRanking[j].PURL
	})
	
	return nil
}

func layoutWithApp(g *gocui.Gui, app *AppState) error {
	maxX, maxY := g.Size()
	splitX := int(float64(maxX) * app.PaneWidth)

	// Status pane - 2 lines high at top
	if v, err := g.SetView("status", 0, 0, maxX-1, 3, 0); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "Status"
		v.Wrap = true
	}

	// Directory tree pane
	if v, err := g.SetView("tree", 0, 3, splitX-1, maxY-2, 0); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "Directories" // Default title, will be updated by updatePaneTitles
		v.Highlight = false // Disable gocui highlighting
	}

	// Files pane
	if v, err := g.SetView("files", splitX, 3, maxX-1, maxY-2, 0); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "Files" // Default title, will be updated by updatePaneTitles
		v.Wrap = true
		v.Highlight = false // Disable gocui highlighting
	}

	// Help bar with status on the right
	if v, err := g.SetView("help", 0, maxY-2, maxX-1, maxY, 0); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Frame = false
	}

	return nil
}

func keybindings(g *gocui.Gui, app *AppState) error {
	if err := g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		return err
	}
	if err := g.SetKeybinding("", 'q', gocui.ModNone, quit); err != nil {
		return err
	}
	if err := g.SetKeybinding("", gocui.KeyTab, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		// Don't allow pane switching when viewing file content
		if app.ViewMode == "content" {
			return nil
		}
		return switchPane(g, app)
	}); err != nil {
		return err
	}
	if err := g.SetKeybinding("", gocui.KeyEnter, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		return selectItem(g, app)
	}); err != nil {
		return err
	}
	if err := g.SetKeybinding("", ' ', gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		// In content view, Space = page down
		if app.ViewMode == "content" {
			return scrollFileContent(g, app, "down", true)
		}
		return nil
	}); err != nil {
		return err
	}
	if err := g.SetKeybinding("", 'a', gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		// Only allow accept when NOT in directory pane
		if app.ActivePane == "tree" {
			return nil
		}
		return showAcceptDialog(g, app)
	}); err != nil {
		return err
	}
	if err := g.SetKeybinding("", 'A', gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		// Only allow quick accept when NOT in directory pane
		if app.ActivePane == "tree" {
			return nil
		}
		return quickAccept(g, app)
	}); err != nil {
		return err
	}
	if err := g.SetKeybinding("", 'i', gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		// Only allow ignore when NOT in directory pane
		if app.ActivePane == "tree" {
			return nil
		}
		return showIgnoreDialog(g, app)
	}); err != nil {
		return err
	}
	if err := g.SetKeybinding("", 'I', gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		// Only allow quick ignore when NOT in directory pane
		if app.ActivePane == "tree" {
			return nil
		}
		return quickIgnore(g, app)
	}); err != nil {
		return err
	}
	if err := g.SetKeybinding("", 'e', gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		if isAuditDialogOpen(g) {
			return nil
		}
		return showExportDialog(g, app)
	}); err != nil {
		return err
	}
	if err := g.SetKeybinding("", 'E', gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		if isAuditDialogOpen(g) {
			return nil
		}
		return showExportDialog(g, app)
	}); err != nil {
		return err
	}
	if err := g.SetKeybinding("", gocui.KeyArrowUp, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		// Don't allow navigation if audit dialog is open
		if isAuditDialogOpen(g) {
			return nil
		}
		if app.ViewMode == "content" {
			return scrollFileContent(g, app, "up", false)
		} else if app.ActivePane == "tree" {
			return navigateTree(g, app, "up")
		} else if app.ViewMode == "list" {
			return navigateFileList(g, app, "up")
		}
		return nil
	}); err != nil {
		return err
	}
	if err := g.SetKeybinding("", gocui.KeyArrowDown, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		// Don't allow navigation if audit dialog is open
		if isAuditDialogOpen(g) {
			return nil
		}
		if app.ViewMode == "content" {
			return scrollFileContent(g, app, "down", false)
		} else if app.ActivePane == "tree" {
			return navigateTree(g, app, "down")
		} else if app.ViewMode == "list" {
			return navigateFileList(g, app, "down")
		}
		return nil
	}); err != nil {
		return err
	}
	if err := g.SetKeybinding("", gocui.KeyArrowRight, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		return resizePane(g, app, 0.05)
	}); err != nil {
		return err
	}
	if err := g.SetKeybinding("", gocui.KeyArrowLeft, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		return resizePane(g, app, -0.05)
	}); err != nil {
		return err
	}
	if err := g.SetKeybinding("", 't', gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		if isAuditDialogOpen(g) {
			return nil
		}
		return cycleViewFilter(g, app)
	}); err != nil {
		return err
	}
	if err := g.SetKeybinding("", 'T', gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		if isAuditDialogOpen(g) {
			return nil
		}
		return cycleViewFilter(g, app)
	}); err != nil {
		return err
	}
	if err := g.SetKeybinding("", gocui.KeyEsc, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		return handleEscape(g, app)
	}); err != nil {
		return err
	}
	
	// Shift+Up for page up scrolling
	if err := g.SetKeybinding("", gocui.KeyArrowUp, gocui.ModShift, func(g *gocui.Gui, v *gocui.View) error {
		if app.ViewMode == "content" {
			return scrollFileContent(g, app, "up", true)
		} else if app.ActivePane == "files" && app.ViewMode == "list" {
			return navigateFileListPage(g, app, "up")
		}
		return nil
	}); err != nil {
		return err
	}
	
	// Shift+Down for page down scrolling  
	if err := g.SetKeybinding("", gocui.KeyArrowDown, gocui.ModShift, func(g *gocui.Gui, v *gocui.View) error {
		if app.ViewMode == "content" {
			return scrollFileContent(g, app, "down", true)
		} else if app.ActivePane == "files" && app.ViewMode == "list" {
			return navigateFileListPage(g, app, "down")
		}
		return nil
	}); err != nil {
		return err
	}
	
	// Page Up key for page up scrolling
	if err := g.SetKeybinding("", gocui.KeyPgup, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		if app.ViewMode == "content" {
			return scrollFileContent(g, app, "up", true)
		} else if app.ActivePane == "files" && app.ViewMode == "list" {
			return navigateFileListPage(g, app, "up")
		}
		return nil
	}); err != nil {
		return err
	}
	
	// Page Down key for page down scrolling
	if err := g.SetKeybinding("", gocui.KeyPgdn, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		if app.ViewMode == "content" {
			return scrollFileContent(g, app, "down", true)
		} else if app.ActivePane == "files" && app.ViewMode == "list" {
			return navigateFileListPage(g, app, "down")
		}
		return nil
	}); err != nil {
		return err
	}
	
	// Shift+Space for page up scrolling
	if err := g.SetKeybinding("", ' ', gocui.ModShift, func(g *gocui.Gui, v *gocui.View) error {
		if app.ViewMode == "content" {
			return scrollFileContent(g, app, "up", true)
		}
		return nil
	}); err != nil {
		return err
	}
	
	// Toggle between PURLs and Directories view
	if err := g.SetKeybinding("", 'p', gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		if isAuditDialogOpen(g) || app.ViewMode == "content" {
			return nil
		}
		return toggleTreeViewType(g, app)
	}); err != nil {
		return err
	}
	if err := g.SetKeybinding("", 'P', gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		if isAuditDialogOpen(g) || app.ViewMode == "content" {
			return nil
		}
		return toggleTreeViewType(g, app)
	}); err != nil {
		return err
	}
	if err := g.SetKeybinding("", 'd', gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		if isAuditDialogOpen(g) || app.ViewMode == "content" {
			return nil
		}
		return toggleTreeViewType(g, app)
	}); err != nil {
		return err
	}
	if err := g.SetKeybinding("", 'D', gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		if isAuditDialogOpen(g) || app.ViewMode == "content" {
			return nil
		}
		return toggleTreeViewType(g, app)
	}); err != nil {
		return err
	}

	return nil
}

func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}

func switchPane(g *gocui.Gui, app *AppState) error {
	if app.ActivePane == "tree" {
		app.ActivePane = "files"
		// Re-render file list to show active highlighting
		if v, err := g.View("files"); err == nil {
			isActive := (app.ActivePane == "files")
			app.FileList.Render(v, isActive)
		}
	} else {
		app.ActivePane = "tree"
		// Re-render tree list to show active highlighting
		if v, err := g.View("tree"); err == nil {
			isActive := (app.ActivePane == "tree")
			app.TreeList.Render(v, isActive)
		}
	}
	
	// Force immediate update of pane titles
	updatePaneTitles(g, app)
	return nil
}

func selectItem(g *gocui.Gui, app *AppState) error {
	if app.ActivePane == "tree" {
		return toggleTreeNode(g, app)
	} else {
		if app.ViewMode == "list" {
			// Show file content
			if len(app.CurrentFileList) > 0 && app.SelectedFileIndex >= 0 && app.SelectedFileIndex < len(app.CurrentFileList) {
				app.ViewMode = "content"
				selectedFile := app.CurrentFileList[app.SelectedFileIndex]
				app.CurrentFile = selectedFile
				return displayFileContent(g, app, selectedFile)
			}
		}
		return nil
	}
}

func resizePane(g *gocui.Gui, app *AppState, delta float64) error {
	app.PaneWidth += delta
	if app.PaneWidth < 0.2 {
		app.PaneWidth = 0.2
	}
	if app.PaneWidth > 0.8 {
		app.PaneWidth = 0.8
	}
	
	// Save pane width to config
	if err := savePaneWidth(app.PaneWidth); err != nil {
		// Don't fail the resize operation if config save fails
		// Just log the error (could be improved with proper logging)
	}
	
	return nil
}

func cycleViewFilter(g *gocui.Gui, app *AppState) error {
	if app.TreeViewType == "purls" {
		// In PURL mode, only cycle between matched and pending
		switch app.ViewFilter {
		case "matched":
			app.ViewFilter = "pending"
		case "pending":
			app.ViewFilter = "matched"
		default:
			app.ViewFilter = "matched" // Default to matched in PURL mode
		}
	} else {
		// In directory mode, cycle through: all -> matched -> pending -> all
		switch app.ViewFilter {
		case "all":
			app.ViewFilter = "matched"
		case "matched":
			app.ViewFilter = "pending"
		case "pending":
			app.ViewFilter = "all"
		default:
			app.ViewFilter = "all" // Default case
		}
	}
	
	// Save the new setting to config
	if err := saveViewFilter(app.ViewFilter); err != nil {
		// Don't fail the toggle operation if config save fails
		// Just continue with the toggle
	}
	
	updateTreeDisplay(app)
	
	// If the current selection is no longer visible, select the first visible node
	if len(app.TreeState.displayLines) > 0 {
		currentVisible := false
		for _, line := range app.TreeState.displayLines {
			if line.Node == app.TreeState.selectedNode {
				currentVisible = true
				break
			}
		}
		
		// If current selection is not visible, select first available node
		if !currentVisible {
			app.TreeState.selectedNode = app.TreeState.displayLines[0].Node
			app.TreeList.SelectedIndex = 0
			app.TreeList.adjustScroll()
		}
	}
	
	displayTree(g, app)
	updateFileList(g, app)
	return nil
}

func toggleTreeViewType(g *gocui.Gui, app *AppState) error {
	if app.TreeViewType == "directories" {
		app.TreeViewType = "purls"
		// When switching to PURL mode, if currently in "all" mode, switch to "matched"
		if app.ViewFilter == "all" {
			app.ViewFilter = "matched"
		}
		// Select first PURL if available
		if len(app.PURLRanking) > 0 {
			app.TreeState.selectedNode = &TreeNode{
				Name:  app.PURLRanking[0].PURL,
				Path:  "purl_0",
				IsDir: false,
				Files: app.PURLRanking[0].Files,
			}
		}
	} else {
		app.TreeViewType = "directories"
		// Select first directory child if available
		if len(app.FileTree.Children) > 0 {
			app.TreeState.selectedNode = app.FileTree.Children[0]
		} else {
			app.TreeState.selectedNode = app.FileTree
		}
	}
	
	updateTreeDisplay(app)
	displayTree(g, app)
	updateFileList(g, app)
	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func handleEscape(g *gocui.Gui, app *AppState) error {
	if app.ViewMode == "content" {
		app.ViewMode = "list"
		app.CurrentMatch = nil // Clear current match to show general status
		updateFileList(g, app)
		return nil
	}
	return nil
}

func isAuditDialogOpen(g *gocui.Gui) bool {
	_, err1 := g.View("audit_dialog")
	_, err2 := g.View("audit_input")
	_, err3 := g.View("assessment_input")
	_, err4 := g.View("audit_error")
	_, err5 := g.View("export_dialog")
	_, err6 := g.View("export_error")
	return err1 == nil || err2 == nil || err3 == nil || err4 == nil || err5 == nil || err6 == nil
}

func updatePaneTitles(g *gocui.Gui, app *AppState) error {
	// Update tree pane title
	if v, err := g.View("tree"); err == nil {
		var title string
		if app.TreeViewType == "purls" {
			if app.ActivePane == "tree" {
				title = "[ PURLs ]"
			} else {
				title = "PURLs"
			}
		} else {
			if app.ActivePane == "tree" {
				title = "[ Directories ]"
			} else {
				title = "Directories"
			}
		}
		
		v.Title = title
		if app.ActivePane == "tree" {
			v.TitleColor = gocui.ColorYellow
		} else {
			v.TitleColor = gocui.ColorDefault
		}
	}
	
	// Update files pane title
	if v, err := g.View("files"); err == nil {
		if app.ActivePane == "files" {
			if app.ViewMode == "content" {
				v.Title = fmt.Sprintf("[ %s ]", app.CurrentFile)
			} else {
				v.Title = "[ Files ]"
			}
			v.TitleColor = gocui.ColorYellow
		} else {
			if app.ViewMode == "content" {
				v.Title = app.CurrentFile
			} else {
				v.Title = "Files"
			}
			v.TitleColor = gocui.ColorDefault
		}
	}
	
	return nil
}

func updateHelpBar(g *gocui.Gui, app *AppState) error {
	v, err := g.View("help")
	if err != nil {
		return err
	}

	v.Clear()

	// Get progress information
	auditedFiles, totalFiles, percentage := calculateProgress(app)
	statusText := fmt.Sprintf("%d%% done (%d/%d)", percentage, auditedFiles, totalFiles)
	
	// Help text
	var toggleViewText string
	if app.TreeViewType == "purls" {
		toggleViewText = "[D]irectories"
	} else {
		toggleViewText = "[P]URLs"
	}
	helpText := fmt.Sprintf("Tab: Switch panes | [T]oggle view | [a]ccept [A]quick | [i]gnore [I]quick | [E]xport CSV | %s | [Q]uit", toggleViewText)
	
	// Calculate padding to right-justify status
	maxX, _ := v.Size()
	if maxX <= 0 {
		maxX = 80 // Fallback width
	}
	
	totalContentLen := len(helpText) + len(statusText)
	if totalContentLen < maxX {
		padding := strings.Repeat(" ", maxX-totalContentLen)
		fmt.Fprintf(v, "%s%s%s", helpText, padding, statusText)
	} else {
		// If content is too long, just show help text
		fmt.Fprint(v, helpText)
	}

	return nil
}

func updateCursorPositions(g *gocui.Gui, app *AppState) error {
	// Tree cursor is now handled by custom ScrollableList - no need to manage gocui cursor
	
	// Files cursor is also handled by custom ScrollableList - no need to manage gocui cursor
	
	return nil
}
