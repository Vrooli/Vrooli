package capabilities

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"os/exec"
	"strings"
	"time"
)

type ResourceChecker struct {
	URL    string
	Client *http.Client
}

// StaticChecker adapts a host probe to the capability registry without
// duplicating the registry's status vocabulary in a handler.
type StaticChecker struct {
	Available func() (bool, string)
}

func (c *StaticChecker) Check(context.Context) (Status, string) {
	if c == nil || c.Available == nil {
		return StatusUnavailable, "host capability probe is not configured"
	}
	ok, reason := c.Available()
	if ok {
		return StatusAvailable, "available"
	}
	return StatusUnavailable, reason
}

// BridgeChecker probes the Bridge control plane without exposing its owner
// credentials to the browser. Configuration failures are typed so the
// capability surface can explain the recovery path instead of collapsing all
// remote-terminal failures into a generic unavailable state.
type BridgeChecker struct {
	BaseURL     string
	OwnerToken  string
	ReauthToken string
	Client      *http.Client
	Probe       bool
}

func (c *BridgeChecker) Check(ctx context.Context) (Status, string) {
	result := c.CheckResult(ctx)
	return result.Status, result.Message
}

func (c *BridgeChecker) CheckResult(ctx context.Context) CheckResult {
	start := CheckResult{
		Status:          StatusUnavailable,
		ActionKind:      ActionKindScenarioStart,
		ActionLabel:     "Start Bridge",
		OperatorCommand: "vrooli scenario start vrooli-bridge --json",
	}
	base := strings.TrimRight(strings.TrimSpace(c.BaseURL), "/")
	if base == "" {
		start.Message = "Bridge URL is not configured"
		start.ReasonCode = "bridge_url_missing"
		return start
	}
	u, err := url.Parse(base)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		start.Message = "Bridge URL is invalid"
		start.ReasonCode = "bridge_url_invalid"
		return start
	}
	if strings.TrimSpace(c.OwnerToken) == "" || (strings.TrimSpace(c.ReauthToken) == "" && !strings.HasPrefix(strings.TrimSpace(c.OwnerToken), "LocalSession ")) {
		start.Message = "Bridge credentials are not configured"
		start.ReasonCode = "bridge_credentials_missing"
		return start
	}
	if !c.Probe {
		return CheckResult{Status: StatusAvailable, Message: "Bridge is configured"}
	}

	client := c.Client
	if client == nil {
		client = &http.Client{Timeout: 3 * time.Second}
	}
	probeCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(probeCtx, http.MethodGet, base+"/health", nil)
	if err != nil {
		start.Message = "Bridge health request could not be created"
		start.ReasonCode = "bridge_unreachable"
		return start
	}
	req.Header.Set("Authorization", c.OwnerToken)
	if strings.TrimSpace(c.ReauthToken) != "" {
		req.Header.Set("X-Bridge-Owner-Reauth", c.ReauthToken)
	}
	resp, err := client.Do(req)
	if err != nil {
		start.Message = "Bridge registry is unreachable"
		start.ReasonCode = "bridge_unreachable"
		return start
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		start.Message = "Bridge registry is unreachable"
		start.ReasonCode = "bridge_unreachable"
		return start
	}
	return CheckResult{Status: StatusAvailable, Message: "Bridge is reachable and ready"}
}

func (c *ResourceChecker) Check(ctx context.Context) (Status, string) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.URL, nil)
	if err != nil {
		return StatusUnavailable, "failed to create request: " + err.Error()
	}

	client := c.Client
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}

	resp, err := client.Do(req)
	if err != nil {
		return StatusUnavailable, "resource is not responding"
	}
	resp.Body.Close()

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusTemporaryRedirect {
		return StatusAvailable, "resource is healthy"
	}

	return StatusUnavailable, "resource returned unexpected status"
}

// WhisperChecker / KokoroChecker removed in the audio-tools adoption —
// Whisper + Kokoro are owned end-to-end by the audio-tools scenario, and
// web-console talks to audio-tools (not the raw resources). The
// `audio-tools` capability entry above is checked via ScenarioChecker.

// OllamaChecker verifies that Ollama is running by hitting its /api/tags
// endpoint.
type OllamaChecker struct {
	BaseURL string
	Client  *http.Client
}

func (c *OllamaChecker) Check(ctx context.Context) (Status, string) {
	client := c.Client
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}

	req, err := http.NewRequestWithContext(ctx, "GET", c.BaseURL+"/api/tags", nil)
	if err != nil {
		return StatusUnavailable, "failed to create request: " + err.Error()
	}

	resp, err := client.Do(req)
	if err != nil {
		return StatusUnavailable, "Ollama is not responding"
	}
	resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return StatusAvailable, "Ollama is running"
	}

	return StatusUnavailable, "Ollama returned unexpected status"
}

// ScenarioChecker verifies that a sibling Vrooli scenario is installed and
// running by shelling out to `vrooli scenario status <slug> --json`. It is
// the runtime probe behind every DependencyScenario entry in the registry,
// and the seam audio-tools (and any future connected scenario) will plug
// into once it ships.
//
// CLIPath/Args/Slug are injectable so tests can substitute a fake.
type ScenarioChecker struct {
	// Slug is the scenario directory name under scenarios/, e.g. "audio-tools".
	Slug string

	// CLIPath defaults to "vrooli" (resolved via PATH). Override in tests.
	CLIPath string

	// Args defaults to ["scenario", "status", Slug, "--json"]. Override only
	// when the CLI subcommand surface changes.
	Args []string

	// Run is the command-runner seam. Defaults to exec.CommandContext.
	Run func(ctx context.Context, name string, args ...string) ([]byte, error)

	// Timeout caps the probe latency. Defaults to 5s.
	Timeout time.Duration
}

func (c *ScenarioChecker) Check(ctx context.Context) (Status, string) {
	result := c.CheckResult(ctx)
	return result.Status, result.Message
}

func (c *ScenarioChecker) CheckResult(ctx context.Context) CheckResult {
	slug := c.Slug
	if slug == "" {
		return CheckResult{
			Status:     StatusUnavailable,
			Message:    "scenario slug not configured",
			ReasonCode: "scenario_slug_missing",
		}
	}
	cli := c.CLIPath
	if cli == "" {
		cli = "vrooli"
	}
	args := c.Args
	if len(args) == 0 {
		args = []string{"scenario", "status", slug, "--json"}
	}
	timeout := c.Timeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	probeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	run := c.Run
	if run == nil {
		run = func(ctx context.Context, name string, a ...string) ([]byte, error) {
			return exec.CommandContext(ctx, name, a...).Output()
		}
	}

	out, err := run(probeCtx, cli, args...)
	if err != nil {
		return CheckResult{
			Status:          StatusUnavailable,
			Message:         "scenario status unavailable; use the Vrooli lifecycle to inspect or start it",
			ReasonCode:      "scenario_status_cli_failed",
			ActionKind:      ActionKindOperatorCommand,
			ActionLabel:     "Inspect scenario",
			OperatorCommand: "vrooli scenario status " + slug + " --json",
		}
	}

	return classifyScenarioStatus(slug, out)
}

type scenarioStatusPayload struct {
	Success  bool               `json:"success"`
	Scenario scenarioStatusItem `json:"scenario"`
	// Legacy/list forms emitted scenario rows directly under "scenarios".
	Scenarios []scenarioStatusItem `json:"scenarios"`
}

type scenarioStatusItem struct {
	Name           string                  `json:"name"`
	Status         string                  `json:"status"`
	HealthStatus   any                     `json:"health_status"`
	HealthError    string                  `json:"health_error"`
	StartOperation *scenarioStartOperation `json:"start_operation"`
}

type scenarioStartOperation struct {
	Status      string `json:"status"`
	Verdict     string `json:"verdict"`
	Error       string `json:"error"`
	CurrentStep string `json:"current_step"`
}

func classifyScenarioStatus(slug string, out []byte) CheckResult {
	var payload scenarioStatusPayload
	if err := json.Unmarshal(out, &payload); err != nil {
		return CheckResult{
			Status:          StatusUnknown,
			Message:         "scenario status response was not valid JSON",
			ReasonCode:      "scenario_status_malformed_json",
			ActionKind:      ActionKindOperatorCommand,
			ActionLabel:     "Inspect scenario",
			OperatorCommand: "vrooli scenario status " + slug + " --json",
		}
	}

	item, ok := selectScenarioStatusItem(slug, payload)
	if !ok {
		return CheckResult{
			Status:          StatusUnknown,
			Message:         "scenario status response did not include " + slug,
			ReasonCode:      "scenario_status_missing_scenario",
			ActionKind:      ActionKindOperatorCommand,
			ActionLabel:     "Inspect scenario",
			OperatorCommand: "vrooli scenario status " + slug + " --json",
		}
	}

	if item.StartOperation != nil {
		switch item.StartOperation.Status {
		case "running":
			return CheckResult{
				Status:          StatusUnavailable,
				Message:         "scenario start is in progress: " + item.StartOperation.CurrentStep,
				ReasonCode:      "scenario_start_in_progress",
				ActionKind:      ActionKindOperatorCommand,
				ActionLabel:     "Wait for scenario",
				OperatorCommand: "vrooli scenario wait " + slug + " --json",
			}
		case "failed":
			return CheckResult{
				Status:          StatusUnavailable,
				Message:         joinStatusMessage("scenario start failed", item.StartOperation.Error),
				ReasonCode:      "scenario_start_failed",
				ActionKind:      ActionKindScenarioRestart,
				ActionLabel:     "Restart scenario",
				OperatorCommand: "vrooli scenario restart " + slug + " --json",
			}
		case "abandoned":
			return CheckResult{
				Status:          StatusUnavailable,
				Message:         joinStatusMessage("scenario start was abandoned", item.StartOperation.Error),
				ReasonCode:      "scenario_start_abandoned",
				ActionKind:      ActionKindScenarioStart,
				ActionLabel:     "Start scenario",
				OperatorCommand: "vrooli scenario start " + slug + " --json",
			}
		}
	}

	switch item.Status {
	case "running":
		return classifyScenarioHealth(slug, item)
	case "stopped", "not_running":
		return CheckResult{
			Status:          StatusUnavailable,
			Message:         "scenario is installed but stopped",
			ReasonCode:      "scenario_stopped",
			ActionKind:      ActionKindScenarioStart,
			ActionLabel:     "Start scenario",
			OperatorCommand: "vrooli scenario start " + slug + " --json",
		}
	case "starting":
		return CheckResult{
			Status:          StatusUnavailable,
			Message:         "scenario is starting",
			ReasonCode:      "scenario_starting",
			ActionKind:      ActionKindOperatorCommand,
			ActionLabel:     "Wait for scenario",
			OperatorCommand: "vrooli scenario wait " + slug + " --json",
		}
	case "":
		return CheckResult{
			Status:     StatusUnknown,
			Message:    "scenario status response omitted runtime status",
			ReasonCode: "scenario_status_missing",
		}
	default:
		return CheckResult{
			Status:          StatusUnknown,
			Message:         "scenario status unrecognised: " + item.Status,
			ReasonCode:      "scenario_status_unknown",
			ActionKind:      ActionKindOperatorCommand,
			ActionLabel:     "Inspect scenario",
			OperatorCommand: "vrooli scenario status " + slug + " --json",
		}
	}
}

func selectScenarioStatusItem(slug string, payload scenarioStatusPayload) (scenarioStatusItem, bool) {
	if payload.Scenario.Name == slug || (payload.Scenario.Name == "" && len(payload.Scenarios) == 0) {
		return payload.Scenario, payload.Scenario.Status != "" || payload.Scenario.HealthStatus != nil || payload.Scenario.StartOperation != nil
	}
	for _, item := range payload.Scenarios {
		if item.Name == slug {
			return item, true
		}
	}
	return scenarioStatusItem{}, false
}

func classifyScenarioHealth(slug string, item scenarioStatusItem) CheckResult {
	health := stringHealth(item.HealthStatus)
	switch health {
	case "healthy", "running":
		return CheckResult{Status: StatusAvailable, Message: "scenario is healthy"}
	case "degraded":
		return CheckResult{
			Status:          StatusUnavailable,
			Message:         joinStatusMessage("scenario is degraded", item.HealthError),
			ReasonCode:      "scenario_degraded",
			ActionKind:      ActionKindScenarioRestart,
			ActionLabel:     "Restart scenario",
			OperatorCommand: "vrooli scenario restart " + slug + " --json",
		}
	case "not_running", "stopped":
		return CheckResult{
			Status:          StatusUnavailable,
			Message:         "scenario health reports not running",
			ReasonCode:      "scenario_health_not_running",
			ActionKind:      ActionKindScenarioStart,
			ActionLabel:     "Start scenario",
			OperatorCommand: "vrooli scenario start " + slug + " --json",
		}
	case "":
		if item.HealthError != "" {
			return CheckResult{
				Status:          StatusUnavailable,
				Message:         "scenario health check failed: " + item.HealthError,
				ReasonCode:      "scenario_health_error",
				ActionKind:      ActionKindScenarioRestart,
				ActionLabel:     "Restart scenario",
				OperatorCommand: "vrooli scenario restart " + slug + " --json",
			}
		}
		return CheckResult{Status: StatusAvailable, Message: "scenario is running"}
	default:
		return CheckResult{
			Status:          StatusUnknown,
			Message:         "scenario health status unrecognised: " + health,
			ReasonCode:      "scenario_health_unknown",
			ActionKind:      ActionKindOperatorCommand,
			ActionLabel:     "Inspect scenario",
			OperatorCommand: "vrooli scenario status " + slug + " --json",
		}
	}
}

func stringHealth(v any) string {
	switch typed := v.(type) {
	case string:
		return typed
	case map[string]any:
		if state, ok := typed["status"].(string); ok {
			return state
		}
		if state, ok := typed["state"].(string); ok {
			return state
		}
	}
	return ""
}

func joinStatusMessage(prefix, detail string) string {
	if detail == "" {
		return prefix
	}
	return prefix + ": " + detail
}

// OpenRouterChecker verifies that OpenRouter is configured and reachable.
type OpenRouterChecker struct {
	APIKey  string
	BaseURL string
	Client  *http.Client
}

func (c *OpenRouterChecker) Check(ctx context.Context) (Status, string) {
	if c.APIKey == "" {
		return StatusUnavailable, "OPENROUTER_API_KEY not configured"
	}

	client := c.Client
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}

	baseURL := c.BaseURL
	if baseURL == "" {
		baseURL = "https://openrouter.ai"
	}

	req, err := http.NewRequestWithContext(ctx, "GET", baseURL+"/api/v1/models", nil)
	if err != nil {
		return StatusUnavailable, "failed to create request: " + err.Error()
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := client.Do(req)
	if err != nil {
		return StatusUnavailable, "OpenRouter is not reachable"
	}
	resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return StatusAvailable, "OpenRouter is configured and reachable"
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return StatusUnavailable, "OpenRouter API key is invalid"
	}

	return StatusUnavailable, "OpenRouter returned unexpected status"
}
