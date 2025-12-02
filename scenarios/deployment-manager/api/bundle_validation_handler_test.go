package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestHandleValidateBundleAcceptsSampleManifest(t *testing.T) {
	srv := &Server{}
	samplePath := filepath.Join("..", "..", "..", "docs", "deployment", "examples", "manifests", "desktop-happy.json")
	data, err := os.ReadFile(samplePath)
	if err != nil {
		t.Fatalf("failed to read sample manifest: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/bundles/validate", bytes.NewReader(data))
	rec := httptest.NewRecorder()

	srv.handleValidateBundle(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleValidateBundleRejectsInvalidManifest(t *testing.T) {
	srv := &Server{}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/bundles/validate", bytes.NewReader([]byte(`{"schema_version":"v0.1"}`)))
	rec := httptest.NewRecorder()

	srv.handleValidateBundle(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid manifest, got %d", rec.Code)
	}
}
