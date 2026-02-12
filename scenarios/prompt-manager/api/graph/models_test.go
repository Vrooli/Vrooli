package graph

import (
	"testing"
	"time"
)

func TestNewGraphIndex(t *testing.T) {
	g := Graph{
		Nodes: []Node{{ID: "test", Type: NodeSkill, Label: "Test"}},
	}

	idx := NewGraphIndex(g)
	if idx.GeneratedAt == "" {
		t.Error("expected GeneratedAt to be set")
	}

	// Verify it's a valid RFC3339 timestamp
	if _, err := time.Parse(time.RFC3339, idx.GeneratedAt); err != nil {
		t.Errorf("expected valid RFC3339 timestamp, got %s: %v", idx.GeneratedAt, err)
	}

	if len(idx.Graph.Nodes) != 1 {
		t.Errorf("expected 1 node, got %d", len(idx.Graph.Nodes))
	}
}
