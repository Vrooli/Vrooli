package monetization

import (
	"path/filepath"
	"testing"
)

func TestBuildMeterInventoryAggregatesScenarioDeclarations(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	inventory, err := BuildMeterInventory(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(inventory.Meters) != 3 {
		t.Fatalf("meter count = %d, want 3", len(inventory.Meters))
	}
	if inventory.Meters[0].LimitKey != "ai_credits" || len(inventory.Meters[0].DeclaredBy) != 5 || !inventory.Meters[0].Byok {
		t.Fatalf("ai_credits summary = %+v", inventory.Meters[0])
	}
	if inventory.Meters[1].LimitKey != "voice_minutes" || len(inventory.Meters[1].DeclaredBy) != 1 {
		t.Fatalf("voice_minutes summary = %+v", inventory.Meters[1])
	}
	if inventory.Meters[2].LimitKey != "workflow_executions" || inventory.Meters[2].Class != "B" {
		t.Fatalf("workflow_executions summary = %+v", inventory.Meters[2])
	}
}
