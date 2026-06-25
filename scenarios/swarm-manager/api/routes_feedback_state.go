package main

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"swarm-manager/internal/agentactivity"
	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/backlog"
	"swarm-manager/internal/eventlog"
	"swarm-manager/internal/feedback"
	"swarm-manager/internal/graph"
	"swarm-manager/internal/initiatives"
	"swarm-manager/internal/proposals"
)

// newFeedbackStateBuilder closes over the materializer + initiative store
// so the feedback service can rebuild proposal CurrentState from disk on
// each Decide call.
func newFeedbackStateBuilder(
	m *graph.Materializer,
	initStore *initiatives.Store,
	backlogStore backlog.Store,
) func(string) (proposals.CurrentState, error) {
	return func(initiativeName string) (proposals.CurrentState, error) {
		mg, err := m.ReadGraph(initiativeName)
		if err != nil {
			return proposals.CurrentState{}, fmt.Errorf("read initiative graph: %w", err)
		}
		// First-touch materialization: initiatives without a graph.json
		// yet get one synchronously so proposal validation reasons
		// against real nodes. Silently falling back to a nil graph
		// would misreport every existing-target mutation as
		// "unknown target".
		if mg == nil {
			if mErr := m.MaterializeInitiative(context.Background(), initiativeName); mErr != nil {
				return proposals.CurrentState{}, fmt.Errorf("materialize initiative graph: %w", mErr)
			}
			mg, err = m.ReadGraph(initiativeName)
			if err != nil {
				return proposals.CurrentState{}, fmt.Errorf("read initiative graph after materialize: %w", err)
			}
			if mg == nil {
				return proposals.CurrentState{}, fmt.Errorf("initiative %q graph is still unavailable after materialize", initiativeName)
			}
		}
		known, err := knownInitiativeNames(initStore)
		if err != nil {
			return proposals.CurrentState{}, err
		}
		inProgress := inProgressRefs(backlogStore, mg)
		return proposals.FromMaterializedGraph(mg, known, inProgress)
	}
}

func knownInitiativeNames(store *initiatives.Store) ([]string, error) {
	all, err := store.LoadAll()
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(all))
	for _, i := range all {
		if strings.TrimSpace(i.Name) == "" {
			continue
		}
		names = append(names, i.Name)
	}
	return names, nil
}

func inProgressRefs(store backlog.Store, mg *graph.MaterializedGraph) []string {
	if mg == nil {
		return nil
	}
	refs := make([]string, 0)
	for _, n := range mg.Nodes {
		item, err := store.LoadItem(backlog.BacklogKind(n.Kind), n.Name)
		if err != nil {
			continue
		}
		if item.Status == backlog.StatusInProgress {
			refs = append(refs, n.ID)
		}
	}
	return refs
}

// --- Activity / Poller / Events ---------------------------------------

// feedbackActivityChecker enumerates each initiative item and asks
// agentactivity.Service whether a backlog-owned agent is running for it.
type feedbackActivityChecker struct {
	svc       *agentactivity.Service
	initStore *initiatives.Store
}

func newFeedbackActivityChecker(svc *agentactivity.Service, initStore *initiatives.Store) *feedbackActivityChecker {
	if svc == nil || initStore == nil {
		return nil
	}
	return &feedbackActivityChecker{svc: svc, initStore: initStore}
}

func (c *feedbackActivityChecker) ActiveRunsForInitiative(initiativeName string) ([]feedback.ItemActivity, error) {
	if c == nil {
		return nil, nil
	}
	init, err := c.initStore.Load(initiativeName)
	if err != nil || init == nil {
		return nil, err
	}
	var out []feedback.ItemActivity
	for _, ref := range init.Items {
		parts := strings.SplitN(ref, "/", 2)
		if len(parts) != 2 {
			continue
		}
		if !c.svc.HasActiveAgent(context.Background(), parts[0], parts[1]) {
			continue
		}
		records, listErr := c.svc.List(context.Background(), agentactivity.ListFilters{
			OwnerType:  string(agentactivity.OwnerBacklog),
			OwnerKind:  parts[0],
			OwnerName:  parts[1],
			ActiveOnly: true,
		})
		if listErr != nil || len(records) == 0 {
			out = append(out, feedback.ItemActivity{Ref: ref})
			continue
		}
		out = append(out, feedback.ItemActivity{
			Ref:     ref,
			RunID:   records[0].RunID,
			Purpose: string(records[0].Purpose),
		})
	}
	return out, nil
}

// feedbackPoller adapts agentmanager.AgentService to feedback.AgentRunPoller
// so Handler.Get can advance rounds via the same pull pattern that
// clarification uses (see internal/backlog/clarification_state.go).
type feedbackPoller struct {
	agent *agentmanager.AgentService
}

func newFeedbackPoller(agent *agentmanager.AgentService) *feedbackPoller {
	if agent == nil {
		return nil
	}
	return &feedbackPoller{agent: agent}
}

func (p *feedbackPoller) IsEnabled() bool {
	return p != nil && p.agent != nil && p.agent.IsEnabled()
}

func (p *feedbackPoller) GetRunState(ctx context.Context, runID string) (feedback.RunState, error) {
	state, err := p.agent.GetRunState(ctx, runID)
	if err != nil {
		return feedback.RunState{}, err
	}
	return feedback.RunState{
		Status:   state.Status,
		Summary:  state.Summary,
		ErrorMsg: state.ErrorMsg,
	}, nil
}

// feedbackCanceller adapts agentmanager.AgentService to feedback.RunCanceller.
// Used by the override path to cancel the preempted holder (and any busy
// item runs) before the new round takes the lock — turning "override" from
// a lock-file rename into actual single-agent enforcement.
//
// Nil-safe: when the agent service is not wired in (tests, degraded mode),
// we return nil so feedback.Service treats the canceller as absent and
// falls back to the plain lock-overwrite behavior rather than panicking.
type feedbackCanceller struct {
	agent *agentmanager.AgentService
}

func newFeedbackCanceller(agent *agentmanager.AgentService) *feedbackCanceller {
	if agent == nil {
		return nil
	}
	return &feedbackCanceller{agent: agent}
}

func (c *feedbackCanceller) StopRun(ctx context.Context, runID string) error {
	if c == nil || c.agent == nil {
		return nil
	}
	if strings.TrimSpace(runID) == "" {
		return nil
	}
	return c.agent.StopRun(ctx, runID)
}

// feedbackEventEmitter satisfies proposals.EventEmitter by appending a
// EventBacklogProposalApplied to the durable event log so attribution
// (feedback round / review round, decided-by, mutation id, op, target)
// survives restarts and is queryable per affected backlog item. A nil
// eventlog falls through to slog so test wiring still records the call.
type feedbackEventEmitter struct {
	eventlog *eventlog.Emitter
}

func (e *feedbackEventEmitter) EmitProposalMutationApplied(source proposals.Source, m proposals.Mutation) {
	target := proposalEventTarget(m)
	payload := eventlog.ProposalAppliedPayload{
		InitiativeName:  source.InitiativeName,
		Mode:            source.Mode,
		Phase:           source.Phase,
		FeedbackRoundID: source.FeedbackRoundID,
		ReviewRoundID:   source.ReviewRoundID,
		RoundNumber:     source.RoundNumber,
		RoundSlug:       source.RoundSlug,
		RunID:           source.RunID,
		Entrypoint:      source.Entrypoint,
		DecidedBy:       source.DecidedBy,
		MutationID:      m.ID,
		Op:              string(m.Op),
		Target:          target,
	}
	if m.Op == proposals.OpMergeItems && len(m.Sources) > 0 {
		payload.Sources = append([]string(nil), m.Sources...)
	}
	if e.eventlog != nil {
		e.eventlog.EmitBacklogProposalApplied(target, payload)
		return
	}
	slog.Info("proposals: mutation applied (no eventlog wired)",
		"initiative", source.InitiativeName,
		"feedback_round", source.FeedbackRoundID,
		"mutation_id", m.ID,
		"op", m.Op,
		"target", target,
	)
}

// proposalEventTarget picks the backlog ref to attach the event to:
// add_item and merge_items use the new item's ref; edge ops use From; all
// others use Target. Empty when the op has no natural per-item entity.
//
// merge_items intentionally attaches to the merged item rather than any
// individual source: the *primary new state* after merge is one item,
// and per-source history is preserved in the payload's Sources field
// (so source items still show "merged into X" via the same payload).
// Split is the inverse — its primary new state is N children, but
// attaching to one ref would be misleading, so split keeps the source
// ref as the event target.
func proposalEventTarget(m proposals.Mutation) string {
	switch m.Op {
	case proposals.OpAddItem, proposals.OpMergeItems:
		if m.Item != nil {
			return m.Item.Ref()
		}
	case proposals.OpAddEdge, proposals.OpRemoveEdge:
		return m.From
	}
	return m.Target
}
