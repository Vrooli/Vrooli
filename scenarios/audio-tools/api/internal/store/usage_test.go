package store_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/store"
)

func TestUsageStore_InsertListSummary(t *testing.T) {
	d := newTestDB(t)
	s := store.NewUsageStore(d)
	ctx := context.Background()
	now := time.Now().UTC()
	require.NoError(t, s.Insert(ctx, store.UsageRow{
		OperationID: "op-1", EmittedAt: now.Add(-1 * time.Minute),
		Capability: "stt", Operation: "transcribe",
		ProviderTier: "local", ProviderID: "whisper", LatencyMs: 120, CreditsCharged: 0,
	}))
	require.NoError(t, s.Insert(ctx, store.UsageRow{
		OperationID: "op-2", EmittedAt: now.Add(-30 * time.Second),
		Capability: "tts", Operation: "synthesize",
		ProviderTier: "local", ProviderID: "kokoro", LatencyMs: 200, CreditsCharged: 1,
	}))
	// Idempotent on op id
	require.NoError(t, s.Insert(ctx, store.UsageRow{OperationID: "op-1", Capability: "stt", Operation: "transcribe", ProviderTier: "local", ProviderID: "whisper"}))

	rows, err := s.ListRecent(ctx, now.Add(-1*time.Hour), 10, "", "")
	require.NoError(t, err)
	require.Len(t, rows, 2)

	sum, err := s.Summary(ctx, now.Add(-1*time.Hour), "")
	require.NoError(t, err)
	require.EqualValues(t, 2, sum.OperationsTotal)
	require.EqualValues(t, 1, sum.CreditsTotal)
	require.Len(t, sum.Distribution, 2)
}
