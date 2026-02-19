package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"gopkg.in/yaml.v3"
)

// LocalConfigManager handles reading, writing, and backing up cloudflared config. [REQ:LOCAL-001]
type LocalConfigManager struct {
	configPath string
	cmdRunner  func(ctx context.Context, name string, args ...string) ([]byte, error)
}

type LocalConfigOption func(*LocalConfigManager)

func WithLocalConfigPath(path string) LocalConfigOption {
	return func(m *LocalConfigManager) { m.configPath = path }
}

func WithLocalConfigCmdRunner(fn func(ctx context.Context, name string, args ...string) ([]byte, error)) LocalConfigOption {
	return func(m *LocalConfigManager) { m.cmdRunner = fn }
}

func NewLocalConfigManager(opts ...LocalConfigOption) *LocalConfigManager {
	m := &LocalConfigManager{
		configPath: filepath.Join(os.Getenv("HOME"), ".cloudflared", "config.yml"),
		cmdRunner:  defaultCmdRunner,
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

// Parse reads and parses the cloudflared config.yml. [REQ:LOCAL-001]
func (m *LocalConfigManager) Parse() (*CloudflaredConfig, error) {
	data, err := os.ReadFile(m.configPath)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg CloudflaredConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	return &cfg, nil
}

// GenerateFromRoutes creates a CloudflaredConfig from routes. [REQ:LOCAL-002]
// It preserves non-ingress settings from the existing config if available.
func (m *LocalConfigManager) GenerateFromRoutes(routes []Route, existing *CloudflaredConfig) *CloudflaredConfig {
	cfg := &CloudflaredConfig{}

	// Preserve non-ingress settings
	if existing != nil {
		cfg.Tunnel = existing.Tunnel
		cfg.CredentialsFile = existing.CredentialsFile
		cfg.WarpRouting = existing.WarpRouting
	}

	// Build ingress rules from routes
	for _, r := range routes {
		if !r.Enabled {
			continue
		}
		rule := CloudflaredIngress{
			Hostname: r.Subdomain + ".vrooli.com",
			Service:  fmt.Sprintf("http://localhost:%d", r.LocalPort),
		}
		if r.PublicURL != "" {
			rule.Hostname = extractHostname(r.PublicURL)
		}
		cfg.Ingress = append(cfg.Ingress, rule)
	}

	// Add catch-all 404 rule
	cfg.Ingress = append(cfg.Ingress, CloudflaredIngress{
		Service: "http_status:404",
	})

	return cfg
}

// WriteWithBackup writes the config to disk, creating a backup first. [REQ:LOCAL-003]
// Keeps the last maxBackups backups.
func (m *LocalConfigManager) WriteWithBackup(cfg *CloudflaredConfig, maxBackups int) error {
	if maxBackups <= 0 {
		maxBackups = 5
	}

	// Create backup if existing config exists
	if _, err := os.Stat(m.configPath); err == nil {
		backupName := fmt.Sprintf("%s.backup.%s", m.configPath, time.Now().Format("20060102-150405"))
		data, err := os.ReadFile(m.configPath)
		if err != nil {
			return fmt.Errorf("read for backup: %w", err)
		}
		if err := os.WriteFile(backupName, data, 0o644); err != nil {
			return fmt.Errorf("write backup: %w", err)
		}

		// Prune old backups
		if err := m.pruneBackups(maxBackups); err != nil {
			return fmt.Errorf("prune backups: %w", err)
		}
	}

	// Marshal and write new config
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	dir := filepath.Dir(m.configPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	if err := os.WriteFile(m.configPath, data, 0o644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	return nil
}

// RestartCloudflared restarts the cloudflared service and verifies readiness. [REQ:LOCAL-004]
func (m *LocalConfigManager) RestartCloudflared(ctx context.Context) error {
	_, err := m.cmdRunner(ctx, "sudo", "systemctl", "restart", "cloudflared")
	if err != nil {
		return fmt.Errorf("restart cloudflared: %w", err)
	}
	return pollReady(ctx, 30*time.Second, 1*time.Second)
}

func (m *LocalConfigManager) pruneBackups(keep int) error {
	dir := filepath.Dir(m.configPath)
	base := filepath.Base(m.configPath)
	pattern := base + ".backup.*"

	matches, err := filepath.Glob(filepath.Join(dir, pattern))
	if err != nil {
		return err
	}

	if len(matches) <= keep {
		return nil
	}

	// Sort oldest first
	sort.Strings(matches)

	// Remove oldest, keeping 'keep' most recent
	for _, f := range matches[:len(matches)-keep] {
		os.Remove(f)
	}

	return nil
}
