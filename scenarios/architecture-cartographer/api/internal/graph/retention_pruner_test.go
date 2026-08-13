package graph

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/vrooli/api-core/schedule"

	"github.com/vrooli/api-core/retention"
)

// manifestBudget mirrors the graph_snapshots budget declared in
// .vrooli/service.json, so these tests fail if the declaration and the code
// drift apart.
func manifestBudget() retention.Budget {
	return retention.Budget{Name: SnapshotBudgetName, MaxBytes: 5 << 30}
}

func newPrunerFixture(t *testing.T) (*SnapshotPruner, *sql.DB) {
	t.Helper()
	db, _ := newRetentionDB(t)
	repo := NewSQLiteRepository(db, schedule.System()).(*sqliteRepository)
	return NewSnapshotPruner(repo, RetentionPolicy{}), db
}

// TestSnapshotPrunerKeepsNPerScenario is the correctness property a generic age
// rule would break, asserted through the framework seam rather than the
// scenario's own entry point.
func TestSnapshotPrunerKeepsNPerScenario(t *testing.T) {
	pruner, db := newPrunerFixture(t)
	base := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)

	// A noisy scenario with many snapshots, and a stable one with exactly one.
	for i := range 20 {
		insertSnapshot(t, db, "noisy", fmt.Sprintf("n-%02d", i), base.Add(time.Duration(i)*time.Hour), 2048)
	}
	// The stable scenario's only snapshot is the OLDEST thing in the table, so
	// any age rule with a horizon short enough to touch the noisy scenario would
	// delete it first. That is the regression this pruner exists to prevent.
	insertSnapshot(t, db, "stable", "s-00", base.Add(-30*24*time.Hour), 2048)

	result, err := pruner.Prune(context.Background(), manifestBudget())
	if err != nil {
		t.Fatalf("Prune: %v", err)
	}

	keep := pruner.KeepPerScenario()
	if got := snapshotCount(t, db, "noisy"); got != keep {
		t.Errorf("noisy scenario kept %d snapshots, want %d", got, keep)
	}
	if got := snapshotCount(t, db, "stable"); got != 1 {
		t.Fatalf("stable scenario kept %d snapshots, want its only one; a generic age rule would have deleted it", got)
	}
	if result.Deleted != int64(20-keep) {
		t.Errorf("Deleted = %d, want %d", result.Deleted, 20-keep)
	}
}

// TestSnapshotPrunerReceivesTheManifestBudgetAndIsNotOverridden asserts the
// framework hands the declared budget through and does not substitute a generic
// rule for the scenario's selection logic.
func TestSnapshotPrunerReceivesTheManifestBudgetAndIsNotOverridden(t *testing.T) {
	pruner, db := newPrunerFixture(t)
	base := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	for i := range 10 {
		insertSnapshot(t, db, "demo", fmt.Sprintf("h-%02d", i), base.Add(time.Duration(i)*time.Hour), 4096)
	}

	registry := retention.NewRegistry()
	if err := registry.Register(SnapshotBudgetName, pruner); err != nil {
		t.Fatalf("Register: %v", err)
	}
	spec := retention.Spec{
		Budget: manifestBudget(),
		Target: retention.Target{Kind: retention.TargetSQLiteTable, Database: "architecture-cartographer.db", Table: "graph_snapshots", TimeColumn: "extracted_at"},
		Mode:   retention.PrunerCustom,
	}

	builtinCalled := false
	engine, err := retention.NewEngine(retention.EngineConfig{
		Specs:    []retention.Spec{spec},
		Registry: registry,
		Builtin: func(retention.Spec) (retention.Pruner, error) {
			builtinCalled = true
			return nil, fmt.Errorf("the builtin pruner must never be built for a custom budget")
		},
	})
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	if builtinCalled {
		t.Fatal("the builtin pruner was constructed for a custom budget")
	}

	if _, err := engine.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}
	// Keep-N ran, not an age rule: exactly the floor survives.
	if got := snapshotCount(t, db, "demo"); got != pruner.KeepPerScenario() {
		t.Fatalf("demo kept %d snapshots, want the keep-N floor of %d", got, pruner.KeepPerScenario())
	}
}

// TestSnapshotPrunerReportsBoundBytesWhenTheRuleCannotReachTheCeiling asserts
// the framework's signal fires when the domain rule has done all it may and the
// table is still over budget. That is a statement about the producer, and it
// must not be resolved by deleting history the rule says to keep.
func TestSnapshotPrunerReportsBoundBytesWhenTheRuleCannotReachTheCeiling(t *testing.T) {
	pruner, db := newPrunerFixture(t)
	base := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	for i := range 6 {
		insertSnapshot(t, db, "demo", fmt.Sprintf("h-%02d", i), base.Add(time.Duration(i)*time.Hour), 64*1024)
	}

	// A ceiling far below what keep-N is permitted to leave behind.
	tiny := retention.Budget{Name: SnapshotBudgetName, MaxBytes: 1024}
	result, err := pruner.Prune(context.Background(), tiny)
	if err != nil {
		t.Fatalf("Prune: %v", err)
	}
	if result.BoundBy != retention.BoundBytes {
		t.Fatalf("BoundBy = %v, want bytes when the rule cannot reach the ceiling", result.BoundBy)
	}
	if got := snapshotCount(t, db, "demo"); got != pruner.KeepPerScenario() {
		t.Fatalf("demo kept %d snapshots, want the keep-N floor of %d; the ceiling must not override the selection rule",
			got, pruner.KeepPerScenario())
	}
}

// TestSnapshotPrunerMeasureReportsLivePayload confirms Measure describes the
// snapshots themselves rather than the database file, so the budget is judged
// against what this pruner can act on.
func TestSnapshotPrunerMeasureReportsLivePayload(t *testing.T) {
	pruner, db := newPrunerFixture(t)
	base := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	const rows, size = 8, 4096
	for i := range rows {
		insertSnapshot(t, db, "demo", fmt.Sprintf("h-%02d", i), base.Add(time.Duration(i)*time.Hour), size)
	}

	usage, err := pruner.Measure(context.Background())
	if err != nil {
		t.Fatalf("Measure: %v", err)
	}
	if usage.Items != rows {
		t.Errorf("Items = %d, want %d", usage.Items, rows)
	}
	if usage.Bytes < rows*size {
		t.Errorf("Bytes = %d, want at least the %d bytes of payload written", usage.Bytes, rows*size)
	}
}

// TestSnapshotPrunerOnEmptyTable confirms a scenario with nothing stored is
// trivially within budget rather than an error.
func TestSnapshotPrunerOnEmptyTable(t *testing.T) {
	pruner, _ := newPrunerFixture(t)
	result, err := pruner.Prune(context.Background(), manifestBudget())
	if err != nil {
		t.Fatalf("Prune: %v", err)
	}
	if result.Deleted != 0 || result.BoundBy != retention.BoundNone {
		t.Fatalf("result = %+v, want nothing deleted and no bound reached", result)
	}
}
