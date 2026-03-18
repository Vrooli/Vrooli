package format

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatalf("failed to read pipe: %v", err)
	}
	return buf.String()
}

func TestMutationResult_WithNextSteps(t *testing.T) {
	output := captureStdout(t, func() {
		MutationResult("Created task: foo [abc-123]", "", []string{
			"ecosystem-manager task show abc-123",
			"ecosystem-manager queue start",
		})
	})

	if !strings.Contains(output, "Created task: foo [abc-123]") {
		t.Errorf("expected result line, got: %s", output)
	}
	if !strings.Contains(output, "Next steps:") {
		t.Errorf("expected next steps header, got: %s", output)
	}
	if !strings.Contains(output, "$ ecosystem-manager task show abc-123") {
		t.Errorf("expected first next step, got: %s", output)
	}
	if !strings.Contains(output, "$ ecosystem-manager queue start") {
		t.Errorf("expected second next step, got: %s", output)
	}
}

func TestMutationResult_NoNextSteps(t *testing.T) {
	output := captureStdout(t, func() {
		MutationResult("Task deleted: abc-123", "", nil)
	})

	if !strings.Contains(output, "Task deleted: abc-123") {
		t.Errorf("expected result line, got: %s", output)
	}
	if strings.Contains(output, "Next steps:") {
		t.Errorf("should not have next steps section, got: %s", output)
	}
}

func TestMutationResult_WithDetails(t *testing.T) {
	output := captureStdout(t, func() {
		MutationResult("Task updated successfully", "Status: pending", []string{
			"ecosystem-manager task show abc-123",
		})
	})

	if !strings.Contains(output, "  Status: pending") {
		t.Errorf("expected details line, got: %s", output)
	}
}
