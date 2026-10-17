package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	contenthttp "landing-page-business-suite-api/handlers/content"
	"landing-page-business-suite-api/internal/testutil"

	"github.com/gorilla/mux"
)

// ============================================================================
// handleGetSectionsFromConfigStore Tests (Admin Endpoint)
// ============================================================================

func TestHandleGetSections_ReturnsAllSections(t *testing.T) {
	cs := setupTestConfigStore(t)

	variants := cs.ListVariants()
	if len(variants) == 0 {
		t.Fatal("tracked test configuration must contain at least one variant")
	}
	slug := variants[0].Variant.Slug

	variant, err := cs.GetVariant(slug)
	if err != nil {
		t.Fatalf("get tracked variant %s: %v", slug, err)
	}

	handler := contenthttp.Admin(contentHTTPDependencies(cs))
	req := httptest.NewRequest(http.MethodGet, "/admin/variants/"+slug+"/sections", nil)
	req = mux.SetURLVars(req, map[string]string{"variant_slug": slug})

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	testutil.RequireHTTPStatus(t, w, http.StatusOK)

	var resp map[string]interface{}
	decodeJSONResponse(t, w.Body.Bytes(), &resp)

	sections, ok := resp["sections"].([]interface{})
	if !ok {
		t.Fatal("expected 'sections' to be an array")
	}

	// Admin endpoint should return ALL sections (enabled and disabled)
	if len(sections) != len(variant.Sections) {
		t.Errorf("expected %d sections (all), got %d", len(variant.Sections), len(sections))
	}
}

func TestHandleGetSections_IncludesDisabled(t *testing.T) {
	cs := setupTestConfigStore(t)

	variants := cs.ListVariants()
	if len(variants) == 0 {
		t.Fatal("tracked test configuration must contain at least one variant")
	}
	slug := variants[0].Variant.Slug

	variant, err := cs.GetVariant(slug)
	if err != nil {
		t.Fatalf("get tracked variant %s: %v", slug, err)
	}

	// Check if there are any disabled sections
	hasDisabled := false
	for _, section := range variant.Sections {
		if !section.Enabled {
			hasDisabled = true
			break
		}
	}

	handler := contenthttp.Admin(contentHTTPDependencies(cs))
	req := httptest.NewRequest(http.MethodGet, "/admin/variants/"+slug+"/sections", nil)
	req = mux.SetURLVars(req, map[string]string{"variant_slug": slug})

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	testutil.RequireHTTPStatus(t, w, http.StatusOK)

	var resp map[string]interface{}
	decodeJSONResponse(t, w.Body.Bytes(), &resp)

	sections, ok := resp["sections"].([]interface{})
	if !ok {
		t.Fatal("expected 'sections' to be an array")
	}

	// If variant has disabled sections, verify admin endpoint includes them
	if hasDisabled {
		foundDisabled := false
		for _, s := range sections {
			section, ok := s.(map[string]interface{})
			if !ok {
				continue
			}
			if enabled, exists := section["enabled"]; exists && !enabled.(bool) {
				foundDisabled = true
				break
			}
		}
		if !foundDisabled {
			t.Error("expected admin endpoint to include disabled sections")
		}
	}
}

func TestHandleGetSections_MissingSlug(t *testing.T) {
	cs := setupTestConfigStore(t)

	handler := contenthttp.Admin(contentHTTPDependencies(cs))

	req := httptest.NewRequest(http.MethodGet, "/admin/variants//sections", nil)
	req = mux.SetURLVars(req, map[string]string{"variant_slug": ""})

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	testutil.RequireHTTPStatus(t, w, http.StatusBadRequest)
}

func TestHandleGetSections_NotFound(t *testing.T) {
	cs := setupTestConfigStore(t)

	handler := contenthttp.Admin(contentHTTPDependencies(cs))

	req := httptest.NewRequest(http.MethodGet, "/admin/variants/nonexistent_variant/sections", nil)
	req = mux.SetURLVars(req, map[string]string{"variant_slug": "nonexistent_variant"})

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	testutil.RequireHTTPStatus(t, w, http.StatusNotFound)
}

// ============================================================================
// Additional Edge Case Tests
// ============================================================================

func TestHandleGetSections_ResponseFormat(t *testing.T) {
	cs := setupTestConfigStore(t)

	variants := cs.ListVariants()
	if len(variants) == 0 {
		t.Fatal("tracked test configuration must contain at least one variant")
	}
	slug := variants[0].Variant.Slug

	handler := contenthttp.Admin(contentHTTPDependencies(cs))
	req := httptest.NewRequest(http.MethodGet, "/admin/variants/"+slug+"/sections", nil)
	req = mux.SetURLVars(req, map[string]string{"variant_slug": slug})

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	testutil.RequireHTTPStatus(t, w, http.StatusOK)

	// Verify Content-Type header
	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type 'application/json', got '%s'", contentType)
	}
}

func TestLegacyPublicSnapshotsAreNotPublicationRoutes(t *testing.T) { // [REQ:LP-PRES-012]
	store := setupTestConfigStore(t)
	server := &Server{router: mux.NewRouter(), configStore: store, sessionManager: NewMockSessionManager()}
	registerVariantRoutes(server)
	for _, route := range []string{"/api/v1/public/variants/control", "/api/v1/public/variants/control/sections", "/api/v1/public/variants/missing/sections"} {
		response := httptest.NewRecorder()
		server.router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, route, nil))
		if response.Code != http.StatusNotFound {
			t.Fatalf("legacy public snapshot route %s remains exposed: %d", route, response.Code)
		}
	}
	for _, route := range []string{"/api/v1/variants/control/sections", "/api/v1/variants/select"} {
		response := httptest.NewRecorder()
		server.router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, route, nil))
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("private snapshot/legacy selector bypassed administrator auth: %s status=%d", route, response.Code)
		}
	}
}
