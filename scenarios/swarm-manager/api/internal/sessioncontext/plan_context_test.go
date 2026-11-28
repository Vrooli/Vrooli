package sessioncontext

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"swarm-manager/internal/agentsessions"
)

func TestResolveGoal(t *testing.T) {
	root := t.TempDir()
	goalDir := filepath.Join(root, "goals", "ship-goal")
	if err := os.MkdirAll(goalDir, 0o755); err != nil {
		t.Fatalf("mkdir goal: %v", err)
	}
	if err := os.WriteFile(filepath.Join(goalDir, "goal.json"), []byte(`{"name":"ship-goal","title":"Ship the feature","status":"active","priority":8,"updated":"2026-07-24T16:00:00Z"}`), 0o600); err != nil {
		t.Fatalf("write goal: %v", err)
	}

	r := NewResolver(root, "/tmp", nil)
	items, err := r.ResolveSessionMessageContext(context.Background(), []agentsessions.ContextRef{{Type: agentsessions.ContextGoal, Ref: "ship-goal"}}, agentsessions.ContextLimits{MaxSummaryRunes: 2000})
	if err != nil {
		t.Fatalf("resolve goal: %v", err)
	}
	if len(items) != 1 || items[0].Title != "Ship the feature" || items[0].NodeID != "goal/ship-goal" {
		t.Fatalf("resolved goal = %+v", items)
	}
	if !strings.Contains(items[0].MetadataJSON, `"base_version":"2026-07-24T16:00:00Z"`) {
		t.Fatalf("goal metadata omits proposal base version: %s", items[0].MetadataJSON)
	}
}

func TestResolvePlanDependencyCycles(t *testing.T) {
	r := NewResolver(t.TempDir(), "/tmp", nil)
	limits := agentsessions.ContextLimits{MaxSummaryRunes: 2000}
	ref := `["backlog-item/fix/a -> backlog-item/fix/b","initiative/x -> initiative/y"]`

	items, err := r.ResolveSessionMessageContext(context.Background(), []agentsessions.ContextRef{
		{Type: agentsessions.ContextPlanDependencyCycles, Ref: ref},
	}, limits)
	if err != nil {
		t.Fatalf("resolve cycles: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	item := items[0]
	if item.Type != agentsessions.ContextPlanDependencyCycles {
		t.Fatalf("unexpected type %q", item.Type)
	}
	if item.Title != "Dependency cycles (2)" {
		t.Fatalf("unexpected title %q", item.Title)
	}
	// The summary must carry the actual chains for the agent to reason about.
	if !strings.Contains(item.Summary, "backlog-item/fix/a -> backlog-item/fix/b") ||
		!strings.Contains(item.Summary, "initiative/x -> initiative/y") {
		t.Fatalf("summary missing cycle chains: %q", item.Summary)
	}
}

func TestResolvePlanEta(t *testing.T) {
	r := NewResolver(t.TempDir(), "/tmp", nil)
	limits := agentsessions.ContextLimits{MaxSummaryRunes: 2000}
	ref := `{"p50Label":"Fri 5pm","p80Label":"Mon 9am","basisLabel":"27 samples","confidence":"high","remainingItems":12,"laneCapacity":3}`

	items, err := r.ResolveSessionMessageContext(context.Background(), []agentsessions.ContextRef{
		{Type: agentsessions.ContextPlanEta, Ref: ref},
	}, limits)
	if err != nil {
		t.Fatalf("resolve eta: %v", err)
	}
	item := items[0]
	if item.Title != "Plan ETA Fri 5pm–Mon 9am" {
		t.Fatalf("unexpected title %q", item.Title)
	}
	for _, want := range []string{"p50 Fri 5pm", "p80 Mon 9am", "12 items", "3 execute lanes", "high", "27 samples"} {
		if !strings.Contains(item.Summary, want) {
			t.Fatalf("summary %q missing %q", item.Summary, want)
		}
	}
}

func TestResolvePlanContextRejectsMalformedRef(t *testing.T) {
	r := NewResolver(t.TempDir(), "/tmp", nil)
	limits := agentsessions.ContextLimits{MaxSummaryRunes: 2000}

	for _, ref := range []agentsessions.ContextRef{
		{Type: agentsessions.ContextPlanDependencyCycles, Ref: "not-json"},
		{Type: agentsessions.ContextPlanDependencyCycles, Ref: "[]"},
		{Type: agentsessions.ContextPlanEta, Ref: "{}"},
	} {
		if _, err := r.ResolveSessionMessageContext(context.Background(), []agentsessions.ContextRef{ref}, limits); err == nil {
			t.Fatalf("expected error for ref %q", ref.Ref)
		}
	}
}
