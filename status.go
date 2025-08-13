// Copyright (c) 2025 SCANOSS
// SPDX-License-Identifier: MIT

package main

import (
	"fmt"
	"strings"

	"github.com/awesome-gocui/gocui"
)

func updateStatus(g *gocui.Gui, app *AppState) error {
	v, err := g.View("status")
	if err != nil {
		return err
	}

	v.Clear()

	if app.CurrentMatch != nil {
		displayFileStatus(v, app.CurrentMatch)
	} else if app.TreeState != nil && app.TreeState.selectedNode != nil && app.TreeState.selectedNode.IsDir {
		displayDirectoryStatus(v, app)
	}

	return nil
}

func displayFileStatus(v *gocui.View, match *FileMatch) {
	// Line 1: Type, component
	component := ""
	if len(match.Purl) > 0 {
		component = match.Purl[0]
	}
	fmt.Fprintf(v, "\033[1mType:\033[0m \033[37m%s\033[0m | \033[1mComponent:\033[0m \033[37m%s\033[0m", strings.ToUpper(match.ID), component)
	
	// Add licenses to line 1
	if len(match.Licenses) > 0 {
		licenseNames := make([]string, 0)
		for _, license := range match.Licenses {
			licenseNames = append(licenseNames, license.Name)
		}
		licenses := strings.Join(licenseNames, ", ")
		fmt.Fprintf(v, " | \033[1mLicenses:\033[0m \033[37m%s\033[0m", licenses)
	}
	fmt.Fprintf(v, "\n")
	
	// Line 2: Audit status
	auditStatus := "PENDING"
	assessment := ""
	if len(match.AuditCmd) > 0 {
		latest := match.AuditCmd[len(match.AuditCmd)-1]
		auditStatus = strings.ToUpper(latest.Decision)
		if latest.Assessment != "" {
			assessment = " (" + latest.Assessment + ")"
		}
	}
	
	fmt.Fprintf(v, "\033[1mAudit:\033[0m \033[37m%s%s\033[0m\n", auditStatus, assessment)
}

func displayDirectoryStatus(v *gocui.View, app *AppState) {
	totalFilesInData := len(app.ScanData.Files)
	matchingFiles := 0
	pendingFiles := 0
	identifiedFiles := 0
	ignoredFiles := 0

	// Count files with valid matches (file or snippet)
	for _, matches := range app.ScanData.Files {
		for _, match := range matches {
			// Only count files with id = "file" or "snippet"
			if match.ID != "file" && match.ID != "snippet" {
				continue
			}
			
			matchingFiles++
			
			if len(match.AuditCmd) > 0 {
				latest := match.AuditCmd[len(match.AuditCmd)-1]
				if latest.Decision == "identified" {
					identifiedFiles++
				} else if latest.Decision == "ignored" {
					ignoredFiles++
				} else {
					pendingFiles++
				}
			} else {
				pendingFiles++
			}
			break // Only count first valid match per file
		}
	}

	// Line 1: File counts overview
	fmt.Fprintf(v, "\033[1mTotal Files:\033[0m \033[37m%d\033[0m | \033[1mMatches:\033[0m \033[37m%d\033[0m", totalFilesInData, matchingFiles)
	
	// Line 2: Audit status breakdown and API status
	apiStatus := "API key \033[1mOK\033[0m"
	if app.APIKey == "" {
		apiStatus = "API key \033[1mNO\033[0m"
	}
	fmt.Fprintf(v, "\n\033[1mPending:\033[0m \033[37m%d\033[0m | \033[1mIdentified:\033[0m \033[37m%d\033[0m | \033[1mIgnored:\033[0m \033[37m%d\033[0m | \033[1mHide Processed:\033[0m \033[37m%t\033[0m | \033[1mShowing:\033[0m \033[37m%d\033[0m | %s", pendingFiles, identifiedFiles, ignoredFiles, app.HideIdentified, len(app.CurrentFileList), apiStatus)
}