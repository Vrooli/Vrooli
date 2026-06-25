package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"swarm-manager/internal/backlog"
	"swarm-manager/internal/execution"
	"swarm-manager/internal/feedback"
	"swarm-manager/internal/graph"
	"swarm-manager/internal/initiativelock"
	"swarm-manager/internal/initiatives"
	"swarm-manager/internal/promptmanager"
	"swarm-manager/internal/proposals"
)

// initiativeFeedbackSkillFallbackID is used when the catalog lookup fails
// (e.g. catalog entry got removed mid-deploy). Hard-coded so feedback
// keeps working even if the catalog drifts.
const initiativeFeedbackSkillFallbackID = "swarm-manager-initiative-feedback"

var errFeedbackExecutionUnavailable = errors.New("execution service is not wired; interrupt_in_progress unavailable")

// registerFeedbackRoutes wires the feedback package: proposals.Applier,
// feedback.Service (with agent-manager spawner adapter, graph-backed state
// builder, on-disk lock, and the agent-state poller used by Handler.Get
// to advance rounds), and mounts feedback.Handler on the router.
func (s *Server) registerFeedbackRoutes(materializer *graph.Materializer) {
	if s.backlogHandler == nil || s.initiativeService == nil || s.initStore == nil {
		return
	}

	applier, err := s.buildProposalApplier(materializer)
	if err != nil {
		slog.Warn("feedback: failed to build proposals.Applier", "err", err)
		return
	}

	store := feedback.NewStore(s.initiativeService.InitDir)
	lock := &initiativelock.Lock{Dir: s.initiativeService.InitDir}
	if s.initStore != nil {
		sweepStaleFeedbackLocks(s.initStore, lock)
	}

	promptClient := promptmanager.NewHTTPClient()
	spawner := newFeedbackSpawnerAdapter(feedbackSpawnerConfig{
		Agent:         s.agentSvc,
		ActivitySvc:   s.agentActivitySvc,
		PromptClient:  promptClient,
		Materializer:  materializer,
		BacklogStore:  s.backlogHandler.Store(),
		InitStore:     s.initStore,
		FeedbackStore: store,
	})
	stateBuilder := newFeedbackStateBuilder(materializer, s.initStore, s.backlogHandler.Store())
	activity := newFeedbackActivityChecker(s.agentActivitySvc, s.initStore)
	poller := newFeedbackPoller(s.agentSvc)
	canceller := newFeedbackCanceller(s.agentSvc)

	svc, err := feedback.NewService(feedback.Config{
		Store:        store,
		Lock:         lock,
		Spawner:      spawner,
		Activity:     activity,
		Poller:       poller,
		Canceller:    canceller,
		Apply:        applier,
		StateBuilder: stateBuilder,
	})
	if err != nil {
		slog.Warn("feedback: failed to build Service", "err", err)
		return
	}

	handler := feedback.NewHandler(svc)
	handler.RegisterRoutes(s.router)

	// Stuck-round safety net: synchronous boot-time sweep clears any
	// rounds left in agent_thinking from a prior process (e.g. server
	// crashed while a feedback agent was running). The ticker goroutine
	// then keeps the invariant going for the lifetime of the process.
	sweeper := feedback.NewSweeper(svc, &initiativeNameLister{store: s.initStore})
	if dismissed, err := sweeper.RunOnce(context.Background()); err != nil {
		slog.Warn("feedback: boot-time stuck-round sweep failed", "err", err)
	} else if dismissed > 0 {
		slog.Info("feedback: boot-time stuck-round sweep dismissed rounds", "count", dismissed)
	}
	go func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		go func() {
			<-s.feedbackSweeperStop
			cancel()
		}()
		sweeper.Start(ctx)
	}()
}

// initiativeNameLister adapts initiatives.Store to feedback.InitiativeLister
// so the sweeper can enumerate initiatives without coupling the feedback
// package to the initiatives package.
type initiativeNameLister struct {
	store *initiatives.Store
}

func (l *initiativeNameLister) ListNames() ([]string, error) {
	if l == nil || l.store == nil {
		return nil, nil
	}
	all, err := l.store.LoadAll()
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

// --- Adapters ----------------------------------------------------------

// executionCancellerAdapter wires execution.Service into the proposals
// Applier's ExecutionCanceller interface — finds the most recent
// non-terminal execution for the backlog ref and cancels it.
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
	return nil
}

func isCancelableStatus(s execution.Status) bool {
	switch s {
	case execution.StatusPending, execution.StatusStarting, execution.StatusRunning,
		execution.StatusNeedsReview, execution.StatusValidating, execution.StatusNeedsFixup:
		return true
	}
	return false
}

func (s *Server) buildProposalApplier(materializer *graph.Materializer) (*proposals.Applier, error) {
	if s.backlogHandler == nil || s.initiativeService == nil {
		return nil, fmt.Errorf("backlog and initiative services are required")
	}
	// Guard against the typed-nil-into-interface trap: ServiceConfig.Events
	// is the CreationEventEmitter interface. Assigning a nil *eventlog.Emitter
	// would yield a non-nil interface that passes the `s.events != nil` check
	// in Service.Create and then panic on first method call. Only set Events
	// when we actually have a constructed emitter.
	cfg := backlog.ServiceConfig{
		Store:       s.backlogHandler.Store(),
		Assigner:    s.initiativeService,
		Invalidator: materializer,
		// Workshop and CycleChecker intentionally omitted: proposal-
		// applied items skip auto-workshop (agent already chose the
		// item) and cycle validation is performed by the Applier's
		// Validate phase using CurrentState.
	}
	if s.emitter != nil {
		cfg.Events = s.emitter
	}
	creator, err := backlog.NewService(cfg)
	if err != nil {
		return nil, fmt.Errorf("build backlog.Service for proposals: %w", err)
	}
	return proposals.NewApplier(proposals.Config{
		Store:       s.backlogHandler.Store(),
		Assigner:    s.initiativeService,
		Creator:     creator,
		Canceller:   newExecutionCancellerAdapter(s.executionSvc),
		Invalidator: materializer,
		Events:      &feedbackEventEmitter{eventlog: s.emitter},
	})
}

// sweepStaleFeedbackLocks releases lock files older than MaxAge for every
// initiative on disk. Run on boot so a server crash doesn't strand the
// initiative behind a dead lock.
func sweepStaleFeedbackLocks(initStore *initiatives.Store, lock *initiativelock.Lock) {
	all, err := initStore.LoadAll()
	if err != nil {
		slog.Warn("feedback: stale-lock sweep skipped", "err", err)
		return
	}
	swept := 0
	for _, init := range all {
		if init.Name == "" {
			continue
		}
		ok, err := lock.SweepStale(init.Name)
		if err != nil {
			slog.Warn("feedback: stale-lock sweep error", "initiative", init.Name, "err", err)
			continue
		}
		if ok {
			slog.Info("feedback: swept stale lock", "initiative", init.Name)
			swept++
		}
	}
	if swept > 0 {
		slog.Info("feedback: swept stale locks on boot", "count", swept)
	}
}
