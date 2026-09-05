package families_test

import (
	"os/exec"
	"path/filepath"
	"testing"
)

func TestExecutionBoardOwnerJoins(t *testing.T) {
	source, err := filepath.Abs("../../../.vrooli/program-runtime/execution-board.py")
	if err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("python3", "testdata/execution_board_behavior.py", source).CombinedOutput(); err != nil {
		t.Fatalf("execution board: %v\n%s", err, out)
	}
}
