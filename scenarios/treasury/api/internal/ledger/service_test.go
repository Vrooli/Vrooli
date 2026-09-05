package ledger_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"treasury/internal/ledger"
)

type memoryRepository struct {
	value ledger.Emission
}

func (r *memoryRepository) Pending(context.Context, int) ([]ledger.Emission, error) {
	if r.value.Status != ledger.StatusQueued {
		return nil, nil
	}
	return []ledger.Emission{r.value}, nil
}

func (r *memoryRepository) GetBySettlement(context.Context, string) (ledger.Emission, error) {
	return r.value, nil
}

func (r *memoryRepository) MarkFailure(_ context.Context, _ string, detail string) error {
	r.value.Attempts++
	r.value.LastError = detail
	return nil
}

func (r *memoryRepository) MarkAccepted(_ context.Context, _ string, at time.Time) error {
	r.value.Attempts++
	r.value.Status = ledger.StatusAccepted
	r.value.LastError = ""
	r.value.AcceptedAt = at
	return nil
}

type sequenceEmitter struct {
	calls int
	err   error
}

func (e *sequenceEmitter) Emit(context.Context, ledger.Emission) (bool, error) {
	e.calls++
	return e.calls > 1, e.err
}

// [REQ:TRS-P0-008] An outage changes only outbox delivery metadata. Once the
// downstream contract accepts the stable event, later drains emit nothing.
func TestDeferredEmissionIsAcceptedExactlyOnce(t *testing.T) {
	now := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	repository := &memoryRepository{value: ledger.Emission{ID: "settlement-1:ledger", SettlementID: "settlement-1", Status: ledger.StatusQueued}}
	emitter := &sequenceEmitter{err: errors.New("connection refused")}
	service := ledger.NewService(repository, emitter, func() time.Time { return now })
	require.ErrorContains(t, service.DrainPending(context.Background()), "connection refused")
	require.Equal(t, ledger.StatusQueued, repository.value.Status)
	require.Equal(t, 1, repository.value.Attempts)

	emitter.err = nil
	require.NoError(t, service.DrainPending(context.Background()))
	require.Equal(t, ledger.StatusAccepted, repository.value.Status)
	require.Equal(t, 2, repository.value.Attempts)
	require.NoError(t, service.DrainPending(context.Background()))
	require.Equal(t, 2, emitter.calls, "accepted outbox rows must never be delivered again")
}
