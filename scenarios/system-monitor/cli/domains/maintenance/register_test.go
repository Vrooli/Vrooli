package maintenance

import (
	"strings"
	"testing"
)

// The --confirm guard runs before any API call, so a nil ScenarioApp is safe
// here: the functions must return the confirmation error first.

func TestRetentionApply_RequiresConfirm(t *testing.T) {
	err := runRetentionApply(nil, []string{"--days", "30"})
	if err == nil || !strings.Contains(err.Error(), "--confirm is required") {
		t.Fatalf("expected --confirm required error, got %v", err)
	}
}

func TestCompactApply_RequiresConfirm(t *testing.T) {
	err := runCompactApply(nil, nil)
	if err == nil || !strings.Contains(err.Error(), "--confirm is required") {
		t.Fatalf("expected --confirm required error, got %v", err)
	}
}

func TestRetentionPreview_RejectsBadDays(t *testing.T) {
	err := runRetentionPreview(nil, []string{"--days", "0"})
	if err == nil || !strings.Contains(err.Error(), "--days must be greater than 0") {
		t.Fatalf("expected --days validation error, got %v", err)
	}
}

func TestRetention_UnknownAction(t *testing.T) {
	if err := runRetention(nil, []string{"bogus"}); err == nil {
		t.Fatal("expected usage error for unknown action")
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
