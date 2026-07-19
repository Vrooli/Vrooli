package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"workspace-sandbox/internal/process"
)

var hostScenarioNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

type hostVrooliScenarioRequest struct {
	Action     string `json:"action"`
	Name       string `json:"name"`
	PortName   string `json:"port_name,omitempty"`
	BestEffort bool   `json:"best_effort,omitempty"`
	CleanStale bool   `json:"clean_stale,omitempty"`
	CustomPath string `json:"custom_path,omitempty"`
}

type hostVrooliScenarioResponse struct {
	Success  bool   `json:"success"`
	Action   string `json:"action"`
	Name     string `json:"name"`
	ExitCode int    `json:"exit_code"`
	Stdout   string `json:"stdout,omitempty"`
	Stderr   string `json:"stderr,omitempty"`
	Error    string `json:"error,omitempty"`
}

// HostVrooliScenario runs a narrow set of host-side Vrooli scenario lifecycle
// commands for sandboxed agents. It is intentionally not a generic host exec
// endpoint: the only accepted operations are start, restart, stop, and the
// read-only port lookup used by scenario CLIs.
func (h *Handlers) HostVrooliScenario(w http.ResponseWriter, r *http.Request) {
	var req hostVrooliScenarioRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeHostJSON(w, http.StatusBadRequest, hostVrooliScenarioResponse{Success: false, Error: "invalid JSON request: " + err.Error()})
		return
	}
	if err := validateHostVrooliScenarioRequest(req); err != nil {
		writeHostJSON(w, http.StatusBadRequest, hostVrooliScenarioResponse{Success: false, Action: req.Action, Name: req.Name, Error: err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Minute)
	defer cancel()

	args := []string{"--no-stale-check", "scenario", req.Action, req.Name}
	if req.Action == "port" && req.PortName != "" {
		args = append(args, req.PortName)
	}
	if req.CustomPath != "" {
		args = append(args, "--path", req.CustomPath)
	}
	if req.BestEffort {
		args = append(args, "--best-effort")
	}
	if req.CleanStale {
		args = append(args, "--clean-stale")
	}

	result, err := process.Run(ctx, h.Starter, process.StartOpts{
		Path:        "vrooli",
		Args:        args,
		Dir:         hostLifecycleDir(),
		Env:         hostLifecycleEnv(os.Environ()),
		SysProcAttr: process.NewProcessGroupSysProcAttr(),
	})
	resp := hostVrooliScenarioResponse{
		Success:  err == nil && result.Exit.ExitCode == 0,
		Action:   req.Action,
		Name:     req.Name,
		ExitCode: result.Exit.ExitCode,
		Stdout:   string(result.Stdout),
		Stderr:   string(result.Stderr),
	}
	if err != nil {
		resp.Error = err.Error()
	}
	status := http.StatusOK
	if !resp.Success {
		status = http.StatusBadGateway
	}
	writeHostJSON(w, status, resp)
}

func writeHostJSON(w http.ResponseWriter, status int, resp hostVrooliScenarioResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(resp)
}

func hostLifecycleDir() string {
	if root := strings.TrimSpace(os.Getenv("VROOLI_ROOT")); root != "" {
		return filepath.Clean(root)
	}
	return ""
}

func validateHostVrooliScenarioRequest(req hostVrooliScenarioRequest) error {
	switch req.Action {
	case "start", "restart", "stop", "port":
	default:
		return fmt.Errorf("unsupported action %q", req.Action)
	}
	if !hostScenarioNamePattern.MatchString(req.Name) {
		return fmt.Errorf("invalid scenario name %q", req.Name)
	}
	return nil
}

func hostLifecycleEnv(env []string) []string {
	out := make([]string, 0, len(env))
	blocked := map[string]struct{}{
		"VROOLI_SANDBOX_ID":     {},
		"VROOLI_SANDBOX_MERGED": {},
		"VROOLI_SANDBOX_SCOPE":  {},
	}
	for _, item := range env {
		key, _, ok := strings.Cut(item, "=")
		if !ok {
			continue
		}
		if _, deny := blocked[key]; deny {
			continue
		}
		out = append(out, item)
	}
	return out
}
