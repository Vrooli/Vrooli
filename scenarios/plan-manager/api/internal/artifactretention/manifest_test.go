package artifactretention

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/api-core/retention"
)

// The plan-artifacts root is durable plan evidence. Its declaration must make
// storage-manager attribute it (alarm-only, agents denied) while leaving every
// deletion to this owner pruner: the class-root retention engine must never
// turn the external path into a builtin directory budget.
func TestServiceManifestDeclaresPlanArtifactsAsDurableOwnerPruned(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", ".vrooli", "service.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Storage struct {
			Entries map[string]struct {
				Path         string `json:"path"`
				Subpath      string `json:"subpath"`
				Class        string `json:"class"`
				Regenerable  *bool  `json:"regenerable"`
				AgentRemoval string `json:"agent_removal"`
				Reclaim      *struct {
					Pruner string `json:"pruner"`
				} `json:"reclaim"`
				Budget *struct {
					MaxBytes string `json:"max_bytes"`
					MaxAge   string `json:"max_age"`
				} `json:"budget"`
			} `json:"entries"`
		} `json:"storage"`
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatal(err)
	}
	entry, ok := manifest.Storage.Entries["plan_artifacts"]
	if !ok {
		t.Fatal("plan_artifacts storage entry missing")
	}
	if entry.Path != "$USER_HOME/.vrooli/plan-artifacts" || entry.Regenerable == nil || *entry.Regenerable ||
		entry.AgentRemoval != "deny" || entry.Reclaim == nil || entry.Reclaim.Pruner != "custom" ||
		entry.Budget == nil || entry.Budget.MaxBytes == "" || entry.Budget.MaxAge != "" {
		t.Fatalf("plan_artifacts must be durable, agent-denied, owner-pruned and byte-alarmed only: %+v", entry)
	}
	receipt, ok := manifest.Storage.Entries["plan_artifacts_retention_receipt"]
	if !ok || receipt.Class != "state" || receipt.Subpath != "retention/isolated-tree-receipt.json" {
		t.Fatalf("receipt entry must declare the path the owner writes: %+v", receipt)
	}
	specs, err := retention.ParseManifest(raw)
	if err != nil {
		t.Fatal(err)
	}
	for _, spec := range specs {
		if spec.Budget.Name == "plan_artifacts" {
			t.Fatalf("class-root retention engine must not own plan_artifacts: %+v", spec)
		}
	}
}
