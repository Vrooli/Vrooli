package hub_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"

	"notification-hub/internal/hub"
)

func insertPendingUrgent(t *testing.T, db *database.RoutedDB, recipient, urgency string) string {
	t.Helper()
	id := insertPending(t, db, recipient, "quiet", "public")
	_, err := db.ExecContext(context.Background(), `UPDATE notifications SET urgency=? WHERE id=?`, urgency, id)
	require.NoError(t, err)
	return id
}

// heldAt reports whether a notification of the given urgency is held by the
// recipient's quiet windows at the given moment.
func heldAt(t *testing.T, service *hub.Service, db *database.RoutedDB, urgency string, at time.Time) bool {
	t.Helper()
	service.SetClockForTest(fixedClock{now: at})
	id := insertPendingUrgent(t, db, "alice", urgency)
	require.NoError(t, service.Process(context.Background(), id))
	n, _, err := service.Get(context.Background(), id)
	require.NoError(t, err)
	return n.State == "held"
}

func utc(date, clock string) time.Time {
	parsed, err := time.Parse("2006-01-02 15:04", date+" "+clock)
	if err != nil {
		panic(err)
	}
	return parsed
}

func TestSetQuietWindowAcceptsOvernightAndStillRejectsEqualTimes(t *testing.T) {
	service, _ := testService(t)
	ctx := context.Background()
	window, err := service.SetQuietWindow(ctx, "alice", hub.QuietWindow{Weekday: 1, Start: "22:00", End: "07:00", Timezone: "UTC"})
	require.NoError(t, err)
	windows, err := service.ListQuietWindows(ctx, "alice")
	require.NoError(t, err)
	require.Len(t, windows, 1)
	require.Equal(t, window.ID, windows[0].ID)
	require.Equal(t, "07:00", windows[0].End)

	_, err = service.SetQuietWindow(ctx, "alice", hub.QuietWindow{Weekday: 1, Start: "07:00", End: "07:00", Timezone: "UTC"})
	require.ErrorIs(t, err, hub.ErrInvalidArgument)
	windows, err = service.ListQuietWindows(ctx, "alice")
	require.NoError(t, err)
	require.Equal(t, window.ID, windows[0].ID, "a rejected replacement keeps the stored window")
}

func TestQuietWindowHoldsAcrossMidnightAndReleasesInTheMorning(t *testing.T) {
	service, db := testService(t)
	ctx := context.Background()
	_, err := service.SetQuietWindow(ctx, "alice", hub.QuietWindow{Weekday: 0, Start: "22:00", End: "07:00", Timezone: "UTC"}) // Sunday night
	require.NoError(t, err)
	for _, tc := range []struct {
		name string
		at   time.Time
		want bool
	}{
		{"Sunday evening", utc("2026-03-08", "23:30"), true},
		{"Monday after midnight", utc("2026-03-09", "01:00"), true},
		{"Monday just before the end", utc("2026-03-09", "06:59"), true},
		{"Monday at the end", utc("2026-03-09", "07:00"), false},
		{"Sunday morning before the night", utc("2026-03-08", "03:00"), false},
		{"Tuesday morning is not covered", utc("2026-03-10", "03:00"), false},
	} {
		require.Equal(t, tc.want, heldAt(t, service, db, "normal", tc.at), tc.name)
	}
}

func TestQuietWindowSaturdayNightWrapsIntoSunday(t *testing.T) {
	service, db := testService(t)
	_, err := service.SetQuietWindow(context.Background(), "alice", hub.QuietWindow{Weekday: 6, Start: "23:00", End: "08:00", Timezone: "UTC"})
	require.NoError(t, err)
	require.True(t, heldAt(t, service, db, "normal", utc("2026-03-07", "23:30")))
	require.True(t, heldAt(t, service, db, "normal", utc("2026-03-08", "07:59")))
	require.False(t, heldAt(t, service, db, "normal", utc("2026-03-08", "08:00")))
}

func TestOvernightQuietWindowHonoursCriticalOverride(t *testing.T) {
	for _, override := range []bool{true, false} {
		service, db := testService(t)
		_, err := service.SetQuietWindow(context.Background(), "alice", hub.QuietWindow{Weekday: 1, Start: "22:00", End: "07:00", Timezone: "UTC", CriticalOverride: override})
		require.NoError(t, err)
		night, morning := utc("2026-03-02", "23:00"), utc("2026-03-03", "02:00")
		for _, at := range []time.Time{night, morning} {
			require.True(t, heldAt(t, service, db, "normal", at), "normal urgency is always held")
			require.Equal(t, !override, heldAt(t, service, db, "critical", at), "critical passes only when the window allows it (override=%v)", override)
		}
	}
}

// An overnight window from Monday and Tuesday's own window can both be active
// on Tuesday morning. Critical passes only when every active window allows it.
func TestOvernightWindowOverlappingNextDaysWindow(t *testing.T) {
	for _, tc := range []struct {
		name                               string
		overnightOverride, sameDayOverride bool
		wantCriticalHeld                   bool
	}{
		{"both allow the override", true, true, false},
		{"the overnight window forbids it", false, true, true},
		{"the same-day window forbids it", true, false, true},
	} {
		service, db := testService(t)
		ctx := context.Background()
		_, err := service.SetQuietWindow(ctx, "alice", hub.QuietWindow{Weekday: 1, Start: "22:00", End: "07:00", Timezone: "UTC", CriticalOverride: tc.overnightOverride})
		require.NoError(t, err)
		_, err = service.SetQuietWindow(ctx, "alice", hub.QuietWindow{Weekday: 2, Start: "05:00", End: "09:00", Timezone: "UTC", CriticalOverride: tc.sameDayOverride})
		require.NoError(t, err)
		windows, err := service.ListQuietWindows(ctx, "alice")
		require.NoError(t, err)
		require.Len(t, windows, 2, tc.name)

		// Only the overnight window is active at 03:00, only the same-day one at 08:00, both at 06:00.
		require.True(t, heldAt(t, service, db, "normal", utc("2026-03-03", "03:00")), tc.name)
		require.True(t, heldAt(t, service, db, "normal", utc("2026-03-03", "08:00")), tc.name)
		require.Equal(t, tc.wantCriticalHeld, heldAt(t, service, db, "critical", utc("2026-03-03", "06:00")), tc.name)
		require.False(t, heldAt(t, service, db, "normal", utc("2026-03-03", "09:00")), tc.name)
	}
}

func TestOvernightWindowIsReadInItsOwnTimezone(t *testing.T) {
	service, db := testService(t)
	_, err := service.SetQuietWindow(context.Background(), "alice", hub.QuietWindow{Weekday: 1, Start: "22:00", End: "07:00", Timezone: "Asia/Tokyo"})
	require.NoError(t, err)
	// Tuesday 02:00 in Tokyo is Monday 17:00 UTC.
	require.True(t, heldAt(t, service, db, "normal", utc("2026-03-02", "17:00")))
	// Tuesday 10:00 in Tokyo is Tuesday 01:00 UTC.
	require.False(t, heldAt(t, service, db, "normal", utc("2026-03-03", "01:00")))
}
