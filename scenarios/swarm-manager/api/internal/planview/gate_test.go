package planview

import (
	"testing"

	"swarm-manager/internal/backlog"
	"swarm-manager/internal/initiatives"
)

func gateItem(name, status string) backlog.BacklogItem {
	return backlog.BacklogItem{Kind: backlog.KindExecute, Name: name, Status: backlog.BacklogStatus(status)}
}

// TestNewProjection_InitiativeGateFoldsIntoWaves verifies D2: an item in
// initiative A (which depends on initiative B) is pushed past wave 0 until B's
// items complete, and returns to wave 0 once B is done.
func TestNewProjection_InitiativeGateFoldsIntoWaves(t *testing.T) {
	items := []backlog.BacklogItem{
		gateItem("a1", "ready"),
		gateItem("b1", "ready"),
	}
	inits := []initiatives.Initiative{
		{Name: "A", Items: []string{"execute/a1"}, DependsOn: []string{"B"}},
		{Name: "B", Items: []string{"execute/b1"}},
	}

	// Baseline (no gate): both items are wave 0.
	base := newProjection(items, nil, nil)
	if base.waves.Waves["execute/a1"] != 0 || base.waves.Waves["execute/b1"] != 0 {
		t.Fatalf("without gate both should be wave 0, got a1=%d b1=%d",
			base.waves.Waves["execute/a1"], base.waves.Waves["execute/b1"])
	}

	// With gate, B incomplete: a1 is pushed to a later wave than b1.
	gated := newProjection(items, nil, inits)
	if gated.waves.Waves["execute/a1"] <= gated.waves.Waves["execute/b1"] {
		t.Fatalf("gate should push a1 past b1: a1=%d b1=%d",
			gated.waves.Waves["execute/a1"], gated.waves.Waves["execute/b1"])
	}

	// Once B completes, a1 returns to wave 0.
	itemsDone := []backlog.BacklogItem{
		gateItem("a1", "ready"),
		gateItem("b1", "completed"),
	}
	unblocked := newProjection(itemsDone, nil, inits)
	if unblocked.waves.Waves["execute/a1"] != 0 {
		t.Fatalf("a1 should be wave 0 once B complete, got %d", unblocked.waves.Waves["execute/a1"])
	}
}
