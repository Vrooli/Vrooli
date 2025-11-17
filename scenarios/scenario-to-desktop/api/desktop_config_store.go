package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// DesktopConnectionConfig captures the operator-provided connection info for a scenario's desktop build.
type DesktopConnectionConfig struct {
	ServerURL       string    `json:"server_url,omitempty"`
	APIURL          string    `json:"api_url,omitempty"`
	DeploymentMode  string    `json:"deployment_mode,omitempty"`
	AutoManageTier1 bool      `json:"auto_manage_tier1"`
	VrooliBinary    string    `json:"vrooli_binary_path,omitempty"`
	ServerType      string    `json:"server_type,omitempty"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func desktopConfigPath(scenarioRoot string) string {
	return filepath.Join(scenarioRoot, ".vrooli", "desktop-config.json")
}

func loadDesktopConnectionConfig(scenarioRoot string) (*DesktopConnectionConfig, error) {
	if scenarioRoot == "" {
		return nil, nil
	}

	path := desktopConfigPath(scenarioRoot)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read desktop config: %w", err)
	}

	var cfg DesktopConnectionConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse desktop config: %w", err)
	}
	return &cfg, nil
}

func saveDesktopConnectionConfig(scenarioRoot string, cfg *DesktopConnectionConfig) error {
	if scenarioRoot == "" || cfg == nil {
		return nil
	}

	path := desktopConfigPath(scenarioRoot)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create desktop config dir: %w", err)
	}

	cfg.UpdatedAt = time.Now().UTC()
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal desktop config: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write desktop config: %w", err)
	}

	return nil
}
