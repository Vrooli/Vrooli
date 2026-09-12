package store

import (
	"database/sql"
	"testing"
	"time"

	"github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/interfacegraph"

	_ "modernc.org/sqlite"
)

func TestInterfaceGraphCacheRoundTrip(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()
	if _, err := db.Exec(Schema()); err != nil {
		t.Fatalf("apply schema: %v", err)
	}

	store := New(db)
	computedAt := time.Date(2026, 6, 15, 12, 34, 56, 789, time.UTC)
	entry := InterfaceGraphCacheEntry{
		Signature:  "fleet-signature",
		ComputedAt: computedAt,
		Graph: interfacegraph.Graph{
			Nodes: []interfacegraph.Node{{Scenario: "api"}, {Scenario: "proto-health"}},
			Edges: []interfacegraph.Edge{{
				FromScenario:   "api",
				ToScenario:     "proto-health",
				TransportWorld: "connect",
				Stability:      []string{"stable"},
				Evidence: []interfacegraph.Evidence{{
					Source: "go_import",
					Path:   "scenarios/api/client.go",
				}},
			}},
		},
	}

	if err := store.StoreInterfaceGraphCache(entry); err != nil {
		t.Fatalf("store cache: %v", err)
	}
	got, ok, err := store.LoadInterfaceGraphCache(entry.Signature)
	if err != nil {
		t.Fatalf("load cache: %v", err)
	}
	if !ok {
		t.Fatal("expected cache hit")
	}
	if !got.ComputedAt.Equal(computedAt) {
		t.Fatalf("computedAt = %s, want %s", got.ComputedAt, computedAt)
	}
	if len(got.Graph.Nodes) != 2 || got.Graph.Nodes[1].Scenario != "proto-health" {
		t.Fatalf("nodes = %#v", got.Graph.Nodes)
	}
	if len(got.Graph.Edges) != 1 || got.Graph.Edges[0].Evidence[0].Source != "go_import" {
		t.Fatalf("edges = %#v", got.Graph.Edges)
	}
}
