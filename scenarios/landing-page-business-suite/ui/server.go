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
	"strings"
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
	h := newUIHandlerWithHealth(root, httputil.NewSingleHostReverseProxy(apiURL), func() error {
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
	})
	log.Printf("landing-page-business-suite UI listening on %s", port)
	log.Fatal(http.ListenAndServe(":"+port, h))
}

func newUIHandler(root string, apiProxy http.Handler) http.Handler {
	return newUIHandlerWithHealth(root, apiProxy, nil)
}

func newUIHandlerWithHealth(root string, apiProxy http.Handler, presentationCheck func() error) http.Handler {
	files := http.FileServer(http.Dir(root))
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			writeUIHealthStatus(w, presentationCheck)
			return
		}
		// Connect procedures are mounted at the UI origin by the shared
		// scenario client. Forward only the exact Connect URL shape so SPA
		// routes and static assets remain owned by the UI server.
		if isConnectProcedurePath(r.URL.Path) || strings.HasPrefix(r.URL.Path, "/api/") {
			apiProxy.ServeHTTP(w, r)
			return
		}
		files.ServeHTTP(w, r)
	})
	return h
}
