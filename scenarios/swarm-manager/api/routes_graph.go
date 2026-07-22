package main

import (
	"context"
	"log/slog"
	"path/filepath"

	"swarm-manager/internal/goals"
	"swarm-manager/internal/graph"
	"swarm-manager/internal/scenarios"
)

// registerGraphRoutes constructs the projection service, WebSocket broker,
// and per-goal Materializer, then wires invalidation dispatch across
// mutating services. Returns the Materializer so downstream callers
// (proposal routes) can share the same instance instead of constructing
// a second one that wouldn't be connected to the invalidation hook.
func (s *Server) registerGraphRoutes(scenarioRoot string) *graph.Materializer {
	if s.backlogHandler == nil {
		return nil
	}

	projCfg := graph.ProjectionConfig{
		Backlog:    s.backlogHandler.Store(),
		Goal:       graph.NewGoalAdapter(s.goalService),
		Capture:    graph.NewCaptureAdapter(scenarioRoot),
		Scenario: graph.NewScenarioSourceAdapter(
			scenarios.NewDirectoryProvider(filepath.Dir(scenarioRoot)),
		),
	}
	if s.executionSvc != nil {
		projCfg.Execution = graph.NewExecutionAdapter(s.executionSvc)
	}
	projSvc := graph.NewProjectionService(projCfg)
	s.graphProjection = projSvc
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

	// Wire graph invalidation into mutating services and handlers. The
	// dispatcher is also stashed on the server so execution polling and
	// tests can reach it without constructing a parallel instance.
	dispatch := graph.NewDispatch(s.graphBroker, projectionCache)
	s.graphDispatch = dispatch

	// Per-goal graph.json materialization. The materializer writes a canonical
	// projection of each goal's item graph that agents
	// and UI components read instead of inferring from raw depends_on.
	// Boot-time: seed graph.json for every existing goal. Ongoing:
	// rebuild on any topology or backlog invalidation (coalesced).
	var materializer *graph.Materializer
	if s.goalService != nil && s.backlogHandler != nil {
		materializer = graph.NewMaterializer(
			graph.NewGoalAdapter(s.goalService),
			s.backlogHandler.Store(),
			goals.NewStore(s.dataRoot).GoalDir,
		)
		if err := materializer.MaterializeAll(context.Background()); err != nil {
			slog.Warn("boot-time graph.json materialization failed", "err", err)
		}
		dispatch.AddInvalidateHook(func(lenses []graph.Lens) {
			for _, lens := range lenses {
				if lens == graph.LensTopology {
					materializer.ScheduleAll()
					return
				}
			}
		})
	}

	if s.backlogHandler != nil {
		s.backlogHandler.SetEventDispatcher(dispatch)
	}
	if s.capturesHandler != nil {
		s.capturesHandler.SetEventDispatcher(dispatch)
	}
	if s.goalService != nil {
		s.goalService.SetEventDispatcher(dispatch)
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
	return materializer
}
