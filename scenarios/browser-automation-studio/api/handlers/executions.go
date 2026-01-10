package handlers

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/constants"
	"github.com/vrooli/browser-automation-studio/internal/protoconv"
	basexecution "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/execution"
	"google.golang.org/protobuf/proto"
)

// GetExecutionScreenshots handles GET /api/v1/executions/{id}/screenshots
func (h *Handler) GetExecutionScreenshots(w http.ResponseWriter, r *http.Request) {
	executionID, ok := h.parseUUIDParam(w, r, "id", ErrInvalidExecutionID)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), constants.DefaultRequestTimeout)
	defer cancel()

	screenshots, err := h.executionService.GetExecutionScreenshots(ctx, executionID)
	if err != nil {
		h.log.WithError(err).WithField("execution_id", executionID).Error("Failed to get screenshots")
		h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "get_screenshots", "execution_id": executionID.String()}))
		return
	}
	if len(screenshots) == 0 {
		if details, missing := h.resolveArtifactGap(ctx, executionID, "screenshots"); missing {
			h.respondError(w, ErrExecutionArtifactsUnavailable.WithDetails(details))
			return
		}
	}

	h.respondProto(w, http.StatusOK, &basexecution.GetScreenshotsResponse{
		ExecutionId: executionID.String(),
		Screenshots: screenshots,
		Total:       int32(len(screenshots)),
	})
}

// GetExecutionTimeline handles GET /api/v1/executions/{id}/timeline
func (h *Handler) GetExecutionTimeline(w http.ResponseWriter, r *http.Request) {
	executionID, ok := h.parseUUIDParam(w, r, "id", ErrInvalidExecutionID)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), constants.ExtendedRequestTimeout)
	defer cancel()

	// Preferred: proto timeline persisted to disk.
	if pbTimeline, err := h.executionService.GetExecutionTimelineProto(ctx, executionID); err == nil && pbTimeline != nil {
		if len(pbTimeline.Entries) == 0 && len(pbTimeline.Logs) == 0 {
			if details, missing := h.resolveArtifactGap(ctx, executionID, "timeline"); missing {
				h.respondError(w, ErrExecutionArtifactsUnavailable.WithDetails(details))
				return
			}
		}
		h.respondProto(w, http.StatusOK, pbTimeline)
		return
	}

	// Fallback: legacy result.json timeline conversion.
	timeline, err := h.executionService.GetExecutionTimeline(ctx, executionID)
	if err != nil {
		h.log.WithError(err).WithField("execution_id", executionID).Error("Failed to get execution timeline")
		h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "get_timeline", "execution_id": executionID.String()}))
		return
	}
	pbTimeline, err := protoconv.TimelineToProto(timeline)
	if err != nil {
		if h.log != nil {
			h.log.WithError(err).WithField("execution_id", executionID).Error("Failed to convert execution timeline to proto")
		}
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
			"operation": "timeline_to_proto",
			"error":     err.Error(),
		}))
		return
	}
	if len(pbTimeline.Entries) == 0 && len(pbTimeline.Logs) == 0 {
		if details, missing := h.resolveArtifactGap(ctx, executionID, "timeline"); missing {
			h.respondError(w, ErrExecutionArtifactsUnavailable.WithDetails(details))
			return
		}
	}
	h.respondProto(w, http.StatusOK, pbTimeline)
}

func (h *Handler) resolveArtifactGap(ctx context.Context, executionID uuid.UUID, artifact string) (map[string]string, bool) {
	execution, err := h.executionService.GetExecution(ctx, executionID)
	if err != nil {
		return map[string]string{
			"execution_id": executionID.String(),
			"artifact":     artifact,
			"reason":       "execution_lookup_failed",
			"suggestion":   "The execution record could not be found. It may have been deleted or the ID is incorrect.",
		}, true
	}

	resultPath := strings.TrimSpace(execution.ResultPath)
	if resultPath == "" {
		return map[string]string{
			"execution_id": executionID.String(),
			"artifact":     artifact,
			"reason":       "artifacts_not_saved",
			"suggestion":   "This execution did not save artifacts. This can happen if the execution was stopped early or encountered an error during setup.",
		}, true
	}

	if _, err := os.Stat(resultPath); err != nil {
		if os.IsNotExist(err) {
			return map[string]string{
				"execution_id": executionID.String(),
				"artifact":     artifact,
				"reason":       "artifacts_deleted",
				"suggestion":   "The execution artifacts have been deleted from storage. Re-run the workflow to generate new artifacts.",
			}, true
		}
		return map[string]string{
			"execution_id": executionID.String(),
			"artifact":     artifact,
			"reason":       "artifacts_inaccessible",
			"suggestion":   "The execution artifacts exist but cannot be read. This may indicate a storage permission issue.",
		}, true
	}

	if artifact == "timeline" {
		timelinePath := filepath.Join(filepath.Dir(resultPath), "timeline.proto.json")
		if _, err := os.Stat(timelinePath); err != nil {
			if os.IsNotExist(err) {
				return map[string]string{
					"execution_id": executionID.String(),
					"artifact":     artifact,
					"reason":       "timeline_not_generated",
					"suggestion":   "The timeline was not generated for this execution. This can happen with older executions or if the workflow completed too quickly.",
				}, true
			}
			return map[string]string{
				"execution_id": executionID.String(),
				"artifact":     artifact,
				"reason":       "timeline_inaccessible",
				"suggestion":   "The timeline file exists but cannot be read. This may indicate a storage permission issue.",
			}, true
		}
	}

	return map[string]string{}, false
}

// GetExecution handles GET /api/v1/executions/{id}
func (h *Handler) GetExecution(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseUUIDParam(w, r, "id", ErrInvalidExecutionID)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), constants.DefaultRequestTimeout)
	defer cancel()

	execution, err := h.executionService.GetExecution(ctx, id)
	if err != nil {
		h.log.WithError(err).WithField("id", id).Error("Failed to get execution")
		h.respondError(w, ErrExecutionNotFound.WithDetails(map[string]string{"execution_id": id.String()}))
		return
	}

	pbExecution, err := h.executionService.HydrateExecutionProto(ctx, execution)
	if err != nil {
		if h.log != nil {
			h.log.WithError(err).WithField("execution_id", id).Error("Failed to hydrate execution proto")
		}
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "hydrate_execution"}))
		return
	}

	h.respondProto(w, http.StatusOK, pbExecution)
}

// ListExecutions handles GET /api/v1/executions
// Query parameters:
//   - workflow_id: Filter by workflow ID
//   - project_id: Filter by project ID
//   - limit, offset: Pagination
//   - include_exportability: If "true", include exportability info for each execution
func (h *Handler) ListExecutions(w http.ResponseWriter, r *http.Request) {
	workflowIDStr := r.URL.Query().Get("workflow_id")
	var workflowID *uuid.UUID
	if workflowIDStr != "" {
		if id, err := uuid.Parse(workflowIDStr); err == nil {
			workflowID = &id
		}
	}

	projectIDStr := r.URL.Query().Get("project_id")
	var projectID *uuid.UUID
	if projectIDStr != "" {
		if id, err := uuid.Parse(projectIDStr); err == nil {
			projectID = &id
		}
	}

	includeExportability := strings.ToLower(r.URL.Query().Get("include_exportability")) == "true"

	ctx, cancel := context.WithTimeout(r.Context(), constants.DefaultRequestTimeout)
	defer cancel()

	limit, offset := parsePaginationParams(r, 0, 0)

	executions, err := h.executionService.ListExecutions(ctx, workflowID, projectID, limit, offset)
	if err != nil {
		h.log.WithError(err).Error("Failed to list executions")
		h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "list_executions"}))
		return
	}

	protoExecutions := make([]proto.Message, 0, len(executions))
	var exportabilityMap map[string]ExecutionExportability
	if includeExportability {
		exportabilityMap = make(map[string]ExecutionExportability, len(executions))
	}

	for idx := range executions {
		if executions[idx] == nil {
			continue
		}
		pbExec, convErr := h.executionService.HydrateExecutionProto(ctx, executions[idx])
		if convErr != nil {
			if h.log != nil {
				h.log.WithError(convErr).WithField("execution_id", executions[idx].ID).Error("Failed to hydrate execution proto")
			}
			h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
				"operation": fmt.Sprintf("hydrate_execution[%d]", idx),
				"error":     convErr.Error(),
			}))
			return
		}
		protoExecutions = append(protoExecutions, pbExec)

		// Compute exportability if requested
		if includeExportability {
			execID := executions[idx].ID.String()
			exportability := checkExecutionExportability(
				executions[idx].ResultPath,
				h.recordingsRoot,
				execID,
			)
			exportabilityMap[execID] = exportability
		}
	}

	// If exportability was requested, use custom response format
	if includeExportability {
		h.respondExecutionsWithExportability(w, http.StatusOK, protoExecutions, exportabilityMap)
		return
	}

	h.respondProtoList(w, http.StatusOK, "executions", protoExecutions)
}

// StopExecution handles POST /api/v1/executions/{id}/stop
func (h *Handler) StopExecution(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseUUIDParam(w, r, "id", ErrInvalidExecutionID)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), constants.DefaultRequestTimeout)
	defer cancel()

	if err := h.executionService.StopExecution(ctx, id); err != nil {
		h.log.WithError(err).WithField("id", id).Error("Failed to stop execution")
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "stop_execution", "execution_id": id.String()}))
		return
	}

	h.respondSuccess(w, http.StatusOK, map[string]any{
		"status": "stopped",
	})
}

// resumeExecutionRequest represents the request body for resuming an execution.
type resumeExecutionRequest struct {
	// Parameters are optional overrides merged with original execution params.
	Parameters map[string]any `json:"parameters,omitempty"`
	// ResumeURL is an optional URL to navigate the browser to before resuming.
	// Useful when the browser state needs to be re-established after session loss.
	ResumeURL string `json:"resume_url,omitempty"`
}

// ResumeExecution handles POST /api/v1/executions/{id}/resume
// Resumes an interrupted execution from its last checkpoint.
func (h *Handler) ResumeExecution(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseUUIDParam(w, r, "id", ErrInvalidExecutionID)
	if !ok {
		return
	}

	var body resumeExecutionRequest
	if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		if err := decodeJSONBodyAllowEmpty(w, r, &body); err != nil {
			h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": "invalid json payload"}))
			return
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), constants.ExtendedRequestTimeout)
	defer cancel()

	// Merge resume_url into parameters for the service layer
	params := body.Parameters
	if body.ResumeURL != "" {
		if params == nil {
			params = make(map[string]any)
		}
		params["resume_url"] = body.ResumeURL
	}

	newExecution, err := h.executionService.ResumeExecution(ctx, id, params)
	if err != nil {
		h.log.WithError(err).WithField("id", id).Error("Failed to resume execution")
		// Check if it's a "cannot be resumed" error vs a server error
		errMsg := err.Error()
		if strings.Contains(errMsg, "cannot be resumed") || strings.Contains(errMsg, "not resumable") {
			h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{
				"operation":    "resume_execution",
				"execution_id": id.String(),
				"error":        errMsg,
			}))
			return
		}
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
			"operation":    "resume_execution",
			"execution_id": id.String(),
		}))
		return
	}

	pbExecution, err := protoconv.ExecutionToProto(newExecution)
	if err != nil {
		if h.log != nil {
			h.log.WithError(err).WithField("execution_id", newExecution.ID).Error("Failed to convert resumed execution to proto")
		}
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
			"operation": "execution_to_proto",
			"error":     err.Error(),
		}))
		return
	}

	h.respondProto(w, http.StatusOK, pbExecution)
}
