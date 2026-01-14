// Package toolexecution implements the Tool Execution Protocol for scenario-to-cloud.
package toolexecution

import (
	"encoding/json"
	"net/http"
)

// Handler handles HTTP requests for tool execution.
type Handler struct {
	executor ToolExecutor
}

// NewHandler creates a new Handler with the given executor.
func NewHandler(executor ToolExecutor) *Handler {
	return &Handler{executor: executor}
}

// Execute handles POST /api/v1/tools/execute requests.
// This is the main entry point for the Tool Execution Protocol.
func (h *Handler) Execute(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers for all responses
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Accept")

	// Handle CORS preflight
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Parse request
	var req ExecuteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeResult(w, http.StatusBadRequest, ErrorResult("invalid JSON request body", CodeInvalidArgs))
		return
	}

	// Validate request
	if req.ToolName == "" {
		h.writeResult(w, http.StatusBadRequest, ErrorResult("tool_name is required", CodeInvalidArgs))
		return
	}

	// Execute the tool
	result, err := h.executor.Execute(r.Context(), req.ToolName, req.Arguments)
	if err != nil {
		// Execution error (not a tool error)
		h.writeResult(w, http.StatusInternalServerError, ErrorResult(err.Error(), CodeInternalError))
		return
	}

	// Determine HTTP status based on result
	status := http.StatusOK
	if !result.Success {
		switch result.Code {
		case CodeNotFound:
			status = http.StatusNotFound
		case CodeConflict:
			status = http.StatusConflict
		case CodeInvalidArgs, CodeValidation:
			status = http.StatusBadRequest
		case CodeUnknownTool:
			status = http.StatusNotFound
		default:
			status = http.StatusInternalServerError
		}
	} else if result.IsAsync {
		status = http.StatusAccepted
	}

	h.writeResult(w, status, result)
}

// writeResult writes an ExecutionResult as JSON.
func (h *Handler) writeResult(w http.ResponseWriter, status int, result *ExecutionResult) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(result)
}
