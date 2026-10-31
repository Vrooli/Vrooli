package main

import (
	"context"
	"fmt"
	"net/http"
	"sort"

	"swarm-manager/internal/apierr"
	"swarm-manager/internal/backlog"
	"swarm-manager/internal/execution"
	"swarm-manager/internal/goals"
	"swarm-manager/internal/httputil"
	"swarm-manager/internal/planview"
)

// nextActionFeed is the cross-entity operator inbox projection. It deliberately
// composes domain-owned resolvers rather than reimplementing status logic.
type nextActionFeed struct {
	backlog   *backlog.Handler
	goals     *goals.Service
	decisions decisionCounter
}

// decisionCounter reads ready mutation proposals for a whole request. The
// projection depends on exactly one call per request, so the seam exists to
// let a test assert that count rather than infer it from latency.
type decisionCounter interface {
	countReadyDecisions(context.Context) (readyDecisionCounts, error)
}

type nextActionFeedEntry struct {
	EntityKind   string                       `json:"entity_kind"`
	EntityRef    string                       `json:"entity_ref"`
	EntityTitle  string                       `json:"entity_title"`
	Action       backlog.NextActionProjection `json:"action"`
	Tier         int                          `json:"tier"`
	GoalPriority int                          `json:"goal_priority,omitempty"`
	BacklogRank  int                          `json:"backlog_rank,omitempty"`
	CreatedAt    string                       `json:"created_at,omitempty"`
	ChainedRef   string                       `json:"chained_ref,omitempty"`
}

// planGoalActionAdapter lets planview display goal-owned actions without
// importing the HTTP composition layer or duplicating the goal funnel. It
// reads the shared projection, so the board's goal cards cost nothing beyond
// the projection its item cards already needed.
type planGoalActionAdapter struct{ projection *nextActionProjectionCache }

func (a planGoalActionAdapter) ListGoalActions(ctx context.Context) ([]planview.GoalAction, error) {
	entries, err := a.projection.Entries(ctx)
	if err != nil {
		return nil, err
	}
	actions := make([]planview.GoalAction, 0)
	for _, entry := range entries {
		if entry.EntityKind != "goal" {
			continue
		}
		actions = append(actions, planview.GoalAction{Name: entry.EntityRef, Title: entry.EntityTitle, Action: planActionFor(entry.Action.ID), Priority: entry.GoalPriority})
	}
	return actions, nil
}

func planActionFor(id backlog.NextActionID) string {
	switch id {
	case backlog.NextActionDecide:
		return planview.ActionDecide
	case backlog.NextActionReview:
		return planview.ActionReview
	case backlog.NextActionRun, backlog.NextActionDispatchFollowup, backlog.NextActionResolveDependencies:
		return planview.ActionRun
	case backlog.NextActionCloseOut:
		return planview.ActionFinalize
	default:
		return planview.ActionWorkshop
	}
}

// NextActionsFeed serves one ranked feed for the decision drawer and board.
func (c *nextActionProjectionCache) NextActionsFeed(w http.ResponseWriter, r *http.Request) {
	entries, err := c.Entries(r.Context())
	if err != nil {
		apierr.MapError(w, "[next-actions] feed", apierr.Internal("failed to resolve next-action feed"))
		return
	}
	if err := httputil.JSON(w, map[string]any{"entries": entries}); err != nil {
		apierr.MapError(w, "[next-actions] feed", apierr.Internal("failed to encode response"))
	}
}

// nextActionProjection is one whole-request answer: the resolved action for
// every backlog item plus the ranked inbox feed built from it. Both readers —
// the feed endpoint and the Plan board — take the same computation, so neither
// can pay for it twice nor drift from the other.
type nextActionProjection struct {
	actions map[string]backlog.NextActionProjection
	entries []nextActionFeedEntry
}

func (f nextActionFeed) resolve(ctx context.Context) ([]nextActionFeedEntry, error) {
	projection, err := f.project(ctx)
	if err != nil {
		return nil, err
	}
	return projection.entries, nil
}

func (f nextActionFeed) project(ctx context.Context) (nextActionProjection, error) {
	// Items that share a plan share its rendered content; without this scope
	// each one would issue its own plan-manager render RPC.
	ctx = execution.WithPlanRenderMemo(ctx)
	items, err := f.backlog.Store().LoadAll(nil)
	if err != nil {
		return nextActionProjection{}, err
	}
	itemByRef := make(map[string]backlog.BacklogItem, len(items))
	for _, item := range items {
		itemByRef[backlog.ItemRef(item)] = item
	}
	// One pass over the session store answers both halves of the inbox, and
	// one pass over the review archives answers every item.
	decisions := readyDecisionCounts{items: map[string]int{}, goals: map[string]int{}}
	if f.decisions != nil {
		decisions, err = f.decisions.countReadyDecisions(ctx)
		if err != nil {
			return nextActionProjection{}, err
		}
	}
	inputs := f.backlog.NextActionInputs(items, decisions.items)
	// The goal list and the item priority map come from one scope computation
	// over the items already loaded above: requesting them separately reloaded
	// the goal and backlog stores twice more.
	goalList, priorities, err := f.goals.ListWithItemPriorities(items)
	if err != nil {
		return nextActionProjection{}, err
	}
	actions := make(map[string]backlog.NextActionProjection, len(items))
	entries := make([]nextActionFeedEntry, 0, len(items))
	for _, item := range items {
		ref := backlog.ItemRef(item)
		action, resolveErr := f.backlog.ResolveNextActionWith(ctx, item, inputs[ref])
		if resolveErr != nil {
			continue
		}
		actions[ref] = action
		if !isInboxAction(action) {
			continue
		}
		entries = append(entries, nextActionFeedEntry{EntityKind: "backlog_item", EntityRef: ref, EntityTitle: item.Title, Action: action, Tier: actionTier(action.ID), GoalPriority: priorities[ref], BacklogRank: item.Priority, CreatedAt: item.Created})
	}
	for _, listed := range goalList {
		if entry, ok := goalEntry(listed, itemByRef, actions, decisions.goals[listed.Goal.Name]); ok {
			entries = append(entries, entry)
		}
	}
	for ref, count := range decisions.captures {
		if count < 1 {
			continue
		}
		entries = append(entries, nextActionFeedEntry{
			EntityKind: "capture", EntityRef: ref, EntityTitle: "Capture " + ref,
			Action: backlog.NextActionProjection{ID: backlog.NextActionDecide, CompactLabel: "Decide", ExpandedLabel: "Review capture proposals", Enabled: true, Reason: fmt.Sprintf("%d capture proposal(s) are waiting for an operator decision.", count), Target: "decision_stream"},
			Tier:   1,
		})
	}
	sort.SliceStable(entries, func(i, j int) bool {
		left, right := entries[i], entries[j]
		if left.Tier != right.Tier {
			return left.Tier < right.Tier
		}
		if left.GoalPriority != right.GoalPriority {
			return left.GoalPriority > right.GoalPriority
		}
		if left.BacklogRank != right.BacklogRank {
			return left.BacklogRank > right.BacklogRank
		}
		return left.CreatedAt < right.CreatedAt
	})
	return nextActionProjection{actions: actions, entries: entries}, nil
}

// goalEntry reads the item actions the projection already resolved rather than
// resolving a goal's closure members a second time.
func goalEntry(listed goals.GoalWithScope, items map[string]backlog.BacklogItem, actions map[string]backlog.NextActionProjection, proposalCount int) (nextActionFeedEntry, bool) {
	goal := listed.Goal
	entry := nextActionFeedEntry{EntityKind: "goal", EntityRef: goal.Name, EntityTitle: goal.Title, GoalPriority: goal.Priority, CreatedAt: goal.Created}
	input := goals.NextActionInput{ReadyProposalCount: proposalCount, ReviewMilestone: goals.ReviewableMilestone(goal, items)}
	for _, ref := range listed.Scope.Closure {
		if _, ok := items[ref]; !ok {
			continue
		}
		action, ok := actions[ref]
		if ok && isInboxAction(action) {
			input.ChainedAction, input.ChainedRef = action, ref
			break
		}
	}
	entry.Action, entry.ChainedRef = goals.ResolveNextAction(goal, input)
	if !isInboxAction(entry.Action) {
		return nextActionFeedEntry{}, false
	}
	entry.Tier = actionTier(entry.Action.ID)
	return entry, true
}

func isInboxAction(action backlog.NextActionProjection) bool {
	return action.Enabled && action.ID != backlog.NextActionNone && action.ID != backlog.NextActionViewExecution
}

func actionTier(id backlog.NextActionID) int {
	switch id {
	case backlog.NextActionDecide, backlog.NextActionReview:
		return 1
	case backlog.NextActionAcceptPlan, backlog.NextActionAuthorPlan, backlog.NextActionRepairPlan, backlog.NextActionPlanGoal, backlog.NextActionDefineCriteria:
		return 2
	case backlog.NextActionRun, backlog.NextActionDispatchFollowup, backlog.NextActionAuthorFollowup, backlog.NextActionResolveDependencies:
		return 3
	default:
		return 4
	}
}
