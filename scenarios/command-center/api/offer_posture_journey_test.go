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

func TestLedgerPostureSaveReloadPartialOutageAndRecoveryJourney(t *testing.T) {
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

	registry := loadLedgerPostureRegistry(t)
	server := isolatedLedgerPostureServer(t, registry, &recoveringUpstream{})
	room := roomByID(registry, "ledger")
	room.Title = "Ledger · verified posture"
	room.MetricIDs = []string{"offer_posture"}
	room.Beats = []Beat{{Hero: "offer_posture", DwellSeconds: 24}}
	room.Bind = map[string]string{"posture": "offer_posture"}
	roomJSON, err := json.Marshal(room)
	if err != nil {
		t.Fatal(err)
	}
	saveRoom := httptest.NewRecorder()
	server.router.ServeHTTP(saveRoom, httptest.NewRequest(http.MethodPut, "/api/v1/catalogs/rooms/ledger", strings.NewReader(string(roomJSON))))
	if saveRoom.Code != http.StatusOK {
		t.Fatalf("save Ledger room status=%d body=%s", saveRoom.Code, saveRoom.Body.String())
	}
	settingsJSON := `{"cycleSeconds":90,"transition":"crossfade","rooms":[{"id":"ledger","enabled":true}]}`
	saveSettings := httptest.NewRecorder()
	server.router.ServeHTTP(saveSettings, httptest.NewRequest(http.MethodPut, "/api/v1/board-settings", strings.NewReader(settingsJSON)))
	if saveSettings.Code != http.StatusOK {
		t.Fatalf("save Ledger selection status=%d body=%s", saveSettings.Code, saveSettings.Body.String())
	}

	client := &recoveringUpstream{}
	reloaded := isolatedLedgerPostureServer(t, loadLedgerPostureRegistry(t), client)
	boardResponse := httptest.NewRecorder()
	reloaded.router.ServeHTTP(boardResponse, httptest.NewRequest(http.MethodGet, "/api/v1/board", nil))
	var savedBoard BoardShape
	if err := json.Unmarshal(boardResponse.Body.Bytes(), &savedBoard); err != nil {
		t.Fatalf("decode reloaded board: %v", err)
	}
	if boardResponse.Code != http.StatusOK || len(savedBoard.Rooms) != 1 || savedBoard.Rooms[0].ID != "ledger" || savedBoard.Rooms[0].Title != room.Title {
		t.Fatalf("reloaded board status=%d rooms=%+v, want saved Ledger room", boardResponse.Code, savedBoard.Rooms)
	}
	settingsResponse := httptest.NewRecorder()
	reloaded.router.ServeHTTP(settingsResponse, httptest.NewRequest(http.MethodGet, "/api/v1/board-settings", nil))
	var savedSettings BoardSettings
	if err := json.Unmarshal(settingsResponse.Body.Bytes(), &savedSettings); err != nil {
		t.Fatalf("decode reloaded board settings: %v", err)
	}
	if settingsResponse.Code != http.StatusOK || savedSettings.CycleSeconds != 90 || savedSettings.Transition != "crossfade" || len(savedSettings.Rooms) != 1 || !savedSettings.Rooms[0].Enabled {
		t.Fatalf("reloaded board settings status=%d settings=%+v", settingsResponse.Code, savedSettings)
	}

	readRoom := func() (RoomReadings, *httptest.ResponseRecorder) {
		t.Helper()
		response := httptest.NewRecorder()
		reloaded.router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/rooms/ledger", nil))
		var body RoomReadings
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode Ledger room: %v", err)
		}
		return body, response
	}
	postureReading := func(body RoomReadings) MetricEntry {
		t.Helper()
		for _, reading := range body.Readings {
			if reading.ID == "offer_posture" {
				return reading
			}
		}
		t.Fatalf("Ledger readings omit offer_posture: %+v", body.Readings)
		return MetricEntry{}
	}
	feed := func(observed time.Time, partial bool, cash int64) json.RawMessage {
		t.Helper()
		body, err := json.Marshal(map[string]any{
			"observedAt": observed.UTC().Format(time.RFC3339Nano),
			"position": map[string]any{
				"cashMinor": cash, "burnMinor": int64(4000), "revenueMinor": int64(12000),
				"runwayMonths": 20.5, "runwayAvailable": true, "currency": "USD", "partial": partial,
			},
			"postureSource":     "money-ledger.position: operator-book",
			"postureAgeSeconds": int64(time.Since(observed).Seconds()),
			"defaultAliveGap":   "default-alive threshold met with 1.4 buffer",
		})
		if err != nil {
			t.Fatal(err)
		}
		return body
	}

	firstObserved := time.Now().UTC().Add(-time.Minute)
	client.body = feed(firstObserved, false, 82000)
	fresh, freshResponse := readRoom()
	reading := postureReading(fresh)
	value, valueOK := reading.Value.(map[string]any)
	if freshResponse.Code != http.StatusOK || reading.Coverage != CoverageNow || reading.Trust != TrustValid || !valueOK || value["cashMinor"] != float64(82000) || value["currency"] != "USD" {
		t.Fatalf("fresh Ledger posture status=%d reading=%+v", freshResponse.Code, reading)
	}
	if reading.Source.Read != "/vrooli.offer_desk.v1.offers.BoardService/GetBoard?projection=ledger" || reading.ObservedAt == nil || reading.ObservedAt.After(time.Now().UTC()) {
		t.Fatalf("fresh Ledger source metadata=%+v, want producer-timestamped BoardService response", reading)
	}

	client.body = feed(time.Now().UTC(), true, 99999)
	cacheKey := observationCacheKey(SourceOffer, reading.Source.Read, "offer_posture")
	reloaded.cache.mu.Lock()
	cacheEntry := reloaded.cache.entries[cacheKey]
	cacheEntry.env.CachedAt = time.Now().UTC().Add(-10 * time.Minute)
	reloaded.cache.entries[cacheKey] = cacheEntry
	reloaded.cache.mu.Unlock()
	partial, _ := readRoom()
	reading = postureReading(partial)
	value, valueOK = reading.Value.(map[string]any)
	if reading.Trust != TrustCached || !valueOK || value["cashMinor"] != float64(82000) || partial.Sources[string(SourceOffer)].StalenessTS == nil {
		t.Fatalf("partial posture reading=%+v source=%+v; want stale last-good value, not partial 99999", reading, partial.Sources[string(SourceOffer)])
	}

	client.err = errors.New("offer desk unavailable")
	outage, _ := readRoom()
	reading = postureReading(outage)
	if reading.Trust != TrustCached || reading.Value.(map[string]any)["cashMinor"] != float64(82000) {
		t.Fatalf("outage posture=%+v, want retained last-good position", reading)
	}

	reloaded.cache.mu.Lock()
	cacheEntry = reloaded.cache.entries[cacheKey]
	expiredObserved := time.Now().UTC().Add(-10 * time.Minute)
	cacheEntry.env.ObservationAt = &expiredObserved
	reloaded.cache.entries[cacheKey] = cacheEntry
	reloaded.cache.mu.Unlock()
	expired, _ := readRoom()
	reading = postureReading(expired)
	if reading.Trust != TrustUnavailable || reading.Value.(map[string]any)["cashMinor"] != float64(82000) {
		t.Fatalf("expired posture=%+v, want unavailable while retaining last-good API data", reading)
	}

	recoveredAt := time.Now().UTC()
	client.err = nil
	client.body = feed(recoveredAt, false, 90000)
	recovered, _ := readRoom()
	reading = postureReading(recovered)
	if reading.Trust != TrustValid || reading.Value.(map[string]any)["cashMinor"] != float64(90000) || reading.ObservedAt == nil || recovered.Sources[string(SourceOffer)].StalenessTS != nil {
		t.Fatalf("recovered posture=%+v source=%+v, want refreshed value and cleared stale marker", reading, recovered.Sources[string(SourceOffer)])
	}
}

func loadLedgerPostureRegistry(t *testing.T) *Registry {
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
		if candidate.ID == "offer_posture" {
			metric = candidate
			break
		}
	}
	if metric.ID == "" || metric.Source.SourceTimePolicy != "producer_required" || metric.Coverage != CoverageNow {
		t.Fatalf("offer posture config=%+v, want producer-time-qualified live signal", metric)
	}
	var room Room
	for _, candidate := range registry.Rooms {
		if candidate.ID == "ledger" {
			room = candidate
		}
	}
	if room.ID == "" {
		t.Fatal("split room catalog has no Ledger")
	}
	room.MetricIDs = []string{"offer_posture"}
	room.Beats = []Beat{{Hero: "offer_posture", DwellSeconds: 24}}
	registry.Metrics = []MetricEntry{metric}
	registry.Dashboards = nil
	registry.Rooms = []Room{room}
	return registry
}

func isolatedLedgerPostureServer(t *testing.T, registry *Registry, client *recoveringUpstream) *Server {
	t.Helper()
	server := NewServer(registry)
	server.offer = client
	server.vrooli = unavailableIntegrationClient{}
	server.swarm = unavailableIntegrationClient{}
	server.lpbs = unavailableIntegrationClient{}
	server.deploy = unavailableIntegrationClient{}
	server.promptManager.resolve = func() string { return "" }
	server.integrationRegistry = commandCenterIntegrationRegistry(server)
	return server
}
