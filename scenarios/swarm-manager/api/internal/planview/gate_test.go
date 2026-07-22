package planview

import (
	"testing"

	"swarm-manager/internal/backlog"
)

func gateItem(name, status string) backlog.BacklogItem {
	return backlog.BacklogItem{Kind: backlog.KindExecute, Name: name, Status: backlog.BacklogStatus(status)}
}

func TestNewProjection_DerivesWavesFromItemDependencies(t *testing.T) {
	items := []backlog.BacklogItem{
		gateItem("a1", "ready"),
		{Name: "b1", Kind: backlog.KindExecute, Status: "completed"},
	}
	projection := newProjection(items, nil)
	if projection.waves.Waves["execute/a1"] != 0 || projection.waves.Waves["execute/b1"] != 0 {
		t.Fatalf("items with no item dependencies should be wave 0, got a1=%d b1=%d",
			projection.waves.Waves["execute/a1"], projection.waves.Waves["execute/b1"])
	}
}
