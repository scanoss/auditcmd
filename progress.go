// Copyright (c) 2025 SCANOSS
// SPDX-License-Identifier: MIT

package main

import (
	"fmt"
	"strings"

	"github.com/awesome-gocui/gocui"
)

func calculateProgress(app *AppState) (int, int, int) {
	totalFiles := 0
	auditedFiles := 0

	// Only count files with valid matches (file or snippet)
	for _, matches := range app.ScanData.Files {
		for _, match := range matches {
			// Only count files with id = "file" or "snippet"
			if match.ID != "file" && match.ID != "snippet" {
				continue
			}

			totalFiles++

			// Check if file has been audited (any decision made)
			if len(match.AuditCmd) > 0 {
				auditedFiles++
			}
			break // Only count first valid match per file
		}
	}

	percentage := 0
	if totalFiles > 0 {
		percentage = (auditedFiles * 100) / totalFiles
	}

	return auditedFiles, totalFiles, percentage
}

func displayProgressBar(g *gocui.Gui, app *AppState) error {
	v, err := g.View("progress")
	if err != nil {
		return err
	}

	v.Clear()

	auditedFiles, totalFiles, percentage := calculateProgress(app)

	// Get the width of the progress bar view
	maxX, _ := v.Size()
	if maxX <= 0 {
		maxX = 80 // Fallback width
	}

	// Reserve space for percentage text " XXX% (XXX/XXX) "
	textSpace := 15
	barWidth := maxX - textSpace
	if barWidth < 10 {
		barWidth = 10
	}

	// Calculate filled portion
	filledWidth := (barWidth * percentage) / 100
	emptyWidth := barWidth - filledWidth

	// Build progress bar
	var progressBar strings.Builder

	// Add filled portion (green background)
	for i := 0; i < filledWidth; i++ {
		progressBar.WriteString("█")
	}

	// Add empty portion
	for i := 0; i < emptyWidth; i++ {
		progressBar.WriteString("░")
	}

	// Add percentage and count text
	progressText := fmt.Sprintf(" %3d%% (%d/%d)", percentage, auditedFiles, totalFiles)

	// Display with colors
	if percentage == 100 {
		fmt.Fprintf(v, "\033[42m\033[30m%s\033[0m\033[92m%s\033[0m", progressBar.String(), progressText)
	} else if percentage >= 75 {
		fmt.Fprintf(v, "\033[46m\033[30m%s\033[0m\033[96m%s\033[0m", progressBar.String(), progressText)
	} else if percentage >= 50 {
		fmt.Fprintf(v, "\033[43m\033[30m%s\033[0m\033[93m%s\033[0m", progressBar.String(), progressText)
	} else if percentage >= 25 {
		fmt.Fprintf(v, "\033[45m\033[30m%s\033[0m\033[95m%s\033[0m", progressBar.String(), progressText)
	} else {
		fmt.Fprintf(v, "\033[41m\033[30m%s\033[0m\033[91m%s\033[0m", progressBar.String(), progressText)
	}

	return nil
}
