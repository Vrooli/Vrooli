package main

import (
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
	}
	s.reviewSvc = review.NewService(cfg)
	s.reviewHandler = review.NewHandler(s.reviewSvc)
	s.reviewHandler.RegisterRoutes(s.router)

	// Wire review service into execution service for finalization integration.
	if execSvc != nil {
		execSvc.SetReviewService(s.reviewSvc)
	}
}
