package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigRootLossRetainsCatalogAndBoardWorkUntilRestored(t *testing.T) {
	rootA := t.TempDir()
	rootB := filepath.Join(t.TempDir(), "unavailable-config")
	for _, name := range []string{"themes", "rooms"} {
		if err := os.MkdirAll(filepath.Join(rootA, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	theme := []byte(`{"id":"operator-theme","tokens":{"accent":"#34d399"}}`)
	room := []byte(`{"id":"operator-room","title":"Saved operator room","theme":"operator-theme","composition":"flow-current"}`)
	board := []byte(`{"cycleSeconds":75,"transition":"cut","rooms":[{"id":"operator-room","enabled":true}]}`)
	for path, raw := range map[string][]byte{
		filepath.Join(rootA, "themes", "operator-theme.json"): theme,
		filepath.Join(rootA, "rooms", "operator-room.json"):   room,
		filepath.Join(rootA, "board.json"):                    board,
	} {
		if err := os.WriteFile(path, raw, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("COMMAND_CENTER_CONFIG_DIR", rootA)
	s := NewServer(&Registry{Version: "1.0.0", Rooms: []Room{{ID: "operator-room", Title: "Saved operator room", Theme: "operator-theme"}}})
	request := func(method, path, body string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		s.router.ServeHTTP(rec, httptest.NewRequest(method, path, strings.NewReader(body)))
		return rec
	}
	getTheme := func() *httptest.ResponseRecorder { return request(http.MethodGet, "/api/v1/catalogs/themes", "") }
	getRoom := func() *httptest.ResponseRecorder { return request(http.MethodGet, "/api/v1/catalogs/rooms", "") }
	getBoardSettings := func() *httptest.ResponseRecorder { return request(http.MethodGet, "/api/v1/board-settings", "") }
	getBoard := func() *httptest.ResponseRecorder { return request(http.MethodGet, "/api/v1/board", "") }

	initialTheme, initialRoom, initialBoard := getTheme(), getRoom(), getBoardSettings()
	for name, response := range map[string]*httptest.ResponseRecorder{"theme": initialTheme, "room": initialRoom, "board settings": initialBoard} {
		if response.Code != http.StatusOK {
			t.Fatalf("initial %s status=%d body=%s", name, response.Code, response.Body.String())
		}
	}
	if !bytes.Contains(initialTheme.Body.Bytes(), []byte("operator-theme")) || !bytes.Contains(initialRoom.Body.Bytes(), []byte("Saved operator room")) || !bytes.Contains(initialBoard.Body.Bytes(), []byte(`"cycleSeconds":75`)) {
		t.Fatalf("initial saved work missing: theme=%s room=%s board=%s", initialTheme.Body, initialRoom.Body, initialBoard.Body)
	}

	// A failed configuration-root transition must neither present defaults as
	// saved work nor create a new instance directory through a write endpoint.
	if err := os.Setenv("COMMAND_CENTER_CONFIG_DIR", rootB); err != nil {
		t.Fatal(err)
	}
	for name, response := range map[string]*httptest.ResponseRecorder{
		"catalog": getTheme(), "board settings": getBoardSettings(), "board": getBoard(),
		"room":                   request(http.MethodGet, "/api/v1/rooms/operator-room", ""),
		"focus":                  request(http.MethodGet, "/api/v1/focus", ""),
		"open loop":              request(http.MethodGet, "/api/v1/open-loop", ""),
		"capability description": request(http.MethodGet, "/api/v1/capabilities/describe", ""),
		"catalog write":          request(http.MethodPut, "/api/v1/catalogs/themes/other", `{"id":"other"}`),
		"board write":            request(http.MethodPut, "/api/v1/board-settings", `{"cycleSeconds":90,"transition":"crossfade","rooms":[]}`),
		"catalog delete":         request(http.MethodDelete, "/api/v1/catalogs/themes/operator-theme", ""),
	} {
		if response.Code != http.StatusServiceUnavailable {
			t.Errorf("during loss %s status=%d body=%s, want 503", name, response.Code, response.Body.String())
		}
		var body map[string]any
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil || body["error"] != "config_unavailable" {
			t.Errorf("during loss %s error=%v body=%s, want config_unavailable", name, body["error"], response.Body.String())
		}
	}
	if _, err := os.Stat(rootB); !os.IsNotExist(err) {
		t.Fatalf("unavailable root was created during failed reads/writes: stat err=%v", err)
	}
	for path, want := range map[string][]byte{
		filepath.Join(rootA, "themes", "operator-theme.json"): theme,
		filepath.Join(rootA, "rooms", "operator-room.json"):   room,
		filepath.Join(rootA, "board.json"):                    board,
	} {
		got, err := os.ReadFile(path)
		if err != nil || !bytes.Equal(got, want) {
			t.Errorf("saved file %s changed during loss: err=%v got=%s", path, err, got)
		}
	}

	if err := os.Setenv("COMMAND_CENTER_CONFIG_DIR", rootA); err != nil {
		t.Fatal(err)
	}
	recoveredTheme, recoveredRoom, recoveredBoard := getTheme(), getRoom(), getBoardSettings()
	for name, pair := range map[string][2]*httptest.ResponseRecorder{
		"theme": {initialTheme, recoveredTheme}, "room": {initialRoom, recoveredRoom}, "board settings": {initialBoard, recoveredBoard},
	} {
		if pair[1].Code != http.StatusOK || !bytes.Equal(pair[0].Body.Bytes(), pair[1].Body.Bytes()) {
			t.Errorf("recovered %s status=%d body=%s, want original response %s", name, pair[1].Code, pair[1].Body, pair[0].Body)
		}
	}
	for path, want := range map[string][]byte{
		filepath.Join(rootA, "themes", "operator-theme.json"): theme,
		filepath.Join(rootA, "rooms", "operator-room.json"):   room,
		filepath.Join(rootA, "board.json"):                    board,
	} {
		got, err := os.ReadFile(path)
		if err != nil || !bytes.Equal(got, want) {
			t.Errorf("recovery changed saved file %s: err=%v got=%s", path, err, got)
		}
	}
}
