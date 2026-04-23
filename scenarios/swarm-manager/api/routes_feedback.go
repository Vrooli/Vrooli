package main

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/backlog"
	"swarm-manager/internal/execution"
	"swarm-manager/internal/feedback"
	"swarm-manager/internal/graph"
	"swarm-manager/internal/initiatives"
	"swarm-manager/internal/proposals"
)

var errFeedbackExecutionUnavailable = errors.New("execution service is not wired; interrupt_in_progress unavailable")

// registerFeedbackRoutes wires the feedback package: proposals.Applier,
// feedback.Service (with agent-manager spawner adapter, graph-backed state
// builder, and on-disk lock), and mounts feedback.Handler on the router.
//
// Depends on backlogHandler and initiativeService already being live
// (they are, per setupRoutes ordering) and on the graph Materializer
// which is constructed in registerGraphRoutes. Call order:
// backlog → initiatives → graph → feedback.
func (s *Server) registerFeedbackRoutes(materializer *graph.Materializer) {
	if s.backlogHandler == nil || s.initiativeService == nil || s.initStore == nil {
		return
	}

	applier, err := proposals.NewApplier(proposals.Config{
		Store:       s.backlogHandler.Store(),
		Assigner:    s.initiativeService,
		Canceller:   newExecutionCancellerAdapter(s.executionSvc),
		Invalidator: materializer,
	})
	if err != nil {
		slog.Warn("feedback: failed to build proposals.Applier", "err", err)
		return
	}

	store := feedback.NewStore(s.initiativeService.InitDir)
	lock := &feedback.Lock{Dir: s.initiativeService.InitDir}
	spawner := newFeedbackSpawnerAdapter(s.agentSvc)
	stateBuilder := newFeedbackStateBuilder(materializer, s.initStore, s.backlogHandler.Store())

	svc, err := feedback.NewService(feedback.Config{
		Store:        store,
		Lock:         lock,
		Spawner:      spawner,
		Apply:        applier,
		StateBuilder: stateBuilder,
	})
	if err != nil {
		slog.Warn("feedback: failed to build Service", "err", err)
		return
	}

	handler := feedback.NewHandler(svc)
	handler.RegisterRoutes(s.router)
}

// --- Adapters ----------------------------------------------------------

// executionCancellerAdapter wires execution.Service into the proposals
// Applier's ExecutionCanceller interface. The adapter finds the most
// recent non-terminal execution for the backlog ref and cancels it,
// mirroring what the backlog UI's "cancel run" button does.
type executionCancellerAdapter struct {
	svc *execution.Service
}

func newExecutionCancellerAdapter(svc *execution.Service) *executionCancellerAdapter {
	if svc == nil {
		return nil
	}
	return &executionCancellerAdapter{svc: svc}
}

func (a *executionCancellerAdapter) CancelForBacklog(ctx context.Context, kind, name string) error {
	if a == nil || a.svc == nil {
		// Degraded mode: no execution service wired. Surface a clear
		// error so the apply outcome records a Reason the user can act
		// on, rather than silently pretending to cancel.
		return errFeedbackExecutionUnavailable
	}
	records, err := a.svc.List(ctx, execution.ListFilters{
		BacklogKind: kind,
		BacklogName: name,
	})
	if err != nil {
		return err
	}
	for _, r := range records {
		if isCancelableStatus(r.Status) {
			if _, err := a.svc.Cancel(ctx, r.ExecutionID); err != nil {
				return err
			}
			return nil
		}
	}
	return nil // nothing to cancel is not an error
}

func isCancelableStatus(s execution.Status) bool {
	switch s {
	case execution.StatusPending, execution.StatusStarting, execution.StatusRunning,
		execution.StatusNeedsReview, execution.StatusValidating, execution.StatusNeedsFixup:
		return true
	}
	return false
}

// feedbackSpawnerAdapter wires agentmanager.SpawnInitiative into the
// feedback service's injected interface. Nil agent service degrades
// cleanly — the service falls back to the "no spawner" path which keeps
// round lifecycle moving without an agent (useful in disabled/test envs).
type feedbackSpawnerAdapter struct {
	agent *agentmanager.AgentService
}

func newFeedbackSpawnerAdapter(a *agentmanager.AgentService) *feedbackSpawnerAdapter {
	if a == nil {
		return nil
	}
	return &feedbackSpawnerAdapter{agent: a}
}

func (a *feedbackSpawnerAdapter) SpawnInitiativeFeedback(ctx context.Context, req feedback.SpawnRequest) (string, error) {
	res, err := a.agent.SpawnInitiative(ctx, agentmanager.InitiativeSpawnRequest{
		Name:        req.InitiativeName,
		Title:       "",
		Description: req.SubmissionText,
		Prompt:      req.SubmissionText,
		Purpose:     req.Purpose,
		RoundNumber: req.RoundNumber,
		RoundSlug:   req.RoundSlug,
	})
	if err != nil {
		return "", err
	}
	return res.RunID, nil
}

func (a *feedbackSpawnerAdapter) ContinueRun(ctx context.Context, runID, message string, _ []string) error {
	return a.agent.ContinueRun(ctx, runID, message)
}

// newFeedbackStateBuilder wraps the materializer + initiative store into a
// zero-arg closure the feedback service can call per-decide. The state is
// rebuilt from disk on each call so the Applier always sees the latest
// graph, even if other mutations landed between the agent's proposal and
// the user's decision.
func newFeedbackStateBuilder(
	m *graph.Materializer,
	initStore *initiatives.Store,
	backlogStore backlog.Store,
) func(string) (proposals.CurrentState, error) {
	return func(initiativeName string) (proposals.CurrentState, error) {
		mg, err := m.ReadGraph(initiativeName)
		if err != nil {
			return proposals.CurrentState{}, err
		}
		if mg == nil {
			// Graph hasn't been materialized yet (initiative just created,
			// or the materializer hasn't caught up). Trigger a build, then
			// fall through with an empty state so validation still enforces
			// "target must be a known ref" correctly against an empty set.
			_ = m.MaterializeInitiative(context.Background(), initiativeName)
			mg, _ = m.ReadGraph(initiativeName)
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
