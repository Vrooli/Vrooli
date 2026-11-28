package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type readinessUpstream struct {
	body  json.RawMessage
	err   error
	paths []string
}

func (c *readinessUpstream) Name() string { return "deployment-manager" }
func (c *readinessUpstream) Fetch(_ context.Context, path string) (json.RawMessage, error) {
	c.paths = append(c.paths, path)
	if c.err != nil {
		return nil, c.err
	}
	return append(json.RawMessage(nil), c.body...), nil
}

func TestForgeReadinessSaveReloadFailureAndRecoveryJourney(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("COMMAND_CENTER_CONFIG_DIR", configDir)
	for _, group := range []string{"signals", "rooms", "connectors"} {
		copyJSONDir(t, filepath.Join("..", "config", group), filepath.Join(configDir, group))
	}
	copySetpointFixture(t, configDir)
	board, err := os.ReadFile(filepath.Join("..", "config", "board.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "board.json"), board, 0o600); err != nil {
		t.Fatal(err)
	}

	registry := loadForgeReadinessRegistry(t)
	server := isolatedReadinessServer(t, registry, &readinessUpstream{})
	room := roomByID(registry, "forge")
	room.Title = "The Forge · Command Center readiness"
	room.MetricIDs = []string{"deployment_readiness"}
	room.Beats = []Beat{{Hero: "deployment_readiness", DwellSeconds: 18}}
	room.Bind = map[string]string{"readiness": "deployment_readiness"}
	roomJSON, err := json.Marshal(room)
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	server.router.ServeHTTP(response, httptest.NewRequest(http.MethodPut, "/api/v1/catalogs/rooms/forge", strings.NewReader(string(roomJSON))))
	if response.Code != http.StatusOK {
		t.Fatalf("save readiness room status=%d body=%s", response.Code, response.Body.String())
	}
	settings := `{"cycleSeconds":80,"transition":"cut","rooms":[{"id":"forge","enabled":true}]}`
	response = httptest.NewRecorder()
	server.router.ServeHTTP(response, httptest.NewRequest(http.MethodPut, "/api/v1/board-settings", strings.NewReader(settings)))
	if response.Code != http.StatusOK {
		t.Fatalf("save readiness board settings status=%d body=%s", response.Code, response.Body.String())
	}

	client := &readinessUpstream{}
	reloaded := isolatedReadinessServer(t, loadForgeReadinessRegistry(t), client)
	response = httptest.NewRecorder()
	reloaded.router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/board", nil))
	var boardBody BoardShape
	if err := json.Unmarshal(response.Body.Bytes(), &boardBody); err != nil {
		t.Fatalf("decode reloaded board: %v", err)
	}
	if response.Code != http.StatusOK || len(boardBody.Rooms) != 1 || boardBody.Rooms[0].ID != "forge" || boardBody.Rooms[0].Title != room.Title {
		t.Fatalf("reloaded board status=%d rooms=%+v", response.Code, boardBody.Rooms)
	}
	response = httptest.NewRecorder()
	reloaded.router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/board-settings", nil))
	var settingsBody BoardSettings
	if err := json.Unmarshal(response.Body.Bytes(), &settingsBody); err != nil {
		t.Fatalf("decode reloaded settings: %v", err)
	}
	if response.Code != http.StatusOK || settingsBody.CycleSeconds != 80 || settingsBody.Transition != "cut" || len(settingsBody.Rooms) != 1 || !settingsBody.Rooms[0].Enabled {
		t.Fatalf("reloaded settings status=%d settings=%+v", response.Code, settingsBody)
	}

	readRoom := func() (RoomReadings, *httptest.ResponseRecorder) {
		t.Helper()
		out := httptest.NewRecorder()
		reloaded.router.ServeHTTP(out, httptest.NewRequest(http.MethodGet, "/api/v1/rooms/forge", nil))
		var body RoomReadings
		if err := json.Unmarshal(out.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode readiness room: %v", err)
		}
		return body, out
	}
	readingFor := func(body RoomReadings) MetricEntry {
		t.Helper()
		for _, reading := range body.Readings {
			if reading.ID == "deployment_readiness" {
				return reading
			}
		}
		t.Fatalf("Forge omitted deployment readiness: %+v", body.Readings)
		return MetricEntry{}
	}
	feed := func(at time.Time, goal, closed bool) json.RawMessage {
		t.Helper()
		body, err := json.Marshal(map[string]any{
			"observed_at":     at.UTC().Format(time.RFC3339Nano),
			"goal_exists":     goal,
			"goal_closed":     closed,
			"approved_commit": "",
		})
		if err != nil {
			t.Fatal(err)
		}
		return body
	}

	freshAt := time.Now().UTC()
	client.body = feed(freshAt, true, false)
	client.paths = nil
	fresh, httpResponse := readRoom()
	reading := readingFor(fresh)
	if httpResponse.Code != http.StatusOK || reading.Trust != TrustValid || reading.Coverage != CoverageNow || reading.Value != float64(1) || len(reading.Rows) != 1 || reading.Rows[0].Label != "Readiness goal present" {
		t.Fatalf("fresh readiness=%+v status=%d", reading, httpResponse.Code)
	}
	if reading.Source.Read != "/api/v1/readiness/state?scenario=command-center" || reading.ObservedAt == nil || !reading.ObservedAt.Equal(freshAt.Truncate(time.Nanosecond)) {
		t.Fatalf("fresh source=%+v observed=%v, want exact configured profile and producer time", reading.Source, reading.ObservedAt)
	}
	if len(client.paths) == 0 || client.paths[len(client.paths)-1] != reading.Source.Read {
		t.Fatalf("producer path=%v, want configured profile query %q", client.paths, reading.Source.Read)
	}
	expireCache := func() {
		t.Helper()
		key := observationCacheKey(SourceDeploy, reading.Source.Read, "deployment_readiness")
		reloaded.cache.mu.Lock()
		entry := reloaded.cache.entries[key]
		entry.env.CachedAt = time.Now().UTC().Add(-10 * time.Minute)
		reloaded.cache.entries[key] = entry
		reloaded.cache.mu.Unlock()
	}

	client.body = json.RawMessage(`{"observed_at":"` + time.Now().UTC().Format(time.RFC3339Nano) + `","goal_exists":"yes","goal_closed":false}`)
	expireCache()
	invalid, _ := readRoom()
	reading = readingFor(invalid)
	if reading.Trust != TrustCached || reading.Value != float64(1) || len(reading.Rows) != 1 || invalid.Sources[string(SourceDeploy)].StalenessTS == nil {
		t.Fatalf("invalid readiness response=%+v source=%+v, want cached last-good", reading, invalid.Sources[string(SourceDeploy)])
	}

	client.err = errors.New("deployment manager unavailable")
	expireCache()
	outage, _ := readRoom()
	reading = readingFor(outage)
	if reading.Trust != TrustCached || reading.Value != float64(1) || outage.Sources[string(SourceDeploy)].StalenessTS == nil {
		t.Fatalf("outage readiness=%+v source=%+v, want stale cached value", reading, outage.Sources[string(SourceDeploy)])
	}

	cacheKey := observationCacheKey(SourceDeploy, reading.Source.Read, "deployment_readiness")
	reloaded.cache.mu.Lock()
	entry := reloaded.cache.entries[cacheKey]
	expired := time.Now().UTC().Add(-10 * time.Minute)
	entry.env.ObservationAt = &expired
	reloaded.cache.entries[cacheKey] = entry
	reloaded.cache.mu.Unlock()
	aged, _ := readRoom()
	reading = readingFor(aged)
	if reading.Trust != TrustUnavailable || reading.Value != float64(1) || aged.Sources[string(SourceDeploy)].StalenessTS == nil {
		t.Fatalf("aged readiness=%+v source=%+v, want retained value marked unavailable", reading, aged.Sources[string(SourceDeploy)])
	}

	recoveryAt := time.Now().UTC()
	client.err = nil
	client.body = feed(recoveryAt, true, true)
	recovered, _ := readRoom()
	reading = readingFor(recovered)
	if reading.Trust != TrustValid || reading.Value != float64(1) || reading.Rows[0].Detail != "closed=true" || reading.ObservedAt == nil || recovered.Sources[string(SourceDeploy)].StalenessTS != nil {
		t.Fatalf("recovered readiness=%+v source=%+v", reading, recovered.Sources[string(SourceDeploy)])
	}
}

func loadForgeReadinessRegistry(t *testing.T) *Registry {
	t.Helper()
	registry, err := LoadRegistry(filepath.Join("..", "config", "outcome-registry.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := LoadSplitRoomCatalog(registry); err != nil {
		t.Fatal(err)
	}
	if err := LoadSplitSignalCatalog(registry); err != nil {
		t.Fatal(err)
	}
	var metric MetricEntry
	for _, candidate := range registry.Metrics {
		if candidate.ID == "deployment_readiness" {
			metric = candidate
			break
		}
	}
	if metric.ID == "" || metric.Source.Read != "/api/v1/readiness/state?scenario=command-center" || metric.Coverage != CoverageNow || metric.Sample != nil {
		t.Fatalf("deployment readiness signal=%+v, expected profiled producer-time source without sample", metric)
	}
	room := roomByID(registry, "forge")
	if room.ID == "" {
		t.Fatal("split room catalog has no Forge")
	}
	room.MetricIDs = []string{"deployment_readiness"}
	room.Beats = []Beat{{Hero: "deployment_readiness", DwellSeconds: 18}}
	registry.Metrics, registry.Dashboards, registry.Rooms = []MetricEntry{metric}, nil, []Room{room}
	return registry
}

func isolatedReadinessServer(t *testing.T, registry *Registry, client *readinessUpstream) *Server {
	t.Helper()
	server := NewServer(registry)
	server.deploy = client
	server.vrooli = unavailableIntegrationClient{}
	server.swarm = unavailableIntegrationClient{}
	server.lpbs = unavailableIntegrationClient{}
	server.offer = unavailableIntegrationClient{}
	server.promptManager.resolve = func() string { return "" }
	server.integrationRegistry = commandCenterIntegrationRegistry(server)
	return server
}
