package bridge

import (
	"context"
	"testing"

	"github.com/vrooli/api-core/nodereach"
	relayv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/relay"
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

func TestTypedObservationUsesBridgeRelay(t *testing.T) {
	c := &fakeClient{call: nodereach.CallResponse{CorrelationID: "corr-observe", Outcome: relayv1.RelayCallOutcome_RELAY_CALL_OUTCOME_COMPLETED, Data: []byte(`{"result":{"stdout":"Filesystem 1024-blocks Used Available Capacity Mounted on\n"}}`)}}
	a := &Adapter{Client: c}
	cmd, err := reach.NewObservation("df", "-Pk", "/")
	if err != nil {
		t.Fatal(err)
	}
	result, err := a.Exec(context.Background(), target(), cmd)
	if err != nil {
		t.Fatal(err)
	}
	if result.CorrelationID != "corr-observe" || len(c.calls) != 1 || c.calls[0].Command != "cloud-target" || len(c.calls[0].Args) == 0 || c.calls[0].Args[0] != "host" {
		t.Fatalf("typed observation relay call=%+v result=%+v", c.calls, result)
	}
}
