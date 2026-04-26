package main

import (
	"context"
	"log/slog"

	"swarm-manager/internal/backlog"
	"swarm-manager/internal/execution"
	"swarm-manager/internal/graph"
	"swarm-manager/internal/initiativelock"
	"swarm-manager/internal/initiativereview"
)

// registerInitiativeReviewRoutes wires the initiative review surface:
// constructs the service from the already-initialized initiative store,
// backlog store, graph materializer, agent service, per-initiative lock,
// execution-record lookup, and GCT (git-control-tower) client; mounts
// the HTTP handler; and installs the backlog review-decide trigger hook
// so that when an item flips to a terminal status the initiative review
// is considered automatically.
//
// Depends on registerInitiativeRoutes, registerBacklogRoutes,
// registerGraphRoutes, and registerExecutionRoutes having already run.
// Safe to call when the agent service is disabled — the service degrades
// to a no-spawner round that still records the lifecycle but doesn't
// reach out to agent-manager.
//
// The lock is the same `.feedback-lock` file the feedback service acquires,
// so feedback-in-flight blocks review spawn and vice versa. This is the
// single-agent-per-initiative guarantee the plan required.
func (s *Server) registerInitiativeReviewRoutes(materializer *graph.Materializer) {
	if s.initStore == nil || s.initiativeService == nil || s.backlogHandler == nil {
		return
	}

	lock := &initiativelock.Lock{Dir: s.initiativeService.InitDir}

	svc, err := initiativereview.NewService(initiativereview.Config{
		InitStore:       s.initStore,
		BacklogLoader:   newInitiativeReviewBacklogAdapter(s.backlogHandler.Store()),
		GraphReader:     materializer,
		Spawner:         s.agentSvc,
		Lock:            lock,
		ExecutionLookup: newInitiativeReviewExecutionAdapter(s.executionSvc, s.backlogHandler.Store()),
		GCTClient:       newInitiativeReviewGCTAdapter(execution.NewHTTPReviewClient(nil)),
	})
	if err != nil {
		slog.Warn("initiative-review: build service", "err", err)
		return
	}
	s.initiativeReviewSvc = svc

	handler := initiativereview.NewHandler(svc)
	handler.RegisterRoutes(s.router)

	// Install trigger hook so the backlog review-decide endpoint notifies
	// initiative review when an item reaches terminal. The hook is
	// idempotent: it no-ops if the initiative is already under review or
	// has outstanding non-terminal items.
	s.backlogHandler.SetItemTerminalHandler(func(ctx context.Context, kind, name string, _ backlog.BacklogStatus) {
		svc.TriggerForItem(ctx, kind, name)
	})
}

// initiativeReviewBacklogAdapter narrows the backlog.Store surface down to
// the two read methods initiativereview.BacklogLoader requires. Isolating
// the adapter here keeps initiativereview itself free of package coupling
// to backlog.Store's full interface.
type initiativeReviewBacklogAdapter struct {
	store backlog.Store
}

func newInitiativeReviewBacklogAdapter(store backlog.Store) *initiativeReviewBacklogAdapter {
	if store == nil {
		return nil
	}
	return &initiativeReviewBacklogAdapter{store: store}
}

func (a *initiativeReviewBacklogAdapter) LoadItem(kind backlog.BacklogKind, name string) (backlog.BacklogItem, error) {
	return a.store.LoadItem(kind, name)
}

func (a *initiativeReviewBacklogAdapter) ItemDir(kind backlog.BacklogKind, name string) string {
	return a.store.ItemDir(kind, name)
}

// initiativeReviewExecutionAdapter satisfies initiativereview.ExecutionLookup
// by pulling the latest completed-or-accepted execution record for a
// backlog item and surfacing its affected-scenarios list — initiative
// review unions these across member items to decide which scenarios to
// run a fresh GCT pass against.
//
// Items that never ran finalization (research, archived pre-finalization,
// etc.) return (nil, nil) — absence is not an error, just "no scenarios
// in scope for this item".
type initiativeReviewExecutionAdapter struct {
	exec  *execution.Service
	store backlog.Store
}

func newInitiativeReviewExecutionAdapter(exec *execution.Service, store backlog.Store) *initiativeReviewExecutionAdapter {
	if exec == nil || store == nil {
		return nil
	}
	return &initiativeReviewExecutionAdapter{exec: exec, store: store}
}

func (a *initiativeReviewExecutionAdapter) LatestFinalizationFor(kind backlog.BacklogKind, name string) (*initiativereview.ItemFinalization, error) {
	if a == nil || a.exec == nil {
		return nil, nil
	}
	records, err := a.exec.List(context.Background(), execution.ListFilters{
		BacklogKind: string(kind),
		BacklogName: name,
	})
	if err != nil {
		return nil, err
	}
	// execution.Service.List is ordered by created_at descending, so the
	// first record with a populated Finalization carries the most recent
	// affected-scenarios list. Manually-accepted records still carry the
	// previous finalization — we surface whichever exists most recently.
	for _, r := range records {
		if r.Finalization == nil {
			continue
		}
		return &initiativereview.ItemFinalization{
			AffectedScenarios: append([]string(nil), r.Finalization.AffectedScenarios...),
		}, nil
	}
	return nil, nil
}

// initiativeReviewGCTAdapter wraps execution.ReviewClient to satisfy
// initiativereview.GCTClient. Initiative review needs only the
// "scenario name → verdict" shape, so this adapter builds a minimal
// ReviewRequest (no SandboxID, no ExpectedPaths, no thresholds) and
// projects execution.ReviewResult into initiativereview.GCTResult.
//
// Thresholds are intentionally omitted: initiative review assesses
// cross-item integration health, not a specific change's readiness
// gate. Using GCT's defaults keeps verdicts comparable across runs.
type initiativeReviewGCTAdapter struct {
	client execution.ReviewClient
}

func newInitiativeReviewGCTAdapter(client execution.ReviewClient) *initiativeReviewGCTAdapter {
	if client == nil {
		return nil
	}
	return &initiativeReviewGCTAdapter{client: client}
}

func (a *initiativeReviewGCTAdapter) TriggerReview(ctx context.Context, scenarioName string) (string, error) {
	if a == nil || a.client == nil {
		return "", nil
	}
	return a.client.TriggerReview(ctx, execution.ReviewRequest{ScenarioName: scenarioName})
}

func (a *initiativeReviewGCTAdapter) PollReview(ctx context.Context, jobID string) (*initiativereview.GCTResult, bool, error) {
	if a == nil || a.client == nil {
		return nil, true, nil
	}
	result, done, err := a.client.PollReview(ctx, jobID)
	if err != nil || !done || result == nil {
		return nil, done, err
	}
	return &initiativereview.GCTResult{
		JobID:          result.JobID,
		Classification: result.Classification,
		Summary:        result.Summary,
		RawDimensions:  result.RawDimensions,
		ReviewedAt:     result.ReviewedAt,
	}, true, nil
}
