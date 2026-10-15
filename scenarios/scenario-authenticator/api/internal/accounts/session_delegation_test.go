package accounts

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// fakeHumanAssertion models the Cloudflare Access verifier: it yields a verified
// human only for the one assertion registered as valid; anything else (forged,
// empty, non-human) fails — exactly as the real managed-Cloudflare verifier would.
type fakeHumanAssertion struct {
	validAssertion string
	subject        string
	email          string
}

func (f fakeHumanAssertion) VerifyHuman(_ context.Context, raw string) (string, string, bool) {
	if raw != "" && raw == f.validAssertion {
		return f.subject, f.email, true
	}
	return "", "", false
}

// A tunnel operator authenticates via Cloudflare Access, not an account bearer.
// MintSessionDelegation must accept that verified assertion as human proof (taking
// the subject from it) and still refuse a forged/unverifiable one.
func TestMintSessionDelegationAcceptsCloudflareAccess(t *testing.T) {
	ctx := context.Background()
	f := newServicePrincipalFixture(t)
	// A sibling service sharing the fixture's repo+signer, plus a Cf-Access verifier.
	svc := f.service
	svc.humanAssertion = fakeHumanAssertion{validAssertion: "cf-access-assertion", subject: "operator@tunnel", email: "operator@tunnel"}
	const sid = "33333333-3333-3333-3333-333333333333"

	// Positive: a verified Cf-Access assertion (NOT an account bearer) mints a
	// human, session-bound, scope-free delegation attributed to the Cf-Access subject.
	token, _, err := svc.MintSessionDelegation(ctx, "cf-access-assertion", sid, "", RequestMeta{})
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
	if claims.UserID != "operator@tunnel" {
		t.Fatalf("subject = %q, want operator@tunnel (taken from the verified assertion)", claims.UserID)
	}
	if len(claims.Scopes) != 0 {
		t.Fatalf("cf-access delegation carried scopes %v; must be attribution-only", claims.Scopes)
	}

	// Negative: a forged / unverifiable assertion yields no human and is refused.
	if _, _, err := svc.MintSessionDelegation(ctx, "forged-assertion", sid, "", RequestMeta{}); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("forged cf-access mint: got %v, want ErrInvalidCredentials", err)
	}
}

// The mint is gated to a verified human, binds the delegation to the exact Web
// Console session, carries no capability scopes, and is refused for a
// service/agent token or a missing session/resource. These are the laundering
// and scope-of-power guards for the attach-attribution path.
func TestMintSessionDelegation(t *testing.T) {
	ctx := context.Background()
	f := newServicePrincipalFixture(t)
	const resource = "web-console"
	const wantAud = "scenario-authenticator:web-console"
	const sid = "11111111-1111-1111-1111-111111111111"

	// Positive: a verified human mints a session-bound, human, scope-free delegation.
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
		t.Fatalf("delegation carried capability scopes %v; it must be attribution-only", claims.Scopes)
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
}
