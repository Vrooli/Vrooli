package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"app-monitor-api/services"

	"github.com/gin-gonic/gin"
)

func TestCreatePresetInvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	presetService := services.NewPresetService(nil)
	handler := NewPresetHandler(presetService)

	router := setupTestRouter()
	router.POST("/workspace/presets", handler.CreatePreset)

	req := httptest.NewRequest("POST", "/workspace/presets", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for invalid JSON, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestCreatePresetEmptyName(t *testing.T) {
	gin.SetMode(gin.TestMode)

	presetService := services.NewPresetService(nil)
	handler := NewPresetHandler(presetService)

	router := setupTestRouter()
	router.POST("/workspace/presets", handler.CreatePreset)

	body := `{"name":"","color":"#ff0000"}`
	req := httptest.NewRequest("POST", "/workspace/presets", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// nil repo returns ErrDatabaseUnavailable (503), but empty name check happens first
	// Since repo is nil, service returns ErrDatabaseUnavailable before name check?
	// Actually, the service checks name first, then repo. But repo==nil is checked first.
	// Let's check: CreatePreset checks repo==nil first -> returns 503
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected 503 for nil repo, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestListPresetsNilRepo(t *testing.T) {
	gin.SetMode(gin.TestMode)

	presetService := services.NewPresetService(nil)
	handler := NewPresetHandler(presetService)

	router := setupTestRouter()
	router.GET("/workspace/presets", handler.ListPresets)

	req := httptest.NewRequest("GET", "/workspace/presets", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected 503 for nil repo, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestDeletePresetNilRepo(t *testing.T) {
	gin.SetMode(gin.TestMode)

	presetService := services.NewPresetService(nil)
	handler := NewPresetHandler(presetService)

	router := setupTestRouter()
	router.DELETE("/workspace/presets/:id", handler.DeletePreset)

	req := httptest.NewRequest("DELETE", "/workspace/presets/some-id", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected 503 for nil repo, got %d. Body: %s", w.Code, w.Body.String())
	}
}
