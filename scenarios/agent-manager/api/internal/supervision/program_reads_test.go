package supervision

import (
	"os/exec"
	"path/filepath"
	"testing"
)

func TestOptionalSupervisionReadFailuresPreserveAvailableEvidence(t *testing.T) {
	caseProgram, err := filepath.Abs("../../../.vrooli/program-runtime/supervision-case-read.py")
	if err != nil {
		t.Fatal(err)
	}
	experimentProgram, err := filepath.Abs("../../../.vrooli/program-runtime/supervision-experiment-read.py")
	if err != nil {
		t.Fatal(err)
	}
	harness, err := filepath.Abs("testdata/supervision_optional_reads.py")
	if err != nil {
		t.Fatal(err)
	}
	kernel, err := filepath.Abs("../../../../program-runtime/kernel")
	if err != nil {
		t.Fatal(err)
	}
	command := exec.Command("python3", harness, caseProgram, experimentProgram, kernel)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("optional-read behavior failed: %v\n%s", err, output)
	}
}
