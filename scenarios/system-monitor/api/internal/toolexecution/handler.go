// Package toolexecution provides the tool execution service for system-monitor.
package toolexecution

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/httputil"
)

// Handler handles HTTP requests for tool execution.
type Handler struct {
	executor *ServerExecutor
	log      *slog.Logger
}

// NewHandler creates a new tool execution handler.
func NewHandler(executor *ServerExecutor, log *slog.Logger) *Handler {
	if log == nil {
		log = slog.Default()
	}
	return &Handler{
		executor: executor,
		log:      log,
	}
}

// Execute handles POST /api/v1/tools/execute
func (h *Handler) Execute(w http.ResponseWriter, r *http.Request) {
	var req ExecuteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		result := NewErrorResult(ErrorCodeInvalidArgs, "invalid request body")
		result.RequestID = r.Header.Get("X-Request-ID")
		httputil.JSONWithStatus(w, http.StatusBadRequest, result) //nolint:errcheck
		return
	}

	if req.ToolName == "" {
		result := NewErrorResult(ErrorCodeInvalidArgs, "tool_name is required")
		result.RequestID = r.Header.Get("X-Request-ID")
		httputil.JSONWithStatus(w, http.StatusBadRequest, result) //nolint:errcheck
		return
	}

	h.log.Debug("executing tool", "tool", req.ToolName)

	result, err := h.executor.Execute(r.Context(), req.ToolName, req.Arguments)
	if err != nil {
		h.log.Error("tool execution failed", "tool", req.ToolName, "error", err)
		result = NewErrorResult(ErrorCodeInternalError, "An internal error occurred")
		result.RequestID = r.Header.Get("X-Request-ID")
		httputil.JSONWithStatus(w, http.StatusInternalServerError, result) //nolint:errcheck
		return
	}

	// Inject request ID
	result.RequestID = r.Header.Get("X-Request-ID")

	// Map error codes to HTTP status
	status := http.StatusOK
	if !result.Success {
		switch result.Code {
		case ErrorCodeNotFound:
			status = http.StatusNotFound
		case ErrorCodeInvalidArgs:
			status = http.StatusBadRequest
		case ErrorCodeConflict:
			status = http.StatusConflict
		case ErrorCodeUnknownTool:
			status = http.StatusNotFound
		case ErrorCodeCooldown:
			status = http.StatusTooManyRequests
		case ErrorCodeUnavailable:
			status = http.StatusServiceUnavailable
		default:
			status = http.StatusInternalServerError
		}
	}

	if err := httputil.JSONWithStatus(w, status, result); err != nil {
		h.log.Error("failed to encode response", "error", err)
	}
}
