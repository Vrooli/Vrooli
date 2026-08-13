package runhistory

import (
	"context"
	"database/sql"
	"testing"
	"time"

	uhdb "github.com/vrooli/api-core/databasetest"
)

func newRepo(t *testing.T, retention int) (*Repository, *sql.DB) {
	t.Helper()
	db := uhdb.NewSQLite(t)
	if _, err := db.Exec(Schema()); err != nil {
		t.Fatalf("apply schema: %v", err)
	}
	r := NewRepository(db)
	r.retention = retention
	return r, db
}

func sampleRun(id string, at time.Time, cmdStatus string, dur int64) RunRecord {
	return RunRecord{
		RunID: id, Scenario: "demo", StartedAt: at, Status: "passed", MaturityRung: 3,
		Commands: []CommandSample{{
			RunID: id, StartedAt: at, WorkspaceID: "api", Command: "go test ./...",
			DurationMS: dur, Status: cmdStatus,
		}},
		Coverage: []CoverageSample{{WorkspaceID: "api", File: "a.go", Percent: 80}},
	}
}

func TestRecordAndCommandHistory(t *testing.T) {
	r, _ := newRepo(t, DefaultRetention)
	ctx := context.Background()
	base := time.Unix(1_700_000_000, 0).UTC()

	if err := r.Record(ctx, sampleRun("r1", base, "passed", 3000)); err != nil {
		t.Fatalf("record r1: %v", err)
	}
	if err := r.Record(ctx, sampleRun("r2", base.Add(time.Minute), "failed", 3200)); err != nil {
		t.Fatalf("record r2: %v", err)
	}

	hist, err := r.CommandHistory(ctx, "demo", 10)
	if err != nil {
		t.Fatalf("history: %v", err)
	}
	if len(hist) != 2 {
		t.Fatalf("expected 2 command samples, got %d: %+v", len(hist), hist)
	}
	// Newest first.
	if hist[0].RunID != "r2" || hist[0].Status != "failed" {
		t.Errorf("expected r2 first, got %+v", hist[0])
	}
	if hist[1].DurationMS != 3000 {
		t.Errorf("expected r1 duration 3000, got %d", hist[1].DurationMS)
	}
}

func TestRetentionKeepsNewest(t *testing.T) {
	r, db := newRepo(t, 2)
	ctx := context.Background()
	base := time.Unix(1_700_000_000, 0).UTC()
	for i, id := range []string{"r1", "r2", "r3", "r4"} {
		if err := r.Record(ctx, sampleRun(id, base.Add(time.Duration(i)*time.Minute), "passed", 1000)); err != nil {
			t.Fatalf("record %s: %v", id, err)
		}
	}

	var runs int
	if err := db.QueryRow(`SELECT COUNT(*) FROM unit_runs WHERE scenario='demo'`).Scan(&runs); err != nil {
		t.Fatal(err)
	}
	if runs != 2 {
		t.Errorf("retention=2 should keep 2 runs, got %d", runs)
	}
	// Child rows of pruned runs are gone too.
	var cmds int
	if err := db.QueryRow(`SELECT COUNT(*) FROM unit_run_commands WHERE scenario='demo'`).Scan(&cmds); err != nil {
		t.Fatal(err)
	}
	if cmds != 2 {
		t.Errorf("pruned runs' commands should be deleted, got %d", cmds)
	}

	hist, err := r.CommandHistory(ctx, "demo", 10)
	if err != nil {
		t.Fatal(err)
	}
	for _, h := range hist {
		if h.RunID == "r1" || h.RunID == "r2" {
			t.Errorf("run %s should have been pruned, history=%+v", h.RunID, hist)
		}
	}
}

func TestRecordRequiresIDAndScenario(t *testing.T) {
	r, _ := newRepo(t, DefaultRetention)
	if err := r.Record(context.Background(), RunRecord{Scenario: "demo"}); err == nil {
		t.Error("expected error for empty run_id")
	}
}

func TestNilRepoIsNoOp(t *testing.T) {
	var r *Repository
	if err := r.Record(context.Background(), RunRecord{RunID: "x", Scenario: "y"}); err != nil {
		t.Errorf("nil repo Record should be a no-op, got %v", err)
	}
	if got, err := r.CommandHistory(context.Background(), "demo", 5); err != nil || got != nil {
		t.Errorf("nil repo history should be (nil,nil), got %v,%v", got, err)
	}
}
