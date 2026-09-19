package perfbudget

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/vrooli/api-core/database"
	_ "modernc.org/sqlite"

	"scenario-to-cloud/backup"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/execplan"
	"scenario-to-cloud/operations"
	"scenario-to-cloud/persistence"
)

// TestRepeatedUpdateCyclesLeaveNoAccumulation [REQ:STC-P0-043] is the
// package-lane proof behind OPS-04: repeated update cycles through the
// durable operations owner leave no non-terminal operation, no held lease,
// no active step marker, no goroutine growth beyond the frozen post-cleanup
// delta, and a recovery-point set bounded by the frozen retention budget.
//
// It uses only exported seams: persistence.Repository over in-memory
// SQLite, operations.NewService with a RunnerFunc, backup.Plan.
func TestRepeatedUpdateCyclesLeaveNoAccumulation(t *testing.T) {
	b := loadBudgets(t)
	const cycles = 12

	db, err := sql.Open("sqlite", "file:perfbudget-cycles?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })
	repo := persistence.NewRepository(db)
	ctx := database.WithTestMode(context.Background())
	if err := repo.InitSchemaOnDialect(ctx, db, "sqlite"); err != nil {
		t.Fatal(err)
	}
	manifest, _ := json.Marshal(domain.CloudManifest{Version: "1", Target: domain.ManifestTarget{Type: "vps", VPS: &domain.ManifestVPS{Host: "203.0.113.10", Port: 22, User: "root", Workdir: "/root/Vrooli"}}, Scenario: domain.ManifestScenario{ID: "app"}})
	now := time.Now().UTC()
	if err := repo.CreateDeployment(ctx, &domain.Deployment{ID: "dep-1", Name: "app", ScenarioID: "app", Environment: "production", Status: domain.StatusPending, Manifest: manifest, CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}

	runner := operations.RunnerFunc(func(ctx context.Context, ec *operations.ExecutionContext) error {
		for _, action := range ec.Plan.Actions {
			decision, err := ec.Steps.Begin(ctx, action)
			if err != nil {
				return err
			}
			if decision == operations.DecisionSkip {
				continue
			}
			if err := ec.Steps.Commit(ctx, action, operations.StepSucceeded, "cycle", nil); err != nil {
				return err
			}
		}
		return nil
	})
	cfg := operations.Config{WorkerID: "cycles", Workers: 2, LeaseTTL: 500 * time.Millisecond, HeartbeatInterval: 100 * time.Millisecond, ReconcileInterval: time.Hour, ExecutionTimeout: 10 * time.Second, TransportTimeout: time.Second, ObserverTimeout: 5 * time.Second, QueueTimeout: time.Minute}
	svc := operations.NewService(cfg, repo, runner, nil, nil)
	svc.Start()
	t.Cleanup(svc.Stop)

	runtime.GC()
	baseline := SnapshotProcess()
	var points []domain.RecoveryPoint
	growth := b.Phase23.ManagementProcess.PostCleanupGrowth
	rpBudget := backup.Policy{KeepLast: b.Phase23.RetentionBudgets.RecoveryPoints.KeepLast, MaxAge: time.Duration(b.Phase23.RetentionBudgets.RecoveryPoints.Days) * 24 * time.Hour}

	for cycle := 1; cycle <= cycles; cycle++ {
		release := fmt.Sprintf("sha256:release-%d", cycle)
		plan := &execplan.Plan{
			SchemaVersion: "1", DeploymentID: "dep-1", ScenarioID: "app", Environment: "production", Scope: execplan.ScopeFull, Outcome: execplan.OutcomeApply,
			ReleaseDigest: release,
			Actions: []execplan.Action{
				{ID: "release.stage", OwnerOperation: "release.stage", Effect: "deployment_write", Retry: execplan.RetrySafeReplay, CancelPoint: true},
				{ID: "release.activate", OwnerOperation: "release.activate", Effect: "deployment_write", Retry: execplan.RetryRecover, Recovery: "rollback_to_predecessor"},
				{ID: "verify.readiness", OwnerOperation: "verify.readiness", Effect: "none", Retry: execplan.RetryObserveThenReplay, CancelPoint: true},
			},
		}
		raw, _ := json.Marshal(plan)
		digest, _ := plan.SemanticDigest()
		opID := fmt.Sprintf("op-%d", cycle)
		if _, err := repo.AdmitOperation(ctx, &domain.CloudOperation{ID: opID, DeploymentID: "dep-1", RequestKey: "cycle-" + opID, PlanDigest: digest, Plan: raw}); err != nil {
			t.Fatalf("cycle %d admit: %v", cycle, err)
		}
		// A recovery point captured under this release, as the update
		// plan's data.backup step would.
		points = append(points, domain.RecoveryPoint{ID: "rp-" + opID, DeploymentID: "dep-1", ReleaseDigest: release, OperationID: opID, CapturedAt: now.Add(time.Duration(cycle) * time.Minute)})

		svc.Submit(ctx, opID, operations.ExecuteOptions{})
		op, pending, err := svc.Wait(ctx, opID, 5*time.Second)
		if err != nil {
			t.Fatalf("cycle %d wait: %v", cycle, err)
		}
		if pending || op.State != domain.OperationSucceeded {
			t.Fatalf("cycle %d ended %s (pending=%v)", cycle, op.State, pending)
		}
		if op.WorkerID != "" || op.LeaseExpiresAt != nil || len(op.ActiveStep) > 0 && string(op.ActiveStep) != "null" {
			t.Fatalf("cycle %d terminal record still holds worker/lease/marker: worker=%q lease=%v active=%s", cycle, op.WorkerID, op.LeaseExpiresAt, op.ActiveStep)
		}
		if uint64(cycle) != op.Fence {
			t.Fatalf("cycle %d fence %d: one acquisition per cycle expected", cycle, op.Fence)
		}

		nonTerminal, err := repo.ListNonTerminal(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if len(nonTerminal) != 0 {
			t.Fatalf("cycle %d left %d non-terminal operations", cycle, len(nonTerminal))
		}

		// Retention over everything captured so far: only the active release
		// (this cycle) and its predecessor hold points; the rest are
		// reclaimable and the kept set is bounded by keep_last + protected.
		refs := backup.References{ActiveReleases: []string{release}}
		if cycle > 1 {
			refs.RetainedReleases = []string{fmt.Sprintf("sha256:release-%d", cycle-1)}
		}
		prune := backup.Plan(points, refs, rpBudget, now.Add(time.Duration(cycle)*time.Minute))
		if len(prune.Keep) > rpBudget.KeepLast+2 {
			t.Fatalf("cycle %d keeps %d recovery points (budget keep_last %d + 2 protected)", cycle, len(prune.Keep), rpBudget.KeepLast)
		}
		for _, id := range prune.Delete {
			if id == "rp-"+opID || (cycle > 1 && id == fmt.Sprintf("rp-op-%d", cycle-1)) {
				t.Fatalf("cycle %d retention deleted an active/predecessor point %s", cycle, id)
			}
		}
		if cycle > 1 {
			if _, held := prune.Protected["rp-"+opID]; !held {
				t.Fatalf("cycle %d active point unprotected: %+v", cycle, prune)
			}
		}
		// Apply the plan so the next cycle starts from the retained set.
		kept := points[:0]
		deleted := map[string]bool{}
		for _, id := range prune.Delete {
			deleted[id] = true
		}
		for _, p := range points {
			if !deleted[p.ID] {
				kept = append(kept, p)
			}
		}
		points = kept
	}

	all, err := repo.ListOperationsByDeployment(ctx, "dep-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != cycles {
		t.Fatalf("operation records = %d, want exactly %d (one per cycle, all terminal)", len(all), cycles)
	}
	for _, op := range all {
		if op.TerminalAt == nil || op.WorkerID != "" || op.LeaseExpiresAt != nil {
			t.Fatalf("operation %s not cleanly terminal: %+v", op.ID, op)
		}
	}

	// Retention over terminal records under the frozen 90-day budget: a
	// protected id and anything younger than the window survive; the rest
	// is reclaimable. Everything here is younger than the window, so the
	// frozen policy deletes nothing; a far-future cutoff deletes all but the
	// protected id and never a non-terminal record.
	receiptWindow := time.Duration(b.Phase23.RetentionBudgets.OperationReceipts.Days) * 24 * time.Hour
	if n, err := repo.PruneTerminalOperations(ctx, time.Now().UTC().Add(-receiptWindow), nil); err != nil || n != 0 {
		t.Fatalf("frozen retention pruned %d fresh records (err %v)", n, err)
	}
	if n, err := repo.PruneTerminalOperations(ctx, time.Now().UTC().Add(time.Hour), []string{"op-1"}); err != nil || n != cycles-1 {
		t.Fatalf("far-future prune removed %d records, want %d (err %v)", n, cycles-1, err)
	}
	if rest, err := repo.ListOperationsByDeployment(ctx, "dep-1"); err != nil || len(rest) != 1 || rest[0].ID != "op-1" {
		t.Fatalf("protected operation not kept: %v (err %v)", rest, err)
	}

	// Let heartbeat goroutines wind down, then compare with the baseline.
	deadline := time.Now().Add(2 * time.Second)
	var after ProcessSnapshot
	for {
		runtime.GC()
		after = SnapshotProcess()
		if after.Goroutines-baseline.Goroutines <= growth.GoroutinesMaxDelta || time.Now().After(deadline) {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if delta := after.Goroutines - baseline.Goroutines; delta > growth.GoroutinesMaxDelta {
		t.Fatalf("goroutines grew by %d over %d cycles (budget %d): %d → %d", delta, cycles, growth.GoroutinesMaxDelta, baseline.Goroutines, after.Goroutines)
	}
	if baseline.Available && after.Available {
		if delta := after.OpenFDs - baseline.OpenFDs; delta > growth.OpenFDsMaxDelta {
			t.Fatalf("open fds grew by %d over %d cycles (budget %d)", delta, cycles, growth.OpenFDsMaxDelta)
		}
		if delta := after.RSSMiB() - baseline.RSSMiB(); delta > growth.RSSMiBMaxDelta {
			t.Fatalf("rss grew by %.1f MiB over %d cycles (budget %.0f)", delta, cycles, growth.RSSMiBMaxDelta)
		}
	} else {
		t.Logf("/proc/self unavailable; fd and rss growth unobserved (goroutines only)")
	}
}
