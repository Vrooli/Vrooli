package main

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	capabilityregistry "github.com/vrooli/vrooli/packages/capability-registry-go"
)

type recoveringUpstream struct {
	body json.RawMessage
	err  error
}

func (c *recoveringUpstream) Name() string { return "vrooli-core" }
func (c *recoveringUpstream) Fetch(context.Context, string) (json.RawMessage, error) {
	if c.err != nil {
		return nil, c.err
	}
	return append(json.RawMessage(nil), c.body...), nil
}

func TestObservationTimeAcceptsProducerTimestampSpellings(t *testing.T) {
	for _, key := range []string{"observedAt", "observed_at", "observationTime", "timestamp", "generatedAt", "generated_at"} {
		t.Run(key, func(t *testing.T) {
			raw, err := json.Marshal(map[string]string{key: "2026-11-21T10:00:00Z"})
			if err != nil {
				t.Fatal(err)
			}
			if got := observationTime(raw); got == nil || got.Format(time.RFC3339) != "2026-11-21T10:00:00Z" {
				t.Fatalf("observationTime(%s) = %v, want producer timestamp", key, got)
			}
		})
	}
}

func TestStructuredAndScalarReadingsShareFreshnessPolicy(t *testing.T) {
	ttl := time.Minute
	for _, tc := range []struct {
		name string
		age  time.Duration
		err  error
	}{
		{name: "fresh", age: 10 * time.Second},
		{name: "cached", age: 90 * time.Second, err: errors.New("source unavailable")},
		{name: "expired", age: 3 * time.Minute, err: errors.New("source unavailable")},
		{name: "future", age: -time.Second},
	} {
		t.Run(tc.name, func(t *testing.T) {
			observed := time.Now().UTC().Add(-tc.age)
			want, wantReason := qualifyProducerTime(observed, ttl, tc.err)
			structured := MetricEntry{TTLSeconds: int(ttl.Seconds())}
			structured.trustFromProducerTime(Envelope{ObservationAt: &observed}, tc.err)
			if structured.Trust != want || structured.TrustReason != wantReason {
				t.Fatalf("structured trust = %s (%q), scalar policy = %s (%q)", structured.Trust, structured.TrustReason, want, wantReason)
			}
		})
	}
}

func TestReadingsRecoverAcrossFreshCachedExpiredAndRecoveredStates(t *testing.T) {
	ttl := time.Minute
	sources := []struct {
		name        string
		source      UpstreamSource
		integration string
	}{
		{name: "control-plane", source: SourceVrooli, integration: "vrooli-core"},
		{name: "swarm", source: SourceSwarm, integration: "swarm-manager"},
		{name: "lpbs", source: SourceLPBS, integration: "landing-page-business-suite"},
		{name: "offer-desk", source: SourceOffer, integration: "offer-desk"},
		{name: "deployment-manager", source: SourceDeploy, integration: "deployment-manager"},
	}
	for _, tc := range sources {
		t.Run(tc.name, func(t *testing.T) {
			path := "/feed"
			if tc.source == SourceVrooli {
				path = "/scenarios"
			}
			metric := MetricEntry{
				ID: "total_scenarios", Label: "Total scenarios", Kind: "number", Unit: "count",
				Coverage: CoverageNow, UpstreamSource: tc.source,
				Source: SourceBinding{Binding: string(tc.source), Read: path, Selector: "total_scenarios", IntegrationID: tc.integration, FeatureID: "total_scenarios", TTLSeconds: int(ttl.Seconds())},
			}
			client := &recoveringUpstream{}
			s := NewServer(&Registry{Dashboards: map[string][]MetricEntry{"hive": {metric}}})
			s.integrationRegistry = capabilityregistry.New(nil, nil, 5*time.Second)
			switch tc.source {
			case SourceVrooli:
				s.vrooli = client
			case SourceSwarm:
				s.swarm = client
			case SourceLPBS:
				s.lpbs = client
			case SourceOffer:
				s.offer = client
			case SourceDeploy:
				s.deploy = client
			}
			key := observationCacheKey(tc.source, path, metric.ID)
			read := func() (MetricEntry, sourceMetadata) {
				metrics, metadata := s.readings(context.Background(), []MetricEntry{metric})
				return metrics[0], metadata[string(tc.source)]
			}
			payload := func(observed time.Time, count int) json.RawMessage {
				rows := make([]any, count)
				for i := range rows {
					rows[i] = map[string]any{"status": "running", "health_status": "running", "ports": map[string]any{}}
				}
				body, err := json.Marshal(map[string]any{"observed_at": observed.UTC().Format(time.RFC3339Nano), "total_scenarios": count, "data": rows})
				if err != nil {
					t.Fatal(err)
				}
				return body
			}
			setCachedAge := func(producerAge, cacheAge time.Duration) {
				t.Helper()
				observed := time.Now().UTC().Add(-producerAge)
				s.cache.mu.Lock()
				entry := s.cache.entries[key]
				entry.env.CachedAt = time.Now().UTC().Add(-cacheAge)
				entry.env.ObservationAt = &observed
				entry.env.Data = payload(observed, 2)
				s.cache.entries[key] = entry
				s.cache.mu.Unlock()
			}

			client.body = payload(time.Now().UTC(), 2)
			fresh, metadata := read()
			if fresh.Trust != TrustValid || fresh.Value != float64(2) || metadata.FromCache || fresh.ObservedAt == nil {
				t.Fatalf("fresh reading=%+v metadata=%+v, want current valid value from producer", fresh, metadata)
			}

			setCachedAge(30*time.Second, ttl+time.Second)
			client.body = json.RawMessage(`{"data":[{"name":"unqualified"}]}`)
			unqualified, metadata := read()
			if unqualified.Trust != TrustCached || unqualified.Value != float64(2) || metadata.StalenessTS == nil {
				t.Fatalf("unqualified response reading=%+v metadata=%+v, want retained last-good value marked stale", unqualified, metadata)
			}

			client.err = errors.New("connection refused")
			cached, metadata := read()
			if cached.Trust != TrustCached || cached.Value != float64(2) || metadata.StalenessTS == nil || !metadata.FromCache {
				t.Fatalf("outage reading=%+v metadata=%+v, want retained last-good value marked stale", cached, metadata)
			}

			setCachedAge(3*ttl, ttl+time.Second)
			expired, metadata := read()
			if expired.Trust != TrustUnavailable || expired.Value != float64(2) || expired.TrustReason != "producer observation is stale" || metadata.StalenessTS == nil {
				t.Fatalf("expired reading=%+v metadata=%+v, want unavailable trust with retained last-good value", expired, metadata)
			}

			newObserved := time.Now().UTC()
			client.err = nil
			client.body = payload(newObserved, 3)
			recovered, metadata := read()
			if recovered.Trust != TrustValid || recovered.Value != float64(3) || recovered.ObservedAt == nil || !recovered.ObservedAt.Equal(newObserved.Truncate(time.Nanosecond)) {
				t.Fatalf("recovered reading=%+v, want fresh current count 3", recovered)
			}
			if metadata.StalenessTS != nil || metadata.FromCache {
				t.Fatalf("recovery metadata=%+v, want stale marker cleared and new producer response", metadata)
			}
		})
	}
}

func TestStructuredReadingsRetainLastGoodAcrossTimestampedSchemaFailure(t *testing.T) {
	ttl := time.Second
	cases := []struct {
		name, kind, selector string
		valid, invalid       func(time.Time) json.RawMessage
		wantValue            float64
		wantRows             int
	}{
		{
			name: "panel", kind: "panel", selector: "traffic_countries", wantValue: 100, wantRows: 2,
			valid: func(at time.Time) json.RawMessage {
				return recoveryJSON(t, map[string]any{"observed_at": at.Format(time.RFC3339Nano), "rows": []any{
					map[string]any{"key": "us", "label": "United States", "value": 70, "share": .7},
					map[string]any{"key": "de", "label": "Germany", "value": 30, "share": .3},
				}})
			},
			invalid: func(at time.Time) json.RawMessage {
				return recoveryJSON(t, map[string]any{"observed_at": at.Format(time.RFC3339Nano), "rows": []any{
					map[string]any{"key": "us", "label": "United States", "value": 70, "share": .9},
					map[string]any{"key": "de", "label": "Germany", "value": 30, "share": .3},
				}})
			},
		},
		{
			name: "ladder", kind: "ladder", selector: "release_ladder", wantValue: 1, wantRows: 1,
			valid: func(at time.Time) json.RawMessage {
				return recoveryJSON(t, map[string]any{"observed_at": at.Format(time.RFC3339Nano), "entries": []any{
					map[string]any{"deliverable": map[string]any{"id": "one", "name": "First", "status": "ACTIVE", "releaseRank": 1}},
				}})
			},
			invalid: func(at time.Time) json.RawMessage {
				return recoveryJSON(t, map[string]any{"observed_at": at.Format(time.RFC3339Nano), "entries": []any{
					map[string]any{"deliverable": map[string]any{"id": "", "name": "Broken", "status": "ACTIVE", "releaseRank": 1}},
				}})
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			metric := MetricEntry{ID: tc.name, Label: tc.name, Kind: tc.kind, Coverage: CoverageNow, UpstreamSource: SourceOffer,
				Source: SourceBinding{Binding: "offer-desk", Read: "/feed", Selector: tc.selector, IntegrationID: "offer-desk", TTLSeconds: int(ttl.Seconds())}}
			client := &recoveringUpstream{}
			s := NewServer(&Registry{Dashboards: map[string][]MetricEntry{"hive": {metric}}})
			s.integrationRegistry = capabilityregistry.New(nil, nil, 5*time.Second)
			s.offer = client
			read := func() (MetricEntry, sourceMetadata) {
				metrics, metadata := s.readings(context.Background(), []MetricEntry{metric})
				return metrics[0], metadata["offer-desk"]
			}
			key := observationCacheKey(SourceOffer, "/feed", metric.ID)
			client.body = tc.valid(time.Now().UTC())
			fresh, metadata := read()
			if fresh.Trust != TrustValid || fresh.Value != tc.wantValue || len(fresh.Rows) != tc.wantRows || metadata.FromCache {
				t.Fatalf("fresh reading=%+v metadata=%+v", fresh, metadata)
			}

			s.cache.mu.Lock()
			entry := s.cache.entries[key]
			entry.env.CachedAt = time.Now().UTC().Add(-2 * ttl)
			s.cache.entries[key] = entry
			s.cache.mu.Unlock()
			client.body = tc.invalid(time.Now().UTC())
			degraded, metadata := read()
			if degraded.Trust != TrustCached || degraded.Value != tc.wantValue || len(degraded.Rows) != tc.wantRows || degraded.TrustReason == "" || metadata.StalenessTS == nil || !metadata.FromCache {
				t.Fatalf("schema-invalid refresh reading=%+v metadata=%+v, want last-good cached value", degraded, metadata)
			}

			s.cache.mu.Lock()
			entry = s.cache.entries[key]
			old := time.Now().UTC().Add(-3 * ttl)
			entry.env.ObservationAt = &old
			entry.env.CachedAt = time.Now().UTC().Add(-3 * ttl)
			s.cache.entries[key] = entry
			s.cache.mu.Unlock()
			expired, metadata := read()
			if expired.Trust != TrustUnavailable || expired.Value != tc.wantValue || len(expired.Rows) != tc.wantRows || metadata.StalenessTS == nil {
				t.Fatalf("expired reading=%+v metadata=%+v, want unavailable while retaining last-good value", expired, metadata)
			}

			client.body = tc.valid(time.Now().UTC())
			recovered, metadata := read()
			if recovered.Trust != TrustValid || recovered.Value != tc.wantValue || len(recovered.Rows) != tc.wantRows || metadata.StalenessTS != nil || metadata.FromCache {
				t.Fatalf("recovered reading=%+v metadata=%+v", recovered, metadata)
			}
		})
	}
}

func recoveryJSON(t *testing.T, value any) json.RawMessage {
	t.Helper()
	body, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return body
}
