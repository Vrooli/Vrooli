package main

import (
	"log"

	"swarm-manager/internal/gates"
	"swarm-manager/internal/graph"
	"swarm-manager/internal/planview"
)

// registerPlanRoutes wires the Plan-lens board projection:
//
//	GET /api/v1/plan — Now/Next/Later/Done board (waves + gates read-model)
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

	reviewSource := gates.ReviewSource{Store: store}
	if s.executionSvc != nil {
		reviewSource.Executions = graph.NewExecutionAdapter(s.executionSvc)
	}
	gateSvc := gates.NewService(
		gates.DecideSource{Store: store},
		gates.WorkshopSource{Store: store},
		reviewSource,
		gates.ClassifySource{Captures: planCaptureAdapter{inner: graph.NewCaptureAdapter(scenarioRoot)}},
	)

	cfg := planview.Config{
		Backlog: store,
		Gates:   gateSvc,
	}
	if s.executionSvc != nil {
		cfg.Executions = graph.NewExecutionAdapter(s.executionSvc)
	}
	if s.opsAggregator != nil {
		cfg.Ops = s.opsAggregator
	}
	svc, err := planview.NewService(cfg)
	if err != nil {
		log.Fatalf("plan: failed to build projection service: %v", err)
	}
	planview.NewHandler(svc).RegisterRoutes(s.router)
}

// planCaptureAdapter converts the graph capture adapter's entries into the
// gates classify-source shape.
type planCaptureAdapter struct {
	inner graph.CaptureLister
}

func (a planCaptureAdapter) ListCaptures() ([]gates.CaptureEntry, error) {
	caps, err := a.inner.ListCaptures()
	if err != nil {
		return nil, err
	}
	out := make([]gates.CaptureEntry, 0, len(caps))
	for _, c := range caps {
		out = append(out, gates.CaptureEntry{
			ID:              c.ID,
			Text:            c.Text,
			Status:          c.Status,
			ClassifiedItems: len(c.Items),
		})
	}
	return out, nil
}
