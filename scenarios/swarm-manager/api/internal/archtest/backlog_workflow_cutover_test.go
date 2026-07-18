package archtest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMigratedBacklogEntryPointsDoNotInvokeLegacyOperations(t *testing.T) {
	root := swarmScenarioRoot(t)
	for _, name := range []string{"research.go", "handler_create.go", "workshop_save.go", "clarification.go", "clarification_state.go"} {
		data, err := os.ReadFile(filepath.Join(root, "api", "internal", "backlog", name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if strings.Contains(string(data), "invokeItemOperation(") {
			t.Fatalf("%s reintroduced a legacy operation invocation; use a declared workflow adapter", name)
		}
	}
}
