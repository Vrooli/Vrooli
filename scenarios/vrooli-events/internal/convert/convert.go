// DOC: docs/reference/api-endpoints.md
package convert

import (
	"time"

	domain "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-events/v1/domain"
	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/store"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// EnvelopeToEvent converts a proto EventEnvelope to an internal store.Event.
// The payload is stored as proto binary bytes of the Any message.
func EnvelopeToEvent(env *domain.EventEnvelope) (store.Event, error) {
	var payload []byte
	if env.Payload != nil {
		var err error
		payload, err = proto.Marshal(env.Payload)
		if err != nil {
			return store.Event{}, err
		}
	}

	var createdAt time.Time
	if env.Timestamp != nil {
		createdAt = env.Timestamp.AsTime()
	} else {
		createdAt = time.Now()
	}

	return store.Event{
		EventID:        env.EventId,
		SourceScenario: env.SourceScenario,
		TargetScenario: env.TargetScenario,
		EventType:      env.EventType,
		CorrelationID:  env.CorrelationId,
		Payload:        payload,
		Metadata:       env.Metadata,
		CreatedAt:      createdAt,
	}, nil
}

// EventToEnvelope converts an internal store.Event to a proto EventEnvelope.
func EventToEnvelope(e store.Event) (*domain.EventEnvelope, error) {
	env := &domain.EventEnvelope{
		EventId:        e.EventID,
		SourceScenario: e.SourceScenario,
		TargetScenario: e.TargetScenario,
		EventType:      e.EventType,
		CorrelationId:  e.CorrelationID,
		Timestamp:      timestamppb.New(e.CreatedAt),
		Metadata:       e.Metadata,
	}

	if len(e.Payload) > 0 {
		any := &anypb.Any{}
		if err := proto.Unmarshal(e.Payload, any); err != nil {
			// Payload bytes may not be valid proto (e.g. migrated data or raw JSON).
			// Dropping the payload silently is safer than returning an error that
			// would prevent the entire event from being read.
			any = nil
		}
		env.Payload = any
	}

	return env, nil
}
