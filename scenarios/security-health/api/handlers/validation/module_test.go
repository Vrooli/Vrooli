package validation

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadControlPlaneErrorBudgetPreservesExplicitZero(t *testing.T) {
	dir := t.TempDir()
	configDir := filepath.Join(dir, ".vrooli")
	if err := os.Mkdir(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	raw := []byte(`{"phases":{"security":{"budgets":{"error_findings":0,"baseline_error_findings":0,"ratchet":true}}}}`)
	if err := os.WriteFile(filepath.Join(configDir, "testing.json"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	budget := loadControlPlaneErrorBudget(dir, nil)
	if !budget.Declared || budget.Limit != 0 || budget.Baseline != 0 || !budget.Ratchet {
		t.Fatalf("loaded budget = %+v", budget)
	}
}
