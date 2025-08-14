# AuditCmd

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Report Card](https://goreportcard.com/badge/github.com/scanoss/auditcmd)](https://goreportcard.com/report/github.com/scanoss/auditcmd)

A command-line auditing tool for reviewing SCANOSS Open Source scanning results with a console UI built using gocui.

> **Note**: This is a console-based TUI application that requires a proper terminal environment to run.

## Features

### Core Functionality
- **Directory Tree Navigation**: Browse through directory structure with collapsible directories
- **PURL Ranking View**: Switch to component-centric view showing PURLs ranked by file count
- **Dual View Toggle**: Press [P] for PURL view or [D] for Directory view
- **Resizable Panes**: Use Left/Right arrow keys to adjust pane sizes
- **File Filtering**: Only displays files with actual Open Source matches (`id = "file"` or `id = "snippet"`)
- **Visual File Status**: Files show ✓ (identified), ✗ (ignored), or no symbol (unprocessed)
- **Smart Hide/Show Toggle**: Press [T] to toggle visibility of audited files (works in both Directory and PURL modes)

### Data Export & Persistence
- **CSV Export**: Press [E] to export audit results to CSV with comprehensive file information
- **Configuration Persistence**: All settings (API key, pane width, audited filter state) saved to `~/.auditcmd`
- **Auto-restore Settings**: Application remembers your preferences across sessions

### API Integration & Content Viewing  
- **Secure API Integration**: Automatic API key management with secure storage and SCANOSS API authentication
- **File Content Viewer**: Display file contents fetched from SCANOSS API with line highlighting for snippet matches
- **Direct Audit Actions**: Press [A]ccept or [I]gnore to make instant audit decisions with optional comments

### User Experience
- **Progress Tracking**: Real-time progress bar showing audit completion percentage across all files
- **Comprehensive Status Display**: Shows file/directory statistics, audit counts, and API status
- **Full Keyboard Navigation**: Efficient keyboard-only interface with context-sensitive help

## Usage

```bash
./auditcmd <scanoss-result.json>
./auditcmd --reset-api-key      # Remove stored API key
./auditcmd --api-key-status     # Check API key configuration
```

Where `<scanoss-result.json>` is the JSON file containing SCANOSS scan results.

## API Key Management

The application requires a SCANOSS API key to fetch file contents. On first run:

1. **Initial Setup**: You'll be prompted to enter your SCANOSS API key or skip
2. **Skip Option**: Enter 'skip' to run in limited mode without file content viewing
3. **Secure Storage**: API key is saved to `~/.auditcmd` with secure file permissions (600)
4. **Automatic Loading**: Subsequent runs will automatically use the stored API key
5. **Status Check**: Use `./auditcmd --api-key-status` to check if an API key is configured
6. **Reset Option**: Use `./auditcmd --reset-api-key` to remove and reset your stored API key

### Limited Mode (No API Key)
When running without an API key, you can still:
- ✅ Navigate directory tree and file lists
- ✅ View file metadata (PURL, licenses, match type)
- ✅ Make audit decisions (IDENTIFY/IGNORE with assessments)
- ✅ Save audit results to JSON
- ❌ View actual file contents
- ❌ See highlighted snippet matches

The API key is sent with requests using the `X-API-Key` header as required by the SCANOSS API.

## Interface Layout

The application is divided into four main sections:

### Status Panel (Top, 2 lines)
- **Line 1**: File/Directory info, component PURL, licenses 
- **Line 2**: Audit statistics (Pending, Identified, Ignored), Audited filter status, API key status
- Shows comprehensive audit progress and current filter state
- Works independently in both Directory and PURL view modes

### Left Panel (Resizable - Dual View)
**Directory View (Default)**:
- Collapsible directory structure (directories only, no files shown in tree)
- **Dynamic Count**: Shows file count based on current filter state (e.g., "src (23)" or "src (5)" when hiding audited)
- Navigate with Up/Down arrow keys, Enter to expand/collapse
- Only shows directories containing files with valid Open Source matches

**PURL View (Press [P] to switch)**:
- Component-centric view showing Package URLs ranked by file count
- **Ranked by Impact**: PURLs with most files appear first 
- **Dynamic Count**: Shows count based on filter (e.g., "pkg:npm/react@18.2.0 (45)" or "(12)" when hiding audited)
- Navigate with Up/Down arrow keys to select PURL

### Right Panel (Resizable - Files/Content)
**List Mode**: Shows files from selected directory or PURL
- **Clean Display**: File paths only (no clutter)
- **Visual Status**: Files show ✓ (identified), ✗ (ignored), or no symbol (unprocessed)
- Navigate with Up/Down arrow keys
- **Smart Filtering**: [T] key toggles audited files in both Directory and PURL modes

**Content Mode**: Shows actual file source code
- **Syntax Highlighting**: Line numbers and highlighted snippet matches
- **ESC key**: Return to file list
- **[A]ccept/[I]gnore**: Make audit decisions while viewing content

### Export & Configuration
- **[E] CSV Export**: Export comprehensive audit results to CSV file  
- **Auto-naming**: Defaults to input filename with .csv extension
- **Overwrite Confirmation**: Shows file existence warning before export
- **Persistent Settings**: All preferences saved automatically to `~/.auditcmd`

## Keyboard Controls

### Navigation
- **Tab**: Switch between left panel (Directories/PURLs) and Files panel
- **Up/Down**: Navigate in the active panel (directory tree, PURL list, or file list)
- **Left/Right**: 
  - In Directories panel: Collapse/expand directories
  - In Files panel: Resize panels (make left panel smaller/larger)
- **Enter**: 
  - In Directories: Expand/collapse directory
  - In Files List: View file content
- **ESC**: Return from file content view to file list

### View Controls
- **[P]**: Switch to PURL ranking view (component-centric)
- **[D]**: Switch to Directory tree view (file system structure)
- **[T]**: Toggle audited files visibility (works in both Directory and PURL modes)

### Audit Actions
- **[A]**: Accept/Identify current file as valid Open Source match with optional comment
- **[I]**: Ignore current file as false positive with optional comment

### Export & System
- **[E]**: Export audit results to CSV file
- **[Q]** or **Ctrl+C**: Quit application

### Content Viewing (when viewing file content)
- **Space**: Page down
- **Shift+Space**: Page up  
- **Shift+Up/Down**: Page up/down
- **Page Up/Page Down**: Page navigation

## Dual View System

The application offers two complementary ways to navigate your scan results:

### Directory View (Default)
- **File System Structure**: Traditional directory tree showing how files are organized
- **Directory Focus**: Navigate by folder structure to understand codebase organization  
- **Collapsible Tree**: Expand/collapse directories to focus on specific areas
- **Best For**: Understanding file organization, working through directories systematically

### PURL View ([P] to switch)
- **Component Focus**: Shows Package URLs (PURLs) ranked by number of matching files
- **Impact-Based**: Most prevalent components appear first
- **Dependency Analysis**: Quickly identify which components affect the most files
- **Best For**: Understanding component dependencies, focusing on high-impact packages

Both views show dynamic file counts that update based on the audited filter state, and file navigation works identically in both modes.

## Audit Process

1. Navigate to a file using either directory tree, PURL ranking, or file list
2. Press **[A]** to accept the match or **[I]** to ignore it
3. A compact modal appears with:
   - Line 1: "Comment (Optional)" label
   - Lines 2-3: Text entry area for optional assessment comment
   - Line 4: "ENTER: Accept/Ignore  ESC: Cancel"
4. Type your optional comment (or leave blank)
5. Press **Enter** to save the decision or **ESC** to cancel

Audit decisions are saved directly to the original JSON file in an `audit` array for each file match.

## CSV Export

The application provides comprehensive CSV export functionality:

### Export Process
1. Press **[E]** from any view (Directory or PURL mode)
2. Review the export dialog showing:
   - Target filename (automatically generated from input JSON)
   - Overwrite warning if file exists
3. Press **Enter** to export or **ESC** to cancel
4. Export completes silently and returns to main interface

### CSV Format
The exported CSV includes the following columns:
- **File Path**: Full path to each file in the scan results
- **Match Type**: "file", "snippet", or "no-match" for files without valid matches
- **PURL**: Package URL(s) - concatenated with "; " separator for multiple PURLs
- **License**: License name(s) - concatenated with "; " separator for multiple licenses  
- **Status**: "Pending", "Accepted" (identified), or "Ignored"
- **Comment**: Auditor assessment/comment if provided

### Export Features
- **Comprehensive**: Exports ALL files from scan data, including those without matches
- **Current State**: Reflects all audit decisions made during the session
- **Auto-naming**: Uses input JSON filename with `.csv` extension (e.g., `scan-results.json` → `scan-results.csv`)
- **Overwrite**: Silently overwrites existing files after confirmation

## Data Structure

The tool expects SCANOSS JSON format with the following key fields:
- `id`: "file", "snippet", or "none"
- `file`: File path
- `file_url`: URL to fetch file content
- `oss_lines`: Line ranges for snippet matches
- `purl`: Package URL identifiers
- `licenses`: License information
- `audit`: Array of audit decisions (added by this tool)

## Configuration

The application automatically manages configuration in `~/.auditcmd`:

### Stored Settings
- **API Key**: SCANOSS API key for content fetching (secure 600 permissions)
- **Pane Width**: Left panel width ratio (0.2 to 0.8)
- **Audited Filter**: Hide/show audited files state (true/false)

### Configuration Format
```ini
# AuditCmd Configuration
# This file stores settings for the AuditCmd application

api_key=your_scanoss_api_key_here
pane_width=0.50
hide_identified=false
```

### Management Commands
- **Status Check**: `./auditcmd --api-key-status`
- **Reset API Key**: `./auditcmd --reset-api-key`
- **Auto-save**: All UI changes (pane resize, filter toggle) save automatically

## Building

```bash
go mod tidy
go build
```

## Dependencies

- github.com/awesome-gocui/gocui: Console UI framework
- golang.org/x/term: Terminal functionality for secure API key input

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request. For major changes, please open an issue first to discuss what you would like to change.

## Support

For questions, issues, or feature requests, please visit the [SCANOSS website](https://www.scanoss.com) or open an issue in this repository.

## File Structure

- `main.go`: Application entry point, core logic, and dual-view management
- `models.go`: Data structures for SCANOSS JSON format and PURL ranking
- `tree.go`: Directory tree and PURL ranking navigation with dynamic counting
- `filelist.go`: File listing and content viewing for both view modes
- `status.go`: Status panel implementation with comprehensive audit statistics
- `audit.go`: Audit decision functionality and dialog management
- `export.go`: CSV export functionality with comprehensive file reporting
- `apikey.go`: Configuration management, API key storage, and settings persistence
- `progress.go`: Progress tracking and completion percentage calculations