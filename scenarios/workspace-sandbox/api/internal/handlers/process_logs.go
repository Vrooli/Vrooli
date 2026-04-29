package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	"workspace-sandbox/internal/process"
	"workspace-sandbox/internal/sse"
)

// process_logs.go: per-process log retrieval + SSE streaming.
//
// SSE wire-format details (Content-Type, Flusher assertion, multi-line
// data encoding, write-deadline reset, trailing `event: end`) live in
// internal/sse. This file only chooses *what* to emit and in what
// order; *how* it reaches the wire is the sse package's job. That
// split is the point of Round 4 Phase 5 — the 2026-04-28 SSE Flusher
// bug shipped because every handler re-derived the wire format
// inline; one missed assertion was enough to break streaming.

// exitInfoWaitTimeout caps how long StreamProcessLogs waits after the
// log stream closes for the wait reaper to record exit info. 5s is
// well above the typical reap latency (microseconds) but short enough
// to fail fast when the reaper goroutine genuinely never runs (a bug).
const exitInfoWaitTimeout = 5 * time.Second

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

// StreamProcessLogs streams one stream of a process's log via
// Server-Sent Events. Required query parameter: stream=stdout|stderr.
//
// Wire contract (enforced by the sse package):
//
//   - Replays existing on-disk content as default-event frames.
//   - On any underlying stream error other than client cancellation,
//     emits one `event: error` frame.
//   - Once the process has exited, emits exactly one `event: exit`
//     frame carrying ExitInfo as JSON.
//   - The closing `event: end` is emitted by sse.Writer.Close.
func (h *Handlers) StreamProcessLogs(w http.ResponseWriter, r *http.Request) {
	id, pid, stream, ok := h.parseProcessLogParams(w, r)
	if !ok {
		return
	}
	if h.ProcessLogger == nil {
		h.JSONError(w, "process logging not available", http.StatusServiceUnavailable)
		return
	}

	sw, err := sse.NewHTTPWriter(w)
	if err != nil {
		// Only ErrFlusherUnsupported lands here. The 2026-04-28
		// Flusher regression now surfaces as a structured 500
		// instead of a silently-buffered stream.
		h.JSONError(w, "streaming not supported", http.StatusInternalServerError)
		return
	}
	defer sw.Close()

	ctx := r.Context()

	// StreamLog replays existing disk content first so a late subscriber
	// doesn't lose what was already written, then fans out new chunks.
	streamErr := h.ProcessLogger.StreamLog(ctx, id, pid, stream, func(chunk []byte) {
		_ = sw.WriteData(chunk)
	})

	if streamErr != nil && !errors.Is(streamErr, context.Canceled) && !errors.Is(streamErr, context.DeadlineExceeded) {
		_ = sw.WriteEvent("error", []byte(streamErr.Error()))
	}

	// Push the structured exit info so consumers don't need a second
	// request. Wait for it (bounded) instead of best-effort GetExitInfo:
	// fast-failing processes (e.g. bwrap chdir errors that exit in
	// <100ms) used to lose this race and never sent `event: exit`.
	if h.ProcessTracker != nil {
		waitCtx, cancel := context.WithTimeout(context.Background(), exitInfoWaitTimeout)
		info, waitErr := h.ProcessTracker.WaitForExit(waitCtx, id, pid)
		cancel()
		switch {
		case info != nil:
			payload, _ := json.Marshal(info)
			_ = sw.WriteEvent("exit", payload)
		case waitErr != nil:
			_ = sw.WriteEvent("error", []byte("exit info unavailable: "+waitErr.Error()))
		}
	}
}

// ListProcessLogs returns all log files for a sandbox.
func (h *Handlers) ListProcessLogs(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		h.JSONError(w, "invalid sandbox ID", http.StatusBadRequest)
		return
	}

	if _, err := h.Service.Get(r.Context(), id); h.HandleDomainError(w, err) {
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

// parseProcessLogParams parses the sandbox ID, PID, and stream from a
// process-log request. On error it writes the response and returns
// ok=false so the handler can short-circuit.
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
