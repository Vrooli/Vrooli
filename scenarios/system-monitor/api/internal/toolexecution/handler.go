// Package toolexecution provides the tool execution service for system-monitor.
package toolexecution

import (
	"encoding/json"
	"log/slog"
	"net/http"
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
		h.writeError(w, "invalid request body", ErrorCodeInvalidArgs, http.StatusBadRequest)
		return
	}

	if req.ToolName == "" {
		h.writeError(w, "tool_name is required", ErrorCodeInvalidArgs, http.StatusBadRequest)
		return
	}

	h.log.Debug("executing tool", "tool", req.ToolName)

	result, err := h.executor.Execute(r.Context(), req.ToolName, req.Arguments)
	if err != nil {
		h.log.Error("tool execution failed", "tool", req.ToolName, "error", err)
		h.writeError(w, err.Error(), ErrorCodeInternalError, http.StatusInternalServerError)
		return
	}

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

	h.writeJSON(w, result, status)
}

// writeJSON writes a JSON response with the given status code.
func (h *Handler) writeJSON(w http.ResponseWriter, data interface{}, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.log.Error("failed to encode response", "error", err)
	}
}

// writeError writes a JSON error response.
func (h *Handler) writeError(w http.ResponseWriter, message string, code string, status int) {
	h.writeJSON(w, &ExecutionResult{
		Success: false,
		Error:   message,
		Code:    code,
	}, status)
}
