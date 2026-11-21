// Request-local verified human caller propagation for CreateRun adapters.
package owneridentity

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/scopecatalog"
)

type (
	createRunCallerKey struct{}
	createRunCaller    struct {
		token    string
		identity Identity
		headers  http.Header
	}
)

var ErrCreateRunCaller = errors.New("verified human agent-manager:write caller required")

// AuthorizeCreateRunCaller verifies existing human proof, preserves its deadline,
// and refuses offered run identity until an adapter has an exact-parent route.
// No proof is minted, persisted, refreshed, or granted additional capabilities.
//
// enforced selects the single AUTH-01 behavior (the Phase-3 switch read is the
// caller's: pass true to always enforce, or CreateRunCallerEnforced() to honor
// the operator switch). When enforced is false, absent or unverifiable proof is
// NOT a refusal: a single verified human Bearer is still attached for
// attribution, and every other caller (scheduler, agent identity) continues
// without a qualified caller. No proof is minted, and an offered run identity is
// not verified here.
func AuthorizeCreateRunCaller(ctx context.Context, headers http.Header, verifier Validator, now time.Time, enforced bool) (context.Context, error) {
	if !enforced {
		// Unenforced attribution-only path (former AdmitCreateRunCaller switch-off
		// behavior): opportunistically qualify a lone verified human Bearer, never
		// refuse. Other callers pass through unqualified.
		if values := headers.Values("Authorization"); len(values) == 1 {
			if qualified, err := authorizeCreateRunCaller(ctx, http.Header{"Authorization": values}, verifier, now); err == nil {
				return qualified, nil
			}
		}
		return ctx, nil
	}
	return authorizeCreateRunCaller(ctx, headers, verifier, now)
}

// authorizeCreateRunCaller is the strict verification. It is the single
// implementation both the enforced and unenforced paths share.
func authorizeCreateRunCaller(ctx context.Context, headers http.Header, verifier Validator, now time.Time) (context.Context, error) {
	if principal, ok := identity.PrincipalFromContext(ctx); ok {
		if !principal.IsHuman() || len(headers.Values("X-Agent-Identity-Token")) != 0 || principal.ExpiresAt.IsZero() || !principal.ExpiresAt.After(now) || !scopecatalog.Resolve(principal.Scopes, "agent-manager:write") {
			return nil, ErrCreateRunCaller
		}
		proof := make(http.Header)
		for _, name := range []string{"Authorization", "Cf-Access-Jwt-Assertion"} {
			if values := headers.Values(name); len(values) > 1 {
				return nil, ErrCreateRunCaller
			} else if len(values) == 1 {
				if len(values[0]) > 16384 {
					return nil, ErrCreateRunCaller
				}
				proof[name] = append([]string(nil), values...)
			}
		}
		if len(proof) == 0 && principal.Source == identity.SourceScenarioAuthenticator {
			cookieName := strings.TrimSpace(os.Getenv("VROOLI_AUTH_SCENARIO_COOKIE"))
			if cookieName == "" {
				cookieName = "vrooli_access_token"
			}
			request := &http.Request{Header: headers}
			if cookie, err := request.Cookie(cookieName); err == nil && cookie.Value != "" && len(cookie.Value) <= 16384 {
				proof.Set("Authorization", "Bearer "+cookie.Value)
			}
		}
		if len(proof) == 0 {
			return nil, ErrCreateRunCaller
		}
		bearer := strings.TrimPrefix(proof.Get("Authorization"), "Bearer ")
		return context.WithValue(ctx, createRunCallerKey{}, createRunCaller{token: bearer, identity: Identity{Subject: principal.Subject, Email: principal.Email, Scopes: append([]string(nil), principal.Scopes...), ExpiresAt: principal.ExpiresAt}, headers: proof}), nil
	}

	values := headers.Values("Authorization")
	if len(values) != 1 || len(headers.Values("X-Agent-Identity-Token")) != 0 || !strings.HasPrefix(values[0], "Bearer ") || len(values[0]) > 16384 || verifier == nil {
		return nil, ErrCreateRunCaller
	}
	token := strings.TrimPrefix(values[0], "Bearer ")
	if token == "" || strings.ContainsAny(token, " \t\r\n") {
		return nil, ErrCreateRunCaller
	}
	principal, err := verifier.Validate(ctx, token)
	if err != nil || strings.TrimSpace(principal.Subject) == "" || !principal.ExpiresAt.After(now) {
		return nil, ErrCreateRunCaller
	}
	allowed := false
	for _, scope := range principal.Scopes {
		if scopecatalog.MatchCapability([]string{scope}, "agent-manager:write") {
			allowed = true
			break
		}
	}
	if !allowed {
		return nil, ErrCreateRunCaller
	}
	principal.Scopes = append([]string(nil), principal.Scopes...)
	return context.WithValue(ctx, createRunCallerKey{}, createRunCaller{token: token, identity: principal}), nil
}

// RequireCreateRunCaller holds work on a qualified caller. When enforced is
// false, an absent or expired caller does not hold work (former
// RequireAdmittedCreateRunCaller switch-off behavior); when true, a missing or
// expired caller is refused.
func RequireCreateRunCaller(ctx context.Context, now time.Time, enforced bool) error {
	if !enforced {
		return nil
	}
	p, ok := ctx.Value(createRunCallerKey{}).(createRunCaller)
	if !ok || (p.token == "" && len(p.headers) == 0) || !p.identity.ExpiresAt.After(now) || ctx.Err() != nil {
		return ErrCreateRunCaller
	}
	return nil
}

// CreateRunCallerBinding returns a stable, non-secret fingerprint of the
// verified human subject attached to this request. It deliberately excludes
// credentials, email, expiry, and mutable request metadata so a caller can be
// reverified after restart without allowing a different human to take over an
// uncertain owner operation.
func CreateRunCallerBinding(ctx context.Context) (string, error) {
	p, ok := ctx.Value(createRunCallerKey{}).(createRunCaller)
	if !ok || strings.TrimSpace(p.identity.Subject) == "" || p.token == "" && len(p.headers) == 0 {
		return "", ErrCreateRunCaller
	}
	// Both supported verification paths admit only human callers. Keep the
	// actor kind in the digest domain so future identity kinds cannot collide.
	sum := sha256.Sum256([]byte("human\x00" + strings.TrimSpace(p.identity.Subject)))
	return "v1:" + hex.EncodeToString(sum[:]), nil
}

// CreateRunAuthorization is usable only for the exact owner creation endpoint.
func CreateRunAuthorization(ctx context.Context, now time.Time) (string, error) {
	if err := RequireCreateRunCaller(ctx, now, true); err != nil {
		return "", err
	}
	p, _ := ctx.Value(createRunCallerKey{}).(createRunCaller)
	if p.token == "" {
		return "", ErrCreateRunCaller
	}
	return "Bearer " + p.token, nil
}

func RedactCreateRunCaller(ctx context.Context, text string) string {
	p, _ := ctx.Value(createRunCallerKey{}).(createRunCaller)
	if p.token != "" {
		text = strings.ReplaceAll(text, p.token, "[redacted]")
	}
	for _, values := range p.headers {
		for _, value := range values {
			text = strings.ReplaceAll(text, value, "[redacted]")
		}
	}

	return text
}

// CarryCreateRunCaller transfers an already verified request-local proof to an
// immediately queued worker context. Its original expiry is retained; it is
// neither durable admission nor authority for a future scheduled job.
//
// When enforced is false (former CarryAdmittedCreateRunCaller switch-off
// behavior), a still-valid caller is carried for attribution and an absent one
// leaves the target context without a caller (no error); when true, a missing
// or expired caller is refused.
func CarryCreateRunCaller(from, to context.Context, now time.Time, enforced bool) (context.Context, error) {
	if !enforced {
		if RequireCreateRunCaller(from, now, true) != nil {
			return to, nil
		}
		p, _ := from.Value(createRunCallerKey{}).(createRunCaller)
		return context.WithValue(to, createRunCallerKey{}, p), nil
	}
	if err := RequireCreateRunCaller(from, now, true); err != nil {
		return nil, err
	}
	p, _ := from.Value(createRunCallerKey{}).(createRunCaller)
	return context.WithValue(to, createRunCallerKey{}, p), nil
}

// WithCreateRunHeaders retains the complete offered fields until verification.
// It is request-local and must never be serialized into a queued record.
type createRunHeadersKey struct{}

func WithCreateRunHeaders(ctx context.Context, headers http.Header) context.Context {
	h := make(http.Header)
	for _, key := range []string{"Authorization", "Cf-Access-Jwt-Assertion", "Cookie", "X-Agent-Identity-Token"} {
		if values, ok := headers[http.CanonicalHeaderKey(key)]; ok {
			h[http.CanonicalHeaderKey(key)] = append([]string{}, values...)
		}
	}
	return context.WithValue(ctx, createRunHeadersKey{}, h)
}

func AuthorizeBoundCreateRunCaller(ctx context.Context, verifier Validator, now time.Time) (context.Context, error) {
	headers, _ := ctx.Value(createRunHeadersKey{}).(http.Header)
	return AuthorizeCreateRunCaller(ctx, headers, verifier, now, true)
}

// CreateRunRequestHeaders carries verified request-local provider proof only
// to the exact internal CreateRun hop. It never changes an agent into a human.
func CreateRunRequestHeaders(ctx context.Context, now time.Time) (http.Header, error) {
	if err := RequireCreateRunCaller(ctx, now, true); err != nil {
		return nil, err
	}
	p, _ := ctx.Value(createRunCallerKey{}).(createRunCaller)
	if len(p.headers) > 0 {
		return p.headers.Clone(), nil
	}
	return http.Header{"Authorization": []string{"Bearer " + p.token}}, nil
}
