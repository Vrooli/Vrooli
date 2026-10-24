package handlers

import (
	"testing"

	"agent-manager/internal/domain"
)

func TestParseTaskStatusAcceptsNeutralCompleted(t *testing.T) {
	got, ok := parseTaskStatus("completed")
	if !ok || got != domain.TaskStatusCompleted {
		t.Fatalf("parseTaskStatus(completed) = %q, %v", got, ok)
	}
}
