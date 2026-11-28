package main

import (
	"context"
	"encoding/json"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"command-center/upstream"
)

type countingObservationSource struct {
	calls atomic.Int32
	body  json.RawMessage
	err   error
}

func (s *countingObservationSource) Name() string { return "swarm" }
func (s *countingObservationSource) Fetch(context.Context, string) (json.RawMessage, error) {
	s.calls.Add(1)
	if s.err != nil {
		return nil, s.err
	}
	return append(json.RawMessage(nil), s.body...), nil
}

func TestReadingsShareEndpointRefreshAndRecoverEachMetric(t *testing.T) {
	observed := time.Now().UTC().Add(-30 * time.Second).Truncate(time.Second)
	client := &countingObservationSource{body: json.RawMessage(`{"observed_at":"` + observed.Format(time.RFC3339) + `","throughput":{"completed_last_7_days":11,"created_last_7_days":22}}`)}
	metrics := []MetricEntry{
		{ID: "swarm_throughput", Kind: "number", Unit: "count", Coverage: CoverageNow, UpstreamSource: SourceSwarm, Source: SourceBinding{Binding: "swarm-manager", Read: "/api/v1/stats", Selector: "swarm_throughput", IntegrationID: "swarm-manager", TTLSeconds: 60}},
		{ID: "throughput_stats", Kind: "number", Unit: "count", Coverage: CoverageNow, UpstreamSource: SourceSwarm, Source: SourceBinding{Binding: "swarm-manager", Read: "/api/v1/stats", Selector: "throughput_stats", IntegrationID: "swarm-manager", TTLSeconds: 60}},
	}
	server := NewServer(&Registry{Dashboards: map[string][]MetricEntry{"mission-control": metrics}})
	server.swarm = client
	read := func() []MetricEntry {
		readings, _ := server.readings(context.Background(), metrics)
		return readings
	}

	first := read()
	if client.calls.Load() != 1 || first[0].Trust != TrustValid || first[0].Value != float64(11) || first[1].Trust != TrustValid || first[1].Value != float64(22) || first[0].ObservedAt == nil || first[1].ObservedAt == nil || !first[0].ObservedAt.Equal(*first[1].ObservedAt) {
		t.Fatalf("first readings=%+v calls=%d, want one shared fresh response with both values", first, client.calls.Load())
	}

	cached := read()
	if client.calls.Load() != 1 || cached[0].Value != float64(11) || cached[1].Value != float64(22) {
		t.Fatalf("fresh-cache readings=%+v calls=%d, want no redundant source fetch", cached, client.calls.Load())
	}

	expiredKey := observationCacheKey(SourceSwarm, "/api/v1/stats", metrics[1].ID)
	server.cache.mu.Lock()
	entry := server.cache.entries[expiredKey]
	entry.env.CachedAt = time.Now().UTC().Add(-time.Minute - time.Second)
	server.cache.entries[expiredKey] = entry
	server.cache.mu.Unlock()
	client.err = errors.New("source outage")
	degraded := read()
	if client.calls.Load() != 2 || degraded[0].Trust != TrustValid || degraded[0].Value != float64(11) || degraded[1].Trust != TrustCached || degraded[1].Value != float64(22) || degraded[1].TrustReason == "" {
		t.Fatalf("degraded readings=%+v calls=%d, want one failed refresh and independent last-good values", degraded, client.calls.Load())
	}

	recoveredAt := time.Now().UTC().Add(-time.Second)
	client.err = nil
	client.body = json.RawMessage(`{"observed_at":"` + recoveredAt.Format(time.RFC3339Nano) + `","throughput":{"completed_last_7_days":12,"created_last_7_days":23}}`)
	recovered := read()
	if client.calls.Load() != 3 || recovered[0].Trust != TrustValid || recovered[0].Value != float64(12) || recovered[1].Trust != TrustValid || recovered[1].Value != float64(23) || recovered[0].ObservedAt == nil || !recovered[0].ObservedAt.Equal(recoveredAt) {
		t.Fatalf("recovered readings=%+v calls=%d, want both fresh values from one recovered response", recovered, client.calls.Load())
	}
}

func TestCachedEndpointSnapshotStillSeedsMissingMetricCache(t *testing.T) {
	newerAt := time.Now().UTC().Add(-10 * time.Second)
	client := &countingObservationSource{body: json.RawMessage(`{"observed_at":"` + time.Now().UTC().Add(-20*time.Second).Format(time.RFC3339Nano) + `","throughput":{"completed_last_7_days":11}}`)}
	metric := MetricEntry{
		ID: "swarm_throughput", Kind: "number", Unit: "count", Coverage: CoverageNow, UpstreamSource: SourceSwarm,
		Source: SourceBinding{Binding: "swarm-manager", Read: "/api/v1/stats", Selector: "swarm_throughput", IntegrationID: "swarm-manager", TTLSeconds: 60},
	}
	server := NewServer(&Registry{Dashboards: map[string][]MetricEntry{"mission-control": {metric}}})
	server.swarm = client
	server.cache.CommitEndpoint(observationEndpointCacheKey(SourceSwarm, metric.Source.Read), Envelope{
		Data:     json.RawMessage(`{"observed_at":"` + newerAt.Format(time.RFC3339Nano) + `","throughput":{"completed_last_7_days":22}}`),
		CachedAt: time.Now().UTC(), Source: string(SourceSwarm), ObservationAt: &newerAt,
	})

	read := func() MetricEntry {
		readings, _ := server.readings(context.Background(), []MetricEntry{metric})
		return readings[0]
	}
	first := read()
	if first.Trust != TrustValid || first.Value != float64(22) || first.ObservedAt == nil || !first.ObservedAt.Equal(newerAt) || client.calls.Load() != 1 {
		t.Fatalf("first reading=%+v calls=%d, want cached endpoint value 22", first, client.calls.Load())
	}
	client.err = errors.New("source outage")
	second := read()
	if second.Trust != TrustValid || second.Value != float64(22) || client.calls.Load() != 1 {
		t.Fatalf("second reading=%+v calls=%d, want committed metric cache to avoid outage refetch", second, client.calls.Load())
	}
}

func TestUnavailableClientKeepsFreshAndStaleMetricSnapshots(t *testing.T) {
	observed := time.Now().UTC().Add(-30 * time.Second).Truncate(time.Second)
	client := &countingObservationSource{body: json.RawMessage(`{"observed_at":"` + observed.Format(time.RFC3339) + `","throughput":{"completed_last_7_days":11,"created_last_7_days":22}}`)}
	metrics := []MetricEntry{
		{ID: "swarm_throughput", Kind: "number", Unit: "count", Coverage: CoverageNow, UpstreamSource: SourceSwarm, Source: SourceBinding{Binding: "swarm-manager", Read: "/api/v1/stats", Selector: "swarm_throughput", TTLSeconds: 60}},
		{ID: "throughput_stats", Kind: "number", Unit: "count", Coverage: CoverageNow, UpstreamSource: SourceSwarm, Source: SourceBinding{Binding: "swarm-manager", Read: "/api/v1/stats", Selector: "throughput_stats", TTLSeconds: 60}},
	}
	server := NewServer(&Registry{Dashboards: map[string][]MetricEntry{"mission-control": metrics}})
	server.swarm = client
	read := func() []MetricEntry {
		readings, _ := server.readings(context.Background(), metrics)
		return readings
	}
	if first := read(); client.calls.Load() != 1 || first[0].Value != float64(11) || first[1].Value != float64(22) {
		t.Fatalf("initial readings=%+v calls=%d", first, client.calls.Load())
	}
	staleKey := observationCacheKey(SourceSwarm, "/api/v1/stats", metrics[1].ID)
	server.cache.mu.Lock()
	entry := server.cache.entries[staleKey]
	entry.env.CachedAt = time.Now().UTC().Add(-time.Minute - time.Second)
	server.cache.entries[staleKey] = entry
	server.cache.mu.Unlock()
	server.providers[SourceSwarm] = upstreamProvider{client: func() upstream.Client { return nil }}
	degraded := read()
	if client.calls.Load() != 1 || degraded[0].Trust != TrustValid || degraded[0].Value != float64(11) || degraded[1].Trust != TrustCached || degraded[1].Value != float64(22) || degraded[1].TrustReason == "" {
		t.Fatalf("degraded readings=%+v calls=%d, want fresh metric preserved and stale metric labeled with last-good value", degraded, client.calls.Load())
	}
}
