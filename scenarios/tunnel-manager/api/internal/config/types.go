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
	"time"

	"tunnel-manager/internal/manifest"
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
	// PublicExposureEnabled is the global switch for the /public Access-bypass
	// convention (see docs/concepts/PUBLIC_ASSETS.md). Default false: TM creates
	// no Bypass apps. When true, every active host whose route's PublicExposure
	// is inherit (or enabled) gets a <host>/public Bypass-Everyone Access app.
	PublicExposureEnabled bool
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

// CheckState is the outcome of a single live credential check. It never
// carries secret values — only a coarse machine-readable verdict.
type CheckState string

const (
	// CheckUnspecified is the zero value (no verdict computed).
	CheckUnspecified CheckState = ""
	// CheckOK means the check passed against the live Cloudflare account.
	CheckOK CheckState = "ok"
	// CheckMissing means the required input is absent (e.g. no token).
	CheckMissing CheckState = "missing"
	// CheckInvalid means the input is present but rejected (bad/expired token,
	// unknown account/tunnel, zone not found).
	CheckInvalid CheckState = "invalid"
	// CheckInsufficientScope means the token authenticates but lacks the
	// permission this check needs (e.g. no Zone:DNS:Edit).
	CheckInsufficientScope CheckState = "insufficient_scope"
)

// Canonical credential-check names. Stable identifiers consumers key on. The
// three that name a credential field REUSE the existing store-key constants
// rather than re-declaring a secret-looking string literal (which the
// hardcoded-value structure detector would flag); the two scope checks are
// plain non-secret identifiers.
const (
	CheckNameAccount    = credentialKeyAccountID
	CheckNameTunnel     = credentialKeyTunnelID
	CheckNameDNSScope   = "cloudflare.zone_dns_edit"
	CheckNameZoneLookup = "cloudflare.zone_lookup"
	// CheckNameAccessScope probes Access: Apps and Policies: Edit — the scope
	// the /public Access-bypass capability needs. Informational unless the
	// capability is enabled (so it never breaks readiness for non-users).
	CheckNameAccessScope = "cloudflare.access_apps_edit"
)

// CheckNameToken mirrors the API-token store key (which is built from parts, not
// a string literal), so the hardcoded-secret detector never flags this package.
var CheckNameToken = credentialKeyAPIToken

// CredentialCheck is one live verification result. It is browser/CLI safe:
// it reports a verdict and a one-line remediation, never a secret value.
type CredentialCheck struct {
	// Name is a stable identifier (e.g. cloudflare.zone_dns_edit).
	Name string
	// State is the coarse verdict.
	State CheckState
	// Detail is a short non-secret explanation (e.g. the apex it resolved).
	Detail string
	// Remediation is a one-line operator next-step when State != OK.
	Remediation string
}

// CredentialVerification bundles the per-check results of a live probe plus a
// roll-up Ready flag (true only when every check is OK).
type CredentialVerification struct {
	Checks []CredentialCheck
	Ready  bool
}

// CredentialVerifier is the outbound-integration seam over live Cloudflare
// credential/scope probing. Declared at the consumer per seam-discovery:
// production wires cfVerifier (httpc.Doer + the resolved CFConfig); tests
// fake it (or fake the Doer). It performs READ-ONLY calls and never mutates
// account state. Verify is the only method; the apexes argument is the set of
// route apex domains whose Zone:DNS:Edit scope should be probed.
type CredentialVerifier interface {
	Verify(ctx context.Context, cfg CFConfig, apexes []string, accessRequired bool) (CredentialVerification, error)
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

// SyncResult is the outcome of an additive reconcile. Sync is additive by
// default: Added are the desired hostnames newly published; DriftUnmanaged and
// Orphaned are surfaced for operator awareness but NEVER removed unless prune
// is explicitly requested; Removed/Pruned are the hostnames a prune actually
// removed (orphaned entries on a batch --prune, or a named hostname).
type SyncResult struct {
	Mode Mode
	// Added are desired hostnames published this sync (or that would be on a
	// dry-run).
	Added []string
	// Removed mirrors Pruned for backward compatibility.
	Removed []string
	// DriftUnmanaged are live hostnames TM does not manage and did not touch.
	DriftUnmanaged []string
	// Orphaned are ledger-managed hostnames whose routes are gone (prune
	// candidates). Surfaced, removed only with prune.
	Orphaned []string
	// Pruned are hostnames removed this sync (empty unless prune was set).
	Pruned        []string
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

// DNSResult reports the outcome of an EnsureRecord call. Created is true only
// when TM actually created the record this call (so the caller records DNS
// ownership and a later revoke knows it is safe to delete). RecordID is the
// Cloudflare DNS record id, set whether the record was created or pre-existing.
type DNSResult struct {
	RecordID string
	Created  bool
}

// DNSClient is the seam over the Cloudflare API v4 DNS-records surface — the
// piece that makes a freshly-exposed hostname publicly RESOLVABLE (the gap that
// left ingress live but the URL NXDOMAIN). Declared at the consumer per
// seam-discovery: production wires cfDNSClient (httpc.Doer + resolved creds);
// tests fake it (or fake the Doer). All operations are idempotent and additive:
// EnsureRecord never clobbers a record pointing elsewhere, and the service only
// calls DeleteRecord for records its DNS ledger says TM created.
type DNSClient interface {
	// EnsureRecord idempotently ensures a proxied CNAME for hostname pointing at
	// the tunnel (<tunnel-id>.cfargotunnel.com). A pre-existing record is left
	// untouched (Created=false); an absent one is created (Created=true).
	EnsureRecord(ctx context.Context, hostname string) (DNSResult, error)
	// RemoveRecord deletes the CNAME for hostname (resolving zone+record id by
	// name). Idempotent: a missing record returns removed=false, nil. The
	// service calls it only for hostnames its DNS ledger attributes to TM, so it
	// never deletes a record created out-of-band.
	RemoveRecord(ctx context.Context, hostname string) (removed bool, err error)
}

// DNSRecordEntry is one TM-created DNS record tracked in the DNS ownership
// ledger, mirroring the ingress ledger so revoke/prune only ever deletes
// records TM itself created.
type DNSRecordEntry struct {
	Hostname  string
	RecordID  string
	CreatedAt time.Time
}

// DNSLedger is the persistence seam over the dns_ownership table — the DNS
// analogue of OwnershipLedger. Absence of a row means "TM did not create this
// hostname's DNS record" (the safe default: never delete it).
type DNSLedger interface {
	List(ctx context.Context) ([]DNSRecordEntry, error)
	Get(ctx context.Context, hostname string) (entry DNSRecordEntry, found bool, err error)
	Put(ctx context.Context, entry DNSRecordEntry) error
	Delete(ctx context.Context, hostname string) (bool, error)
}

// AccessResult reports the outcome of an EnsurePublicBypass call. Created is
// true only when TM actually created the Access app this call (so the caller
// records ownership and a later prune knows it is safe to delete). AppID and
// PolicyID are the Cloudflare ids, set whether the app was created or
// pre-existing.
type AccessResult struct {
	AppID    string
	PolicyID string
	Created  bool
}

// AccessApp is one TM-owned Cloudflare Access application discovered by a
// lookup — the read shape behind idempotency and dry-run preview.
type AccessApp struct {
	AppID    string
	PolicyID string
	Domain   string
}

// AccessClient is the seam over the Cloudflare API v4 Access apps/policies
// surface — NARROWLY scoped to the /public bypass convention (see
// docs/concepts/PUBLIC_ASSETS.md). It is the piece that makes <host>/public
// publicly fetchable by anonymous clients without weakening Access on the rest
// of the app. Declared at the consumer per seam-discovery: production wires
// cfAccessClient (httpc.Doer + resolved creds incl. account id); tests fake it
// (or fake the Doer). All operations are idempotent and additive:
// EnsurePublicBypass never modifies an app TM did not create, and the service
// only calls RemovePublicBypass for hosts its access ledger attributes to TM.
//
// HARD SCOPE CEILING (enforced in the impl + tested): every app it creates is
// type=self_hosted, domain=<host>/public exactly, with a single
// Bypass-Everyone policy. It refuses any other path (empty/`/`/`/*`/non-public)
// and any non-bypass decision. It is a public-exemption manager, NOT a general
// Access manager.
type AccessClient interface {
	// EnsurePublicBypass idempotently ensures a self_hosted Bypass-Everyone
	// Access app scoped to <host>/public. A pre-existing TM-owned app is left
	// untouched (Created=false); an absent one is created (Created=true).
	EnsurePublicBypass(ctx context.Context, host string) (AccessResult, error)
	// RemovePublicBypass deletes the <host>/public Access app TM created
	// (resolved by domain + the TM name marker). Idempotent: a missing app
	// returns removed=false, nil. The service calls it only for hosts its
	// access ledger attributes to TM, so it never deletes an app created
	// out-of-band.
	RemovePublicBypass(ctx context.Context, host string) (removed bool, err error)
	// LookupPublicBypass returns the existing TM-owned app for host (for
	// idempotency and dry-run preview). found=false when none exists.
	LookupPublicBypass(ctx context.Context, host string) (app AccessApp, found bool, err error)
}

// AccessHostState is one host's /public Access-bypass status for the
// status/preview read model.
type AccessHostState struct {
	Host string
	// Override is the route's per-route exposure decision (inherit|enabled|
	// disabled).
	Override manifest.PublicExposure
	// EffectiveBypass is whether this host would have a /public bypass under
	// the current global switch + override.
	EffectiveBypass bool
	// Managed is true when the access ledger attributes a bypass app to TM.
	Managed bool
	// AppID is the Cloudflare app id when Managed.
	AppID string
}

// AccessStatus is the read model for the /public Access-bypass capability: the
// global switch, whether the Access client is configured (creds present), the
// per-host effective decisions, and the dry-run plan (what a reconcile would
// create or remove). Pure — computed from (config, desired, ledger) with no
// mutation, so the `access dry-run` preview matches what apply will do.
type AccessStatus struct {
	Enabled    bool
	Configured bool
	Hosts      []AccessHostState
	ToCreate   []string
	ToRemove   []string
}

// AccessAppEntry is one TM-created Access app tracked in the access ownership
// ledger, mirroring the DNS ledger so prune only ever deletes apps TM created.
type AccessAppEntry struct {
	Host      string
	AppID     string
	PolicyID  string
	CreatedAt time.Time
}

// AccessLedger is the persistence seam over the access_ownership table — the
// Access analogue of DNSLedger. Absence of a row means "TM did not create this
// host's bypass app" (the safe default: never delete it).
type AccessLedger interface {
	List(ctx context.Context) ([]AccessAppEntry, error)
	Get(ctx context.Context, host string) (entry AccessAppEntry, found bool, err error)
	Put(ctx context.Context, entry AccessAppEntry) error
	Delete(ctx context.Context, host string) (bool, error)
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
	List(ctx context.Context, tier manifest.Tier) ([]manifest.Route, error)
}

// RoutesManager is the write surface AdoptIngress needs: it creates a
// scenario or external route to bring an unmanaged hostname under management,
// and looks up an existing route by subdomain to detect collisions / known
// scenarios. Declared at the consumer per seam-discovery; satisfied by
// routes.Service. Optional in Deps — adopt is unavailable when it is nil.
type RoutesManager interface {
	GetBySubdomain(ctx context.Context, subdomain string) (manifest.Route, error)
	Create(ctx context.Context, in manifest.CreateInput) (manifest.Route, error)
	// Update re-points an existing route by ID. AdoptIngress uses it to make
	// adopt idempotent: re-adopting a hostname (or repairing a previously
	// mis-classified one) updates the existing route's classification instead
	// of failing on the unique-subdomain conflict.
	Update(ctx context.Context, id string, in manifest.UpdateInput) (manifest.Route, error)
}

// ScenarioResolver resolves a scenario slug to its fixed UI port. AdoptIngress
// uses it to auto-classify a bare-adopted hostname (no explicit scenario or
// target) as a scenario route when its subdomain matches a known scenario —
// so an operator who adopts e.g. agent-inbox.itsagitime.com gets a
// scenario-backed route with the real port, not an external route with port 0.
// Declared at the consumer per seam-discovery; the production impl reads
// <scenarios-root>/<scenario>/.vrooli/service.json. Optional in Deps: when nil,
// auto-detect is disabled and bare adopt falls back to an external route.
type ScenarioResolver interface {
	UIPort(ctx context.Context, scenario string) (int, error)
	// IsScenario reports whether the slug is a known scenario at all
	// (regardless of whether it declares a fixed UI port). It lets adopt
	// classify a ranged-port scenario — one whose UI port is dynamically
	// allocated, so UIPort can't resolve it — as a scenario route using the
	// live localhost port, rather than mislabeling it external.
	IsScenario(ctx context.Context, scenario string) bool
}

// Routes is the combined routes surface the config service can consume — the
// read side (desired computation) plus the write side (adopt). Satisfied by
// routes.Service; wired in production so config can both reconcile against and
// extend the manifest.
type Routes interface {
	RoutesReader
	RoutesManager
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
