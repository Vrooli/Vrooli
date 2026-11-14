package hub_test

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"notification-hub/internal/hub"
)

// SQLite orders timestamps as text, so the persisted form has to make text order
// time order. RFC 3339 with trimmed zeros does not within one second.
func TestStoredTimeTextOrderIsTimeOrder(t *testing.T) {
	second := time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC)
	instants := []time.Time{
		second.Add(-time.Nanosecond), second,
		second.Add(time.Nanosecond), second.Add(time.Microsecond), second.Add(100 * time.Millisecond),
		second.Add(500 * time.Millisecond), second.Add(999999999 * time.Nanosecond),
		second.Add(time.Second), second.Add(time.Minute),
		time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 11, 14, 14, 0, 0, 0, time.FixedZone("EST+2", 2*3600)), // 12:00:00Z in another zone
	}
	texts := make([]string, 0, len(instants))
	for _, instant := range instants {
		text := hub.StoredTime(instant)
		require.Len(t, text, len("2026-11-14T12:00:00.000000000Z"), "every value is the same width")
		parsed, err := time.Parse(time.RFC3339Nano, text)
		require.NoError(t, err, "the readers' parse accepts it")
		require.True(t, parsed.Equal(instant), "%s names the same instant", text)
		texts = append(texts, text)
	}
	byTime := append([]time.Time(nil), instants...)
	sort.SliceStable(byTime, func(i, j int) bool { return byTime[i].Before(byTime[j]) })
	sort.Strings(texts)
	for i, instant := range byTime {
		require.Equal(t, hub.StoredTime(instant), texts[i], "position %d: sorting the text sorts the time", i)
	}
	// The case RFC 3339 got wrong: the second with no fraction sorts before half a second later.
	require.Less(t, hub.StoredTime(second), hub.StoredTime(second.Add(500*time.Millisecond)))
}

func TestListOrdersAsksRaisedWithinOneSecondByTime(t *testing.T) {
	l := newListAsks(t)
	ctx := context.Background()
	// Deadlines half a second apart, created half a second apart: the earlier deadline
	// is a whole second ("…:00Z" once trimmed) and the later one has a fraction.
	soon := l.ask(raise{age: 2 * time.Second, due: time.Hour})
	later := l.ask(raise{age: 1500 * time.Millisecond, due: time.Hour + 500*time.Millisecond})
	latest := l.ask(raise{age: 500 * time.Millisecond, due: time.Hour + time.Second})

	newest, err := l.service.ListAsksPage(ctx, "alice", hub.AskListOptions{OpenOnly: true})
	require.NoError(t, err)
	require.Equal(t, []string{latest.ID, later.ID, soon.ID}, askIDs(newest.Asks), "newest first, by the instant raised")
	attention, err := l.service.ListAsksPage(ctx, "alice", hub.AskListOptions{OpenOnly: true, Order: hub.AskOrderAttention})
	require.NoError(t, err)
	require.Equal(t, []string{soon.ID, later.ID, latest.ID}, askIDs(attention.Asks), "the earliest deadline first, by the instant it is due")

	// The same through a keyset page boundary that falls inside the second.
	var got []string
	cursor := ""
	for {
		page, err := l.service.ListAsksPage(ctx, "alice", hub.AskListOptions{OpenOnly: true, Limit: 1, Order: hub.AskOrderAttention, Cursor: cursor})
		require.NoError(t, err)
		got = append(got, askIDs(page.Asks)...)
		if cursor = page.NextCursor; cursor == "" {
			break
		}
	}
	require.Equal(t, []string{soon.ID, later.ID, latest.ID}, got)
}

func TestNormalizeStoredTimesRewritesOlderRowsAndIsIdempotent(t *testing.T) {
	l := newListAsks(t)
	ctx := context.Background()
	early := l.ask(raise{age: time.Minute, due: time.Hour})
	late := l.ask(raise{age: 30 * time.Second, due: time.Hour})
	none := l.ask(raise{age: 10 * time.Second, noDeadline: true})
	odd := l.ask(raise{age: 5 * time.Second, due: 2 * time.Hour})

	// Rows as they were written before the fixed form: trimmed fraction, a
	// fraction, an offset other than UTC, and a value that is not a timestamp.
	l.set(early.ID, `deadline = '2026-11-14T13:00:00Z', created_at = '2026-11-14T11:58:00Z', resolved_at = '2026-11-14T11:59:30.25Z'`)
	l.set(late.ID, `deadline = '2026-11-14T13:00:00.5Z', created_at = '2026-11-14T11:58:30.123456Z'`)
	l.set(odd.ID, `deadline = '2026-11-14T15:00:00+02:00', created_at = 'not a time'`)
	_, err := l.db.ExecContext(ctx, `UPDATE notifications SET created_at = '2026-11-14T11:58:00Z' WHERE id = ?`, early.NotificationID)
	require.NoError(t, err)

	hub.NormalizeStoredTimes(l.service, ctx)

	read := func(id, column string) string {
		var text string
		require.NoError(t, l.db.QueryRowContext(ctx, `SELECT `+column+` FROM asks WHERE id = ?`, id).Scan(&text))
		return text
	}
	require.Equal(t, "2026-11-14T13:00:00.000000000Z", read(early.ID, "deadline"))
	require.Equal(t, "2026-11-14T11:58:00.000000000Z", read(early.ID, "created_at"))
	require.Equal(t, "2026-11-14T11:59:30.250000000Z", read(early.ID, "resolved_at"))
	require.Equal(t, "2026-11-14T13:00:00.500000000Z", read(late.ID, "deadline"))
	require.Equal(t, "2026-11-14T11:58:30.123456000Z", read(late.ID, "created_at"))
	require.Equal(t, "2026-11-14T13:00:00.000000000Z", read(odd.ID, "deadline"), "an offset is converted to UTC, the same instant")
	require.Equal(t, "not a time", read(odd.ID, "created_at"), "a value that is not a timestamp is left as it is")
	require.Equal(t, "", read(none.ID, "deadline"), "no deadline stays none")
	require.Equal(t, "", read(early.ID, "resolution_published_at"), "empty stays empty")
	var notified string
	require.NoError(t, l.db.QueryRowContext(ctx, `SELECT created_at FROM notifications WHERE id = ?`, early.NotificationID).Scan(&notified))
	require.Equal(t, "2026-11-14T11:58:00.000000000Z", notified)

	// A second run changes nothing.
	before := map[string]string{}
	for _, ask := range []hub.AskRecord{early, late, none, odd} {
		before[ask.ID] = read(ask.ID, "deadline") + "|" + read(ask.ID, "created_at") + "|" + read(ask.ID, "updated_at")
	}
	hub.NormalizeStoredTimes(l.service, ctx)
	for _, ask := range []hub.AskRecord{early, late, none, odd} {
		require.Equal(t, before[ask.ID], read(ask.ID, "deadline")+"|"+read(ask.ID, "created_at")+"|"+read(ask.ID, "updated_at"))
	}

	// And the order over the migrated rows is the order of the instants: early and
	// odd share an instant to the second, late is half a second after.
	page, err := l.service.ListAsksPage(ctx, "alice", hub.AskListOptions{OpenOnly: true, Order: hub.AskOrderAttention})
	require.NoError(t, err)
	require.Len(t, page.Asks, 4)
	require.Equal(t, none.ID, page.Asks[3].ID, "no deadline last")
	require.Equal(t, late.ID, page.Asks[2].ID, "half a second after the other two")
}
