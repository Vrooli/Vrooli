package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

const buildIdentityEnv = "VROOLI_BUILD_IDENTITY"

// writeUIHealth serves the Vrooli health response schema the control plane
// recognizes. A process that answers 200 with a non-conforming body is reported
// degraded ("invalid health response schema"), so the UI must speak the same
// envelope as the API: service, timestamp, and a known status.
func writeUIHealth(w http.ResponseWriter) {
	writeUIHealthStatus(w, nil)
}

func writeUIHealthStatus(w http.ResponseWriter, presentationCheck func() error) {
	response := map[string]any{
		"status":    "healthy",
		"service":   "landing-page-business-suite-ui",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"readiness": true,
	}
	if presentationCheck != nil {
		if err := presentationCheck(); err != nil {
			response["status"] = "degraded"
			response["readiness"] = false
			response["detail"] = "published presentation unavailable"
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(response)
			return
		}
	}
	if identity := strings.TrimSpace(os.Getenv(buildIdentityEnv)); identity != "" {
		response["build_identity"] = identity
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(response)
}

func isConnectProcedurePath(path string) bool {
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" || !strings.Contains(parts[0], ".") {
		return false
	}
	for _, part := range parts {
		for _, character := range part {
			if !(character >= 'a' && character <= 'z' || character >= 'A' && character <= 'Z' || character >= '0' && character <= '9' || character == '_' || character == '.') {
				return false
			}
		}
	}
	first := parts[1][0]
	return first >= 'A' && first <= 'Z'
}

func isPublicSPARoute(path string) bool {
	switch path {
	case "/contact", "/privacy", "/terms", "/thank-you", "/account", "/account/security":
		return true
	}
	if path == "/" || path == "/checkout" || path == "/feedback" || strings.HasPrefix(path, "/auth/") || strings.HasPrefix(path, "/admin") {
		return true
	}
	parts := strings.Split(strings.Trim(path, "/"), "/")
	return len(parts) == 2 && parts[0] == "apps" && parts[1] != "" || len(parts) == 3 && parts[0] == "apps" && parts[1] != "" && parts[2] == "download"
}

// isPageLikePath reports whether a path names a page rather than a static file.
func isPageLikePath(path string) bool {
	last := path[strings.LastIndex(path, "/")+1:]
	return !strings.Contains(last, ".")
}

// socialImageAttr matches same-origin social preview image tags in the entry document.
var socialImageAttr = regexp.MustCompile(`(<meta (?:property="og:image"|name="twitter:image") content=")\.?(/[^"]*)(")`)

// absolutizeSocialImages rewrites root-relative social preview image URLs
// against the configured canonical base. Link-preview crawlers do not run
// scripts and many reject relative image URLs. The base is operator
// configuration, never the request Host.
func absolutizeSocialImages(html, base string) string {
	parsed, err := url.Parse(strings.TrimSpace(base))
	if err != nil || parsed.Host == "" || (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.User != nil {
		return html
	}
	origin := strings.TrimRight(parsed.Scheme+"://"+parsed.Host+parsed.Path, "/")
	return socialImageAttr.ReplaceAllString(html, "${1}"+origin+"${2}${3}")
}

func serveSPAIndex(w http.ResponseWriter, r *http.Request, root string, status int, canonicalBase string) {
	indexPath := filepath.Join(root, "index.html")
	data, err := os.ReadFile(indexPath)
	if err != nil {
		http.Error(w, "Application not built. Run build command first.", http.StatusNotFound)
		return
	}
	html := string(data)
	if !strings.Contains(strings.ToLower(html), "<base ") {
		html = strings.Replace(html, "<head>", "<head><base href=\"/\">", 1)
	}
	if canonicalBase != "" {
		html = absolutizeSocialImages(html, canonicalBase)
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = io.WriteString(w, html)
}

func main() {
	port := os.Getenv("UI_PORT")
	if port == "" {
		port = "3000"
	}
	root := filepath.Join(filepath.Dir(os.Args[0]), "dist")
	if _, err := os.Stat(root); err != nil {
		root = "dist"
	}
	apiPort := os.Getenv("API_PORT")
	if apiPort == "" {
		apiPort = "8080"
	}
	apiURL, err := url.Parse("http://127.0.0.1:" + apiPort)
	if err != nil {
		log.Fatal(err)
	}
	h := newUIHandlerWithOptions(root, httputil.NewSingleHostReverseProxy(apiURL), func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		body := strings.NewReader(`{"route":"/","locale":"","variantSlug":"","visitorId":"health"}`)
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL.String()+"/landing_page_business_suite.v1.LandingConfigService/GetLandingConfig", body)
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := (&http.Client{Timeout: 3 * time.Second}).Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return fmt.Errorf("presentation RPC returned %s", resp.Status)
		}
		var payload struct {
			Presentation struct {
				Page struct {
					Blocks []json.RawMessage `json:"blocks"`
				} `json:"page"`
			} `json:"presentation"`
		}
		if err := json.NewDecoder(io.LimitReader(resp.Body, 8<<20)).Decode(&payload); err != nil {
			return err
		}
		if len(payload.Presentation.Page.Blocks) == 0 {
			return fmt.Errorf("presentation has no blocks")
		}
		return nil
	}, cachedCanonicalBase(apiURL.String()))
	log.Printf("landing-page-business-suite UI listening on %s", port)
	log.Fatal(http.ListenAndServe(":"+port, h))
}

// cachedCanonicalBase reads the public canonical base from branding, caching it
// briefly so an entry-document request never waits on the API for long.
func cachedCanonicalBase(apiBase string) func() string {
	var (
		mu      sync.Mutex
		value   string
		fetched time.Time
	)
	client := &http.Client{Timeout: 2 * time.Second}
	return func() string {
		mu.Lock()
		defer mu.Unlock()
		if !fetched.IsZero() && time.Since(fetched) < time.Minute {
			return value
		}
		fetched = time.Now()
		request, err := http.NewRequest(http.MethodPost, apiBase+"/landing_page_business_suite.v1.BrandingService/GetPublicBranding", strings.NewReader("{}"))
		if err != nil {
			return value
		}
		request.Header.Set("Content-Type", "application/json")
		response, err := client.Do(request)
		if err != nil {
			return value
		}
		defer response.Body.Close()
		var payload struct {
			Branding struct {
				CanonicalBaseURL string `json:"canonicalBaseUrl"`
			} `json:"branding"`
		}
		if response.StatusCode == http.StatusOK && json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&payload) == nil {
			value = payload.Branding.CanonicalBaseURL
		}
		return value
	}
}

func newUIHandler(root string, apiProxy http.Handler) http.Handler {
	return newUIHandlerWithHealth(root, apiProxy, nil)
}

func newUIHandlerWithHealth(root string, apiProxy http.Handler, presentationCheck func() error) http.Handler {
	return newUIHandlerWithOptions(root, apiProxy, presentationCheck, nil)
}

// newUIHandlerWithOptions serves the UI. canonicalBase, when set, returns the
// operator-configured public origin used to make social preview URLs absolute.
func newUIHandlerWithOptions(root string, apiProxy http.Handler, presentationCheck func() error, canonicalBase func() string) http.Handler {
	files := http.FileServer(http.Dir(root))
	serveIndex := func(w http.ResponseWriter, r *http.Request, status int) {
		base := ""
		if canonicalBase != nil {
			base = canonicalBase()
		}
		serveSPAIndex(w, r, root, status, base)
	}
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			writeUIHealthStatus(w, presentationCheck)
			return
		}
		// Connect procedures are mounted at the UI origin by the shared
		// scenario client. Forward only the exact Connect URL shape so SPA
		// routes and static assets remain owned by the UI server.
		// Crawler documents are owned by the API's SEO service, which renders
		// them from the configured canonical base and published routes.
		if isConnectProcedurePath(r.URL.Path) || strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/robots.txt" || r.URL.Path == "/sitemap.xml" {
			apiProxy.ServeHTTP(w, r)
			return
		}
		// The UI is a client-routed application. Serve its entry document for
		// known public routes so a direct download/detail link can be opened or
		// refreshed instead of being mistaken for a missing static file.
		if (r.Method == http.MethodGet || r.Method == http.MethodHead) && r.URL.Path == "/" {
			serveIndex(w, r, http.StatusOK)
			return
		}
		if r.Method == http.MethodGet || r.Method == http.MethodHead {
			if _, err := os.Stat(filepath.Join(root, filepath.Clean(r.URL.Path))); os.IsNotExist(err) {
				if isPublicSPARoute(r.URL.Path) {
					serveIndex(w, r, http.StatusOK)
					return
				}
				// An unknown page still gets the branded not-found experience,
				// but with a real 404 so crawlers never index it. Missing static
				// assets keep the plain file-server 404.
				if isPageLikePath(r.URL.Path) {
					serveIndex(w, r, http.StatusNotFound)
					return
				}
			}
		}
		files.ServeHTTP(w, r)
	})
	return h
}
