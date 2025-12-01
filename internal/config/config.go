package config

import (
	"encoding/json"
	"fmt"
	"os"
)

type Config struct {
	Username  string `json:"username"`
	FullName  string `json:"name"`
	Directory string `json:"directory"` // e.g., /home/username
	Shell     string `json:"shell"`
	Timezone  string `json:"timezone"`
	Title     string `json:"title"`
}

// DefaultConfig returns the default configuration based on environment variables
func DefaultConfig() Config {
	user := os.Getenv("USER")
	home := os.Getenv("HOME")
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/bash"
	}

	return Config{
		Username:  user,
		FullName:  user, // Fallback
		Directory: home,
		Shell:     shell,
		Timezone:  "America/Los_Angeles", // Default fallback
		Title:     "Plan",
	}
}

// Load reads settings.json and overlays it on top of defaults
func Load(path string) (Config, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return cfg, nil // No settings file, just return defaults
	}
	if err != nil {
		return cfg, fmt.Errorf("reading settings file: %w", err)
	}

	// Unmarshal into a temporary map or directly into struct? 
	// To preserve defaults for missing fields, we decode into the struct.
	// JSON unmarshal doesn't reset fields that are missing in JSON.
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parsing settings file: %w", err)
	}

	return cfg, nil
}
