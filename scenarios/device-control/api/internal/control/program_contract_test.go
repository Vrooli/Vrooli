package control

import (
	"os/exec"
	"testing"
)

func TestGovernedWorkflowPrograms(t *testing.T) { // [REQ:DVC-AGENT-REUSE]
	command := exec.Command("python3", "../../../.vrooli/program-runtime/tests/test_workflows.py")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("program contract probes: %v\n%s", err, output)
	}
}
