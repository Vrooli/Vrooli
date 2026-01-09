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
	ProxyURL           string    `json:"proxy_url,omitempty"`
	ServerURL          string    `json:"server_url,omitempty"`
	APIURL             string    `json:"api_url,omitempty"`
	AppDisplayName     string    `json:"app_display_name,omitempty"`
	AppDescription     string    `json:"app_description,omitempty"`
	Icon               string    `json:"icon,omitempty"`
	DeploymentMode     string    `json:"deployment_mode,omitempty"`
	AutoManageVrooli   bool      `json:"auto_manage_vrooli"`
	VrooliBinary       string    `json:"vrooli_binary_path,omitempty"`
	ServerType         string    `json:"server_type,omitempty"`
	BundleManifestPath string    `json:"bundle_manifest_path,omitempty"`
	UpdatedAt          time.Time `json:"updated_at"`
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
	var legacy struct {
		AutoManageTier1 *bool  `json:"auto_manage_tier1"`
		ProxyURL        string `json:"proxy_url"`
	}
	_ = json.Unmarshal(data, &legacy)
	if cfg.ProxyURL == "" {
		if legacy.ProxyURL != "" {
			cfg.ProxyURL = legacy.ProxyURL
		} else if cfg.ServerURL != "" {
			cfg.ProxyURL = cfg.ServerURL
		}
	}
	if !cfg.AutoManageVrooli && legacy.AutoManageTier1 != nil {
		cfg.AutoManageVrooli = *legacy.AutoManageTier1
	}
	if cfg.ProxyURL == "" && cfg.APIURL != "" {
		cfg.ProxyURL = cfg.APIURL
	}
	if cfg.ProxyURL != "" {
		if normalized, err := normalizeProxyURL(cfg.ProxyURL); err == nil {
			cfg.ProxyURL = normalized
		}
		cfg.ServerURL = cfg.ProxyURL
		cfg.APIURL = proxyAPIURL(cfg.ProxyURL)
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

	if cfg.ProxyURL != "" {
		if normalized, err := normalizeProxyURL(cfg.ProxyURL); err == nil {
			cfg.ProxyURL = normalized
		}
		cfg.ServerURL = cfg.ProxyURL
		cfg.APIURL = proxyAPIURL(cfg.ProxyURL)
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
