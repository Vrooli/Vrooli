package filepreview

import (
	"errors"
	"testing"
	"time"
)

func newTestTarget() *Target {
	return &Target{
		ResolvedPath:    "/tmp/x.png",
		Basename:        "x.png",
		MIMEType:        "image/png",
		Kind:            KindImage,
		SizeBytes:       100,
		ModTimeUnixNano: 12345,
		CanDownload:     true,
		CanPreview:      true,
	}
}

func TestStoreIssueAndLookup(t *testing.T) {
	s := NewStore(time.Minute)
	id, expiry, err := s.Issue("sess-1", newTestTarget())
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	if id == "" || expiry.IsZero() {
		t.Fatalf("id=%q expiry=%v", id, expiry)
	}
	e, err := s.Lookup("sess-1", id)
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if e.ResolvedPath != "/tmp/x.png" || e.MIMEType != "image/png" || e.SizeBytes != 100 {
		t.Fatalf("entry mismatch: %+v", e)
	}
}

func TestStoreSessionBinding(t *testing.T) {
	s := NewStore(time.Minute)
	id, _, _ := s.Issue("sess-1", newTestTarget())
	if _, err := s.Lookup("sess-2", id); !errors.Is(err, ErrPreviewNotFound) {
		t.Fatalf("expected not-found for wrong session, got %v", err)
	}
}

func TestStoreUnknownID(t *testing.T) {
	s := NewStore(time.Minute)
	if _, err := s.Lookup("sess-1", "deadbeef"); !errors.Is(err, ErrPreviewNotFound) {
		t.Fatalf("expected not-found, got %v", err)
	}
}

func TestStoreExpiry(t *testing.T) {
	s := NewStore(time.Minute)
	now := time.Unix(1000, 0)
	s.now = func() time.Time { return now }
	id, _, _ := s.Issue("sess-1", newTestTarget())
	// Advance past expiry.
	now = now.Add(2 * time.Minute)
	if _, err := s.Lookup("sess-1", id); !errors.Is(err, ErrPreviewNotFound) {
		t.Fatalf("expected expired not-found, got %v", err)
	}
}

func TestStoreSweep(t *testing.T) {
	s := NewStore(time.Minute)
	now := time.Unix(1000, 0)
	s.now = func() time.Time { return now }
	s.Issue("sess-1", newTestTarget())
	now = now.Add(2 * time.Minute)
	s.Sweep()
	s.mu.Lock()
	n := len(s.byID)
	s.mu.Unlock()
	if n != 0 {
		t.Fatalf("expected 0 entries after sweep, got %d", n)
	}
}

func TestStoreUniqueIDs(t *testing.T) {
	s := NewStore(time.Minute)
	seen := map[string]bool{}
	for i := 0; i < 100; i++ {
		id, _, err := s.Issue("s", newTestTarget())
		if err != nil {
			t.Fatalf("issue: %v", err)
		}
		if seen[id] {
			t.Fatalf("duplicate id %q", id)
		}
		seen[id] = true
	}
}
