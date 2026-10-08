package seo

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestVariantReturnsNotFoundForMissingVariant(t *testing.T) {
	status := 0
	deps := testDependencies()
	deps.VariantSEO = func(string) (any, error) { return nil, errors.New("variant not found") }
	deps.WriteError = func(_ http.ResponseWriter, got int, _, _ string) { status = got }
	Variant(deps).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/seo/spring", nil))
	if status != http.StatusNotFound {
		t.Fatalf("status=%d", status)
	}
}

func TestSitemapDoesNotTrustRequestHost(t *testing.T) {
	seen := ""
	deps := testDependencies()
	deps.Sitemap = func(_ context.Context, base string) (string, error) { seen = base; return "<xml/>", nil }
	w := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "http://example.test/sitemap.xml", nil)
	Sitemap(deps).ServeHTTP(w, request)
	if seen != "" || w.Header().Get("Content-Type") != "application/xml; charset=utf-8" {
		t.Fatalf("seen=%q content-type=%q", seen, w.Header().Get("Content-Type"))
	}
}

func TestSitemapReturnsUnavailableInsteadOfAdvertisingWithoutCanonical(t *testing.T) {
	deps := testDependencies()
	status := 0
	deps.Sitemap = func(context.Context, string) (string, error) {
		return "", errors.New("trusted canonical base URL is not configured")
	}
	deps.WriteError = func(_ http.ResponseWriter, code int, _, _ string) { status = code }
	w := httptest.NewRecorder()
	Sitemap(deps).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "http://attacker.example/sitemap.xml", nil))
	if status != http.StatusServiceUnavailable {
		t.Fatalf("status=%d, want explicit unavailable response", status)
	}
}

func TestRobotsFailsClosedWhenBrandingFails(t *testing.T) {
	deps := testDependencies()
	deps.Robots = func(context.Context, string) (string, error) { return "", errors.New("branding unavailable") }
	w := httptest.NewRecorder()
	Robots(deps).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "http://example.test/robots.txt", nil))
	if w.Code != http.StatusServiceUnavailable || w.Body.String() != "User-agent: *\nDisallow: /\n" {
		t.Fatalf("status=%d body=%q", w.Code, w.Body.String())
	}
}

func testDependencies() Dependencies {
	return Dependencies{
		VariantSEO: func(string) (any, error) { return map[string]any{}, nil }, Update: func(string, json.RawMessage) (bool, error) { return true, nil }, Sitemap: func(context.Context, string) (string, error) { return "", nil }, Robots: func(context.Context, string) (string, error) { return "", nil }, Path: func(*http.Request, string) (string, bool) { return "spring", true }, DecodeJSON: func(http.ResponseWriter, *http.Request, any) bool { return true }, WriteJSON: func(http.ResponseWriter, any) {}, WriteError: func(http.ResponseWriter, int, string, string) {}, Log: func(string, map[string]any) {}, Now: func() time.Time { return time.Unix(0, 0) },
	}
}
