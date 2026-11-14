package eventbus

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestRetainedDomainEventExactIdentityAndDeterminism(t *testing.T) {
	event := DomainEvent{Source: "agent-manager", EventType: "agent_manager.run.terminal.v1", Occurred: time.Date(2026, 11, 7, 0, 1, 2, 123456789, time.UTC), Payload: map[string]any{"z": "last", "a": "first", "nested": map[string]any{"c": "c", "b": "b"}}}
	seed, _ := json.Marshal(event)
	id := digestBytes(seed)
	env, encoded, err := RetainedDomainEventEnvelope(string(seed), id)
	if err != nil {
		t.Fatal(err)
	}
	var payload structpb.Struct
	if err := env.Data.UnmarshalTo(&payload); err != nil {
		t.Fatal(err)
	}
	if payload.AsMap()["a"] != "first" || env.EventId != id || !env.OccurredAt.AsTime().Equal(event.Occurred) {
		t.Fatal("identity changed")
	}
	for i := 0; i < 100; i++ {
		again, wire, err := RetainedDomainEventEnvelope(string(seed), id)
		if err != nil || !proto.Equal(env, again) || string(wire) != string(encoded) {
			t.Fatalf("unstable encoding: %v", err)
		}
	}
	ack := DomainEventAcknowledgment{DomainEventAcknowledgmentSchema, id, digestBytes(encoded), 1}
	if err := ValidateDomainEventAcknowledgment(string(seed), id, ack); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []DomainEventAcknowledgment{{}, {"other", id, ack.CanonicalSHA256, 1}, {ack.Schema, strings.Repeat("a", 64), ack.CanonicalSHA256, 1}, {ack.Schema, id, strings.Repeat("a", 64), 1}, {ack.Schema, id, ack.CanonicalSHA256, 0}} {
		if ValidateDomainEventAcknowledgment(string(seed), id, bad) == nil {
			t.Fatal("accepted invalid acknowledgment")
		}
	}
}

func TestRetainedDomainEventRejectsAmbiguousSeeds(t *testing.T) {
	good := `{"Source":"agent-manager","EventType":"terminal","Payload":{"id":"one"},"Occurred":"2026-11-07T00:00:00Z"}`
	for _, seed := range []string{"", good + " ", good + "{}", strings.Replace(good, `"Source":"agent-manager"`, `"Source":"bad","Source":"agent-manager"`, 1), strings.Replace(good, `"Source":"agent-manager"`, `"Extra":"bad","Source":"agent-manager"`, 1), strings.Replace(good, "2026-11-07T00:00:00Z", "0001-01-01T00:00:00Z", 1), strings.Repeat("x", 16385)} {
		if _, _, err := RetainedDomainEventEnvelope(seed, digestBytes([]byte(seed))); err == nil {
			t.Fatalf("accepted invalid seed: %.100s", seed)
		}
	}
	if _, _, err := RetainedDomainEventEnvelope(good, strings.Repeat("0", 64)); err == nil {
		t.Fatal("accepted mismatched ID")
	}
}
