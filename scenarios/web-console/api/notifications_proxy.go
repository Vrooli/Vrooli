package main

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type notificationsURLResolver func(context.Context) (string, error)

// notificationsProxy keeps the notification-hub preference surface same-origin
// for the browser. It resolves the hub on every request via scenario
// discovery, forwards the bounded REST body, and carries the operator's owner
// identity by copying the incoming Authorization header onto the upstream
// request. The hub validates the SAME owner token Web Console already sends.
//
// This mirrors desktopProxy, with two deliberate differences: notification-hub
// authenticates with the browser's own Bearer token (not a server-side node
// credential), so the Authorization header is forwarded; and an unreachable
// hub returns a clean JSON 503 so the Notifications settings tab can render an
// "unavailable" state instead of crashing.
type notificationsProxy struct {
	resolve notificationsURLResolver
	client  *http.Client
}

func newNotificationsProxy(resolve notificationsURLResolver) *notificationsProxy {
	return &notificationsProxy{resolve: resolve, client: &http.Client{Timeout: 15 * time.Second}}
}

func notificationsUnavailable(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusServiceUnavailable)
	_, _ = io.WriteString(w, `{"error":"notification-hub is unavailable"}`)
}

func (p *notificationsProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if p == nil || p.resolve == nil {
		notificationsUnavailable(w)
		return
	}
	base, err := p.resolve(r.Context())
	if err != nil {
		notificationsUnavailable(w)
		return
	}
	baseURL, err := url.Parse(strings.TrimRight(base, "/"))
	if err != nil || baseURL.Scheme == "" || baseURL.Host == "" {
		notificationsUnavailable(w)
		return
	}
	baseURL.Path = strings.TrimRight(baseURL.Path, "/") + r.URL.Path
	baseURL.RawQuery = r.URL.RawQuery
	request, err := http.NewRequestWithContext(r.Context(), r.Method, baseURL.String(), io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "notification request is invalid", http.StatusBadRequest)
		return
	}
	request.ContentLength = r.ContentLength
	if ct := r.Header.Get("Content-Type"); ct != "" {
		request.Header.Set("Content-Type", ct)
	}
	if rid := r.Header.Get("X-Request-ID"); rid != "" {
		request.Header.Set("X-Request-ID", rid)
	}
	// Carry the operator's owner identity so notification-hub validates the
	// same token Web Console already received from the browser.
	if auth := r.Header.Get("Authorization"); auth != "" {
		request.Header.Set("Authorization", auth)
	}
	if dev := r.Header.Get("X-Vrooli-Device-Id"); dev != "" {
		request.Header.Set("X-Vrooli-Device-Id", dev)
	}
	response, err := p.client.Do(request)
	if err != nil {
		notificationsUnavailable(w)
		return
	}
	defer response.Body.Close()
	for key, values := range response.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(response.StatusCode)
	_, _ = io.CopyN(w, response.Body, 1<<20+1)
}
