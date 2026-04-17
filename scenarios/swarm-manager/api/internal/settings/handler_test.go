package settings

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"swarm-manager/internal/testutil"
	"testing"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
)

func TestStore_LoadDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")

	store := NewStore(path)
	settings, err := store.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if settings.Theme != "dark" {
		t.Errorf("expected default theme dark, got %s", settings.Theme)
	}
	// Verify new workshop boolean defaults.
	if !settings.AutoInitializeWorkshop {
		t.Error("expected AutoInitializeWorkshop default true")
	}
	if !settings.AutoAdvanceWorkshop {
		t.Error("expected AutoAdvanceWorkshop default true")
	}
	if !settings.AutoCascadeWorkshop {
		t.Error("expected AutoCascadeWorkshop default true")
	}
}

func TestStore_LoadDefaults_MaxAutoRoundsZero(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")

	// Write settings with max_auto_rounds=0 — should be valid now.
	testutil.WriteJSONFile(t, path, map[string]any{
		"theme":           "dark",
		"default_mode":    "manual",
		"max_auto_rounds": 0,
	})

	store := NewStore(path)
	settings, err := store.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if settings.MaxAutoRounds != 0 {
		t.Errorf("expected max_auto_rounds 0, got %d", settings.MaxAutoRounds)
	}
}

func TestHandler_UpdatePartial(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")

	handler := &Handler{store: NewStore(path)}
	router := httptest.NewRecorder()

	payload := map[string]any{
		"theme": "light",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/settings", bytes.NewReader(body))
	handler.Update(router, req)

	if router.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", router.Code)
	}

	response := testutil.DecodeProtoJSON(t, router, &apipb.SettingsResponse{})
	settings := response.GetSettings()
	if settings.GetTheme() != "light" {
		t.Errorf("expected theme light, got %s", settings.GetTheme())
	}
}

func TestHandler_UpdatePartial_WorkshopBooleans(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")

	handler := &Handler{store: NewStore(path)}

	// Disable auto-initialize, leave others as defaults.
	payload := map[string]any{
		"auto_initialize_workshop": false,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/settings", bytes.NewReader(body))
	w := httptest.NewRecorder()
	handler.Update(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	response := testutil.DecodeProtoJSON(t, w, &apipb.SettingsResponse{})
	settings := response.GetSettings()
	if settings.GetAutoInitializeWorkshop() {
		t.Error("expected auto_initialize_workshop=false after update")
	}
	// Other booleans should still be default (true).
	if !settings.GetAutoAdvanceWorkshop() {
		t.Error("expected auto_advance_workshop to remain true")
	}
	if !settings.GetAutoCascadeWorkshop() {
		t.Error("expected auto_cascade_workshop to remain true")
	}
}
