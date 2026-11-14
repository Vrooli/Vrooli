package store

import (
	"database/sql"
	"testing"
	"time"

	types "github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/types"

	_ "modernc.org/sqlite"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(Schema()); err != nil {
		t.Fatalf("apply schema: %v", err)
	}
	return New(db)
}

func TestCleanupInvalidScenarioDependenciesDoesNotExhaustSingleConnectionPool(t *testing.T) {
	db, err := sql.Open("sqlite", "file:cleanup-single-connection?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(Schema()); err != nil {
		t.Fatalf("apply schema: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO scenario_dependencies (scenario_name, dependency_type, dependency_name, required)
		VALUES ('consumer', 'scenario', 'orphaned-scenario', 1)`); err != nil {
		t.Fatalf("seed dependency: %v", err)
	}

	done := make(chan error, 1)
	go func() {
		done <- New(db).CleanupInvalidScenarioDependencies(map[string]struct{}{"known-scenario": {}})
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("cleanup dependencies: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("cleanup blocked while a single-connection SQLite pool still had query rows open")
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM scenario_dependencies WHERE dependency_name = 'orphaned-scenario'`).Scan(&count); err != nil {
		t.Fatalf("count orphaned dependencies: %v", err)
	}
	if count != 0 {
		t.Fatalf("orphaned dependency was not removed; count=%d", count)
	}
}

func TestGraphEdgesReplaceLoadRoundTrip(t *testing.T) {
	s := newTestStore(t)
	now := time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC)
	edges := []types.UnifiedGraphEdge{
		{
			From: "consumer", To: "core-a", Kind: "scenario", Source: "proto_import", Confidence: 1.0, Required: true, LastVerified: now,
			Evidence: []types.UnifiedEdgeEvidence{{Source: "proto_import", FromFile: "x.proto"}},
		},
		{From: "consumer", To: "postgres", Kind: "resource", Source: "resource", Confidence: 0.8, LastVerified: now},
	}
	if err := s.ReplaceGraphEdges(edges); err != nil {
		t.Fatalf("ReplaceGraphEdges: %v", err)
	}
	loaded, err := s.LoadGraphEdges()
	if err != nil {
		t.Fatalf("LoadGraphEdges: %v", err)
	}
	if len(loaded) != 2 {
		t.Fatalf("loaded %d edges, want 2", len(loaded))
	}
	if loaded[0].To != "core-a" || loaded[0].Source != "proto_import" || !loaded[0].Required {
		t.Fatalf("unexpected first edge %+v", loaded[0])
	}
	if len(loaded[0].Evidence) != 1 || loaded[0].Evidence[0].FromFile != "x.proto" {
		t.Fatalf("evidence not round-tripped: %+v", loaded[0].Evidence)
	}
}

func TestReplaceGraphEdgesIsIdempotent(t *testing.T) {
	s := newTestStore(t)
	edges := []types.UnifiedGraphEdge{{From: "a", To: "b", Kind: "scenario", Source: "declared", Confidence: 0.7}}
	for i := 0; i < 3; i++ {
		if err := s.ReplaceGraphEdges(edges); err != nil {
			t.Fatalf("ReplaceGraphEdges: %v", err)
		}
	}
	loaded, _ := s.LoadGraphEdges()
	if len(loaded) != 1 {
		t.Fatalf("expected idempotent replace to keep 1 edge, got %d", len(loaded))
	}
}

func TestUpsertGraphEdgesForScenarioReplacesOnlyThatScenario(t *testing.T) {
	s := newTestStore(t)
	if err := s.ReplaceGraphEdges([]types.UnifiedGraphEdge{
		{From: "a", To: "x", Kind: "scenario", Source: "declared", Confidence: 0.7},
		{From: "b", To: "y", Kind: "scenario", Source: "declared", Confidence: 0.7},
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if err := s.UpsertGraphEdgesForScenario("a", []types.UnifiedGraphEdge{
		{From: "a", To: "z", Kind: "scenario", Source: "proto_import", Confidence: 1.0},
	}); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	loaded, _ := s.LoadGraphEdges()
	if len(loaded) != 2 {
		t.Fatalf("expected 2 edges, got %d", len(loaded))
	}
	var aTargets, bTargets []string
	for _, e := range loaded {
		switch e.From {
		case "a":
			aTargets = append(aTargets, e.To)
		case "b":
			bTargets = append(bTargets, e.To)
		}
	}
	if len(aTargets) != 1 || aTargets[0] != "z" {
		t.Fatalf("a edges = %v, want [z]", aTargets)
	}
	if len(bTargets) != 1 || bTargets[0] != "y" {
		t.Fatalf("b edges should be untouched, got %v", bTargets)
	}
}

func TestMarkScenarioEdgesStale(t *testing.T) {
	s := newTestStore(t)
	_ = s.ReplaceGraphEdges([]types.UnifiedGraphEdge{{From: "a", To: "b", Kind: "scenario", Source: "declared"}})
	if err := s.MarkScenarioEdgesStale("a"); err != nil {
		t.Fatalf("mark stale: %v", err)
	}
	loaded, _ := s.LoadGraphEdges()
	if len(loaded) != 1 || !loaded[0].Stale {
		t.Fatalf("expected edge marked stale, got %+v", loaded)
	}
}

func TestIngestDigestRoundTrip(t *testing.T) {
	s := newTestStore(t)
	if _, ok, _ := s.GetIngestDigest("a"); ok {
		t.Fatalf("expected no digest initially")
	}
	if err := s.SetIngestDigest("a", "td:1"); err != nil {
		t.Fatalf("set digest: %v", err)
	}
	if err := s.SetIngestDigest("a", "td:2"); err != nil {
		t.Fatalf("update digest: %v", err)
	}
	got, ok, _ := s.GetIngestDigest("a")
	if !ok || got != "td:2" {
		t.Fatalf("digest = %q ok=%v, want td:2", got, ok)
	}
}

func TestGraphEdgeStats(t *testing.T) {
	s := newTestStore(t)
	_ = s.ReplaceGraphEdges([]types.UnifiedGraphEdge{
		{From: "a", To: "b", Kind: "scenario", Source: "proto_import"},
		{From: "a", To: "pg", Kind: "resource", Source: "resource"},
		{From: "c", To: "d", Kind: "scenario", Source: "declared", Stale: true},
	})
	stats, err := s.GraphEdgeStats()
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if stats.TotalEdges != 3 || stats.ScenarioEdges != 2 || stats.ResourceEdges != 1 {
		t.Fatalf("counts = total %d scenario %d resource %d", stats.TotalEdges, stats.ScenarioEdges, stats.ResourceEdges)
	}
	if stats.StaleEdges != 1 {
		t.Fatalf("stale = %d, want 1", stats.StaleEdges)
	}
	if stats.BySource["proto_import"] != 1 {
		t.Fatalf("by_source proto_import = %d, want 1", stats.BySource["proto_import"])
	}
}
