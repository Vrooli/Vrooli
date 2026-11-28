package publicproxy

import (
	"io"
	"net/http"
	"strings"

	"github.com/vrooli/api-core/discovery"
)

const maxRPCBytes int64 = 1 << 20

// DeviceSyncRPC proxies only the transfer metadata procedures needed for
// governed cleanup and inspection. The byte content edge remains separate so
// this surface cannot become a general public proxy.
func DeviceSyncRPC(baseURL string, client *http.Client) http.Handler {
	return rpcHandler{baseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"), client: client, resolver: discovery.NewResolver(discovery.ResolverConfig{})}
}

// DeviceSyncHealth exposes only the hub readiness probe under the same public
// prefix used as the CLI API base. It remains token-gated to avoid turning the
// public route into an unauthenticated service fingerprint.
func DeviceSyncHealth(baseURL string, client *http.Client) http.Handler {
	return healthHandler{baseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"), client: client, resolver: discovery.NewResolver(discovery.ResolverConfig{})}
}

type healthHandler struct {
	baseURL  string
	client   *http.Client
	resolver *discovery.Resolver
}

func (h healthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET required", http.StatusMethodNotAllowed)
		return
	}
	baseURL := h.baseURL
	if baseURL == "" && h.resolver != nil {
		baseURL, _ = h.resolver.ResolveScenarioURLDefault(r.Context(), "device-sync-hub")
	}
	if baseURL == "" {
		http.Error(w, "device-sync-hub proxy is not configured", http.StatusServiceUnavailable)
		return
	}
	request, err := http.NewRequestWithContext(r.Context(), http.MethodGet, strings.TrimRight(baseURL, "/")+"/health", nil)
	if err != nil {
		http.Error(w, "unable to create hub request", http.StatusBadGateway)
		return
	}
	request.Header.Set("X-Device-Token", r.Header.Get("X-Device-Token"))
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
	w.Header().Set("Content-Type", response.Header.Get("Content-Type"))
	w.WriteHeader(response.StatusCode)
	_, _ = io.Copy(w, io.LimitReader(response.Body, 64<<10))
}

type rpcHandler struct {
	baseURL  string
	client   *http.Client
	resolver *discovery.Resolver
}

func (h rpcHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost || strings.TrimSpace(r.Header.Get("X-Device-Token")) == "" {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "device token and POST required", http.StatusUnauthorized)
		return
	}
	const (
		publicPrefix   = "/public/device-sync-hub/"
		originPrefix   = "/device-sync-hub/"
		strippedPrefix = "/"
	)
	suffix := ""
	for _, prefix := range []string{publicPrefix, originPrefix, strippedPrefix} {
		candidate := strings.TrimPrefix(r.URL.Path, prefix)
		if candidate != r.URL.Path && allowedTransferRPC(candidate) {
			suffix = candidate
			break
		}
	}
	if suffix == "" || r.URL.RawQuery != "" {
		http.NotFound(w, r)
		return
	}
	baseURL := h.baseURL
	if baseURL == "" && h.resolver != nil {
		baseURL, _ = h.resolver.ResolveScenarioURLDefault(r.Context(), "device-sync-hub")
	}
	if baseURL == "" {
		http.Error(w, "device-sync-hub proxy is not configured", http.StatusServiceUnavailable)
		return
	}
	request, err := http.NewRequestWithContext(r.Context(), http.MethodPost, strings.TrimRight(baseURL, "/")+"/"+suffix, io.LimitReader(r.Body, maxRPCBytes+1))
	if err != nil {
		http.Error(w, "unable to create hub request", http.StatusBadGateway)
		return
	}
	request.Header.Set("X-Device-Token", r.Header.Get("X-Device-Token"))
	request.Header.Set("Content-Type", r.Header.Get("Content-Type"))
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
	w.Header().Set("Content-Type", response.Header.Get("Content-Type"))
	w.WriteHeader(response.StatusCode)
	_, _ = io.Copy(w, io.LimitReader(response.Body, maxRPCBytes))
}

func allowedTransferRPC(path string) bool {
	const base = "vrooli.device_sync_hub.v1.transfer.TransferService/"
	return path == base+"ListItems" || path == base+"GetItem" || path == base+"DeleteItem"
}
