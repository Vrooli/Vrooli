package telemetry

import (
	telemetryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/telemetry"
	"testing"
)

func TestTypedEventsRoundTrip(t *testing.T) {
	s := NewStore()
	s.Append(&telemetryv1.ProgramEvent{SessionId: "s1", Kind: telemetryv1.EventKind_PROGRAM_FAILED})
	if len(s.List("s1", telemetryv1.EventKind_EVENT_KIND_UNSPECIFIED)) != 1 {
		t.Fatal("event was not retained")
	}
}
