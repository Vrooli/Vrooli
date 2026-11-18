package core

import (
	"testing"

	"agent-manager/internal/domain"
	"github.com/google/uuid"
)

func TestWithClaudeSourceRunIDAddsAuthoritativeRunIDOnlyForClaude(t *testing.T) {
	runID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	input := []string{"PATH=/bin", "VROOLI_ANTHROPIC_SOURCE_RUN_ID=caller-value", "HOME=/tmp"}
	got := withClaudeSourceRunID(input, domain.RunnerTypeClaudeCode, runID)
	if len(got) != 3 || got[1] != "HOME=/tmp" || got[2] != "VROOLI_ANTHROPIC_SOURCE_RUN_ID="+runID.String() {
		t.Fatalf("managed Claude environment = %#v", got)
	}
	other := withClaudeSourceRunID(input, domain.RunnerTypeCodex, runID)
	if len(other) != len(input) || other[1] != input[1] {
		t.Fatalf("non-Claude environment was changed: %#v", other)
	}
}
