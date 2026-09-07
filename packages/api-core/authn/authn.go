// Package authn composes verified request identity providers and installs the
// passive middleware used by standard API servers.
package authn

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/vrooli/api-core/cloudflareaccess"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/provenance"
)

var ErrNoProvider = errors.New("authentication provider not configured")

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
	Providers   []Provider
	RecoveryURL string
}

// FromEnvironment builds the standard Cloudflare provider from the
// manifest-backed runtime binding. Scenario-authenticator is intentionally not
// constructed here because its verifier and account contract remain owned by
// that provider; consumers adapt it with TokenProvider or a local adapter.
func FromEnvironment(getenv func(string) string) (Config, error) {
	if getenv == nil {
		getenv = func(string) string { return "" }
	}
	var providers []Provider
	recoveryURL := strings.TrimSpace(getenv("VROOLI_AUTH_RECOVERY_URL"))
	for _, name := range strings.Split(strings.TrimSpace(getenv("VROOLI_AUTH_PROVIDERS")), ",") {
		switch strings.ToLower(strings.TrimSpace(name)) {
		case "", "scenario_authenticator":
			continue
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
	return Config{Providers: providers, RecoveryURL: recoveryURL}, nil
}

func (c Config) Enabled() bool { return len(c.Providers) > 0 }

// Authenticate evaluates all configured providers. A presented invalid or
// service credential wins over a valid provider, and different verified
// subjects fail closed as a conflict. Equal subjects may be supplied by more
// than one provider during migration.
func (c Config) Authenticate(ctx context.Context, req *http.Request) (identity.Principal, error) {
	if len(c.Providers) == 0 {
		return identity.Principal{}, identity.NewFailure(identity.FailureMissing, identity.SourceUnknown)
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
		if selected.Kind != candidate.Kind || !strings.EqualFold(strings.TrimSpace(selected.Subject), strings.TrimSpace(candidate.Subject)) {
			return identity.Principal{}, identity.NewFailure(identity.FailureConflict, identity.SourceUnknown)
		}
		selected.Sources = appendUnique(selected.Sources, candidate.Sources...)
		selected.Sources = appendUnique(selected.Sources, candidate.Source)
		selected.Roles = appendUniqueStrings(selected.Roles, candidate.Roles...)
		selected.Scopes = appendUniqueStrings(selected.Scopes, candidate.Scopes...)
	}
	return selected, nil
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
// Domain mutation boundaries must require a verified human principal.
func Middleware(cfg Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			if !cfg.Enabled() {
				next.ServeHTTP(w, r.WithContext(identity.WithStatus(ctx, identity.Status{State: identity.StateSignedOut})))
				return
			}
			principal, err := cfg.Authenticate(ctx, r)
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
					principal := identity.Principal{Kind: identity.ActorAgent, Subject: agent.RunID, Verified: true, Source: identity.SourceAgentProvenance, Scopes: append([]string(nil), agent.Scopes...)}
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
