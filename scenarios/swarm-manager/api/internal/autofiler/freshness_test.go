package autofiler

import (
	"path/filepath"
	"testing"
	"time"
)

func TestReviewFreshnessStore(t *testing.T) {
	store := NewReviewFreshnessStorePath(filepath.Join(t.TempDir(), "review_freshness.json"))
	now := time.Date(2026, 7, 9, 12, 0, 0, 0, time.UTC)

	fresh, err := store.Fresh("alpha", time.Hour, now)
	if err != nil {
		t.Fatalf("Fresh initial: %v", err)
	}
	if fresh {
		t.Fatalf("Fresh initial = true, want false")
	}
	if err := store.Mark("alpha", now.Add(-30*time.Minute)); err != nil {
		t.Fatalf("Mark: %v", err)
	}
	fresh, err = store.Fresh("alpha", time.Hour, now)
	if err != nil {
		t.Fatalf("Fresh second: %v", err)
	}
	if !fresh {
		t.Fatalf("Fresh within window = false, want true")
	}
	fresh, err = store.Fresh("alpha", 15*time.Minute, now)
	if err != nil {
		t.Fatalf("Fresh stale: %v", err)
	}
	if fresh {
		t.Fatalf("Fresh beyond window = true, want false")
	}
}
