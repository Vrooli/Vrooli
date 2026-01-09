package workspace

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var timeoutPattern = regexp.MustCompile(`^([0-9]+)([smh]?)$`)

type Config struct {
	Phases       map[string]PhaseSettings
	Presets      map[string][]string
	Requirements RequirementSettings
	Integration  IntegrationSettings
}

type PhaseSettings struct {
	Enabled *bool
	Timeout time.Duration
}

// RequirementSettings mirrors the legacy bash orchestrator flags so the Go
// runner can make the same sync decisions without shell scripts.
type RequirementSettings struct {
	Enforce *bool
	Sync    *bool
}

// IntegrationSettings configures the integration phase's runtime validation.
// These settings control how API health checks and WebSocket validation are performed.
//
// WebSocket URL Resolution:
// The WebSocket URL is derived from the API URL by converting the protocol (http→ws, https→wss)
// and appending the WebSocketPath. This follows the pattern established by @vrooli/api-base
// where WebSocket connections are proxied through the same server as HTTP API requests.
// See: packages/api-base/docs/concepts/websocket-support.md
type IntegrationSettings struct {
	// API configures HTTP health check validation.
	API APISettings

	// WebSocket configures WebSocket connection validation.
	// The URL is derived from the API URL + WebSocketPath following api-base conventions.
	WebSocket WebSocketSettings

	// CLI configures CLI binary validation behavior.
	CLI CLISettings
}

// CLISettings configures CLI validation behavior.
// This allows scenarios to customize how their CLI is validated, accommodating
// different CLI frameworks (cli-core subcommands vs standard flag-based CLIs).
type CLISettings struct {
	// HelpArgs specifies argument patterns to try for help command, in order of preference.
	// The validator tries each pattern until one succeeds.
	// Default: ["help", "--help", "-h"]
	HelpArgs []string

	// VersionArgs specifies argument patterns to try for version command, in order of preference.
	// Default: ["version", "--version", "-v"]
	VersionArgs []string

	// RequireVersionKeyword controls whether version output must contain the word "version".
	// Set to false for CLIs that output just the version number (e.g., "1.2.3").
	// Default: false (relaxed check - any non-empty output passes)
	RequireVersionKeyword *bool

	// CheckUnknownCommand controls whether to verify the CLI handles unknown commands gracefully.
	// When true, runs the CLI with a nonsense argument and expects non-zero exit.
	// Default: true
	CheckUnknownCommand *bool

	// CheckNoArgs controls whether to verify the CLI handles no arguments gracefully.
	// When true, runs the CLI with no arguments and expects it to not hang and exit 0.
	// Default: true
	CheckNoArgs *bool

	// NoArgsTimeoutMs is the maximum time to wait for the no-args check in milliseconds.
	// Default: 5000 (5 seconds)
	NoArgsTimeoutMs int64
}

// APISettings configures API health check behavior.
type APISettings struct {
	// HealthEndpoint is the path to check for API health (default: "/health").
	HealthEndpoint string

	// MaxResponseMs is the maximum acceptable response time in milliseconds (default: 1000).
	MaxResponseMs int64
}

// WebSocketSettings configures WebSocket validation behavior.
// WebSocket URLs are derived from the API URL following the pattern from @vrooli/api-base:
// - Protocol is converted: http:// → ws://, https:// → wss://
// - Path is appended: {api_url}{websocket_path}
// See: packages/api-base/docs/concepts/websocket-support.md
type WebSocketSettings struct {
	// Enabled controls whether WebSocket validation runs (default: true if Path is set).
	Enabled *bool

	// Path is the WebSocket endpoint path appended to the API URL (e.g., "/api/v1/ws").
	// If empty, WebSocket validation is skipped.
	// This follows the api-base convention where WebSocket connections use the same
	// host:port as HTTP but with a different path and upgraded protocol.
	Path string

	// MaxConnectionMs is the maximum time to establish a connection in milliseconds (default: 2000).
	MaxConnectionMs int64
}

type rawTestingConfig struct {
	Phases       map[string]rawPhaseSettings `json:"phases"`
	Presets      map[string][]string         `json:"presets"`
	Requirements rawRequirementSettings      `json:"requirements"`
	Integration  rawIntegrationSettings      `json:"integration"`
}

type rawPhaseSettings struct {
	Enabled *bool  `json:"enabled"`
	Timeout string `json:"timeout"`
}

type rawRequirementSettings struct {
	Enforce *bool `json:"enforce"`
	Sync    *bool `json:"sync"`
}

type rawIntegrationSettings struct {
	API       rawAPISettings       `json:"api"`
	WebSocket rawWebSocketSettings `json:"websocket"`
	CLI       rawCLISettings       `json:"cli"`
}

type rawCLISettings struct {
	HelpArgs              []string `json:"help_args"`
	VersionArgs           []string `json:"version_args"`
	RequireVersionKeyword *bool    `json:"require_version_keyword"`
	CheckUnknownCommand   *bool    `json:"check_unknown_command"`
	CheckNoArgs           *bool    `json:"check_no_args"`
	NoArgsTimeoutMs       int64    `json:"no_args_timeout_ms"`
}

type rawAPISettings struct {
	HealthEndpoint string `json:"health_endpoint"`
	MaxResponseMs  int64  `json:"max_response_ms"`
}

type rawWebSocketSettings struct {
	Enabled         *bool  `json:"enabled"`
	Path            string `json:"path"`
	MaxConnectionMs int64  `json:"max_connection_ms"`
}

func LoadTestingConfig(scenarioDir string) (*Config, error) {
	configPath := filepath.Join(scenarioDir, ".vrooli", "testing.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read %s: %w", configPath, err)
	}

	var raw rawTestingConfig
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", configPath, err)
	}

	cfg := &Config{
		Phases:       map[string]PhaseSettings{},
		Presets:      map[string][]string{},
		Requirements: RequirementSettings{},
		Integration:  IntegrationSettings{},
	}

	for name, phase := range raw.Phases {
		normalized := strings.ToLower(strings.TrimSpace(name))
		if normalized == "" {
			continue
		}
		timeout, err := parsePhaseTimeout(phase.Timeout)
		if err != nil {
			return nil, fmt.Errorf("invalid timeout for phase '%s': %w", name, err)
		}
		cfg.Phases[normalized] = PhaseSettings{
			Enabled: phase.Enabled,
			Timeout: timeout,
		}
	}

	for preset, phases := range raw.Presets {
		normalized := strings.ToLower(strings.TrimSpace(preset))
		if normalized == "" {
			continue
		}
		filtered := normalizePhaseList(phases)
		if len(filtered) == 0 {
			continue
		}
		cfg.Presets[normalized] = filtered
	}

	cfg.Requirements = RequirementSettings{
		Enforce: raw.Requirements.Enforce,
		Sync:    raw.Requirements.Sync,
	}

	// Parse integration settings for API health checks, WebSocket validation, and CLI validation.
	// WebSocket URL is derived from API URL + path, following @vrooli/api-base conventions.
	// See: packages/api-base/docs/concepts/websocket-support.md
	cfg.Integration = IntegrationSettings{
		API: APISettings{
			HealthEndpoint: raw.Integration.API.HealthEndpoint,
			MaxResponseMs:  raw.Integration.API.MaxResponseMs,
		},
		WebSocket: WebSocketSettings{
			Enabled:         raw.Integration.WebSocket.Enabled,
			Path:            raw.Integration.WebSocket.Path,
			MaxConnectionMs: raw.Integration.WebSocket.MaxConnectionMs,
		},
		CLI: CLISettings{
			HelpArgs:              raw.Integration.CLI.HelpArgs,
			VersionArgs:           raw.Integration.CLI.VersionArgs,
			RequireVersionKeyword: raw.Integration.CLI.RequireVersionKeyword,
			CheckUnknownCommand:   raw.Integration.CLI.CheckUnknownCommand,
			CheckNoArgs:           raw.Integration.CLI.CheckNoArgs,
			NoArgsTimeoutMs:       raw.Integration.CLI.NoArgsTimeoutMs,
		},
	}

	// Check if any meaningful configuration was provided
	hasCLIConfig := len(cfg.Integration.CLI.HelpArgs) > 0 ||
		len(cfg.Integration.CLI.VersionArgs) > 0 ||
		cfg.Integration.CLI.RequireVersionKeyword != nil ||
		cfg.Integration.CLI.CheckUnknownCommand != nil ||
		cfg.Integration.CLI.CheckNoArgs != nil ||
		cfg.Integration.CLI.NoArgsTimeoutMs != 0

	hasIntegration := cfg.Integration.API.HealthEndpoint != "" ||
		cfg.Integration.API.MaxResponseMs != 0 ||
		cfg.Integration.WebSocket.Path != "" ||
		cfg.Integration.WebSocket.Enabled != nil ||
		hasCLIConfig

	if len(cfg.Phases) == 0 && len(cfg.Presets) == 0 &&
		cfg.Requirements.Enforce == nil && cfg.Requirements.Sync == nil &&
		!hasIntegration {
		return nil, nil
	}
	return cfg, nil
}

func normalizePhaseList(phases []string) []string {
	var filtered []string
	for _, phase := range phases {
		normalized := strings.ToLower(strings.TrimSpace(phase))
		if normalized != "" {
			filtered = append(filtered, normalized)
		}
	}
	return filtered
}

func parsePhaseTimeout(raw string) (time.Duration, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0, nil
	}
	matches := timeoutPattern.FindStringSubmatch(value)
	if len(matches) != 3 {
		return 0, fmt.Errorf("unsupported timeout value '%s'", raw)
	}
	number, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, fmt.Errorf("invalid timeout number '%s': %w", matches[1], err)
	}
	unit := matches[2]
	switch unit {
	case "", "s":
		return time.Duration(number) * time.Second, nil
	case "m":
		return time.Duration(number) * time.Minute, nil
	case "h":
		return time.Duration(number) * time.Hour, nil
	default:
		return 0, fmt.Errorf("unsupported timeout unit '%s'", unit)
	}
}
