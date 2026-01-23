package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
)

// --- handleGetVariantSEO Tests ---

func TestHandleGetVariantSEO_Success(t *testing.T) {
	cs := setupTestConfigStore(t)
	variants := cs.ListVariants()
	if len(variants) == 0 {
		t.Skip("No variants available for testing")
	}

	seoService := NewSEOService(cs)
	handler := handleGetVariantSEO(seoService)

	slug := variants[0].Variant.Slug

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/seo/{slug}", handler).Methods("GET")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/seo/"+slug, nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	// Verify response has expected fields
	var resp SEOResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.SiteName == "" {
		t.Error("Expected site_name in response")
	}
	if resp.Title == "" {
		t.Error("Expected title in response")
	}
}

func TestHandleGetVariantSEO_NotFound(t *testing.T) {
	cs := setupTestConfigStore(t)
	seoService := NewSEOService(cs)
	handler := handleGetVariantSEO(seoService)

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/seo/{slug}", handler).Methods("GET")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/seo/nonexistent-slug", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d for not found, got %d", http.StatusNotFound, w.Code)
	}
}

func TestHandleGetVariantSEO_EmptySlug(t *testing.T) {
	cs := setupTestConfigStore(t)
	seoService := NewSEOService(cs)
	handler := handleGetVariantSEO(seoService)

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/seo/{slug}", handler).Methods("GET")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/seo/", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Empty slug should not match the route or return 400
	if w.Code == http.StatusOK {
		t.Error("Expected error for empty slug")
	}
}

// --- handleUpdateVariantSEOConfigStore Tests ---

func TestHandleUpdateVariantSEO_NotFound(t *testing.T) {
	cs := setupTestConfigStore(t)
	handler := handleUpdateVariantSEOConfigStore(cs)

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/admin/seo/{slug}", handler).Methods("PUT")

	body := `{"title": "New Title"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/seo/nonexistent", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestHandleUpdateVariantSEO_InvalidJSON(t *testing.T) {
	cs := setupTestConfigStore(t)
	handler := handleUpdateVariantSEOConfigStore(cs)

	variants := cs.ListVariants()
	if len(variants) == 0 {
		t.Skip("No variants available for testing")
	}

	slug := variants[0].Variant.Slug

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/admin/seo/{slug}", handler).Methods("PUT")

	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/seo/"+slug, strings.NewReader("{invalid"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d for invalid JSON, got %d", http.StatusBadRequest, w.Code)
	}
}

// --- handleSitemapXML Tests ---

func TestHandleSitemapXML_Success(t *testing.T) {
	cs := setupTestConfigStore(t)
	seoService := NewSEOService(cs)
	handler := handleSitemapXML(seoService)

	req := httptest.NewRequest(http.MethodGet, "/sitemap.xml", nil)
	req.Host = "example.com"
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	// Check Content-Type
	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "application/xml") {
		t.Errorf("Expected XML Content-Type, got '%s'", contentType)
	}

	// Check XML structure
	body := w.Body.String()
	if !strings.Contains(body, "<?xml version") {
		t.Error("Expected XML declaration in response")
	}
	if !strings.Contains(body, "<urlset") {
		t.Error("Expected urlset element in sitemap")
	}
	if !strings.Contains(body, "<url>") {
		t.Error("Expected url elements in sitemap")
	}
}

func TestHandleSitemapXML_UsesCanonicalURL(t *testing.T) {
	cs := setupTestConfigStore(t)

	// Update branding to have a canonical URL
	branding := cs.GetBranding()
	canonicalURL := "https://mysite.com"
	branding.CanonicalBaseURL = &canonicalURL
	_ = cs.SaveBranding(branding)

	seoService := NewSEOService(cs)
	handler := handleSitemapXML(seoService)

	req := httptest.NewRequest(http.MethodGet, "/sitemap.xml", nil)
	req.Host = "localhost:8080"
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "https://mysite.com") {
		t.Errorf("Expected canonical URL in sitemap, got: %s", body)
	}
}

// --- handleRobotsTXT Tests ---

func TestHandleRobotsTXT_Success(t *testing.T) {
	cs := setupTestConfigStore(t)
	seoService := NewSEOService(cs)
	handler := handleRobotsTXT(seoService)

	req := httptest.NewRequest(http.MethodGet, "/robots.txt", nil)
	req.Host = "example.com"
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Check Content-Type
	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/plain") {
		t.Errorf("Expected text/plain Content-Type, got '%s'", contentType)
	}

	body := w.Body.String()
	if !strings.Contains(body, "User-agent") {
		t.Error("Expected User-agent directive in robots.txt")
	}
}

func TestHandleRobotsTXT_IncludesSitemap(t *testing.T) {
	cs := setupTestConfigStore(t)

	// Update branding to have a canonical URL
	branding := cs.GetBranding()
	canonicalURL := "https://mysite.com"
	branding.CanonicalBaseURL = &canonicalURL
	branding.RobotsTxt = nil // Clear any custom robots.txt
	_ = cs.SaveBranding(branding)

	seoService := NewSEOService(cs)
	handler := handleRobotsTXT(seoService)

	req := httptest.NewRequest(http.MethodGet, "/robots.txt", nil)
	req.Host = "localhost:8080"
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Sitemap:") {
		t.Errorf("Expected Sitemap directive in robots.txt, got: %s", body)
	}
	if !strings.Contains(body, "https://mysite.com/sitemap.xml") {
		t.Errorf("Expected canonical sitemap URL, got: %s", body)
	}
}

func TestHandleRobotsTXT_UsesCustomRobotsTxt(t *testing.T) {
	cs := setupTestConfigStore(t)

	// Set custom robots.txt
	branding := cs.GetBranding()
	customRobots := "User-agent: Googlebot\nDisallow: /private/\n"
	branding.RobotsTxt = &customRobots
	_ = cs.SaveBranding(branding)

	seoService := NewSEOService(cs)
	handler := handleRobotsTXT(seoService)

	req := httptest.NewRequest(http.MethodGet, "/robots.txt", nil)
	req.Host = "localhost:8080"
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Googlebot") {
		t.Errorf("Expected custom robots.txt content, got: %s", body)
	}
	if !strings.Contains(body, "Disallow: /private/") {
		t.Errorf("Expected custom disallow rule, got: %s", body)
	}
}

// --- SEOService Tests ---

func TestSEOService_VariantSEO_Success(t *testing.T) {
	cs := setupTestConfigStore(t)
	variants := cs.ListVariants()
	if len(variants) == 0 {
		t.Skip("No variants available for testing")
	}

	seoService := NewSEOService(cs)
	slug := variants[0].Variant.Slug

	resp, err := seoService.VariantSEO(slug)
	if err != nil {
		t.Fatalf("VariantSEO failed: %v", err)
	}

	if resp.SiteName == "" {
		t.Error("Expected site_name in response")
	}
}

func TestSEOService_VariantSEO_NotFound(t *testing.T) {
	cs := setupTestConfigStore(t)
	seoService := NewSEOService(cs)

	_, err := seoService.VariantSEO("nonexistent-slug")
	if err == nil {
		t.Error("Expected error for nonexistent variant")
	}
}

func TestSEOService_SitemapXML_AllVariants(t *testing.T) {
	cs := setupTestConfigStore(t)

	// Clear any CanonicalBaseURL to ensure fallback is used
	branding := cs.GetBranding()
	branding.CanonicalBaseURL = nil
	_ = cs.SaveBranding(branding)

	seoService := NewSEOService(cs)

	sitemap, err := seoService.SitemapXML("https://example.com")
	if err != nil {
		t.Fatalf("SitemapXML failed: %v", err)
	}

	if !strings.Contains(sitemap, "https://example.com/") {
		t.Error("Expected root URL in sitemap")
	}
}

func TestSEOService_RobotsTXT_IncludesSitemap(t *testing.T) {
	cs := setupTestConfigStore(t)
	seoService := NewSEOService(cs)

	robots, err := seoService.RobotsTXT("https://example.com")
	if err != nil {
		t.Fatalf("RobotsTXT failed: %v", err)
	}

	if !strings.Contains(robots, "User-agent") {
		t.Error("Expected User-agent directive")
	}
	if !strings.Contains(robots, "Sitemap:") {
		t.Error("Expected Sitemap directive")
	}
}
