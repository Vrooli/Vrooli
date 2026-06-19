// Package config is the domain-scoped home for the tunnel's management
// configuration and the ingress reconciler — the adapter that makes
// exposure programmatic. Remote mode pushes ingress through the
// Cloudflare API v4 (hot reload); local mode generates a cloudflared
// config.yml from the routes manifest.
//
// Layering mirrors the canonical Vrooli pattern (see internal/routes for
// the worked sibling reference):
//
//	HTTP → handler → Service (reconciles, applies) → Repository (persists)
//	                     ↑              ↑                   ↑
//	                     FakeService    FakeIngressClient   FakeRepository / real sqlite
//
// types.go owns the domain entity, the seam interfaces the service
// depends on, and the typed sentinels handlers translate at the
// transport edge. The proto wire types live one floor up
// (packages/proto/...) and never import this package; the handler is the
// only translation point (api-steer §7).
package config

import (
	"context"
	"fmt"

	internalroutes "tunnel-manager/internal/routes"
)

// Mode is the tunnel management mode. Remote pushes ingress through the
// Cloudflare API; local writes a cloudflared config.yml on disk.
type Mode string

const (
	// ModeUnspecified is the zero value (no mode persisted yet).
	ModeUnspecified Mode = ""
	// ModeRemote manages ingress through the Cloudflare API v4 (hot reload).
	ModeRemote Mode = "remote"
	// ModeLocal manages ingress through a local cloudflared config.yml.
	ModeLocal Mode = "local"
)

// DefaultMode is the mode a fresh config defaults to when none is
// persisted. Remote is the live tunnel's operating mode.
const DefaultMode = ModeRemote

// DefaultPromEndpoint is the cloudflared Prometheus metrics endpoint used
// when none is configured.
const DefaultPromEndpoint = "127.0.0.1:20241"

// TunnelConfig is the persisted, single-row configuration for the tunnel.
// Distinct from the proto wire type at packages/proto/gen/go/.../v1/config
// — handlers translate at the boundary so the domain never imports proto.
type TunnelConfig struct {
	Mode      Mode
	TunnelID  string
	AccountID string
	// CredRef is a reference (not the secret itself) to the Cloudflare API
	// credential.
	CredRef string
	// PromEndpoint is the Prometheus metrics endpoint exposed by cloudflared.
	PromEndpoint string
}

// SyncResult is the outcome of a reconcile: which hostnames changed and
// whether anything changed at all.
type SyncResult struct {
	Mode      Mode
	Added     []string
	Removed   []string
	NoChanges bool
}

// IngressRule is one desired ingress mapping: a public hostname routed to
// a local service. The catch-all rule (404) has an empty Hostname.
type IngressRule struct {
	Hostname string
	Service  string
}

// IngressClient is the seam over the Cloudflare API v4 ingress surface.
// Declared at the consumer per seam-discovery: production wires cfClient
// (httpc.Doer + creds); service tests fake it. Nil when CF creds are
// absent, in which case remote operations return ErrRemoteUnavailable.
type IngressClient interface {
	// ReadIngress returns the live ingress rules (including the catch-all).
	ReadIngress(ctx context.Context) ([]IngressRule, error)
	// PushIngress replaces the live ingress with the supplied rules. The
	// implementation ensures a trailing catch-all 404 rule.
	PushIngress(ctx context.Context, rules []IngressRule) error
}

// RoutesReader is the narrow read surface the config service needs from
// the routes domain. Satisfied by routes.Service (which exposes List).
type RoutesReader interface {
	List(ctx context.Context, tier internalroutes.Tier) ([]internalroutes.Route, error)
}

// ErrRemoteUnavailable is the typed sentinel returned when a remote-mode
// operation is requested but Cloudflare credentials are absent (no
// IngressClient wired). Handlers translate via errors.As into a
// connect.CodeFailedPrecondition.
type ErrRemoteUnavailable struct {
	Reason string
}

func (e ErrRemoteUnavailable) Error() string {
	if e.Reason == "" {
		return "remote mode unavailable: CF_API_TOKEN, CF_ACCOUNT_ID, and CF_TUNNEL_ID are required"
	}
	return fmt.Sprintf("remote mode unavailable: %s", e.Reason)
}

// ErrInvalidConfig is the typed sentinel returned when an input fails
// validation (e.g. an unknown SwitchMode target). Handlers translate via
// errors.As into a connect.CodeInvalidArgument.
type ErrInvalidConfig struct {
	Field  string
	Reason string
}

func (e ErrInvalidConfig) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Reason)
}
