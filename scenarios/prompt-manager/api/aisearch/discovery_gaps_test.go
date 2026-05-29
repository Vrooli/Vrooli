package aisearch

import (
	"context"
	"errors"
	"testing"
	"time"

	"prompt-manager/store"
)

type fakeMissStore struct {
	appended  []store.DiscoveryMiss
	readResp  []store.DiscoveryMiss
	appendErr error
}

func (f *fakeMissStore) Append(miss store.DiscoveryMiss) error {
	if f.appendErr != nil {
		return f.appendErr
	}
	f.appended = append(f.appended, miss)
	return nil
}

func (f *fakeMissStore) ReadSince(window time.Duration) ([]store.DiscoveryMiss, error) {
	return f.readResp, nil
}

func serviceWithMissStore(sink *fakeMissStore) *Service {
	s := &Service{}
	s.SetDiscoveryMissStore(sink)
	return s
}

func TestRecordDiscoveryMissZeroResults(t *testing.T) {
	sink := &fakeMissStore{}
	s := serviceWithMissStore(sink)
	s.recordDiscoveryMiss(context.Background(), &DiscoverResponse{Query: "scrape pricing", Results: nil}, "all", "moderate")
	if len(sink.appended) != 1 {
		t.Fatalf("expected 1 miss recorded, got %d", len(sink.appended))
	}
	got := sink.appended[0]
	if got.Type != "all" || got.ResultCount != 0 || got.TopScore != 0 || got.Query != "scrape pricing" {
		t.Fatalf("unexpected miss record: %#v", got)
	}
}

func TestRecordDiscoveryMissSubThreshold(t *testing.T) {
	sink := &fakeMissStore{}
	s := serviceWithMissStore(sink)
	resp := &DiscoverResponse{Query: "q", Results: []DiscoverResult{{ID: "a", Score: 0.30}, {ID: "b", Score: 0.20}}}
	s.recordDiscoveryMiss(context.Background(), resp, "action", "")
	if len(sink.appended) != 1 {
		t.Fatalf("expected 1 miss recorded, got %d", len(sink.appended))
	}
	if sink.appended[0].TopScore != 0.30 || sink.appended[0].Type != "action" || sink.appended[0].ResultCount != 2 {
		t.Fatalf("unexpected sub-threshold miss: %#v", sink.appended[0])
	}
}

func TestRecordDiscoveryMissAboveThresholdSkips(t *testing.T) {
	sink := &fakeMissStore{}
	s := serviceWithMissStore(sink)
	resp := &DiscoverResponse{Query: "q", Results: []DiscoverResult{{ID: "a", Score: 0.90}}}
	s.recordDiscoveryMiss(context.Background(), resp, "skill", "")
	if len(sink.appended) != 0 {
		t.Fatalf("expected no miss for a useful result, got %d", len(sink.appended))
	}
}

func TestRecordDiscoveryMissSwallowsStoreError(t *testing.T) {
	sink := &fakeMissStore{appendErr: errors.New("disk full")}
	s := serviceWithMissStore(sink)
	// Must not panic or propagate — the discover response is unaffected.
	s.recordDiscoveryMiss(context.Background(), &DiscoverResponse{Query: "q"}, "all", "")
	if len(sink.appended) != 0 {
		t.Fatalf("errored append should record nothing")
	}
}

func TestDiscoveryGapsClustersAndWindows(t *testing.T) {
	sink := &fakeMissStore{readResp: []store.DiscoveryMiss{
		{Query: "Scrape competitor pricing", Type: "all", At: "2026-05-29T10:00:00Z"},
		{Query: "scrape   competitor   pricing", Type: "action", At: "2026-05-29T11:00:00Z"},
		{Query: "scrape competitor pricing", Type: "skill", At: "2026-05-29T09:00:00Z"},
		{Query: "translate document", Type: "skill", At: "2026-05-28T09:00:00Z"},
	}}
	s := serviceWithMissStore(sink)

	clusters, err := s.DiscoveryGaps(7*24*time.Hour, "all")
	if err != nil {
		t.Fatal(err)
	}
	if len(clusters) != 2 {
		t.Fatalf("expected 2 clusters, got %d: %#v", len(clusters), clusters)
	}
	// Largest cluster first; normalized query collapses whitespace + case.
	if clusters[0].Query != "scrape competitor pricing" || clusters[0].Count != 3 {
		t.Fatalf("unexpected top cluster: %#v", clusters[0])
	}
	if clusters[0].LastSeen != "2026-05-29T11:00:00Z" {
		t.Fatalf("expected lastSeen to be the newest At, got %q", clusters[0].LastSeen)
	}
}

func TestDiscoveryGapsTypeFilter(t *testing.T) {
	sink := &fakeMissStore{readResp: []store.DiscoveryMiss{
		{Query: "a", Type: "skill", At: "2026-05-29T10:00:00Z"},
		{Query: "b", Type: "action", At: "2026-05-29T10:00:00Z"},
	}}
	s := serviceWithMissStore(sink)
	clusters, err := s.DiscoveryGaps(7*24*time.Hour, "action")
	if err != nil {
		t.Fatal(err)
	}
	if len(clusters) != 1 || clusters[0].Query != "b" {
		t.Fatalf("expected only the action miss, got %#v", clusters)
	}
}

func TestDiscoveryGapsEmptyFilterReturnsAllTypes(t *testing.T) {
	// An empty type filter must NOT be treated as "skill" — it means all types.
	sink := &fakeMissStore{readResp: []store.DiscoveryMiss{
		{Query: "a", Type: "all", At: "2026-05-29T10:00:00Z"},
		{Query: "b", Type: "action", At: "2026-05-29T10:00:00Z"},
		{Query: "c", Type: "skill", At: "2026-05-29T10:00:00Z"},
	}}
	s := serviceWithMissStore(sink)
	clusters, err := s.DiscoveryGaps(7*24*time.Hour, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(clusters) != 3 {
		t.Fatalf("expected all 3 misses with empty filter, got %d: %#v", len(clusters), clusters)
	}
}

func TestParseSinceWindow(t *testing.T) {
	tests := []struct {
		in   string
		want time.Duration
	}{
		{"", 7 * 24 * time.Hour},
		{"7d", 7 * 24 * time.Hour},
		{"24h", 24 * time.Hour},
		{"30m", 30 * time.Minute},
		{"1d12h", 36 * time.Hour},
	}
	for _, tt := range tests {
		got, err := parseSinceWindow(tt.in)
		if err != nil {
			t.Fatalf("parseSinceWindow(%q) error: %v", tt.in, err)
		}
		if got != tt.want {
			t.Fatalf("parseSinceWindow(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
	if _, err := parseSinceWindow("nonsense"); err == nil {
		t.Fatalf("expected error for invalid window")
	}
}
