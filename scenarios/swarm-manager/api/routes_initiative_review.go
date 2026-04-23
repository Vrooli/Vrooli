package main

import (
	"context"
	"log/slog"

	"swarm-manager/internal/backlog"
	"swarm-manager/internal/graph"
	"swarm-manager/internal/initiativereview"
)

// registerInitiativeReviewRoutes wires the initiative review surface:
// constructs the service from the already-initialized initiative store,
// backlog store, graph materializer, and agent service, mounts the HTTP
// handler, and installs the backlog review-decide trigger hook so that
// when an item flips to a terminal status the initiative review is
// considered automatically.
//
// Depends on registerInitiativeRoutes, registerBacklogRoutes, and
// registerGraphRoutes having already run. Safe to call when the agent
// service is disabled — the service degrades to a no-spawner round that
// still records the lifecycle but doesn't reach out to agent-manager.
func (s *Server) registerInitiativeReviewRoutes(materializer *graph.Materializer) {
	if s.initStore == nil || s.initiativeService == nil || s.backlogHandler == nil {
		return
	}

	svc, err := initiativereview.NewService(initiativereview.Config{
		InitStore:     s.initStore,
		BacklogLoader: newInitiativeReviewBacklogAdapter(s.backlogHandler.Store()),
		GraphReader:   materializer,
		Spawner:       s.agentSvc,
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

// recoverInitiativeReviewRounds loads initiative names from disk and asks
// the service to re-populate its in-memory tracking for any gathering
// rounds that survived a restart. Called once from main() after server
// wiring but before background workers start.
func (s *Server) recoverInitiativeReviewRounds() {
	if s.initiativeReviewSvc == nil || s.initStore == nil {
		return
	}
	inits, err := s.initStore.LoadAll()
	if err != nil {
		slog.Warn("initiative-review: load initiatives for recovery", "err", err)
		return
	}
	names := make([]string, 0, len(inits))
	for _, init := range inits {
		names = append(names, init.Name)
	}
	s.initiativeReviewSvc.RecoverActiveRounds(names)
}
