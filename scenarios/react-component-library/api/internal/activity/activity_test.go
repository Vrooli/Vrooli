package activity

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	db "github.com/vrooli/api-core/databasetest"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	sqlDB := db.NewSQLite(t)
	_, err := sqlDB.ExecContext(context.Background(), Schema())
	require.NoError(t, err)
	store := NewSQLiteStore(sqlDB)
	store.clock = func() time.Time { return time.Date(2026, 11, 19, 12, 0, 0, 0, time.UTC) }
	// A fixed stream keeps generated ids deterministic without reusing the
	// same suffix for every event.
	store.random = bytes.NewReader(bytes.Repeat([]byte{0x5a, 0x1c, 0x9e, 0x42}, 64))
	return store
}

func eventAt(at time.Time, actor, action, kind, id string) Event {
	return Event{
		Actor:       actor,
		Action:      action,
		SubjectKind: kind,
		SubjectID:   id,
		Summary:     action + " " + id,
		BeforeJSON:  `{"state":"before"}`,
		AfterJSON:   `{"state":"after"}`,
		At:          at.Format(time.RFC3339Nano),
		TraceID:     "trace-" + id,
	}
}

func TestRecordAndListNewestFirst(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	base := time.Date(2026, 11, 18, 9, 0, 0, 0, time.UTC)

	events := []Event{
		eventAt(base, "ada", "workspace.save", "workspace", "ws-1"),
		eventAt(base.Add(time.Hour), "lin", "asset.publish", "asset", "button"),
		eventAt(base.Add(2*time.Hour), "ada", "asset.publish", "asset", "card"),
	}
	for _, event := range events {
		require.NoError(t, store.Record(ctx, event))
	}

	got, err := store.List(ctx, Filter{}, 50)
	require.NoError(t, err)
	require.Len(t, got, 3)
	require.Equal(t, "asset", got[0].SubjectKind)
	require.Equal(t, "card", got[0].SubjectID)
	require.Equal(t, "ada", got[0].Actor)
	require.Equal(t, "asset.publish", got[0].Action)
	require.Equal(t, `{"state":"after"}`, got[0].AfterJSON)
	require.Equal(t, "trace-card", got[0].TraceID)
	require.Equal(t, time.Date(2026, 11, 18, 11, 0, 0, 0, time.UTC).Format(timestampLayout), got[0].At)
	require.NotEmpty(t, got[0].ID)
	// Newest-first ordering is the contract the Activity list relies on.
	require.Equal(t, "button", got[1].SubjectID)
	require.Equal(t, "ws-1", got[2].SubjectID)
}

func TestListFiltersAndSince(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	base := time.Date(2026, 11, 10, 9, 0, 0, 0, time.UTC)

	require.NoError(t, store.Record(ctx, eventAt(base, "ada", "workspace.save", "workspace", "ws-1")))
	require.NoError(t, store.Record(ctx, eventAt(base.Add(time.Hour), "lin", "workspace.save", "workspace", "ws-2")))
	require.NoError(t, store.Record(ctx, eventAt(base.Add(2*time.Hour), "ada", "asset.publish", "asset", "button")))

	byKind, err := store.List(ctx, Filter{SubjectKind: "workspace"}, 50)
	require.NoError(t, err)
	require.Len(t, byKind, 2)

	bySubject, err := store.List(ctx, Filter{SubjectKind: "workspace", SubjectID: "ws-2"}, 50)
	require.NoError(t, err)
	require.Len(t, bySubject, 1)
	require.Equal(t, "lin", bySubject[0].Actor)

	byActor, err := store.List(ctx, Filter{Actor: "ada"}, 50)
	require.NoError(t, err)
	require.Len(t, byActor, 2)

	since := base.Add(90 * time.Minute).Format(time.RFC3339Nano)
	recent, err := store.List(ctx, Filter{Since: since}, 50)
	require.NoError(t, err)
	require.Len(t, recent, 1)
	require.Equal(t, "button", recent[0].SubjectID)

	_, err = store.List(ctx, Filter{Since: "not-a-timestamp"}, 50)
	require.ErrorIs(t, err, ErrInvalid)
}

func TestListLimitIsBounded(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	base := time.Date(2026, 11, 18, 9, 0, 0, 0, time.UTC)
	for index := range 5 {
		require.NoError(t, store.Record(ctx, eventAt(base.Add(time.Duration(index)*time.Minute), "ada", "asset.publish", "asset", "asset-"+string(rune('a'+index)))))
	}

	limited, err := store.List(ctx, Filter{}, 2)
	require.NoError(t, err)
	require.Len(t, limited, 2)

	require.Equal(t, defaultListLimit, clampLimit(0))
	require.Equal(t, defaultListLimit, clampLimit(-4))
	require.Equal(t, maxListLimit, clampLimit(10_000))
}

func TestSameTimestampOrdersByIDDescending(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	at := time.Date(2026, 11, 18, 9, 0, 0, 0, time.UTC)
	first := eventAt(at, "ada", "asset.publish", "asset", "asset-a")
	first.ID = "aaa"
	second := eventAt(at, "ada", "asset.publish", "asset", "asset-b")
	second.ID = "bbb"
	require.NoError(t, store.Record(ctx, first))
	require.NoError(t, store.Record(ctx, second))

	got, err := store.List(ctx, Filter{}, 10)
	require.NoError(t, err)
	require.Len(t, got, 2)
	require.Equal(t, "bbb", got[0].ID)
	require.Equal(t, "aaa", got[1].ID)
}

func TestSummaryGroupsAndEmpty(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	today := time.Date(2026, 11, 19, 10, 0, 0, 0, time.UTC)
	yesterday := today.Add(-24 * time.Hour)

	require.NoError(t, store.Record(ctx, eventAt(today, "ada", "workspace.save", "workspace", "ws-1")))
	require.NoError(t, store.Record(ctx, eventAt(today.Add(-time.Hour), "ada", "asset.publish", "asset", "button")))
	require.NoError(t, store.Record(ctx, eventAt(yesterday, "lin", "asset.publish", "asset", "card")))

	summary, err := store.Summary(ctx, 30)
	require.NoError(t, err)
	require.Equal(t, 3, summary.Total)
	require.False(t, summary.Empty)
	require.Equal(t, []DayCount{
		{Date: "2026-11-18", Count: 1},
		{Date: "2026-11-19", Count: 2},
	}, summary.ByDay)
	require.Equal(t, []ActorCount{
		{Actor: "ada", Count: 2},
		{Actor: "lin", Count: 1},
	}, summary.ByActor)
	require.Equal(t, []ActionCount{
		{Action: "asset.publish", Count: 2},
		{Action: "workspace.save", Count: 1},
	}, summary.ByAction)

	// days=0 selects the server default window rather than an empty window.
	narrow, err := store.Summary(ctx, 0)
	require.NoError(t, err)
	require.Equal(t, 3, narrow.Total, "0 days means the default window")

	emptyStore := newTestStore(t)
	empty, err := emptyStore.Summary(ctx, 30)
	require.NoError(t, err)
	require.Zero(t, empty.Total)
	require.True(t, empty.Empty)
	require.Empty(t, empty.ByDay)
	require.Empty(t, empty.ByActor)
	require.Empty(t, empty.ByAction)
}

func TestRecordValidationAndGeneratedIDs(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	require.ErrorIs(t, store.Record(ctx, Event{Action: "a", SubjectKind: "k", SubjectID: "i"}), ErrInvalid)
	require.ErrorIs(t, store.Record(ctx, Event{Actor: "a", SubjectKind: "k", SubjectID: "i"}), ErrInvalid)
	require.ErrorIs(t, store.Record(ctx, Event{Actor: "a", Action: "x", SubjectID: "i"}), ErrInvalid)
	require.ErrorIs(t, store.Record(ctx, Event{Actor: "a", Action: "x", SubjectKind: "k"}), ErrInvalid)
	require.ErrorIs(t, store.Record(ctx, Event{Actor: "a", Action: "x", SubjectKind: "k", SubjectID: "i", At: "yesterday"}), ErrInvalid)

	event := eventAt(time.Date(2026, 11, 18, 9, 0, 0, 0, time.UTC), "ada", "asset.publish", "asset", "button")
	event.ID = "stable-event-id"
	require.NoError(t, store.Record(ctx, event))
	require.NoError(t, store.Record(ctx, event), "an outbox retry with the same id is a no-op")
	got, err := store.List(ctx, Filter{}, 10)
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, "stable-event-id", got[0].ID)
	require.Equal(t, "2026-11-18", got[0].At[:10])
}

func TestStoreRequiresDatabase(t *testing.T) {
	var store *Store
	require.Error(t, store.Record(context.Background(), Event{}))
	_, err := store.List(context.Background(), Filter{}, 1)
	require.Error(t, err)
	_, err = store.Summary(context.Background(), 1)
	require.Error(t, err)

	unconfigured := NewSQLiteStore(nil)
	require.Error(t, unconfigured.Record(context.Background(), Event{}))
}
