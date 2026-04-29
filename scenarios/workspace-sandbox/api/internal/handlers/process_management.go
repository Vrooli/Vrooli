package handlers

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	"workspace-sandbox/internal/driver"
	"workspace-sandbox/internal/process"
)

// process_management.go: small process-control endpoints — list,
// kill, kill-all, post-stdin, plus the read-only diagnostics
// (ProcessStats, BwrapInfo). Each one is short and follows the same
// shape: parse path vars → verify sandbox → call tracker/driver →
// JSON result.

// ListProcesses returns the tracked processes for a sandbox.
// [OT-P0-008] Process/Session Tracking
func (h *Handlers) ListProcesses(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		h.JSONError(w, "invalid sandbox ID", http.StatusBadRequest)
		return
	}

	if _, err := h.Service.Get(r.Context(), id); h.HandleDomainError(w, err) {
		return
	}

	if h.ProcessTracker == nil {
		h.JSONError(w, "process tracking not available", http.StatusServiceUnavailable)
		return
	}

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

// KillProcess kills a specific tracked process by PID.
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

	if _, err := h.Service.Get(r.Context(), id); h.HandleDomainError(w, err) {
		return
	}

	if h.ProcessTracker == nil {
		h.JSONError(w, "process tracking not available", http.StatusServiceUnavailable)
		return
	}

	if err := h.ProcessTracker.KillProcess(r.Context(), id, pidInt); err != nil {
		h.JSONError(w, err.Error(), http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// KillAllProcesses kills every tracked process for a sandbox and
// returns the number killed plus any per-process errors.
// [OT-P0-008] Process/Session Tracking
func (h *Handlers) KillAllProcesses(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		h.JSONError(w, "invalid sandbox ID", http.StatusBadRequest)
		return
	}

	if _, err := h.Service.Get(r.Context(), id); h.HandleDomainError(w, err) {
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

// ProcessStats returns aggregate process statistics across all sandboxes.
// [OT-P0-008] Process/Session Tracking
func (h *Handlers) ProcessStats(w http.ResponseWriter, r *http.Request) {
	if h.ProcessTracker == nil {
		h.JSONError(w, "process tracking not available", http.StatusServiceUnavailable)
		return
	}

	h.JSONSuccess(w, h.ProcessTracker.GetAllStats())
}

// BwrapInfo returns bubblewrap capabilities and version info.
// [OT-P0-003] Bubblewrap Process Isolation
func (h *Handlers) BwrapInfo(w http.ResponseWriter, r *http.Request) {
	info, err := driver.GetBwrapInfo(r.Context(), h.Starter)
	if err != nil {
		h.JSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.JSONSuccess(w, info)
}

// PostProcessStdin streams the request body to the process's stdin
// pipe. Optional query parameter: close=true closes the stdin pipe
// after the body is consumed (signaling EOF to the process). Without
// close=true, the pipe remains open for subsequent writes.
//
// Returns 404 when the PID is not tracked, 409 when the process was
// not started with WithStdin, and 200 with bytesWritten on success.
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
			if strings.Contains(werr.Error(), "no stdin pipe") {
				status = http.StatusConflict
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
