package main

import (
	"encoding/json"
	"testing"
	"time"
)

func TestCache_GetMiss(t *testing.T) {
	c := NewCache()
	_, fresh, ok := c.Get("nope")
	if ok {
		t.Fatal("expected no hit")
	}
	if fresh {
		t.Fatal("expected not fresh")
	}
}

func TestCache_PutThenGetIsFresh(t *testing.T) {
	c := NewCache()
	env := Envelope{
		Data:     json.RawMessage(`{"foo":"bar"}`),
		CachedAt: time.Now().UTC(),
		Source:   "swarm",
	}
	c.Put("swarm:/api/v1/stats", env, time.Minute)
	got, fresh, ok := c.Get("swarm:/api/v1/stats")
	if !ok {
		t.Fatal("expected hit")
	}
	if !fresh {
		t.Fatal("expected fresh within TTL")
	}
	if !got.FromCache {
		t.Error("FromCache should flip on read")
	}
	if string(got.Data) != `{"foo":"bar"}` {
		t.Errorf("unexpected data: %s", got.Data)
	}
}

func TestCache_ExpiredEntryIsStillReturnedButNotFresh(t *testing.T) {
	c := NewCache()
	env := Envelope{
		Data:     json.RawMessage(`{}`),
		CachedAt: time.Now().Add(-2 * time.Hour).UTC(),
		Source:   "swarm",
	}
	c.Put("k", env, time.Minute)
	_, fresh, ok := c.Get("k")
	if !ok {
		t.Fatal("expected entry to still exist")
	}
	if fresh {
		t.Fatal("entry is way past TTL — should not be fresh")
	}
}

func TestCache_MarkStaleSetsStalenessTS(t *testing.T) {
	c := NewCache()
	c.Put("k", Envelope{CachedAt: time.Now().UTC(), Source: "swarm"}, time.Minute)

	staleAt := time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC)
	c.MarkStale("k", staleAt)

	got, _, ok := c.Get("k")
	if !ok {
		t.Fatal("expected entry present")
	}
	if got.StalenessTS == nil || !got.StalenessTS.Equal(staleAt) {
		t.Errorf("staleness TS not applied: %+v", got.StalenessTS)
	}
}

func TestCache_MarkStaleMissingKeyIsNoop(t *testing.T) {
	c := NewCache()
	c.MarkStale("missing", time.Now())
	if _, _, ok := c.Get("missing"); ok {
		t.Error("MarkStale should not create a new entry")
	}
}

func TestCache_PutNeverRollsBackLastGoodObservation(t *testing.T) {
	c := NewCache()
	observed := time.Now().UTC().Truncate(time.Millisecond)
	key := "swarm:/api/v1/stats:agent_stats"
	latest := Envelope{
		Data:          json.RawMessage(`{"value":2}`),
		CachedAt:      observed,
		ObservationAt: &observed,
		Source:        "swarm",
	}
	if !c.Put(key, latest, time.Minute) {
		t.Fatal("initial last-good observation was rejected")
	}

	olderAt := observed.Add(-time.Second)
	older := Envelope{
		Data:          json.RawMessage(`{"value":1}`),
		CachedAt:      observed.Add(time.Second),
		ObservationAt: &olderAt,
		Source:        "swarm",
	}
	if c.Put(key, older, time.Minute) {
		t.Fatal("out-of-order source response replaced the newer observation")
	}
	if c.Put(key, Envelope{Data: json.RawMessage(`{"value":0}`), CachedAt: observed.Add(2 * time.Second)}, time.Minute) {
		t.Fatal("untimestamped response replaced a qualified last-good observation")
	}

	c.MarkStale(key, observed.Add(2*time.Second))
	degraded, _, ok := c.Get(key)
	if !ok || string(degraded.Data) != `{"value":2}` || degraded.ObservationAt == nil || !degraded.ObservationAt.Equal(observed) {
		t.Fatalf("out-of-order refresh replaced last-good observation: %+v", degraded)
	}
	if degraded.StalenessTS == nil {
		t.Fatal("failed refresh did not retain stale marker")
	}

	recoveredAt := observed.Add(3 * time.Second)
	recovered := Envelope{
		Data:          json.RawMessage(`{"value":3}`),
		CachedAt:      recoveredAt,
		ObservationAt: &recoveredAt,
		Source:        "swarm",
	}
	if !c.Put(key, recovered, time.Minute) {
		t.Fatal("newer recovery observation was rejected")
	}
	current, fresh, ok := c.Get(key)
	if !ok || !fresh || string(current.Data) != `{"value":3}` || current.ObservationAt == nil || !current.ObservationAt.Equal(recoveredAt) {
		t.Fatalf("recovered observation not installed: %+v fresh=%v ok=%v", current, fresh, ok)
	}
	if current.StalenessTS != nil {
		t.Fatalf("recovery retained stale marker: %v", current.StalenessTS)
	}
}

func TestTTLFor(t *testing.T) {
	cases := []struct {
		src  UpstreamSource
		want time.Duration
	}{
		{SourceSwarm, 30 * time.Second},
		{SourceVrooli, 60 * time.Second},
		{SourceLPBS, 5 * time.Minute},
		{SourceNone, 30 * time.Second},
	}
	for _, tc := range cases {
		if got := TTLFor(tc.src); got != tc.want {
			t.Errorf("TTLFor(%q)=%v, want %v", tc.src, got, tc.want)
		}
	}
}
