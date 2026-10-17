package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUIHandlerHealthMatchesRecognizedSchema(t *testing.T) {
	t.Setenv(buildIdentityEnv, "sha256:test-identity")
	h := newUIHandler(t.TempDir(), http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("health check must not reach the API proxy")
	}))

	response := httptest.NewRecorder()
	h.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/health", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("health status = %d, want %d", response.Code, http.StatusOK)
	}

	var body struct {
		Status        string `json:"status"`
		Service       string `json:"service"`
		Timestamp     string `json:"timestamp"`
		BuildIdentity string `json:"build_identity"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode health response: %v", err)
	}
	if body.Status != "healthy" {
		t.Fatalf("status = %q, want healthy", body.Status)
	}
	if body.Service == "" {
		t.Fatal("service must be set for the control plane to recognize the response")
	}
	if body.Timestamp == "" {
		t.Fatal("timestamp must be set for the control plane to recognize the response")
	}
	if body.BuildIdentity != "sha256:test-identity" {
		t.Fatalf("build_identity = %q, want sha256:test-identity", body.BuildIdentity)
	}
}

func TestUIHandlerHealthFailsWhenPublishedPresentationIsUnavailable(t *testing.T) {
	h := newUIHandlerWithHealth(t.TempDir(), http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}), func() error {
		return errors.New("presentation store unavailable")
	})
	response := httptest.NewRecorder()
	h.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/health", nil))
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("health status = %d, want %d", response.Code, http.StatusServiceUnavailable)
	}
	var body struct {
		Status    string `json:"status"`
		Readiness bool   `json:"readiness"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode degraded health response: %v", err)
	}
	if body.Status != "degraded" || body.Readiness {
		t.Fatalf("degraded health = %+v", body)
	}
}

func TestUIHandlerProxiesConnectAndAPIRequests(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "index.html"), []byte("ui"), 0o600); err != nil {
		t.Fatal(err)
	}
	h := newUIHandler(root, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/landing_page_business_suite.v1.LandingConfigService/GetLandingConfig" || r.URL.Path == "/api/v1/health" {
			w.WriteHeader(http.StatusCreated)
			return
		}
		http.Error(w, "unexpected path", http.StatusBadGateway)
	}))

	for _, path := range []string{
		"/landing_page_business_suite.v1.LandingConfigService/GetLandingConfig",
		"/api/v1/health",
	} {
		response := httptest.NewRecorder()
		h.ServeHTTP(response, httptest.NewRequest(http.MethodPost, path, nil))
		if response.Code != http.StatusCreated {
			t.Fatalf("%s status = %d, want %d", path, response.Code, http.StatusCreated)
		}
	}
}

func TestUIHandlerServesStaticRoutesLocally(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "index.html"), []byte("ui"), 0o600); err != nil {
		t.Fatal(err)
	}
	h := newUIHandler(root, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unexpected proxy", http.StatusBadGateway)
	}))
	response := httptest.NewRecorder()
	h.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	if response.Code != http.StatusOK || response.Body.String() != "ui" {
		t.Fatalf("root response = %d %q", response.Code, response.Body.String())
	}
}

func TestUIHandlerServesClientEntryForDirectPublicRoutes(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "index.html"), []byte("<html><head></head><body>ui</body></html>"), 0o600); err != nil {
		t.Fatal(err)
	}
	h := newUIHandler(root, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unexpected proxy", http.StatusBadGateway)
	}))
	for _, path := range []string{"/apps/aquila", "/apps/aquila/download", "/checkout", "/contact", "/privacy", "/terms", "/thank-you"} {
		response := httptest.NewRecorder()
		h.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `<base href="/">`) {
			t.Fatalf("%s response = %d %q", path, response.Code, response.Body.String())
		}
	}
}

func TestUIHandlerServesBrandedNotFoundWithRealStatus(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "index.html"), []byte("<html><head></head><body>ui</body></html>"), 0o600); err != nil {
		t.Fatal(err)
	}
	h := newUIHandler(root, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unexpected proxy", http.StatusBadGateway)
	}))
	for path, wantEntry := range map[string]bool{"/no-such-page": true, "/pricing/old/link": true, "/assets/missing.js": false} {
		response := httptest.NewRecorder()
		h.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusNotFound {
			t.Fatalf("%s status = %d, want 404", path, response.Code)
		}
		if got := strings.Contains(response.Body.String(), "<body>ui</body>"); got != wantEntry {
			t.Fatalf("%s served client entry = %v, want %v: %q", path, got, wantEntry, response.Body.String())
		}
	}
}

func TestUIHandlerForwardsCrawlerDocumentsToAPI(t *testing.T) {
	var proxied []string
	h := newUIHandler(t.TempDir(), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		proxied = append(proxied, r.URL.Path)
		_, _ = w.Write([]byte("crawler"))
	}))
	for _, path := range []string{"/robots.txt", "/sitemap.xml"} {
		response := httptest.NewRecorder()
		h.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusOK || response.Body.String() != "crawler" {
			t.Fatalf("%s response = %d %q", path, response.Code, response.Body.String())
		}
	}
	if strings.Join(proxied, ",") != "/robots.txt,/sitemap.xml" {
		t.Fatalf("proxied = %v", proxied)
	}
}

func TestUIHandlerMakesSocialImagesAbsoluteFromConfiguredBase(t *testing.T) {
	root := t.TempDir()
	index := `<html><head><meta property="og:image" content="/public/og-image.jpg" /><meta name="twitter:image" content="./public/og-image.jpg" /><link rel="icon" href="/public/favicon.ico" /></head><body>ui</body></html>`
	if err := os.WriteFile(filepath.Join(root, "index.html"), []byte(index), 0o600); err != nil {
		t.Fatal(err)
	}
	for base, want := range map[string]string{
		"https://shop.example.test/": `content="https://shop.example.test/public/og-image.jpg"`,
		"":                           `public/og-image.jpg"`,
		"javascript:alert(1)":        `public/og-image.jpg"`,
	} {
		h := newUIHandlerWithOptions(root, http.NotFoundHandler(), nil, func() string { return base })
		response := httptest.NewRecorder()
		h.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
		body := response.Body.String()
		if strings.Count(body, want) != 2 || !strings.Contains(body, `href="/public/favicon.ico"`) {
			t.Fatalf("base %q: body = %s", base, body)
		}
	}
}
