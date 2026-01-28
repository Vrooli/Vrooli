package settings

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/gorilla/mux"
	"swarm-manager/internal/testutil"
)

type settingsResponse struct {
	Settings Settings `json:"settings"`
}

func setupHandler(t *testing.T) (*Handler, string) {
	t.Helper()
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, "settings.json")
	return NewHandler(settingsPath), settingsPath
}

func TestGet_DefaultsWhenMissing(t *testing.T) {
	handler, _ := setupHandler(t)
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/settings", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	testutil.AssertStatusOK(t, rec)
	resp := testutil.DecodeJSON[settingsResponse](t, rec)

	if resp.Settings.Theme != "dark" {
		t.Errorf("expected default theme 'dark', got %q", resp.Settings.Theme)
	}
	if resp.Settings.RecommendationMode != "off" {
		t.Errorf("expected default recommendationMode 'off', got %q", resp.Settings.RecommendationMode)
	}
	if resp.Settings.CustomFocus != "" {
		t.Errorf("expected default customFocus '', got %q", resp.Settings.CustomFocus)
	}
	if resp.Settings.InsightsEnabled {
		t.Errorf("expected insightsEnabled false")
	}
	if resp.Settings.InsightsAutoAnalyze {
		t.Errorf("expected insightsAutoAnalyze false")
	}
}

func TestUpdate_PersistsSettings(t *testing.T) {
	handler, settingsPath := setupHandler(t)
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	body := []byte(`{
		"theme": "light",
		"recommendationMode": "suggestions",
		"customFocus": "  improve tests  ",
		"insightsEnabled": true,
		"insightsAutoAnalyze": true
	}`)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/settings", bytes.NewBuffer(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	testutil.AssertStatusOK(t, rec)
	resp := testutil.DecodeJSON[settingsResponse](t, rec)

	if resp.Settings.Theme != "light" {
		t.Errorf("expected theme 'light', got %q", resp.Settings.Theme)
	}
	if resp.Settings.RecommendationMode != "suggestions" {
		t.Errorf("expected recommendationMode 'suggestions', got %q", resp.Settings.RecommendationMode)
	}
	if resp.Settings.CustomFocus != "improve tests" {
		t.Errorf("expected customFocus trimmed, got %q", resp.Settings.CustomFocus)
	}

	persisted := testutil.ReadJSONFile[Settings](t, settingsPath)
	if persisted.Theme != "light" || persisted.RecommendationMode != "suggestions" {
		t.Errorf("expected persisted settings to match update")
	}
}

func TestUpdate_Partial(t *testing.T) {
	handler, _ := setupHandler(t)
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	body := []byte(`{"customFocus": "  ship v1  "}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/settings", bytes.NewBuffer(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	testutil.AssertStatusOK(t, rec)
	resp := testutil.DecodeJSON[settingsResponse](t, rec)

	if resp.Settings.CustomFocus != "ship v1" {
		t.Errorf("expected customFocus trimmed, got %q", resp.Settings.CustomFocus)
	}
	if resp.Settings.Theme != "dark" {
		t.Errorf("expected theme default 'dark', got %q", resp.Settings.Theme)
	}
	if resp.Settings.RecommendationMode != "off" {
		t.Errorf("expected recommendationMode default 'off', got %q", resp.Settings.RecommendationMode)
	}
}

func TestUpdate_InvalidTheme(t *testing.T) {
	handler, _ := setupHandler(t)
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	body := []byte(`{"theme": "neon"}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/settings", bytes.NewBuffer(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	testutil.AssertStatusBadRequest(t, rec)
}

func TestUpdate_InvalidJSON(t *testing.T) {
	handler, _ := setupHandler(t)
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/settings", bytes.NewBuffer([]byte("{")))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	testutil.AssertStatusBadRequest(t, rec)
}
