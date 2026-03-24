// Queue operations for backlog items: preflight checks, queueing for
// execution, and helper functions for evaluating blocking reasons.
package backlog

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/execution"
	"swarm-manager/internal/httputil"
)

// isQueueableStatus checks if an item can be queued from its current status.
func isQueueableStatus(kind BacklogKind, status BacklogStatus) bool {
	switch status {
	case StatusBacklog, StatusResearching, StatusReady:
		return true
	case StatusArchived:
		return kind == KindIdea
	default:
		return false
	}
}

// dedupeQueueReasons removes duplicate and empty blocking reasons.
func dedupeQueueReasons(reasons []string) []string {
	if len(reasons) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(reasons))
	result := make([]string, 0, len(reasons))
	for _, reason := range reasons {
		trimmed := strings.TrimSpace(reason)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}

// hasNonForceableQueueReasons returns true if any reason cannot be overridden
// with the force flag.
func hasNonForceableQueueReasons(reasons []string) bool {
	for _, reason := range reasons {
		if !isForceableQueueReason(reason) {
			return true
		}
	}
	return false
}

// isForceableQueueReason returns true if a blocking reason can be bypassed
// with force=true (currently only workshop/pending decision reasons).
func isForceableQueueReason(reason string) bool {
	normalized := strings.ToLower(strings.TrimSpace(reason))
	return strings.Contains(normalized, "workshop decision") ||
		strings.Contains(normalized, "pending decision")
}

// appendDependencyBlockingReasons checks an item's dependencies and appends
// a blocking reason if any are unmet. Used by the single-item Queue handler.
func appendDependencyBlockingReasons(item BacklogItem, store Store, reasons []string) ([]string, error) {
	if len(item.DependsOn) == 0 {
		return reasons, nil
	}
	unmet, err := store.CheckDependencies(item.DependsOn)
	if err != nil {
		return reasons, err
	}
	if len(unmet) > 0 {
		reasons = append(reasons, fmt.Sprintf("unmet dependencies: %s", strings.Join(unmet, ", ")))
	}
	return reasons, nil
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
			httputil.NotFound(w, "[backlog] queue", "backlog item not found")
			return
		}
		log.Printf("[backlog] queue: failed to load %q: %v", name, err)
		httputil.InternalError(w, "[backlog] queue", httputil.TruncateErrorMessage(err, 240))
		return
	}

	if !isQueueableStatus(item.Kind, item.Status) {
		httputil.BadRequest(w, "[backlog] queue", "backlog item cannot be queued from current status: "+string(item.Status))
		return
	}

	var pbReq apipb.QueueBacklogItemRequest
	if r.Body != nil {
		if err := httputil.DecodeProtoJSON(r, &pbReq); err != nil {
			// Tolerate empty bodies (all fields optional).
			if !errors.Is(err, io.EOF) && r.ContentLength != 0 {
				httputil.BadRequest(w, "[backlog] queue", "invalid request body")
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
			httputil.BadRequest(w, "[backlog] queue", fmt.Sprintf("invalid execution mode %q: must be manual, scheduled, or yolo", mode))
			return
		}
	}
	startedBy := strings.TrimSpace(pbReq.GetStartedBy())
	if startedBy == "" {
		startedBy = "swarm-manager"
	}

	if kind == KindResearch {
		httputil.BadRequest(w, "[backlog] queue", "research items must be converted before processing")
		return
	}

	executionService := execution.NewService(execution.ServiceConfig{
		RootDir:      h.rootDir,
		StorePath:    filepath.Join(h.rootDir, ".vrooli", "execution-runs.json"),
		PolicyPath:   filepath.Join(h.rootDir, ".vrooli", "execution-policy.json"),
		AgentService: h.agentService,
	})
	preflight, preflightErr := executionService.ProcessPreflight(r.Context(), string(kind), name)
	if preflightErr != nil {
		if os.IsNotExist(preflightErr) {
			httputil.NotFound(w, "[backlog] queue", "backlog item not found")
			return
		}
		log.Printf("[backlog] queue: process preflight failed for %s/%s: %v", kind, name, preflightErr)
		httputil.InternalError(w, "[backlog] queue", "failed to evaluate process preflight")
		return
	}
	// When !preflight.Ready we keep evaluating feedback gates and force
	// overrides below so callers receive one canonical queue response
	// shape with clear next actions.

	// Check workshop feedback state for additional blocking signals.
	itemDir := h.store.ItemDir(kind, item.Name)
	latestRound, _, _ := LoadLatestRound(itemDir)
	pendingDecisions := CountPendingDecisions(latestRound)
	blockingReasons := append([]string{}, preflight.BlockingReasons...)
	if pendingDecisions > 0 {
		blockingReasons = append(blockingReasons, fmt.Sprintf("%d workshop decision(s) still pending", pendingDecisions))
	}

	// Check dependency readiness: all depends_on items must be completed.
	var depErr error
	blockingReasons, depErr = appendDependencyBlockingReasons(item, h.store, blockingReasons)
	if depErr != nil {
		log.Printf("[backlog] queue: dependency check failed for %s/%s: %v", kind, name, depErr)
		httputil.InternalError(w, "[backlog] queue", "failed to check dependencies")
		return
	}

	blockingReasons = dedupeQueueReasons(blockingReasons)

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
			BlockingReasons:     blockingReasons,
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
			httputil.InternalError(w, "[backlog] queue", "failed to encode dry-run response")
		}
		return
	}

	if len(blockingReasons) > 0 {
		if !force || hasNonForceableQueueReasons(blockingReasons) {
			resp := buildQueueResponse(true, false, "Queue blocked by readiness checks.", "", "", time.Now().UTC().Format(time.RFC3339))
			if err := httputil.ProtoJSONWithStatus(w, http.StatusOK, resp); err != nil {
				httputil.InternalError(w, "[backlog] queue", "failed to encode blocked response")
			}
			return
		}
	}
	record, err := executionService.QueueBacklog(r.Context(), execution.CreateRequest{
		BacklogKind:  string(kind),
		BacklogName:  name,
		Mode:         mode,
		DelaySeconds: pbReq.GetDelaySeconds(),
		StartedBy:    startedBy,
		Operation:    operation,
		Force:        force,
	})
	if err != nil {
		if errors.Is(err, agentmanager.ErrNotAvailable) {
			httputil.ServiceUnavailable(w, "[backlog] queue", "agent-manager is not available")
			return
		}
		if os.IsNotExist(err) {
			httputil.NotFound(w, "[backlog] queue", "backlog item not found")
			return
		}
		if strings.Contains(err.Error(), "cannot be queued") || strings.Contains(err.Error(), "process preflight failed") {
			httputil.BadRequest(w, "[backlog] queue", err.Error())
			return
		}
		httputil.InternalError(w, "[backlog] queue", "failed to queue execution: "+httputil.TruncateErrorMessage(err, 240))
		return
	}

	item, err = h.store.LoadItem(kind, name)
	if err != nil {
		log.Printf("[backlog] queue: failed to reload %q after queue: %v", name, err)
		httputil.InternalError(w, "[backlog] queue", "failed to load updated backlog item")
		return
	}

	log.Printf("[backlog] queued: %q (kind=%s, status=%s, taskId=%s, executionId=%s)", name, kind, item.Status, record.TaskID, record.ExecutionID)

	resp := &apipb.QueueBacklogItemResponse{
		Item:                backlogToProto(item),
		TaskId:              record.TaskID,
		RunId:               record.RunID,
		BaseUrl:             "",
		Created:             record.CreatedAt,
		DryRun:              false,
		Queued:              true,
		Message:             "Queue created successfully.",
		BlockingReasons:     []string{},
		UnansweredQuestions: 0,
		PendingSuggestions:  int32(pendingDecisions),
	}
	if err := httputil.ProtoJSONWithStatus(w, http.StatusAccepted, resp); err != nil {
		httputil.InternalError(w, "[backlog] queue", "failed to encode response")
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
			httputil.NotFound(w, "[backlog] process-preflight", "backlog item not found")
			return
		}
		httputil.InternalError(w, "[backlog] process-preflight", "failed to load backlog item")
		return
	}

	executionService := execution.NewService(execution.ServiceConfig{
		RootDir:      h.rootDir,
		StorePath:    filepath.Join(h.rootDir, ".vrooli", "execution-runs.json"),
		PolicyPath:   filepath.Join(h.rootDir, ".vrooli", "execution-policy.json"),
		AgentService: h.agentService,
	})
	preflight, err := executionService.ProcessPreflight(r.Context(), string(kind), name)
	if err != nil {
		if os.IsNotExist(err) {
			httputil.NotFound(w, "[backlog] process-preflight", "backlog item not found")
			return
		}
		httputil.InternalError(w, "[backlog] process-preflight", "failed to evaluate preflight")
		return
	}

	if err := httputil.JSON(w, map[string]any{
		"item":      item,
		"preflight": preflight,
	}); err != nil {
		httputil.InternalError(w, "[backlog] process-preflight", "failed to encode response")
	}
}
