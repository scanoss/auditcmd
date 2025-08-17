// Copyright (c) 2025 SCANOSS
// SPDX-License-Identifier: MIT

package main

import (
	"fmt"

	"github.com/awesome-gocui/gocui"
)

// ScrollableList represents a scrollable list component
type ScrollableList struct {
	Items         []string
	SelectedIndex int
	ScrollOffset  int
	ViewHeight    int
	ShowScrollbar bool
}

// NewScrollableList creates a new scrollable list
func NewScrollableList(items []string) *ScrollableList {
	return &ScrollableList{
		Items:         items,
		SelectedIndex: 0,
		ScrollOffset:  0,
		ViewHeight:    20,
		ShowScrollbar: true,
	}
}

// SetItems updates the list items and resets selection
func (sl *ScrollableList) SetItems(items []string) {
	sl.Items = items
	if sl.SelectedIndex >= len(items) {
		sl.SelectedIndex = 0
	}
	sl.adjustScroll()
}

// Navigate moves the selection up or down
func (sl *ScrollableList) Navigate(direction string) {
	if len(sl.Items) == 0 {
		return
	}

	switch direction {
	case "up":
		if sl.SelectedIndex > 0 {
			sl.SelectedIndex--
		}
	case "down":
		if sl.SelectedIndex < len(sl.Items)-1 {
			sl.SelectedIndex++
		}
	}
	sl.adjustScroll()
}

// NavigatePage moves by a page
func (sl *ScrollableList) NavigatePage(direction string) {
	if len(sl.Items) == 0 {
		return
	}

	pageSize := sl.ViewHeight - 1
	switch direction {
	case "up":
		newIndex := sl.SelectedIndex - pageSize
		if newIndex < 0 {
			newIndex = 0
		}
		sl.SelectedIndex = newIndex
	case "down":
		newIndex := sl.SelectedIndex + pageSize
		if newIndex >= len(sl.Items) {
			newIndex = len(sl.Items) - 1
		}
		sl.SelectedIndex = newIndex
	}
	sl.adjustScroll()
}

// adjustScroll ensures the selected item is visible
func (sl *ScrollableList) adjustScroll() {
	if len(sl.Items) == 0 {
		sl.ScrollOffset = 0
		return
	}

	// Scroll up if selection is above visible area
	if sl.SelectedIndex < sl.ScrollOffset {
		sl.ScrollOffset = sl.SelectedIndex
	}
	// Scroll down if selection is below visible area
	if sl.SelectedIndex >= sl.ScrollOffset+sl.ViewHeight {
		sl.ScrollOffset = sl.SelectedIndex - sl.ViewHeight + 1
	}

	// Ensure scroll offset is valid
	if sl.ScrollOffset < 0 {
		sl.ScrollOffset = 0
	}
	maxScroll := len(sl.Items) - sl.ViewHeight
	if maxScroll < 0 {
		maxScroll = 0
	}
	if sl.ScrollOffset > maxScroll {
		sl.ScrollOffset = maxScroll
	}
}

// Render displays the list in the given view
func (sl *ScrollableList) Render(v *gocui.View, isActive bool) {
	v.Clear()

	_, viewHeight := v.Size()
	sl.ViewHeight = viewHeight

	if len(sl.Items) == 0 {
		return
	}

	sl.adjustScroll()

	// Render visible items
	endIndex := sl.ScrollOffset + sl.ViewHeight
	if endIndex > len(sl.Items) {
		endIndex = len(sl.Items)
	}

	for i := sl.ScrollOffset; i < endIndex; i++ {
		item := sl.Items[i]

		// Highlight selected item if this pane is active
		if i == sl.SelectedIndex && isActive {
			fmt.Fprintf(v, "\033[43m\033[30m%s\033[0m\n", item)
		} else {
			fmt.Fprintf(v, "%s\n", item)
		}
	}

	// Add scrollbar if needed
	if sl.ShowScrollbar && len(sl.Items) > sl.ViewHeight {
		sl.renderScrollbar(v)
	}
}

// renderScrollbar draws a simple scrollbar on the right side
func (sl *ScrollableList) renderScrollbar(v *gocui.View) {
	viewWidth, viewHeight := v.Size()
	if viewWidth < 2 || viewHeight < 3 {
		return
	}

	// Simple scroll indicator
	totalItems := len(sl.Items)
	if totalItems > sl.ViewHeight {
		scrollInfo := fmt.Sprintf("[%d/%d]", sl.SelectedIndex+1, totalItems)
		fmt.Fprintf(v, "\033[0;0H\033[K%s", scrollInfo) // Move to top right and show info
	}
}

// GetSelectedItem returns the currently selected item
func (sl *ScrollableList) GetSelectedItem() string {
	if sl.SelectedIndex >= 0 && sl.SelectedIndex < len(sl.Items) {
		return sl.Items[sl.SelectedIndex]
	}
	return ""
}

// GetSelectedIndex returns the currently selected index
func (sl *ScrollableList) GetSelectedIndex() int {
	return sl.SelectedIndex
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
