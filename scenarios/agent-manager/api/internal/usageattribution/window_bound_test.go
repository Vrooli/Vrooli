package usageattribution

import (
	"strings"
	"testing"
	"time"
)

func TestParseWindowBound(t *testing.T) {
	now := time.Date(2026, 11, 20, 12, 30, 0, 0, time.UTC)
	cases := []struct {
		in   string
		want time.Time
	}{
		{"", time.Time{}},
		{"2026-10-01T08:00:00Z", time.Date(2026, 10, 1, 8, 0, 0, 0, time.UTC)},
		{"2026-10-01T08:00:00.123+02:00", time.Date(2026, 10, 1, 6, 0, 0, 123000000, time.UTC)},
		{"2026-10-01", time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)},
		{" 7d ", now.Add(-7 * 24 * time.Hour)},
		{"36h", now.Add(-36 * time.Hour)},
		{"1h30m", now.Add(-90 * time.Minute)},
	}
	for _, c := range cases {
		got, err := ParseWindowBound("since", c.in, now)
		if err != nil {
			t.Fatalf("%q: %v", c.in, err)
		}
		if !got.Equal(c.want) {
			t.Fatalf("%q: got %s want %s", c.in, got, c.want)
		}
	}
}

func TestParseWindowBoundRejects(t *testing.T) {
	for _, in := range []string{"yesterday", "0d", "-3d", "-2h", "2026-13-01", "7days"} {
		_, err := ParseWindowBound("until", in, time.Now())
		if err == nil {
			t.Fatalf("%q: expected an error", in)
		}
		if !strings.Contains(err.Error(), "until") || !strings.Contains(err.Error(), "YYYY-MM-DD") {
			t.Fatalf("%q: error does not name the field and accepted forms: %v", in, err)
		}
	}
}
