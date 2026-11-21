package invocationreadmodel

import (
	"reflect"
	"testing"

	"agent-manager/internal/domain"
	"agent-manager/internal/runsignal"
)

func TestDeriveRunSubjectUsesRecordedInvocationDimensions(t *testing.T) {
	facts := []Fact{
		{InvocationFact: runsignal.InvocationFact{Executable: "agent-manager", CommandPath: "agent-manager run list"}},
		{InvocationFact: runsignal.InvocationFact{Executable: "agent-manager", CommandPath: "agent-manager run list"}},
		{InvocationFact: runsignal.InvocationFact{Executable: "", CommandPath: ""}},
	}
	want := []string{"tool:agent-manager", "tool:agent-manager run list"}
	if got := DeriveRunSubject(facts, nil); !reflect.DeepEqual(got, want) {
		t.Fatalf("subject=%v, want %v", got, want)
	}
}

// "Which tools ran" and "what was worked on" are different questions. The
// subject must answer both without collapsing them into one list, so a reader
// can tell a run that touched agent-manager from one that merely ran its CLI.
func TestDeriveRunSubjectSeparatesToolsFromProjectAreas(t *testing.T) {
	facts := []Fact{{InvocationFact: runsignal.InvocationFact{Executable: "bash", CommandPath: "bash"}}}
	events := []*domain.RunEvent{
		toolCallEvent(t, map[string]any{"command": "sed -n '1,40p' scenarios/agent-manager/api/internal/durability/projector.go"}),
		toolCallEvent(t, map[string]any{"command": "grep -rn Provenance packages/api-core/provenance/provenance.go docs/README.md"}),
	}

	got := DeriveRunSubject(facts, events)
	want := []string{
		"path:docs/README.md",
		"path:packages/api-core",
		"path:scenarios/agent-manager",
		"tool:bash",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("subject=%v, want %v", got, want)
	}
}

// Only tool calls name paths. Log lines quoting a path are narration, not work.
func TestDeriveRunAreasIgnoresNonToolCallEvents(t *testing.T) {
	events := []*domain.RunEvent{{EventType: domain.EventTypeLog, Data: &domain.LogEventData{Message: "see scenarios/web-console/README.md"}}}
	if got := DeriveRunAreas(events); len(got) != 0 {
		t.Fatalf("areas=%v, want none", got)
	}
}

func toolCallEvent(t *testing.T, input map[string]any) *domain.RunEvent {
	t.Helper()
	return &domain.RunEvent{EventType: domain.EventTypeToolCall, Data: &domain.ToolCallEventData{ToolName: "bash", Input: input}}
}
