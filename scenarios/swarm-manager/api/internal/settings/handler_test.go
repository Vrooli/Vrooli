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
	if !settings.RecommendationSources.Problems {
		t.Errorf("expected default recommendationSources.problems true")
	}
}

func TestStore_LoadLegacyMissingSources(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")

	legacy := []byte(`{"theme":"dark","recommendationMode":"suggestions"}`)
	if err := os.WriteFile(path, legacy, 0o644); err != nil {
		t.Fatalf("write legacy file: %v", err)
	}

	store := NewStore(path)
	settings, err := store.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if settings.RecommendationSources == (RecommendationSources{}) {
		t.Errorf("expected recommendationSources defaults to be applied")
	}
}

func TestHandler_UpdatePartial(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")

	handler := &Handler{store: NewStore(path)}
	router := httptest.NewRecorder()

	payload := map[string]any{
		"theme": "light",
		"recommendationSources": map[string]any{
			"tests": false,
		},
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
	if settings.GetRecommendationSources().GetTests() != false {
		t.Errorf("expected recommendationSources.tests false")
	}
}
