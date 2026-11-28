package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/vrooli/api-core/authn"
	"web-console/internal/sessionstore"
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
	resolve      notificationsURLResolver
	client       *http.Client
	ownedSession func(context.Context, string) bool
}

const webConsoleNotificationSource = "web-console"

func newNotificationsProxy(resolve notificationsURLResolver, stores ...sessionstore.Store) *notificationsProxy {
	p := &notificationsProxy{resolve: resolve, client: &http.Client{Timeout: 15 * time.Second}}
	if len(stores) > 0 && stores[0] != nil {
		store := stores[0]
		p.ownedSession = func(ctx context.Context, id string) bool {
			principal, err := authn.RequireIndependentHuman(ctx)
			if err != nil || !principal.IsHuman() || principal.ExpiresAt.IsZero() || !time.Now().Before(principal.ExpiresAt) {
				return false
			}
			meta, err := store.Get(ctx, id)
			return err == nil && strings.TrimSpace(meta.Owner) != "" && meta.Owner == principal.OwnerSubject()
		}
	}
	return p
}

func notificationsUnavailable(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusServiceUnavailable)
	_, _ = io.WriteString(w, `{"error":"notification-hub is unavailable"}`)
}

func (p *notificationsProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Settings in Web Console manage only Web Console's notification source.
	// The browser's source filter is a convenience, never an authorization
	// boundary: pin it here before the request reaches Notification Hub.
	if r.Method == http.MethodGet && (r.URL.Path == "/api/v1/notifications/catalog" || r.URL.Path == "/api/v1/notifications/preferences") {
		query := r.URL.Query()
		query.Set("source_scenario", webConsoleNotificationSource)
		r.URL.RawQuery = query.Encode()
	}
	if r.Method == http.MethodPut && r.URL.Path == "/api/v1/notifications/preferences" {
		rawBody, err := io.ReadAll(io.LimitReader(r.Body, 64*1024+1))
		if err != nil {
			http.Error(w, "invalid notification preference", http.StatusBadRequest)
			return
		}
		if len(rawBody) > 64*1024 {
			http.Error(w, "notification preference is too large", http.StatusRequestEntityTooLarge)
			return
		}
		var body map[string]json.RawMessage
		if err := json.Unmarshal(rawBody, &body); err != nil || body == nil {
			http.Error(w, "invalid notification preference", http.StatusBadRequest)
			return
		}
		var source string
		if raw := body["source_scenario"]; len(raw) != 0 {
			if err := json.Unmarshal(raw, &source); err != nil {
				http.Error(w, "invalid notification source", http.StatusBadRequest)
				return
			}
		}
		if source != "" && source != webConsoleNotificationSource {
			http.Error(w, "notification source is outside Web Console scope", http.StatusForbidden)
			return
		}
		body["source_scenario"], _ = json.Marshal(webConsoleNotificationSource)
		if raw := body["scope_key"]; len(raw) != 0 {
			var scope string
			if err := json.Unmarshal(raw, &scope); err != nil {
				http.Error(w, "invalid notification scope", http.StatusBadRequest)
				return
			}
			if scope != "" && (strings.TrimSpace(scope) == "" || p == nil || p.ownedSession == nil || !p.ownedSession(r.Context(), scope)) {
				http.Error(w, "notification scope is not owned by the signed-in operator", http.StatusForbidden)
				return
			}
		}
		encoded, err := json.Marshal(body)
		if err != nil {
			http.Error(w, "invalid notification preference", http.StatusBadRequest)
			return
		}
		r.Body = io.NopCloser(strings.NewReader(string(encoded)))
		r.ContentLength = int64(len(encoded))
	}
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
	// Forward the Cloudflare Access assertion so the hub's human-proof path runs
	// on answers submitted through the inbox: the hub verifies it and attributes
	// the answer to the ACTUAL signed-in human (answered_by), not the single
	// configured operator subject. Cloudflare sets this header on every request
	// that passed Access; a request without Access simply omits it and the hub
	// falls back to the bearer/cookie owner identity.
	if access := r.Header.Get("Cf-Access-Jwt-Assertion"); access != "" {
		request.Header.Set("Cf-Access-Jwt-Assertion", access)
	}
	response, err := p.client.Do(request)
	if err != nil {
		notificationsUnavailable(w)
		return
	}
	defer response.Body.Close()
	if r.Method == http.MethodGet && (r.URL.Path == "/api/v1/notifications/catalog" || r.URL.Path == "/api/v1/notifications/preferences") {
		if response.StatusCode >= 200 && response.StatusCode < 300 {
			body, readErr := io.ReadAll(io.LimitReader(response.Body, 1<<20+1))
			if readErr != nil || len(body) > 1<<20 {
				http.Error(w, "invalid notification preference response", http.StatusBadGateway)
				return
			}
			filtered, filterErr := filterNotificationScopes(body, p.ownedSession, r.Context())
			if filterErr != nil {
				http.Error(w, "invalid notification preference response", http.StatusBadGateway)
				return
			}
			for key, values := range response.Header {
				for _, value := range values {
					w.Header().Add(key, value)
				}
			}
			w.Header().Del("Content-Length")
			w.WriteHeader(response.StatusCode)
			_, _ = w.Write(filtered)
			return
		}
	}
	for key, values := range response.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(response.StatusCode)
	_, _ = io.CopyN(w, response.Body, 1<<20+1)
}

// filterNotificationScopes preserves global preferences and only exposes a
// scoped preference when the authenticated owner owns the referenced session.
func filterNotificationScopes(raw []byte, owns func(context.Context, string) bool, ctx context.Context) ([]byte, error) {
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}
	filter := func(rawPrefs json.RawMessage) (json.RawMessage, error) {
		var prefs []map[string]json.RawMessage
		if err := json.Unmarshal(rawPrefs, &prefs); err != nil {
			return nil, err
		}
		out := make([]map[string]json.RawMessage, 0, len(prefs))
		for _, pref := range prefs {
			var scope string
			if len(pref["scope_key"]) == 0 || json.Unmarshal(pref["scope_key"], &scope) != nil {
				continue
			}
			if scope == "" || (owns != nil && owns(ctx, scope)) {
				out = append(out, pref)
			}
		}
		return json.Marshal(out)
	}
	if catalog, ok := payload["preferences"]; ok {
		filtered, err := filter(catalog)
		if err != nil {
			return nil, err
		}
		payload["preferences"] = filtered
	}
	return json.Marshal(payload)
}
