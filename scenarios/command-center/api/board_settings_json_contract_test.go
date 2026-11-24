package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
