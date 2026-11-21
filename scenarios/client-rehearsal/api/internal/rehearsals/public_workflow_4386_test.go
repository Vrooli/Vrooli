package rehearsals

import (
	"strings"
	"testing"
)

func TestPublic4386ImportAdvancesPastLegacyEmptySetsAndFailsClosedOnEmptySwitch(t *testing.T) {
	workflowJSON := readFixture(t, "public-workflows/4386.json")
	_, _, _, err := normalizeWorkflow(workflowJSON)
	if err == nil {
		t.Fatal("expected the complete sanitized public graph to remain unqualified at its enabled empty Switch node")
	}
	if !strings.Contains(err.Error(), "nodes.parameters.rules: Switch requires 1 to 32 rules") {
		t.Fatalf("expected generic empty legacy Set support then fail-closed at the unsupported empty Switch, got %v", err)
	}
}
