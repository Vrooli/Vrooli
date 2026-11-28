package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	capabilityregistry "github.com/vrooli/vrooli/packages/capability-registry-go"
)

type swarmExecutionFixture struct{ body json.RawMessage }

func (f *swarmExecutionFixture) Name() string { return "swarm-manager" }
func (f *swarmExecutionFixture) Fetch(context.Context, string) (json.RawMessage, error) {
	return append(json.RawMessage(nil), f.body...), nil
}

func TestSwarmMeanWithNoSamplesRetainsLastGoodRoomValue(t *testing.T) {
	const ttl = 30
	metric := MetricEntry{
		ID: "timing_stats", Label: "Mean execution", Kind: "number", Shape: "scalar", Unit: "minutes",
		Coverage: CoverageNow, Trust: TrustValid, TTLSeconds: ttl, UpstreamSource: SourceSwarm, DataSource: StatusLive,
		Source: SourceBinding{Binding: "scenario:swarm-manager", Read: "/api/v1/stats", Selector: "timing_stats", IntegrationID: "swarm-manager", FeatureID: "timing_stats", ExpectedUnit: "minutes", SourceTimePolicy: "producer_required", TTLSeconds: ttl},
	}
	room := Room{ID: "mission-control", Title: "Mission Control", MetricIDs: []string{metric.ID}}
	s := NewServer(&Registry{Version: "1.0.0", Dashboards: map[string][]MetricEntry{room.ID: {metric}}, Rooms: []Room{room}})
	s.integrationRegistry = capabilityregistry.New(nil, nil, 5*time.Second)
	client := &swarmExecutionFixture{}
	s.swarm = client
	readRoom := func() RoomReadings {
		t.Helper()
		recorder := httptest.NewRecorder()
		s.router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/rooms/mission-control", nil))
		if recorder.Code != http.StatusOK {
			t.Fatalf("room status=%d body=%s", recorder.Code, recorder.Body.String())
		}
		var result RoomReadings
		if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		return result
	}
	payload := func(mean *float64, samples int) json.RawMessage {
		agent := map[string]any{"execution_duration_samples": samples}
		if mean != nil {
			agent["avg_execution_minutes"] = *mean
		}
		body, err := json.Marshal(map[string]any{"generated_at": time.Now().UTC(), "agent": agent})
		if err != nil {
			t.Fatal(err)
		}
		return body
	}

	firstMean := 24.5
	client.body = payload(&firstMean, 2)
	fresh := readRoom()
	if got := fresh.Readings[0]; got.Trust != TrustValid || got.Value != 24.5 {
		t.Fatalf("fresh room reading=%+v, want measured mean 24.5", got)
	}

	key := observationCacheKey(SourceSwarm, "/api/v1/stats", metric.ID)
	s.cache.mu.Lock()
	entry := s.cache.entries[key]
	entry.env.CachedAt = time.Now().UTC().Add(-time.Duration(ttl+1) * time.Second)
	s.cache.entries[key] = entry
	s.cache.mu.Unlock()
	client.body = payload(nil, 0)
	degraded := readRoom()
	if got := degraded.Readings[0]; got.Trust != TrustCached || got.Value != 24.5 || got.TrustReason == "" {
		t.Fatalf("zero-sample room reading=%+v, want retained last-good 24.5 marked cached", got)
	}
	if source := degraded.Sources[string(SourceSwarm)]; !source.FromCache || source.StalenessTS == nil {
		t.Fatalf("zero-sample source metadata=%+v, want retained stale source marker", source)
	}

	recoveredMean := 18.0
	client.body = payload(&recoveredMean, 3)
	recovered := readRoom()
	if got := recovered.Readings[0]; got.Trust != TrustValid || got.Value != 18.0 || got.ObservedAt == nil {
		t.Fatalf("recovered room reading=%+v, want current measured mean 18", got)
	}
	if source := recovered.Sources[string(SourceSwarm)]; source.FromCache || source.StalenessTS != nil {
		t.Fatalf("recovered source metadata=%+v, want fresh source and cleared stale marker", source)
	}
}
