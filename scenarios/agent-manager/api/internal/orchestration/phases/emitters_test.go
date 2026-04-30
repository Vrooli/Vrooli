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
