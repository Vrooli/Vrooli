package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	capabilityregistry "github.com/vrooli/vrooli/packages/capability-registry-go"
)

type setpointReadingSource struct {
	body json.RawMessage
	err  error
}

func TestRoomReadingRejectsMalformedSavedSetpoint(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("COMMAND_CENTER_CONFIG_DIR", configDir)
	if err := os.WriteFile(filepath.Join(configDir, "setpoint.json"), []byte(`{"schemaVersion":"1.0.0","bars":[]} {}`), 0o600); err != nil {
		t.Fatal(err)
	}
	metric := testRegistry().Dashboards["mission-control"][0]
	metric.Unit = "count"
	registry := &Registry{Version: "1.0.0", Dashboards: map[string][]MetricEntry{"mission-control": {metric}}, Rooms: []Room{{ID: "mission-control", Title: "Mission Control"}}}
	server := newSetpointRoomServer(registry, &setpointReadingSource{})
	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/rooms/mission-control", nil))
	if recorder.Code != http.StatusInternalServerError || !strings.Contains(recorder.Body.String(), `"setpoint_invalid"`) {
		t.Fatalf("malformed setpoint status=%d body=%s, want safe setpoint error", recorder.Code, recorder.Body.String())
	}
}

func TestSetpointDistanceUsesDirectionAndNeverGoesBelowZero(t *testing.T) {
	readings := []MetricEntry{{ID: "decreasing", Value: float64(12)}, {ID: "met", Value: float64(15)}}
	setpoint := &Setpoint{Bars: []SetpointBar{
		{MetricID: "decreasing", Direction: "down", Bar: float64(10)},
		{MetricID: "met", Direction: "up", Bar: float64(12)},
	}}
	applySetpoint(readings, setpoint)
	if readings[0].Target == nil || readings[0].Target.DistanceToTarget == nil || *readings[0].Target.DistanceToTarget != 2 {
		t.Fatalf("downward target=%+v, want distance 2", readings[0].Target)
	}
	if readings[1].Target == nil || readings[1].Target.DistanceToTarget == nil || *readings[1].Target.DistanceToTarget != 0 {
		t.Fatalf("met upward target=%+v, want distance 0", readings[1].Target)
	}
}

func (c *setpointReadingSource) Name() string { return "swarm" }
func (c *setpointReadingSource) Fetch(context.Context, string) (json.RawMessage, error) {
	if c.err != nil {
		return nil, c.err
	}
	return append(json.RawMessage(nil), c.body...), nil
}

func TestRoomReadingUsesSavedSetpointAfterReloadAndSourceFailure(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("COMMAND_CENTER_CONFIG_DIR", configDir)
	writeSetpointBar(t, configDir, 100)

	metric := testRegistry().Dashboards["mission-control"][0]
	metric.Unit = "count"
	registry := &Registry{
		Version:    "1.0.0",
		Dashboards: map[string][]MetricEntry{"mission-control": {metric}},
		Rooms:      []Room{{ID: "mission-control", Title: "Mission Control", MetricIDs: []string{metric.ID}}},
	}
	observed := time.Now().UTC().Add(-time.Second)
	client := &setpointReadingSource{body: json.RawMessage(fmt.Sprintf(`{"observed_at":%q,"throughput":{"completed_last_7_days":11}}`, observed.Format(time.RFC3339Nano)))}

	first := newSetpointRoomServer(registry, client)
	if got := getSetpointRoom(t, first); got.Target == nil || got.Target.Bar != float64(100) || got.Target.DistanceToTarget == nil || *got.Target.DistanceToTarget != 89 || got.Value != float64(11) || got.Trust != TrustValid {
		t.Fatalf("initial reading=%+v, want value 11 with saved target 100", got)
	}

	// A saved target is read again on the next request, without restarting the API.
	writeSetpointBar(t, configDir, 110)
	reloaded := newSetpointRoomServer(registry, client)
	if got := getSetpointRoom(t, reloaded); got.Target == nil || got.Target.Bar != float64(110) || got.Target.DistanceToTarget == nil || *got.Target.DistanceToTarget != 99 || got.Value != float64(11) {
		t.Fatalf("reloaded reading=%+v, want value 11 with saved target 110", got)
	}

	// Once the producer fails, the last-good value and the newly saved target
	// remain distinct: value is cached, while the target is still current.
	writeSetpointBar(t, configDir, 120)
	key := observationCacheKey(SourceSwarm, "/api/v1/stats", metric.ID)
	reloaded.cache.mu.Lock()
	entry := reloaded.cache.entries[key]
	entry.env.CachedAt = time.Now().UTC().Add(-entry.ttl - time.Second)
	reloaded.cache.entries[key] = entry
	reloaded.cache.mu.Unlock()
	client.err = fmt.Errorf("source unavailable")
	if got := getSetpointRoom(t, reloaded); got.Target == nil || got.Target.Bar != float64(120) || got.Target.DistanceToTarget == nil || *got.Target.DistanceToTarget != 109 || got.Value != float64(11) || got.Trust != TrustCached {
		t.Fatalf("outage reading=%+v, want cached value 11 with current target 120", got)
	}
}

func newSetpointRoomServer(registry *Registry, source *setpointReadingSource) *Server {
	server := NewServer(registry)
	for _, entries := range registry.Dashboards {
		registry.Metrics = append(registry.Metrics, entries...)
	}
	server.swarm = source
	server.integrationRegistry = capabilityregistry.New(nil, nil, time.Second)
	server.instrumentProvider = nil
	server.objectiveProvider = nil
	return server
}

func getSetpointRoom(t *testing.T, server *Server) MetricEntry {
	t.Helper()
	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/rooms/mission-control", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("room status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var body RoomReadings
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode room: %v", err)
	}
	if len(body.Readings) != 1 {
		t.Fatalf("room readings=%d, want 1", len(body.Readings))
	}
	return body.Readings[0]
}

func writeSetpointBar(t *testing.T, configDir string, bar int) {
	t.Helper()
	path := filepath.Join(configDir, "setpoint.json")
	tmp, err := os.CreateTemp(configDir, ".setpoint-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmp.Name())
	body := fmt.Sprintf(`{"$schema":"./setpoint.schema.json","schemaVersion":"1.0.0","bars":[{"metricId":"live_swarm","direction":"up","bar":%d,"unit":"count","authoredAgainst":10,"decisionRef":"decision:test"}],"notGradeable":[{"metricId":"all","reason":"test"}]}`, bar)
	if _, err := tmp.WriteString(body); err != nil {
		t.Fatal(err)
	}
	if err := tmp.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		t.Fatal(err)
	}
}
