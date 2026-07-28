package seo

import (
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

func TestSitemapUsesRequestSchemeAndHost(t *testing.T) {
	seen := ""
	deps := testDependencies()
	deps.Sitemap = func(base string) (string, error) { seen = base; return "<xml/>", nil }
	w := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "http://example.test/sitemap.xml", nil)
	Sitemap(deps).ServeHTTP(w, request)
	if seen != "http://example.test" || w.Header().Get("Content-Type") != "application/xml; charset=utf-8" {
		t.Fatalf("seen=%q content-type=%q", seen, w.Header().Get("Content-Type"))
	}
}

func TestRobotsFallsBackWhenBrandingFails(t *testing.T) {
	deps := testDependencies()
	deps.Robots = func(string) (string, error) { return "", errors.New("branding unavailable") }
	w := httptest.NewRecorder()
	Robots(deps).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "http://example.test/robots.txt", nil))
	if w.Code != http.StatusOK || w.Body.String() != "User-agent: *\nAllow: /\n" {
		t.Fatalf("status=%d body=%q", w.Code, w.Body.String())
	}
}

func testDependencies() Dependencies {
	return Dependencies{
		VariantSEO: func(string) (any, error) { return map[string]any{}, nil }, Update: func(string, json.RawMessage) (bool, error) { return true, nil }, Sitemap: func(string) (string, error) { return "", nil }, Robots: func(string) (string, error) { return "", nil }, Path: func(*http.Request, string) (string, bool) { return "spring", true }, DecodeJSON: func(http.ResponseWriter, *http.Request, any) bool { return true }, WriteJSON: func(http.ResponseWriter, any) {}, WriteError: func(http.ResponseWriter, int, string, string) {}, Log: func(string, map[string]any) {}, Now: func() time.Time { return time.Unix(0, 0) },
	}
}
