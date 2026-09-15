package inference

import (
	"context"
	"os/exec"
	"testing"
	"time"
)

// The subprocess covers shipped Python programs through the production kernel.
// Use an uncached test execution when those external-to-module sources change:
// Go's result cache cannot track files read by this Python subprocess.
func TestClassificationProgramsPreserveValidatedEvidence(t *testing.T) { // [REQ:AIGW-CONTRACT-BATCH-PROGRAM]
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "python3", "../../../.vrooli/program-runtime/tests/test_classify_batch.py")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("classification program contract: %v\n%s", err, output)
	}
	t.Logf("production-kernel classification regressions:\n%s", output)
}
