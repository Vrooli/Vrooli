package bundles

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestHandleValidateBundleAcceptsSampleManifest(t *testing.T) {
	h := NewHandler(nil, nil, func(msg string, fields map[string]interface{}) {})

	samplePath := filepath.Join("..", "..", "docs", "examples", "manifests", "desktop-happy.json")
	data, err := os.ReadFile(samplePath)
	if err != nil {
		t.Fatalf("failed to read sample manifest: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/bundles/validate", bytes.NewReader(data))
	rec := httptest.NewRecorder()

	h.ValidateBundle(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleValidateBundleAcceptsPeerWithoutStaticBindings(t *testing.T) {
	h := NewHandler(nil, nil, func(msg string, fields map[string]interface{}) {})
	samplePath := filepath.Join("..", "..", "docs", "examples", "manifests", "desktop-happy.json")
	data, err := os.ReadFile(samplePath)
	if err != nil {
		t.Fatalf("read sample manifest: %v", err)
	}
	var manifest map[string]interface{}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("decode sample manifest: %v", err)
	}
	manifest["peers"] = []map[string]interface{}{{
		"scenario":         "helper-scenario",
		"bundle_policy":    "either",
		"versionRange":     ">=1.2.0",
		"embedded_version": "1.4.0",
	}}
	data, err = json.Marshal(manifest)
	if err != nil {
		t.Fatalf("encode peer manifest: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/bundles/validate", bytes.NewReader(data))
	rec := httptest.NewRecorder()
	h.ValidateBundle(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("no-bindings peer should pass real validation route, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleValidateBundleRejectsInvalidManifest(t *testing.T) {
	h := NewHandler(nil, nil, func(msg string, fields map[string]interface{}) {})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/bundles/validate", bytes.NewReader([]byte(`{"schema_version":"v0.1"}`)))
	rec := httptest.NewRecorder()

	h.ValidateBundle(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid manifest, got %d", rec.Code)
	}
}
