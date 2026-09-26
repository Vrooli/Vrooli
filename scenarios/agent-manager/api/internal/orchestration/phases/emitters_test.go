// Regression gates for the typed-error emission seam.
//
// The motivating bug: codex stderr like `record_rollout_items: thread …
// not found` was landing on the run timeline as `code: INTERNAL` because
// the failure was constructed from raw stderr text rather than a typed
// DomainError. Phase 4 wired Codec.ClassifyTerminalError → typed
// *domain.RunnerError → ExecErr promotion in phases/execute.go →
// EmitFailureEvent's typed-error branch. These tests assert each link.

package phases

import (
	"context"
	"errors"
	"testing"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

func TestEmitFailureEvent_TypedRunnerError_PreservesCode(t *testing.T) {
	cases := []struct {
		name     string
		err      domain.DomainError
		wantCode domain.ErrorCode
	}{
		{
			name:     "session expired",
			err:      domain.NewRunnerSessionExpiredError(domain.RunnerTypeCodex, errors.New("thread abc not found")),
			wantCode: domain.ErrCodeRunnerSessionExpired,
		},
		{
			name:     "session state lost",
			err:      domain.NewRunnerSessionStateLostError(domain.RunnerTypeCodex, errors.New("record_rollout_items: thread abc not found")),
			wantCode: domain.ErrCodeRunnerSessionStateLost,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			store := &filterableEventStore{}
			deps := Deps{Events: store}
			runID := uuid.New()

			EmitFailureEvent(context.Background(), deps, runID, tc.err)

			if len(store.events) != 1 {
				t.Fatalf("expected 1 event, got %d", len(store.events))
			}
			data, ok := store.events[0].Data.(*domain.ErrorEventData)
			if !ok {
				t.Fatalf("expected *ErrorEventData, got %T", store.events[0].Data)
			}
			if data.Code != string(tc.wantCode) {
				t.Errorf("Code = %q, want %q", data.Code, tc.wantCode)
			}
			if data.Code == string(domain.ErrCodeInternal) {
				t.Errorf("typed RunnerError leaked as INTERNAL — regression of the Phase 4 typed-error contract")
			}
		})
	}
}

// TestEmitGenericFailureEvent_FallsBackToInternal confirms the
// fallback path still works when no typed code is available.
func TestEmitGenericFailureEvent_FallsBackToInternal(t *testing.T) {
	store := &filterableEventStore{}
	deps := Deps{Events: store}
	runID := uuid.New()

	EmitGenericFailureEvent(context.Background(), deps, runID,
		errors.New("some bare untyped failure"))

	if len(store.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(store.events))
	}
	data := store.events[0].Data.(*domain.ErrorEventData)
	if data.Code != string(domain.ErrCodeInternal) {
		t.Errorf("Code = %q, want INTERNAL for bare error", data.Code)
	}
}

type failingPhaseEventStore struct{}

func (failingPhaseEventStore) Append(context.Context, uuid.UUID, ...*domain.RunEvent) error {
	return errors.New("append failed")
}

func (failingPhaseEventStore) Get(context.Context, uuid.UUID, event.GetOptions) ([]*domain.RunEvent, error) {
	return nil, nil
}

func (failingPhaseEventStore) Stream(context.Context, uuid.UUID, event.StreamOptions) (<-chan *domain.RunEvent, error) {
	return nil, nil
}

func (failingPhaseEventStore) Count(context.Context, uuid.UUID) (int64, error) {
	return 0, nil
}

func (failingPhaseEventStore) Delete(context.Context, uuid.UUID) error {
	return nil
}

type recordingPhaseBroadcaster struct {
	count int
}

func (b *recordingPhaseBroadcaster) BroadcastEvent(*domain.RunEvent) {
	b.count++
}

func (b *recordingPhaseBroadcaster) BroadcastRunStatus(*domain.Run) {}

func (b *recordingPhaseBroadcaster) BroadcastProgress(uuid.UUID, domain.RunPhase, int, string) {}

func TestEmitSystemEvent_DoesNotBroadcastWhenAppendFails(t *testing.T) {
	broadcaster := &recordingPhaseBroadcaster{}
	deps := Deps{Events: failingPhaseEventStore{}, Broadcaster: broadcaster}

	EmitSystemEvent(context.Background(), deps, uuid.New(), "info", "hello")

	if broadcaster.count != 0 {
		t.Fatalf("expected no broadcasts after append failure, got %d", broadcaster.count)
	}
}
