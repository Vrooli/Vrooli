package maintenance

import (
	"strings"
	"testing"
)

func TestParseDaysRejectsBadDays(t *testing.T) {
	_, err := parseDays("0")
	if err == nil || !strings.Contains(err.Error(), "--days must be greater than 0") {
		t.Fatalf("expected --days validation error, got %v", err)
	}
}

func TestParseDaysRejectsNonInteger(t *testing.T) {
	_, err := parseDays("soon")
	if err == nil || !strings.Contains(err.Error(), "--days must be an integer") {
		t.Fatalf("expected integer validation error, got %v", err)
	}
}

func TestParseDaysAcceptsPositiveInteger(t *testing.T) {
	got, err := parseDays("30")
	if err != nil {
		t.Fatalf("expected valid days, got %v", err)
	}
	if got != 30 {
		t.Fatalf("parseDays returned %d, want 30", got)
	}
}

func TestFormatBytes(t *testing.T) {
	cases := map[int64]string{
		0:       "0 B",
		512:     "512 B",
		1024:    "1.0 KB",
		1048576: "1.0 MB",
	}
	for in, want := range cases {
		if got := formatBytes(in); got != want {
			t.Errorf("formatBytes(%d) = %q, want %q", in, got, want)
		}
	}
}
