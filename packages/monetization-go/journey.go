package monetization

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// JourneyOperation is the provider-neutral operation vocabulary shared by
// scenario-to-desktop and LPBS. New operations must be added deliberately so
// evidence remains comparable across consumers.
type JourneyOperation string

const (
	JourneySignInSharedSession JourneyOperation = "signin_shared_session"
	JourneySecondAppResolves   JourneyOperation = "second_app_resolves"
	JourneyTamperedClassA      JourneyOperation = "tampered_class_a"
	JourneyClassBLocal         JourneyOperation = "class_b_local"
	JourneyOfflineClassB       JourneyOperation = "offline_class_b"
	JourneyOfflineDegrades     JourneyOperation = "offline_gate_degrades"
	JourneyOutboxDrainsOnce    JourneyOperation = "outbox_drains_once"
	JourneyExpiredLease        JourneyOperation = "expired_lease_falls_back"
	JourneyProviderObservation JourneyOperation = "provider_observation"
	JourneyCommunication       JourneyOperation = "communication_operation"
)

type JourneyObservation struct {
	Operation JourneyOperation `json:"operation"`
	Observed  string           `json:"observed"`
	Route     string           `json:"route"`
	AppKey    string           `json:"app_key,omitempty"`
	BundleKey string           `json:"bundle_key,omitempty"`
}

type JourneyProbe struct {
	BaseURL        string
	HTTPClient     *http.Client
	ExpectedAppKey string
}

func (p JourneyProbe) Run(ctx context.Context, operation JourneyOperation) (JourneyObservation, error) {
	base := strings.TrimRight(strings.TrimSpace(p.BaseURL), "/")
	if base == "" {
		return JourneyObservation{}, fmt.Errorf("journey probe base URL is required")
	}
	if strings.TrimSpace(string(operation)) == "" {
		return JourneyObservation{}, fmt.Errorf("journey operation is required")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/api/v1/internal/monetization/journey?operation="+url.QueryEscape(string(operation)), nil)
	if err != nil {
		return JourneyObservation{}, fmt.Errorf("create journey probe request: %w", err)
	}
	client := p.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	response, err := client.Do(request)
	if err != nil {
		return JourneyObservation{}, fmt.Errorf("run journey probe: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return JourneyObservation{}, fmt.Errorf("journey probe returned status %d", response.StatusCode)
	}
	var result struct {
		Observed  string `json:"observed"`
		Route     string `json:"route"`
		AppKey    string `json:"app_key"`
		BundleKey string `json:"bundle_key"`
	}
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return JourneyObservation{}, fmt.Errorf("decode journey probe: %w", err)
	}
	if strings.TrimSpace(result.Observed) == "" || strings.TrimSpace(result.Route) == "" {
		return JourneyObservation{}, fmt.Errorf("journey probe returned incomplete observation")
	}
	if expected := strings.TrimSpace(p.ExpectedAppKey); expected != "" && strings.TrimSpace(result.AppKey) == "" {
		return JourneyObservation{}, fmt.Errorf("journey probe app_key is missing")
	}
	if expected := strings.TrimSpace(p.ExpectedAppKey); expected != "" && result.AppKey != expected {
		return JourneyObservation{}, fmt.Errorf("journey probe app_key mismatch")
	}
	return JourneyObservation{Operation: operation, Observed: result.Observed, Route: result.Route, AppKey: result.AppKey, BundleKey: result.BundleKey}, nil
}

// JourneyDeps contains the app-owned checks behind the provider-neutral
// journey vocabulary. Missing callbacks are reported as unsupported rather
// than replaced with success-shaped fixtures.
type JourneyDeps struct {
	AppKey, BundleKey string
	Now               func() time.Time
	Operations        map[JourneyOperation]func(context.Context) (string, error)
}

type JourneyModule struct{ Deps JourneyDeps }

func (m JourneyModule) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !journeyLoopback(r.RemoteAddr) {
			http.NotFound(w, r)
			return
		}
		op := JourneyOperation(strings.TrimSpace(r.URL.Query().Get("operation")))
		if op == "" {
			http.Error(w, "operation is required", http.StatusBadRequest)
			return
		}
		if op == "capabilities" {
			operations := make([]string, 0, len(m.Deps.Operations))
			for _, candidate := range []JourneyOperation{JourneySignInSharedSession, JourneySecondAppResolves, JourneyTamperedClassA, JourneyClassBLocal, JourneyOfflineClassB, JourneyOfflineDegrades, JourneyOutboxDrainsOnce, JourneyExpiredLease, JourneyProviderObservation, JourneyCommunication} {
				if m.Deps.Operations[candidate] != nil {
					operations = append(operations, string(candidate))
				}
			}
			writeJourneyJSON(w, http.StatusOK, map[string]any{"operation": string(op), "observed": "capabilities", "route": "/api/v1/internal/monetization/journey", "app_key": m.Deps.AppKey, "bundle_key": m.Deps.BundleKey, "operations": operations})
			return
		}
		if !journeyKnownOperation(op) {
			http.Error(w, "unknown operation", http.StatusBadRequest)
			return
		}
		operation, ok := m.Deps.Operations[op]
		if !ok {
			writeJourneyJSON(w, http.StatusOK, map[string]string{"operation": string(op), "observed": "unsupported:dependency unavailable", "route": "/api/v1/internal/monetization/journey", "app_key": m.Deps.AppKey, "bundle_key": m.Deps.BundleKey})
			return
		}
		observed, err := operation(r.Context())
		if err != nil {
			observed = "unsupported:" + strings.TrimSpace(err.Error())
		}
		writeJourneyJSON(w, http.StatusOK, map[string]string{"operation": string(op), "observed": observed, "route": "/api/v1/internal/monetization/journey", "app_key": m.Deps.AppKey, "bundle_key": m.Deps.BundleKey})
	})
}

func journeyKnownOperation(operation JourneyOperation) bool {
	switch operation {
	case JourneySignInSharedSession, JourneySecondAppResolves, JourneyTamperedClassA, JourneyClassBLocal, JourneyOfflineClassB, JourneyOfflineDegrades, JourneyOutboxDrainsOnce, JourneyExpiredLease, JourneyProviderObservation, JourneyCommunication:
		return true
	}
	return false
}

func journeyLoopback(remote string) bool {
	host, _, err := net.SplitHostPort(remote)
	if err != nil {
		host = remote
	}
	ip := net.ParseIP(strings.Trim(host, "[]"))
	return ip != nil && ip.IsLoopback()
}

func writeJourneyJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
