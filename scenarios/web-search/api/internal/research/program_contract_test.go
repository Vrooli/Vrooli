package research_test

import (
	"os/exec"
	"testing"
)

// [REQ:REQ-P0-009] [REQ:REQ-P0-010] [REQ:REQ-P0-011] [REQ:REQ-P0-012]
func TestResearchPrograms(t *testing.T) {
	out, err := exec.Command("python3", "../../../.vrooli/program-runtime/tests/test_research.py").CombinedOutput()
	if err != nil {
		t.Fatalf("research contract probes: %v\n%s", err, out)
	}
}
