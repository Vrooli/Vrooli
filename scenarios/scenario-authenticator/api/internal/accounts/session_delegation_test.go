package accounts

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	coreidentity "github.com/vrooli/api-core/identity"

	"github.com/vrooli/api-core/schedule"
	"scenario-authenticator/internal/authcrypto"
)

// fakeHumanAssertion models the Cloudflare Access verifier: it yields a verified
// human only for the one assertion registered as valid; anything else (forged,
// empty, non-human) fails — exactly as the real managed-Cloudflare verifier would.
type fakeHumanAssertion struct {
	validAssertion string
	principal      coreidentity.Principal
}

func (f fakeHumanAssertion) VerifyHuman(_ context.Context, raw string) (coreidentity.Principal, bool) {
	if raw != "" && raw == f.validAssertion {
		return f.principal, true
	}
	return coreidentity.Principal{}, false
}

// A tunnel operator authenticates via Cloudflare Access, not an account bearer.
// MintSessionDelegation must accept that verified assertion as human proof (taking
// the subject from it) and still refuse a forged/unverifiable one.
func TestMintSessionDelegationAcceptsCloudflareAccess(t *testing.T) {
	ctx := context.Background()
	f := newServicePrincipalFixture(t)
	clock := f.service.clock.(*schedule.Fake)
	// A sibling service sharing the fixture's repo+signer, plus a Cf-Access verifier.
	svc := f.service
	svc.humanAssertion = fakeHumanAssertion{
		validAssertion: "cf-access-assertion",
		principal: coreidentity.Principal{
			Kind: coreidentity.ActorHuman, Subject: "canonical-owner", Email: "operator@tunnel",
			Scopes:   []string{"agent-manager:read", "agent-manager:write"},
			Verified: true, ExpiresAt: clock.Now().Add(35 * time.Minute),
		},
	}
	const sid = "33333333-3333-3333-3333-333333333333"

	// Positive: a verified Cf-Access assertion (NOT an account bearer) mints a
	// human, session-bound delegation carrying only the scopes supplied by the
	// explicit configured owner binding resolved before it reached this service.
	proofExpiresAt := svc.humanAssertion.(fakeHumanAssertion).principal.ExpiresAt
	token, delegationExpiresAt, err := svc.MintSessionDelegation(ctx, "cf-access-assertion", sid, "", RequestMeta{})
	if err != nil {
		t.Fatalf("cf-access mint: %v", err)
	}
	claims, err := f.signer.Validate(token, "")
	if err != nil {
		t.Fatalf("cf-access delegation failed verification: %v", err)
	}
	if claims.ActorKind != HumanActorKind || claims.WCSessionID != sid {
		t.Fatalf("unexpected cf-access delegation claims: actor=%q sid=%q", claims.ActorKind, claims.WCSessionID)
	}
	if claims.UserID != "canonical-owner" {
		t.Fatalf("subject = %q, want canonical-owner from the explicit owner binding", claims.UserID)
	}
	if !slices.Equal(claims.Scopes, []string{"agent-manager:read", "agent-manager:write"}) {
		t.Fatalf("cf-access delegation scopes = %v, want explicit owner-binding scopes", claims.Scopes)
	}
	if delegationExpiresAt.After(proofExpiresAt) || delegationExpiresAt.After(clock.Now().Add(f.signer.Expiry())) {
		t.Fatalf("delegation expires at %s after source proof %s or signer limit", delegationExpiresAt, proofExpiresAt)
	}
	if claims.ExpiresAt.Time != delegationExpiresAt || claims.ExpiresAt.Time.Sub(claims.IssuedAt.Time) <= DefaultSessionDelegationTTL {
		t.Fatalf("signed scoped lifetime = %s, metadata expiry %s; want > attribution limit %s", claims.ExpiresAt.Time.Sub(claims.IssuedAt.Time), delegationExpiresAt, DefaultSessionDelegationTTL)
	}
	if _, _, err := svc.MintSessionDelegation(ctx, "cf-access-assertion", sid, "web-console", RequestMeta{}); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("CF AM binding accepted named resource: %v", err)
	}

	// Negative: a forged / unverifiable assertion yields no human and is refused.
	if _, _, err := svc.MintSessionDelegation(ctx, "forged-assertion", sid, "", RequestMeta{}); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("forged cf-access mint: got %v, want ErrInvalidCredentials", err)
	}
}

func TestMintScopedDelegationHonorsShorterConfiguredSignerTTL(t *testing.T) {
	ctx := context.Background()
	f := newServicePrincipalFixture(t)
	clock := f.service.clock.(*schedule.Fake)
	keys, err := authcrypto.LoadOrGenerate(t.TempDir())
	if err != nil {
		t.Fatal("create scoped delegation test signer")
	}
	shortSigner := authcrypto.NewSigner(keys, authcrypto.SignerConfig{
		Issuer: "scenario-authenticator", Expiry: 20 * time.Minute, Now: clock.Now,
	})
	f.service.signer = shortSigner
	f.service.humanAssertion = fakeHumanAssertion{
		validAssertion: "proof",
		principal: coreidentity.Principal{
			Kind: coreidentity.ActorHuman, Subject: "canonical-owner", Verified: true,
			Scopes: []string{"agent-manager:read"}, ExpiresAt: clock.Now().Add(2 * time.Hour),
		},
	}
	token, expiresAt, err := f.service.MintSessionDelegation(ctx, "proof", "88888888-8888-8888-8888-888888888888", "", RequestMeta{})
	if err != nil {
		t.Fatalf("mint with 20m configured signer: %v", err)
	}
	claims, err := shortSigner.Validate(token, "")
	if err != nil {
		t.Fatal("verify token signed with configured signer")
	}
	lifetime := claims.ExpiresAt.Time.Sub(claims.IssuedAt.Time)
	if lifetime != 20*time.Minute || lifetime <= DefaultSessionDelegationTTL {
		t.Fatalf("scoped delegation lifetime = %s, want exactly 20m and >%s", lifetime, DefaultSessionDelegationTTL)
	}
	if expiresAt != claims.ExpiresAt.Time || expiresAt.After(clock.Now().Add(20*time.Minute)) || expiresAt.After(f.service.humanAssertion.(fakeHumanAssertion).principal.ExpiresAt) {
		t.Fatal("returned expiry does not match signed exp within signer/source proof limits")
	}

	// Attribution-only identity stays capped at ten minutes, but a shorter
	// configured signer remains the ceiling for every token class.
	shortSigner = authcrypto.NewSigner(keys, authcrypto.SignerConfig{
		Issuer: "scenario-authenticator", Expiry: 5 * time.Minute, Now: clock.Now,
	})
	f.service.signer = shortSigner
	f.service.humanAssertion = fakeHumanAssertion{
		validAssertion: "attribution-proof",
		principal: coreidentity.Principal{
			Kind: coreidentity.ActorHuman, Subject: "canonical-owner", Verified: true,
			ExpiresAt: clock.Now().Add(time.Hour),
		},
	}
	attributionToken, attributionExpiry, err := f.service.MintSessionDelegation(ctx, "attribution-proof", "99999999-9999-9999-9999-999999999999", "", RequestMeta{})
	if err != nil {
		t.Fatalf("mint attribution-only token with 5m signer: %v", err)
	}
	attributionClaims, err := shortSigner.Validate(attributionToken, "")
	if err != nil {
		t.Fatal("verify attribution-only token signed with 5m signer")
	}
	if lifetime := attributionClaims.ExpiresAt.Time.Sub(attributionClaims.IssuedAt.Time); lifetime != 5*time.Minute || attributionExpiry != attributionClaims.ExpiresAt.Time {
		t.Fatalf("attribution-only lifetime = %s, want 5m signer cap and matching signed exp", lifetime)
	}
}

func TestMintSessionDelegationRequiresVerifiedUnexpiredCloudflareHuman(t *testing.T) {
	ctx := context.Background()
	f := newServicePrincipalFixture(t)
	const sid = "55555555-5555-5555-5555-555555555555"
	cases := []struct {
		name      string
		principal coreidentity.Principal
	}{
		{
			name: "unverified human",
			principal: coreidentity.Principal{
				Kind: coreidentity.ActorHuman, Subject: "owner", Verified: false, ExpiresAt: time.Now().Add(time.Hour),
			},
		},
		{
			name: "agent principal",
			principal: coreidentity.Principal{
				Kind: coreidentity.ActorAgent, Subject: "owner", Verified: true, ExpiresAt: time.Now().Add(time.Hour),
			},
		},
		{
			name: "expired human",
			principal: coreidentity.Principal{
				Kind: coreidentity.ActorHuman, Subject: "owner", Verified: true, ExpiresAt: time.Now().Add(-time.Minute),
			},
		},
		{
			name: "missing expiry",
			principal: coreidentity.Principal{
				Kind: coreidentity.ActorHuman, Subject: "owner", Verified: true,
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := *f.service
			svc.humanAssertion = fakeHumanAssertion{validAssertion: "proof", principal: tc.principal}
			if _, _, err := svc.MintSessionDelegation(ctx, "proof", sid, "", RequestMeta{}); !errors.Is(err, ErrInvalidCredentials) {
				t.Fatalf("mint = %v, want ErrInvalidCredentials", err)
			}
		})
	}
}

// The mint is gated to a verified human, binds the delegation to the exact Web
// Console session, carries the verified account's current authorization scopes,
// and is refused for a service/agent token or a missing session/resource. These
// are the laundering and scope-of-power guards for the attach path.
func TestMintSessionDelegation(t *testing.T) {
	ctx := context.Background()
	f := newServicePrincipalFixture(t)
	const resource = "web-console"
	const wantAud = "scenario-authenticator:web-console"
	const sid = "11111111-1111-1111-1111-111111111111"

	// Positive: a verified human mints a session-bound human delegation.
	token, expiresAt, err := f.service.MintSessionDelegation(ctx, f.member, sid, resource, RequestMeta{})
	if err != nil {
		t.Fatalf("verified-human mint: %v", err)
	}
	if expiresAt.IsZero() {
		t.Fatal("mint returned a zero expiry")
	}
	claims, err := f.signer.Validate(token, wantAud)
	if err != nil {
		t.Fatalf("delegation failed verification at its audience: %v", err)
	}
	if claims.ActorKind != HumanActorKind {
		t.Fatalf("actor_kind = %q, want %q", claims.ActorKind, HumanActorKind)
	}
	if claims.WCSessionID != sid {
		t.Fatalf("wc_session_id = %q, want %q", claims.WCSessionID, sid)
	}
	if len(claims.Scopes) != 0 {
		t.Fatalf("delegation carried unexpected scopes %v for an account with no assignments", claims.Scopes)
	}
	if got := claims.ExpiresAt.Time.Sub(claims.IssuedAt.Time); got != DefaultSessionDelegationTTL || expiresAt != claims.ExpiresAt.Time {
		t.Fatalf("attribution-only lifetime = %s / metadata expiry %s, want %s and signed expiry", got, expiresAt, DefaultSessionDelegationTTL)
	}
	if _, err := f.signer.Validate(token, "scenario-authenticator:other"); err == nil {
		t.Fatal("delegation verified for an unrelated resource audience")
	}
	if strings.TrimSpace(claims.UserID) == "" {
		t.Fatal("delegation has no subject")
	}

	// Negative: a SERVICE token cannot mint (Validate rejects a non-human actor_kind).
	issued, err := f.service.IssueServicePrincipal(ctx, f.admin, "agent-manager", RequestMeta{})
	if err != nil {
		t.Fatalf("fixture service principal: %v", err)
	}
	_ = issued
	serviceCred := f.custody.values["agent-manager/web-console"]
	serviceAccess, err := f.service.ExchangeServiceCredential(ctx, serviceCred, RequestMeta{})
	if err != nil {
		t.Fatalf("fixture service exchange: %v", err)
	}
	if _, _, err := f.service.MintSessionDelegation(ctx, serviceAccess.AccessToken, sid, resource, RequestMeta{}); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("service token mint: got %v, want ErrInvalidCredentials", err)
	}

	// Negative: anonymous / empty token is refused.
	if _, _, err := f.service.MintSessionDelegation(ctx, "", sid, resource, RequestMeta{}); err == nil {
		t.Fatal("anonymous mint was not refused")
	}

	// Negative: a missing session id is a validation error (no token).
	if _, _, err := f.service.MintSessionDelegation(ctx, f.member, "", resource, RequestMeta{}); err == nil {
		t.Fatal("empty wc_session_id was not refused")
	}

	// Empty resource is allowed: it resolves to the realm default audience, which
	// is what a relying party accepts with no explicit audience override.
	defaultToken, _, err := f.service.MintSessionDelegation(ctx, f.member, sid, "", RequestMeta{})
	if err != nil {
		t.Fatalf("empty-resource (realm default) mint: %v", err)
	}
	if _, err := f.signer.Validate(defaultToken, ""); err != nil {
		t.Fatalf("realm-default delegation failed verification: %v", err)
	}

	// Negative: an unknown (non-registered) resource is refused.
	if _, _, err := f.service.MintSessionDelegation(ctx, f.member, sid, "no-such-resource", RequestMeta{}); err == nil {
		t.Fatal("unknown resource was not refused")
	}

	// The token's short lifetime is enforced by the signer, not just reported in
	// the response metadata.
	clock, ok := f.service.clock.(*schedule.Fake)
	if !ok {
		t.Fatal("fixture clock is not controllable")
	}
	clock.Advance(DefaultSessionDelegationTTL)
	if _, err := f.signer.Validate(token, wantAud); err == nil {
		t.Fatal("expired session delegation still verified")
	}
}

func TestMintSessionDelegationUsesCurrentAccountAuthorizationScopes(t *testing.T) {
	ctx := context.Background()
	f := newServicePrincipalFixture(t)
	clock, ok := f.service.clock.(*schedule.Fake)
	if !ok {
		t.Fatal("fixture clock is not controllable")
	}
	const sid = "44444444-4444-4444-4444-444444444444"

	for _, scope := range []string{"agent-manager:read", "agent-manager:write"} {
		if _, err := f.service.GrantScope(ctx, f.member, "", scope, RequestMeta{}); err != nil {
			t.Fatalf("grant %s to fixture member: %v", scope, err)
		}
	}
	// A previously issued bearer must not gain scopes granted afterward.
	staleToken, _, err := f.service.MintSessionDelegation(ctx, f.member, sid, "", RequestMeta{})
	if err != nil {
		t.Fatalf("mint with stale verified account bearer: %v", err)
	}
	staleClaims, err := f.signer.Validate(staleToken, "")
	if err != nil {
		t.Fatalf("verify stale-bearer delegation: %v", err)
	}
	if len(staleClaims.Scopes) != 0 {
		t.Fatalf("stale bearer gained current scopes %v", staleClaims.Scopes)
	}

	fresh, err := f.service.ExchangeMachinePrincipal(ctx, "disposable-host", "unix:1002", "", RequestMeta{})
	if err != nil {
		t.Fatalf("issue fresh token with current account scopes: %v", err)
	}
	token, _, err := f.service.MintSessionDelegation(ctx, fresh.AccessToken, sid, "", RequestMeta{})
	if err != nil {
		t.Fatalf("mint with fresh verified account bearer: %v", err)
	}
	claims, err := f.signer.Validate(token, "")
	if err != nil {
		t.Fatalf("verify account delegation: %v", err)
	}
	if got := claims.ExpiresAt.Time.Sub(claims.IssuedAt.Time); got <= DefaultSessionDelegationTTL {
		t.Fatalf("scoped account delegation lifetime = %s, want > attribution-only %s", got, DefaultSessionDelegationTTL)
	}
	if claims.ExpiresAt.Time.After(fresh.AccessExpiresAt) || claims.ExpiresAt.Time.After(clock.Now().Add(f.signer.Expiry())) {
		t.Fatalf("delegation expiry %s exceeds source bearer %s or signer TTL", claims.ExpiresAt.Time, fresh.AccessExpiresAt)
	}
	want := []string{"agent-manager:read", "agent-manager:write"}
	if !slices.Equal(claims.Scopes, want) {
		t.Fatalf("delegated scopes = %v, want exact current account assignments %v", claims.Scopes, want)
	}
	if claims.WCSessionID != sid || claims.ActorKind != HumanActorKind {
		t.Fatalf("delegation identity/session binding = actor %q, session %q", claims.ActorKind, claims.WCSessionID)
	}

	if _, err := f.service.RevokeScope(ctx, f.member, "", "agent-manager:write", RequestMeta{}); err != nil {
		t.Fatalf("revoke member scope: %v", err)
	}
	refreshed, _, err := f.service.MintSessionDelegation(ctx, fresh.AccessToken, sid, "", RequestMeta{})
	if err != nil {
		t.Fatalf("mint after scope revocation: %v", err)
	}
	refreshedClaims, err := f.signer.Validate(refreshed, "")
	if err != nil {
		t.Fatalf("verify refreshed delegation: %v", err)
	}
	if !slices.Equal(refreshedClaims.Scopes, []string{"agent-manager:read"}) {
		t.Fatalf("delegated scopes after revocation = %v, want [agent-manager:read]", refreshedClaims.Scopes)
	}
}

func TestMintSessionDelegationExpiresNoLaterThanAccountBearer(t *testing.T) {
	ctx := context.Background()
	f := newServicePrincipalFixture(t)
	clock, ok := f.service.clock.(*schedule.Fake)
	if !ok {
		t.Fatal("fixture clock is not controllable")
	}
	clock.Advance(time.Hour - time.Minute)

	token, delegationExpiresAt, err := f.service.MintSessionDelegation(ctx, f.member, "66666666-6666-6666-6666-666666666666", "", RequestMeta{})
	if err != nil {
		t.Fatalf("mint near account-token expiry: %v", err)
	}
	accountExpiresAt := clock.Now().Add(time.Minute)
	if delegationExpiresAt.After(accountExpiresAt) {
		t.Fatalf("delegation expires at %s after verified account bearer at %s", delegationExpiresAt, accountExpiresAt)
	}
	claims, err := f.signer.Validate(token, "")
	if err != nil || claims.ExpiresAt.Time != delegationExpiresAt {
		t.Fatal("returned delegation expiry does not match the signed near-expiry claim")
	}
}

func TestMintSessionDelegationRefusesProofTooNearForSecondPrecision(t *testing.T) {
	ctx := context.Background()
	f := newServicePrincipalFixture(t)
	clock, ok := f.service.clock.(*schedule.Fake)
	if !ok {
		t.Fatal("fixture clock is not controllable")
	}
	// A provider may return a proof deadline within the current Unix second.
	// NumericDate would floor that deadline to the current second, so the
	// service must refuse instead of returning a token that cannot be used once.
	f.service.humanAssertion = fakeHumanAssertion{
		validAssertion: "near-expiry-proof",
		principal: coreidentity.Principal{
			Kind: coreidentity.ActorHuman, Subject: "canonical-owner", Verified: true,
			ExpiresAt: clock.Now().Add(500 * time.Millisecond),
		},
	}
	if _, _, err := f.service.MintSessionDelegation(ctx, "near-expiry-proof", "77777777-7777-7777-7777-777777777777", "", RequestMeta{}); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("near-second-expiry mint = %v, want refusal", err)
	}
}
