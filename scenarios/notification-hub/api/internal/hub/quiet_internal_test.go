package hub

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// The week of Monday 2026-03-02 through Sunday 2026-03-08, so every date
// below has a known weekday.
const (
	sunday   = 0
	monday   = 1
	saturday = 6
)

func quietAt(t *testing.T, zone, date, clock string) time.Time {
	t.Helper()
	location, err := time.LoadLocation(zone)
	require.NoError(t, err)
	parsed, err := time.ParseInLocation("2006-01-02 15:04", date+" "+clock, location)
	require.NoError(t, err)
	return parsed
}

func TestQuietWindowCovers(t *testing.T) {
	for _, tc := range []struct {
		name       string
		weekday    int
		start, end string
		date       string
		clock      string
		want       bool
	}{
		{"same day inside", monday, "13:00", "15:00", "2026-03-02", "14:00", true},
		{"same day start is inclusive", monday, "13:00", "15:00", "2026-03-02", "13:00", true},
		{"same day end is exclusive", monday, "13:00", "15:00", "2026-03-02", "15:00", false},
		{"same day is not repeated on the next day", monday, "13:00", "15:00", "2026-03-03", "14:00", false},
		{"same day is not repeated on the day before", monday, "13:00", "15:00", "2026-03-01", "14:00", false},

		{"overnight is not active before its start", monday, "22:00", "07:00", "2026-03-02", "21:59", false},
		{"overnight start is inclusive", monday, "22:00", "07:00", "2026-03-02", "22:00", true},
		{"overnight covers the rest of its own day", monday, "22:00", "07:00", "2026-03-02", "23:59", true},
		{"overnight covers the next morning", monday, "22:00", "07:00", "2026-03-03", "00:00", true},
		{"overnight covers the last minute before its end", monday, "22:00", "07:00", "2026-03-03", "06:59", true},
		{"overnight end is exclusive", monday, "22:00", "07:00", "2026-03-03", "07:00", false},
		{"overnight is over by the afternoon after", monday, "22:00", "07:00", "2026-03-03", "12:00", false},
		{"overnight does not cover its own early morning", monday, "22:00", "07:00", "2026-03-02", "06:00", false},
		{"overnight does not repeat two days later", monday, "22:00", "07:00", "2026-03-04", "06:00", false},

		{"sunday night into monday morning: the evening", sunday, "22:00", "07:00", "2026-03-08", "23:00", true},
		{"sunday night into monday morning: the morning", sunday, "22:00", "07:00", "2026-03-09", "03:00", true},
		{"sunday night is not active on the morning before it", sunday, "22:00", "07:00", "2026-03-08", "03:00", false},
		{"saturday night into sunday morning", saturday, "22:00", "07:00", "2026-03-08", "03:00", true},
		{"saturday night does not run into monday", saturday, "22:00", "07:00", "2026-03-09", "03:00", false},

		{"one-minute overnight window", monday, "23:59", "00:01", "2026-03-03", "00:00", true},
		{"one-minute overnight window is over after", monday, "23:59", "00:01", "2026-03-03", "00:01", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			from, until, ok := quietMinutes(tc.start, tc.end)
			require.True(t, ok)
			require.Equal(t, tc.want, quietWindowCovers(tc.weekday, from, until, quietAt(t, "UTC", tc.date, tc.clock)))
		})
	}
}

// The window's weekday is its own timezone's weekday, not the server's.
func TestQuietWindowCoversUsesTheWindowsTimezone(t *testing.T) {
	from, until, ok := quietMinutes("22:00", "07:00")
	require.True(t, ok)
	// Monday 23:00 in Tokyo is Monday 14:00 UTC.
	tokyoEvening := quietAt(t, "Asia/Tokyo", "2026-03-02", "23:00")
	require.True(t, quietWindowCovers(monday, from, until, tokyoEvening.In(mustZone(t, "Asia/Tokyo"))))
	// The same instant read as UTC is Monday afternoon and outside the window.
	require.False(t, quietWindowCovers(monday, from, until, tokyoEvening.UTC()))
}

func mustZone(t *testing.T, name string) *time.Location {
	t.Helper()
	location, err := time.LoadLocation(name)
	require.NoError(t, err)
	return location
}

func TestQuietMinutesRejectsLooseClockText(t *testing.T) {
	for _, tc := range [][2]string{{"7:00", "08:00"}, {"07:00", "24:00"}, {"07:00", "8:00am"}, {"", "08:00"}, {"07:00", "07:60"}} {
		_, _, ok := quietMinutes(tc[0], tc[1])
		require.False(t, ok, "%q to %q", tc[0], tc[1])
	}
}
