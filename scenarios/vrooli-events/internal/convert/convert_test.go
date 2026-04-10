package convert

import (
	"testing"
	"time"

	domain "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-events/v1/domain"
	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/store"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// [REQ:REQ-ES-002] Verify protobuf EventEnvelope to store.Event round-trip conversion
func TestEnvelopeToEventRoundTrip(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Millisecond)

	// Create an Any payload from a simple wrapper
	payload, err := anypb.New(wrapperspb.String("test-payload"))
	if err != nil {
		t.Fatalf("create any: %v", err)
	}

	env := &domain.EventEnvelope{
		EventId:        "evt-123",
		SourceScenario: "source-1",
		TargetScenario: "target-1",
		EventType:      "test.domain.action.v1",
		CorrelationId:  "corr-456",
		Timestamp:      timestamppb.New(now),
		Payload:        payload,
		Metadata:       map[string]string{"key": "value"},
	}

	// Convert to store.Event
	event, err := EnvelopeToEvent(env)
	if err != nil {
		t.Fatalf("envelopeToEvent: %v", err)
	}

	if event.EventID != "evt-123" {
		t.Fatalf("expected evt-123, got %s", event.EventID)
	}
	if event.SourceScenario != "source-1" {
		t.Fatalf("expected source-1, got %s", event.SourceScenario)
	}
	if event.EventType != "test.domain.action.v1" {
		t.Fatalf("expected test.domain.action.v1, got %s", event.EventType)
	}
	if len(event.Payload) == 0 {
		t.Fatal("expected non-empty payload")
	}
	if event.Metadata["key"] != "value" {
		t.Fatalf("expected metadata key=value, got %v", event.Metadata)
	}

	// Convert back to envelope
	envOut, err := EventToEnvelope(event)
	if err != nil {
		t.Fatalf("eventToEnvelope: %v", err)
	}

	if envOut.EventId != env.EventId {
		t.Fatalf("round-trip event_id: got %s, want %s", envOut.EventId, env.EventId)
	}
	if envOut.SourceScenario != env.SourceScenario {
		t.Fatalf("round-trip source: got %s, want %s", envOut.SourceScenario, env.SourceScenario)
	}
	if envOut.Payload == nil {
		t.Fatal("round-trip payload is nil")
	}
	if envOut.Payload.TypeUrl != payload.TypeUrl {
		t.Fatalf("round-trip type_url: got %s, want %s", envOut.Payload.TypeUrl, payload.TypeUrl)
	}
}

// [REQ:REQ-ES-002] Verify nil payload handling in envelope-to-event conversion
func TestEnvelopeToEventNilPayload(t *testing.T) {
	env := &domain.EventEnvelope{
		EventId:        "evt-no-payload",
		SourceScenario: "src",
		EventType:      "test.v1",
	}

	event, err := EnvelopeToEvent(env)
	if err != nil {
		t.Fatalf("envelopeToEvent: %v", err)
	}
	if len(event.Payload) != 0 {
		t.Fatalf("expected empty payload, got %d bytes", len(event.Payload))
	}

	envOut, err := EventToEnvelope(event)
	if err != nil {
		t.Fatalf("eventToEnvelope: %v", err)
	}
	if envOut.Payload != nil {
		t.Fatal("expected nil payload in round-trip")
	}
}

// [REQ:REQ-ES-002] Verify invalid payload bytes are silently dropped in event-to-envelope conversion
func TestEventToEnvelope_InvalidPayload(t *testing.T) {
	event := store.Event{
		EventID:        "evt-bad-payload",
		SourceScenario: "src",
		EventType:      "test.v1",
		Payload:        []byte("not-valid-protobuf"),
		CreatedAt:      time.Now().UTC(),
	}

	env, err := EventToEnvelope(event)
	if err != nil {
		t.Fatalf("expected no error for invalid payload, got %v", err)
	}
	if env.Payload != nil {
		t.Fatal("expected nil payload when bytes are invalid proto, got non-nil")
	}
	if env.EventId != "evt-bad-payload" {
		t.Fatalf("expected event ID preserved, got %s", env.EventId)
	}
}

// [REQ:REQ-ES-002] Verify nil timestamp in envelope uses current time
func TestEnvelopeToEvent_NilTimestamp(t *testing.T) {
	before := time.Now().UTC()
	env := &domain.EventEnvelope{
		EventId:        "evt-no-ts",
		SourceScenario: "src",
		EventType:      "test.v1",
		// Timestamp deliberately nil
	}

	event, err := EnvelopeToEvent(env)
	if err != nil {
		t.Fatalf("envelopeToEvent: %v", err)
	}
	after := time.Now().UTC()

	if event.CreatedAt.Before(before) || event.CreatedAt.After(after) {
		t.Fatalf("expected CreatedAt between %v and %v, got %v", before, after, event.CreatedAt)
	}
}

// [REQ:REQ-ES-002] Verify default timestamp population in event-to-envelope conversion
func TestEventToEnvelopeDefaults(t *testing.T) {
	event := store.Event{
		EventID:        "evt-1",
		SourceScenario: "src",
		EventType:      "test.v1",
		CreatedAt:      time.Now().UTC(),
	}

	env, err := EventToEnvelope(event)
	if err != nil {
		t.Fatalf("eventToEnvelope: %v", err)
	}
	if env.Timestamp == nil {
		t.Fatal("expected non-nil timestamp")
	}
}
