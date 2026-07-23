package backlog

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"

	"swarm-manager/internal/apierr"
	"swarm-manager/internal/httputil"
)

// NextActionID is the stable, server-owned vocabulary for the primary action
// an operator can take on a backlog item. Presentation copy belongs to clients;
// policy does not.
type NextActionID string

const (
	NextActionNone                NextActionID = "none"
	NextActionAcceptSuggestion    NextActionID = "accept_suggestion"
	NextActionAuthorPlan          NextActionID = "author_plan"
	NextActionAcceptPlan          NextActionID = "accept_plan"
	NextActionRepairPlan          NextActionID = "repair_plan"
	NextActionResolveDependencies NextActionID = "resolve_dependencies"
	NextActionReview              NextActionID = "review"
	NextActionViewExecution       NextActionID = "view_execution"
	NextActionRun                 NextActionID = "run"
	NextActionRetry               NextActionID = "retry"
	NextActionArchive             NextActionID = "archive"
)

// NextActionProjection is a read-only resolution. Existing backlog, plan,
// review, and queue endpoints remain the only mutation authority.
type NextActionProjection struct {
	ID            NextActionID     `json:"id"`
	CompactLabel  string           `json:"compact_label"`
	ExpandedLabel string           `json:"expanded_label"`
	Enabled       bool             `json:"enabled"`
	Reason        string           `json:"reason,omitempty"`
	Blockers      []BlockingReason `json:"blockers,omitempty"`
	Target        string           `json:"target,omitempty"`
}

type nextActionBatchRequest struct {
	Items []string `json:"items"`
}

type nextActionBatchResult struct {
	Item   string                `json:"item"`
	Action *NextActionProjection `json:"action,omitempty"`
	Error  string                `json:"error,omitempty"`
}

const maxNextActionBatch = 100

// ResolveNextAction resolves one item with the same ProcessPreflight used by
// queueing. It deliberately never infers plan readiness from plan_ref alone.
func (h *Handler) ResolveNextAction(ctx context.Context, item BacklogItem) (NextActionProjection, error) {
	if IsArchived(item) {
		return nextAction(NextActionNone, "", "", false, "This backlog item is archived.", nil, ""), nil
	}
	if item.Status == StatusSuggested {
		return nextAction(NextActionAcceptSuggestion, "Accept", "Accept suggestion", true, "Accept this suggestion before planning or execution.", nil, "suggestion_accept"), nil
	}
	if IsTerminalStatus(item.Status) {
		if item.Status == StatusFailed {
			return nextAction(NextActionRetry, "Retry", "Retry execution", true, "The latest work ended in failure.", nil, "retry"), nil
		}
		return nextAction(NextActionArchive, "Archive", "Archive item", true, "Work is complete.", nil, "archive"), nil
	}
	if IsReviewStatus(item.Status) {
		return nextAction(NextActionReview, "Review", "Review evidence", true, "This item is awaiting review.", nil, "review"), nil
	}
	if item.Status == StatusQueued || item.Status == StatusInProgress {
		return nextAction(NextActionViewExecution, "View run", "View execution", true, "Work is already active.", nil, "execution"), nil
	}
	if !hasCanonicalExecutionPlan(item) {
		return nextAction(NextActionAuthorPlan, "Plan", "Author plan", true, "A canonical execution plan is required before this item can run.", nil, "plan_author"), nil
	}
	if h.executionQueuer == nil {
		return nextAction(NextActionNone, "Unavailable", "Execution readiness unavailable", false, "Execution service is not available.", nil, ""), nil
	}
	preflight, err := h.executionQueuer.ProcessPreflight(ctx, string(item.Kind), item.Name)
	if err != nil {
		return NextActionProjection{}, err
	}
	blockers, err := h.collectQueueBlockingReasons(item, item.Kind, item.Name, preflight)
	if err != nil {
		return NextActionProjection{}, err
	}
	if preflight.Ready && len(blockers) == 0 {
		return nextAction(NextActionRun, "Run", "Run item", true, "The current accepted plan is ready for execution.", nil, "run"), nil
	}
	if hasBlocker(blockers, "changed after acceptance") || hasBlocker(blockers, "not been explicitly accepted") {
		return nextAction(NextActionAcceptPlan, "Plan", "Accept plan", true, firstBlocker(blockers), blockers, "plan_accept"), nil
	}
	if hasBlocker(blockers, "not valid") || hasBlocker(blockers, "not executable") {
		return nextAction(NextActionRepairPlan, "Plan", "Repair plan", true, firstBlocker(blockers), blockers, "plan_repair"), nil
	}
	if hasBlocker(blockers, "unmet dependencies") {
		return nextAction(NextActionResolveDependencies, "Blocked", "Resolve dependencies", true, firstBlocker(blockers), blockers, "dependencies"), nil
	}
	return nextAction(NextActionNone, "Blocked", "Resolve blockers", false, firstBlocker(blockers), blockers, ""), nil
}

func hasCanonicalExecutionPlan(item BacklogItem) bool {
	return item.PlanRef != nil && strings.TrimSpace(item.PlanRef.Provider) == PlanRefProviderPlanManager &&
		strings.TrimSpace(item.PlanRef.Role) == PlanRefRoleExecutionSpec &&
		(strings.TrimSpace(item.PlanRef.PlanID) != "" || strings.TrimSpace(item.PlanRef.Slug) != "")
}

func nextAction(id NextActionID, compact, expanded string, enabled bool, reason string, blockers []BlockingReason, target string) NextActionProjection {
	return NextActionProjection{ID: id, CompactLabel: compact, ExpandedLabel: expanded, Enabled: enabled, Reason: reason, Blockers: blockers, Target: target}
}

func hasBlocker(blockers []BlockingReason, fragment string) bool {
	for _, blocker := range blockers {
		if strings.Contains(strings.ToLower(blocker.Message), strings.ToLower(fragment)) {
			return true
		}
	}
	return false
}

func firstBlocker(blockers []BlockingReason) string {
	if len(blockers) == 0 {
		return "This item is not ready for execution."
	}
	return blockers[0].Message
}

// NextAction serves the canonical single-item projection.
func (h *Handler) NextAction(w http.ResponseWriter, r *http.Request) {
	kind, name, ok := h.parseKindAndName(w, r, "next-action")
	if !ok {
		return
	}
	item, err := h.store.LoadItem(kind, name)
	if err != nil {
		h.writeNextActionError(w, err)
		return
	}
	action, err := h.ResolveNextAction(r.Context(), item)
	if err != nil {
		h.writeNextActionError(w, err)
		return
	}
	if err := httputil.JSON(w, map[string]any{"item": string(kind) + "/" + name, "action": action}); err != nil {
		apierr.MapError(w, "[backlog] next-action", apierr.Internal("failed to encode response"))
	}
}

// NextActionBatch resolves a bounded visible-list batch without mutating work.
func (h *Handler) NextActionBatch(w http.ResponseWriter, r *http.Request) {
	var req nextActionBatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierr.MapError(w, "[backlog] next-action batch", apierr.BadRequest("invalid request body"))
		return
	}
	if len(req.Items) == 0 || len(req.Items) > maxNextActionBatch {
		apierr.MapError(w, "[backlog] next-action batch", apierr.BadRequest("items must contain between 1 and %d references", maxNextActionBatch))
		return
	}
	results := make([]nextActionBatchResult, 0, len(req.Items))
	for _, ref := range req.Items {
		kind, name, err := parseDependencyRef(ref)
		if err != nil {
			results = append(results, nextActionBatchResult{Item: ref, Error: "invalid backlog reference"})
			continue
		}
		item, err := h.store.LoadItem(kind, name)
		if err != nil {
			results = append(results, nextActionBatchResult{Item: ref, Error: nextActionErrorMessage(err)})
			continue
		}
		action, err := h.ResolveNextAction(r.Context(), item)
		if err != nil {
			results = append(results, nextActionBatchResult{Item: ref, Error: nextActionErrorMessage(err)})
			continue
		}
		results = append(results, nextActionBatchResult{Item: ref, Action: &action})
	}
	if err := httputil.JSON(w, map[string]any{"results": results}); err != nil {
		apierr.MapError(w, "[backlog] next-action batch", apierr.Internal("failed to encode response"))
	}
}

func (h *Handler) writeNextActionError(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrNotFound) || os.IsNotExist(err) {
		apierr.MapError(w, "[backlog] next-action", apierr.NotFound("backlog item not found"))
		return
	}
	apierr.MapError(w, "[backlog] next-action", apierr.Internal("failed to resolve next action"))
}

func nextActionErrorMessage(err error) string {
	if errors.Is(err, ErrNotFound) || os.IsNotExist(err) {
		return "backlog item not found"
	}
	return "failed to resolve next action"
}
