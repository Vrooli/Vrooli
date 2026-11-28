package telemetry

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"testing"
	"time"

	"program-runtime/internal/bindings"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	db "github.com/vrooli/api-core/databasetest"
	telemetryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/telemetry"
)

type fakePublisher struct {
	err  error
	seen []*telemetryv1.ProgramEvent
}

type failFirstPendingRepository struct {
	Repository
	mu    sync.Mutex
	calls int
	err   error
}

func (r *failFirstPendingRepository) Pending(ctx context.Context, now time.Time, limit int) ([]outboxRow, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls++
	if r.calls == 1 {
		return nil, r.err
	}
	return r.Repository.Pending(ctx, now, limit)
}

func (r *failFirstPendingRepository) callCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.calls
}

func (p *fakePublisher) Publish(_ context.Context, event *telemetryv1.ProgramEvent) error {
	if p.err != nil {
		return p.err
	}
	p.seen = append(p.seen, protoClone(event))
	return nil
}

func newTelemetryTestDB(t *testing.T) *sql.DB {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d, apidb.SchemaProviderFunc(Schema)))
	return d
}

func TestUnavailablePublisherLeavesPendingEventWithError(t *testing.T) { // [REQ:PRT-P1-009]
	now := time.Date(2026, 8, 11, 15, 0, 0, 0, time.UTC)
	d := newTelemetryTestDB(t)
	publisher := &fakePublisher{err: errors.New("events unavailable")}
	store := NewStoreWithOptions(Options{DB: d, Publisher: publisher, Clock: func() time.Time { return now }})
	store.Append(&telemetryv1.ProgramEvent{EventId: "event_pending", OccurredAt: now.Format(time.RFC3339Nano), Kind: telemetryv1.EventKind_PROGRAM_FAILED})
	require.NoError(t, store.DrainOnce(context.Background()))
	var state, lastError string
	var attempts int
	require.NoError(t, d.QueryRowContext(context.Background(), `SELECT state, attempt_count, last_error FROM event_outbox WHERE event_id = ?`, "event_pending").Scan(&state, &attempts, &lastError))
	require.Equal(t, "pending", state)
	require.Equal(t, 1, attempts)
	require.Equal(t, "events unavailable", lastError)
}

func TestRecoveredPublisherDrainsPendingEventToDelivered(t *testing.T) { // [REQ:PRT-P1-009]
	now := time.Date(2026, 8, 11, 15, 0, 0, 0, time.UTC)
	d := newTelemetryTestDB(t)
	publisher := &fakePublisher{err: errors.New("events unavailable")}
	store := NewStoreWithOptions(Options{DB: d, Publisher: publisher, Clock: func() time.Time { return now }})
	store.Append(&telemetryv1.ProgramEvent{EventId: "event_recovered", OccurredAt: now.Format(time.RFC3339Nano), Kind: telemetryv1.EventKind_PROGRAM_SUCCEEDED})
	require.NoError(t, store.DrainOnce(context.Background()))
	publisher.err = nil
	now = now.Add(2 * time.Second)
	require.NoError(t, store.DrainOnce(context.Background()))
	var state string
	require.NoError(t, d.QueryRowContext(context.Background(), `SELECT state FROM event_outbox WHERE event_id = ?`, "event_recovered").Scan(&state))
	require.Equal(t, "delivered", state)
	require.Len(t, publisher.seen, 1)
}

func TestExpiredPendingEventBecomesDeadLetter(t *testing.T) { // [REQ:PRT-P1-009]
	now := time.Date(2026, 8, 11, 15, 0, 0, 0, time.UTC)
	d := newTelemetryTestDB(t)
	publisher := &fakePublisher{err: errors.New("events unavailable")}
	store := NewStoreWithOptions(Options{DB: d, Publisher: publisher, Clock: func() time.Time { return now }})
	store.Append(&telemetryv1.ProgramEvent{EventId: "event_dead", OccurredAt: now.Add(-8 * 24 * time.Hour).Format(time.RFC3339Nano), Kind: telemetryv1.EventKind_PROGRAM_FAILED})
	require.NoError(t, store.DrainOnce(context.Background()))
	var state, lastError string
	require.NoError(t, d.QueryRowContext(context.Background(), `SELECT state, last_error FROM event_outbox WHERE event_id = ?`, "event_dead").Scan(&state, &lastError))
	require.Equal(t, "dead", state)
	require.Contains(t, lastError, "dead-letter")
	require.Empty(t, publisher.seen)
}

func TestTelemetryListingSurvivesRepositoryRestart(t *testing.T) { // [REQ:PRT-P1-009]
	d := newTelemetryTestDB(t)
	event := &telemetryv1.ProgramEvent{EventId: "event_restart", SessionId: "sess_1", OccurredAt: "2026-08-11T15:00:00Z", Kind: telemetryv1.EventKind_PROGRAM_SUBMITTED}
	NewStoreWithOptions(Options{DB: d}).Append(event)
	restarted := NewStoreWithOptions(Options{DB: d})
	events := restarted.List("sess_1", telemetryv1.EventKind_EVENT_KIND_UNSPECIFIED)
	require.Len(t, events, 1)
	require.Equal(t, "event_restart", events[0].GetEventId())
}

func TestDrainerReportsFirstAsyncFailureOnceAndStopsOnCancellation(t *testing.T) {
	wantErr := errors.New("temporary repository failure")
	repo := &failFirstPendingRepository{Repository: newMemoryRepository(), err: wantErr}
	reported := make(chan error, 1)
	ctx, cancel := context.WithCancel(context.Background())
	drainer := NewDrainer(repo, &fakePublisher{}, nil, func(err error) {
		reported <- err
		cancel()
	})
	drainer.interval = time.Second

	drainer.Start(ctx)

	select {
	case gotErr := <-reported:
		require.ErrorIs(t, gotErr, wantErr)
	case <-time.After(3 * time.Second):
		t.Fatal("drainer did not report its asynchronous repository failure")
	}

	drainer.Stop()

	require.Equal(t, 1, repo.callCount(), "cancellation should stop the next drain attempt")
	select {
	case err := <-reported:
		t.Fatalf("reported more than the first failure: %v", err)
	default:
	}
}

func TestMeasuredTelemetryOutboxRoundTripsAndCorrelatesBindingLedger(t *testing.T) {
	ctx := context.Background()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(ctx, d, apidb.SchemaProviderFunc(Schema)))
	require.NoError(t, apidb.EnsureSchemas(ctx, d, apidb.SchemaProviderFunc(bindings.Schema)))

	now := time.Date(2026, 11, 24, 6, 35, 0, 0, time.UTC)
	store := NewStoreWithOptions(Options{DB: d, Clock: func() time.Time { return now }})
	executionWallTimeMS := int64(1234)
	require.NoError(t, store.AppendContext(ctx, &telemetryv1.ProgramEvent{
		EventId: "event_execution", OccurredAt: now.Format(time.RFC3339Nano),
		Kind: telemetryv1.EventKind_PROGRAM_SUCCEEDED, ProgramId: "program-1", SessionId: "session-1",
		ExecutionWallTimeMs: &executionWallTimeMS,
	}))

	ledger := bindings.NewInvocationRepository(d)
	recorder := WithInvocationTelemetry(ledger, store)
	invocation := bindings.Invocation{
		InvocationID: "inv_ledger_1", BindingID: "ai-gateway/inference/run", TargetScenario: "ai-gateway",
		SessionID: "session-1", ProgramID: "program-1", Provenance: "PROVENANCE_AGENT",
		Outcome: "success", LatencyMS: 42, OccurredAt: now.Add(time.Second),
	}
	require.NoError(t, recorder.RecordInvocation(ctx, invocation))

	// A new Store instance forces the test through SQLite payload persistence,
	// then the production typed JSON decode path in Repository.List.
	restarted := NewStoreWithOptions(Options{DB: d})
	events := restarted.List("session-1", telemetryv1.EventKind_EVENT_KIND_UNSPECIFIED)
	require.Len(t, events, 2)
	var lifecycle, measured *telemetryv1.ProgramEvent
	for _, event := range events {
		switch event.GetKind() {
		case telemetryv1.EventKind_PROGRAM_SUCCEEDED:
			lifecycle = event
		case telemetryv1.EventKind_BINDING_CALL_MEASURED:
			measured = event
		}
	}
	require.NotNil(t, lifecycle)
	require.NotNil(t, lifecycle.ExecutionWallTimeMs, "measured zero and unknown must remain distinguishable")
	require.Equal(t, executionWallTimeMS, lifecycle.GetExecutionWallTimeMs())
	require.NotNil(t, measured)
	require.Equal(t, "program-1", measured.GetProgramId())
	require.Equal(t, "ai-gateway/inference/run", measured.GetBindingId())
	require.Equal(t, invocation.InvocationID, measured.GetBindingInvocationId())
	require.NotNil(t, measured.BindingCallLatencyMs)
	require.Equal(t, int64(42), measured.GetBindingCallLatencyMs())
	require.Equal(t, "success", measured.GetBindingCallOutcome())
	require.Nil(t, measured.ExecutionWallTimeMs, "binding event has no total execution measurement")

	ledgerRows, err := bindings.NewInvocationRepository(d).ListInvocations(ctx, now, invocation.BindingID, "ai-gateway")
	require.NoError(t, err)
	require.Len(t, ledgerRows, 1)
	require.Equal(t, ledgerRows[0].InvocationID, measured.GetBindingInvocationId())
	require.Equal(t, ledgerRows[0].ProgramID, measured.GetProgramId())
	require.Equal(t, ledgerRows[0].BindingID, measured.GetBindingId())
	require.Equal(t, ledgerRows[0].LatencyMS, measured.GetBindingCallLatencyMs())
	require.Equal(t, ledgerRows[0].Outcome, measured.GetBindingCallOutcome())
	require.Equal(t, lifecycle.GetProgramId(), measured.GetProgramId())
}
