package main

import (
	"context"
	"net/http"
	"sort"

	"swarm-manager/internal/agentsessions"
	"swarm-manager/internal/apierr"
	"swarm-manager/internal/backlog"
	"swarm-manager/internal/goals"
	"swarm-manager/internal/httputil"
	"swarm-manager/internal/planview"
)

// nextActionFeed is the cross-entity operator inbox projection. It deliberately
// composes domain-owned resolvers rather than reimplementing status logic.
type nextActionFeed struct {
	backlog  *backlog.Handler
	goals    *goals.Service
	sessions *agentsessions.Service
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
// importing the HTTP composition layer or duplicating the goal funnel.
type planGoalActionAdapter struct{ feed nextActionFeed }

func (a planGoalActionAdapter) ListGoalActions(ctx context.Context) ([]planview.GoalAction, error) {
	entries, err := a.feed.resolve(ctx)
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
func (f nextActionFeed) NextActionsFeed(w http.ResponseWriter, r *http.Request) {
	entries, err := f.resolve(r.Context())
	if err != nil {
		apierr.MapError(w, "[next-actions] feed", apierr.Internal("failed to resolve next-action feed"))
		return
	}
	if err := httputil.JSON(w, map[string]any{"entries": entries}); err != nil {
		apierr.MapError(w, "[next-actions] feed", apierr.Internal("failed to encode response"))
	}
}

func (f nextActionFeed) resolve(ctx context.Context) ([]nextActionFeedEntry, error) {
	items, err := f.backlog.Store().LoadAll(nil)
	if err != nil {
		return nil, err
	}
	itemByRef := make(map[string]backlog.BacklogItem, len(items))
	for _, item := range items {
		itemByRef[string(item.Kind)+"/"+item.Name] = item
	}
	priorities, err := f.goals.ItemGoalPriorities()
	if err != nil {
		return nil, err
	}
	entries := make([]nextActionFeedEntry, 0, len(items))
	goalProposalCounts, err := f.readyGoalProposalCounts(ctx)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		action, resolveErr := f.backlog.ResolveNextAction(ctx, item)
		if resolveErr != nil || !isInboxAction(action) {
			continue
		}
		ref := string(item.Kind) + "/" + item.Name
		entries = append(entries, nextActionFeedEntry{EntityKind: "backlog_item", EntityRef: ref, EntityTitle: item.Title, Action: action, Tier: actionTier(action.ID), GoalPriority: priorities[ref], BacklogRank: item.Priority, CreatedAt: item.Created})
	}
	goalList, err := f.goals.List()
	if err != nil {
		return nil, err
	}
	for _, listed := range goalList {
		if entry, ok := f.goalEntry(ctx, listed, itemByRef, goalProposalCounts[listed.Goal.Name]); ok {
			entries = append(entries, entry)
		}
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
	return entries, nil
}

// readyGoalProposalCounts reads the durable proposal state used by the goal
// funnel. Backlog proposal counts are supplied to the backlog resolver, so a
// feed never duplicates one entity as both a proposal and a backlog entry.
func (f nextActionFeed) readyGoalProposalCounts(ctx context.Context) (map[string]int, error) {
	counts := map[string]int{}
	if f.sessions == nil {
		return counts, nil
	}
	sessions, err := f.sessions.List(ctx, agentsessions.ListFilters{})
	if err != nil {
		return nil, err
	}
	for _, session := range sessions {
		for _, proposal := range session.Proposals {
			if proposal.Status != agentsessions.ProposalStatusReady || proposal.NeedsRevision || proposal.Target == nil || proposal.Target.Type != agentsessions.ContextGoal {
				continue
			}
			counts[proposal.Target.Ref]++
		}
	}
	return counts, nil
}

func (f nextActionFeed) goalEntry(ctx context.Context, listed goals.GoalWithScope, items map[string]backlog.BacklogItem, proposalCount int) (nextActionFeedEntry, bool) {
	goal := listed.Goal
	entry := nextActionFeedEntry{EntityKind: "goal", EntityRef: goal.Name, EntityTitle: goal.Title, GoalPriority: goal.Priority, CreatedAt: goal.Created}
	input := goals.NextActionInput{ReadyProposalCount: proposalCount, ReviewMilestone: goals.ReviewableMilestone(goal, items)}
	for _, ref := range listed.Scope.Closure {
		item, ok := items[ref]
		if !ok {
			continue
		}
		action, err := f.backlog.ResolveNextAction(ctx, item)
		if err == nil && isInboxAction(action) {
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

func feedAction(id backlog.NextActionID, compact, expanded, reason, target string) backlog.NextActionProjection {
	return backlog.NextActionProjection{ID: id, CompactLabel: compact, ExpandedLabel: expanded, Enabled: true, Reason: reason, Target: target}
}

func isInboxAction(action backlog.NextActionProjection) bool {
	return action.Enabled && action.ID != backlog.NextActionNone && action.ID != backlog.NextActionViewExecution
}

func actionTier(id backlog.NextActionID) int {
	switch id {
	case backlog.NextActionDecide, backlog.NextActionReview:
		return 1
	case backlog.NextActionAcceptPlan, backlog.NextActionAuthorPlan, backlog.NextActionRepairPlan, backlog.NextActionPlanGoal:
		return 2
	case backlog.NextActionRun, backlog.NextActionDispatchFollowup, backlog.NextActionAuthorFollowup, backlog.NextActionResolveDependencies:
		return 3
	default:
		return 4
	}
}
