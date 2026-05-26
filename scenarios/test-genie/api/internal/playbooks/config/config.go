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
	// DefaultSeedTimeoutMs is the default timeout for seed script execution.
	DefaultSeedTimeoutMs = 30000
)

// Config holds playbooks phase configuration.
type Config struct {
	Enabled   bool            `json:"enabled"`
	BAS       BASConfig       `json:"bas"`
	Seeds     SeedsConfig     `json:"seeds"`
	Artifacts ArtifactsConfig `json:"artifacts"`

	// Diagnostics selects which rich artifacts (video, trace, HAR, etc.) BAS
	// captures during workflow execution. Defaults are cheap (console only).
	Diagnostics DiagnosticsConfig `json:"diagnostics"`

	// ContinueOnFailure runs all selected workflows even after one fails,
	// instead of stopping at the first failure. Default false.
	ContinueOnFailure bool `json:"continue_on_failure"`

	// WorkflowFilter, when non-empty, restricts execution to the named
	// workflows (by file/registry key). Empty means run all.
	WorkflowFilter []string `json:"workflow_filter"`

	// AllowEmptyTestPool opts the scenario out of the hard-fail check on
	// routed runs where zero requests exercised the test pool. Default false:
	// a routed run that never hits the test DB is treated as
	// FailureClassMisconfiguration. Set true for playbooks that legitimately
	// never read or write through the API (pure UI navigation, static-page
	// smoke checks, etc).
	AllowEmptyTestPool bool `json:"allow_empty_test_pool"`
}

// BASConfig holds Vrooli Ascension connection settings.
type BASConfig struct {
	Endpoint                 string `json:"endpoint"`
	TimeoutMs                int    `json:"timeout_ms"`
	LaunchTimeoutMs          int    `json:"launch_timeout_ms"`
	HealthCheckTimeoutMs     int    `json:"health_check_timeout_ms"`
	HealthCheckWaitTimeoutMs int    `json:"health_check_wait_timeout_ms"`
	WorkflowExecutionTimeout int    `json:"workflow_execution_timeout_ms"`
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

// DiagnosticsConfig toggles the rich diagnostic artifacts BAS captures per
// workflow. These map onto BAS ExecuteWorkflowOptions and ArtifactCollectionConfig.
type DiagnosticsConfig struct {
	Video   bool `json:"video"`   // Playwright recordVideo (default false)
	Console bool `json:"console"` // console logs (default true; cheap)
	Network bool `json:"network"` // network events (default false)
	HAR     bool `json:"har"`     // HAR archive (default false)
	Trace   bool `json:"trace"`   // Playwright trace (default false)
	DOM     bool `json:"dom"`     // DOM snapshots (default false)
}

// Diagnostics preset names accepted by the --diagnostics-preset CLI flag.
const (
	DiagnosticsPresetNone  = "none"
	DiagnosticsPresetLight = "light"
	DiagnosticsPresetFull  = "full"
)

// DiagnosticsPreset maps a preset name to a DiagnosticsConfig. "light" captures
// console only (screenshots/DOM are governed by ArtifactsConfig); "full"
// captures everything; "none" captures nothing. The bool reports whether the
// name was recognized.
func DiagnosticsPreset(name string) (DiagnosticsConfig, bool) {
	switch name {
	case DiagnosticsPresetNone:
		return DiagnosticsConfig{}, true
	case DiagnosticsPresetLight, "":
		return DiagnosticsConfig{Console: true}, true
	case DiagnosticsPresetFull:
		return DiagnosticsConfig{Video: true, Console: true, Network: true, HAR: true, Trace: true, DOM: true}, true
	default:
		return DiagnosticsConfig{}, false
	}
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
			OutputDir:       "coverage/automation",
			RetainOnSuccess: false,
		},
		Diagnostics: DiagnosticsConfig{
			Console: true,
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

// testingJSON represents the structure of .vrooli/testing.json.
type testingJSON struct {
	Playbooks *rawConfig `json:"playbooks"`
}

// rawConfig is the JSON shape for playbooks config that preserves "unset" vs "false"
// for booleans via pointer fields. Config is the merged runtime config.
type rawConfig struct {
	Enabled            *bool           `json:"enabled"`
	BAS                *rawBASConfig   `json:"bas"`
	Seeds              *rawSeedsConfig `json:"seeds"`
	Artifacts          *rawArtifacts   `json:"artifacts"`
	Diagnostics        *rawDiagnostics `json:"diagnostics"`
	ContinueOnFailure  *bool           `json:"continue_on_failure"`
	WorkflowFilter     []string        `json:"workflow_filter"`
	AllowEmptyTestPool *bool           `json:"allow_empty_test_pool"`
}

type rawDiagnostics struct {
	Video   *bool `json:"video"`
	Console *bool `json:"console"`
	Network *bool `json:"network"`
	HAR     *bool `json:"har"`
	Trace   *bool `json:"trace"`
	DOM     *bool `json:"dom"`
}

type rawBASConfig struct {
	Endpoint                 string `json:"endpoint"`
	TimeoutMs                int    `json:"timeout_ms"`
	LaunchTimeoutMs          int    `json:"launch_timeout_ms"`
	HealthCheckTimeoutMs     int    `json:"health_check_timeout_ms"`
	HealthCheckWaitTimeoutMs int    `json:"health_check_wait_timeout_ms"`
	WorkflowExecutionTimeout int    `json:"workflow_execution_timeout_ms"`
}

type rawSeedsConfig struct {
	Enabled   *bool `json:"enabled"`
	Cleanup   *bool `json:"cleanup"`
	TimeoutMs int   `json:"timeout_ms"`
}

type rawArtifacts struct {
	Screenshots     *bool  `json:"screenshots"`
	DOMSnapshots    *bool  `json:"dom_snapshots"`
	OutputDir       string `json:"output_dir"`
	RetainOnSuccess *bool  `json:"retain_on_success"`
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

	loaded := tj.Playbooks

	// Top-level enabled flag (only when explicitly set)
	if loaded.Enabled != nil {
		cfg.Enabled = *loaded.Enabled
	}

	if loaded.AllowEmptyTestPool != nil {
		cfg.AllowEmptyTestPool = *loaded.AllowEmptyTestPool
	}

	if loaded.ContinueOnFailure != nil {
		cfg.ContinueOnFailure = *loaded.ContinueOnFailure
	}

	if loaded.WorkflowFilter != nil {
		cfg.WorkflowFilter = loaded.WorkflowFilter
	}

	// Diagnostics config (only override fields that are explicitly set)
	if loaded.Diagnostics != nil {
		d := loaded.Diagnostics
		if d.Video != nil {
			cfg.Diagnostics.Video = *d.Video
		}
		if d.Console != nil {
			cfg.Diagnostics.Console = *d.Console
		}
		if d.Network != nil {
			cfg.Diagnostics.Network = *d.Network
		}
		if d.HAR != nil {
			cfg.Diagnostics.HAR = *d.HAR
		}
		if d.Trace != nil {
			cfg.Diagnostics.Trace = *d.Trace
		}
		if d.DOM != nil {
			cfg.Diagnostics.DOM = *d.DOM
		}
	}

	// BAS config
	if loaded.BAS != nil {
		if loaded.BAS.Endpoint != "" {
			cfg.BAS.Endpoint = loaded.BAS.Endpoint
		}
		if loaded.BAS.TimeoutMs > 0 {
			cfg.BAS.TimeoutMs = loaded.BAS.TimeoutMs
		}
		if loaded.BAS.LaunchTimeoutMs > 0 {
			cfg.BAS.LaunchTimeoutMs = loaded.BAS.LaunchTimeoutMs
		}
		if loaded.BAS.HealthCheckTimeoutMs > 0 {
			cfg.BAS.HealthCheckTimeoutMs = loaded.BAS.HealthCheckTimeoutMs
		}
		if loaded.BAS.HealthCheckWaitTimeoutMs > 0 {
			cfg.BAS.HealthCheckWaitTimeoutMs = loaded.BAS.HealthCheckWaitTimeoutMs
		}
		if loaded.BAS.WorkflowExecutionTimeout > 0 {
			cfg.BAS.WorkflowExecutionTimeout = loaded.BAS.WorkflowExecutionTimeout
		}
	}

	// Artifacts config
	if loaded.Artifacts != nil {
		if loaded.Artifacts.OutputDir != "" {
			cfg.Artifacts.OutputDir = loaded.Artifacts.OutputDir
		}
		if loaded.Artifacts.Screenshots != nil {
			cfg.Artifacts.Screenshots = *loaded.Artifacts.Screenshots
		}
		if loaded.Artifacts.DOMSnapshots != nil {
			cfg.Artifacts.DOMSnapshots = *loaded.Artifacts.DOMSnapshots
		}
		if loaded.Artifacts.RetainOnSuccess != nil {
			cfg.Artifacts.RetainOnSuccess = *loaded.Artifacts.RetainOnSuccess
		}
	}

	// Seeds config
	if loaded.Seeds != nil {
		if loaded.Seeds.TimeoutMs > 0 {
			cfg.Seeds.TimeoutMs = loaded.Seeds.TimeoutMs
		}
		if loaded.Seeds.Enabled != nil {
			cfg.Seeds.Enabled = *loaded.Seeds.Enabled
		}
		if loaded.Seeds.Cleanup != nil {
			cfg.Seeds.Cleanup = *loaded.Seeds.Cleanup
		}
	}

	return cfg, nil
}
