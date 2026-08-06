package telemetry

import (
	"context"
	"errors"
	"testing"

	programsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/programs"
	telemetryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/telemetry"
	"program-runtime/internal/programs"
)

type eventRunner struct{}

func (eventRunner) Execute(context.Context, string, string) (programs.Result, error) {
	return programs.Result{Invocations: []programs.Invocation{{BindingID: "test-genie/runs/list", Effect: "read"}}}, errors.New("line 7: invalid field")
}

func TestEmitsSubmissionInvocationAndFailureEvents(t *testing.T) { // [REQ:PRT-P0-006]
	store := NewStore()
	service := programs.NewService(programs.Options{Runner: eventRunner{}, Events: store})
	program, err := service.Submit(context.Background(), "session-1", "raise ValueError()", programsv1.Provenance_PROVENANCE_AGENT, false)
	if err != nil {
		t.Fatal(err)
	}
	events := store.List(program.GetSessionId(), telemetryv1.EventKind_EVENT_KIND_UNSPECIFIED)
	if len(events) != 3 {
		t.Fatalf("events=%d, want submission, invocation, and failure", len(events))
	}
	seen := map[telemetryv1.EventKind]bool{}
	for _, event := range events {
		seen[event.GetKind()] = true
		if event.GetProgramId() != program.GetId() {
			t.Fatalf("event program=%q, want %q", event.GetProgramId(), program.GetId())
		}
	}
	for _, kind := range []telemetryv1.EventKind{telemetryv1.EventKind_PROGRAM_SUBMITTED, telemetryv1.EventKind_BINDING_INVOKED, telemetryv1.EventKind_PROGRAM_FAILED} {
		if !seen[kind] {
			t.Fatalf("missing event kind %v", kind)
		}
	}
}

func TestFailureEventCarriesProgramLocator(t *testing.T) { // [REQ:PRT-P0-006]
	store := NewStore()
	service := programs.NewService(programs.Options{Runner: eventRunner{}, Events: store})
	program, _ := service.Submit(context.Background(), "session-1", "bad", programsv1.Provenance_PROVENANCE_AGENT, false)
	events := store.List(program.GetSessionId(), telemetryv1.EventKind_PROGRAM_FAILED)
	if len(events) != 1 || events[0].GetProgramId() != program.GetId() || events[0].GetFailureLocation() != "line 7" {
		t.Fatalf("failure event=%v", events)
	}
}

func TestNoScenarioLocalAnalysisStack(t *testing.T) { // [REQ:PRT-P0-006]
	store := NewStore()
	store.Append(&telemetryv1.ProgramEvent{Kind: telemetryv1.EventKind_PROGRAM_FAILED, FailureShape: "line 1"})
	store.Append(&telemetryv1.ProgramEvent{Kind: telemetryv1.EventKind_PROGRAM_FAILED, FailureShape: "line 1"})
	if got := store.List("", telemetryv1.EventKind_PROGRAM_FAILED); len(got) != 2 {
		t.Fatalf("telemetry store transformed facts instead of retaining events: %d", len(got))
	}
}
