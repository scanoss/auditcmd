// Copyright (c) 2025 SCANOSS
// SPDX-License-Identifier: MIT

package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
		// Close dialog first
		g.DeleteKeybindings("export_dialog")
		g.DeleteView("export_dialog")
		
		// Restore current view
		if app.ActivePane == "tree" {
			g.SetCurrentView("tree")
		} else {
			g.SetCurrentView("files")
		}
		
		// Perform export
		return performCSVExport(g, app, filename)
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

func performCSVExport(g *gocui.Gui, app *AppState, filename string) error {
	// Create or overwrite the CSV file
	file, err := os.Create(filename)
	if err != nil {
		return showExportError(g, app, fmt.Sprintf("Failed to create file: %v", err))
	}
	defer file.Close()
	
	writer := csv.NewWriter(file)
	defer writer.Flush()
	
	// Write CSV header
	header := []string{"File Path", "Match Type", "PURL", "License", "Status", "Comment"}
	if err := writer.Write(header); err != nil {
		return showExportError(g, app, fmt.Sprintf("Failed to write header: %v", err))
	}
	
	// Collect all files from the scan data
	allFiles := make(map[string]bool)
	for filePath := range app.ScanData.Files {
		allFiles[filePath] = true
	}
	
	// Export each file
	for filePath := range allFiles {
		matches, exists := app.ScanData.Files[filePath]
		
		if !exists || len(matches) == 0 {
			// File with no matches
			record := []string{filePath, "no-match", "", "", "Pending", ""}
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
			// No valid match found
			record := []string{filePath, "no-match", "", "", "Pending", ""}
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
		
		// Write record
		record := []string{filePath, match.ID, purlStr, licenseStr, status, comment}
		if err := writer.Write(record); err != nil {
			return showExportError(g, app, fmt.Sprintf("Failed to write record: %v", err))
		}
	}
	
	// Export completed successfully - return to main interface
	return nil
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

