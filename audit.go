// Copyright (c) 2025 SCANOSS
// SPDX-License-Identifier: MIT

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"github.com/awesome-gocui/gocui"
)

func showAcceptDialog(g *gocui.Gui, app *AppState) error {
	// If no current match is set, try to get it from the selected file
	if app.CurrentMatch == nil {
		if app.ActivePane == "files" && len(app.CurrentFileList) > 0 && app.SelectedFileIndex < len(app.CurrentFileList) {
			selectedFile := app.CurrentFileList[app.SelectedFileIndex]
			matches, exists := app.ScanData.Files[selectedFile]
			if exists && len(matches) > 0 {
				// Find the first valid match (file or snippet)
				for i, m := range matches {
					if m.ID == "file" || m.ID == "snippet" {
						app.CurrentMatch = &matches[i]
						break
					}
				}
			}
		}
	}
	
	if app.CurrentMatch == nil {
		// Show a message if no auditable file is selected
		maxX, maxY := g.Size()
		if v, err := g.SetView("audit_error", maxX/4, maxY/3, 3*maxX/4, maxY/3+4, 0); err != nil {
			if err != gocui.ErrUnknownView {
				return err
			}
			v.Title = "No File Selected"
			v.Frame = true
			fmt.Fprint(v, "Please select a file with matches to audit.\nPress ESC to close this message.")
			
			g.SetKeybinding("audit_error", gocui.KeyEsc, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
				g.DeleteKeybindings("audit_error")
				g.DeleteView("audit_error")
				if app.ActivePane == "tree" {
					g.SetCurrentView("tree")
				} else {
					g.SetCurrentView("files")
				}
				return nil
			})
			
			if _, err := g.SetCurrentView("audit_error"); err != nil {
				return err
			}
		}
		return nil
	}

	maxX, maxY := g.Size()
	
	// Set decision to identified for accept dialog
	app.PendingDecision = "identified"
	
	// Main dialog frame - fixed 4-line height
	if v, err := g.SetView("audit_dialog", maxX/4, maxY/3, 3*maxX/4, maxY/3+5, 0); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "ACCEPT Identification"
		v.Frame = true
		v.Editable = false
		v.TitleColor = gocui.ColorYellow
		v.BgColor = gocui.ColorBlack
		v.FgColor = gocui.ColorYellow
	}
	
	// Input field - 2 lines in the middle (lines 2-3)
	if v, err := g.SetView("audit_input", maxX/4+1, maxY/3+1, 3*maxX/4-1, maxY/3+3, 0); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Frame = false
		v.Editable = true
		v.Wrap = true
		v.BgColor = gocui.ColorBlack
		v.FgColor = gocui.ColorYellow
		
		if _, err := g.SetCurrentView("audit_input"); err != nil {
			return err
		}
	}
	
	// Update the dialog display
	updateAcceptDialog(g, app)
	
	// Clear any existing keybindings first
	g.DeleteKeybindings("audit_dialog")
	g.DeleteKeybindings("audit_input")
	
	// Set up keybindings for the input field
	g.SetKeybinding("audit_input", gocui.KeyEnter, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		return saveAuditDecision(g, app)
	})
	
	g.SetKeybinding("audit_input", gocui.KeyEsc, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		return closeAuditDialog(g, app)
	})

	return nil
}

func showIgnoreDialog(g *gocui.Gui, app *AppState) error {
	// If no current match is set, try to get it from the selected file
	if app.CurrentMatch == nil {
		if app.ActivePane == "files" && len(app.CurrentFileList) > 0 && app.SelectedFileIndex < len(app.CurrentFileList) {
			selectedFile := app.CurrentFileList[app.SelectedFileIndex]
			matches, exists := app.ScanData.Files[selectedFile]
			if exists && len(matches) > 0 {
				// Find the first valid match (file or snippet)
				for i, m := range matches {
					if m.ID == "file" || m.ID == "snippet" {
						app.CurrentMatch = &matches[i]
						break
					}
				}
			}
		}
	}
	
	if app.CurrentMatch == nil {
		// Show a message if no auditable file is selected
		maxX, maxY := g.Size()
		if v, err := g.SetView("audit_error", maxX/4, maxY/3, 3*maxX/4, maxY/3+4, 0); err != nil {
			if err != gocui.ErrUnknownView {
				return err
			}
			v.Title = "No File Selected"
			v.Frame = true
			fmt.Fprint(v, "Please select a file with matches to audit.\nPress ESC to close this message.")
			
			g.SetKeybinding("audit_error", gocui.KeyEsc, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
				g.DeleteKeybindings("audit_error")
				g.DeleteView("audit_error")
				if app.ActivePane == "tree" {
					g.SetCurrentView("tree")
				} else {
					g.SetCurrentView("files")
				}
				return nil
			})
			
			if _, err := g.SetCurrentView("audit_error"); err != nil {
				return err
			}
		}
		return nil
	}

	maxX, maxY := g.Size()
	
	// Set decision to ignored for ignore dialog
	app.PendingDecision = "ignored"
	
	// Main dialog frame - fixed 4-line height
	if v, err := g.SetView("audit_dialog", maxX/4, maxY/3, 3*maxX/4, maxY/3+5, 0); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "IGNORE Identification"
		v.Frame = true
		v.Editable = false
		v.TitleColor = gocui.ColorYellow
		v.BgColor = gocui.ColorBlack
		v.FgColor = gocui.ColorYellow
	}
	
	// Input field - 2 lines in the middle (lines 2-3)
	if v, err := g.SetView("audit_input", maxX/4+1, maxY/3+1, 3*maxX/4-1, maxY/3+3, 0); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Frame = false
		v.Editable = true
		v.Wrap = true
		v.BgColor = gocui.ColorBlack
		v.FgColor = gocui.ColorYellow
		
		if _, err := g.SetCurrentView("audit_input"); err != nil {
			return err
		}
	}
	
	// Update the dialog display
	updateIgnoreDialog(g, app)
	
	// Clear any existing keybindings first
	g.DeleteKeybindings("audit_dialog")
	g.DeleteKeybindings("audit_input")
	
	// Set up keybindings for the input field
	g.SetKeybinding("audit_input", gocui.KeyEnter, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		return saveAuditDecision(g, app)
	})
	
	g.SetKeybinding("audit_input", gocui.KeyEsc, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		return closeAuditDialog(g, app)
	})

	return nil
}

func updateAcceptDialog(g *gocui.Gui, app *AppState) error {
	v, err := g.View("audit_dialog")
	if err != nil {
		return err
	}
	
	// Line 1: Comment label, Lines 2-3: input area, Line 4: help
	v.Clear()
	fmt.Fprintf(v, " Comment (Optional)\n")
	fmt.Fprintf(v, "\n")
	fmt.Fprintf(v, "\n")
	fmt.Fprintf(v, " ENTER: Accept  ESC: Cancel")
	
	// Clear input field
	if iv, err := g.View("audit_input"); err == nil {
		iv.Clear()
		iv.SetCursor(0, 0)
	}
	
	return nil
}

func updateIgnoreDialog(g *gocui.Gui, app *AppState) error {
	v, err := g.View("audit_dialog")
	if err != nil {
		return err
	}
	
	// Line 1: Comment label, Lines 2-3: input area, Line 4: help
	v.Clear()
	fmt.Fprintf(v, " Comment (Optional)\n")
	fmt.Fprintf(v, "\n")
	fmt.Fprintf(v, "\n")
	fmt.Fprintf(v, " ENTER: Ignore  ESC: Cancel")
	
	// Clear input field
	if iv, err := g.View("audit_input"); err == nil {
		iv.Clear()
		iv.SetCursor(0, 0)
	}
	
	return nil
}


func promptAssessment(g *gocui.Gui, app *AppState, decision string) error {
	app.PendingDecision = decision
	
	maxX, maxY := g.Size()
	
	if v, err := g.SetView("assessment_input", maxX/4, maxY/3+5, 3*maxX/4, 2*maxY/3, 0); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = fmt.Sprintf("Assessment for %s", strings.ToUpper(decision))
		v.Frame = true
		v.Editable = true
		v.Wrap = true
		
		fmt.Fprintf(v, "Decision: %s\n", strings.ToUpper(decision))
		fmt.Fprint(v, "Assessment (optional): ")
		
		if _, err := g.SetCurrentView("assessment_input"); err != nil {
			return err
		}
	}

	g.SetKeybinding("assessment_input", gocui.KeyEnter, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		return saveAuditDecision(g, app)
	})
	
	g.SetKeybinding("assessment_input", gocui.KeyEsc, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		return closeAuditDialog(g, app)
	})

	return nil
}

func saveAuditDecision(g *gocui.Gui, app *AppState) error {
	if app.CurrentMatch == nil || app.PendingDecision == "" {
		return closeAuditDialog(g, app)
	}

	// Get assessment from the input field
	v, err := g.View("audit_input")
	if err != nil {
		return err
	}
	assessment := strings.TrimSpace(v.Buffer())

	decision := AuditDecision{
		Decision:   app.PendingDecision,
		Assessment: assessment,
		Timestamp:  time.Now(),
	}

	app.CurrentMatch.AuditCmd = append(app.CurrentMatch.AuditCmd, decision)

	if err := saveToFile(app); err != nil {
		// Show error dialog instead of printf
		maxX, maxY := g.Size()
		if v, errView := g.SetView("save_error", maxX/4, maxY/2-2, 3*maxX/4, maxY/2+2, 0); errView != nil {
			if errView != gocui.ErrUnknownView {
				return errView
			}
			v.Title = "Save Error"
			v.Frame = true
			fmt.Fprintf(v, "Error saving audit decision: %v\nPress ESC to continue", err)
			
			g.SetKeybinding("save_error", gocui.KeyEsc, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
				g.DeleteKeybindings("save_error")
				g.DeleteView("save_error")
				return closeAuditDialog(g, app)
			})
			
			g.SetCurrentView("save_error")
		}
		return nil
	}

	app.PendingDecision = ""
	app.PendingAssessment = ""
	
	// Clear current match so subsequent audits work correctly
	app.CurrentMatch = nil
	
	closeAuditDialog(g, app)
	updateFileList(g, app)

	return nil
}

func closeAuditDialog(g *gocui.Gui, app *AppState) error {
	g.DeleteKeybindings("audit_dialog")
	g.DeleteKeybindings("audit_input")
	g.DeleteKeybindings("assessment_input")
	
	if err := g.DeleteView("audit_dialog"); err != nil && err != gocui.ErrUnknownView {
		return err
	}
	
	if err := g.DeleteView("audit_input"); err != nil && err != gocui.ErrUnknownView {
		return err
	}
	
	if err := g.DeleteView("assessment_input"); err != nil && err != gocui.ErrUnknownView {
		return err
	}

	// Reset pending decision and assessment
	app.PendingDecision = ""
	app.PendingAssessment = ""
	
	// Clear current match so status pane returns to directory info
	app.CurrentMatch = nil

	if app.ActivePane == "tree" {
		g.SetCurrentView("tree")
	} else {
		g.SetCurrentView("files")
	}

	return nil
}

func saveToFile(app *AppState) error {
	data, err := json.MarshalIndent(app.ScanData.Files, "", "  ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(app.FilePath, data, 0644)
}