// Copyright (c) 2025 SCANOSS
// SPDX-License-Identifier: MIT

package main

import (
	"time"
)

type ScanResult struct {
	Files map[string][]FileMatch `json:",inline"`
}

type FileMatch struct {
	Component     string           `json:"component"`
	Copyrights    []Copyright      `json:"copyrights"`
	Cryptography  []interface{}    `json:"cryptography"`
	Dependencies  []interface{}    `json:"dependencies"`
	File          string           `json:"file"`
	FileHash      string           `json:"file_hash"`
	FileURL       string           `json:"file_url"`
	Health        Health           `json:"health"`
	ID            string           `json:"id"`
	Latest        string           `json:"latest"`
	Licenses      []License        `json:"licenses"`
	OSSLines      interface{}      `json:"oss_lines"`
	Purl          []string         `json:"purl"`
	Quality       []Quality        `json:"quality"`
	ReleaseDate   string           `json:"release_date"`
	Server        Server           `json:"server"`
	SourceHash    string           `json:"source_hash"`
	Status        string           `json:"status"`
	URL           string           `json:"url"`
	URLHash       string           `json:"url_hash"`
	URLStats      URLStats         `json:"url_stats"`
	Version       string           `json:"version"`
	AuditCmd      []AuditDecision  `json:"audit,omitempty"`
}

type Copyright struct {
	Name   string `json:"name"`
	Source string `json:"source"`
}

type Health struct {
	CreationDate string `json:"creation_date"`
	Forks        int    `json:"forks"`
	Issues       int    `json:"issues"`
	LastPush     string `json:"last_push"`
	LastUpdate   string `json:"last_update"`
	Stars        int    `json:"stars"`
}

type License struct {
	ChecklistURL  string `json:"checklist_url,omitempty"`
	Copyleft      string `json:"copyleft,omitempty"`
	Name          string `json:"name"`
	OSADLUpdated  string `json:"osadl_updated,omitempty"`
	PatentHints   string `json:"patent_hints,omitempty"`
	Source        string `json:"source"`
	URL           string `json:"url,omitempty"`
}

type Quality struct {
	Score  string `json:"score"`
	Source string `json:"source"`
}

type Server struct {
	Elapsed   string            `json:"elapsed"`
	Flags     string            `json:"flags"`
	Hostname  string            `json:"hostname"`
	KBVersion map[string]string `json:"kb_version"`
	Version   string            `json:"version"`
}

type URLStats struct {
	IgnoredFiles  int `json:"ignored_files"`
	IndexedFiles  int `json:"indexed_files"`
	PackageSize   int `json:"package_size"`
	SourceFiles   int `json:"source_files"`
}

type AuditDecision struct {
	Decision   string    `json:"decision"`
	Assessment string    `json:"assessment,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
}

type PURLRankEntry struct {
	PURL     string
	Files    []string
	Count    int
}

type AppState struct {
	ScanData          ScanResult
	CurrentFile       string
	CurrentMatch      *FileMatch
	FileTree          *TreeNode
	TreeState         *TreeState
	ActivePane        string
	FilePath          string
	CurrentFileList   []string
	SelectedFileIndex int
	PendingDecision   string
	PendingAssessment string
	PaneWidth         float64
	ViewFilter        string // "all", "matched", "pending"
	APIKey            string
	ViewMode          string // "list" or "content"
	TreeViewType      string // "directories" or "purls"
	PURLRanking       []PURLRankEntry
	InitialFileListDone bool   // Track if initial file list has been populated
	FileList          *ScrollableList // Custom scrollable file list
	TreeList          *ScrollableList // Custom scrollable tree list
	ProcessingQuickAction bool // Flag to prevent concurrent quick actions
}

type TreeNode struct {
	Name     string
	Path     string
	IsDir    bool
	Children []*TreeNode
	Parent   *TreeNode
	Files    []string
}

type TreeState struct {
	selectedNode *TreeNode
	expandedDirs map[string]bool
	displayLines []TreeDisplayLine
}

type TreeDisplayLine struct {
	Node   *TreeNode
	Indent int
	Line   string
}