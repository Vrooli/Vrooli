package handlers

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"encoding/json"
	"time"

	"github.com/vrooli/browser-automation-studio/constants"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/services/workflow"
	browser_automation_studio_v1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

const (
	workflowPayloadPreviewBytes = 512
)

// toWorkflowVersionResponse converts a WorkflowVersionSummary from the service layer
// into an HTTP response-ready structure with properly formatted timestamps.
func toWorkflowVersionResponse(summary *workflow.WorkflowVersionSummary) workflowVersionResponse {
	if summary == nil {
		return workflowVersionResponse{}
	}
	return workflowVersionResponse{
		Version:           summary.Version,
		WorkflowID:        summary.WorkflowID,
		CreatedAt:         summary.CreatedAt.UTC().Format(time.RFC3339Nano),
		CreatedBy:         summary.CreatedBy,
		ChangeDescription: summary.ChangeDescription,
		DefinitionHash:    summary.DefinitionHash,
		NodeCount:         summary.NodeCount,
		EdgeCount:         summary.EdgeCount,
		FlowDefinition:    jsonMapToStd(summary.FlowDefinition),
	}
}

// jsonMapToStd converts a database.JSONMap to a standard map[string]any.
// This is necessary because database.JSONMap has custom JSON marshaling behavior
// that we need to normalize for API responses.
func jsonMapToStd(m database.JSONMap) map[string]any {
	if m == nil {
		return nil
	}
	result := make(map[string]any, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}

// validateWorkflowProtoShape ensures incoming JSON maps conform to the WorkflowDefinition proto shape.
// This catches unknown/incorrectly cased fields early instead of silently discarding them.
func validateWorkflowProtoShape(definition map[string]any) error {
	if definition == nil || len(definition) == 0 {
		return nil
	}
	// Nodes/edges often carry UI-only fields (positions, handles). Ignore them for
	// proto shape validation so we catch casing/field drift on the typed sections
	// without rejecting legitimately richer UI payloads.
	clean := make(map[string]any, len(definition))
	for k, v := range definition {
		if k == "nodes" || k == "edges" {
			continue
		}
		clean[k] = v
	}

	raw, err := json.Marshal(clean)
	if err != nil {
		return err
	}

	// Disallow unknown fields so UI/CLI drifts surface immediately.
	return (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(raw, &browser_automation_studio_v1.WorkflowDefinition{})
}

// definitionFromNodesEdges constructs a workflow definition map from separate
// nodes and edges arrays. If a fallback definition is provided and non-empty,
// it takes precedence. Returns nil if no definition can be constructed.
func definitionFromNodesEdges(nodes, edges []any, fallback map[string]any) map[string]any {
	if fallback != nil && len(fallback) > 0 {
		return fallback
	}
	if len(nodes) == 0 && len(edges) == 0 {
		return nil
	}
	return map[string]any{
		"nodes": nodes,
		"edges": edges,
	}
}

// readLimitedBody reads an HTTP request body with a maximum size limit.
// This prevents memory exhaustion from excessively large payloads.
// The http.MaxBytesReader will return an error if the limit is exceeded.
func readLimitedBody(w http.ResponseWriter, r *http.Request, maxBytes int64) ([]byte, error) {
	reader := http.MaxBytesReader(w, r.Body, maxBytes)
	defer reader.Close()
	return io.ReadAll(reader)
}

// buildPayloadPreview creates a truncated preview of a request payload for logging.
// This is useful for debugging without logging potentially sensitive full payloads.
// Returns empty string if the body is empty or whitespace-only.
func buildPayloadPreview(body []byte) string {
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 {
		return ""
	}
	if len(trimmed) > workflowPayloadPreviewBytes {
		trimmed = trimmed[:workflowPayloadPreviewBytes]
		return string(trimmed) + "…"
	}
	return string(trimmed)
}

// waitForExecutionCompletion polls the workflow service until an execution
// reaches a terminal status (completed, failed, or cancelled). This is used
// to support synchronous workflow execution requests.
//
// The function will return immediately if the execution is already in a
// terminal state. Otherwise, it polls at regular intervals until either:
// - The execution reaches a terminal status
// - The context is cancelled or times out
//
// Returns the final execution state or an error if polling fails.
func (h *Handler) waitForExecutionCompletion(ctx context.Context, execution *database.Execution) (*database.Execution, error) {
	if execution == nil {
		return nil, errors.New("execution cannot be nil")
	}
	if workflow.IsTerminalExecutionStatus(execution.Status) {
		return execution, nil
	}

	executionID := execution.ID
	ticker := time.NewTicker(executionCompletionPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			pollCtx, cancel := context.WithTimeout(ctx, constants.DefaultRequestTimeout)
			latest, err := h.workflowService.GetExecution(pollCtx, executionID)
			cancel()
			if err != nil {
				return nil, err
			}
			if workflow.IsTerminalExecutionStatus(latest.Status) {
				return latest, nil
			}
		}
	}
}

// logInvalidWorkflowPayload logs a detailed error message when workflow payload
// decoding fails, including a preview of the payload for debugging purposes.
func (h *Handler) logInvalidWorkflowPayload(err error, body []byte) {
	preview := buildPayloadPreview(body)
	entry := h.log.WithError(err).
		WithField("payload_bytes", len(body))
	if preview != "" {
		entry = entry.WithField("payload_preview", preview)
	}
	entry.Warn("Failed to decode adhoc workflow request payload")
}
