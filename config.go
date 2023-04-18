package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/bubbles/list"
)

type prefix struct {
	T string `json:"title"`
	D string `json:"description"`
}

type config struct {
	Prefixes             []prefix `json:"prefixes"`
	SignOffCommits       bool     `json:"signOffCommits"`
	ScopeInputCharLimit  int      `json:"scopeInputCharLimit"`
	CommitInputCharLimit int      `json:"commitInputCharLimit"`
	TotalInputCharLimit  int      `json:"totalInputCharLimit"`
}

func (i prefix) Title() string       { return i.T }
func (i prefix) Description() string { return i.D }
func (i prefix) FilterValue() string { return i.T }

var defaultPrefixes = []list.Item{
	prefix{
		T: "feat",
		D: "Introduces a new feature",
	},
	prefix{
		T: "fix",
		D: "Patches a bug",
	},
	prefix{
		T: "docs",
		D: "Documentation changes only",
	},
	prefix{
		T: "test",
		D: "Adding missing tests or correcting existing tests",
	},
	prefix{
		T: "build",
		D: "Changes that affect the build system",
	},
	prefix{
		T: "ci",
		D: "Changes to CI configuration files and scripts",
	},
	prefix{
		T: "perf",
		D: "A code change that improves performance",
	},
	prefix{
		T: "refactor",
		D: "A code change that neither fixes a bug nor adds a feature",
	},
	prefix{
		T: "revert",
		D: "Reverts a previous change",
	},
	prefix{
		T: "style",
		D: "Changes that do not affect the meaning of the code (white-space, formatting, missing semi-colons, etc)",
	},
	prefix{
		T: "chore",
		D: "A minor change which does not fit into any other category",
	},
}

const applicationName = "cometary"

func loadConfig() ([]list.Item, bool, *config, error) {
	nonXdgConfigFile := ".comet.json"

	// Check for configuration file local to current directory
	if _, err := os.Stat(nonXdgConfigFile); err == nil {
		return loadConfigFile(nonXdgConfigFile)
	}

	// Check for configuration file local to user's home directory
	if home, err := os.UserHomeDir(); err == nil {
		path := filepath.Join(home, nonXdgConfigFile)
		if _, err := os.Stat(path); err == nil {
			return loadConfigFile(path)
		}
	}

	// Check for configuration file according to XDG Base Directory Specification
	if cfgDir, err := getConfigDir(); err == nil {
		path := filepath.Join(cfgDir, "config.json")
		if _, err := os.Stat(path); err == nil {
			return loadConfigFile(path)
		}
	}

	return defaultPrefixes, false, nil, nil
}

func getConfigDir() (string, error) {
	configDir := os.Getenv("XDG_CONFIG_HOME")

	// If the value of the environment variable is unset, empty, or not an absolute path, use the default
	if configDir == "" || configDir[0:1] != "/" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(homeDir, ".config", applicationName), nil
	}

	// The value of the environment variable is valid; use it
	return filepath.Join(configDir, applicationName), nil
}

func loadConfigFile(path string) ([]list.Item, bool, *config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false, nil, fmt.Errorf("failed to read config file: %w", err)
	}
	var c config
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, false, nil, fmt.Errorf("invalid json in config file '%s': %w", path, err)
	}

	return convertPrefixes(c.Prefixes), c.SignOffCommits, &c, nil
}

func convertPrefixes(prefixes []prefix) []list.Item {
	var output []list.Item
	for _, prefix := range prefixes {
		output = append(output, prefix)
	}
	if len(output) == 0 {
		return defaultPrefixes
	}
	return output
}
