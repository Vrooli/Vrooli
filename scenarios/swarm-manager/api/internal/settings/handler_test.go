package settings

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"swarm-manager/internal/testutil"

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

func TestHandler_UpdateRejectsRetiredWorkshopAutomationFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")

	handler := &Handler{store: NewStore(path)}

	payload := map[string]any{
		"auto_advance_workshop": true,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/settings", bytes.NewReader(body))
	w := httptest.NewRecorder()
	handler.Update(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}
