package main

import (
	"swarm-manager/internal/graph"
	"swarm-manager/internal/scenarios"
)

func (s *Server) registerGraphRoutes(scenarioRoot string) {
	if s.backlogHandler == nil {
		return
	}

	projCfg := graph.ProjectionConfig{
		Backlog:    s.backlogHandler.Store(),
		Initiative: graph.NewInitiativeAdapter(s.initStore),
		Capture:    graph.NewCaptureAdapter(scenarioRoot),
		Scenario: graph.NewScenarioSourceAdapter(
			scenarios.NewCLIProviderWithOptions(scenarios.CLIProviderOptions{
				IncludePorts: false,
			}),
		),
	}
	if s.executionSvc != nil {
		projCfg.Execution = graph.NewExecutionAdapter(s.executionSvc)
	}
	if s.agentActivitySvc != nil {
		projCfg.Activity = s.agentActivitySvc
	}
	projSvc := graph.NewProjectionService(projCfg)
	projectionCache := graph.NewProjectionCache(graph.ProjectionCacheConfig{
		Projector: projSvc,
	})

	// HTTP handler.
	graphHandler := graph.NewHandler(projectionCache)
	graphHandler.RegisterRoutes(s.router)

	// WebSocket broker and stream handler.
	s.graphBroker = graph.NewBroker()
	streamHandler := graph.NewStreamHandler(s.graphBroker)
	streamHandler.RegisterRoutes(s.router)

	// Wire graph invalidation into mutating services and handlers.
	dispatch := graph.NewDispatch(s.graphBroker, projectionCache)
	if s.backlogHandler != nil {
		s.backlogHandler.SetEventDispatcher(dispatch)
	}
	if s.capturesHandler != nil {
		s.capturesHandler.SetEventDispatcher(dispatch)
	}
	if s.initiativeService != nil {
		s.initiativeService.SetEventDispatcher(dispatch)
	}
	if s.executionSvc != nil {
		s.executionSvc.SetEventDispatcher(dispatch)
	}
	if s.agentActivitySvc != nil {
		s.agentActivitySvc.SetEventDispatcher(dispatch)
	}
	if s.scenariosHandler != nil {
		s.scenariosHandler.SetEventDispatcher(dispatch)
	}
}
