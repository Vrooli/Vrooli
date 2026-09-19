package cliapptest

import (
	"strings"
	"testing"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// TestMustMarshalProto_RoundTrips uses google.protobuf.StringValue as a
// stable, scenario-independent proto.Message — cliapptest must stay decoupled
// from any per-scenario generated type, so a well-known wrapper is the right
// test target. Pinned: bytes the helper emits decode back into the original
// message via the same protojson dialect.
func TestMustMarshalProto_RoundTrips(t *testing.T) {
	original := wrapperspb.String("hello")
	body := MustMarshalProto(t, original)

	var got wrapperspb.StringValue
	if err := protojson.Unmarshal(body, &got); err != nil {
		t.Fatalf("decode round-trip: %v", err)
	}
	if got.Value != "hello" {
		t.Errorf("Value = %q, want hello", got.Value)
	}
}

// TestMustMarshalProto_PayloadAppearsInOutput is a smoke check that the
// marshalled bytes contain the user-supplied payload — drift on the
// MarshalOptions configuration would surface here as missing or transformed
// content.
func TestMustMarshalProto_PayloadAppearsInOutput(t *testing.T) {
	original := wrapperspb.String("snake-case-check")
	body := MustMarshalProto(t, original)
	if !strings.Contains(string(body), "snake-case-check") {
		t.Errorf("marshalled body should contain payload; got %s", body)
	}
}
