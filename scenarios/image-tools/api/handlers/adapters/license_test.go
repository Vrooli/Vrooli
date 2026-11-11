package adapters

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	internaladapters "image-tools/internal/adapters"
)

func TestLicenseTermsEndpointServesOnlyDeclaredVersion(t *testing.T) {
	registry, err := internaladapters.Load()
	if err != nil {
		t.Fatal(err)
	}
	handler := licenseTermsHandler(registry)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/adapters/controlnet-openpose-sd15/license-terms", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "controlnet-openpose-sd15"})
	resp := httptest.NewRecorder()
	handler(resp, req)
	if resp.Code != http.StatusOK || resp.Header().Get("Content-Type") != "text/plain; charset=utf-8" {
		t.Fatalf("response = %d %q", resp.Code, resp.Header().Get("Content-Type"))
	}
	want, err := internaladapters.LicenseTerms(internaladapters.OpenRAILMTermsVersion)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Body.String() != string(want) {
		t.Fatal("endpoint body differs from the embedded immutable license")
	}
	req = httptest.NewRequest(http.MethodGet, "/api/v1/adapters/unknown/license-terms", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "unknown"})
	resp = httptest.NewRecorder()
	handler(resp, req)
	if resp.Code != http.StatusNotFound {
		t.Fatalf("unknown terms response = %d, want 404", resp.Code)
	}
}
