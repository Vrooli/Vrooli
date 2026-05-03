package main

import (
	"log"

	"swarm-manager/internal/operations"
)

// registerOperationsRoutes wires the Operations Center aggregate endpoint
// (`GET /api/v1/operations`).
//
// Must run after registerAgentActivityRoutes, registerExecutionRoutes, and
// registerOperatingModeRoutes — operations joins those three readers and
// will panic at construction if any are absent.
func (s *Server) registerOperationsRoutes() {
	if s.agentActivitySvc == nil {
		log.Fatalf("operations: agent activity service not initialized")
	}
	if s.executionSvc == nil {
		log.Fatalf("operations: execution service not initialized")
	}
	// Operating-mode service is optional — when absent (test wiring) the
	// aggregator omits round joins and ActivityRow.Mode/Phase/Round stay
	// empty for non-initiative records.
	cfg := operations.AggregatorConfig{
		Activities: s.agentActivitySvc,
		Governance: s.executionSvc,
	}
	if s.operatingModeSvc != nil {
		cfg.Rounds = s.operatingModeSvc
	}
	aggregator, err := operations.NewAggregator(cfg)
	if err != nil {
		log.Fatalf("operations: failed to build Aggregator: %v", err)
	}
	operations.NewHandler(aggregator).RegisterRoutes(s.router)
}
