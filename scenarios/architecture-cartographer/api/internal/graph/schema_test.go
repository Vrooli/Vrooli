package graph

import (
	"strings"
	"testing"
)

func TestSchema_NotEmpty(t *testing.T) {
	if Schema() == "" {
		t.Fatal("graph.Schema() returned empty; check go:embed wiring")
	}
}

func TestSchema_ContainsSnapshotsTable(t *testing.T) {
	if !strings.Contains(Schema(), "CREATE TABLE IF NOT EXISTS graph_snapshots") {
		t.Fatalf("graph.Schema() missing graph_snapshots table")
	}
}
