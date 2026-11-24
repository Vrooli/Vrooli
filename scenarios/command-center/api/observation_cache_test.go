package main

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"testing"
	"time"
)

type delayedObservationSource struct {
	calls        atomic.Int32
	firstStarted chan struct{}
	releaseFirst chan struct{}
	firstBody    json.RawMessage
	secondBody   json.RawMessage
}

func (s *delayedObservationSource) Name() string { return "swarm" }
func (s *delayedObservationSource) Fetch(context.Context, string) (json.RawMessage, error) {
	if s.calls.Add(1) == 1 {
		close(s.firstStarted)
		<-s.releaseFirst
		return append(json.RawMessage(nil), s.firstBody...), nil
	}
	return append(json.RawMessage(nil), s.secondBody...), nil
}

func TestConcurrentOlderResponseUsesLatestQualifiedObservation(t *testing.T) {
	olderAt := time.Now().UTC().Add(-2 * time.Second)
	newerAt := time.Now().UTC().Add(-time.Second)
	client := &delayedObservationSource{
		firstStarted: make(chan struct{}),
		releaseFirst: make(chan struct{}),
		firstBody:    json.RawMessage(`{"observed_at":"` + olderAt.Format(time.RFC3339Nano) + `","throughput":{"completed_last_7_days":11,"created_last_7_days":33}}`),
		secondBody:   json.RawMessage(`{"observed_at":"` + newerAt.Format(time.RFC3339Nano) + `","throughput":{"completed_last_7_days":22,"created_last_7_days":44}}`),
	}
	metrics := []MetricEntry{{
		ID: "swarm_throughput", Kind: "number", Unit: "count", Coverage: CoverageNow, UpstreamSource: SourceSwarm,
		Source: SourceBinding{Binding: "swarm-manager", Read: "/api/v1/stats", Selector: "swarm_throughput", IntegrationID: "swarm-manager", TTLSeconds: 60},
	}, {
		ID: "throughput_stats", Kind: "number", Unit: "count", Coverage: CoverageNow, UpstreamSource: SourceSwarm,
		Source: SourceBinding{Binding: "swarm-manager", Read: "/api/v1/stats", Selector: "throughput_stats", IntegrationID: "swarm-manager", TTLSeconds: 60},
	}}
	server := NewServer(&Registry{Dashboards: map[string][]MetricEntry{"mission-control": metrics}})
	server.swarm = client
	type result struct {
		readings []MetricEntry
		metadata sourceMetadata
	}
	read := func() result {
		readings, sources := server.readings(context.Background(), metrics)
		return result{readings: readings, metadata: sources[string(SourceSwarm)]}
	}
	olderResult := make(chan result, 1)
	go func() { olderResult <- read() }()
	<-client.firstStarted

	newer := read()
	if len(newer.readings) != 2 || newer.readings[0].Trust != TrustValid || newer.readings[0].Value != float64(22) || newer.readings[1].Trust != TrustValid || newer.readings[1].Value != float64(44) || newer.readings[0].ObservedAt == nil || newer.readings[1].ObservedAt == nil || !newer.readings[0].ObservedAt.Equal(*newer.readings[1].ObservedAt) {
		t.Fatalf("newer response=%+v, want both fresh values 22/44 from one observation", newer.readings)
	}
	close(client.releaseFirst)
	older := <-olderResult
	if len(older.readings) != 2 || older.readings[0].Trust != TrustValid || older.readings[0].Value != float64(22) || older.readings[1].Trust != TrustValid || older.readings[1].Value != float64(44) || !older.metadata.FromCache || older.readings[0].ObservedAt == nil || older.readings[1].ObservedAt == nil || !older.readings[0].ObservedAt.Equal(*older.readings[1].ObservedAt) {
		t.Fatalf("late older response=%+v metadata=%+v, want newest values 22/44 from one cached observation", older.readings, older.metadata)
	}
	key := observationCacheKey(SourceSwarm, metrics[0].Source.Read, metrics[0].ID)
	lastGood, _, ok := server.cache.Get(key)
	if !ok || lastGood.ObservationAt == nil || !lastGood.ObservationAt.Equal(newerAt) {
		t.Fatalf("last-good cache=%+v ok=%t, want newer producer timestamp %s", lastGood, ok, newerAt)
	}
}
