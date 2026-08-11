package telemetry

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	telemetryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/telemetry"
	"program-runtime/internal/testutil/db"
)

type fakePublisher struct {
	err  error
	seen []*telemetryv1.ProgramEvent
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
	NewStoreWithDB(d, nil).Append(event)
	restarted := NewStoreWithDB(d, nil)
	events := restarted.List("sess_1", telemetryv1.EventKind_EVENT_KIND_UNSPECIFIED)
	require.Len(t, events, 1)
	require.Equal(t, "event_restart", events[0].GetEventId())
}
