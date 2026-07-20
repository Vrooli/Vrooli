package hostlifecycle

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/vrooli/vrooli/internal/lifecycle"
)

// HostLifecycleBaseEnv is the workspace-sandbox API base made available to a
// Vrooli-aware process. It is a bootstrap transport for the narrow host
// lifecycle proxy, not a target scenario endpoint: target ports continue to
// be resolved by the host-side Vrooli CLI and its runtime registry.
const HostLifecycleBaseEnv = "VROOLI_HOST_LIFECYCLE_BASE"

type ScenarioRequest struct {
	Action     string `json:"action"`
	Name       string `json:"name"`
	PortName   string `json:"port_name,omitempty"`
	BestEffort bool   `json:"best_effort,omitempty"`
	CleanStale bool   `json:"clean_stale,omitempty"`
	CustomPath string `json:"custom_path,omitempty"`
}

type ScenarioResponse struct {
	Success  bool   `json:"success"`
	Action   string `json:"action"`
	Name     string `json:"name"`
	ExitCode int    `json:"exit_code"`
	Stdout   string `json:"stdout,omitempty"`
	Stderr   string `json:"stderr,omitempty"`
	Error    string `json:"error,omitempty"`
}

func InSandbox() bool {
	return strings.TrimSpace(os.Getenv("VROOLI_SANDBOX_MERGED")) != ""
}

func RunScenario(ctx context.Context, req ScenarioRequest) (ScenarioResponse, error) {
	baseURL, err := workspaceSandboxBaseURL()
	if err != nil {
		return ScenarioResponse{}, err
	}
	payload, err := json.Marshal(req)
	if err != nil {
		return ScenarioResponse{}, err
	}
	endpoint := baseURL + "/api/v1/host/vrooli/scenario"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return ScenarioResponse{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return ScenarioResponse{}, fmt.Errorf("call workspace-sandbox host lifecycle proxy: %w", err)
	}
	defer resp.Body.Close()
	var out ScenarioResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return ScenarioResponse{}, fmt.Errorf("decode workspace-sandbox host lifecycle response: %w", err)
	}
	if resp.StatusCode >= 400 || !out.Success {
		msg := strings.TrimSpace(out.Error)
		if msg == "" {
			msg = strings.TrimSpace(out.Stderr)
		}
		if msg == "" {
			msg = fmt.Sprintf("host lifecycle proxy exited with status %d", out.ExitCode)
		}
		return out, fmt.Errorf("%s", msg)
	}
	return out, nil
}

func StartOptionsRequest(action, name string, opts lifecycle.StartOptions) ScenarioRequest {
	return ScenarioRequest{
		Action:     action,
		Name:       name,
		BestEffort: opts.BestEffort,
		CleanStale: opts.CleanStale,
		CustomPath: opts.CustomPath,
	}
}

func workspaceSandboxBaseURL() (string, error) {
	raw := strings.TrimSpace(os.Getenv(HostLifecycleBaseEnv))
	if raw == "" {
		return "", fmt.Errorf("sandbox host lifecycle transport is unavailable: %s is not set", HostLifecycleBaseEnv)
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("sandbox host lifecycle transport is invalid: %s=%q", HostLifecycleBaseEnv, raw)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("sandbox host lifecycle transport must use http or https: %s=%q", HostLifecycleBaseEnv, raw)
	}
	return strings.TrimRight(raw, "/"), nil
}
