package settings

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"swarm-manager/internal/testutil"
	"testing"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
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

func TestHandler_UpdateSuccess(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	handler := &Handler{store: NewStore(path)}

	payload := []byte(`{"theme":"light"}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/settings", bytes.NewReader(payload))
	rec := httptest.NewRecorder()
	handler.Update(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	response := testutil.DecodeProtoJSON(t, rec, &apipb.SettingsResponse{})
	settings := response.GetSettings()
	if settings.GetTheme() != "light" {
		t.Fatalf("expected theme light, got %q", settings.GetTheme())
	}
}
