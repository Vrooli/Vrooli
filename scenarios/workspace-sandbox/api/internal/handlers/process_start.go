package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	"workspace-sandbox/internal/driver"
	driverexec "workspace-sandbox/internal/driver/exec"
	"workspace-sandbox/internal/process"
	"workspace-sandbox/internal/runtime"
	"workspace-sandbox/internal/types"
)

// process_start.go: the StartProcess endpoint for background processes,
// plus the small stdin-pipe helper it uses to wire request-body bytes
// into the running process.

// StartProcessRequest is the body for POST /sandboxes/{id}/processes.
type StartProcessRequest struct {
	Command      string            `json:"command"`
	Args         []string          `json:"args,omitempty"`
	AllowNetwork bool              `json:"allowNetwork,omitempty"`
	Env          map[string]string `json:"env,omitempty"`
	WorkingDir   string            `json:"workingDir,omitempty"`
	SessionID    string            `json:"sessionId,omitempty"`
	Name         string            `json:"name,omitempty"` // Optional friendly name

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
	MemoryLimitMB int `json:"memoryLimitMB,omitempty"`
	CPUTimeSec    int `json:"cpuTimeSec,omitempty"`
	MaxProcesses  int `json:"maxProcesses,omitempty"`
	MaxOpenFiles  int `json:"maxOpenFiles,omitempty"`
}

// StartProcess starts a background process in a sandbox.
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

	sb, err := h.Service.Get(r.Context(), id)
	if h.HandleDomainError(w, err) {
		return
	}

	if !types.CanRunProcess(sb.Status) {
		h.JSONError(w, "sandbox must be active to start processes", http.StatusConflict)
		return
	}

	// Protected-mode git allowlist enforcement. Mirrors Exec — agent
	// processes launched here would otherwise be able to bypass the
	// /exec allowlist by spawning git directly.
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

	if req.AllowNetwork {
		cfg.AllowNetwork = true
	}

	// Background processes ignore TimeoutSec — use manual kill.
	requestedLimits := driverexec.ResourceLimits{
		MemoryLimitMB: req.MemoryLimitMB,
		CPUTimeSec:    req.CPUTimeSec,
		MaxProcesses:  req.MaxProcesses,
		MaxOpenFiles:  req.MaxOpenFiles,
		TimeoutSec:    0,
	}
	cfg.ResourceLimits = runtime.ApplyResourceLimitDefaults(requestedLimits, h.Config.Execution)
	cfg.ResourceLimits.TimeoutSec = 0

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

	var stdinWriter *processStdinPipe
	if req.WithStdin {
		stdinReader, sw := newStdinPipe()
		cfg.StdinReader = stdinReader
		stdinWriter = sw
	}

	// onExit fires from the driver's wait reaper after the spawned process exits.
	// It records ExitInfo on the tracker (which closes the per-process exit
	// channel and unblocks subscribers / SSE consumers) and finalises the
	// log pair so subscribers see EOF.
	pidCh := make(chan int, 1)
	var onExitOnce sync.Once
	cfg.OnExit = func(exitCode, signal int, oomKilled bool) {
		var pid int
		select {
		case pid = <-pidCh:
		case <-time.After(2 * time.Second):
			// PID never published — process must have failed to start in a
			// way the driver still surfaced; nothing to record.
			return
		}
		// Republish so any racing reads still see it.
		select {
		case pidCh <- pid:
		default:
		}
		onExitOnce.Do(func() {
			// StoppedAt is left zero so the tracker stamps it via its
			// injected clock — keeps a single source of truth for the
			// exit timestamp and avoids handlers needing a clock of
			// their own for fallback exit info.
			info := process.ExitInfo{ExitCode: exitCode, Signal: signal, OOMKilled: oomKilled}
			if h.ProcessTracker != nil {
				h.ProcessTracker.RecordExit(id, pid, info)
			}
			if h.ProcessLogger != nil {
				_ = h.ProcessLogger.CloseLogPair(id, pid, info)
			}
		})
	}

	d := h.Driver()
	level := d.RequiredContainment()
	pid, backendID, err := driverexec.StartProcess(r.Context(), h.Starter, sb, level, cfg, req.Command, req.Args...)
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

	// Effective containment: the backend that actually launched this process
	// plus the enforcements it provides on this host. Stamped on the tracked
	// process (for /processes) and echoed in the response so the launch's
	// provenance is truth, not inference.
	containmentInfo, _ := driver.GetContainmentInfo(r.Context(), h.Starter)
	effectiveContainment := driver.EffectiveContainment(level, backendID, containmentInfo)

	var stdoutPath, stderrPath string
	if pendingPair != nil {
		var logErr error
		stdoutPath, stderrPath, logErr = h.ProcessLogger.FinalizePair(pendingPair, pid)
		if logErr != nil {
			h.JSONError(w, logErr.Error(), http.StatusInternalServerError)
			return
		}
	}

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
		_ = h.ProcessTracker.SetContainment(id, pid, effectiveContainment)
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
		"pid":         pid,
		"sandboxId":   id,
		"command":     req.Command,
		"containment": effectiveContainment,
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

// processStdinPipe wraps an *io.PipeWriter so the stdin endpoint can
// close it idempotently (a double-close on a real *io.PipeWriter would
// not panic, but we want to track state for the close-on-EOF flag).
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
