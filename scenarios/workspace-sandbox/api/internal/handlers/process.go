package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

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
//	                       PostProcessStdin / ProcessStats / ContainmentInfo
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

// validateExecutionMode makes the protected/tracking boundary explicit. Auto
// negotiates the selected driver's truthful capability; it never turns a
// protected request into tracking execution.
func (h *Handlers) validateExecutionMode(ctx context.Context, requested string) error {
	mode := strings.ToLower(strings.TrimSpace(requested))
	if mode == "" || mode == "auto" || mode == "tracking" {
		return nil
	}
	if mode != "protected" {
		return &types.ExecutionModeUnavailableError{Mode: mode, Driver: "unknown", Reason: "mode must be auto, tracking, or protected"}
	}
	d := h.Driver()
	if d == nil {
		return &types.ExecutionModeUnavailableError{Mode: mode, Reason: "no workspace driver is configured"}
	}
	if !d.Capabilities().Protected {
		return &types.ExecutionModeUnavailableError{Mode: mode, Driver: string(d.ID()), Reason: "the selected driver provides tracking only and has no process containment contract"}
	}
	if h.Starter == nil {
		return &types.ExecutionModeUnavailableError{Mode: mode, Driver: string(d.ID()), Reason: "containment preflight is unavailable"}
	}
	info, err := driver.GetContainmentInfo(ctx, h.Starter)
	if err != nil || info == nil || !info.Available {
		reason := "containment backend is unavailable"
		if err != nil {
			reason = err.Error()
		} else if info != nil && info.Error != "" {
			reason = info.Error
		}
		return &types.ExecutionModeUnavailableError{Mode: mode, Driver: string(d.ID()), Reason: reason}
	}
	return nil
}

// ExecRequest represents a request to execute a command in a sandbox.
type ExecRequest struct {
	Command        string            `json:"command"`
	Args           []string          `json:"args,omitempty"`
	AllowNetwork   bool              `json:"allowNetwork,omitempty"`
	Env            map[string]string `json:"env,omitempty"`
	WorkingDir     string            `json:"workingDir,omitempty"`
	SessionID      string            `json:"sessionId,omitempty"`
	WritableMounts []WritableMount   `json:"writableMounts,omitempty"`

	// IsolationLevel controls filesystem access.
	// "full" (default): maximum isolation, only /workspace accessible.
	// "vrooli-aware": can access Vrooli CLIs, configs, and localhost APIs.
	IsolationLevel string `json:"isolationLevel,omitempty"`
	ExecutionMode  string `json:"executionMode,omitempty"`

	// Resource limits (0 = unlimited)
	MemoryLimitMB int `json:"memoryLimitMB,omitempty"` // Max address space in MB
	CPUTimeSec    int `json:"cpuTimeSec,omitempty"`    // Max CPU time in seconds
	TimeoutSec    int `json:"timeoutSec,omitempty"`    // Wall-clock timeout in seconds
	MaxProcesses  int `json:"maxProcesses,omitempty"`  // Max child processes
	MaxOpenFiles  int `json:"maxOpenFiles,omitempty"`  // Max open file descriptors
}

// WritableMount is a caller-declared writable host directory.
type WritableMount struct {
	Path    string `json:"path"`
	Purpose string `json:"purpose"`
}

// ExecResponse is the structured response from a successful Exec call.
type ExecResponse struct {
	ExitCode int    `json:"exitCode"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	PID      int    `json:"pid,omitempty"`
	TimedOut bool   `json:"timedOut,omitempty"` // True if process was killed due to timeout

	// Containment is the process-containment that actually ran this command
	// (backend + enforcement guarantees), stamped from the exec backend
	// dispatch so callers get per-launch provenance rather than inference.
	Containment *types.SandboxContainment `json:"containment,omitempty"`
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
	if err := h.validateExecutionMode(r.Context(), req.ExecutionMode); err != nil {
		h.HandleDomainError(w, err)
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
	if reason := runtime.EvaluateProtectedGitAllowlist(sb.Behavior.Protected, runtime.GitDenyMessages{
		Blocked: h.Behavior.Protected.GitDenyMessageTemplate,
		NoVerb:  h.Behavior.Protected.GitNoVerbMessageTemplate,
	}, req.Command, req.Args); reason != "" {
		writeJSONStatus(w, http.StatusForbidden, map[string]any{
			"error":   "git_verb_blocked",
			"verb":    firstArg(req.Args),
			"message": reason,
		})
		return
	}
	if decision := runtime.EvaluateVrooliCommandPolicy(req.Command, req.Args); !decision.Allowed {
		writeJSONStatus(w, http.StatusForbidden, map[string]any{
			"error":                decision.Code,
			"message":              decision.Reason,
			"suggestedAlternative": decision.SuggestedAlternative,
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
	if err := addWritableMounts(&cfg, sb, req.WritableMounts); err != nil {
		h.JSONError(w, err.Error(), http.StatusBadRequest)
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
	level := d.RequiredContainment()
	result, err := driverexec.Exec(r.Context(), h.Starter, sb, level, cfg, req.Command, req.Args...)
	if err != nil {
		h.JSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Effective containment: the backend that actually ran this launch plus
	// the enforcements it provides on this host.
	containmentInfo, _ := driver.GetContainmentInfo(r.Context(), h.Starter)
	effective := driver.AdjustForLaunch(
		driver.EffectiveContainment(level, result.Backend, containmentInfo), cfg.AllowNetwork)

	if h.ProcessTracker != nil && result.PID > 0 {
		proc, err := h.ProcessTracker.Track(id, result.PID, req.Command, req.SessionID)
		if err != nil {
			h.JSONError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if proc != nil {
			_ = h.ProcessTracker.SetContainment(id, result.PID, effective)
		}
	}

	timedOut := result.ExitCode == 124 && result.Error != nil

	h.JSONSuccess(w, ExecResponse{
		ExitCode:    result.ExitCode,
		Stdout:      string(result.Stdout),
		Stderr:      string(result.Stderr),
		PID:         result.PID,
		TimedOut:    timedOut,
		Containment: effective,
	})
}

func addWritableMounts(cfg *driverexec.BwrapConfig, sb *types.Sandbox, mounts []WritableMount) error {
	if len(mounts) == 0 {
		return nil
	}
	for _, mount := range mounts {
		if err := validateWritableMount(sb, mount); err != nil {
			return err
		}
		if cfg.ReadWriteBinds == nil {
			cfg.ReadWriteBinds = make(map[string]string)
		}
		cfg.ReadWriteBinds[mount.Path] = mount.Path
	}
	return nil
}

func validateWritableMount(sb *types.Sandbox, mount WritableMount) error {
	if mount.Purpose == "" {
		return fmt.Errorf("writable mount purpose is required")
	}
	if !filepath.IsAbs(mount.Path) {
		return fmt.Errorf("path must be absolute")
	}
	resolved, err := filepath.EvalSymlinks(mount.Path)
	if err != nil {
		return fmt.Errorf("writable mount is unavailable: %w", err)
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return fmt.Errorf("writable mount is unavailable: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("writable mount is not a directory")
	}
	roots := append([]string{sb.ProjectRoot}, sb.AuxiliaryRoots...)
	for _, root := range roots {
		resolvedRoot, rootErr := filepath.EvalSymlinks(root)
		if rootErr != nil {
			continue
		}
		rel, relErr := filepath.Rel(resolvedRoot, resolved)
		if relErr == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return nil
		}
	}
	return fmt.Errorf("writable mount must be below a registered root")
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
