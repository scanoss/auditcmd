// Copyright (c) 2025 SCANOSS
// SPDX-License-Identifier: MIT

package main

import (
	"testing"
)

func TestIsGeneratedWithAPIKey(t *testing.T) {
	tests := []struct {
		name     string
		files    map[string][]FileMatch
		expected bool
	}{
		{
			name: "File with valid API key URL",
			files: map[string][]FileMatch{
				"test.go": {
					{
						ID:      "file",
						FileURL: "https://api.scanoss.com/file_contents/123abc",
					},
				},
			},
			expected: true,
		},
		{
			name: "File without API key (empty URL)",
			files: map[string][]FileMatch{
				"test.go": {
					{
						ID:      "file",
						FileURL: "",
					},
				},
			},
			expected: false,
		},
		{
			name: "File without API key (space only)",
			files: map[string][]FileMatch{
				"test.go": {
					{
						ID:      "file",
						FileURL: " ",
					},
				},
			},
			expected: false,
		},
		{
			name: "File without API key (whitespace)",
			files: map[string][]FileMatch{
				"test.go": {
					{
						ID:      "file",
						FileURL: "   \t\n   ",
					},
				},
			},
			expected: false,
		},
		{
			name: "Snippet with valid API key URL",
			files: map[string][]FileMatch{
				"test.go": {
					{
						ID:      "snippet",
						FileURL: "https://api.scanoss.com/file_contents/456def",
					},
				},
			},
			expected: true,
		},
		{
			name: "Mix of file types with one invalid",
			files: map[string][]FileMatch{
				"test1.go": {
					{
						ID:      "file",
						FileURL: "https://api.scanoss.com/file_contents/123abc",
					},
				},
				"test2.go": {
					{
						ID:      "snippet",
						FileURL: " ",
					},
				},
			},
			expected: false,
		},
		{
			name: "Non-file/snippet matches should be ignored",
			files: map[string][]FileMatch{
				"test.go": {
					{
						ID:      "none",
						FileURL: "",
					},
				},
			},
			expected: true,
		},
		{
			name: "Invalid URL format",
			files: map[string][]FileMatch{
				"test.go": {
					{
						ID:      "file",
						FileURL: "invalid-url",
					},
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isGeneratedWithAPIKey(tt.files)
			if result != tt.expected {
				t.Errorf("isGeneratedWithAPIKey() = %v, expected %v", result, tt.expected)
			}
		})
	}
}
