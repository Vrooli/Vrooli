package main

import (
	"log"

	"swarm-manager/internal/graph"
	"swarm-manager/internal/planview"
	"swarm-manager/internal/stats"
)

// registerPlanRoutes wires the Plan-lens board projection:
//
//	GET /api/v1/plan — Now/Next/Later/Done board (waves + next-action markers)
//
// Must run after registerBacklogRoutes (item store), registerExecutionRoutes
// (review gates / Done outcomes), and registerOperationsRoutes (Now summary
// counts). Board invalidation rides the graph dispatch: mutating services
// include the dispatch-only "plan" lens in their invalidation payloads, so
// /ws/graph clients on the Plan lens refetch without extra plumbing.
func (s *Server) registerPlanRoutes(scenarioRoot string) {
	if s.backlogHandler == nil {
		return
	}
	store := s.backlogHandler.Store()

	reviewSource := planview.ReviewSource{Store: store}
	if s.executionSvc != nil {
		reviewSource.Executions = graph.NewExecutionAdapter(s.executionSvc)
	}
	attentionSvc := planview.NewAttentionService(
		reviewSource,
		planview.ProposalSource{Store: s.agentSessionStore},
	)

	cfg := planview.Config{
		Backlog: store,
		Gates:   attentionSvc,
		// Both board inputs read the one shared projection, so a board render
		// computes it once instead of resolving every item itself and then
		// running the whole feed again for goal cards.
		NextActions: s.nextActions,
		GoalActions: planGoalActionAdapter{projection: s.nextActions},
	}
	if s.executionSvc != nil {
		cfg.Executions = graph.NewExecutionAdapter(s.executionSvc)
	}
	if s.opsAggregator != nil {
		cfg.Ops = s.opsAggregator
	}
	// Goal scoping: ?goal=<name> subsets the board to that goal's closure. The
	// goals service is registered before plan routes (main.go), so it is set here.
	if s.goalService != nil {
		cfg.Goals = s.goalService
	}
	// Board-wide ETA band: the whole backlog is the implicit closure. Reads the
	// same event-sourced samples + execute-lane capacity as the goals ETA.
	cfg.ETA = s.newETAEstimator
	if s.statsEngine != nil {
		s.statsEngine.Configure(stats.Config{
			Backlog: store,
			Goals:   s.goalService,
			ETA:     s.newETAEstimator,
		})
	}
	svc, err := planview.NewService(cfg)
	if err != nil {
		log.Fatalf("plan: failed to build projection service: %v", err)
	}
	planview.NewHandler(svc).RegisterRoutes(s.router)
}
