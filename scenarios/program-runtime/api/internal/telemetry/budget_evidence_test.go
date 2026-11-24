package telemetry

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"program-runtime/internal/bindings"
	"program-runtime/internal/budgets"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	telemetryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/telemetry"
)

func TestBudgetEvidenceAdapterProjectsCorrelatedEventsAndKeepsMissingStrataUnknown(t *testing.T) {
	ctx := context.Background()
	d := newTelemetryTestDB(t)
	require.NoError(t, apidb.EnsureSchemas(ctx, d, apidb.SchemaProviderFunc(bindings.Schema)))
	now := time.Date(2026, 11, 24, 6, 40, 0, 0, time.UTC)
	store := NewStoreWithOptions(Options{DB: d, Clock: func() time.Time { return now }})
	wallTimeMS := int64(2500)
	require.NoError(t, store.AppendContext(ctx, &telemetryv1.ProgramEvent{
		EventId: "terminal-1", OccurredAt: now.Format(time.RFC3339Nano),
		Kind: telemetryv1.EventKind_PROGRAM_SUCCEEDED, ProgramId: "program-1", SessionId: "session-1",
		ExecutionWallTimeMs: &wallTimeMS, CallerRunId: "private-caller", Reason: "private-error-detail",
	}))
	// This unrelated kind must not enter the budget projection.
	require.NoError(t, store.AppendContext(ctx, &telemetryv1.ProgramEvent{
		EventId: "submitted-1", OccurredAt: now.Format(time.RFC3339Nano),
		Kind: telemetryv1.EventKind_PROGRAM_SUBMITTED, ProgramId: "program-1", SessionId: "session-1",
		CallerRunId: "private-caller", Reason: "private-error-detail",
	}))

	ledger := bindings.NewInvocationRepository(d)
	recorder := WithInvocationTelemetry(ledger, store)
	require.NoError(t, recorder.RecordInvocation(ctx, bindings.Invocation{
		InvocationID: "inv_1", BindingID: "ai-gateway/inference/run", TargetScenario: "ai-gateway",
		SessionID: "session-1", ProgramID: "program-1", Provenance: "PROVENANCE_AGENT",
		Outcome: "success", LatencyMS: 700, OccurredAt: now.Add(-2 * time.Hour),
	}))

	restarted := NewStoreWithOptions(Options{DB: d})
	evidence, err := restarted.ReadBudgetLatencyEvidence(ctx, now.Add(-time.Minute), now.Add(time.Minute))
	require.NoError(t, err)
	require.Len(t, evidence, 1, "only terminal execution events should create budget observations")
	got := evidence[0]
	require.Equal(t, "program-1", got.ProgramID)
	require.Equal(t, "success", got.Status)
	require.True(t, got.TotalKnown)
	require.Equal(t, 2500*time.Millisecond, got.Total)
	require.True(t, got.BindingPhaseKnown)
	require.Equal(t, 700*time.Millisecond, got.BindingPhase)
	require.Len(t, got.BindingCalls, 1)
	require.Equal(t, "inv_1", got.BindingCalls[0].InvocationID)
	require.Equal(t, "ai-gateway/inference/run", got.BindingCalls[0].BindingID)
	require.Equal(t, "success", got.BindingCalls[0].Outcome)
	require.True(t, got.BindingCalls[0].LatencyKnown)
	require.False(t, got.ModelPhaseKnown)
	require.Zero(t, got.ModelPhase)
	require.False(t, got.ThermalStateKnown)
	require.Equal(t, "unknown", got.ThermalState)

	ledgerRows, err := bindings.NewInvocationRepository(d).ListInvocations(ctx, now.Add(-3*time.Hour), "", "")
	require.NoError(t, err)
	require.Len(t, ledgerRows, 1)
	require.Equal(t, ledgerRows[0].InvocationID, got.BindingCalls[0].InvocationID)
	require.Equal(t, ledgerRows[0].ProgramID, got.ProgramID)
	require.Equal(t, ledgerRows[0].BindingID, got.BindingCalls[0].BindingID)

	// The adapter's returned projection must not contain unrelated payload data.
	projected, err := json.Marshal(got)
	require.NoError(t, err)
	require.NotContains(t, string(projected), "private-caller")
	require.NotContains(t, string(projected), "private-error-detail")

	_, err = restarted.QualifyPersistedBudgetLatencyEvidence(ctx, now.Add(-time.Minute), now.Add(time.Minute), budgets.LatencyEvidencePolicy{MinSamplesPerStratum: 1})
	require.ErrorContains(t, err, "unknown thermal state", "missing cold/warm evidence must keep the gate closed")
}

func TestBudgetEvidenceRequiresPersistentOutboxReader(t *testing.T) {
	store := NewStore()
	_, err := store.ReadBudgetLatencyEvidence(context.Background(), time.Now().Add(-time.Hour), time.Now())
	require.ErrorContains(t, err, "persistent telemetry outbox")
}
