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

func TestRoomCatalogEditValidatesBeforeReplacingLastGoodLayout(t *testing.T) {
	dir := t.TempDir()
	roomsDir := filepath.Join(dir, "rooms")
	if err := os.MkdirAll(roomsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("COMMAND_CENTER_CONFIG_DIR", dir)
	original := `{"id":"mission-control","title":"Original","beats":[{"hero":"active_scenarios","dwellSeconds":20}]}`
	if err := os.WriteFile(filepath.Join(roomsDir, "mission-control.json"), []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}
	s := NewServer(&Registry{Version: "1.0.0", Rooms: []Room{{ID: "mission-control", Title: "Original"}}})
	put := func(body string) *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		s.router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPut, "/api/v1/catalogs/rooms/mission-control", strings.NewReader(body)))
		return recorder
	}

	valid := put(`{"id":"mission-control","title":"Operator layout","beats":[{"hero":"active_scenarios","dwellSeconds":45}]}`)
	if valid.Code != http.StatusOK {
		t.Fatalf("valid room save status=%d body=%s", valid.Code, valid.Body.String())
	}

	// A new registry models the next API process loading the saved layout.
	fresh := &Registry{}
	if err := LoadSplitRoomCatalog(fresh); err != nil {
		t.Fatalf("load saved room into fresh registry: %v", err)
	}
	if len(fresh.Rooms) != 1 || fresh.Rooms[0].Title != "Operator layout" || fresh.Rooms[0].Beats[0].DwellSeconds != 45 {
		t.Fatalf("fresh room = %+v, want durable operator layout", fresh.Rooms)
	}

	invalid := put(`{"id":"mission-control","title":"Broken edit","beats":"not-an-array"}`)
	if invalid.Code != http.StatusBadRequest {
		t.Fatalf("invalid room save status=%d body=%s, want 400", invalid.Code, invalid.Body.String())
	}
	var persisted Room
	raw, err := os.ReadFile(filepath.Join(roomsDir, "mission-control.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &persisted); err != nil {
		t.Fatal(err)
	}
	if persisted.Title != "Operator layout" || s.registry.Rooms[0].Title != "Operator layout" {
		t.Fatalf("after rejected edit disk=%q runtime=%q; both must retain last-good layout", persisted.Title, s.registry.Rooms[0].Title)
	}

	removeLast := httptest.NewRecorder()
	s.router.ServeHTTP(removeLast, httptest.NewRequest(http.MethodDelete, "/api/v1/catalogs/rooms/mission-control", nil))
	if removeLast.Code != http.StatusBadRequest {
		t.Fatalf("delete last room status=%d body=%s, want 400", removeLast.Code, removeLast.Body.String())
	}
}
