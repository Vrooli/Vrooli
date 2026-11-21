package main

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/nodereach"
)

type desktopURLResolver func(context.Context) (string, error)

// desktopProxy keeps Device Control's Connect surface same-origin for the
// browser. It resolves the scenario on every request, forwards only the
// bounded typed body, and never exposes the upstream URL or node credentials
// to Web Console state.
type desktopProxy struct {
	resolve desktopURLResolver
	client  *http.Client
}

func newDesktopProxy(resolve desktopURLResolver) *desktopProxy {
	return &desktopProxy{resolve: resolve, client: &http.Client{Timeout: 45 * time.Second}}
}

type desktopTargetClientResolver func(context.Context) (*nodereach.Client, error)

// desktopTargetProxy is the browser-facing target route. It keeps Bridge
// credentials server-side and uses the Bridge target proxy to reach the
// selected node's Device Control service. The browser never receives a node
// URL or owner credential.
type desktopTargetProxy struct {
	resolve desktopTargetClientResolver
	client  *http.Client
}

type bridgeControlProxy struct {
	resolve desktopTargetClientResolver
	client  *http.Client
}

func newBridgeControlProxy(resolve desktopTargetClientResolver) *bridgeControlProxy {
	return &bridgeControlProxy{resolve: resolve, client: &http.Client{Timeout: 45 * time.Second}}
}

func (p *bridgeControlProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if p == nil || p.resolve == nil {
		http.Error(w, "Bridge unavailable", http.StatusServiceUnavailable)
		return
	}
	bridge, err := p.resolve(r.Context())
	if err != nil || bridge == nil {
		http.Error(w, "Bridge unavailable", http.StatusServiceUnavailable)
		return
	}
	bridgeURL, err := bridge.ResolveURL(r.Context())
	if err != nil {
		http.Error(w, "Bridge unavailable", http.StatusServiceUnavailable)
		return
	}
	procedure := strings.TrimPrefix(r.URL.Path, "/api/v1/desktop-bridge/")
	if strings.TrimSpace(procedure) == "" || strings.Contains(procedure, "..") {
		http.Error(w, "Bridge procedure is invalid", http.StatusBadRequest)
		return
	}
	targetURL := strings.TrimRight(bridgeURL, "/") + "/" + procedure
	request, err := http.NewRequestWithContext(r.Context(), http.MethodPost, targetURL, io.LimitReader(r.Body, 128*1024))
	if err != nil {
		http.Error(w, "Bridge request is invalid", http.StatusBadRequest)
		return
	}
	request.ContentLength = r.ContentLength
	request.Header.Set("Content-Type", r.Header.Get("Content-Type"))
	request.Header.Set("Accept", r.Header.Get("Accept"))
	request.Header.Set("X-Request-ID", r.Header.Get("X-Request-ID"))
	response, err := bridge.ConnectTransport(r.Context(), bridgeURL).Do(request)
	if err != nil {
		http.Error(w, "Bridge request failed", http.StatusBadGateway)
		return
	}
	defer response.Body.Close()
	for key, values := range response.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(response.StatusCode)
	_, _ = io.CopyN(w, response.Body, 128*1024+1)
}

func newDesktopTargetProxy(resolve desktopTargetClientResolver) *desktopTargetProxy {
	return &desktopTargetProxy{resolve: resolve, client: &http.Client{Timeout: 45 * time.Second}}
}

func (p *desktopTargetProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if p == nil || p.resolve == nil {
		http.Error(w, "Bridge unavailable", http.StatusServiceUnavailable)
		return
	}
	nodeID := mux.Vars(r)["nodeID"]
	procedure := strings.TrimPrefix(r.URL.Path, "/api/v1/desktop/"+url.PathEscape(nodeID)+"/")
	if strings.TrimSpace(nodeID) == "" || strings.TrimSpace(procedure) == "" || strings.Contains(procedure, "..") {
		http.Error(w, "desktop target is invalid", http.StatusBadRequest)
		return
	}
	bridge, err := p.resolve(r.Context())
	if err != nil || bridge == nil {
		http.Error(w, "Bridge unavailable", http.StatusServiceUnavailable)
		return
	}
	bridgeURL, err := bridge.ResolveURL(r.Context())
	if err != nil {
		http.Error(w, "Bridge unavailable", http.StatusServiceUnavailable)
		return
	}
	targetURL := strings.TrimRight(bridgeURL, "/") + "/api/v1/targets/" + url.PathEscape(nodeID) + "/scenarios/device-control/" + strings.TrimPrefix(procedure, "/")
	request, err := http.NewRequestWithContext(r.Context(), http.MethodPost, targetURL, io.LimitReader(r.Body, 128*1024))
	if err != nil {
		http.Error(w, "desktop request is invalid", http.StatusBadRequest)
		return
	}
	request.ContentLength = r.ContentLength
	request.Header.Set("Content-Type", r.Header.Get("Content-Type"))
	request.Header.Set("Accept", r.Header.Get("Accept"))
	request.Header.Set("X-Request-ID", r.Header.Get("X-Request-ID"))
	response, err := bridge.ConnectTransport(r.Context(), bridgeURL).Do(request)
	if err != nil {
		http.Error(w, "desktop target request failed", http.StatusBadGateway)
		return
	}
	defer response.Body.Close()
	for key, values := range response.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(response.StatusCode)
	_, _ = io.CopyN(w, response.Body, 128*1024+1)
}

func (p *desktopProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if p == nil || p.resolve == nil {
		http.Error(w, "device-control unavailable", http.StatusServiceUnavailable)
		return
	}
	base, err := p.resolve(r.Context())
	if err != nil {
		http.Error(w, "device-control unavailable", http.StatusServiceUnavailable)
		return
	}
	baseURL, err := url.Parse(strings.TrimRight(base, "/"))
	if err != nil || baseURL.Scheme == "" || baseURL.Host == "" {
		http.Error(w, "device-control endpoint is invalid", http.StatusBadGateway)
		return
	}
	baseURL.Path = strings.TrimRight(baseURL.Path, "/") + r.URL.Path
	baseURL.RawQuery = r.URL.RawQuery
	request, err := http.NewRequestWithContext(r.Context(), r.Method, baseURL.String(), io.LimitReader(r.Body, 128*1024))
	if err != nil {
		http.Error(w, "device-control request is invalid", http.StatusBadRequest)
		return
	}
	request.ContentLength = r.ContentLength
	request.Header.Set("Content-Type", r.Header.Get("Content-Type"))
	request.Header.Set("X-Request-ID", r.Header.Get("X-Request-ID"))
	response, err := p.client.Do(request)
	if err != nil {
		http.Error(w, "device-control request failed", http.StatusBadGateway)
		return
	}
	defer response.Body.Close()
	for key, values := range response.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(response.StatusCode)
	_, _ = io.CopyN(w, response.Body, 128*1024+1)
}
