# AuditCmd

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Report Card](https://goreportcard.com/badge/github.com/scanoss/auditcmd)](https://goreportcard.com/report/github.com/scanoss/auditcmd)

A command-line auditing tool for reviewing SCANOSS Open Source scanning results with a console UI built using gocui.

> **Note**: This is a console-based TUI application that requires a proper terminal environment to run.

## Features

- **Directory Tree Navigation**: Browse through directory structure with collapsible directories (left pane)
- **Resizable Panes**: Use Left/Right arrow keys to adjust pane sizes
- **File Filtering**: Only displays files with actual Open Source matches (`id = "file"` or `id = "snippet"`)
- **Visual File Status**: Files show ✓ (identified), ✗ (ignored), or no symbol (unprocessed)
- **Hide/Show Toggle**: Press 'T' to toggle visibility of processed files (both identified and ignored)
- **Secure API Integration**: Automatic API key management with secure storage and SCANOSS API authentication
- **File Content Viewer**: Display file contents fetched from SCANOSS API with line highlighting for snippet matches
- **Direct Audit Actions**: Press [A]ccept or [I]gnore to make instant audit decisions with optional comments
- **Progress Tracking**: Real-time progress bar showing audit completion percentage across all files
- **Compact Status Display**: 2-line status showing essential information (PURL, licenses, audit status)
- **Keyboard Navigation**: Full keyboard interface with efficient pane switching

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

### Progress Bar (Top line)
- Visual progress indicator showing audit completion (0% to 100%)
- Color-coded: Red (0-24%), Purple (25-49%), Yellow (50-74%), Cyan (75-99%), Green (100%)
- Displays current percentage and count (e.g., "67% (150/224)")
- Updates in real-time as audit decisions are made

### Status Panel (2 lines)
- **Line 1**: File/Directory info, type, component PURL
- **Line 2**: Licenses and audit status with assessment
- Compact display optimized for essential information

### Bottom Left Panel (Resizable - Directories) 
- Collapsible directory structure (directories only, no files shown in tree)
- Navigate with Up/Down arrow keys  
- Press Enter or Right Arrow to expand/collapse directories
- **Pending Count**: Shows number of unaudited files in parentheses (e.g., "src (23)")
- Left/Right arrows resize the pane when in Files pane
- Only shows directories that contain files with valid Open Source matches

### Bottom Right Panel (Resizable - Files/Content)
- **List Mode**: Shows all files in selected directory and subdirectories
  - **Clean Display**: File paths only (no type indicators)
  - **Visual Status**: Files show ✓ (identified), ✗ (ignored), or no symbol (unprocessed)
  - Navigate with Up/Down arrow keys
  - Press **Enter** to view file content
- **Content Mode**: Shows actual file source code
  - **Syntax Highlighting**: Line numbers and highlighted snippet matches
  - **ESC key**: Return to file list
  - **[A]ccept/[I]gnore**: Make audit decisions while viewing content
- **T key**: Toggle hide/show processed files (both identified and ignored)

## Keyboard Controls

- **Tab**: Switch between Directories and Files panes
- **Up/Down**: Navigate in the active pane (directory tree or file list) 
- **Left/Right**: 
  - In Directories pane: Collapse/expand directories
  - In Files pane: Resize panes (make left pane smaller/larger)
- **Enter**: 
  - In Directories: Expand/collapse directory
  - In Files List: View file content
- **ESC**: Return from file content view to file list
- **[A]ccept**: Mark current file as valid Open Source match with optional comment
- **[I]gnore**: Mark current file as false positive with optional comment
- **T**: Toggle hide/show processed files (both identified and ignored)
- **q** or **Ctrl+C**: Quit application

## Audit Process

1. Navigate to a file using the tree or file list
2. Press **[A]** to accept the match or **[I]** to ignore it
3. A compact modal appears with:
   - Line 1: "Comment (Optional)" label
   - Lines 2-3: Text entry area for optional assessment comment
   - Line 4: "ENTER: Accept/Ignore  ESC: Cancel"
4. Type your optional comment (or leave blank)
5. Press **Enter** to save the decision or **ESC** to cancel

Audit decisions are saved directly to the original JSON file in an `audit` array for each file match.

## Data Structure

The tool expects SCANOSS JSON format with the following key fields:
- `id`: "file", "snippet", or "none"
- `file`: File path
- `file_url`: URL to fetch file content
- `oss_lines`: Line ranges for snippet matches
- `purl`: Package URL identifiers
- `licenses`: License information
- `audit`: Array of audit decisions (added by this tool)

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

- `main.go`: Application entry point and core logic
- `models.go`: Data structures for SCANOSS JSON format
- `tree.go`: File tree navigation and display
- `filelist.go`: File listing and content viewing
- `status.go`: Status bar implementation  
- `audit.go`: Audit decision functionality