package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	"workspace-sandbox/internal/driver"
	driverexec "workspace-sandbox/internal/driver/exec"
	"workspace-sandbox/internal/runtime"
	"workspace-sandbox/internal/types"
)

// process.go: the Exec endpoint plus shared bwrap-config helpers used
// by every process-related handler. The other process handlers live in:
//
//	process_start.go      StartProcess (background processes)
//	process_logs.go       GetProcessLogs / StreamProcessLogs / ListProcessLogs
//	process_management.go ListProcesses / KillProcess / KillAllProcesses
//	                       PostProcessStdin / ProcessStats / BwrapInfo
//
// Profile resolution + home-overlay enforcement live in
// internal/runtime/profile.go (see runtime.ProfileResolver). The
// protected-mode git allowlist lives in internal/runtime/git_allowlist.go.

// profileResolver constructs a runtime.ProfileResolver wired to the
// handler's startup-cached profile snapshot, default profile ID, and
// active driver capabilities. Built per-call so a hot-swapped driver
// is reflected immediately in subsequent requests; the profile snapshot
// itself is loaded once at startup and refreshed only on admin
// Save/Delete (Round 4 Phase 9).
func (h *Handlers) profileResolver() *runtime.ProfileResolver {
	caps := driver.DriverCapabilities{}
	if d := h.Driver(); d != nil {
		caps = d.Capabilities()
	}
	return &runtime.ProfileResolver{
		Profiles:  h.ProfileSnapshot(),
		DefaultID: h.Config.Execution.DefaultIsolationProfile,
		Caps:      caps,
	}
}

// applyIsolationProfile is a thin shim onto runtime.ProfileResolver so
// existing call sites stay readable. Returns the same typed errors as
// before (IsolationProfileNotFoundError / HomeOverlayRequiredError).
//
// DOC: home-overlay seam — handler-side enforcement.
func (h *Handlers) applyIsolationProfile(sb *types.Sandbox, cfg *driverexec.BwrapConfig, requestedID string) error {
	return h.profileResolver().ResolveAndApply(sb, cfg, requestedID)
}

// ExecRequest represents a request to execute a command in a sandbox.
type ExecRequest struct {
	Command      string            `json:"command"`
	Args         []string          `json:"args,omitempty"`
	AllowNetwork bool              `json:"allowNetwork,omitempty"`
	Env          map[string]string `json:"env,omitempty"`
	WorkingDir   string            `json:"workingDir,omitempty"`
	SessionID    string            `json:"sessionId,omitempty"`

	// IsolationLevel controls filesystem access.
	// "full" (default): maximum isolation, only /workspace accessible.
	// "vrooli-aware": can access Vrooli CLIs, configs, and localhost APIs.
	IsolationLevel string `json:"isolationLevel,omitempty"`

	// Resource limits (0 = unlimited)
	MemoryLimitMB int `json:"memoryLimitMB,omitempty"` // Max address space in MB
	CPUTimeSec    int `json:"cpuTimeSec,omitempty"`    // Max CPU time in seconds
	TimeoutSec    int `json:"timeoutSec,omitempty"`    // Wall-clock timeout in seconds
	MaxProcesses  int `json:"maxProcesses,omitempty"`  // Max child processes
	MaxOpenFiles  int `json:"maxOpenFiles,omitempty"`  // Max open file descriptors
}

// ExecResponse is the structured response from a successful Exec call.
type ExecResponse struct {
	ExitCode int    `json:"exitCode"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	PID      int    `json:"pid,omitempty"`
	TimedOut bool   `json:"timedOut,omitempty"` // True if process was killed due to timeout
}

// Exec executes a command inside a sandbox with bubblewrap isolation.
// [OT-P0-003] Bubblewrap Process Isolation
func (h *Handlers) Exec(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		h.JSONError(w, "invalid sandbox ID", http.StatusBadRequest)
		return
	}

	var req ExecRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.JSONError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Command == "" {
		h.JSONError(w, "command is required", http.StatusBadRequest)
		return
	}

	sb, err := h.Service.Get(r.Context(), id)
	if h.HandleDomainError(w, err) {
		return
	}

	if sb.Status != types.StatusActive {
		h.JSONError(w, "sandbox must be active to execute commands", http.StatusConflict)
		return
	}

	// Protected-mode git allowlist enforcement. When the sandbox carries a
	// non-empty Behavior.Protected.GitAllowlist, requests targeting `git`
	// (by basename) are restricted to the allowed verbs.
	if reason := runtime.EvaluateProtectedGitAllowlist(sb.Behavior.Protected, req.Command, req.Args); reason != "" {
		writeJSONStatus(w, http.StatusForbidden, map[string]any{
			"error":   "git_verb_blocked",
			"verb":    firstArg(req.Args),
			"message": reason,
		})
		return
	}

	cfg := driverexec.DefaultBwrapConfig()
	driverexec.CaptureEnv().ApplyTo(&cfg)
	if req.WorkingDir != "" {
		cfg.WorkingDir = req.WorkingDir
	}
	for k, v := range req.Env {
		cfg.Env[k] = v
	}

	if err := h.applyIsolationProfile(sb, &cfg, req.IsolationLevel); err != nil {
		h.HandleDomainError(w, err)
		return
	}

	if req.AllowNetwork {
		cfg.AllowNetwork = true
	}

	requestedLimits := driverexec.ResourceLimits{
		MemoryLimitMB: req.MemoryLimitMB,
		CPUTimeSec:    req.CPUTimeSec,
		MaxProcesses:  req.MaxProcesses,
		MaxOpenFiles:  req.MaxOpenFiles,
		TimeoutSec:    req.TimeoutSec,
	}
	cfg.ResourceLimits = runtime.ApplyResourceLimitDefaults(requestedLimits, h.Config.Execution)

	d := h.Driver()
	result, err := driverexec.Exec(r.Context(), sb, d.RequiresBwrap(), cfg, req.Command, req.Args...)
	if err != nil {
		h.JSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if h.ProcessTracker != nil && result.PID > 0 {
		if _, err := h.ProcessTracker.Track(id, result.PID, req.Command, req.SessionID); err != nil {
			h.JSONError(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	timedOut := result.ExitCode == 124 && result.Error != nil

	h.JSONSuccess(w, ExecResponse{
		ExitCode: result.ExitCode,
		Stdout:   string(result.Stdout),
		Stderr:   string(result.Stderr),
		PID:      result.PID,
		TimedOut: timedOut,
	})
}

// firstArg returns args[0] or "" — used by Exec/StartProcess to surface
// the rejected verb in the structured 403 response when the protected
// git allowlist blocks a request.
func firstArg(args []string) string {
	if len(args) == 0 {
		return ""
	}
	return args[0]
}

// writeJSONStatus mirrors h.JSONSuccess but lets callers pick an HTTP
// status. Used for structured 403 responses (git allowlist refusal).
func writeJSONStatus(w http.ResponseWriter, status int, body map[string]any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
