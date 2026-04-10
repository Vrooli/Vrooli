package integrations

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"agent-inbox/config"
	"agent-inbox/resilience"

	"google.golang.org/protobuf/encoding/protojson"
)

// protojson marshal/unmarshal options matching agent-manager's configuration.
// UseProtoNames ensures snake_case keys (matching agent-manager's serialization).
// DiscardUnknown on the consumer side ensures forward-compatibility.
var (
	protoMarshalOpts   = protojson.MarshalOptions{UseProtoNames: true}
	protoUnmarshalOpts = protojson.UnmarshalOptions{DiscardUnknown: true}
)

// AgentManagerClient provides direct REST API access to agent-manager.
// This client is used for reconciliation during server restart recovery.
// Tool execution flows through the Tool Execution Protocol (ProtocolHandler) instead.
//
// URL Resolution: The client resolves the agent-manager URL lazily and
// re-resolves on connection failures (e.g., after agent-manager restarts
// on a different port). This ensures resilience across service restarts.
type AgentManagerClient struct {
	baseURL    string
	httpClient *http.Client
	timeout    time.Duration
	retryCfg   resilience.RetryConfig
	cb         *resilience.CircuitBreaker
}

// NewAgentManagerClient creates a new agent-manager client.
// The client lazily resolves the agent-manager URL and re-resolves on connection failure.
func NewAgentManagerClient() (*AgentManagerClient, error) {
	cfg := config.Default()
	return NewAgentManagerClientWithConfig(cfg.Integration.AgentManagerTimeout)
}

// NewAgentManagerClientWithConfig creates a new agent-manager client with explicit timeout.
// This enables testing and custom configuration injection.
//
// The initial URL resolution is best-effort: if agent-manager is not yet
// available, the client is still created and will attempt re-resolution
// on the first request.
func NewAgentManagerClientWithConfig(timeout time.Duration) (*AgentManagerClient, error) {
	baseURL, _ := getAgentManagerURL() // best-effort; re-resolved on failure

	cfg := config.Default()
	retryCfg := resilience.RetryConfig{
		MaxAttempts: cfg.Resilience.RetryAttempts,
		BaseDelay:   cfg.Resilience.RetryBaseDelay,
		MaxDelay:    cfg.Resilience.RetryMaxDelay,
		Jitter:      0.1,
	}
	cb := resilience.NewCircuitBreaker(resilience.CircuitBreakerConfig{
		FailureThreshold: cfg.Resilience.CircuitBreakerThreshold,
		Cooldown:         cfg.Resilience.CircuitBreakerCooldown,
	})

	return &AgentManagerClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		timeout:  timeout,
		retryCfg: retryCfg,
		cb:       cb,
	}, nil
}

// getAgentManagerURL discovers the agent-manager API URL.
func getAgentManagerURL() (string, error) {
	// Try environment variable first (set by lifecycle system)
	if url := os.Getenv("AGENT_MANAGER_API_URL"); url != "" {
		return url, nil
	}

	// Fall back to port discovery via CLI
	cmd := exec.Command("vrooli", "scenario", "port", "agent-manager", "API_PORT")
	output, err := cmd.Output()
	if err == nil {
		port := strings.TrimSpace(string(output))
		if port != "" {
			return fmt.Sprintf("http://localhost:%s", port), nil
		}
	}

	// Fall back to reading agent-manager's service.json for a static port
	port, jsonErr := readAgentManagerPortFromServiceJSON()
	if jsonErr == nil && port != "" {
		return fmt.Sprintf("http://localhost:%s", port), nil
	}

	return "", fmt.Errorf("agent-manager not available: CLI discovery failed (%w), service.json fallback failed (%v)", err, jsonErr)
}

// readAgentManagerPortFromServiceJSON reads the API port from agent-manager's service.json.
func readAgentManagerPortFromServiceJSON() (string, error) {
	root := os.Getenv("VROOLI_ROOT")
	if root == "" {
		return "", fmt.Errorf("VROOLI_ROOT not set")
	}
	data, err := os.ReadFile(root + "/scenarios/agent-manager/.vrooli/service.json")
	if err != nil {
		return "", fmt.Errorf("read service.json: %w", err)
	}
	var svc struct {
		Ports map[string]struct {
			Port int `json:"port"`
		} `json:"ports"`
	}
	if err := json.Unmarshal(data, &svc); err != nil {
		return "", fmt.Errorf("parse service.json: %w", err)
	}
	if api, ok := svc.Ports["api"]; ok && api.Port > 0 {
		return fmt.Sprintf("%d", api.Port), nil
	}
	return "", fmt.Errorf("no api port in service.json")
}

// reResolveURL attempts to re-discover the agent-manager URL.
// Called after a connection failure to handle port drift after restart.
func (c *AgentManagerClient) reResolveURL() error {
	newURL, err := getAgentManagerURL()
	if err != nil {
		return err
	}
	c.baseURL = newURL
	return nil
}

// getBaseURL returns the current base URL, attempting re-resolution if empty.
func (c *AgentManagerClient) getBaseURL() (string, error) {
	if c.baseURL != "" {
		return c.baseURL, nil
	}
	if err := c.reResolveURL(); err != nil {
		return "", fmt.Errorf("agent-manager not available: %w", err)
	}
	return c.baseURL, nil
}

// doWithRetry performs an HTTP request with retry, circuit breaker, and URL re-resolution.
// On retry attempts > 1, it re-resolves the agent-manager URL to handle port drift.
// 4xx responses are marked as permanent (non-retryable) errors.
func (c *AgentManagerClient) doWithRetry(ctx context.Context, method, path string, body []byte) (*http.Response, error) {
	var resp *http.Response

	err := resilience.Retry(ctx, c.retryCfg, func(ctx context.Context, attempt int) error {
		// Re-resolve URL on retries to handle port drift
		if attempt > 1 {
			_ = c.reResolveURL()
		}

		baseURL, err := c.getBaseURL()
		if err != nil {
			return err
		}

		var reqBody io.Reader
		if body != nil {
			reqBody = bytes.NewReader(body)
		}

		req, err := http.NewRequestWithContext(ctx, method, baseURL+path, reqBody)
		if err != nil {
			return resilience.Permanent(err)
		}
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}

		doReq := func(ctx context.Context) error {
			var doErr error
			resp, doErr = c.httpClient.Do(req)
			return doErr
		}

		if c.cb != nil {
			err = c.cb.Execute(ctx, doReq)
		} else {
			err = doReq(ctx)
		}
		if err != nil {
			return err
		}

		// Mark 4xx responses as permanent (non-retryable)
		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			respBody, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return resilience.Permanent(fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody)))
		}

		return nil
	})

	return resp, err
}

// GetWebSocketURL returns the WebSocket URL for connecting to agent-manager events.
// Re-resolves the URL if not yet available.
func (c *AgentManagerClient) GetWebSocketURL() string {
	baseURL, err := c.getBaseURL()
	if err != nil {
		return "" // Caller should handle empty URL
	}
	// Convert http:// to ws://
	wsURL := strings.Replace(baseURL, "http://", "ws://", 1)
	wsURL = strings.Replace(wsURL, "https://", "wss://", 1)
	return wsURL + "/api/v1/ws"
}
