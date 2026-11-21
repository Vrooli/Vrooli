package bridge

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	domain "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-events/v1/domain"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
	"time"
)

// EvidenceBoundary is an opt-in trusted owner capability, never request data.
// It MUST share live authority/session serialization with Boundary.Within.
// The source instance comes from current approved configuration while held;
// recheck and validate must cover that same binding and selected event.
// No production implementation is installed; a disconnected mutex is insufficient.
type EvidenceBoundary interface {
	Boundary
	WithinEvidence(context.Context, func(context.Context, string) error) error
}

// SelectedEvidence exposes observations only, never raw payload or Work lineage.
// The digest binds exact persisted protobuf bytes, not JSON serialization.
type SelectedEvidence struct {
	SourceInstance string `json:"source_instance"`
	WakeEvent
	Category string `json:"category,omitempty"`
	Status   string `json:"status,omitempty"`
}

// ReadSelectedEvidence is a dormant minimized counterpart to ReadSelected.
// Missing records return nil; unavailable authority and unsupported events fail
// closed. It registers no route and leaves ReadSelected's DTO unchanged.
func (r *SelectedReader) ReadSelectedEvidence(ctx context.Context, id string, recheck func(context.Context) error, validate func(context.Context, string, string, string) error) ([]byte, error) {
	if r == nil || r.store == nil || r.boundary == nil || ctx == nil || ctx.Err() != nil || !selectedID.MatchString(id) || recheck == nil || validate == nil {
		return nil, ErrSelectedRead
	}
	boundary, ok := r.boundary.(EvidenceBoundary)
	if !ok {
		return nil, ErrSelectedRead
	}
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	var result []byte
	invocations := 0
	completed := false
	err := boundary.WithinEvidence(ctx, func(inner context.Context, sourceInstance string) error {
		invocations++
		if invocations != 1 || inner == nil || ctx.Err() != nil || inner.Err() != nil || !opaque.MatchString(sourceInstance) {
			return ErrSelectedRead
		}
		deadline, _ := ctx.Deadline()
		bounded, stop := context.WithDeadline(inner, deadline)
		defer stop()
		stopParent := context.AfterFunc(ctx, stop)
		defer stopParent()
		if recheck(bounded) != nil {
			return ErrSelectedRead
		}
		e, err := r.store.SelectedCanonical(bounded, id)
		if err != nil {
			return ErrSelectedRead
		}
		if e != nil {
			var env domain.EventEnvelope
			if len(e.Payload) == 0 || len(e.Payload) > 1<<20 || proto.Unmarshal(e.Payload, &env) != nil || env.EventId != id || env.EventId != e.EventID || !supportedRunEvent(env.EventType) || env.EventType != e.EventType || env.GetSource().GetScenario() != "agent-manager" || env.GetSource().GetScenario() != e.SourceScenario || env.OccurredAt == nil || env.OccurredAt.CheckValid() != nil || !env.OccurredAt.AsTime().Truncate(time.Millisecond).Equal(e.CreatedAt.Truncate(time.Millisecond)) {
				return ErrSelectedRead
			}
			if validate(bounded, env.EventId, env.EventType, env.GetSource().GetScenario()) != nil {
				return ErrSelectedRead
			}
			var facts structpb.Struct
			if env.Data == nil || env.Data.UnmarshalTo(&facts) != nil {
				return ErrSelectedRead
			}
			locator, category, status, err := projectRunFacts(env.EventType, &facts)
			if err != nil {
				return ErrSelectedRead
			}
			sum := sha256.Sum256(e.Payload)
			result, err = json.Marshal(SelectedEvidence{SourceInstance: sourceInstance, WakeEvent: WakeEvent{EventID: env.EventId, EventType: env.EventType, Source: env.Source.Scenario, OccurredAt: canonicalOccurrence(env.OccurredAt.AsTime().UTC()), SourceContentDigest: hex.EncodeToString(sum[:]), Locator: locator}, Category: category, Status: status})
			if err != nil || len(result) > maxWakeBytes {
				return ErrSelectedRead
			}
		}
		if ctx.Err() != nil || bounded.Err() != nil || recheck(bounded) != nil {
			return ErrSelectedRead
		}
		completed = true
		return nil
	})
	if err != nil || invocations != 1 || !completed || ctx.Err() != nil {
		return nil, ErrSelectedRead
	}
	return result, nil
}
