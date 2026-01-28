package recommendations

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"swarm-manager/internal/settings"
	"swarm-manager/internal/testutil"
)

func TestHandler_RefreshSettingsLoadError(t *testing.T) {
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, "settings.json")
	if err := os.WriteFile(settingsPath, []byte("{"), 0o644); err != nil {
		t.Fatalf("write invalid settings: %v", err)
	}

	handler := &Handler{
		store:         NewStore(filepath.Join(dir, "recs.json")),
		engine:        NewEngine(dir),
		settingsStore: settings.NewStore(settingsPath),
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/recommendations/refresh", nil)
	rec := httptest.NewRecorder()
	handler.Refresh(rec, req)

	testutil.AssertStatus(t, rec, http.StatusInternalServerError)
}

func TestHandler_RefreshSaveError(t *testing.T) {
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, "settings.json")
	cfg := settings.DefaultSettings()
	cfg.RecommendationMode = "suggestions"
	data, _ := json.Marshal(cfg)
	if err := os.WriteFile(settingsPath, data, 0o644); err != nil {
		t.Fatalf("write settings: %v", err)
	}

	readonlyDir := filepath.Join(dir, "readonly")
	if err := os.MkdirAll(readonlyDir, 0o755); err != nil {
		t.Fatalf("create readonly dir: %v", err)
	}
	storePath := filepath.Join(readonlyDir, "recs.json")
	if err := os.WriteFile(storePath, []byte("[]"), 0o644); err != nil {
		t.Fatalf("write store: %v", err)
	}
	if err := os.Chmod(readonlyDir, 0o555); err != nil {
		t.Fatalf("chmod readonly dir: %v", err)
	}
	defer func() {
		_ = os.Chmod(readonlyDir, 0o755)
	}()

	handler := &Handler{
		store:         NewStore(storePath),
		engine:        NewEngine(dir),
		settingsStore: settings.NewStore(settingsPath),
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/recommendations/refresh", nil)
	rec := httptest.NewRecorder()
	handler.Refresh(rec, req)

	testutil.AssertStatus(t, rec, http.StatusInternalServerError)
}
