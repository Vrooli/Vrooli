package settings

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	"swarm-manager/internal/testutil"
)

func TestStore_SaveInvalidTheme(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	store := NewStore(path)

	invalid := DefaultSettings()
	invalid.Theme = "neon"

	if err := store.Save(invalid); err == nil {
		t.Fatalf("expected error for invalid theme")
	}
}

func TestStore_SaveInvalidRecommendationMode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	store := NewStore(path)

	invalid := DefaultSettings()
	invalid.RecommendationMode = "auto"

	if err := store.Save(invalid); err == nil {
		t.Fatalf("expected error for invalid recommendation mode")
	}
}

func TestStore_LoadEmptyFileDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path, []byte(""), 0o644); err != nil {
		t.Fatalf("write empty file: %v", err)
	}

	store := NewStore(path)
	settings, err := store.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if settings.Theme != "dark" {
		t.Fatalf("expected default theme dark, got %s", settings.Theme)
	}
}

func TestStore_LoadLegacyMissingAutoSync(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	legacy := []byte(`{"theme":"dark","recommendationMode":"suggestions","recommendationSources":{"tests":true}}`)
	if err := os.WriteFile(path, legacy, 0o644); err != nil {
		t.Fatalf("write legacy file: %v", err)
	}

	store := NewStore(path)
	settings, err := store.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if settings.RecommendationAutoSync.Interval == "" {
		t.Fatalf("expected auto sync defaults to be applied")
	}
}

func TestHandler_UpdateRejectsEmptyPatch(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	handler := &Handler{store: NewStore(path)}

	req := httptest.NewRequest(http.MethodPut, "/api/v1/settings", bytes.NewBufferString(`{}`))
	rec := httptest.NewRecorder()
	handler.Update(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandler_UpdateRejectsInvalidTheme(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	handler := &Handler{store: NewStore(path)}

	payload := []byte(`{"theme":"neon"}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/settings", bytes.NewBuffer(payload))
	rec := httptest.NewRecorder()
	handler.Update(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandler_UpdateAutoSyncAndSources(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	handler := &Handler{store: NewStore(path)}

	payload := map[string]any{
		"recommendationAutoSync": map[string]any{
			"enabled":      true,
			"interval":     " 30m ",
			"refreshScope": " scheduled ",
		},
		"recommendationSources": map[string]any{
			"coverage": false,
		},
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/settings", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	handler.Update(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	response := testutil.DecodeProtoJSON(t, rec, &apipb.SettingsResponse{})
	settings := response.GetSettings()
	if !settings.GetRecommendationAutoSync().GetEnabled() {
		t.Fatalf("expected auto sync enabled")
	}
	if settings.GetRecommendationAutoSync().GetInterval() != "30m" {
		t.Fatalf("expected interval trimmed to 30m, got %q", settings.GetRecommendationAutoSync().GetInterval())
	}
	if settings.GetRecommendationAutoSync().GetRefreshScope() != "scheduled" {
		t.Fatalf("expected refresh scope trimmed to scheduled, got %q", settings.GetRecommendationAutoSync().GetRefreshScope())
	}
	if settings.GetRecommendationSources().GetCoverage() != false {
		t.Fatalf("expected coverage disabled")
	}
}
