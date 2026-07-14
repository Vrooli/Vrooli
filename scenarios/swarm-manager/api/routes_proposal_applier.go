package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"swarm-manager/internal/backlog"
	"swarm-manager/internal/eventlog"
	"swarm-manager/internal/execution"
	"swarm-manager/internal/graph"
	"swarm-manager/internal/initiatives"
	"swarm-manager/internal/proposals"
)

var errProposalExecutionUnavailable = errors.New("execution service is not wired; interrupt_in_progress unavailable")

func newProposalStateBuilder(m *graph.Materializer, initStore *initiatives.Store, backlogStore backlog.Store) proposals.StateBuilder {
	return func(initiativeName string) (proposals.CurrentState, error) {
		mg, err := m.ReadGraph(initiativeName)
		if err != nil {
			return proposals.CurrentState{}, fmt.Errorf("read initiative graph: %w", err)
		}
		if mg == nil {
			if err := m.MaterializeInitiative(context.Background(), initiativeName); err != nil {
				return proposals.CurrentState{}, fmt.Errorf("materialize initiative graph: %w", err)
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
		return proposals.FromMaterializedGraph(mg, known, inProgressRefs(backlogStore, mg))
	}
}

func knownInitiativeNames(store *initiatives.Store) ([]string, error) {
	all, err := store.LoadAll()
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(all))
	for _, i := range all {
		if strings.TrimSpace(i.Name) != "" {
			names = append(names, i.Name)
		}
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
		if err == nil && item.Status == backlog.StatusInProgress {
			refs = append(refs, n.ID)
		}
	}
	return refs
}

type executionCancellerAdapter struct{ svc *execution.Service }

func newExecutionCancellerAdapter(svc *execution.Service) *executionCancellerAdapter {
	if svc == nil {
		return nil
	}
	return &executionCancellerAdapter{svc: svc}
}
func (a *executionCancellerAdapter) CancelForBacklog(ctx context.Context, kind, name string) error {
	if a == nil || a.svc == nil {
		return errProposalExecutionUnavailable
	}
	records, err := a.svc.List(ctx, execution.ListFilters{BacklogKind: kind, BacklogName: name})
	if err != nil {
		return err
	}
	for _, r := range records {
		if isCancelableStatus(r.Status) {
			_, err := a.svc.Cancel(ctx, r.ExecutionID)
			return err
		}
	}
	return nil
}
func isCancelableStatus(s execution.Status) bool {
	switch s {
	case execution.StatusPending, execution.StatusStarting, execution.StatusRunning, execution.StatusNeedsReview, execution.StatusValidating, execution.StatusNeedsFixup:
		return true
	}
	return false
}

func (s *Server) buildProposalApplier(materializer *graph.Materializer) (*proposals.Applier, error) {
	if s.backlogHandler == nil || s.initiativeService == nil {
		return nil, fmt.Errorf("backlog and initiative services are required")
	}
	cfg := backlog.ServiceConfig{Store: s.backlogHandler.Store(), Assigner: s.initiativeService, Invalidator: materializer}
	if s.emitter != nil {
		cfg.Events = s.emitter
	}
	creator, err := backlog.NewService(cfg)
	if err != nil {
		return nil, fmt.Errorf("build backlog.Service for proposals: %w", err)
	}
	return proposals.NewApplier(proposals.Config{Store: s.backlogHandler.Store(), Assigner: s.initiativeService, Creator: creator, Canceller: newExecutionCancellerAdapter(s.executionSvc), Invalidator: materializer, Events: &proposalEventEmitter{eventlog: s.emitter}})
}

type proposalEventEmitter struct{ eventlog *eventlog.Emitter }

func (e *proposalEventEmitter) EmitProposalMutationApplied(source proposals.Source, m proposals.Mutation) {
	target := proposalEventTarget(m)
	payload := eventlog.ProposalAppliedPayload{InitiativeName: source.InitiativeName, Mode: source.Mode, Phase: source.Phase, FeedbackRoundID: source.FeedbackRoundID, ReviewRoundID: source.ReviewRoundID, RoundNumber: source.RoundNumber, RoundSlug: source.RoundSlug, RunID: source.RunID, SessionID: source.SessionID, Entrypoint: source.Entrypoint, DecidedBy: source.DecidedBy, MutationID: m.ID, Op: string(m.Op), Target: target}
	if m.Op == proposals.OpMergeItems && len(m.Sources) > 0 {
		payload.Sources = append([]string(nil), m.Sources...)
	}
	if e.eventlog != nil {
		e.eventlog.EmitBacklogProposalApplied(target, payload)
		return
	}
	slog.Info("proposals: mutation applied (no eventlog wired)", "initiative", source.InitiativeName, "session", source.SessionID, "mutation_id", m.ID, "op", m.Op, "target", target)
}
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
