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
// persisted. Local is safe on first boot because it does not require
// Cloudflare API credentials.
const DefaultMode = ModeLocal

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

// ConfigState combines the persisted config with process-level readiness.
// Readiness is derived from non-secret credential presence and local path
// wiring; it is safe for UI and CLI consumers.
type ConfigState struct {
	Config    TunnelConfig
	Readiness ConfigReadiness
}

// ConfigReadiness reports whether each operating mode is usable right now.
// It never carries the Cloudflare API token value.
type ConfigReadiness struct {
	DesiredMode      Mode
	RemoteAvailable  bool
	MissingFields    []string
	CredentialSource string
	CredentialRef    string
	CredentialStatus CredentialStatus
	LocalConfigPath  string
	SyncReady        bool
	ModeReason       string
}

// CredentialStatus reports Cloudflare credential presence without carrying
// secret values. It is safe for readiness, CLI, and UI surfaces.
type CredentialStatus struct {
	Fields        []CredentialFieldStatus
	MissingFields []string
	Source        string
	Ref           string
	Ready         bool
}

// CredentialFieldStatus is field-level source metadata for one required
// Cloudflare credential input.
type CredentialFieldStatus struct {
	Name     string
	Present  bool
	Source   string
	Ref      string
	Writable bool
}

// CredentialUpdate is the write-only input accepted by CredentialStore.Save.
// Empty fields are ignored so callers can update one field without knowing the
// existing values.
type CredentialUpdate struct {
	AccountID string
	TunnelID  string
	APIToken  string
}

// CredentialStore is the seam over Cloudflare credential persistence and
// resolution. Resolve may return the API token in process memory; Status never
// returns secret values.
type CredentialStore interface {
	Status(ctx context.Context) (CredentialStatus, error)
	Resolve(ctx context.Context) (CFConfig, error)
	Save(ctx context.Context, values CredentialUpdate) (CredentialStatus, error)
	Delete(ctx context.Context, keys []string) (CredentialStatus, error)
}

// SyncResult is the outcome of a reconcile: which hostnames changed and
// whether anything changed at all.
type SyncResult struct {
	Mode          Mode
	Added         []string
	Removed       []string
	NoChanges     bool
	SetupRequired bool
	MissingFields []string
	Message       string
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
		return "remote mode unavailable: CLOUDFLARE_ACCOUNT_ID, CLOUDFLARE_TUNNEL_ID, and CLOUDFLARE_API_TOKEN are required"
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
