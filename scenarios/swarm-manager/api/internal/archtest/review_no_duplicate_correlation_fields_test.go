package archtest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestReviewDoesNotPersistTransitionCorrelationFields protects the single
// lifecycle-owner boundary. Review references a transition by subject; the
// transitionrun journal owns execution, digest, version, and apply state.
func TestReviewDoesNotPersistTransitionCorrelationFields(t *testing.T) {
	assertNoCorrelationFields(t, filepath.Join("..", "review", "types.go"), "review")
	assertNoCorrelationFields(t, filepath.Join("..", "execution", "model.go"), "execution")
}

func assertNoCorrelationFields(t *testing.T, path, domain string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s types: %v", domain, err)
	}
	for _, field := range []string{
		"agent_workflow_execution_id",
		"agent_workflow_key",
		"agent_workflow_definition_digest",
		"agent_workflow_entity_version",
		"agent_workflow_frontier_digest",
		"agent_workflow_apply_state",
		"agent_workflow_applied_at",
		"agent_workflow_outcome",
		"agent_workflow_terminal_code",
		"agent_workflow_result",
		"agent_workflow_attempts",
		"agent_workflow_approval_at",
		"agent_workflow_approval_by",
	} {
		if strings.Contains(string(data), field) {
			t.Fatalf("%s must not persist duplicate transition correlation field %q", domain, field)
		}
	}
}
