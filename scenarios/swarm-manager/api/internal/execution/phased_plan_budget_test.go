package execution

import (
	"encoding/json"
	"os"
	"testing"
)

// TestPhasedPlanDrainDeclarationBudgetsAffordEverySlice guards the workflow
// declaration against the legacy 144-turn cliff. The declaration budget is the
// ungranted fallback; a granted execution's effective budget is the reviewed
// engagement grant. It must still afford the default drain (six slices) at the
// worker node's own turn and wall limits, so the declaration is never the
// binding ceiling for a normal run.
func TestPhasedPlanDrainDeclarationBudgetsAffordEverySlice(t *testing.T) {
	raw, err := os.ReadFile("../../../.vrooli/agent-manager/phased-plan-drain.json")
	if err != nil {
		t.Fatalf("read workflow declaration: %v", err)
	}
	var declaration struct {
		Nodes []struct {
			Kind string `json:"kind"`
			Run  struct {
				MaxTurns       int `json:"maxTurns"`
				TimeoutSeconds int `json:"timeoutSeconds"`
			} `json:"run"`
		} `json:"nodes"`
		Budgets struct {
			MaxTurns        int `json:"maxTurns"`
			WallTimeSeconds int `json:"wallTimeSeconds"`
		} `json:"budgets"`
	}
	if err := json.Unmarshal(raw, &declaration); err != nil {
		t.Fatalf("parse workflow declaration: %v", err)
	}
	workerTurns, workerWall := 0, 0
	for _, node := range declaration.Nodes {
		if node.Kind != "run" {
			continue
		}
		if node.Run.MaxTurns > workerTurns {
			workerTurns = node.Run.MaxTurns
		}
		if node.Run.TimeoutSeconds > workerWall {
			workerWall = node.Run.TimeoutSeconds
		}
	}
	if workerTurns == 0 || workerWall == 0 {
		t.Fatalf("declaration has no run node limits: turns=%d wall=%d", workerTurns, workerWall)
	}
	// The default drain is six slices (firstPositive(record.MaxSlices, 6)); the
	// declaration must afford every one of them at the worker's full limits.
	const defaultSlices = 6
	if declaration.Budgets.MaxTurns < defaultSlices*workerTurns {
		t.Fatalf("declaration maxTurns=%d cannot afford %d slices x %d worker turns", declaration.Budgets.MaxTurns, defaultSlices, workerTurns)
	}
	if declaration.Budgets.WallTimeSeconds < defaultSlices*workerWall {
		t.Fatalf("declaration wallTimeSeconds=%d cannot afford %d slices x %d worker seconds", declaration.Budgets.WallTimeSeconds, defaultSlices, workerWall)
	}
	if declaration.Budgets.MaxTurns == 144 {
		t.Fatalf("declaration still carries the legacy 144-turn cliff")
	}
}
