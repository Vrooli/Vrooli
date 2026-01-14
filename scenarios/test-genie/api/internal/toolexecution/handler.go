package toolexecution

import (
	"encoding/json"
	"net/http"
)

// Handler provides HTTP handlers for the Tool Execution Protocol.
type Handler struct {
	executor *ServerExecutor
}

// NewHandler creates a new Handler with the given executor.
func NewHandler(executor *ServerExecutor) *Handler {
	return &Handler{executor: executor}
}

// Execute handles POST /api/v1/tools/execute requests.
// This is the main entry point for the Tool Execution Protocol.
func (h *Handler) Execute(w http.ResponseWriter, r *http.Request) {
	// Handle CORS preflight
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Parse request
	var req ExecuteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeJSON(w, ErrorResult("invalid request body: "+err.Error(), CodeInvalidArgs), http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.ToolName == "" {
		h.writeJSON(w, ErrorResult("tool_name is required", CodeInvalidArgs), http.StatusBadRequest)
		return
	}

	// Execute the tool
	result, err := h.executor.Execute(r.Context(), req.ToolName, req.Arguments)
	if err != nil {
		// Unexpected error (executor should return errors via ExecutionResult)
		h.writeJSON(w, ErrorResult("internal error: "+err.Error(), CodeInternalError), http.StatusInternalServerError)
		return
	}

	// Determine HTTP status based on result
	status := http.StatusOK
	if !result.Success {
		switch result.Code {
		case CodeNotFound:
			status = http.StatusNotFound
		case CodeInvalidArgs:
			status = http.StatusBadRequest
		case CodeConflict:
			status = http.StatusConflict
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
	json.NewEncoder(w).Encode(data)
}
