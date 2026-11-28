package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBoardSettingsRejectInvalidTransitionAndRetainSavedChoice(t *testing.T) {
	t.Setenv("COMMAND_CENTER_CONFIG_DIR", t.TempDir())
	s := NewServer(&Registry{Version: "1.0.0", Rooms: []Room{{ID: "mission-control", Title: "Mission Control"}}})
	request := func(body string) *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		s.router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPut, "/api/v1/board-settings", strings.NewReader(body)))
		return recorder
	}

	saved := request(`{"cycleSeconds":75,"transition":"cut","rooms":[{"id":"mission-control","enabled":false}]}`)
	if saved.Code != http.StatusOK {
		t.Fatalf("valid settings save status=%d body=%s", saved.Code, saved.Body.String())
	}

	rejected := request(`{"cycleSeconds":40,"transition":"dissolve","rooms":[]}`)
	if rejected.Code != http.StatusBadRequest {
		t.Fatalf("invalid transition save status=%d body=%s, want 400", rejected.Code, rejected.Body.String())
	}

	reloaded := httptest.NewRecorder()
	s.router.ServeHTTP(reloaded, httptest.NewRequest(http.MethodGet, "/api/v1/board-settings", nil))
	var settings BoardSettings
	if err := json.Unmarshal(reloaded.Body.Bytes(), &settings); err != nil {
		t.Fatalf("decode reloaded settings: %v", err)
	}
	if reloaded.Code != http.StatusOK || settings.CycleSeconds != 75 || settings.Transition != "cut" || len(settings.Rooms) != 1 || settings.Rooms[0].Enabled {
		t.Fatalf("reloaded settings status=%d settings=%+v, want retained saved choice", reloaded.Code, settings)
	}
}
