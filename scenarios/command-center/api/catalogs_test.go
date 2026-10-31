package main

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gorilla/mux"
)

func TestCatalogsAreSplitAndReadable(t *testing.T) {
	dir := t.TempDir()
	for name := range catalogNames {
		if err := os.Mkdir(filepath.Join(dir, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("COMMAND_CENTER_CONFIG_DIR", dir)
	if err := os.WriteFile(filepath.Join(dir, "themes", "test.json"), []byte(`{"id":"test"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("GET", "/api/v1/catalogs/themes", nil)
	req = mux.SetURLVars(req, map[string]string{"catalog": "themes"})
	rec := httptest.NewRecorder()
	(&Server{}).handleCatalog(rec, req)
	if rec.Code != 200 || rec.Body.String() == "[]\n" {
		t.Fatalf("catalog not readable: %d %s", rec.Code, rec.Body.String())
	}
}

func TestReadCatalogExcludesIdlessSidecars(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "compositions"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("COMMAND_CENTER_CONFIG_DIR", dir)
	// A real composition entry alongside its generated slot-manifest sidecar.
	if err := os.WriteFile(filepath.Join(dir, "compositions", "orbital-field.json"), []byte(`{"id":"orbital-field"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "compositions", "orbital-field.slots.json"), []byte(`{"composition":"orbital-field","slots":{}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	entries := readCatalog("compositions")
	if len(entries) != 1 {
		t.Fatalf("readCatalog returned %d entries, want 1 (the sidecar must be excluded): %#v", len(entries), entries)
	}
	if entries[0]["id"] != "orbital-field" {
		t.Fatalf("unexpected entry id: %v", entries[0]["id"])
	}
}

func TestSplitRoomCatalogMatchesRegistryDefaults(t *testing.T) {
	reg, err := LoadRegistry("../config/outcome-registry.json")
	if err != nil {
		t.Fatal(err)
	}
	entries := readCatalog("rooms")
	if len(entries) != len(reg.Rooms) {
		t.Fatalf("split rooms=%d registry rooms=%d", len(entries), len(reg.Rooms))
	}
	for _, room := range reg.Rooms {
		var found Room
		for _, entry := range entries {
			raw, _ := json.Marshal(entry)
			var candidate Room
			_ = json.Unmarshal(raw, &candidate)
			if candidate.ID == room.ID {
				found = candidate
				break
			}
		}
		if found.ID == "" || found.Theme != room.Theme || found.Composition != room.Composition || len(found.Beats) != len(room.Beats) {
			t.Errorf("split room %s does not reproduce registry preset", room.ID)
		}
	}
}

func TestSplitSignalCatalogOverlaysRegistryDefinitions(t *testing.T) {
	reg, err := LoadRegistry("../config/outcome-registry.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := LoadSplitSignalCatalog(reg); err != nil {
		t.Fatal(err)
	}
	if len(reg.Metrics) < 50 {
		t.Fatalf("signals=%d, expected the complete signal catalog", len(reg.Metrics))
	}
	for _, signal := range reg.Metrics {
		if signal.Shape == "" {
			t.Errorf("signal %s lost its shape during overlay", signal.ID)
		}
	}
}

func TestAllVrooliConnectorSourcesHaveDescriptorCatalogEntries(t *testing.T) {
	want := map[string]string{
		"swarm-manager":               "http-json",
		"vrooli-core":                 "http-json",
		"landing-page-business-suite": "http-json",
		"offer-desk":                  "connect",
		"deployment-manager":          "http-json",
		"prompt-manager":              "connect",
		"source-ledger":               "http-json",
	}
	for id, transport := range want {
		path := filepath.Join("../config", "connectors", id+".json")
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("connector %s: %v", id, err)
		}
		var entry connectorCatalogEntry
		if err := json.Unmarshal(raw, &entry); err != nil {
			t.Fatalf("connector %s: %v", id, err)
		}
		if entry.ID != id || entry.Transport != transport || len(entry.Paths) == 0 {
			t.Errorf("connector %s = %#v, want transport %s and at least one path", id, entry, transport)
		}
	}
}
