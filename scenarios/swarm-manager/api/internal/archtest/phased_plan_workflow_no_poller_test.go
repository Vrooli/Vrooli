package archtest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecutionHousekeepingDoesNotApplyWorkflowResults(t *testing.T) {
	path := filepath.Join("..", "execution", "polling.go")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "ApplyPhasedPlanWorkflow(ctx") {
		t.Fatal("execution housekeeping must not poll and apply Agent Manager workflow results")
	}
}
