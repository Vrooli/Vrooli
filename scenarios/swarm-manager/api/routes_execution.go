package main

import (
	"context"
	"path/filepath"
	"time"

	"swarm-manager/internal/backlog"
	"swarm-manager/internal/execution"
	"swarm-manager/internal/review"
	"swarm-manager/internal/scenarios"
	"swarm-manager/internal/settings"
)

func (s *Server) registerExecutionRoutes(scenarioRoot string) *execution.Service {
	// Create archiver from scenarios handler for post-spec-sync archive
	var archiver execution.Archiver
	if s.scenariosHandler != nil {
		archiver = scenarios.NewArchiver(s.scenariosHandler)
	}

	// Execution control endpoints
	cfg := execution.ServiceConfig{
		RootDir:                  scenarioRoot,
		StorePath:                filepath.Join(scenarioRoot, ".vrooli", "execution-runs.json"),
		PolicyProvider:           settings.NewPolicyAdapter(s.settingsStore),
		GovernanceProvider:       settings.NewGovernanceAdapter(s.settingsStore),
		ReviewThresholdsProvider: settings.NewReviewThresholdsAdapter(s.settingsStore),
		AgentService:             s.requireTrackedAgentService(),
		ScenarioLifecycle:        scenarios.NewCLILifecycle(),
		ScenarioHealthChecker:    scenarios.NewCLIHealthChecker(20 * time.Second),
		Archiver:                 archiver,
		ReviewClient:             execution.NewHTTPReviewClient(nil),
	}
	s.executionSvc = execution.NewService(cfg)
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
	backlogStore := backlog.NewFileStore(scenarioRoot)
	cfg := review.ServiceConfig{
		RootDir:      scenarioRoot,
		AgentService: s.requireTrackedAgentService(),
		ItemDirFn:    func(kind, name string) string { return backlogStore.ItemDir(backlog.BacklogKind(kind), name) },
		LoadItemTitle: func(kind, name string) (string, error) {
			item, err := backlogStore.LoadItem(backlog.BacklogKind(kind), name)
			if err != nil {
				return "", err
			}
			return item.Title, nil
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
			for _, scenario := range record.Finalization.Scenarios {
				if len(scenario.ChangedPaths) > 0 {
					ctxOut.ChangedPathsByScenario[scenario.ScenarioName] = append([]string(nil), scenario.ChangedPaths...)
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
}
