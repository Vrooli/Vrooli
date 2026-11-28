package main

import (
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

func TestForgeGoalProgressSaveReloadAndRecoveryJourney(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("COMMAND_CENTER_CONFIG_DIR", configDir)
	for _, group := range []string{"signals", "rooms", "connectors"} {
		copyJSONDir(t, filepath.Join("..", "config", group), filepath.Join(configDir, group))
	}
	copySetpointFixture(t, configDir)
	if board, err := os.ReadFile(filepath.Join("..", "config", "board.json")); err == nil {
		if err := os.WriteFile(filepath.Join(configDir, "board.json"), board, 0o600); err != nil {
			t.Fatal(err)
		}
	}

	registry := loadForgeGoalRegistry(t)
	server := isolatedGoalProgressServer(t, registry, &recoveringUpstream{})

	room := roomByID(registry, "forge")
	room.Title = "The Forge · Goal progress"
	room.MetricIDs = []string{"goal_progress"}
	room.Beats = []Beat{{Hero: "goal_progress", DwellSeconds: 18}}
	room.Bind = map[string]string{"throughput": "goal_progress"}
	roomJSON, err := json.Marshal(room)
	if err != nil {
		t.Fatal(err)
	}
	putRoom := httptest.NewRecorder()
	server.router.ServeHTTP(putRoom, httptest.NewRequest(http.MethodPut, "/api/v1/catalogs/rooms/forge", strings.NewReader(string(roomJSON))))
	if putRoom.Code != http.StatusOK {
		t.Fatalf("save Forge room status=%d body=%s", putRoom.Code, putRoom.Body.String())
	}
	settingsJSON := `{"cycleSeconds":75,"transition":"cut","rooms":[{"id":"forge","enabled":true}]}`
	putSettings := httptest.NewRecorder()
	server.router.ServeHTTP(putSettings, httptest.NewRequest(http.MethodPut, "/api/v1/board-settings", strings.NewReader(settingsJSON)))
	if putSettings.Code != http.StatusOK {
		t.Fatalf("save board settings status=%d body=%s", putSettings.Code, putSettings.Body.String())
	}

	// A new API registry and server model the next client/session after both
	// the room composition and board selection have been saved.
	reloadedRegistry := loadForgeGoalRegistry(t)
	client := &recoveringUpstream{}
	reloaded := isolatedGoalProgressServer(t, reloadedRegistry, client)

	boardResponse := httptest.NewRecorder()
	reloaded.router.ServeHTTP(boardResponse, httptest.NewRequest(http.MethodGet, "/api/v1/board", nil))
	var board BoardShape
	if err := json.Unmarshal(boardResponse.Body.Bytes(), &board); err != nil {
		t.Fatalf("decode reloaded board: %v", err)
	}
	if boardResponse.Code != http.StatusOK || len(board.Rooms) != 1 || board.Rooms[0].ID != "forge" || board.Rooms[0].Title != room.Title {
		t.Fatalf("reloaded board status=%d rooms=%+v, want saved Forge room", boardResponse.Code, board.Rooms)
	}
	settingsResponse := httptest.NewRecorder()
	reloaded.router.ServeHTTP(settingsResponse, httptest.NewRequest(http.MethodGet, "/api/v1/board-settings", nil))
	var settings BoardSettings
	if err := json.Unmarshal(settingsResponse.Body.Bytes(), &settings); err != nil {
		t.Fatalf("decode reloaded board settings: %v", err)
	}
	if settingsResponse.Code != http.StatusOK || settings.CycleSeconds != 75 || settings.Transition != "cut" || len(settings.Rooms) != 1 || !settings.Rooms[0].Enabled {
		t.Fatalf("reloaded settings status=%d settings=%+v, want saved enabled Forge selection", settingsResponse.Code, settings)
	}

	readRoom := func() (RoomReadings, *httptest.ResponseRecorder) {
		t.Helper()
		response := httptest.NewRecorder()
		reloaded.router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/rooms/forge", nil))
		var body RoomReadings
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode Forge room: %v", err)
		}
		return body, response
	}
	goalReading := func(body RoomReadings) MetricEntry {
		t.Helper()
		for _, reading := range body.Readings {
			if reading.ID == "goal_progress" {
				return reading
			}
		}
		t.Fatalf("Forge readings omit goal_progress: %+v", body.Readings)
		return MetricEntry{}
	}
	feed := func(observed time.Time, completed float64, total float64) json.RawMessage {
		t.Helper()
		body, err := json.Marshal(map[string]any{
			"observed_at": observed.UTC().Format(time.RFC3339Nano),
			"items": []any{
				map[string]any{
					"goal":  map[string]any{"name": "unscoped", "title": "Unscoped goal", "status": "active", "priority": float64(0)},
					"scope": map[string]any{"completed_count": float64(0), "total": float64(0)},
				},
				map[string]any{
					"goal":  map[string]any{"name": "delivery", "title": "Delivery", "status": "active", "priority": float64(8)},
					"scope": map[string]any{"completed_count": completed, "total": total},
					"eta":   map[string]any{"p50_hours": float64(4), "p80_hours": float64(9)},
				},
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		return body
	}

	firstObserved := time.Now().UTC()
	client.body = feed(firstObserved, 2, 5)
	fresh, response := readRoom()
	reading := goalReading(fresh)
	if response.Code != http.StatusOK || reading.Coverage != CoverageNow || reading.Trust != TrustValid || len(reading.Rows) != 2 || reading.Rows[0].Detail != "No scoped work" || reading.Rows[1].Detail != "4.0h–9.0h ETA" {
		t.Fatalf("fresh Forge reading status=%d reading=%+v", response.Code, reading)
	}
	if reading.Source.Read != "/api/v1/goals" || reading.ObservedAt == nil || !reading.ObservedAt.Equal(firstObserved.Truncate(time.Nanosecond)) {
		t.Fatalf("fresh goal source metadata=%+v, want producer-stamped GET /api/v1/goals", reading)
	}

	client.body = json.RawMessage(`{"observed_at":"` + time.Now().UTC().Format(time.RFC3339Nano) + `","items":[{"goal":{"name":"bad","status":"active","priority":0},"scope":{"total":2,"completed_count":1}}]}`)
	reloaded.cache.mu.Lock()
	cacheEntry := reloaded.cache.entries[observationCacheKey(SourceSwarm, "/api/v1/goals", "goal_progress")]
	cacheEntry.env.CachedAt = time.Now().UTC().Add(-10 * time.Minute)
	reloaded.cache.entries[observationCacheKey(SourceSwarm, "/api/v1/goals", "goal_progress")] = cacheEntry
	reloaded.cache.mu.Unlock()
	invalid, _ := readRoom()
	reading = goalReading(invalid)
	if reading.Trust != TrustCached || len(reading.Rows) != 2 || invalid.Sources[string(SourceSwarm)].StalenessTS == nil {
		t.Fatalf("invalid refresh reading=%+v source=%+v, want retained stale last-good rows", reading, invalid.Sources[string(SourceSwarm)])
	}

	client.err = errors.New("source unavailable")
	outage, _ := readRoom()
	reading = goalReading(outage)
	if reading.Trust != TrustCached || len(reading.Rows) != 2 || outage.Sources[string(SourceSwarm)].StalenessTS == nil {
		t.Fatalf("outage reading=%+v source=%+v, want retained cached rows", reading, outage.Sources[string(SourceSwarm)])
	}

	cacheKey := observationCacheKey(SourceSwarm, "/api/v1/goals", "goal_progress")
	reloaded.cache.mu.Lock()
	cacheEntry = reloaded.cache.entries[cacheKey]
	expiredObserved := time.Now().UTC().Add(-10 * time.Minute)
	cacheEntry.env.ObservationAt = &expiredObserved
	reloaded.cache.entries[cacheKey] = cacheEntry
	reloaded.cache.mu.Unlock()
	expired, _ := readRoom()
	reading = goalReading(expired)
	if reading.Trust != TrustUnavailable || len(reading.Rows) != 2 || expired.Sources[string(SourceSwarm)].StalenessTS == nil {
		t.Fatalf("expired reading=%+v source=%+v, want last-good rows retained as unavailable", reading, expired.Sources[string(SourceSwarm)])
	}

	recoveredAt := time.Now().UTC()
	client.err = nil
	client.body = feed(recoveredAt, 3, 5)
	recovered, _ := readRoom()
	reading = goalReading(recovered)
	if reading.Trust != TrustValid || reading.Rows[1].Value != 3 || reading.ObservedAt == nil || recovered.Sources[string(SourceSwarm)].StalenessTS != nil {
		t.Fatalf("recovered reading=%+v source=%+v, want refreshed progress and stale marker cleared", reading, recovered.Sources[string(SourceSwarm)])
	}
}

func loadForgeGoalRegistry(t *testing.T) *Registry {
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
	var goal MetricEntry
	for _, metric := range registry.Metrics {
		if metric.ID == "goal_progress" {
			goal = metric
			break
		}
	}
	if goal.ID == "" || goal.Source.Read != "/api/v1/goals" || goal.Source.SourceTimePolicy != "producer_required" {
		t.Fatalf("configured goal_progress signal=%+v, want producer-timestamped /api/v1/goals", goal)
	}
	var forge Room
	for _, room := range registry.Rooms {
		if room.ID == "forge" {
			forge = room
		}
	}
	if forge.ID == "" {
		t.Fatal("split room catalog has no Forge")
	}
	forge.MetricIDs = []string{"goal_progress"}
	forge.Beats = []Beat{{Hero: "goal_progress", DwellSeconds: 18}}
	registry.Metrics = []MetricEntry{goal}
	registry.Dashboards = nil
	registry.Rooms = []Room{forge}
	return registry
}

func isolatedGoalProgressServer(t *testing.T, registry *Registry, client *recoveringUpstream) *Server {
	t.Helper()
	server := NewServer(registry)
	server.swarm = client
	server.vrooli = unavailableIntegrationClient{}
	server.lpbs = unavailableIntegrationClient{}
	server.offer = unavailableIntegrationClient{}
	server.deploy = unavailableIntegrationClient{}
	server.promptManager.resolve = func() string { return "" }
	server.integrationRegistry = commandCenterIntegrationRegistry(server)
	return server
}

func copyJSONDir(t *testing.T, sourceDir, destinationDir string) {
	t.Helper()
	entries, err := os.ReadDir(sourceDir)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(destinationDir, 0o700); err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(sourceDir, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(destinationDir, entry.Name()), raw, 0o600); err != nil {
			t.Fatal(err)
		}
	}
}
