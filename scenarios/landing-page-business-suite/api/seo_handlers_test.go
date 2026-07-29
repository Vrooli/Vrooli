package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	lpbsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/landing_page_business_suite_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/shared"
	"landing-page-business-suite-api/internal/testutil"
)

func TestSEOConnectHandler_GetVariantSEO(t *testing.T) {
	cs := setupTestConfigStore(t)
	variants := cs.ListVariants()
	if len(variants) == 0 {
		t.Fatal("tracked test configuration must contain at least one variant")
	}
	handler := newSEOConnectHandler(NewSEOService(cs), cs)
	response, err := handler.GetVariantSEO(context.Background(), connect.NewRequest(&lpbsv1.GetVariantSEORequest{Slug: variants[0].Variant.Slug}))
	testutil.RequireNoError(t, err)
	if response.Msg.GetSiteName() == "" || response.Msg.GetTitle() == "" {
		t.Fatalf("response must preserve resolved branding defaults: %#v", response.Msg)
	}
}

func TestSEOConnectHandler_GetVariantSEORejectsInvalidAndMissingSlugs(t *testing.T) {
	cs := setupTestConfigStore(t)
	handler := newSEOConnectHandler(NewSEOService(cs), cs)
	for _, request := range []*lpbsv1.GetVariantSEORequest{{}, {Slug: "missing"}} {
		_, err := handler.GetVariantSEO(context.Background(), connect.NewRequest(request))
		if got := connect.CodeOf(err); got != connect.CodeInvalidArgument && got != connect.CodeNotFound {
			t.Fatalf("GetVariantSEO(%q) code = %s, want invalid_argument or not_found (err=%v)", request.GetSlug(), got, err)
		}
	}
}

func TestSEOConnectHandler_UpdateVariantSEOPersistsProtoConfig(t *testing.T) {
	cs := setupTestConfigStore(t)
	variants := cs.ListVariants()
	if len(variants) == 0 {
		t.Fatal("tracked test configuration must contain at least one variant")
	}
	handler := newSEOConnectHandler(NewSEOService(cs), cs)
	response, err := handler.UpdateVariantSEO(context.Background(), connect.NewRequest(&lpbsv1.UpdateVariantSEORequest{
		Slug:   variants[0].Variant.Slug,
		Config: &sharedv1.VariantSEOConfig{Title: "Connect title", OgImageUrl: "https://example.test/og.png"},
	}))
	testutil.RequireNoError(t, err)
	if !response.Msg.GetSuccess() || response.Msg.GetUpdatedAt() == "" {
		t.Fatalf("unexpected update response: %#v", response.Msg)
	}
	saved, err := cs.GetVariant(variants[0].Variant.Slug)
	testutil.RequireNoError(t, err)
	if !strings.Contains(string(saved.Variant.SEOConfig), `"og_image_url":"https://example.test/og.png"`) {
		t.Fatalf("stored config must retain the legacy JSON contract: %s", saved.Variant.SEOConfig)
	}
}

func TestSEOConnectRoutesServeGeneratedClient(t *testing.T) {
	cs := setupTestConfigStore(t)
	variants := cs.ListVariants()
	if len(variants) == 0 {
		t.Fatal("tracked test configuration must contain at least one variant")
	}
	router := mux.NewRouter()
	registerSEOConnectRoutes(router, NewSEOService(cs), cs, func(handler http.HandlerFunc) http.HandlerFunc { return handler })
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	client := lpbsconnect.NewSeoServiceClient(server.Client(), server.URL)
	read, err := client.GetVariantSEO(context.Background(), connect.NewRequest(&lpbsv1.GetVariantSEORequest{Slug: variants[0].Variant.Slug}))
	testutil.RequireNoError(t, err)
	if read.Msg.GetTitle() == "" {
		t.Fatalf("generated client returned incomplete SEO response: %#v", read.Msg)
	}
	updated, err := client.UpdateVariantSEO(context.Background(), connect.NewRequest(&lpbsv1.UpdateVariantSEORequest{
		Slug: variants[0].Variant.Slug, Config: &sharedv1.VariantSEOConfig{Description: "Updated through Connect"},
	}))
	testutil.RequireNoError(t, err)
	if !updated.Msg.GetSuccess() {
		t.Fatalf("generated client update response = %#v", updated.Msg)
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

	testutil.RequireHTTPStatus(t, w, http.StatusOK)

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

	testutil.RequireHTTPStatus(t, w, http.StatusOK)

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

	testutil.RequireHTTPStatus(t, w, http.StatusOK)

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

	testutil.RequireHTTPStatus(t, w, http.StatusOK)

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

	testutil.RequireHTTPStatus(t, w, http.StatusOK)

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
		t.Fatal("tracked test configuration must contain at least one variant")
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
