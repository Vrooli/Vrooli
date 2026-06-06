package main

import (
	"context"
	"log/slog"
	"time"

	"swarm-manager/internal/backlog"
	"swarm-manager/internal/execution"
	"swarm-manager/internal/review"
	"swarm-manager/internal/runtimepaths"
	"swarm-manager/internal/scenarios"
	"swarm-manager/internal/settings"
)

func (s *Server) registerExecutionRoutes(dataRoot, scenarioRoot string) *execution.Service {
	// Create archiver from scenarios handler for post-spec-sync archive
	var archiver execution.Archiver
	if s.scenariosHandler != nil {
		archiver = scenarios.NewArchiver(s.scenariosHandler)
	}

	storePath, err := runtimepaths.StatePath("execution-runs.json")
	if err != nil {
		panic(err)
	}

	// Execution control endpoints
	cfg := execution.ServiceConfig{
		DataRoot:                 dataRoot,
		RepoRoot:                 scenarioRoot,
		StorePath:                storePath,
		SelfScenarioName:         "swarm-manager",
		PolicyProvider:           settings.NewPolicyAdapter(s.settingsStore),
		GovernanceProvider:       settings.NewGovernanceAdapter(s.settingsStore),
		ReviewThresholdsProvider: settings.NewReviewThresholdsAdapter(s.settingsStore),
		AgentService:             s.requireTrackedAgentService(),
		ScenarioLifecycle:        scenarios.NewCLILifecycle(),
		ScenarioHealthChecker:    scenarios.NewCLIHealthChecker(20 * time.Second),
		Archiver:                 archiver,
		ReviewClient:             execution.NewHTTPReviewClient(nil),
		BaselineClient:           execution.NewConnectBaselineClient(nil),
	}
	s.executionSvc = execution.NewService(cfg)
	// Wire the agentactivity service in as the lane reader so
	// GovernanceStatus reports per-lane utilization for all four canonical
	// lanes (Execute is also visible via execution.Records, but the other
	// three only live in the activity store).
	if s.agentActivitySvc != nil {
		s.executionSvc.SetActivityLaneReader(s.agentActivitySvc)
	}
	// Wire the fix-before-feature discovery filer (Tier 2). It files fix items
	// via the canonical backlog creation path. backlogHandler is registered
	// before this route group, so its store is available here.
	if s.backlogHandler != nil {
		filerCfg := backlog.ServiceConfig{
			Store:    s.backlogHandler.Store(),
			Assigner: s.initiativeService,
		}
		if s.emitter != nil {
			filerCfg.Events = s.emitter
		}
		if filerSvc, filerErr := backlog.NewService(filerCfg); filerErr == nil {
			s.executionSvc.SetRemediationFiler(backlog.NewFixDiscoveryFiler(filerSvc))
		}
	}
	s.executionHandler = execution.NewHandlerFromService(s.executionSvc)
	s.executionHandler.RegisterRoutes(s.router)

	// Wire execution queuer back into scenarios handler for spec-sync-archive
	if s.scenariosHandler != nil {
		s.scenariosHandler.SetExecutionQueuer(scenarios.NewExecutionQueuer(s.executionSvc))
	}
	if s.backlogHandler != nil {
		s.backlogHandler.SetExecutionQueuer(s.executionSvc)
	}
	return s.executionSvc
}

func (s *Server) registerReviewRoutes(scenarioRoot string, execSvc *execution.Service) {
	backlogStore := backlog.NewFileStore(s.dataRoot)
	cfg := review.ServiceConfig{
		DataRoot:     s.dataRoot,
		AgentService: s.requireTrackedAgentService(),
		ItemDirFn:    func(kind, name string) string { return backlogStore.ItemDir(backlog.BacklogKind(kind), name) },
		LoadItemTitle: func(kind, name string) (string, error) {
			item, err := backlogStore.LoadItem(backlog.BacklogKind(kind), name)
			if err != nil {
				return "", err
			}
			return item.Title, nil
		},
		// When a review round finishes (complete or failed), flip the backlog
		// item from in_review to review_pending so the user can assess and
		// decide the terminal status via the review-decide endpoint.
		OnRoundTerminal: func(ctx context.Context, kind, name string, round review.Round) {
			item, err := backlogStore.LoadItem(backlog.BacklogKind(kind), name)
			if err != nil {
				return
			}
			// Only transition from in_review; any other state means the user
			// (or another flow) has already taken over — don't overwrite.
			if item.Status != backlog.StatusInReview {
				return
			}
			item.Status = backlog.StatusReviewPending
			_ = backlogStore.SaveItem(item)
		},
	}
	if execSvc != nil {
		cfg.LoadExecutionContext = func(ctx context.Context, executionID string) (*review.ExecutionContext, error) {
			record, err := execSvc.Get(ctx, executionID)
			if err != nil {
				return nil, err
			}

			ctxOut := &review.ExecutionContext{
				BacklogKind:            record.BacklogKind,
				BacklogName:            record.BacklogName,
				ItemTitle:              record.BacklogName,
				AffectedScenarios:      []string{},
				ChangedPathsByScenario: map[string][]string{},
			}
			if item, err := backlogStore.LoadItem(backlog.BacklogKind(record.BacklogKind), record.BacklogName); err == nil {
				ctxOut.ItemTitle = item.Title
			}

			if record.Finalization == nil {
				return ctxOut, nil
			}

			ctxOut.AffectedScenarios = append([]string(nil), record.Finalization.AffectedScenarios...)
			resultsByScenario := make(map[string]any)
			baselineByScenario := make(map[string]execution.BaselineDiffResult)
			for _, scenario := range record.Finalization.Scenarios {
				if len(scenario.ChangedPaths) > 0 {
					ctxOut.ChangedPathsByScenario[scenario.ScenarioName] = append([]string(nil), scenario.ChangedPaths...)
				}
				if scenario.BaselineDiff != nil {
					baselineByScenario[scenario.ScenarioName] = *scenario.BaselineDiff
				}
				if scenario.Review.Result == nil {
					continue
				}
				resultsByScenario[scenario.ScenarioName] = map[string]any{
					"classification": scenario.Review.Result.Classification,
					"dimensions":     scenario.Review.Result.Dimensions,
					"raw_dimensions": scenario.Review.Result.RawDimensions,
					"summary":        scenario.Review.Result.Summary,
				}
			}
			ctxOut.GCTResultsJSON = review.MarshalScenarioGCTResults(resultsByScenario)
			ctxOut.BaselineDiffJSON = execution.MarshalBaselineDiffResults(baselineByScenario)

			return ctxOut, nil
		}
	}
	s.reviewSvc = review.NewService(cfg)
	s.reviewHandler = review.NewHandler(s.reviewSvc)
	s.reviewHandler.RegisterRoutes(s.router)

	// Wire review service into execution service for finalization integration.
	if execSvc != nil {
		execSvc.SetReviewService(s.reviewSvc)
	}

	// Wire the review-round liveness seam so the recover-review endpoint can
	// refuse to short-circuit an in-flight review.
	if s.backlogHandler != nil {
		s.backlogHandler.SetReviewRoundInspector(s.reviewSvc)

		// Orphaned-in_review safety net: a boot-time sweep recovers any items
		// stranded in in_review with no live review round (work done
		// out-of-band, a review run that died, or a premature mark), and the
		// ticker keeps the invariant for the process lifetime. Mirrors the
		// feedback sweeper. Recovery routes items to review_pending so a human
		// still decides the terminal state.
		sweeper := review.NewSweeper(s.reviewSvc, backlogStore, s.backlogHandler.RecoverOrphanedReview)
		if recovered, err := sweeper.RunOnce(context.Background()); err != nil {
			slog.Warn("review: boot-time orphaned-in_review sweep failed", "err", err)
		} else if recovered > 0 {
			slog.Info("review: boot-time orphaned-in_review sweep recovered items", "count", recovered)
		}
		go func() {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			go func() {
				<-s.reviewSweeperStop
				cancel()
			}()
			sweeper.Start(ctx)
		}()
	}
}
