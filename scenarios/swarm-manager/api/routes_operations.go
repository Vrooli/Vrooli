package main

import (
	"log"
	"path/filepath"

	"swarm-manager/internal/operations"
	"swarm-manager/internal/sessioncontext"
)

// registerOperationsRoutes wires the Operations Center HTTP surface:
//
//   - GET  /api/v1/operations           — aggregated lanes / queue / activity view
//   - POST /api/v1/operations/bulk-stop — server-serialized cancellation
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
	// Stashed for the plan-board projection's Now-column summary counts.
	s.opsAggregator = aggregator
	projectRoot := filepath.Dir(filepath.Dir(s.scenarioRoot))
	briefingBuilder, err := operations.NewBriefingBuilder(operations.BriefingBuilderConfig{
		Aggregator: aggregator,
		Overview:   s.overviewSvc,
		Stats:      s.statsEngine,
		Handoffs:   operations.FileDirectorHandoffReader{ProjectRoot: projectRoot},
	})
	if err != nil {
		log.Fatalf("operations: failed to build BriefingBuilder: %v", err)
	}
	handler := operations.NewHandler(aggregator)
	handler.SetBriefingBuilder(briefingBuilder)
	handler.RegisterRoutes(s.router)
	contextResolver := sessioncontext.NewResolver(s.scenarioRoot, filepath.Dir(s.scenarioRoot), s.agentSessionStore, briefingBuilder)
	// Wire the cached, ranked initiative snapshot into the session context so
	// the swarm_operations startup brief carries deterministic rankings. The
	// snapshot reuses the overview service; when it is absent (test wiring),
	// the brief degrades to the activity briefing alone.
	if s.overviewSvc != nil {
		snapshotBuilder, snapErr := operations.NewSnapshotBuilder(operations.SnapshotBuilderConfig{Overview: s.overviewSvc})
		if snapErr != nil {
			log.Fatalf("operations: failed to build SnapshotBuilder: %v", snapErr)
		}
		contextResolver.SetSnapshotBuilder(snapshotBuilder)
	}
	sessioncontext.NewHandler(contextResolver).RegisterRoutes(s.router)
	if s.agentSessionSvc != nil {
		s.agentSessionSvc.SetContextResolver(contextResolver)
	}

	// Bulk-stop reuses the same agentActivitySvc as both Stopper and
	// ActivityLister: StopRun cancels a single run end-to-end (manager
	// stop + ledger update + dispatch), and List feeds the filter-mode
	// resolver with the same source of truth the aggregator uses.
	bulkStop, err := operations.NewBulkStopHandler(s.agentActivitySvc, s.agentActivitySvc)
	if err != nil {
		log.Fatalf("operations: failed to build BulkStopHandler: %v", err)
	}
	bulkStop.RegisterRoutes(s.router)
}
