package hostlifecycle

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/lifecycle"
)

var portPattern = regexp.MustCompile(`\b(\d{2,5})\b`)

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
	port, err := workspaceSandboxPort(ctx)
	if err != nil {
		return ScenarioResponse{}, err
	}
	payload, err := json.Marshal(req)
	if err != nil {
		return ScenarioResponse{}, err
	}
	url := fmt.Sprintf("http://localhost:%s/api/v1/host/vrooli/scenario", port)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
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

func workspaceSandboxPort(ctx context.Context) (string, error) {
	for _, port := range []string{strings.TrimSpace(os.Getenv("WORKSPACE_SANDBOX_API_PORT")), "15120"} {
		if port == "" {
			continue
		}
		if workspaceSandboxHealthy(ctx, port) {
			return port, nil
		}
	}
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "vrooli", "--no-stale-check", "scenario", "port", "workspace-sandbox", "API_PORT")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("resolve workspace-sandbox API port: %w: %s", err, strings.TrimSpace(string(output)))
	}
	match := portPattern.FindStringSubmatch(string(output))
	if len(match) < 2 {
		return "", fmt.Errorf("resolve workspace-sandbox API port: no port in output %q", strings.TrimSpace(string(output)))
	}
	return match[1], nil
}

func workspaceSandboxHealthy(ctx context.Context, port string) bool {
	ctx, cancel := context.WithTimeout(ctx, 750*time.Millisecond)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost:"+port+"/health", nil)
	if err != nil {
		return false
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode >= 200 && resp.StatusCode < 300
}
