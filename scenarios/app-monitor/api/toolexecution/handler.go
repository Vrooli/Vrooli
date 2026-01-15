// Package toolexecution provides HTTP handlers for the Tool Execution Protocol.
package toolexecution

import (
	"net/http"

	"github.com/gin-gonic/gin"
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
func (h *Handler) Execute(c *gin.Context) {
	// Parse request
	var req ExecuteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResult("invalid request body: "+err.Error(), CodeInvalidArgs))
		return
	}

	// Validate required fields
	if req.ToolName == "" {
		c.JSON(http.StatusBadRequest, ErrorResult("tool_name is required", CodeInvalidArgs))
		return
	}

	// Execute the tool
	result, err := h.executor.Execute(c.Request.Context(), req.ToolName, req.Arguments)
	if err != nil {
		// Unexpected error (executor should return errors via ExecutionResult)
		c.JSON(http.StatusInternalServerError, ErrorResult("internal error: "+err.Error(), CodeInternalError))
		return
	}

	// Determine HTTP status based on result
	status := http.StatusOK
	if result.IsAsync {
		status = http.StatusAccepted
	} else if !result.Success {
		switch result.Code {
		case CodeNotFound:
			status = http.StatusNotFound
		case CodeInvalidArgs:
			status = http.StatusBadRequest
		case CodeConflict:
			status = http.StatusConflict
		case CodeUnknownTool:
			status = http.StatusNotFound
		default:
			status = http.StatusInternalServerError
		}
	}

	c.JSON(status, result)
}
