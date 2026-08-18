// DOC: docs/reference/api-endpoints.md
package convert

import (
	"fmt"
	"strings"
	"time"

	domain "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-events/v1/domain"
	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/store"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// EnvelopeToEvent persists the canonical envelope as one protobuf value while
// copying only query indexes into storage columns. No receipt fact is projected
// into untyped storage metadata.
func EnvelopeToEvent(env *domain.EventEnvelope) (store.Event, error) {
	if env == nil {
		return store.Event{}, fmt.Errorf("event envelope is required")
	}
	payload, err := proto.Marshal(env)
	if err != nil {
		return store.Event{}, fmt.Errorf("marshal event envelope: %w", err)
	}
	occurredAt := time.Now().UTC()
	if env.OccurredAt != nil {
		occurredAt = env.OccurredAt.AsTime()
	}
	var source, target, correlation string
	if env.Source != nil {
		source = env.Source.Scenario
	}
	if env.Target != nil {
		target = env.Target.Scenario
	}
	if env.Correlation != nil {
		correlation = env.Correlation.AgentRunId
	}
	return store.Event{EventID: env.EventId, SourceScenario: source, TargetScenario: target,
		EventType: env.EventType, CorrelationID: correlation, Payload: payload,
		CreatedAt: occurredAt}, nil
}

// EventToEnvelope restores only canonical v1 data. Historical payloads are
// intentionally rejected: the hard cut has no compatibility deserializer.
func EventToEnvelope(e store.Event) (*domain.EventEnvelope, error) {
	env := &domain.EventEnvelope{}
	if err := proto.Unmarshal(e.Payload, env); err != nil {
		return nil, fmt.Errorf("decode canonical event envelope: %w", err)
	}
	if env.OccurredAt == nil {
		env.OccurredAt = timestamppb.New(e.CreatedAt)
	}
	// ReceiptData was originally published under vrooli.events.v1.domain.
	// Keep historical envelopes queryable while all new writes use the
	// canonical vrooli.vrooli_events.v1.domain package name.
	if env.Data != nil && strings.HasSuffix(env.Data.TypeUrl, "/vrooli.events.v1.domain.ReceiptData") {
		env.Data.TypeUrl = "type.googleapis.com/vrooli.vrooli_events.v1.domain.ReceiptData"
	}
	return env, nil
}
