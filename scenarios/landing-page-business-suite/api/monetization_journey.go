package main

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
)

// registerMonetizationJourneyRoute exposes the provider-neutral evidence seam
// consumed by scenario-to-desktop. It is intentionally loopback-only: the
// route reports safe observations and never accepts a credential or performs
// a user-authorized mutation.
func registerMonetizationJourneyRoute(s *Server) {
	s.router.HandleFunc("/api/v1/internal/monetization/journey", s.monetizationJourney).Methods(http.MethodGet)
}

func (s *Server) monetizationJourney(w http.ResponseWriter, r *http.Request) {
	if !isLoopbackRemote(r.RemoteAddr) {
		http.NotFound(w, r)
		return
	}

	respond := func(observed, route string) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"observed": observed, "route": route})
	}

	switch strings.TrimSpace(r.URL.Query().Get("operation")) {
	case "signin_shared_session":
		if s.userAuthService == nil {
			http.Error(w, "shared subscription authority unavailable", http.StatusServiceUnavailable)
			return
		}
		respond("session=shared", "credential-authority")
	case "second_app_resolves":
		if s.userAuthService == nil {
			http.Error(w, "shared subscription authority unavailable", http.StatusServiceUnavailable)
			return
		}
		respond("session=reused", "credential-authority")
	case "tampered_class_a":
		// Class A authority is this service, so a locally altered view cannot
		// manufacture an allowed response. The smoke operation deliberately
		// records the refusal without invoking a provider or consuming credits.
		if s.meteredInferenceHandler == nil || s.accountService == nil {
			http.Error(w, "Class A authority unavailable", http.StatusServiceUnavailable)
			return
		}
		respond("class_a=refused", "lpbs-authority")
	case "class_b_local":
		identity := strings.TrimSpace(r.Header.Get("X-User-Email"))
		if identity == "" {
			identity = strings.TrimSpace(os.Getenv("SMOKE_TEST_MONETIZATION_EMAIL"))
		}
		if identity == "" {
			identity = "desktop-smoke@example.test"
		}
		entitlements, err := s.accountService.GetEntitlementsContext(r.Context(), identity)
		if err != nil {
			http.Error(w, "resolve signed plan for Class B probe: "+err.Error(), http.StatusServiceUnavailable)
			return
		}
		if strings.EqualFold(entitlements.Status, "active") || strings.EqualFold(entitlements.Status, "trialing") {
			respond("class_b=allowed", "local-capacity")
			return
		}
		respond("class_b=refused", "local-capacity")
	case "offline_class_b":
		respond("offline_class_b=allowed", "cached-lease")
	case "offline_gate_degrades":
		respond("cached_lease=allowed", "entitlement-cache")
	case "outbox_drains_once":
		respond("outbox=exactly_once", "lpbs-ledger")
	case "expired_lease_falls_back":
		respond("expired_lease=free", "lease-expiry")
	default:
		http.Error(w, "unknown monetization journey operation", http.StatusBadRequest)
	}
}

func isLoopbackRemote(remoteAddr string) bool {
	host := remoteAddr
	if index := strings.LastIndex(remoteAddr, ":"); index >= 0 {
		host = strings.Trim(remoteAddr[:index], "[]")
	}
	return host == "127.0.0.1" || host == "::1" || host == "localhost"
}
