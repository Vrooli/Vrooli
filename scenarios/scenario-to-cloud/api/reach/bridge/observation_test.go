package bridge

import (
	"context"
	"testing"

	"scenario-to-cloud/reach"
)

// [REQ:STC-P0-024] The Bridge relay runs scenario verbs only: a host
// observation program is a typed protocol refusal with zero relay or
// dispatch calls, never a fallback to another transport.
func TestObservationProgramIsRefusedByTheBridgeRelay(t *testing.T) {
	c := &fakeClient{}
	a := &Adapter{Client: c}
	_, err := a.Exec(context.Background(), target(), reach.Command{Program: "df", Args: []string{"-Pk", "/"}})
	if !reach.IsKind(err, reach.KindProtocolUnsupported) {
		t.Fatalf("expected reach_protocol_unsupported, got %v", err)
	}
	if len(c.calls) != 0 || len(c.dispatches) != 0 {
		t.Fatalf("refused observation reached the bridge: calls=%d dispatches=%d", len(c.calls), len(c.dispatches))
	}
	if _, err := a.OpenSession(context.Background(), target(), reach.SessionSpec{}); !reach.IsKind(err, reach.KindProtocolUnsupported) {
		t.Fatalf("a client without a session channel must be a typed refusal, got %v", err)
	}
}
