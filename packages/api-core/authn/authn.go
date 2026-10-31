// Package authn composes verified request identity providers and installs the
// passive middleware used by standard API servers.
package authn

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/api-core/cloudflareaccess"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/provenance"
	"github.com/vrooli/api-core/scopecatalog"
)

var ErrNoProvider = errors.New("authentication provider not configured")

const scenarioAuthenticatorAccessCookie = "vrooli_access_token"

var (
	ErrHumanRequired   = errors.New("verified human identity required")
	ErrAgentRequired   = errors.New("verified agent identity required")
	ErrServiceRequired = errors.New("verified service identity required")
	ErrCapability      = errors.New("required capability not granted")
)

// Provider verifies a request-bound credential. Missing credentials should be
// returned as an identity.Failure with FailureMissing so the chain can safely
// distinguish an absent provider from a malformed presented credential.
type Provider interface {
	Source() identity.AuthSource
	VerifyRequest(context.Context, *http.Request) (identity.Principal, error)
}

// TokenVerifier adapts existing token verifiers (such as the Vrooli-native
// scenario-authenticator verifier) into the shared provider contract.
type TokenVerifier interface {
	Verify(context.Context, string) (identity.Principal, error)
}

type TokenProvider struct {
	ProviderSource identity.AuthSource
	Extract        func(*http.Request) string
	Verifier       TokenVerifier
}

// NewScenarioAuthenticatorProvider constructs the native Vrooli identity
// provider. It verifies the issuer, audience, RS256 key id, expiry, and claims
// through the shared JWT verifier; consumers do not need a private parser.
func NewScenarioAuthenticatorProvider(cfg JWTConfig) *JWTVerifier {
	cfg.Source = identity.SourceScenarioAuthenticator
	if strings.TrimSpace(cfg.Issuer) == "" {
		cfg.Issuer = "scenario-authenticator"
	}
	if strings.TrimSpace(cfg.Audience) == "" {
		cfg.Audience = "scenario-authenticator:default"
	}
	if strings.TrimSpace(cfg.JWKSURL) == "" && cfg.ResolveJWKS == nil {
		cfg.ResolveJWKS = func(ctx context.Context) (string, error) {
			base, err := discovery.ResolveScenarioURLDefault(ctx, "scenario-authenticator")
			if err != nil {
				return "", err
			}
			return strings.TrimRight(base, "/") + "/.well-known/jwks.json", nil
		}
	}
	return NewJWTVerifier(cfg)
}

func (p TokenProvider) Source() identity.AuthSource { return p.ProviderSource }

func (p TokenProvider) VerifyRequest(ctx context.Context, req *http.Request) (identity.Principal, error) {
	if p.Verifier == nil || p.Extract == nil {
		return identity.Principal{}, identity.NewFailure(identity.FailureUnavailable, p.ProviderSource)
	}
	raw := strings.TrimSpace(p.Extract(req))
	if raw == "" {
		return identity.Principal{}, identity.NewFailure(identity.FailureMissing, p.ProviderSource)
	}
	return p.Verifier.Verify(ctx, raw)
}

type Config struct {
	Providers        []Provider
	RecoveryURL      string
	IdentityMappings []IdentityMapping
	OwnerBindings    []OwnerBinding
}

// IdentityMapping is an operator-owned link between two verified provider
// subjects. Provider composition never infers that equal strings, emails, or
// claims identify the same person; a link must name both provider identities.
type IdentityMapping struct {
	LeftSource   identity.AuthSource
	LeftSubject  string
	RightSource  identity.AuthSource
	RightSubject string
}

func (m IdentityMapping) matches(left, right identity.Principal) bool {
	return (m.LeftSource == left.Source && m.LeftSubject == left.Subject && m.RightSource == right.Source && m.RightSubject == right.Subject) ||
		(m.RightSource == left.Source && m.RightSubject == left.Subject && m.LeftSource == right.Source && m.LeftSubject == right.Subject)
}

// FromEnvironment builds the standard provider chain from manifest-backed
// runtime bindings. Scenario-authenticator and Cloudflare are both verified
// through this package; domain scenarios only consume the resulting principal.
func FromEnvironment(getenv func(string) string) (Config, error) {
	if getenv == nil {
		getenv = func(string) string { return "" }
	}
	var providers []Provider
	recoveryURL := strings.TrimSpace(getenv("VROOLI_AUTH_RECOVERY_URL"))
	for _, name := range strings.Split(strings.TrimSpace(getenv("VROOLI_AUTH_PROVIDERS")), ",") {
		switch strings.ToLower(strings.TrimSpace(name)) {
		case "":
			continue
		case "scenario_authenticator":
			cookieName := strings.TrimSpace(getenv("VROOLI_AUTH_SCENARIO_COOKIE"))
			if cookieName == "" {
				cookieName = scenarioAuthenticatorAccessCookie
			}
			providers = append(providers, NewScenarioAuthenticatorProvider(JWTConfig{
				Issuer:     getenv("VROOLI_AUTH_SCENARIO_ISSUER"),
				Audience:   getenv("VROOLI_AUTH_SCENARIO_AUDIENCE"),
				JWKSURL:    getenv("VROOLI_AUTH_SCENARIO_JWKS_URL"),
				Client:     &http.Client{Timeout: 5 * time.Second},
				CookieName: cookieName,
			}))
		case "cloudflare_access":
			cfg, err := cloudflareaccess.ConfigFromEnv(getenv)
			if err != nil {
				return Config{}, err
			}
			verifier, err := cloudflareaccess.NewVerifier(cfg)
			if err != nil {
				return Config{}, err
			}
			providers = append(providers, verifier)
			if recoveryURL == "" {
				recoveryURL = strings.TrimRight(cfg.TeamDomain, "/") + "/cdn-cgi/access/login"
			}
		default:
			return Config{}, errors.New("unsupported VROOLI_AUTH_PROVIDERS value")
		}
	}
	var bindings []OwnerBinding
	raw := strings.TrimSpace(getenv("VROOLI_AUTH_OWNER_BINDINGS"))
	if raw == "" {
		path := strings.TrimSpace(getenv("VROOLI_AUTH_OWNER_BINDINGS_FILE"))
		explicit := path != ""
		if !explicit && strings.TrimSpace(getenv("HOME")) != "" {
			path = filepath.Join(getenv("HOME"), ".config", "vrooli", "owner-bindings.json")
		}
		if path != "" {
			file, err := os.Open(path)
			if err != nil && (explicit || !errors.Is(err, os.ErrNotExist)) {
				return Config{}, errors.New("owner binding policy file unavailable")
			}
			if err == nil {
				data, readErr := io.ReadAll(io.LimitReader(file, 65537))
				_ = file.Close()
				if readErr != nil || len(data) > 65536 || len(strings.TrimSpace(string(data))) == 0 {
					return Config{}, errors.New("invalid owner binding policy file")
				}
				raw = string(data)
			}
		}
	}
	if raw != "" {
		if err := json.Unmarshal([]byte(raw), &bindings); err != nil {
			return Config{}, errors.New("invalid VROOLI_AUTH_OWNER_BINDINGS")
		}
		for _, binding := range bindings {
			if binding.Source == identity.SourceUnknown || binding.Source == "" || binding.Subject == "" || binding.CanonicalSubject == "" {
				return Config{}, errors.New("owner bindings require provider, verified subject and canonical subject")
			}
		}
		// A shared host policy may target selected relying scenarios. Unrelated
		// providers keep their existing assignment instead of losing scopes.
		resource := strings.TrimSpace(getenv("VROOLI_SCENARIO"))
		if resource == "" {
			resource = strings.TrimSpace(getenv("SCENARIO_NAME"))
		}
		selected := make([]OwnerBinding, 0, len(bindings))
		for _, binding := range bindings {
			allowed := len(binding.Resources) == 0
			for _, candidate := range binding.Resources {
				if candidate == resource {
					allowed = true
					break
				}
			}
			if allowed {
				selected = append(selected, binding)
			}
		}
		bindings = selected
	}
	return Config{Providers: providers, RecoveryURL: recoveryURL, OwnerBindings: bindings}, nil
}

func (c Config) Enabled() bool { return len(c.Providers) > 0 }

// WithScenarioAuthenticator retains explicit provider configuration and adds
// the scenario's native audience when no native provider is configured.
func (c Config) WithScenarioAuthenticator(resource string) Config {
	for _, provider := range c.Providers {
		if provider != nil && provider.Source() == identity.SourceScenarioAuthenticator {
			return c
		}
	}
	c.Providers = append(append([]Provider(nil), c.Providers...), NewScenarioAuthenticatorProvider(JWTConfig{Audience: "scenario-authenticator:" + resource, CookieName: scenarioAuthenticatorAccessCookie}))
	return c
}

// Authenticate evaluates all configured providers. A presented invalid or
// service credential wins over a valid provider. Multiple providers may be
// combined only when an explicit IdentityMapping links their verified
// subjects; equal subject strings are not sufficient evidence.
func (c Config) Authenticate(ctx context.Context, req *http.Request) (identity.Principal, error) {
	if len(c.Providers) == 0 {
		if req != nil && (req.Header.Get("Authorization") != "" || req.Header.Get("Cf-Access-Jwt-Assertion") != "") {
			return identity.Principal{}, identity.NewFailure(identity.FailureUnavailable, identity.SourceUnknown)
		}
		return identity.Principal{}, identity.NewFailure(identity.FailureMissing, identity.SourceUnknown)
	}
	if req != nil && len(req.Header.Values("Authorization")) > 0 {
		bearerProvider := false
		for _, provider := range c.Providers {
			if provider != nil && provider.Source() != identity.SourceCloudflareAccess {
				bearerProvider = true
			}
		}
		if !bearerProvider {
			return identity.Principal{}, identity.NewFailure(identity.FailureInvalid, identity.SourceUnknown)
		}
	}
	var principals []identity.Principal
	var firstFailure *identity.Failure
	for _, provider := range c.Providers {
		if provider == nil {
			continue
		}
		principal, err := provider.VerifyRequest(ctx, req)
		if err != nil {
			failure, ok := identity.FailureFromError(err)
			if !ok {
				failure = *identity.NewFailure(identity.FailureInvalid, provider.Source())
			}
			if failure.Class == identity.FailureMissing {
				continue
			}
			if firstFailure == nil {
				firstFailure = &failure
			}
			continue
		}
		if !principal.IsVerified() && principal.Kind != identity.ActorService {
			failure := identity.NewFailure(identity.FailureInvalid, provider.Source())
			if firstFailure == nil {
				firstFailure = failure
			}
			continue
		}
		principals = append(principals, principal)
	}
	if firstFailure != nil {
		return identity.Principal{}, firstFailure
	}
	if len(principals) == 0 {
		return identity.Principal{}, identity.NewFailure(identity.FailureMissing, identity.SourceUnknown)
	}
	selected := principals[0]
	selected.Sources = appendUnique(selected.Sources, selected.Source)
	for _, candidate := range principals[1:] {
		if selected.Kind != candidate.Kind {
			return identity.Principal{}, identity.NewFailure(identity.FailureConflict, identity.SourceUnknown)
		}
		if selected.Source != candidate.Source {
			mapped := false
			for _, mapping := range c.IdentityMappings {
				if mapping.matches(selected, candidate) {
					mapped = true
					break
				}
			}
			if !mapped {
				return identity.Principal{}, identity.NewFailure(identity.FailureConflict, identity.SourceUnknown)
			}
		} else if !strings.EqualFold(strings.TrimSpace(selected.Subject), strings.TrimSpace(candidate.Subject)) {
			return identity.Principal{}, identity.NewFailure(identity.FailureConflict, identity.SourceUnknown)
		}
		selected.Sources = appendUnique(selected.Sources, candidate.Sources...)
		selected.Sources = appendUnique(selected.Sources, candidate.Source)
		selected.Roles = appendUniqueStrings(selected.Roles, candidate.Roles...)
		selected.Scopes = appendUniqueStrings(selected.Scopes, candidate.Scopes...)
		if !candidate.ExpiresAt.IsZero() && (selected.ExpiresAt.IsZero() || candidate.ExpiresAt.Before(selected.ExpiresAt)) {
			selected.ExpiresAt = candidate.ExpiresAt
		}
	}
	for _, binding := range c.OwnerBindings {
		if selected.IsHuman() && binding.Source == selected.Source && binding.Subject == selected.Subject {
			selected.Subject = binding.CanonicalSubject
			selected.Scopes = appendUniqueStrings(selected.Scopes, binding.Scopes...)
			break
		}
	}
	return selected, nil
}

func requireKind(ctx context.Context, kind identity.ActorKind, failure error) (identity.Principal, error) {
	principal, ok := identity.PrincipalFromContext(ctx)
	if !ok || principal.Kind != kind {
		return identity.Principal{}, failure
	}
	return principal, nil
}

// RequireHuman returns the verified human principal from request context.
func RequireHuman(ctx context.Context) (identity.Principal, error) {
	return requireKind(ctx, identity.ActorHuman, ErrHumanRequired)
}

// RequireAgent returns the verified agent principal from request context.
func RequireAgent(ctx context.Context) (identity.Principal, error) {
	return requireKind(ctx, identity.ActorAgent, ErrAgentRequired)
}

// RequireService returns the verified service principal from request context.
func RequireService(ctx context.Context) (identity.Principal, error) {
	return requireKind(ctx, identity.ActorService, ErrServiceRequired)
}

// RequireCapability checks one manifest-derived coarse capability against the
// verified principal's scopes. Domain authorization remains the caller's job.
func RequireCapability(ctx context.Context, required string) (identity.Principal, error) {
	principal, ok := identity.PrincipalFromContext(ctx)
	if !ok || !scopecatalog.Resolve(principal.Scopes, required) {
		return identity.Principal{}, ErrCapability
	}
	return principal, nil
}

func appendUnique(values []identity.AuthSource, additions ...identity.AuthSource) []identity.AuthSource {
	seen := make(map[identity.AuthSource]struct{}, len(values)+len(additions))
	out := make([]identity.AuthSource, 0, len(values)+len(additions))
	for _, value := range append(values, additions...) {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func appendUniqueStrings(values []string, additions ...string) []string {
	seen := make(map[string]struct{}, len(values)+len(additions))
	out := make([]string, 0, len(values)+len(additions))
	for _, value := range append(values, additions...) {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

// Middleware is deliberately passive: authentication failures become a
// browser-safe context status and do not make read-only routes unavailable.
// Domain boundaries select the actor, capability, operation and target policy.
func Middleware(cfg Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			principal, err := cfg.Authenticate(ctx, r)
			agent := provenance.FromContext(ctx)
			if r.Header.Get("X-Agent-Identity-Token") != "" && !agent.IsVerifiedAgent() {
				err = identity.NewFailure(identity.FailureInvalid, identity.SourceAgentProvenance)
			}
			if err == nil && agent.IsVerifiedAgent() && agent.Grant != nil {
				// A forwarded human credential cannot turn bounded delegated work
				// into an unrestricted operator request. Preserve the agent actor
				// and intersect its grant with the currently verified owner policy.
				if !principal.IsHuman() || principal.OwnerSubject() != agent.Subject || agent.Grant.Approver != agent.Subject {
					err = identity.NewFailure(identity.FailureConflict, identity.SourceAgentProvenance)
				} else {
					var scopes []string
					for _, scope := range agent.Scopes {
						if scopecatalog.Resolve(principal.Scopes, scope) {
							scopes = append(scopes, scope)
						}
					}
					expires := principal.ExpiresAt
					if expires.IsZero() || agent.Grant.ExpiresAt.Before(expires) {
						expires = agent.Grant.ExpiresAt
					}
					principal = identity.Principal{Kind: identity.ActorAgent, Subject: agent.RunID, Owner: agent.Subject, Verified: true, Source: identity.SourceAgentProvenance, Scopes: scopes, Grant: agent.Grant, ExpiresAt: expires}
				}
			}

			if err == nil {
				state := identity.StateVerified
				switch principal.Kind {
				case identity.ActorAgent:
					state = identity.StateAgent
				case identity.ActorService:
					state = identity.StateService
				case identity.ActorConflict:
					state = identity.StateConflict
				}
				status := identity.Status{State: state, Authenticated: true, Source: principal.Source, Principal: principal, RecoveryURL: cfg.RecoveryURL, ProviderSources: append([]identity.AuthSource(nil), principal.Sources...)}
				next.ServeHTTP(w, r.WithContext(identity.WithStatus(identity.WithPrincipal(ctx, principal), status)))
				return
			}
			if failure, ok := identity.FailureFromError(err); !ok || failure.Class != identity.FailureMissing {
				// Nested authentication must not retain an earlier principal when
				// the offered proof fails this relying party's verification.
				ctx = identity.WithPrincipal(ctx, identity.Principal{})
			}
			status := statusForFailure(err, cfg.RecoveryURL)
			status.ProviderSources = configuredSources(cfg.Providers)
			if status.Source == identity.SourceUnknown && len(status.ProviderSources) > 0 {
				// Prefer an external browser provider for recovery messaging when
				// a hybrid chain is signed out.
				status.Source = status.ProviderSources[len(status.ProviderSources)-1]
			}
			// Attribution is intentionally separate. It can describe an agent but
			// never turns that agent into a human principal.
			if failure, ok := identity.FailureFromError(err); ok && failure.Class == identity.FailureMissing {
				if agent := provenance.FromContext(ctx); agent.IsVerifiedAgent() {
					principal := identity.Principal{Kind: identity.ActorAgent, Subject: agent.RunID, Owner: agent.Subject, Verified: true, Source: identity.SourceAgentProvenance, Scopes: append([]string(nil), agent.Scopes...), Grant: agent.Grant}
					status.State = identity.StateAgent
					status.Authenticated = true
					status.Source = identity.SourceAgentProvenance
					status.Principal = principal
					status.ProviderSources = []identity.AuthSource{identity.SourceAgentProvenance}
					ctx = identity.WithPrincipal(ctx, principal)
				}
			}
			next.ServeHTTP(w, r.WithContext(identity.WithStatus(ctx, status)))
		})
	}
}

func configuredSources(providers []Provider) []identity.AuthSource {
	result := make([]identity.AuthSource, 0, len(providers))
	for _, provider := range providers {
		if provider == nil || provider.Source() == identity.SourceUnknown {
			continue
		}
		seen := false
		for _, source := range result {
			if source == provider.Source() {
				seen = true
				break
			}
		}
		if !seen {
			result = append(result, provider.Source())
		}
	}
	return result
}

func statusForFailure(err error, recovery string) identity.Status {
	status := identity.Status{State: identity.StateError, RecoveryURL: recovery}
	if failure, ok := identity.FailureFromError(err); ok {
		status.FailureClass = failure.Class
		status.Source = failure.Source
		switch failure.Class {
		case identity.FailureMissing:
			status.State = identity.StateSignedOut
		case identity.FailureExpired:
			status.State = identity.StateExpired
		case identity.FailureService:
			status.State = identity.StateService
		case identity.FailureConflict:
			status.State = identity.StateConflict
		}
	}
	return status
}
