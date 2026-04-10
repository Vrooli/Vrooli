package sqlutil

import (
	"testing"
	"time"
)

func TestBoolToInt(t *testing.T) {
	tests := []struct {
		name string
		in   bool
		want int
	}{
		{"true returns 1", true, 1},
		{"false returns 0", false, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := BoolToInt(tc.in)
			if got != tc.want {
				t.Fatalf("BoolToInt(%v) = %d, want %d", tc.in, got, tc.want)
			}
		})
	}
}

func TestParseTime(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantZero bool
		wantYear int
	}{
		{
			name:     "valid timestamp",
			input:    "2025-03-15T10:30:45.123",
			wantYear: 2025,
		},
		{
			name:     "epoch-ish timestamp",
			input:    "2000-01-01T00:00:00.000",
			wantYear: 2000,
		},
		{
			name:     "empty string returns zero time",
			input:    "",
			wantZero: true,
		},
		{
			name:     "malformed returns zero time",
			input:    "not-a-timestamp",
			wantZero: true,
		},
		{
			name:     "wrong format returns zero time",
			input:    "2025-03-15 10:30:45",
			wantZero: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ParseTime(tc.input)
			if tc.wantZero {
				if !got.IsZero() {
					t.Fatalf("ParseTime(%q) = %v, want zero time", tc.input, got)
				}
				return
			}
			if got.IsZero() {
				t.Fatalf("ParseTime(%q) returned zero time, want year %d", tc.input, tc.wantYear)
			}
			if got.Year() != tc.wantYear {
				t.Fatalf("ParseTime(%q).Year() = %d, want %d", tc.input, got.Year(), tc.wantYear)
			}
		})
	}
}

func TestParseTime_RoundTrip(t *testing.T) {
	// Verify that formatting with TimestampFormat and parsing back yields the
	// same value (truncated to millisecond precision, which is what SQLite stores).
	original := time.Date(2025, 6, 15, 14, 30, 45, 123000000, time.UTC)
	formatted := original.Format(TimestampFormat)
	parsed := ParseTime(formatted)

	if !parsed.Equal(original) {
		t.Fatalf("round-trip failed: formatted=%q, parsed=%v, original=%v", formatted, parsed, original)
	}
}

func TestTimestampFormat(t *testing.T) {
	// The constant must match SQLite's strftime('%Y-%m-%dT%H:%M:%f','now') output.
	ref := time.Date(2006, 1, 2, 15, 4, 5, 0, time.UTC)
	got := ref.Format(TimestampFormat)
	want := "2006-01-02T15:04:05.000"
	if got != want {
		t.Fatalf("TimestampFormat reference = %q, want %q", got, want)
	}
}
