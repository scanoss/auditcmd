// Copyright (c) 2025 SCANOSS
// SPDX-License-Identifier: MIT

package main

import (
	"testing"
)

func TestIsValidFileURL(t *testing.T) {
	tests := []struct {
		name     string
		fileURL  string
		expected bool
	}{
		{
			name:     "Valid HTTPS URL",
			fileURL:  "https://api.scanoss.com/file_contents/123abc",
			expected: true,
		},
		{
			name:     "Valid HTTPS URL with trailing spaces",
			fileURL:  "  https://api.scanoss.com/file_contents/123abc  ",
			expected: true,
		},
		{
			name:     "Empty string",
			fileURL:  "",
			expected: false,
		},
		{
			name:     "Space only",
			fileURL:  " ",
			expected: false,
		},
		{
			name:     "Whitespace only",
			fileURL:  "   \t\n   ",
			expected: false,
		},
		{
			name:     "HTTP URL (not HTTPS)",
			fileURL:  "http://api.scanoss.com/file_contents/123abc",
			expected: false,
		},
		{
			name:     "Invalid URL format",
			fileURL:  "not-a-url",
			expected: false,
		},
		{
			name:     "Different domain HTTPS",
			fileURL:  "https://example.com/file/123",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidFileURL(tt.fileURL)
			if result != tt.expected {
				t.Errorf("isValidFileURL(%q) = %v, expected %v", tt.fileURL, result, tt.expected)
			}
		})
	}
}
