package policygate

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/cli-core/cliutil"
)

type (
	ownerProofKey struct{}
	ownerProof    struct {
		headers http.Header
		subject string
		source  string
		expires time.Time
	}
)

// WithVerifiedOwnerProof retains only the credential channel already verified
// by the inbound authentication owner. It is request-local, never a queued job
// field, and does not require launch/write scope for ordinary context reads.
func WithVerifiedOwnerProof(ctx context.Context, headers http.Header) context.Context {
	p, ok := PrincipalFromContext(ctx)
	if !ok || !p.Verified || p.Kind != cliutil.CallerKindHuman || p.ExpiresAt.IsZero() ||
		!p.ExpiresAt.After(time.Now()) || len(headers.Values(cliutil.HeaderAgentIdentityToken)) != 0 {
		return ctx
	}
	proof := make(http.Header)
	verifiedSources := map[identity.AuthSource]bool{identity.AuthSource(p.AuthSource): true}
	for _, source := range p.AuthSources {
		verifiedSources[identity.AuthSource(source)] = true
	}
	if verifiedSources[identity.SourceScenarioAuthenticator] {
		if len(headers.Values("Authorization")) > 1 {
			return ctx
		}
		raw := extractAccessToken(&http.Request{Header: headers})
		if raw == "" || len(raw) > 16384 || strings.ContainsAny(raw, " \t\r\n") {
			return ctx
		}
		proof.Set("Authorization", "Bearer "+raw)
	}
	if verifiedSources[identity.SourceCloudflareAccess] {
		key := "Cf-Access-Jwt-Assertion"
		if headers.Get(key) == "" {
			key = "Authorization"
		}
		values := headers.Values(key)
		if len(values) != 1 || values[0] == "" || len(values[0]) > 16384 {
			return ctx
		}
		proof.Set(key, values[0])
	}
	if len(proof) == 0 {
		return ctx
	}
	return context.WithValue(ctx, ownerProofKey{}, ownerProof{headers: proof, subject: p.Subject, source: p.AuthSource, expires: p.ExpiresAt})
}

// VerifiedOwnerHeaders carries immediate provider proof to another policy
// owner, which independently validates it. No principal or scopes are invented.
func VerifiedOwnerHeaders(ctx context.Context) http.Header {
	proof, ok := ctx.Value(ownerProofKey{}).(ownerProof)
	p, valid := PrincipalFromContext(ctx)
	if !ok || !valid || !p.Verified || p.Kind != cliutil.CallerKindHuman || p.Subject != proof.subject ||
		p.AuthSource != proof.source || !proof.expires.After(time.Now()) || !p.ExpiresAt.After(time.Now()) || ctx.Err() != nil {
		return nil
	}
	return proof.headers.Clone()
}
