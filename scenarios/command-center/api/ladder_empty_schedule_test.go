package main

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

type ladderSequenceClient struct {
	bodies []json.RawMessage
	index  int
}

func (c *ladderSequenceClient) Name() string { return "offer-desk" }

func (c *ladderSequenceClient) Fetch(context.Context, string) (json.RawMessage, error) {
	if c.index >= len(c.bodies) {
		return nil, fmt.Errorf("unexpected ladder fetch %d", c.index+1)
	}
	body := append(json.RawMessage(nil), c.bodies[c.index]...)
	c.index++
	return body, nil
}

func TestLadderAcceptsEmptyScheduleWithUnscheduledWork(t *testing.T) {
	now := time.Now().UTC().Add(-5 * time.Second)
	m := readLadder(t, json.RawMessage(fmt.Sprintf(
		`{"generatedAt":%q,"unscheduled":[{"name":"treasury","status":"IDEA"}]}`,
		now.Format(time.RFC3339Nano),
	)))
	if m.Trust != TrustValid || m.Value != float64(0) || m.Ladder == nil {
		t.Fatalf("empty schedule trust/value/ladder = %s/%v/%+v", m.Trust, m.Value, m.Ladder)
	}
	if len(m.Rows) != 0 || len(m.Ladder.Rungs) != 0 || len(m.Ladder.Unscheduled) != 1 || m.Ladder.Unscheduled[0].Name != "treasury" {
		t.Fatalf("empty schedule projection lost producer data: rows=%+v ladder=%+v", m.Rows, m.Ladder)
	}
}

func TestLadderInvalidRefreshRetainsEmptyScheduleAndRecovers(t *testing.T) {
	generated := func() string { return time.Now().UTC().Add(-5 * time.Second).Format(time.RFC3339Nano) }
	client := &ladderSequenceClient{bodies: []json.RawMessage{
		json.RawMessage(fmt.Sprintf(`{"generatedAt":%q,"unscheduled":[{"name":"treasury","status":"IDEA"}]}`, generated())),
		json.RawMessage(fmt.Sprintf(`{"generatedAt":%q,"entries":"not-an-array"}`, generated())),
		ladderFixture(t, func() *time.Time { value := time.Now().UTC().Add(-5 * time.Second); return &value }()),
	}}
	s := NewServer(testRegistry())
	s.offer = client
	read := func() MetricEntry {
		t.Helper()
		out, _ := s.readings(context.Background(), []MetricEntry{ladderMetric()})
		return out[0]
	}
	first := read()
	if first.Trust != TrustValid || first.Value != float64(0) || len(first.Ladder.Unscheduled) != 1 {
		t.Fatalf("initial empty schedule = %+v", first)
	}
	cacheKey := observationCacheKey(SourceOffer, ladderMetric().Source.Read, ladderMetric().ID)
	expireLadderCache(t, s, cacheKey)
	invalid := read()
	if invalid.Trust != TrustCached || invalid.TrustReason == "" || invalid.Value != float64(0) || len(invalid.Ladder.Unscheduled) != 1 {
		t.Fatalf("invalid refresh did not retain empty last-good schedule: %+v", invalid)
	}
	expireLadderCache(t, s, cacheKey)
	recovered := read()
	if recovered.Trust != TrustValid || recovered.Value != float64(14) || recovered.Ladder == nil || len(recovered.Ladder.Unscheduled) != 1 {
		t.Fatalf("fresh schedule did not replace retained empty schedule: %+v", recovered)
	}
}

func expireLadderCache(t *testing.T, s *Server, key string) {
	t.Helper()
	s.cache.mu.Lock()
	defer s.cache.mu.Unlock()
	e, ok := s.cache.entries[key]
	if !ok {
		t.Fatalf("last-good ladder cache %q missing", key)
	}
	e.env.CachedAt = time.Now().Add(-e.ttl - time.Second)
	s.cache.entries[key] = e
}
