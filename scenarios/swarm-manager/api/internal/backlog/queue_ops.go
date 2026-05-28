// Queue operations for backlog items: preflight checks, queueing for
// execution, and helper functions for evaluating blocking reasons.
package backlog

import (
	"errors"
	"fmt"
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

	var pbReq apipb.QueueBacklogItemRequest
	if r.Body != nil {
		if err := httputil.DecodeProtoJSON(r, &pbReq); err != nil {
			// Tolerate empty bodies (all fields optional).
			if !errors.Is(err, io.EOF) && r.ContentLength != 0 {
				apierr.MapError(w, "[backlog] queue", apierr.BadRequest("invalid request body"))
				return
			}
		}
		if !httputil.ValidateProtoRequest(w, "[backlog] queue", "invalid queue request", &pbReq) {
			return
		}
	}
	operation := "generator"
	if pbReq.GetOperation() != "" {
		operation = strings.ToLower(strings.TrimSpace(pbReq.GetOperation()))
	}
	confirm := pbReq.GetConfirm()
	force := pbReq.GetForce()
	mode := execution.ModeYOLO
	if pbReq.GetMode() != "" {
		mode = execution.Mode(strings.ToLower(strings.TrimSpace(pbReq.GetMode())))
		if !execution.ValidateMode(mode) {
			apierr.MapError(w, "[backlog] queue", apierr.BadRequest("invalid execution mode %q: must be manual or yolo", mode))
			return
		}
	}
	startedBy := strings.TrimSpace(pbReq.GetStartedBy())
	if startedBy == "" {
		startedBy = "swarm-manager"
	}

	var executionService ExecutionQueuer
	if h.executionQueuer != nil {
		executionService = h.executionQueuer
	} else {
		executionService = execution.NewService(execution.ServiceConfig{
			DataRoot:           h.dataRoot,
			RepoRoot:           h.repoRoot,
			PolicyProvider:     h.policyProvider,
			GovernanceProvider: h.governanceProvider,
			AgentService:       h.agentService,
		})
	}

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

	// Check workshop feedback state for additional blocking signals.
	itemDir := h.store.ItemDir(kind, item.Name)
	latestRound, _, _ := LoadLatestRound(itemDir)
	pendingDecisions := CountPendingDecisions(latestRound)

	// Convert preflight reasons to structured BlockingReasons.
	// Preflight reasons from the execution service (readiness dimensions,
	// missing deliverables) are non-forceable structural blockers.
	var blockingReasons []BlockingReason
	for _, reason := range preflight.BlockingReasons {
		blockingReasons = append(blockingReasons, BlockingReason{
			Message:   reason,
			Forceable: false,
		})
	}
	if pendingDecisions > 0 {
		blockingReasons = append(blockingReasons, BlockingReason{
			Message:   fmt.Sprintf("%d workshop decision(s) still pending", pendingDecisions),
			Forceable: true,
		})
	}

	// Check dependency readiness: all depends_on items must be completed.
	depReasons, depErr := EvaluateDependencyBlocking(item, h.store)
	if depErr != nil {
		slog.Error("dependency check failed", "kind", kind, "name", name, "err", depErr)
		apierr.MapError(w, "[backlog] queue", apierr.Internal("failed to check dependencies"))
		return
	}
	blockingReasons = append(blockingReasons, depReasons...)

	// Check plan validation: failed validation produces a forceable blocker.
	valReport, valErr := LoadValidationReport(itemDir)
	if valErr != nil {
		slog.Warn("failed to load validation report for queue check", "kind", kind, "name", name, "err", valErr)
	}
	if valReport != nil && !valReport.Passed {
		missingStr := strings.Join(valReport.SectionsMissing, ", ")
		msg := "plan validation failed"
		if missingStr != "" {
			msg += ": missing sections: " + missingStr
		}
		if len(valReport.Warnings) > 0 {
			msg += "; warnings: " + strings.Join(valReport.Warnings, "; ")
		}
		blockingReasons = append(blockingReasons, BlockingReason{
			Message:   msg,
			Forceable: true,
		})
	}
	blockingReasons = DedupeReasons(blockingReasons)

	// Convert to proto representation.
	protoReasons := make([]*apipb.BlockingReason, len(blockingReasons))
	for i, r := range blockingReasons {
		protoReasons[i] = &apipb.BlockingReason{Message: r.Message, Forceable: r.Forceable}
	}

	buildQueueResponse := func(dryRun, queued bool, message string, taskID, runID, created string) *apipb.QueueBacklogItemResponse {
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
			PendingSuggestions:  int32(pendingDecisions),
		}
	}

	if !confirm || httputil.IsDryRun(r) {
		message := "Queue request validated. No changes applied."
		if !confirm {
			message = "Preview only. Re-run with confirm=true (CLI: --execute) to queue."
		}
		if len(blockingReasons) > 0 {
			message = "Queue blocked by readiness checks. Resolve blockers or use force=true (CLI: --force) for feedback-gate overrides."
		}
		resp := buildQueueResponse(true, false, message, "dry-run-task", "", time.Now().UTC().Format(time.RFC3339))
		if err := httputil.ProtoJSONWithStatus(w, http.StatusOK, resp); err != nil {
			apierr.MapError(w, "[backlog] queue", apierr.Internal("failed to encode dry-run response"))
		}
		return
	}

	if len(blockingReasons) > 0 {
		if !force || HasNonForceableReasons(blockingReasons) {
			resp := buildQueueResponse(true, false, "Queue blocked by readiness checks.", "", "", time.Now().UTC().Format(time.RFC3339))
			if err := httputil.ProtoJSONWithStatus(w, http.StatusOK, resp); err != nil {
				apierr.MapError(w, "[backlog] queue", apierr.Internal("failed to encode blocked response"))
			}
			return
		}
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
		if errors.Is(err, agentmanager.ErrNotAvailable) {
			apierr.MapError(w, "[backlog] queue", apierr.Unavailable("agent-manager is not available"))
			return
		}
		if os.IsNotExist(err) {
			apierr.MapError(w, "[backlog] queue", apierr.NotFound("backlog item not found"))
			return
		}
		if strings.Contains(err.Error(), "cannot be queued") || strings.Contains(err.Error(), "process preflight failed") {
			apierr.MapError(w, "[backlog] queue", apierr.BadRequest("%s", err.Error()))
			return
		}
		apierr.MapError(w, "[backlog] queue", apierr.Internal("%s", "failed to queue execution: "+httputil.TruncateErrorMessage(err, 240)))
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
		PendingSuggestions:  int32(pendingDecisions),
	}
	if err := httputil.ProtoJSONWithStatus(w, http.StatusAccepted, resp); err != nil {
		apierr.MapError(w, "[backlog] queue", apierr.Internal("failed to encode response"))
	}
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

	executionService := execution.NewService(execution.ServiceConfig{
		DataRoot:           h.dataRoot,
		RepoRoot:           h.repoRoot,
		PolicyProvider:     h.policyProvider,
		GovernanceProvider: h.governanceProvider,
		AgentService:       h.agentService,
	})
	preflight, err := executionService.ProcessPreflight(r.Context(), string(kind), name)
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
