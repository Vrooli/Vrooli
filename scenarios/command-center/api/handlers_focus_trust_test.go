package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type outageAfterFirstRead struct {
	body  json.RawMessage
	reads int
}

func (s *outageAfterFirstRead) Name() string { return "swarm" }
func (s *outageAfterFirstRead) Fetch(context.Context, string) (json.RawMessage, error) {
	s.reads++
	if s.reads > 1 {
		return nil, errors.New("source unavailable")
	}
	return append(json.RawMessage(nil), s.body...), nil
}

func TestFocusSurfacesSourceOutageWithRetainedLastGoodReading(t *testing.T) {
	metric := MetricEntry{
		ID: "live_swarm", Label: "Swarm throughput", Shape: "scalar", Unit: "count",
		Coverage: CoverageNow, Trust: TrustValid, TTLSeconds: 30, DataSource: StatusLive, UpstreamSource: SourceSwarm,
		Source: SourceBinding{Binding: "scenario:swarm-manager", Read: "/api/v1/stats", Selector: "swarm_throughput", IntegrationID: "swarm-manager", ExpectedUnit: "count", TTLSeconds: 30},
	}
	room := Room{ID: "mission-control", Title: "Mission Control", MetricIDs: []string{metric.ID}}
	s := NewServer(&Registry{
		Version: "1.0.0", Dashboards: map[string][]MetricEntry{room.ID: {metric}}, Rooms: []Room{room},
	})
	observed := time.Now().UTC().Add(-5 * time.Second)
	client := &outageAfterFirstRead{body: json.RawMessage(`{"observed_at":"` + observed.Format(time.RFC3339Nano) + `","throughput":{"completed_last_7_days":11}}`)}
	s.swarm = client

	fresh, _ := s.readings(context.Background(), []MetricEntry{metric})
	if len(fresh) != 1 || fresh[0].Trust != TrustValid || fresh[0].Value != float64(11) {
		t.Fatalf("seed reading=%+v, want current measured value 11", fresh)
	}
	focus := func() FocusSurface {
		t.Helper()
		recorder := httptest.NewRecorder()
		s.handleFocus(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/focus", nil))
		if recorder.Code != http.StatusOK {
			t.Fatalf("focus status=%d body=%s", recorder.Code, recorder.Body.String())
		}
		var result FocusSurface
		if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
			t.Fatalf("decode focus: %v", err)
		}
		return result
	}
	for _, entry := range focus().Entries {
		if entry.Kind == "source-unavailable" && entry.MetricID == metric.ID {
			t.Fatalf("current trusted reading should not create a source finding: %+v", entry)
		}
	}

	key := observationCacheKey(SourceSwarm, metric.Source.Read, metric.ID)
	s.cache.mu.Lock()
	cached := s.cache.entries[key]
	cached.env.CachedAt = time.Now().UTC().Add(-31 * time.Second)
	s.cache.entries[key] = cached
	s.cache.mu.Unlock()

	result := focus()
	for _, entry := range result.Entries {
		if entry.Kind == "source-unavailable" && entry.MetricID == metric.ID {
			if entry.Reason != "Source has no current trusted reading; a cached value is last-good only until the source recovers." {
				t.Fatalf("source failure reason=%q", entry.Reason)
			}
			return
		}
	}
	t.Fatalf("focus entries=%+v, want a source-unavailable finding for the cached reading", result.Entries)
}

func TestFocusDoesNotCollapseDistinctProducerSurfacesOwnedByOneTeam(t *testing.T) {
	statsMetric := MetricEntry{
		ID: "swarm_throughput", Label: "Swarm throughput", Shape: "scalar", Unit: "count",
		Coverage: CoverageNow, Trust: TrustValid, DataSource: StatusLive, UpstreamSource: SourceSwarm,
		Source: SourceBinding{Team: "director-swarm", Binding: "scenario:swarm-manager", Read: "/api/v1/stats", Selector: "swarm_throughput", IntegrationID: "swarm-manager", ExpectedUnit: "count"},
	}
	goalsMetric := MetricEntry{
		ID: "goal_progress", Label: "Goal progress", Kind: "panel", Shape: "panel", Unit: "count",
		Coverage: CoverageNow, Trust: TrustValid, DataSource: StatusLive, UpstreamSource: SourceSwarm,
		Source: SourceBinding{Team: "director-swarm", Binding: "scenario:swarm-manager", Read: "/api/v1/goals", Selector: "goal_progress", IntegrationID: "swarm-manager", ExpectedUnit: "count"},
	}
	deploymentMetric := MetricEntry{
		ID: "deployment_readiness", Label: "Deployment readiness", Kind: "panel", Shape: "panel", Unit: "count",
		Coverage: CoverageNow, Trust: TrustValid, DataSource: StatusLive, UpstreamSource: SourceDeploy,
		Source: SourceBinding{Team: "director-swarm", Binding: "scenario:deployment-manager", Read: "/api/v1/readiness/state", Selector: "deployment_readiness", IntegrationID: "deployment-manager", ExpectedUnit: "count"},
	}
	room := Room{ID: "forge", Title: "Forge", MetricIDs: []string{statsMetric.ID, goalsMetric.ID, deploymentMetric.ID}}
	s := NewServer(&Registry{
		Version: "1.0.0", Dashboards: map[string][]MetricEntry{room.ID: {statsMetric, goalsMetric, deploymentMetric}},
		Rooms: []Room{room},
	})
	// Nil clients produce explicit local unavailability without contacting either
	// shared producer; the map mirrors Forge's Swarm/Deployment split.
	s.swarm, s.deploy = nil, nil
	recorder := httptest.NewRecorder()
	s.handleFocus(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/focus", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("focus status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var result FocusSurface
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode focus: %v", err)
	}
	found := map[string]bool{}
	for _, entry := range result.Entries {
		if entry.Kind == "source-unavailable" && entry.Owner == "director-swarm" {
			found[entry.MetricID] = true
		}
	}
	if !found[statsMetric.ID] || !found[goalsMetric.ID] || !found[deploymentMetric.ID] {
		t.Fatalf("Focus findings for director-swarm = %v, want separate findings for endpoints %s, %s, and %s", found, statsMetric.ID, goalsMetric.ID, deploymentMetric.ID)
	}
}
