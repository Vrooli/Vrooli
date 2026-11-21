package eventbus

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"

	domain "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-events/v1/domain"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const DomainEventAcknowledgmentSchema = "vrooli-events.domain-event-ack.v1"

// DomainEventAcknowledgment proves owner acceptance, not current event retention,
// subscriber delivery, authorization, or acceptance of the underlying work.
// Only a trusted durable Events owner may provide it. A successful best-effort
// PublishDomainEvent call (including a no-op or generic 409) is not this evidence.
type DomainEventAcknowledgment struct {
	Schema          string `json:"schema"`
	EventID         string `json:"event_id"`
	CanonicalSHA256 string `json:"canonical_sha256"`
	StoreID         int64  `json:"store_id"`
}

// RetainedDomainEventEnvelope consumes the original immutable JSON seed. It
// never assigns a new identity, occurrence, or wall-clock fallback. The bounded
// canonical seed requirement rejects ambiguous duplicate/unknown fields. The
// Any payload is also deterministically encoded, not just its outer envelope.
// This is a codec only: it does not publish, authorize or configure anything.
func RetainedDomainEventEnvelope(intentJSON, eventID string) (*domain.EventEnvelope, []byte, error) {
	if len(intentJSON) == 0 || len(intentJSON) > 16*1024 || digestBytes([]byte(intentJSON)) != eventID {
		return nil, nil, errors.New("invalid retained domain event identity or size")
	}
	var event DomainEvent
	decoder := json.NewDecoder(bytes.NewBufferString(intentJSON))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&event); err != nil {
		return nil, nil, err
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return nil, nil, errors.New("invalid retained domain event suffix")
	}
	canonical, err := json.Marshal(event)
	if err != nil || !bytes.Equal(canonical, []byte(intentJSON)) || event.Source == "" || event.EventType == "" || event.Occurred.IsZero() || event.Payload == nil {
		return nil, nil, errors.New("invalid retained domain event seed")
	}
	at := timestamppb.New(event.Occurred)
	if err := at.CheckValid(); err != nil {
		return nil, nil, err
	}
	payload, err := structpb.NewStruct(event.Payload)
	if err != nil {
		return nil, nil, err
	}
	value, err := (proto.MarshalOptions{Deterministic: true}).Marshal(payload)
	if err != nil {
		return nil, nil, err
	}
	envelope := &domain.EventEnvelope{EventId: eventID, EventType: event.EventType, OccurredAt: at, Source: &domain.EventSource{Scenario: event.Source, ActorKind: "system"}, Data: &anypb.Any{TypeUrl: "type.googleapis.com/google.protobuf.Struct", Value: value}}
	encoded, err := (proto.MarshalOptions{Deterministic: true}).Marshal(envelope)
	return envelope, encoded, err
}

func ValidateDomainEventAcknowledgment(intentJSON, eventID string, ack DomainEventAcknowledgment) error {
	_, encoded, err := RetainedDomainEventEnvelope(intentJSON, eventID)
	if err != nil {
		return err
	}
	if ack.Schema != DomainEventAcknowledgmentSchema || ack.EventID != eventID || ack.CanonicalSHA256 != digestBytes(encoded) || ack.StoreID < 1 {
		return errors.New("domain event acknowledgment does not match retained intent")
	}
	return nil
}
func digestBytes(value []byte) string { sum := sha256.Sum256(value); return hex.EncodeToString(sum[:]) }
