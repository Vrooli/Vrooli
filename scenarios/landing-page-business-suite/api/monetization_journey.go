package main

import (
	"context"
	"net/http"
	"os"
	"strings"

	monetization "github.com/vrooli/vrooli/packages/monetization-go"
)

type monetizationJourneyIdentityKey struct{}

// registerMonetizationJourneyRoute exposes the shared, provider-neutral
// journey contract. Only operations backed by a live dependency are advertised;
// missing dependencies remain unsupported instead of returning success-shaped
// observations.
func registerMonetizationJourneyRoute(s *Server) {
	operations := map[monetization.JourneyOperation]func(context.Context) (string, error){}
	if s.accountService != nil {
		operations[monetization.JourneyClassBLocal] = func(ctx context.Context) (string, error) {
			identity, _ := ctx.Value(monetizationJourneyIdentityKey{}).(string)
			identity = strings.TrimSpace(identity)
			if identity == "" {
				identity = strings.TrimSpace(os.Getenv("SMOKE_TEST_MONETIZATION_EMAIL"))
			}
			if identity == "" {
				identity = "desktop-smoke@example.test"
			}
			entitlements, err := s.accountService.GetEntitlementsContext(ctx, identity)
			if err != nil {
				return "", err
			}
			if strings.EqualFold(entitlements.Status, "active") || strings.EqualFold(entitlements.Status, "trialing") {
				return "class_b=allowed", nil
			}
			return "class_b=refused", nil
		}
	}
	journey := monetization.JourneyModule{Deps: monetization.JourneyDeps{
		AppKey: "landing-page-business-suite", BundleKey: "business_suite", Operations: operations,
	}}
	handler := journey.Handler()
	s.router.Handle("/api/v1/internal/monetization/journey", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), monetizationJourneyIdentityKey{}, r.Header.Get("X-User-Email"))
		handler.ServeHTTP(w, r.WithContext(ctx))
	})).Methods("GET")
}
