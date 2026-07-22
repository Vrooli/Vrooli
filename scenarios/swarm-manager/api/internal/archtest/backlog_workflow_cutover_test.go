package archtest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLegacyBacklogWorkshopImplementationsAreDeleted(t *testing.T) {
	root := swarmScenarioRoot(t)
	for _, name := range []string{"research.go", "workshop_save.go", "workshop_workflow.go", "clarification.go", "clarification_state.go", "clarification_workflow.go"} {
		if _, err := os.Stat(filepath.Join(root, "api", "internal", "backlog", name)); !os.IsNotExist(err) {
			t.Fatalf("legacy implementation %s remains", name)
		}
	}
}
