package runreport

import (
	"agent-manager/internal/domain"
	"github.com/google/uuid"
	"testing"
)

func TestDeriveInvocationFactsPairsAndRedacts(t *testing.T) {
	runID := uuid.New()
	call := domain.NewToolCallEvent(runID, "shell", "call-1", map[string]any{"command": "external-tool --token=top-secret"})
	result := domain.NewToolResultEvent(runID, "shell", "call-1", "", assertErr{})
	facts := DeriveInvocationFacts([]*domain.RunEvent{call, result})
	if len(facts) != 1 || facts[0].Outcome != "failure" || facts[0].ResultEventID != result.ID.String() {
		t.Fatalf("facts=%+v", facts)
	}
	if facts[0].Ownership != "external" || facts[0].Fingerprint == "" {
		t.Fatalf("fact=%+v", facts[0])
	}
}

func TestDeriveInvocationFactsMarksCompoundShellUnknown(t *testing.T) {
	runID := uuid.New()
	facts := DeriveInvocationFacts([]*domain.RunEvent{domain.NewToolCallEvent(runID, "shell", "call-1", map[string]any{"command": "vrooli help; echo no"})})
	if facts[0].Ownership != "unknown" || facts[0].Availability != "available" {
		t.Fatalf("fact=%+v", facts[0])
	}
}

func TestDeriveInvocationFactsDetectsHelpRecovery(t *testing.T) {
	runID := uuid.New()
	failed := domain.NewToolCallEvent(runID, "shell", "one", map[string]any{"command": "agent-manager"})
	failedResult := domain.NewToolResultEvent(runID, "shell", "one", "", assertErr{})
	help := domain.NewToolCallEvent(runID, "shell", "two", map[string]any{"command": "agent-manager --help"})
	retry := domain.NewToolCallEvent(runID, "shell", "three", map[string]any{"command": "agent-manager"})
	facts := DeriveInvocationFacts([]*domain.RunEvent{failed, failedResult, help, retry})
	if !facts[2].HelpRecovery || facts[2].RetryOfCallEventID != failed.ID.String() {
		t.Fatalf("facts=%+v", facts)
	}
}
