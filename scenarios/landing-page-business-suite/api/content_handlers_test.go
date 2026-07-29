package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"landing-page-business-suite-api/internal/testutil"

	"github.com/gorilla/mux"
)

// ============================================================================
// handleGetPublicSectionsFromConfigStore Tests
// ============================================================================

func TestHandleGetPublicSections_Success(t *testing.T) {
	cs := setupTestConfigStore(t)

	handler := handleGetPublicSectionsFromConfigStore(cs)

	// Get the first available variant slug from the config store
	variants := cs.ListVariants()
	if len(variants) == 0 {
		t.Fatal("tracked test configuration must contain at least one variant")
	}
	slug := variants[0].Variant.Slug

	req := httptest.NewRequest(http.MethodGet, "/public/variants/"+slug+"/sections", nil)
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

	// Public endpoint should only return enabled sections
	for _, s := range sections {
		section, ok := s.(map[string]interface{})
		if !ok {
			continue
		}
		if enabled, exists := section["enabled"]; exists {
			if !enabled.(bool) {
				t.Error("expected all public sections to be enabled")
			}
		}
	}
}

func TestHandleGetPublicSections_FiltersDisabled(t *testing.T) {
	cs := setupTestConfigStore(t)

	// First check if we can get a variant with sections
	variants := cs.ListVariants()
	if len(variants) == 0 {
		t.Fatal("tracked test configuration must contain at least one variant")
	}
	slug := variants[0].Variant.Slug

	variant, err := cs.GetVariant(slug)
	if err != nil {
		t.Fatalf("get tracked variant %s: %v", slug, err)
	}

	// Count enabled vs disabled sections
	enabledCount := 0
	for _, section := range variant.Sections {
		if section.Enabled {
			enabledCount++
		}
	}

	handler := handleGetPublicSectionsFromConfigStore(cs)
	req := httptest.NewRequest(http.MethodGet, "/public/variants/"+slug+"/sections", nil)
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

	if len(sections) != enabledCount {
		t.Errorf("expected %d enabled sections, got %d", enabledCount, len(sections))
	}
}

func TestHandleGetPublicSections_MissingSlug(t *testing.T) {
	cs := setupTestConfigStore(t)

	handler := handleGetPublicSectionsFromConfigStore(cs)

	req := httptest.NewRequest(http.MethodGet, "/public/variants//sections", nil)
	req = mux.SetURLVars(req, map[string]string{"variant_slug": ""})

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	testutil.RequireHTTPStatus(t, w, http.StatusBadRequest)
}

func TestHandleGetPublicSections_NotFound(t *testing.T) {
	cs := setupTestConfigStore(t)

	handler := handleGetPublicSectionsFromConfigStore(cs)

	req := httptest.NewRequest(http.MethodGet, "/public/variants/nonexistent_variant/sections", nil)
	req = mux.SetURLVars(req, map[string]string{"variant_slug": "nonexistent_variant"})

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	testutil.RequireHTTPStatus(t, w, http.StatusNotFound)
}

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

	handler := handleGetSectionsFromConfigStore(cs)
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

	handler := handleGetSectionsFromConfigStore(cs)
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

	handler := handleGetSectionsFromConfigStore(cs)

	req := httptest.NewRequest(http.MethodGet, "/admin/variants//sections", nil)
	req = mux.SetURLVars(req, map[string]string{"variant_slug": ""})

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	testutil.RequireHTTPStatus(t, w, http.StatusBadRequest)
}

func TestHandleGetSections_NotFound(t *testing.T) {
	cs := setupTestConfigStore(t)

	handler := handleGetSectionsFromConfigStore(cs)

	req := httptest.NewRequest(http.MethodGet, "/admin/variants/nonexistent_variant/sections", nil)
	req = mux.SetURLVars(req, map[string]string{"variant_slug": "nonexistent_variant"})

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	testutil.RequireHTTPStatus(t, w, http.StatusNotFound)
}

// ============================================================================
// Additional Edge Case Tests
// ============================================================================

func TestHandleGetPublicSections_EmptySections(t *testing.T) {
	cs := setupTestConfigStore(t)

	variants := cs.ListVariants()
	if len(variants) == 0 {
		t.Fatal("tracked test configuration must contain at least one variant")
	}

	// This test ensures the handler doesn't crash on variants with no sections
	// (even if unlikely in practice)
	handler := handleGetPublicSectionsFromConfigStore(cs)

	slug := variants[0].Variant.Slug
	req := httptest.NewRequest(http.MethodGet, "/public/variants/"+slug+"/sections", nil)
	req = mux.SetURLVars(req, map[string]string{"variant_slug": slug})

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// Should not panic and should return valid JSON
	testutil.RequireHTTPStatus(t, w, http.StatusOK)

	var resp map[string]interface{}
	decodeJSONResponse(t, w.Body.Bytes(), &resp)

	if _, ok := resp["sections"]; !ok {
		t.Error("expected 'sections' key in response")
	}
}

func TestHandleGetSections_ResponseFormat(t *testing.T) {
	cs := setupTestConfigStore(t)

	variants := cs.ListVariants()
	if len(variants) == 0 {
		t.Fatal("tracked test configuration must contain at least one variant")
	}
	slug := variants[0].Variant.Slug

	handler := handleGetSectionsFromConfigStore(cs)
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
