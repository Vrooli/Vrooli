package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBoardSettingsRejectsTrailingJSONWithoutReplacingSavedChoice(t *testing.T) {
	t.Setenv("COMMAND_CENTER_CONFIG_DIR", t.TempDir())
	s := NewServer(&Registry{Version: "1.0.0", Rooms: []Room{{ID: "mission-control", Title: "Mission Control"}}})
	request := func(body string) *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		s.router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPut, "/api/v1/board-settings", strings.NewReader(body)))
		return recorder
	}

	valid := `{"cycleSeconds":75,"transition":"cut","rooms":[{"id":"mission-control","enabled":false}]}`
	if saved := request(valid); saved.Code != http.StatusOK {
		t.Fatalf("initial settings save status=%d body=%s", saved.Code, saved.Body.String())
	}

	for _, suffix := range []string{` {"cycleSeconds":90}`, ` trailing`} {
		if rejected := request(valid + suffix); rejected.Code != http.StatusBadRequest {
			t.Fatalf("save with suffix %q status=%d body=%s, want 400", suffix, rejected.Code, rejected.Body.String())
		}
	}

	reloaded := httptest.NewRecorder()
	s.router.ServeHTTP(reloaded, httptest.NewRequest(http.MethodGet, "/api/v1/board-settings", nil))
	var settings BoardSettings
	if err := json.Unmarshal(reloaded.Body.Bytes(), &settings); err != nil {
		t.Fatalf("decode saved settings: %v", err)
	}
	if reloaded.Code != http.StatusOK || settings.CycleSeconds != 75 || settings.Transition != "cut" || len(settings.Rooms) != 1 || settings.Rooms[0].Enabled {
		t.Fatalf("saved settings after rejected requests status=%d body=%s, want original choice", reloaded.Code, reloaded.Body.String())
	}
}

func TestBoardSettingsReadRejectsInvalidPersistedConfiguration(t *testing.T) {
	root := t.TempDir()
	t.Setenv("COMMAND_CENTER_CONFIG_DIR", root)
	s := NewServer(&Registry{Version: "1.0.0", Rooms: []Room{{ID: "mission-control", Title: "Mission Control"}}})
	for _, tc := range []struct {
		name string
		raw  string
	}{
		{name: "invalid transition", raw: `{"cycleSeconds":60,"transition":"dissolve","rooms":[]}`},
		{name: "malformed JSON", raw: `{"cycleSeconds":60,`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := os.WriteFile(filepath.Join(root, "board.json"), []byte(tc.raw), 0o600); err != nil {
				t.Fatalf("write invalid saved settings: %v", err)
			}
			response := httptest.NewRecorder()
			s.router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/board-settings", nil))
			if response.Code != http.StatusUnprocessableEntity {
				t.Fatalf("GET settings status=%d body=%s, want 422", response.Code, response.Body.String())
			}
			var body map[string]any
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode error response: %v", err)
			}
			if body["error"] != "board_settings_invalid" {
				t.Fatalf("GET settings error=%v, want board_settings_invalid", body["error"])
			}
		})
	}
}

func TestBoardSettingsRepairRestoresDefaultsAndReloads(t *testing.T) {
	root := t.TempDir()
	t.Setenv("COMMAND_CENTER_CONFIG_DIR", root)
	if err := os.WriteFile(filepath.Join(root, "board.json"), []byte(`{"cycleSeconds":60,"transition":"dissolve","rooms":[]}`), 0o600); err != nil {
		t.Fatalf("write invalid saved settings: %v", err)
	}
	s := NewServer(&Registry{Version: "1.0.0", Rooms: []Room{{ID: "mission-control", Title: "Mission Control"}}})
	request := func(method, body string) *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		s.router.ServeHTTP(recorder, httptest.NewRequest(method, "/api/v1/board-settings", strings.NewReader(body)))
		return recorder
	}
	if invalid := request(http.MethodGet, ""); invalid.Code != http.StatusUnprocessableEntity {
		t.Fatalf("initial GET status=%d body=%s, want 422", invalid.Code, invalid.Body.String())
	}
	defaults := `{"cycleSeconds":60,"transition":"crossfade","rooms":[]}`
	if saved := request(http.MethodPut, defaults); saved.Code != http.StatusOK {
		t.Fatalf("restore PUT status=%d body=%s, want 200", saved.Code, saved.Body.String())
	}
	reloaded := request(http.MethodGet, "")
	var settings BoardSettings
	if err := json.Unmarshal(reloaded.Body.Bytes(), &settings); err != nil {
		t.Fatalf("decode repaired settings: %v", err)
	}
	if reloaded.Code != http.StatusOK || settings.CycleSeconds != 60 || settings.Transition != "crossfade" || len(settings.Rooms) != 1 || settings.Rooms[0].ID != "mission-control" || !settings.Rooms[0].Enabled {
		t.Fatalf("reloaded settings status=%d body=%s, want default cycle/transition and every room enabled", reloaded.Code, reloaded.Body.String())
	}
}
