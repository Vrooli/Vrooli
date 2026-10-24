// Package publicproxy contains the deliberately narrow anonymous edge routes
// used by targets that cannot reach the control plane over its LAN address.
// The route is still authenticated by the target's Device Sync Hub token.
package publicproxy

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/discovery"
)

const maxBytes int64 = 2 << 30

// DeviceSyncContent returns an HTTP handler for the single public download
// edge needed by Bridge artifact placement. It never accepts an upload and it
// never forwards arbitrary paths or headers to the local hub.
func DeviceSyncContent(baseURL string, client *http.Client) http.Handler {
	return handler{baseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"), client: client, resolver: discovery.NewResolver(discovery.ResolverConfig{})}
}

type handler struct {
	baseURL  string
	client   *http.Client
	resolver *discovery.Resolver
}

func (h handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodDelete {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	token := strings.TrimSpace(r.Header.Get("X-Device-Token"))
	if token == "" {
		http.Error(w, "device token required", http.StatusUnauthorized)
		return
	}
	const (
		publicPrefix   = "/public/device-sync-hub/api/v1/transfer/items"
		originPrefix   = "/device-sync-hub/api/v1/transfer/items"
		strippedPrefix = "/api/v1/transfer/items"
	)
	prefix := ""
	for _, candidate := range []string{publicPrefix, originPrefix, strippedPrefix} {
		if strings.HasPrefix(r.URL.Path, candidate) {
			prefix = candidate
			break
		}
	}
	if prefix == "" {
		http.NotFound(w, r)
		return
	}
	remainder := strings.TrimPrefix(r.URL.Path, prefix)
	if remainder == "" && r.Method == http.MethodGet {
		h.proxyJSON(w, r, h.baseURL, "/api/v1/transfer/items"+querySuffix(r), token, http.MethodGet)
		return
	}
	if !strings.HasPrefix(remainder, "/") {
		http.NotFound(w, r)
		return
	}
	remainder = strings.TrimPrefix(remainder, "/")
	id := strings.TrimSuffix(remainder, "/content")
	isContent := strings.HasSuffix(remainder, "/content")
	if _, err := uuid.Parse(id); err != nil || (!isContent && r.Method != http.MethodDelete) || (isContent && r.Method != http.MethodGet) {
		http.NotFound(w, r)
		return
	}
	baseURL := h.baseURL
	if baseURL == "" && h.resolver != nil {
		baseURL, _ = h.resolver.ResolveScenarioURLDefault(r.Context(), "device-sync-hub")
		baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	}
	if baseURL == "" {
		http.Error(w, "device-sync-hub proxy is not configured", http.StatusServiceUnavailable)
		return
	}
	base, err := url.Parse(baseURL)
	if err != nil || (base.Scheme != "http" && base.Scheme != "https") || base.Host == "" {
		http.Error(w, "device-sync-hub proxy is misconfigured", http.StatusServiceUnavailable)
		return
	}
	endpoint := strings.TrimRight(baseURL, "/") + "/api/v1/transfer/items/" + url.PathEscape(id)
	if isContent {
		endpoint += "/content"
	}
	request, err := http.NewRequestWithContext(r.Context(), r.Method, endpoint, nil)
	if err != nil {
		http.Error(w, "unable to create hub request", http.StatusBadGateway)
		return
	}
	request.Header.Set("X-Device-Token", token)
	client := h.client
	if client == nil {
		client = http.DefaultClient
	}
	response, err := client.Do(request)
	if err != nil {
		http.Error(w, "device-sync-hub unavailable", http.StatusBadGateway)
		return
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		status := response.StatusCode
		if status < 400 {
			status = http.StatusBadGateway
		}
		http.Error(w, fmt.Sprintf("device-sync-hub returned %s", response.Status), status)
		return
	}
	if !isContent {
		defer response.Body.Close()
		w.Header().Set("Content-Type", response.Header.Get("Content-Type"))
		w.WriteHeader(response.StatusCode)
		_, _ = io.Copy(w, io.LimitReader(response.Body, 1<<20))
		return
	}
	if response.ContentLength > maxBytes {
		http.Error(w, "artifact exceeds proxy limit", http.StatusRequestEntityTooLarge)
		return
	}
	for _, name := range []string{"Content-Type", "Content-Disposition", "ETag"} {
		if value := response.Header.Get(name); value != "" {
			w.Header().Set(name, value)
		}
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(response.StatusCode)
	_, _ = io.Copy(w, io.LimitReader(response.Body, maxBytes+1))
}

func querySuffix(r *http.Request) string {
	if r.URL.RawQuery == "" {
		return ""
	}
	return "?" + r.URL.RawQuery
}

func (h handler) proxyJSON(w http.ResponseWriter, r *http.Request, baseURL, path, token, method string) {
	if baseURL == "" && h.resolver != nil {
		baseURL, _ = h.resolver.ResolveScenarioURLDefault(r.Context(), "device-sync-hub")
	}
	endpoint := strings.TrimRight(baseURL, "/") + path
	request, err := http.NewRequestWithContext(r.Context(), method, endpoint, nil)
	if err != nil {
		http.Error(w, "unable to create hub request", http.StatusBadGateway)
		return
	}
	request.Header.Set("X-Device-Token", token)
	client := h.client
	if client == nil {
		client = http.DefaultClient
	}
	response, err := client.Do(request)
	if err != nil {
		http.Error(w, "device-sync-hub unavailable", http.StatusBadGateway)
		return
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		http.Error(w, fmt.Sprintf("device-sync-hub returned %s", response.Status), response.StatusCode)
		return
	}
	w.Header().Set("Content-Type", response.Header.Get("Content-Type"))
	w.WriteHeader(response.StatusCode)
	_, _ = io.Copy(w, io.LimitReader(response.Body, 1<<20))
}
