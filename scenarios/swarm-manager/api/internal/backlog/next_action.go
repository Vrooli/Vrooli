package backlog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"swarm-manager/internal/apierr"
	"swarm-manager/internal/execution"
	"swarm-manager/internal/httputil"
	"swarm-manager/internal/nextaction"
)

// NextActionID is the stable, server-owned vocabulary for the primary action
// an operator can take on a backlog item. Presentation copy belongs to clients;
// policy does not.
type NextActionID = nextaction.ID

const (
	NextActionNone                = nextaction.None
	NextActionAcceptSuggestion    = nextaction.AcceptSuggestion
	NextActionAuthorPlan          = nextaction.AuthorPlan
	NextActionAcceptPlan          = nextaction.AcceptPlan
	NextActionRepairPlan          = nextaction.RepairPlan
	NextActionResolveDependencies = nextaction.ResolveDependencies
	NextActionReview              = nextaction.Review
	NextActionViewExecution       = nextaction.ViewExecution
	NextActionRun                 = nextaction.Run
	NextActionRetry               = nextaction.Retry
	NextActionArchive             = nextaction.Archive
	NextActionDecide              = nextaction.Decide
	NextActionDispatchFollowup    = nextaction.DispatchFollowup
	NextActionAuthorFollowup      = nextaction.AuthorFollowup
	NextActionPlanGoal            = nextaction.PlanGoal
	NextActionDefineCriteria      = nextaction.DefineCriteria
	NextActionCloseOut            = nextaction.CloseOut
	NextActionChain               = nextaction.Chain
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
	TransitionKey string           `json:"transition_key,omitempty"`
	FollowUp      *FollowUp        `json:"follow_up,omitempty"`
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

// ItemRef is the canonical "<kind>/<name>" reference used to address a
// backlog item across domains.
func ItemRef(item BacklogItem) string {
	return string(item.Kind) + "/" + item.Name
}

// NextActionInput carries the parts of a projection that come from outside
// the item itself. Resolving a list means resolving these once for the whole
// request; resolving them per item is what turns a list read into a scan of
// every contributing store per entry.
type NextActionInput struct {
	// PendingDecisions counts cross-domain decisions — durable mutation
	// proposals — waiting on this item.
	PendingDecisions int
	// PendingQuestions counts unreviewed review targets and requirements
	// held in the item's own archive.
	PendingQuestions int
}

// PendingDecisionCounts reads the cross-domain decision counts for the whole
// store once. Callers resolving more than one item must call it once and feed
// the result to NextActionInputs.
func (h *Handler) PendingDecisionCounts(ctx context.Context) (map[string]int, error) {
	if h.decisionCount == nil {
		return map[string]int{}, nil
	}
	return h.decisionCount.PendingDecisionCounts(ctx)
}

// NextActionInputs composes the per-item shared inputs for a set of items from
// one whole-store decision count map and one pass over the review archives.
func (h *Handler) NextActionInputs(items []BacklogItem, decisions map[string]int) map[string]NextActionInput {
	inputs := make(map[string]NextActionInput, len(items))
	for _, item := range items {
		if IsArchived(item) {
			continue
		}
		ref := ItemRef(item)
		inputs[ref] = NextActionInput{
			PendingDecisions: decisions[ref],
			PendingQuestions: len(collectReviewQuestions(h.store.ItemDir(item.Kind, item.Name), item.Kind, item.Name)),
		}
	}
	return inputs
}

// ResolveNextAction resolves a single item, gathering its shared inputs first.
// List callers must instead resolve inputs once with PendingDecisionCounts and
// NextActionInputs, then call ResolveNextActionWith per item.
func (h *Handler) ResolveNextAction(ctx context.Context, item BacklogItem) (NextActionProjection, error) {
	decisions, err := h.PendingDecisionCounts(ctx)
	if err != nil {
		return NextActionProjection{}, err
	}
	items := []BacklogItem{item}
	return h.ResolveNextActionWith(ctx, item, h.NextActionInputs(items, decisions)[ItemRef(item)])
}

// ResolveNextActionWith resolves one item from pre-resolved shared inputs,
// using the same ProcessPreflight used by queueing. It deliberately never
// infers plan readiness from plan_ref alone.
func (h *Handler) ResolveNextActionWith(ctx context.Context, item BacklogItem, input NextActionInput) (NextActionProjection, error) {
	if IsArchived(item) {
		return nextAction(NextActionNone, "", "", false, "This backlog item is archived.", nil, ""), nil
	}
	decisionCount := input.PendingQuestions + input.PendingDecisions
	if decisionCount > 0 {
		return nextAction(NextActionDecide, "Decide", "Resolve decisions", true, fmt.Sprintf("%d operator decision(s) are waiting on this item.", decisionCount), nil, "decision_stream"), nil
	}
	if item.Status == StatusSuggested {
		return nextAction(NextActionAcceptSuggestion, "Accept", "Accept suggestion", true, "Accept this suggestion before planning or execution.", nil, "suggestion_accept"), nil
	}
	if item.Status == StatusNeedsFollowup {
		if item.PendingFollowUp != nil {
			action := nextAction(NextActionDispatchFollowup, "Dispatch", "Dispatch follow-up", true, "This review decision has pending recovery work.", nil, "follow_up_dispatch")
			action.FollowUp = item.PendingFollowUp
			return action, nil
		}
		return nextAction(NextActionAuthorFollowup, "Follow-up", "Author follow-up", true, "This legacy follow-up decision needs a dispatch instruction.", nil, "follow_up_author"), nil
	}
	if IsTerminalStatus(item.Status) {
		if item.Status == StatusFailed {
			return nextAction(NextActionRetry, "Retry", "Retry execution", true, "The latest work ended in failure.", nil, "retry"), nil
		}
		return nextAction(NextActionArchive, "Archive", "Archive item", true, "Work is complete.", nil, "archive"), nil
	}
	if item.Status == StatusReviewPending {
		return nextAction(NextActionReview, "Review", "Review evidence", true, "This item is awaiting review.", nil, "review"), nil
	}
	if item.Status == StatusQueued || item.Status == StatusInProgress || item.Status == StatusInReview {
		return nextAction(NextActionViewExecution, "View run", "View execution", true, "Work is already active.", nil, "execution"), nil
	}
	if !hasCanonicalExecutionPlan(item) {
		return nextAction(NextActionAuthorPlan, "Plan", "Author plan", true, "A canonical execution plan is required before this item can run.", nil, "plan_author"), nil
	}
	if len(item.AcceptanceCriteria) == 0 {
		return nextAction(NextActionDefineCriteria, "Define criteria", "Define acceptance criteria", true, "This item needs typed acceptance criteria before it can be independently reviewed.", nil, "item_criteria"), nil
	}
	if h.executionQueuer == nil {
		return nextAction(NextActionNone, "Unavailable", "Execution readiness unavailable", false, "Execution service is not available.", nil, ""), nil
	}
	preflight := h.executionQueuer.ProcessPreflightForSpec(ctx, preflightSpec(item))
	blockers, err := h.collectQueueBlockingReasons(item, item.Kind, item.Name, preflight)
	if err != nil {
		return NextActionProjection{}, err
	}
	if err := validateBlockerCodes(blockers); err != nil {
		return NextActionProjection{}, err
	}
	if preflight.Ready && len(blockers) == 0 {
		return nextAction(NextActionRun, "Run", "Run item", true, "The current accepted plan is ready for execution.", nil, "run"), nil
	}
	for _, blocker := range blockers {
		switch nextaction.ActionForBlocker(blocker.Code) {
		case nextaction.AcceptPlan:
			return nextAction(NextActionAcceptPlan, "Plan", "Accept plan", true, firstBlocker(blockers), blockers, "plan_accept"), nil
		case nextaction.RepairPlan:
			return nextAction(NextActionRepairPlan, "Plan", "Repair plan", true, firstBlocker(blockers), blockers, "plan_repair"), nil
		case nextaction.ResolveDependencies:
			return nextAction(NextActionResolveDependencies, "Blocked", "Resolve dependencies", true, firstBlocker(blockers), blockers, "dependencies"), nil
		}
	}
	return nextAction(NextActionRun, "Run", "Retry queue when capacity clears", true, firstBlocker(blockers), blockers, "run"), nil
}

// preflightSpec projects an already-loaded item into the execution readiness
// input, so resolving an action never re-reads the spec it was built from.
func preflightSpec(item BacklogItem) execution.PreflightSpec {
	spec := execution.PreflightSpec{
		Kind:               string(item.Kind),
		Name:               item.Name,
		Title:              item.Title,
		Description:        item.Description,
		Status:             string(item.Status),
		SourceScenarioName: item.SourceScenarioName,
		AcceptanceAllow:    item.AcceptanceAllow,
		AcceptanceDeny:     item.AcceptanceDeny,
		Creates:            item.Creates,
		ArchivedAt:         item.ArchivedAt,
	}
	if item.PlanRef != nil {
		spec.PlanRef = &execution.PlanRefSpec{Provider: item.PlanRef.Provider, PlanID: item.PlanRef.PlanID, Slug: item.PlanRef.Slug, Role: item.PlanRef.Role}
	}
	if item.PlanAcceptance != nil {
		spec.PlanAcceptance = &execution.PlanAcceptanceSpec{
			Actor:           item.PlanAcceptance.Actor,
			AcceptedAt:      item.PlanAcceptance.AcceptedAt,
			PlanContentHash: item.PlanAcceptance.PlanContentHash,
			SubjectVersion:  item.PlanAcceptance.SubjectVersion,
		}
	}
	return spec
}

func hasCanonicalExecutionPlan(item BacklogItem) bool {
	return item.PlanRef != nil && strings.TrimSpace(item.PlanRef.Provider) == PlanRefProviderPlanManager &&
		strings.TrimSpace(item.PlanRef.Role) == PlanRefRoleExecutionSpec &&
		(strings.TrimSpace(item.PlanRef.PlanID) != "" || strings.TrimSpace(item.PlanRef.Slug) != "")
}

func nextAction(id NextActionID, compact, expanded string, enabled bool, reason string, blockers []BlockingReason, target string) NextActionProjection {
	return NextActionProjection{ID: id, CompactLabel: compact, ExpandedLabel: expanded, Enabled: enabled, Reason: reason, Blockers: blockers, Target: target, TransitionKey: TransitionKeyForNextAction(id)}
}

// TransitionKeyForNextAction is the one server-owned bridge from a next-action
// decision to a declared transition. Clients consume TransitionKey from the
// projection and never duplicate this capability mapping.
func TransitionKeyForNextAction(id NextActionID) string {
	return nextaction.TransitionKey(id)
}

func validateBlockerCodes(blockers []BlockingReason) error {
	for _, blocker := range blockers {
		if err := nextaction.ValidateBlockerCode(blocker.Code); err != nil {
			return err
		}
	}
	return nil
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
	// Load every requested item first so the shared inputs are gathered once
	// for the batch rather than once per entry. Results stay in request order:
	// callers index them positionally against the references they sent.
	results := make([]nextActionBatchResult, len(req.Items))
	loadedItems := make([]BacklogItem, 0, len(req.Items))
	resolvable := make(map[int]BacklogItem, len(req.Items))
	for index, ref := range req.Items {
		results[index] = nextActionBatchResult{Item: ref}
		kind, name, parseErr := parseDependencyRef(ref)
		if parseErr != nil {
			results[index].Error = "invalid backlog reference"
			continue
		}
		item, loadErr := h.store.LoadItem(kind, name)
		if loadErr != nil {
			results[index].Error = nextActionErrorMessage(loadErr)
			continue
		}
		resolvable[index] = item
		loadedItems = append(loadedItems, item)
	}
	decisions, err := h.PendingDecisionCounts(r.Context())
	if err != nil {
		apierr.MapError(w, "[backlog] next-action batch", apierr.Internal("failed to resolve next actions"))
		return
	}
	inputs := h.NextActionInputs(loadedItems, decisions)
	for index, item := range resolvable {
		action, resolveErr := h.ResolveNextActionWith(r.Context(), item, inputs[ItemRef(item)])
		if resolveErr != nil {
			results[index].Error = nextActionErrorMessage(resolveErr)
			continue
		}
		results[index].Action = &action
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
