package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	"workspace-sandbox/internal/config"
	"workspace-sandbox/internal/driver"
	"workspace-sandbox/internal/process"
	"workspace-sandbox/internal/types"
)

// --- Process Management Endpoints (OT-P0-003, OT-P0-008) ---

// applyResourceLimitDefaults applies default resource limits from ExecutionConfig
// when request values are 0, and clamps to maximum allowed values.
func applyResourceLimitDefaults(
	req driver.ResourceLimits,
	execCfg config.ExecutionConfig,
) driver.ResourceLimits {
	defaults := execCfg.DefaultResourceLimits
	maxes := execCfg.MaxResourceLimits

	result := req

	// Apply defaults for zero values
	if result.MemoryLimitMB == 0 && defaults.MemoryLimitMB > 0 {
		result.MemoryLimitMB = defaults.MemoryLimitMB
	}
	if result.CPUTimeSec == 0 && defaults.CPUTimeSec > 0 {
		result.CPUTimeSec = defaults.CPUTimeSec
	}
	if result.MaxProcesses == 0 && defaults.MaxProcesses > 0 {
		result.MaxProcesses = defaults.MaxProcesses
	}
	if result.MaxOpenFiles == 0 && defaults.MaxOpenFiles > 0 {
		result.MaxOpenFiles = defaults.MaxOpenFiles
	}
	if result.TimeoutSec == 0 && defaults.TimeoutSec > 0 {
		result.TimeoutSec = defaults.TimeoutSec
	}

	// Clamp to maximums (0 = no maximum)
	if maxes.MemoryLimitMB > 0 && result.MemoryLimitMB > maxes.MemoryLimitMB {
		result.MemoryLimitMB = maxes.MemoryLimitMB
	}
	if maxes.CPUTimeSec > 0 && result.CPUTimeSec > maxes.CPUTimeSec {
		result.CPUTimeSec = maxes.CPUTimeSec
	}
	if maxes.MaxProcesses > 0 && result.MaxProcesses > maxes.MaxProcesses {
		result.MaxProcesses = maxes.MaxProcesses
	}
	if maxes.MaxOpenFiles > 0 && result.MaxOpenFiles > maxes.MaxOpenFiles {
		result.MaxOpenFiles = maxes.MaxOpenFiles
	}
	if maxes.TimeoutSec > 0 && result.TimeoutSec > maxes.TimeoutSec {
		result.TimeoutSec = maxes.TimeoutSec
	}

	return result
}

// convertProfileToDriver converts a config.IsolationProfile to driver.IsolationProfile.
func convertProfileToDriver(p *config.IsolationProfile) *driver.IsolationProfile {
	if p == nil {
		return nil
	}
	return &driver.IsolationProfile{
		ID:             p.ID,
		Name:           p.Name,
		Description:    p.Description,
		Builtin:        p.Builtin,
		NetworkAccess:  p.NetworkAccess,
		ReadOnlyBinds:  p.ReadOnlyBinds,
		ReadWriteBinds: p.ReadWriteBinds,
		Environment:    p.Environment,
		Hostname:       p.Hostname,
	}
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

// ExecResponse represents the result of executing a command.
type ExecResponse struct {
	ExitCode int    `json:"exitCode"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	PID      int    `json:"pid,omitempty"`
	TimedOut bool   `json:"timedOut,omitempty"` // True if process was killed due to timeout
}

// Exec handles executing a command inside a sandbox with bubblewrap isolation.
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

	// Get the sandbox
	sb, err := h.Service.Get(r.Context(), id)
	if h.HandleDomainError(w, err) {
		return
	}

	// Verify sandbox is active
	if sb.Status != types.StatusActive {
		h.JSONError(w, "sandbox must be active to execute commands", http.StatusConflict)
		return
	}

	// Protected-mode git allowlist enforcement. When the sandbox carries a
	// non-empty Behavior.Protected.GitAllowlist, requests targeting `git`
	// (by basename) are restricted to the allowed verbs. The denial shape
	// is structured so agent-manager can surface a typed tool.blocked event.
	if reason := evaluateProtectedGitAllowlist(sb.Behavior.Protected, req.Command, req.Args); reason != "" {
		writeJSONStatus(w, http.StatusForbidden, map[string]any{
			"error":   "git_verb_blocked",
			"verb":    firstArg(req.Args),
			"message": reason,
		})
		return
	}

	// Build bwrap config with resource limits and isolation level
	cfg := driver.DefaultBwrapConfig()
	if req.WorkingDir != "" {
		cfg.WorkingDir = req.WorkingDir
	}
	for k, v := range req.Env {
		cfg.Env[k] = v
	}

	// Determine isolation profile to use
	isolationLevel := req.IsolationLevel
	if isolationLevel == "" {
		isolationLevel = h.Config.Execution.DefaultIsolationProfile
	}
	if isolationLevel == "" {
		isolationLevel = "full" // Ultimate fallback
	}

	// Look up and apply isolation profile
	if h.ProfileStore != nil {
		profile, profErr := h.ProfileStore.Get(isolationLevel)
		if profErr == nil {
			driver.ApplyIsolationProfile(&cfg, convertProfileToDriver(profile))
		} else if isolationLevel == "vrooli-aware" {
			// Fallback for legacy "vrooli-aware" if not found in store
			driver.ApplyVrooliAwareConfig(&cfg)
		}
	} else if isolationLevel == "vrooli-aware" {
		driver.ApplyVrooliAwareConfig(&cfg)
	}

	// Override network if explicitly requested
	if req.AllowNetwork {
		cfg.AllowNetwork = true
	}

	// Set resource limits with defaults and clamping from ExecutionConfig
	requestedLimits := driver.ResourceLimits{
		MemoryLimitMB: req.MemoryLimitMB,
		CPUTimeSec:    req.CPUTimeSec,
		MaxProcesses:  req.MaxProcesses,
		MaxOpenFiles:  req.MaxOpenFiles,
		TimeoutSec:    req.TimeoutSec,
	}
	cfg.ResourceLimits = applyResourceLimitDefaults(requestedLimits, h.Config.Execution)

	// Execute the command (all drivers implement Exec via the Driver interface)
	result, err := h.Driver().Exec(r.Context(), sb, cfg, req.Command, req.Args...)
	if err != nil {
		h.JSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Track the process if tracker is available
	if h.ProcessTracker != nil && result.PID > 0 {
		if _, err := h.ProcessTracker.Track(id, result.PID, req.Command, req.SessionID); err != nil {
			h.JSONError(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Determine if process timed out
	timedOut := result.ExitCode == 124 && result.Error != nil

	h.JSONSuccess(w, ExecResponse{
		ExitCode: result.ExitCode,
		Stdout:   string(result.Stdout),
		Stderr:   string(result.Stderr),
		PID:      result.PID,
		TimedOut: timedOut,
	})
}

// StartProcessRequest represents a request to start a background process.
type StartProcessRequest struct {
	Command      string            `json:"command"`
	Args         []string          `json:"args,omitempty"`
	AllowNetwork bool              `json:"allowNetwork,omitempty"`
	Env          map[string]string `json:"env,omitempty"`
	WorkingDir   string            `json:"workingDir,omitempty"`
	SessionID    string            `json:"sessionId,omitempty"`
	Name         string            `json:"name,omitempty"` // Optional friendly name for the process

	// WithStdin requests the driver create a stdin pipe wired to the
	// process. Callers can then stream input via POST /processes/{pid}/stdin
	// and signal EOF via POST /processes/{pid}/stdin?close=true.
	WithStdin bool `json:"withStdin,omitempty"`

	// IsolationLevel controls filesystem access.
	// "full" (default): maximum isolation, only /workspace accessible.
	// "vrooli-aware": can access Vrooli CLIs, configs, and localhost APIs.
	IsolationLevel string `json:"isolationLevel,omitempty"`

	// Resource limits (0 = unlimited)
	// Note: TimeoutSec is not enforced for background processes; use manual kill.
	MemoryLimitMB int `json:"memoryLimitMB,omitempty"` // Max address space in MB
	CPUTimeSec    int `json:"cpuTimeSec,omitempty"`    // Max CPU time in seconds
	MaxProcesses  int `json:"maxProcesses,omitempty"`  // Max child processes
	MaxOpenFiles  int `json:"maxOpenFiles,omitempty"`  // Max open file descriptors
}

// StartProcess handles starting a background process in a sandbox.
// [OT-P0-003] Bubblewrap Process Isolation
// [OT-P0-008] Process/Session Tracking
func (h *Handlers) StartProcess(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		h.JSONError(w, "invalid sandbox ID", http.StatusBadRequest)
		return
	}

	var req StartProcessRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.JSONError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Command == "" {
		h.JSONError(w, "command is required", http.StatusBadRequest)
		return
	}

	// Get the sandbox
	sb, err := h.Service.Get(r.Context(), id)
	if h.HandleDomainError(w, err) {
		return
	}

	// Verify sandbox is active
	if sb.Status != types.StatusActive {
		h.JSONError(w, "sandbox must be active to start processes", http.StatusConflict)
		return
	}

	// Protected-mode git allowlist enforcement. Mirrors Exec: when the
	// sandbox carries a non-empty Behavior.Protected.GitAllowlist, background
	// process starts targeting `git` are restricted to the allowed verbs.
	// Critical for the runner-fork (execute/protected-sandbox-agent-launch):
	// agent processes launched here would otherwise be able to bypass the
	// /exec allowlist by spawning git directly.
	if reason := evaluateProtectedGitAllowlist(sb.Behavior.Protected, req.Command, req.Args); reason != "" {
		writeJSONStatus(w, http.StatusForbidden, map[string]any{
			"error":   "git_verb_blocked",
			"verb":    firstArg(req.Args),
			"message": reason,
		})
		return
	}

	// Build bwrap config with resource limits and isolation level
	cfg := driver.DefaultBwrapConfig()
	if req.WorkingDir != "" {
		cfg.WorkingDir = req.WorkingDir
	}
	for k, v := range req.Env {
		cfg.Env[k] = v
	}

	// Determine isolation profile to use
	isolationLevel := req.IsolationLevel
	if isolationLevel == "" {
		isolationLevel = h.Config.Execution.DefaultIsolationProfile
	}
	if isolationLevel == "" {
		isolationLevel = "full" // Ultimate fallback
	}

	// Look up and apply isolation profile
	if h.ProfileStore != nil {
		profile, profErr := h.ProfileStore.Get(isolationLevel)
		if profErr == nil {
			driver.ApplyIsolationProfile(&cfg, convertProfileToDriver(profile))
		} else if isolationLevel == "vrooli-aware" {
			// Fallback for legacy "vrooli-aware" if not found in store
			driver.ApplyVrooliAwareConfig(&cfg)
		}
	} else if isolationLevel == "vrooli-aware" {
		driver.ApplyVrooliAwareConfig(&cfg)
	}

	// Override network if explicitly requested
	if req.AllowNetwork {
		cfg.AllowNetwork = true
	}

	// Set resource limits with defaults and clamping from ExecutionConfig
	// Note: TimeoutSec is not used for background processes - use manual kill
	requestedLimits := driver.ResourceLimits{
		MemoryLimitMB: req.MemoryLimitMB,
		CPUTimeSec:    req.CPUTimeSec,
		MaxProcesses:  req.MaxProcesses,
		MaxOpenFiles:  req.MaxOpenFiles,
		TimeoutSec:    0, // Not applicable for background processes
	}
	cfg.ResourceLimits = applyResourceLimitDefaults(requestedLimits, h.Config.Execution)
	cfg.ResourceLimits.TimeoutSec = 0 // Ensure timeout is never applied to background processes

	// Create pending log pair (stdout + stderr) BEFORE starting process.
	var pendingPair *process.PendingLogPair
	if h.ProcessLogger != nil {
		var logErr error
		pendingPair, logErr = h.ProcessLogger.CreatePendingLogPair(id)
		if logErr == nil {
			cfg.StdoutWriter = pendingPair.Stdout
			cfg.StderrWriter = pendingPair.Stderr
		}
	}

	// Optional stdin pipe.
	var stdinWriter *processStdinPipe
	if req.WithStdin {
		stdinReader, sw := newStdinPipe()
		cfg.StdinReader = stdinReader
		stdinWriter = sw
	}

	// onExit fires from the driver's wait reaper after cmd.Wait() returns.
	// It records ExitInfo on the tracker (which closes the per-process exit
	// channel and unblocks subscribers / SSE consumers) and finalises the
	// log pair so subscribers see EOF.
	pidCh := make(chan int, 1)
	var onExitOnce sync.Once
	cfg.OnExit = func(exitCode, signal int, oomKilled bool) {
		// Wait briefly for the StartProcess code below to publish the PID.
		var pid int
		select {
		case pid = <-pidCh:
		case <-time.After(2 * time.Second):
			// PID never published — process must have failed to start
			// in a way the driver still surfaced; nothing to record.
			return
		}
		// Republish so any racing reads still see it. (Idempotent because
		// the channel is buffered=1 and we don't read it again.)
		select {
		case pidCh <- pid:
		default:
		}
		onExitOnce.Do(func() {
			info := process.ExitInfo{ExitCode: exitCode, Signal: signal, OOMKilled: oomKilled, StoppedAt: time.Now()}
			if h.ProcessTracker != nil {
				h.ProcessTracker.RecordExit(id, pid, info)
			}
			if h.ProcessLogger != nil {
				_ = h.ProcessLogger.CloseLogPair(id, pid, info)
			}
		})
	}

	// Start process
	pid, err := h.Driver().StartProcess(r.Context(), sb, cfg, req.Command, req.Args...)
	if err != nil {
		if pendingPair != nil {
			if abortErr := h.ProcessLogger.AbortPair(pendingPair); abortErr != nil {
				h.JSONError(w, abortErr.Error(), http.StatusInternalServerError)
				return
			}
		}
		if stdinWriter != nil {
			_ = stdinWriter.Close()
		}
		h.JSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Publish the pid so any pending OnExit dispatch (process died very
	// quickly) can pick it up.
	pidCh <- pid

	// Finalize log pair with actual PID.
	var stdoutPath, stderrPath string
	if pendingPair != nil {
		var logErr error
		stdoutPath, stderrPath, logErr = h.ProcessLogger.FinalizePair(pendingPair, pid)
		if logErr != nil {
			h.JSONError(w, logErr.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Track the process with optional name
	var trackedProc *process.TrackedProcess
	if h.ProcessTracker != nil {
		displayName := req.Command
		if req.Name != "" {
			displayName = req.Name
		}
		var trackErr error
		trackedProc, trackErr = h.ProcessTracker.Track(id, pid, displayName, req.SessionID)
		if trackErr != nil {
			h.JSONError(w, trackErr.Error(), http.StatusInternalServerError)
			return
		}
		// Attach stdin pipe writer to the tracked process so the stdin
		// endpoint can find it via the tracker.
		if stdinWriter != nil {
			if err := h.ProcessTracker.SetStdin(id, pid, stdinWriter); err != nil {
				h.JSONError(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	} else if stdinWriter != nil {
		// No tracker means there is no place to store the stdin writer;
		// close it so the process gets EOF immediately rather than blocking
		// on stdin forever.
		_ = stdinWriter.Close()
	}

	response := map[string]interface{}{
		"pid":       pid,
		"sandboxId": id,
		"command":   req.Command,
	}
	if req.Name != "" {
		response["name"] = req.Name
	}
	if trackedProc != nil {
		response["startedAt"] = trackedProc.StartedAt
	}
	if stdoutPath != "" {
		response["stdoutLogPath"] = stdoutPath
	}
	if stderrPath != "" {
		response["stderrLogPath"] = stderrPath
	}
	response["withStdin"] = req.WithStdin

	h.JSONCreated(w, response)
}

// processStdinPipe wraps an *io.PipeWriter so we can close it idempotently.
type processStdinPipe struct {
	w        *io.PipeWriter
	closeMu  sync.Mutex
	isClosed bool
}

func newStdinPipe() (*io.PipeReader, *processStdinPipe) {
	r, w := io.Pipe()
	return r, &processStdinPipe{w: w}
}

func (p *processStdinPipe) Write(b []byte) (int, error) {
	return p.w.Write(b)
}

func (p *processStdinPipe) Close() error {
	p.closeMu.Lock()
	defer p.closeMu.Unlock()
	if p.isClosed {
		return nil
	}
	p.isClosed = true
	return p.w.Close()
}

// ListProcesses handles listing processes for a sandbox.
// [OT-P0-008] Process/Session Tracking
func (h *Handlers) ListProcesses(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		h.JSONError(w, "invalid sandbox ID", http.StatusBadRequest)
		return
	}

	// Verify sandbox exists
	_, err = h.Service.Get(r.Context(), id)
	if h.HandleDomainError(w, err) {
		return
	}

	if h.ProcessTracker == nil {
		h.JSONError(w, "process tracking not available", http.StatusServiceUnavailable)
		return
	}

	// Get all processes, optionally filter by running only
	runningOnly := r.URL.Query().Get("running") == "true"
	var procs []*process.TrackedProcess
	if runningOnly {
		procs = h.ProcessTracker.GetRunningProcesses(id)
	} else {
		procs = h.ProcessTracker.GetProcesses(id)
	}

	h.JSONSuccess(w, map[string]interface{}{
		"processes": procs,
		"total":     len(procs),
		"running":   h.ProcessTracker.GetActiveCount(id),
	})
}

// KillProcess handles killing a specific process.
// [OT-P0-008] Process/Session Tracking
func (h *Handlers) KillProcess(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		h.JSONError(w, "invalid sandbox ID", http.StatusBadRequest)
		return
	}

	var pidInt int
	if _, err := parsePositiveInt(vars["pid"], &pidInt); err != nil {
		h.JSONError(w, "invalid PID", http.StatusBadRequest)
		return
	}

	// Verify sandbox exists
	_, err = h.Service.Get(r.Context(), id)
	if h.HandleDomainError(w, err) {
		return
	}

	if h.ProcessTracker == nil {
		h.JSONError(w, "process tracking not available", http.StatusServiceUnavailable)
		return
	}

	err = h.ProcessTracker.KillProcess(r.Context(), id, pidInt)
	if err != nil {
		h.JSONError(w, err.Error(), http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// KillAllProcesses handles killing all processes for a sandbox.
// [OT-P0-008] Process/Session Tracking
func (h *Handlers) KillAllProcesses(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		h.JSONError(w, "invalid sandbox ID", http.StatusBadRequest)
		return
	}

	// Verify sandbox exists
	_, err = h.Service.Get(r.Context(), id)
	if h.HandleDomainError(w, err) {
		return
	}

	if h.ProcessTracker == nil {
		h.JSONError(w, "process tracking not available", http.StatusServiceUnavailable)
		return
	}

	killed, errs := h.ProcessTracker.KillAll(r.Context(), id)

	response := map[string]interface{}{
		"killed": killed,
	}
	if len(errs) > 0 {
		errStrings := make([]string, len(errs))
		for i, e := range errs {
			errStrings[i] = e.Error()
		}
		response["errors"] = errStrings
	}

	h.JSONSuccess(w, response)
}

// ProcessStats handles getting aggregate process statistics.
// [OT-P0-008] Process/Session Tracking
func (h *Handlers) ProcessStats(w http.ResponseWriter, r *http.Request) {
	if h.ProcessTracker == nil {
		h.JSONError(w, "process tracking not available", http.StatusServiceUnavailable)
		return
	}

	stats := h.ProcessTracker.GetAllStats()
	h.JSONSuccess(w, stats)
}

// BwrapInfo handles getting bubblewrap capabilities information.
// [OT-P0-003] Bubblewrap Process Isolation
func (h *Handlers) BwrapInfo(w http.ResponseWriter, r *http.Request) {
	info, err := driver.GetBwrapInfo(r.Context())
	if err != nil {
		h.JSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.JSONSuccess(w, info)
}

// --- Process Log Endpoints (Phase 2) ---

// GetProcessLogs returns logs for a specific stream of a process.
// Required query parameter:
//   - stream: "stdout" or "stderr"
//
// Optional:
//   - tail: number of lines from end (default: all)
//   - offset: byte offset to start reading from
func (h *Handlers) GetProcessLogs(w http.ResponseWriter, r *http.Request) {
	id, pid, stream, ok := h.parseProcessLogParams(w, r)
	if !ok {
		return
	}
	if h.ProcessLogger == nil {
		h.JSONError(w, "process logging not available", http.StatusServiceUnavailable)
		return
	}

	var tail int
	var offset int64
	if tailStr := r.URL.Query().Get("tail"); tailStr != "" {
		tail, _ = strconv.Atoi(tailStr)
	}
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		offset, _ = strconv.ParseInt(offsetStr, 10, 64)
	}

	logInfo, err := h.ProcessLogger.GetLog(id, pid, stream)
	if err != nil {
		h.JSONError(w, fmt.Sprintf("log not found: %v", err), http.StatusNotFound)
		return
	}

	content, err := h.ProcessLogger.ReadLog(id, pid, stream, tail, offset)
	if err != nil {
		h.JSONError(w, fmt.Sprintf("failed to read log: %v", err), http.StatusInternalServerError)
		return
	}

	h.JSONSuccess(w, map[string]interface{}{
		"pid":       pid,
		"sandboxId": id,
		"stream":    string(stream),
		"path":      logInfo.Path,
		"sizeBytes": logInfo.SizeBytes,
		"isActive":  logInfo.IsActive,
		"content":   string(content),
	})
}

// StreamProcessLogs streams one stream of a process's log via Server-Sent
// Events. Required query parameter: stream=stdout|stderr.
//
// Sends `data:` events as bytes are written. When the process terminates,
// sends a single `event: exit` carrying ExitInfo as JSON, then `event: end`
// and closes the connection.
func (h *Handlers) StreamProcessLogs(w http.ResponseWriter, r *http.Request) {
	id, pid, stream, ok := h.parseProcessLogParams(w, r)
	if !ok {
		return
	}
	if h.ProcessLogger == nil {
		h.JSONError(w, "process logging not available", http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		h.JSONError(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	ctx := r.Context()

	// Stream log content. StreamLog replays existing disk content first
	// so a late subscriber doesn't lose what was already written, then
	// fans out new chunks as they're written.
	streamErr := h.ProcessLogger.StreamLog(ctx, id, pid, stream, func(chunk []byte) {
		fmt.Fprintf(w, "data: %s\n\n", string(chunk))
		flusher.Flush()
	})

	if streamErr != nil && !errors.Is(streamErr, context.Canceled) && !errors.Is(streamErr, context.DeadlineExceeded) {
		fmt.Fprintf(w, "event: error\ndata: %s\n\n", streamErr.Error())
		flusher.Flush()
	}

	// If we have a process tracker and the process has terminated, push
	// the structured exit info so consumers don't need a second request.
	if h.ProcessTracker != nil {
		if info := h.ProcessTracker.GetExitInfo(id, pid); info != nil {
			payload, _ := json.Marshal(info)
			fmt.Fprintf(w, "event: exit\ndata: %s\n\n", string(payload))
			flusher.Flush()
		}
	}

	fmt.Fprintf(w, "event: end\ndata: stream closed\n\n")
	flusher.Flush()
}

// PostProcessStdin streams the request body to the process's stdin pipe.
// Optional query parameter: close=true closes the stdin pipe after the
// body is consumed (signaling EOF to the process). Without close=true,
// the pipe remains open for subsequent writes.
//
// Returns 404 when the PID is not tracked, 409 when the process was not
// started with WithStdin, and 200 with bytesWritten on success.
func (h *Handlers) PostProcessStdin(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		h.JSONError(w, "invalid sandbox ID", http.StatusBadRequest)
		return
	}
	pid, err := strconv.Atoi(vars["pid"])
	if err != nil || pid <= 0 {
		h.JSONError(w, "invalid PID", http.StatusBadRequest)
		return
	}
	if _, err := h.Service.Get(r.Context(), id); h.HandleDomainError(w, err) {
		return
	}
	if h.ProcessTracker == nil {
		h.JSONError(w, "process tracking not available", http.StatusServiceUnavailable)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.JSONError(w, fmt.Sprintf("read body: %v", err), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	written := 0
	if len(body) > 0 {
		n, werr := h.ProcessTracker.WriteStdin(id, pid, body)
		if werr != nil {
			status := http.StatusNotFound
			// "no stdin pipe" is a state error; surface 409.
			if msg := werr.Error(); len(msg) > 0 {
				if containsAny(msg, []string{"no stdin pipe"}) {
					status = http.StatusConflict
				}
			}
			h.JSONError(w, werr.Error(), status)
			return
		}
		written = n
	}

	closed := false
	if r.URL.Query().Get("close") == "true" {
		if err := h.ProcessTracker.CloseStdin(id, pid); err != nil {
			h.JSONError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		closed = true
	}

	h.JSONSuccess(w, map[string]interface{}{
		"pid":          pid,
		"sandboxId":    id,
		"bytesWritten": written,
		"closed":       closed,
	})
}

// parseProcessLogParams parses the sandbox ID, PID, and stream from a
// process-log request. On error it writes the response and returns ok=false.
func (h *Handlers) parseProcessLogParams(w http.ResponseWriter, r *http.Request) (uuid.UUID, int, process.Stream, bool) {
	vars := mux.Vars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		h.JSONError(w, "invalid sandbox ID", http.StatusBadRequest)
		return uuid.Nil, 0, "", false
	}
	pid, err := strconv.Atoi(vars["pid"])
	if err != nil || pid <= 0 {
		h.JSONError(w, "invalid PID", http.StatusBadRequest)
		return uuid.Nil, 0, "", false
	}
	if _, err := h.Service.Get(r.Context(), id); h.HandleDomainError(w, err) {
		return uuid.Nil, 0, "", false
	}
	streamStr := r.URL.Query().Get("stream")
	if streamStr == "" {
		h.JSONError(w, "missing required query parameter: stream=stdout|stderr", http.StatusBadRequest)
		return uuid.Nil, 0, "", false
	}
	stream := process.Stream(streamStr)
	if err := stream.Validate(); err != nil {
		h.JSONError(w, err.Error(), http.StatusBadRequest)
		return uuid.Nil, 0, "", false
	}
	return id, pid, stream, true
}

// containsAny returns true if needle appears as a substring of haystack
// for any of the given needles.
func containsAny(haystack string, needles []string) bool {
	for _, n := range needles {
		if n == "" {
			continue
		}
		if indexOf(haystack, n) >= 0 {
			return true
		}
	}
	return false
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

// ListProcessLogs returns all log files for a sandbox.
func (h *Handlers) ListProcessLogs(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		h.JSONError(w, "invalid sandbox ID", http.StatusBadRequest)
		return
	}

	// Verify sandbox exists
	_, err = h.Service.Get(r.Context(), id)
	if h.HandleDomainError(w, err) {
		return
	}

	if h.ProcessLogger == nil {
		h.JSONError(w, "process logging not available", http.StatusServiceUnavailable)
		return
	}

	logs, err := h.ProcessLogger.ListLogs(id)
	if err != nil {
		h.JSONError(w, fmt.Sprintf("failed to list logs: %v", err), http.StatusInternalServerError)
		return
	}

	h.JSONSuccess(w, map[string]interface{}{
		"logs":      logs,
		"total":     len(logs),
		"sandboxId": id,
	})
}

// evaluateProtectedGitAllowlist returns a non-empty denial reason when the
// requested command is a `git` invocation that is NOT in the configured
// allowlist. Returns "" when the command is allowed (either not git, or
// allowlist empty/wildcard, or verb in the list).
//
// Wraps the protected-sandbox-git-and-network-guardrails contract: agents
// should use Git Control Tower for mutating operations; direct `git` is
// limited to read-only verbs by default.
func evaluateProtectedGitAllowlist(cfg types.ProtectedConfig, command string, args []string) string {
	if len(cfg.GitAllowlist) == 0 {
		return ""
	}
	// Wildcard: explicit opt-out for callers that want unrestricted git.
	for _, v := range cfg.GitAllowlist {
		if v == "*" {
			return ""
		}
	}
	// Resolve command basename so /usr/bin/git and git both match.
	base := command
	if idx := lastSlash(command); idx >= 0 {
		base = command[idx+1:]
	}
	if base != "git" {
		return ""
	}
	verb := firstArg(args)
	if verb == "" {
		return "git invoked without a verb; allowlist enforces a verb-level policy. Use one of: " + joinAllowlist(cfg.GitAllowlist)
	}
	for _, allowed := range cfg.GitAllowlist {
		if allowed == verb {
			return ""
		}
	}
	return "git verb \"" + verb + "\" is not in the protected-mode allowlist (" + joinAllowlist(cfg.GitAllowlist) + "). For mutating git operations, route through Git Control Tower instead of direct git invocations."
}

func firstArg(args []string) string {
	if len(args) == 0 {
		return ""
	}
	return args[0]
}

func lastSlash(s string) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == '/' {
			return i
		}
	}
	return -1
}

func joinAllowlist(list []string) string {
	out := ""
	for i, v := range list {
		if i > 0 {
			out += ", "
		}
		out += v
	}
	return out
}

// writeJSONStatus is a small helper that mirrors h.JSON but lets callers
// pick an HTTP status. The handlers package's JSON helper hardcodes 200.
func writeJSONStatus(w http.ResponseWriter, status int, body map[string]any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
