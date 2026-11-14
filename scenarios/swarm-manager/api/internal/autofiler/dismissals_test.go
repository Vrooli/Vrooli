package autofiler

import (
	"path/filepath"
	"testing"
)

func TestDismissalStoreRememberIsIdempotent(t *testing.T) {
	store := NewDismissalStorePath(filepath.Join(t.TempDir(), "dismissed_findings.json"))

	if dismissed, err := store.IsDismissed("gct:alpha:test"); err != nil || dismissed {
		t.Fatalf("initial IsDismissed = %v, %v; want false, nil", dismissed, err)
	}
	if err := store.Remember("gct:alpha:test", "fix/alpha", "operator dismissed"); err != nil {
		t.Fatalf("Remember first: %v", err)
	}
	if err := store.Remember("gct:alpha:test", "fix/other", "new reason"); err != nil {
		t.Fatalf("Remember second: %v", err)
	}

	dismissed, err := store.IsDismissed("gct:alpha:test")
	if err != nil {
		t.Fatalf("IsDismissed: %v", err)
	}
	if !dismissed {
		t.Fatalf("finding should be dismissed")
	}
	all, err := store.LoadAll()
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	got := all["gct:alpha:test"]
	if got.ItemRef != "fix/alpha" || got.Reason != "operator dismissed" || got.At == "" {
		t.Fatalf("dismissal = %+v, want original ref/reason with timestamp", got)
	}
	if count, err := store.Count(); err != nil || count != 1 {
		t.Fatalf("Count = %d, %v; want 1, nil", count, err)
	}
}
