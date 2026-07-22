// Queue operations for backlog items: preflight checks, queueing for
// execution, and helper functions for evaluating blocking reasons.
package backlog

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/apierr"
	"swarm-manager/internal/execution"
	"swarm-manager/internal/httputil"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
)

// isQueueableItem checks if an item can be queued from its current state.
// Archived ideas can be queued (re-executed); other archived items cannot.
func isQueueableItem(item BacklogItem) bool {
	switch item.Status {
	case StatusBacklog, StatusResearching, StatusReady:
		return true
	default:
		// Archived ideas can be queued for re-execution.
		return item.ArchivedAt != nil && item.Kind == KindIdea
	}
}

// Queue queues a backlog item for processing via agent-manager.
func (h *Handler) Queue(w http.ResponseWriter, r *http.Request) {
	kind, name, ok := h.parseKindAndName(w, r, "queue")
	if !ok {
		return
	}

	item, err := h.store.LoadItem(kind, name)
	if err != nil {
		if os.IsNotExist(err) {
			apierr.MapError(w, "[backlog] queue", apierr.NotFound("backlog item not found"))
			return
		}
		slog.Error("failed to load item for queue", "name", name, "err", err)
		apierr.MapError(w, "[backlog] queue", apierr.Internal("%s", httputil.TruncateErrorMessage(err, 240)))
		return
	}

	if !isQueueableItem(item) {
		apierr.MapError(w, "[backlog] queue", apierr.BadRequest("backlog item cannot be queued from current status: %s", item.Status))
		return
	}

	params, ok := parseQueueRequest(w, r)
	if !ok {
		return
	}
	operation, confirm, force, mode, startedBy := params.operation, params.confirm, params.force, params.mode, params.startedBy

	if h.executionQueuer == nil {
		apierr.MapError(w, "[backlog] queue", apierr.Unavailable("execution service is not available"))
		return
	}
	executionService := h.executionQueuer

	preflight, preflightErr := executionService.ProcessPreflight(r.Context(), string(kind), name)
	if preflightErr != nil {
		if os.IsNotExist(preflightErr) {
			apierr.MapError(w, "[backlog] queue", apierr.NotFound("backlog item not found"))
			return
		}
		slog.Error("process preflight failed", "kind", kind, "name", name, "err", preflightErr)
		apierr.MapError(w, "[backlog] queue", apierr.Internal("failed to evaluate process preflight"))
		return
	}
	// When !preflight.Ready we keep evaluating feedback gates and force
	// overrides below so callers receive one canonical queue response
	// shape with clear next actions.

	blockingReasons, blockErr := h.collectQueueBlockingReasons(item, kind, name, preflight)
	if blockErr != nil {
		apierr.MapError(w, "[backlog] queue", apierr.Internal("failed to check dependencies"))
		return
	}

	// Convert to proto representation.
	protoReasons := make([]*apipb.BlockingReason, len(blockingReasons))
	for i, r := range blockingReasons {
		protoReasons[i] = &apipb.BlockingReason{Message: r.Message, Forceable: r.Forceable}
	}

	if done := h.handleQueuePreflightResponse(w, item, protoReasons, preflight.Advisories, confirm, force, blockingReasons, r); done {
		return
	}
	record, err := executionService.QueueBacklog(r.Context(), execution.CreateRequest{
		BacklogKind: string(kind),
		BacklogName: name,
		Mode:        mode,
		StartedBy:   startedBy,
		Operation:   operation,
		Force:       force,
	})
	if err != nil {
		mapQueueBacklogError(w, err)
		return
	}

	item, err = h.store.LoadItem(kind, name)
	if err != nil {
		slog.Error("failed to reload item after queue", "name", name, "err", err)
		apierr.MapError(w, "[backlog] queue", apierr.Internal("failed to load updated backlog item"))
		return
	}

	slog.Info("item queued", "name", name, "kind", kind, "status", item.Status, "task_id", record.TaskID, "execution_id", record.ExecutionID)

	resp := &apipb.QueueBacklogItemResponse{
		Item:                backlogToProto(item),
		TaskId:              record.TaskID,
		RunId:               record.RunID,
		BaseUrl:             "",
		Created:             record.CreatedAt,
		DryRun:              false,
		Queued:              true,
		Message:             "Queue created successfully.",
		BlockingReasons:     nil,
		UnansweredQuestions: 0,
		PendingSuggestions:  0,
		Advisories:          preflight.Advisories,
	}
	if err := httputil.ProtoJSONWithStatus(w, http.StatusAccepted, resp); err != nil {
		apierr.MapError(w, "[backlog] queue", apierr.Internal("failed to encode response"))
	}
}

// queueRequestParams holds the normalized inputs parsed from a queue request.
type queueRequestParams struct {
	operation string
	confirm   bool
	force     bool
	mode      execution.Mode
	startedBy string
}

// parseQueueRequest decodes and normalizes the queue request body, applying
// defaults. It returns false (after writing an error response) when the body is
// malformed or the execution mode is invalid.
func parseQueueRequest(w http.ResponseWriter, r *http.Request) (queueRequestParams, bool) {
	var pbReq apipb.QueueBacklogItemRequest
	if r.Body != nil {
		if err := httputil.DecodeProtoJSON(r, &pbReq); err != nil {
			// Tolerate empty bodies (all fields optional).
			if !errors.Is(err, io.EOF) && r.ContentLength != 0 {
				apierr.MapError(w, "[backlog] queue", apierr.BadRequest("invalid request body"))
				return queueRequestParams{}, false
			}
		}
		if !httputil.ValidateProtoRequest(w, "[backlog] queue", "invalid queue request", &pbReq) {
			return queueRequestParams{}, false
		}
	}

	params := queueRequestParams{
		operation: "generator",
		confirm:   pbReq.GetConfirm(),
		force:     pbReq.GetForce(),
		mode:      execution.ModeYOLO,
		startedBy: strings.TrimSpace(pbReq.GetStartedBy()),
	}
	if pbReq.GetOperation() != "" {
		params.operation = strings.ToLower(strings.TrimSpace(pbReq.GetOperation()))
	}
	if pbReq.GetMode() != "" {
		params.mode = execution.Mode(strings.ToLower(strings.TrimSpace(pbReq.GetMode())))
		if !execution.ValidateMode(params.mode) {
			apierr.MapError(w, "[backlog] queue", apierr.BadRequest("invalid execution mode %q: must be manual or yolo", params.mode))
			return queueRequestParams{}, false
		}
	}
	if params.startedBy == "" {
		params.startedBy = "swarm-manager"
	}
	return params, true
}

// handleQueuePreflightResponse writes a dry-run, preview, or blocked response
// when the queue operation should not proceed, returning true (done). Returns
// false when the caller should continue to actually queue the item. Extracting
// this eliminates the buildQueueResponse closure and four conditional branches
// from Queue, making the handler's happy-path flow linear.
func (h *Handler) handleQueuePreflightResponse(
	w http.ResponseWriter,
	item BacklogItem,
	protoReasons []*apipb.BlockingReason,
	advisories []string,
	confirm, force bool,
	blockingReasons []BlockingReason,
	r *http.Request,
) bool {
	buildResp := func(dryRun, queued bool, message, taskID, runID, created string) *apipb.QueueBacklogItemResponse {
		return &apipb.QueueBacklogItemResponse{
			Item:                backlogToProto(item),
			TaskId:              taskID,
			RunId:               runID,
			BaseUrl:             "",
			Created:             created,
			DryRun:              dryRun,
			Queued:              queued,
			Message:             message,
			BlockingReasons:     protoReasons,
			UnansweredQuestions: 0,
			PendingSuggestions:  0,
			Advisories:          advisories,
		}
	}

	if !confirm || httputil.IsDryRun(r) {
		message := "Queue request validated. No changes applied."
		if !confirm {
			message = "Preview only. Re-run with confirm=true (CLI: --execute) to queue."
		}
		if len(blockingReasons) > 0 {
			message = "Queue blocked by preflight checks. Resolve blockers or use force=true (CLI: --force) for eligible overrides."
		}
		resp := buildResp(true, false, message, "dry-run-task", "", time.Now().UTC().Format(time.RFC3339))
		if err := httputil.ProtoJSONWithStatus(w, http.StatusOK, resp); err != nil {
			apierr.MapError(w, "[backlog] queue", apierr.Internal("failed to encode dry-run response"))
		}
		return true
	}

	if len(blockingReasons) > 0 && (!force || HasNonForceableReasons(blockingReasons)) {
		resp := buildResp(true, false, "Queue blocked by preflight checks.", "", "", time.Now().UTC().Format(time.RFC3339))
		if err := httputil.ProtoJSONWithStatus(w, http.StatusOK, resp); err != nil {
			apierr.MapError(w, "[backlog] queue", apierr.Internal("failed to encode blocked response"))
		}
		return true
	}
	return false
}

// mapQueueBacklogError classifies an error from executionService.QueueBacklog
// into the appropriate API error response.
func mapQueueBacklogError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, agentmanager.ErrNotAvailable):
		apierr.MapError(w, "[backlog] queue", apierr.Unavailable("agent-manager is not available"))
	case os.IsNotExist(err):
		apierr.MapError(w, "[backlog] queue", apierr.NotFound("backlog item not found"))
	case strings.Contains(err.Error(), "cannot be queued") || strings.Contains(err.Error(), "process preflight failed"):
		apierr.MapError(w, "[backlog] queue", apierr.BadRequest("%s", err.Error()))
	default:
		apierr.MapError(w, "[backlog] queue", apierr.Internal("%s", "failed to queue execution: "+httputil.TruncateErrorMessage(err, 240)))
	}
}

// collectQueueBlockingReasons gathers all blocking reasons for a queue request:
// preflight structural/forceable reasons and dependency
// readiness. The returned slice is deduplicated. A non-nil
// error indicates the dependency check itself failed and the queue must abort.
func (h *Handler) collectQueueBlockingReasons(item BacklogItem, kind BacklogKind, name string, preflight execution.ProcessPreflight) ([]BlockingReason, error) {
	// Convert preflight reasons to structured BlockingReasons.
	// Preflight reasons from the execution service are non-forceable structural blockers.
	var blockingReasons []BlockingReason
	for _, reason := range preflight.BlockingReasons {
		blockingReasons = append(blockingReasons, BlockingReason{
			Message:   reason,
			Forceable: false,
		})
	}
	// Forceable preflight reasons (e.g. the fix-before-feature gate in "block"
	// mode) block the queue but can be overridden with force=true.
	for _, reason := range preflight.ForceableBlockingReasons {
		blockingReasons = append(blockingReasons, BlockingReason{
			Message:   reason,
			Forceable: true,
		})
	}
	// Check dependency readiness: all depends_on items must be completed.
	depReasons, depErr := EvaluateDependencyBlocking(item, h.store)
	if depErr != nil {
		slog.Error("dependency check failed", "kind", kind, "name", name, "err", depErr)
		return nil, depErr
	}
	blockingReasons = append(blockingReasons, depReasons...)

	return DedupeReasons(blockingReasons), nil
}

// ProcessPreflight evaluates whether a backlog item is ready for processing.
func (h *Handler) ProcessPreflight(w http.ResponseWriter, r *http.Request) {
	kind, name, ok := h.parseKindAndName(w, r, "process-preflight")
	if !ok {
		return
	}

	item, err := h.store.LoadItem(kind, name)
	if err != nil {
		if os.IsNotExist(err) {
			apierr.MapError(w, "[backlog] process-preflight", apierr.NotFound("backlog item not found"))
			return
		}
		apierr.MapError(w, "[backlog] process-preflight", apierr.Internal("failed to load backlog item"))
		return
	}

	if h.executionQueuer == nil {
		apierr.MapError(w, "[backlog] process-preflight", apierr.Unavailable("execution service is not available"))
		return
	}
	preflight, err := h.executionQueuer.ProcessPreflight(r.Context(), string(kind), name)
	if err != nil {
		if os.IsNotExist(err) {
			apierr.MapError(w, "[backlog] process-preflight", apierr.NotFound("backlog item not found"))
			return
		}
		apierr.MapError(w, "[backlog] process-preflight", apierr.Internal("failed to evaluate preflight"))
		return
	}

	if err := httputil.JSON(w, map[string]any{
		"item":      item,
		"preflight": preflight,
	}); err != nil {
		apierr.MapError(w, "[backlog] process-preflight", apierr.Internal("failed to encode response"))
	}
}
