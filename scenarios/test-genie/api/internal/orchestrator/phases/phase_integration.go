package phases

import (
	"context"
	"fmt"
	"io"
	"strings"

	"test-genie/internal/integration"
	"test-genie/internal/orchestrator/workspace"
	"test-genie/internal/shared"
)

// runIntegrationPhase validates CLI flows and acceptance tests using the integration package.
// This includes CLI binary discovery, help/version command validation, BATS suite execution,
// API health checks, and WebSocket connection validation.
//
// Runtime URL Configuration:
// - API URL: Passed from env.APIURL (auto-detected from lifecycle metadata or service.json)
// - WebSocket URL: Derived from API URL + configured path, following @vrooli/api-base conventions
//
// The WebSocket URL derivation follows the pattern established by @vrooli/api-base where
// WebSocket connections are proxied through the same server as HTTP API requests:
//   - Protocol conversion: http:// → ws://, https:// → wss://
//   - Path appended: {api_url}{websocket_path}
//
// See: packages/api-base/docs/concepts/websocket-support.md
func runIntegrationPhase(ctx context.Context, env workspace.Environment, logWriter io.Writer) RunReport {
	var summary string

	report := RunPhase(ctx, logWriter, "integration",
		func() (*integration.RunResult, error) {
			// Load testing config for integration-specific settings
			config, err := workspace.LoadTestingConfig(env.ScenarioDir)
			if err != nil {
				shared.LogWarn(logWriter, "failed to load testing config: %v (using defaults)", err)
			}

			// Build integration config with runtime URLs and settings from testing.json
			intConfig := buildIntegrationConfig(env, config, logWriter)

			runner := integration.New(
				intConfig,
				integration.WithLogger(logWriter),
				integration.WithCommandExecutor(phaseCommandExecutor),
				integration.WithCommandCapture(phaseCommandCapture),
				integration.WithCommandLookup(commandLookup),
			)
			return runner.Run(ctx), nil
		},
		func(r *integration.RunResult) PhaseResult[integration.Observation] {
			if r != nil {
				summary = r.Summary.String()
			}
			return ExtractWithSummary(
				r.Success,
				r.Error,
				r.FailureClass,
				r.Remediation,
				r.Observations,
				"✅",
				fmt.Sprintf("Integration validation completed (%s)", r.Summary.String()),
			)
		},
	)

	writePhasePointer(env, "integration", report, map[string]any{"summary": summary}, logWriter)
	return report
}

// buildIntegrationConfig constructs the integration.Config from environment and testing.json settings.
// This handles the derivation of WebSocket URL from API URL following @vrooli/api-base conventions.
func buildIntegrationConfig(env workspace.Environment, testConfig *workspace.Config, logWriter io.Writer) integration.Config {
	cfg := integration.Config{
		ScenarioDir:  env.ScenarioDir,
		ScenarioName: env.ScenarioName,
		APIBaseURL:   env.APIURL,
	}

	// Apply defaults
	if cfg.APIHealthEndpoint == "" {
		cfg.APIHealthEndpoint = "/health"
	}
	if cfg.APIMaxResponseMs == 0 {
		cfg.APIMaxResponseMs = 1000
	}
	if cfg.WebSocketMaxConnectionMs == 0 {
		cfg.WebSocketMaxConnectionMs = 2000
	}

	// Override with testing.json settings if available
	if testConfig != nil {
		intSettings := testConfig.Integration

		// API settings
		if intSettings.API.HealthEndpoint != "" {
			cfg.APIHealthEndpoint = intSettings.API.HealthEndpoint
		}
		if intSettings.API.MaxResponseMs > 0 {
			cfg.APIMaxResponseMs = intSettings.API.MaxResponseMs
		}

		// WebSocket settings - derive URL from API URL + path
		// This follows the @vrooli/api-base pattern where WebSocket uses the same host:port
		// as the API but with ws:// or wss:// protocol.
		// See: packages/api-base/docs/concepts/websocket-support.md
		if intSettings.WebSocket.Path != "" && env.APIURL != "" {
			wsEnabled := true
			if intSettings.WebSocket.Enabled != nil {
				wsEnabled = *intSettings.WebSocket.Enabled
			}

			if wsEnabled {
				cfg.WebSocketURL = deriveWebSocketURL(env.APIURL, intSettings.WebSocket.Path)
				shared.LogInfo(logWriter, "WebSocket URL derived from API URL: %s", cfg.WebSocketURL)
			}
		}

		if intSettings.WebSocket.MaxConnectionMs > 0 {
			cfg.WebSocketMaxConnectionMs = intSettings.WebSocket.MaxConnectionMs
		}
	}

	return cfg
}

// deriveWebSocketURL converts an HTTP API URL to a WebSocket URL by changing the protocol
// and appending the WebSocket path. This follows the @vrooli/api-base convention where
// WebSocket connections are proxied through the same server.
//
// Examples:
//   - http://localhost:8080 + /api/v1/ws → ws://localhost:8080/api/v1/ws
//   - https://example.com + /ws → wss://example.com/ws
//
// See: packages/api-base/docs/concepts/websocket-support.md
func deriveWebSocketURL(apiURL, wsPath string) string {
	// Convert HTTP protocol to WebSocket protocol
	wsURL := apiURL
	if strings.HasPrefix(wsURL, "https://") {
		wsURL = "wss://" + strings.TrimPrefix(wsURL, "https://")
	} else if strings.HasPrefix(wsURL, "http://") {
		wsURL = "ws://" + strings.TrimPrefix(wsURL, "http://")
	}

	// Ensure path starts with / and append it
	if !strings.HasPrefix(wsPath, "/") {
		wsPath = "/" + wsPath
	}

	// Remove trailing slash from base URL before appending path
	wsURL = strings.TrimSuffix(wsURL, "/")

	return wsURL + wsPath
}
