package main

import (
	"log/slog"
	"path/filepath"
	"strings"
	"time"

	"swarm-manager/internal/backlog"
	"swarm-manager/internal/operatingmode"
	"swarm-manager/internal/opsbridge"
	"swarm-manager/internal/opscatalog"
	"swarm-manager/internal/opsrunner"
)

// orphanSnapshotGrace is how old an unreferenced execution snapshot must be
// before the startup reconciliation reaps it. It exceeds any realistic window
// between an Invoke writing its snapshot and committing the operation record, so
// a concurrent in-flight Invoke is never mistaken for an orphan.
const orphanSnapshotGrace = 15 * time.Minute

// sanitizePlanExecutionToken maps an arbitrary plan handle (a plan slug or UUID)
// onto a filesystem-safe directory token for the plan-execution workflow scope
// root. Deterministic for the same handle so store/load/scan agree.
func sanitizePlanExecutionToken(id string) string {
	var b strings.Builder
	for _, r := range strings.TrimSpace(id) {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '.', r == '_', r == '-':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	if b.Len() == 0 {
		return "unknown"
	}
	return b.String()
}

// registerBacklogOperationsRunner constructs the production declarative-operation
// runner for the backlog-item workflow and wires its async lifecycle into the
// operating-mode engine: the completion router observes terminal rounds and
// finalizes runner-owned operations, the refresh driver polls active target
// rounds (which nothing else drives), and the durable scheduler fires scheduled
// intents.
//
// This slice keeps the LEGACY backlog flow primary and untouched — the runner is
// constructed and its completion/refresh/scheduler machinery started, but no
// backlog request is rerouted through it yet (that is a later slice). The
// completion router is therefore a no-op for every current production round: it
// acts only on rounds whose run id correlates to a running runner operation, and
// nothing starts those yet. Wiring it now proves the production shape and lets
// the reroute land without touching startup.
//
// It is best-effort: a scenario without an authored operation catalog logs a
// notice and skips, because the runner is not yet a load-bearing dependency of
// any served surface.
func (s *Server) registerBacklogOperationsRunner(scenarioRoot string) {
	if s.operatingModeSvc == nil || s.backlogHandler == nil || s.initiativeService == nil {
		slog.Warn("backlog-ops runner: operating-mode/backlog/initiative services unavailable; skipping")
		return
	}
	catalog, err := opscatalog.Load(scenarioRoot)
	if err != nil {
		slog.Warn("backlog-ops runner: operation catalog not loadable; runner disabled", "err", err)
		return
	}
	modeDefs, err := operatingmode.LoadModesFromDir(scenarioRoot + "/modes")
	if err != nil {
		slog.Warn("backlog-ops runner: modes not loadable; runner disabled", "err", err)
		return
	}

	backlogStore := s.backlogHandler.Store()
	locator := opsrunner.FSLocator{
		BacklogItemDir: func(kind, name string) (string, error) {
			return backlogStore.ItemDir(backlog.BacklogKind(kind), name), nil
		},
		InitiativeDir: func(name string) (string, error) {
			return s.initiativeService.InitDir(name), nil
		},
		// plan-execution is a transient unit of work with no owning entity; its
		// workflow lives under a dedicated scope root beneath dataRoot (covered by
		// ScanRoots), keyed by a filesystem-safe projection of the plan handle.
		PlanExecutionDir: func(id string) (string, error) {
			return filepath.Join(s.dataRoot, "plan-executions", sanitizePlanExecutionToken(id)), nil
		},
		// scenario spec-sync is a transient unit of work with no owning entity; its
		// workflow lives under a dedicated scope root beneath dataRoot (covered by
		// ScanRoots), keyed by a filesystem-safe projection of the scenario name.
		ScenarioDir: func(name string) (string, error) {
			return filepath.Join(s.dataRoot, "scenario-runs", sanitizePlanExecutionToken(name)), nil
		},
		// ScanRoots let the refresh driver + scheduler reload persisted workflows
		// on boot so no in-memory timer state is load-bearing across a restart.
		// dataRoot is the backlog store's root, so it covers every item dir and the
		// plan-executions scope root.
		ScanRoots: []string{s.dataRoot},
	}

	registry := opsrunner.NewActionRegistry()
	backlog.RegisterOpsHandlers(registry, backlog.OpsHandlerDeps{Store: backlogStore})
	// Review completion handlers: commit-review-round + request-evidence
	// (backlog-item) and commit-initiative-review (initiative). Registered here so
	// the dependency edge to the review domains flows through the caller, keeping
	// opsbridge domain-free (mirrors backlog.RegisterOpsHandlers).
	if s.reviewSvc != nil {
		s.reviewSvc.RegisterOpsHandlers(registry)
	}
	if s.initiativeReviewSvc != nil {
		s.initiativeReviewSvc.RegisterOpsHandlers(registry)
	}
	// Execution completion handler: commit-execution-round is the single
	// completion authority for runner-owned execution records (execution-run/-retry
	// on plan-execution, execution-followup/-fixup on backlog-item). Registered here
	// so the dependency edge to the execution domain flows through the caller.
	if s.executionSvc != nil {
		s.executionSvc.RegisterOpsHandlers(registry)
	}

	built, err := opsbridge.BuildBacklogRunner(opsbridge.BacklogRunnerConfig{
		Catalog:     catalog,
		ModeDefs:    modeDefs,
		PhaseEngine: s.operatingModeSvc,
		SimEngine:   s.operatingModeSvc,
		Refresher:   s.operatingModeSvc,
		Locator:     locator,
		Registry:    registry,
		// The backlog handler resolves a deferred auto-advance intent into the
		// concrete workshop-round/finalize Invoke the scheduler firer runs.
		AdvanceResolver: s.backlogHandler,
		// Item targets inherit their initiative's binding overrides.
		InitiativeOfItem: initiativeOfItemFunc(backlogStore),
		RequestedBy:      "swarm-manager",
	})
	if err != nil {
		slog.Warn("backlog-ops runner: construction failed; runner disabled", "err", err)
		return
	}

	s.backlogOpsRunner = built
	// Inject the runner + scheduler into the backlog handler so the pre-execution
	// flows start operations through it (research/workshop refinement,
	// clarification) and schedule the deferred auto-advance timer on it.
	s.backlogHandler.SetRunner(built.Runner, built.Scheduler)
	// Inject the runner into the execution service so the primary execution start
	// launches the execution-run operation (against the item's plan-execution
	// target) instead of a direct agent spawn. Ordering is safe: the execution
	// service is built in registerExecutionRoutes (before this call).
	if s.executionSvc != nil {
		s.executionSvc.SetOperationStarter(&executionOperationStarter{runner: built.Runner})
	}
	// Inject the runner into the review + initiative-review services so their
	// reroutes start review-round / evidence-request / initiative-review
	// operations through it instead of a direct agent spawn.
	if s.reviewSvc != nil {
		s.reviewSvc.SetOperationStarter(&reviewOperationStarter{runner: built.Runner})
	}
	if s.initiativeReviewSvc != nil {
		s.initiativeReviewSvc.SetOperationStarter(&initiativeReviewOperationStarter{runner: built.Runner})
	}
	// Let the plan-execution target adapter inherit the write-scope containment of
	// the backlog item that owns a plan_ref, so an execution-drain run is
	// sandbox-scoped exactly as the legacy execution spawn was.
	s.operatingModeSvc.SetPlanContainmentResolver(&executionPlanContainmentResolver{store: backlogStore})
	// Route a runner-owned round's completion into Runner.CommitResult; every
	// other round (legacy initiative rounds, target rounds no operation started)
	// passes through untouched.
	s.operatingModeSvc.SetRoundObserver(built.Observer)

	// Startup reconciliation: reap orphan execution snapshots left on disk by
	// concurrent Invokes that lost the workflow compare-and-swap (a snapshot is
	// persisted before the operation record is appended, so a CAS conflict leaves
	// the file behind). Bounded, race-safe (only orphans older than the grace
	// period are removed), and idempotent. Runs the locator that was just wired so
	// it sees the same scope roots the runner uses.
	if report, err := opsrunner.ReconcileOrphanSnapshots(locator, time.Now(), orphanSnapshotGrace); err != nil {
		slog.Warn("backlog-ops runner: orphan-snapshot reconciliation failed", "err", err)
	} else if len(report.Reaped) > 0 || report.SkippedTooRecent > 0 {
		slog.Info("backlog-ops runner: orphan-snapshot reconciliation",
			"reaped", len(report.Reaped), "skipped_too_recent", report.SkippedTooRecent,
			"dirs_scanned", report.DirsScanned, "snapshots_seen", report.SnapshotsSeen)
	}
	slog.Info("backlog-ops runner constructed", "modes", len(modeDefs))
}
