package supervision

// [REQ:REQ-P2-010]

import (
	"os/exec"
	"path/filepath"
	"testing"
)

// Execute the actual Python decision program with fake governed bindings. The
// corpus proves decisions and cursor behavior rather than only an envelope status.
func TestSupervisionProgramBehavior(t *testing.T) {
	source, err := filepath.Abs("../../../.vrooli/program-runtime/supervision-evaluate.py")
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("python3", "testdata/supervision_behavior.py", source)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("supervision behavior: %v\n%s", err, out)
	}
}
