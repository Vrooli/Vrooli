package aisearch

import (
	"context"
	"errors"
	"testing"
	"time"

	"prompt-manager/internal/store"
)

type fakeCallStore struct {
	appended  []store.DiscoveryCall
	readResp  []store.DiscoveryCall
	appendErr error
}

func (f *fakeCallStore) Append(call store.DiscoveryCall) error {
	if f.appendErr != nil {
		return f.appendErr
	}
	f.appended = append(f.appended, call)
	return nil
}

func (f *fakeCallStore) ReadSince(window time.Duration) ([]store.DiscoveryCall, error) {
	return f.readResp, nil
}

func serviceWithCallStore(sink *fakeCallStore) *Service {
	s := &Service{threshold: 0.5}
	s.SetDiscoveryCallStore(sink)
	return s
}

func TestRecordDiscoveryCallUnderBudget(t *testing.T) {
	sink := &fakeCallStore{}
	s := serviceWithCallStore(sink)
	resp := &DiscoverResponse{
		Query:             "q",
		BudgetChars:       75000,
		TotalContentChars: 12000,
		BudgetStatus:      "under",
		Results: []DiscoverResult{
			{ID: "a", Score: 0.55, ContentChars: 6000, Source: "search", Type: "skill"},
			{ID: "b", Score: 0.52, ContentChars: 6000, Source: "topic", Type: "skill"},
		},
	}
	s.recordDiscoveryCall(context.Background(), resp, []string{"telemetry", "metrics"}, "skill", "moderate", 0, nil)

	if len(sink.appended) != 1 {
		t.Fatalf("expected 1 call recorded, got %d", len(sink.appended))
	}
	got := sink.appended[0]
	if got.Threshold != 0.5 || got.Type != "skill" || got.Complexity != "moderate" {
		t.Fatalf("unexpected call header: %#v", got)
	}
	if got.ReturnedCount != 2 || got.TrimmedCount != 0 || got.BudgetStatus != "under" {
		t.Fatalf("unexpected call counts: %#v", got)
	}
	if len(got.Queries) != 2 || got.Queries[0] != "telemetry" {
		t.Fatalf("expected queries to be recorded, got %#v", got.Queries)
	}
	if len(got.Results) != 2 || got.Results[0].ID != "a" || got.Results[0].Chars != 6000 {
		t.Fatalf("expected per-result detail, got %#v", got.Results)
	}
	if got.ClippedBelowThreshold != nil {
		t.Fatalf("expected nil clipped (not probed), got %#v", got.ClippedBelowThreshold)
	}
}

func TestRecordDiscoveryCallOverBudgetRecordsTrim(t *testing.T) {
	sink := &fakeCallStore{}
	s := serviceWithCallStore(sink)
	resp := &DiscoverResponse{
		Query:        "q",
		BudgetStatus: "over",
		Results:      []DiscoverResult{{ID: "a", Score: 0.6, ContentChars: 90000, Type: "skill"}},
	}
	s.recordDiscoveryCall(context.Background(), resp, []string{"q"}, "skill", "moderate", 3, nil)

	if len(sink.appended) != 1 {
		t.Fatalf("expected 1 call recorded, got %d", len(sink.appended))
	}
	if sink.appended[0].TrimmedCount != 3 || sink.appended[0].BudgetStatus != "over" {
		t.Fatalf("expected trimmedCount=3 over budget, got %#v", sink.appended[0])
	}
}

func TestRecordDiscoveryCallZeroResults(t *testing.T) {
	sink := &fakeCallStore{}
	s := serviceWithCallStore(sink)
	s.recordDiscoveryCall(context.Background(), &DiscoverResponse{Query: "q", Results: nil}, []string{"q"}, "all", "minor", 0, nil)
	if len(sink.appended) != 1 {
		t.Fatalf("expected zero-result call to still be recorded, got %d", len(sink.appended))
	}
	if sink.appended[0].ReturnedCount != 0 || sink.appended[0].Type != "all" {
		t.Fatalf("unexpected zero-result record: %#v", sink.appended[0])
	}
}

func TestRecordDiscoveryCallRecordsClipProbe(t *testing.T) {
	sink := &fakeCallStore{}
	s := serviceWithCallStore(sink)
	clipped := 4
	s.recordDiscoveryCall(context.Background(), &DiscoverResponse{Query: "q"}, []string{"q"}, "skill", "moderate", 0, &clipped)
	if len(sink.appended) != 1 {
		t.Fatalf("expected 1 call recorded, got %d", len(sink.appended))
	}
	if sink.appended[0].ClippedBelowThreshold == nil || *sink.appended[0].ClippedBelowThreshold != 4 {
		t.Fatalf("expected clipped=4 to be recorded, got %#v", sink.appended[0].ClippedBelowThreshold)
	}
}

func TestRecordDiscoveryCallSwallowsStoreError(t *testing.T) {
	sink := &fakeCallStore{appendErr: errors.New("disk full")}
	s := serviceWithCallStore(sink)
	// Must not panic or propagate — the discover response is unaffected.
	s.recordDiscoveryCall(context.Background(), &DiscoverResponse{Query: "q"}, []string{"q"}, "skill", "moderate", 0, nil)
	if len(sink.appended) != 0 {
		t.Fatalf("errored append should record nothing")
	}
}

func TestMaybeProbeClippingDisabledByDefault(t *testing.T) {
	sink := &fakeCallStore{}
	s := serviceWithCallStore(sink)
	// probeSample defaults to 0 → never probes, regardless of call count.
	for i := 0; i < 5; i++ {
		if got := s.maybeProbeClipping(context.Background(), []string{"q"}, nil, 10); got != nil {
			t.Fatalf("expected no probe when sampling disabled, got %#v", got)
		}
	}
}
