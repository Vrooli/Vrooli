// Package entitlement hosts the BAS EntitlementService Connect-RPC handler.
//
// EntitlementService is the read/write surface for the user's subscription
// state, identity, credit usage, dev overrides, and the entitlement API
// source. The UI Settings panel and the BAS CLI both consume it.
package entitlement

import (
	"context"
	"net"
	"net/http"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/vrooli/api-core/connectx"

	"github.com/vrooli/browser-automation-studio/services/credits"
	entsvc "github.com/vrooli/browser-automation-studio/services/entitlement"
	entitlementconnect "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/entitlement/entitlementconnect"
)

// SettingsStore is the narrow seam used for persistent identity / override /
// api-source settings. The concrete implementation is the SQLite-backed
// repo wired into the API server; tests substitute an in-memory map.
type SettingsStore interface {
	GetSetting(ctx context.Context, key string) (string, error)
	SetSetting(ctx context.Context, key, value string) error
}

// Provider is the narrow seam onto the package-level entitlement.Service.
// Exposing it as an interface lets tests swap a fake without standing up
// the full caching service.
type Provider interface {
	GetEntitlement(ctx context.Context, userIdentity string) (*entsvc.Entitlement, error)
	BuildOverrideEntitlement(userIdentity string, tier entsvc.Tier) *entsvc.Entitlement
	InvalidateCache(userIdentity string)

	SetApiSource(source string, localPort int)

	GetAICreditsLimit(tier entsvc.Tier) int

	TierRequiresWatermark(tier entsvc.Tier) bool
	TierCanUseAI(tier entsvc.Tier) bool
	TierCanUseRecording(tier entsvc.Tier) bool

	RequiresWatermark(ctx context.Context, userIdentity string) bool
	CanUseAI(ctx context.Context, userIdentity string) bool
	CanUseRecording(ctx context.Context, userIdentity string) bool

	MinTierForAI() entsvc.Tier
	MinTierForRecording() entsvc.Tier
	MinTierWithoutWatermark() entsvc.Tier
}

// Deps wires the entitlement handler.
// Logger and Provider are required.
// Credits is optional — when nil, usage endpoints return FailedPrecondition.
// Settings is optional — when nil, identity/override/api-source endpoints
// that touch persistent state return FailedPrecondition.
type Deps struct {
	Provider Provider
	Credits  credits.CreditService
	Settings SettingsStore
	Logger   *logrus.Logger
}

// Module builds the EntitlementService Connect handler and returns it
// wrapped in a connectx.ServiceMount ready for connectx.RegisterChi.
func Module(d Deps) connectx.ServiceMount {
	if d.Logger == nil {
		panic("entitlement.Module requires Deps.Logger")
	}
	if d.Provider == nil {
		panic("entitlement.Module requires Deps.Provider")
	}
	path, handler := entitlementconnect.NewEntitlementServiceHandler(&service{deps: d})
	return connectx.ServiceMount{Path: path, Handler: controlPlaneGuard(handler)}
}

// controlPlaneGuard keeps development-only entitlement controls local to the
// machine. The BAS API normally listens on loopback, but a reverse proxy may
// expose that listener for a VPS deployment; without this guard a remote
// caller could select the disabled source or grant a local tier override.
func controlPlaneGuard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isProtectedControlPath(r.URL.Path) && !isLoopbackPeer(r.RemoteAddr) {
			http.Error(w, "entitlement control requires a local request", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func isProtectedControlPath(path string) bool {
	for _, procedure := range []string{"/SetOverride", "/ClearOverride", "/SetApiSource", "/ClearApiSource"} {
		if strings.HasSuffix(path, procedure) {
			return true
		}
	}
	return false
}

func isLoopbackPeer(remoteAddr string) bool {
	peer := strings.TrimSpace(remoteAddr)
	if host, _, err := net.SplitHostPort(peer); err == nil {
		peer = host
	} else {
		peer = strings.Trim(peer, "[]")
	}
	ip := net.ParseIP(peer)
	return ip != nil && ip.IsLoopback()
}
