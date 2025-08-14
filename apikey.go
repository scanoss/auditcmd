// Copyright (c) 2025 SCANOSS
// SPDX-License-Identifier: MIT

package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"golang.org/x/term"
)

const configFileName = ".auditcmd"

func getConfigFilePath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return configFileName
	}
	return filepath.Join(homeDir, configFileName)
}

type Config struct {
	APIKey        string
	PaneWidth     float64
	HideIdentified bool
}

func loadConfig() (*Config, error) {
	configPath := getConfigFilePath()
	
	// Default config
	config := &Config{
		APIKey:        "",
		PaneWidth:     0.5,
		HideIdentified: false,
	}
	
	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return config, nil // Return default config
	}
	
	// Read the config file
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return config, fmt.Errorf("failed to read config file: %v", err)
	}
	
	// Parse INI format
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		
		if strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			
			switch key {
			case "api_key":
				config.APIKey = value
			case "pane_width":
				if width, err := strconv.ParseFloat(value, 64); err == nil {
					config.PaneWidth = width
				}
			case "hide_identified":
				if hide, err := strconv.ParseBool(value); err == nil {
					config.HideIdentified = hide
				}
			}
		}
	}
	
	return config, nil
}

func loadAPIKey() (string, error) {
	config, err := loadConfig()
	if err != nil {
		return "", err
	}
	
	if config.APIKey == "" {
		return "", fmt.Errorf("API key not found")
	}
	
	return config.APIKey, nil
}

func saveConfig(config *Config) error {
	configPath := getConfigFilePath()
	
	// Create directory if it doesn't exist
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %v", err)
	}
	
	// Create INI content
	content := "# AuditCmd Configuration\n"
	content += "# This file stores settings for the AuditCmd application\n\n"
	content += fmt.Sprintf("api_key=%s\n", config.APIKey)
	content += fmt.Sprintf("pane_width=%.2f\n", config.PaneWidth)
	content += fmt.Sprintf("hide_identified=%t\n", config.HideIdentified)
	
	// Write config to file with secure permissions
	err := ioutil.WriteFile(configPath, []byte(content), 0600)
	if err != nil {
		return fmt.Errorf("failed to save config: %v", err)
	}
	
	return nil
}

func saveAPIKey(apiKey string) error {
	// Load existing config
	config, _ := loadConfig()
	config.APIKey = apiKey
	
	return saveConfig(config)
}

func promptForAPIKey() (string, error) {
	fmt.Println()
	fmt.Println("SCANOSS API Key Required")
	fmt.Println("========================")
	fmt.Println("An API key is required to fetch and display file contents from SCANOSS.")
	fmt.Println("Without an API key, you can still:")
	fmt.Println("  • Navigate the directory tree")
	fmt.Println("  • View file lists and audit status")
	fmt.Println("  • Make audit decisions (IDENTIFY/IGNORE)")
	fmt.Println("  • Save audit results to JSON")
	fmt.Println()
	fmt.Println("But you CANNOT:")
	fmt.Println("  • View actual file contents")
	fmt.Println("  • See highlighted snippet matches")
	fmt.Println()
	
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("Enter your SCANOSS API key (or 'skip' to continue without): ")
		
		// Try to read securely first
		byteInput, err := term.ReadPassword(int(syscall.Stdin))
		var input string
		
		if err != nil {
			// Fallback to regular input if terminal doesn't support hidden input
			fmt.Print("\n[Visible input] API key or 'skip': ")
			input, err = reader.ReadString('\n')
			if err != nil {
				return "", fmt.Errorf("failed to read input: %v", err)
			}
			input = strings.TrimSpace(input)
		} else {
			fmt.Println() // Print newline after hidden input
			input = strings.TrimSpace(string(byteInput))
		}
		
		if strings.ToLower(input) == "skip" {
			fmt.Println("Continuing without API key. File contents will not be available.")
			return "", nil // Return empty string to indicate skipped
		}
		
		if input == "" {
			fmt.Println("Please enter an API key or 'skip' to continue without one.")
			continue
		}
		
		return input, nil
	}
}

func getOrPromptAPIKey() (string, error) {
	// Try to load existing API key
	apiKey, err := loadAPIKey()
	if err == nil {
		return apiKey, nil
	}
	
	// If not found, prompt user
	fmt.Printf("Error loading API key: %v\n", err)
	apiKey, err = promptForAPIKey()
	if err != nil {
		return "", err
	}
	
	// Save the API key for future use
	if err := saveAPIKey(apiKey); err != nil {
		fmt.Printf("Warning: failed to save API key: %v\n", err)
		// Continue anyway, we have the key for this session
	} else {
		fmt.Println("API key saved to", getConfigFilePath())
	}
	
	return apiKey, nil
}

func savePaneWidth(width float64) error {
	// Load existing config
	config, _ := loadConfig()
	config.PaneWidth = width
	
	return saveConfig(config)
}

func loadPaneWidth() float64 {
	config, _ := loadConfig()
	return config.PaneWidth
}

func saveHideIdentified(hideIdentified bool) error {
	// Load existing config
	config, _ := loadConfig()
	config.HideIdentified = hideIdentified
	
	return saveConfig(config)
}

func loadHideIdentified() bool {
	config, _ := loadConfig()
	return config.HideIdentified
}

// validateAPIKey tests the API key by making a simple request
func validateAPIKey(apiKey string) error {
	// This could be enhanced to make a test API call
	if len(apiKey) < 10 {
		return fmt.Errorf("API key appears to be too short (minimum 10 characters)")
	}
	return nil
}