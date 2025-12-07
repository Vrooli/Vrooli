// Package config provides configuration loading for the playbooks phase.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	// DefaultBASEndpoint is the default BAS API endpoint.
	DefaultBASEndpoint = "http://127.0.0.1:8080/api/v1"
	// DefaultBASTimeoutMs is the default HTTP client timeout in milliseconds.
	DefaultBASTimeoutMs = 30000
	// DefaultBASLaunchTimeoutMs is the default browser launch timeout in milliseconds.
	DefaultBASLaunchTimeoutMs = 60000
	// DefaultHealthCheckTimeoutMs is the default health check timeout in milliseconds.
	DefaultHealthCheckTimeoutMs = 5000
	// DefaultHealthCheckWaitTimeoutMs is how long to wait for BAS to become healthy.
	DefaultHealthCheckWaitTimeoutMs = 45000
	// DefaultWorkflowExecutionTimeoutMs is the default workflow execution timeout.
	DefaultWorkflowExecutionTimeoutMs = 180000 // 3 minutes
	// DefaultStepTimeoutMs is the default timeout for individual workflow steps.
	DefaultStepTimeoutMs = 30000
	// DefaultSeedTimeoutMs is the default timeout for seed script execution.
	DefaultSeedTimeoutMs = 30000
)

// Config holds playbooks phase configuration.
type Config struct {
	Enabled   bool            `json:"enabled"`
	BAS       BASConfig       `json:"bas"`
	Seeds     SeedsConfig     `json:"seeds"`
	Artifacts ArtifactsConfig `json:"artifacts"`
	Execution ExecutionConfig `json:"execution"`
}

// BASConfig holds Vrooli Ascension connection settings.
type BASConfig struct {
	Endpoint        string `json:"endpoint"`
	TimeoutMs       int    `json:"timeout_ms"`
	LaunchTimeoutMs int    `json:"launch_timeout_ms"`
	// Derived timeouts (not in JSON, computed from TimeoutMs)
	HealthCheckTimeoutMs     int `json:"-"`
	HealthCheckWaitTimeoutMs int `json:"-"`
	WorkflowExecutionTimeout int `json:"-"`
}

// SeedsConfig holds seed script execution settings.
type SeedsConfig struct {
	Enabled   bool `json:"enabled"`
	Cleanup   bool `json:"cleanup"`
	TimeoutMs int  `json:"timeout_ms"`
}

// ArtifactsConfig holds artifact collection settings.
type ArtifactsConfig struct {
	Screenshots     bool   `json:"screenshots"`
	DOMSnapshots    bool   `json:"dom_snapshots"`
	OutputDir       string `json:"output_dir"`
	RetainOnSuccess bool   `json:"retain_on_success"`
}

// ExecutionConfig holds workflow execution settings.
type ExecutionConfig struct {
	StopOnFirstFailure     bool           `json:"stop_on_first_failure"`
	DefaultStepTimeoutMs   int            `json:"default_step_timeout_ms"`
	Viewport               ViewportConfig `json:"viewport"`
	IgnoreValidationErrors bool           `json:"ignore_validation_errors"` // If true, continue execution even when BAS validation fails
	DryRun                 bool           `json:"dry_run"`                  // If true, validate workflows without executing them
}

// ViewportConfig holds default browser viewport settings.
type ViewportConfig struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// Default returns a Config with default values.
func Default() *Config {
	return &Config{
		Enabled: true,
		BAS: BASConfig{
			Endpoint:                 DefaultBASEndpoint,
			TimeoutMs:                DefaultBASTimeoutMs,
			LaunchTimeoutMs:          DefaultBASLaunchTimeoutMs,
			HealthCheckTimeoutMs:     DefaultHealthCheckTimeoutMs,
			HealthCheckWaitTimeoutMs: DefaultHealthCheckWaitTimeoutMs,
			WorkflowExecutionTimeout: DefaultWorkflowExecutionTimeoutMs,
		},
		Seeds: SeedsConfig{
			Enabled:   true,
			Cleanup:   true,
			TimeoutMs: DefaultSeedTimeoutMs,
		},
		Artifacts: ArtifactsConfig{
			Screenshots:     true,
			DOMSnapshots:    true,
			OutputDir:       "test/artifacts/playbooks",
			RetainOnSuccess: false,
		},
		Execution: ExecutionConfig{
			StopOnFirstFailure:   false,
			DefaultStepTimeoutMs: DefaultStepTimeoutMs,
			Viewport: ViewportConfig{
				Width:  1440,
				Height: 900,
			},
		},
	}
}

// Timeout returns the BAS HTTP client timeout as a Duration.
func (c *BASConfig) Timeout() time.Duration {
	if c.TimeoutMs <= 0 {
		return time.Duration(DefaultBASTimeoutMs) * time.Millisecond
	}
	return time.Duration(c.TimeoutMs) * time.Millisecond
}

// LaunchTimeout returns the browser launch timeout as a Duration.
func (c *BASConfig) LaunchTimeout() time.Duration {
	if c.LaunchTimeoutMs <= 0 {
		return time.Duration(DefaultBASLaunchTimeoutMs) * time.Millisecond
	}
	return time.Duration(c.LaunchTimeoutMs) * time.Millisecond
}

// HealthCheckTimeout returns the health check timeout as a Duration.
func (c *BASConfig) HealthCheckTimeout() time.Duration {
	if c.HealthCheckTimeoutMs <= 0 {
		return time.Duration(DefaultHealthCheckTimeoutMs) * time.Millisecond
	}
	return time.Duration(c.HealthCheckTimeoutMs) * time.Millisecond
}

// HealthCheckWaitTimeout returns how long to wait for BAS to become healthy.
func (c *BASConfig) HealthCheckWaitTimeout() time.Duration {
	if c.HealthCheckWaitTimeoutMs <= 0 {
		return time.Duration(DefaultHealthCheckWaitTimeoutMs) * time.Millisecond
	}
	return time.Duration(c.HealthCheckWaitTimeoutMs) * time.Millisecond
}

// WorkflowTimeout returns the workflow execution timeout as a Duration.
func (c *BASConfig) WorkflowTimeout() time.Duration {
	if c.WorkflowExecutionTimeout <= 0 {
		return time.Duration(DefaultWorkflowExecutionTimeoutMs) * time.Millisecond
	}
	return time.Duration(c.WorkflowExecutionTimeout) * time.Millisecond
}

// SeedTimeout returns the seed script timeout as a Duration.
func (c *SeedsConfig) SeedTimeout() time.Duration {
	if c.TimeoutMs <= 0 {
		return time.Duration(DefaultSeedTimeoutMs) * time.Millisecond
	}
	return time.Duration(c.TimeoutMs) * time.Millisecond
}

// StepTimeout returns the default step timeout as a Duration.
func (c *ExecutionConfig) StepTimeout() time.Duration {
	if c.DefaultStepTimeoutMs <= 0 {
		return time.Duration(DefaultStepTimeoutMs) * time.Millisecond
	}
	return time.Duration(c.DefaultStepTimeoutMs) * time.Millisecond
}

// testingJSON represents the structure of .vrooli/testing.json.
type testingJSON struct {
	Playbooks *Config `json:"playbooks"`
}

// Load reads playbooks configuration from .vrooli/testing.json.
// If the file doesn't exist or has no playbooks section, defaults are used.
func Load(scenarioDir string) (*Config, error) {
	cfg := Default()

	testingPath := filepath.Join(scenarioDir, ".vrooli", "testing.json")
	data, err := os.ReadFile(testingPath)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil // No testing.json, use defaults
		}
		return nil, fmt.Errorf("failed to read testing.json: %w", err)
	}

	var tj testingJSON
	if err := json.Unmarshal(data, &tj); err != nil {
		return nil, fmt.Errorf("failed to parse testing.json: %w", err)
	}

	if tj.Playbooks == nil {
		return cfg, nil // No playbooks section, use defaults
	}

	// Merge loaded config with defaults (loaded values take precedence)
	loaded := tj.Playbooks

	// Only override if explicitly set (non-zero)
	if loaded.BAS.Endpoint != "" {
		cfg.BAS.Endpoint = loaded.BAS.Endpoint
	}
	if loaded.BAS.TimeoutMs > 0 {
		cfg.BAS.TimeoutMs = loaded.BAS.TimeoutMs
	}
	if loaded.BAS.LaunchTimeoutMs > 0 {
		cfg.BAS.LaunchTimeoutMs = loaded.BAS.LaunchTimeoutMs
	}

	// Seeds config - only override if section exists
	if loaded.Seeds.TimeoutMs > 0 {
		cfg.Seeds.TimeoutMs = loaded.Seeds.TimeoutMs
	}
	// For booleans, we need to check if they were explicitly set
	// Since we can't distinguish false from unset in JSON, we'll accept the loaded values
	cfg.Seeds.Enabled = loaded.Seeds.Enabled
	cfg.Seeds.Cleanup = loaded.Seeds.Cleanup

	// Artifacts config
	if loaded.Artifacts.OutputDir != "" {
		cfg.Artifacts.OutputDir = loaded.Artifacts.OutputDir
	}
	cfg.Artifacts.Screenshots = loaded.Artifacts.Screenshots
	cfg.Artifacts.DOMSnapshots = loaded.Artifacts.DOMSnapshots
	cfg.Artifacts.RetainOnSuccess = loaded.Artifacts.RetainOnSuccess

	// Execution config
	if loaded.Execution.DefaultStepTimeoutMs > 0 {
		cfg.Execution.DefaultStepTimeoutMs = loaded.Execution.DefaultStepTimeoutMs
	}
	cfg.Execution.StopOnFirstFailure = loaded.Execution.StopOnFirstFailure
	cfg.Execution.IgnoreValidationErrors = loaded.Execution.IgnoreValidationErrors
	cfg.Execution.DryRun = loaded.Execution.DryRun
	if loaded.Execution.Viewport.Width > 0 {
		cfg.Execution.Viewport.Width = loaded.Execution.Viewport.Width
	}
	if loaded.Execution.Viewport.Height > 0 {
		cfg.Execution.Viewport.Height = loaded.Execution.Viewport.Height
	}

	// Top-level enabled flag
	cfg.Enabled = loaded.Enabled

	return cfg, nil
}
