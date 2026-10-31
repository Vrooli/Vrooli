package orchestration

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	"agent-manager/internal/domain"
	agentidentity "agent-manager/internal/identity"
	"agent-manager/internal/orchestration/testutil"

	"github.com/vrooli/api-core/authn"
	coreidentity "github.com/vrooli/api-core/identity"
)

// fakeDelegationVerifier stands in for the scenario-authenticator JWT verifier.
// It returns a verified principal only for the token registered as valid; every
// other token (wrong key, expired, forged, bare non-JWT) fails, exactly as the
// real root verifier would. The wc_session_id binding is still enforced by the
// production delegationWCSessionID parse, so this fake only models authenticity.
type fakeDelegationVerifier struct {
	validToken string
	principal  coreidentity.Principal
}

func (f fakeDelegationVerifier) Verify(_ context.Context, raw string) (coreidentity.Principal, error) {
	if raw != "" && raw == f.validToken {
		return f.principal, nil
	}
	return coreidentity.Principal{}, errors.New("unverified")
}

// delegationToken builds a real 3-part JWT whose payload carries wc_session_id.
// The signature segment is irrelevant to the fake verifier (which matches by the
// whole string); the payload is what production parses for the session binding.
func delegationToken(t *testing.T, wcSessionID string) string {
	t.Helper()
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","kid":"test"}`))
	payloadBytes, err := json.Marshal(map[string]any{
		"user_id": "operator-subject", "sub": "operator-subject",
		"actor_kind": "human", "wc_session_id": wcSessionID,
	})
	if err != nil {
		t.Fatal(err)
	}
	payload := base64.RawURLEncoding.EncodeToString(payloadBytes)
	return header + "." + payload + ".sig"
}

func humanPrincipal(expires time.Time) coreidentity.Principal {
	return coreidentity.Principal{
		Kind: coreidentity.ActorHuman, Subject: "operator-subject", Verified: true,
		Scopes: []string{"agent-manager:write"}, ExpiresAt: expires,
	}
}

const delegationSession = "11111111-1111-1111-1111-111111111111"

const attachedIdentitySecret = "attached-delegation-test-secret"

func newDelegationOrchestrator(t *testing.T, verifier coreidentity.Principal, validToken string) (*Orchestrator, func()) {
	return newDelegationOrchestratorAt(t, verifier, validToken, time.Time{})
}

func newDelegationOrchestratorAt(t *testing.T, verifier coreidentity.Principal, validToken string, now time.Time) (*Orchestrator, func()) {
	t.Helper()
	repos, events, cleanup := testutil.SetupTestRepos(t)
	options := []Option{
		WithEvents(events),
		WithIdentitySecret([]byte(attachedIdentitySecret)),
		WithOwnerIdentity(fakeDelegationVerifier{validToken: validToken, principal: verifier}),
		WithStandingGrants(&fakeStandingGrants{}),
	}
	if !now.IsZero() {
		options = append(options, WithClock(func() time.Time { return now }))
	}
	orch := New(nil, nil, repos.Runs, options...)
	return orch, cleanup
}

func assertAttachedIdentity(t *testing.T, result *AttachRunResult, subject string, scopes []string) *agentidentity.Claims {
	t.Helper()
	claims, err := agentidentity.VerifyToken(result.Token, []byte(attachedIdentitySecret))
	if err != nil {
		t.Fatalf("verify attached agent identity token: %v", err)
	}
	if claims.Subject != subject {
		t.Fatalf("attached token subject = %q, want %q", claims.Subject, subject)
	}
	if !slices.Equal(claims.Scopes, scopes) {
		t.Fatalf("attached token scopes = %v, want %v", claims.Scopes, scopes)
	}
	if claims.RunID != result.Run.ID || claims.TaskID != result.Run.TaskID {
		t.Fatalf("attached token run/task = %s/%s, want %s/%s", claims.RunID, claims.TaskID, result.Run.ID, result.Run.TaskID)
	}
	if claims.Meta["execution_mode"] != string(domain.ExecutionModeAttached) {
		t.Fatalf("attached token execution mode = %q", claims.Meta["execution_mode"])
	}
	return claims
}

// A verified human delegation bound to the web-console session attributes the
// run to the operator. The harness/coding-agent session id is deliberately
// DIFFERENT from the web-console session id (as in production, where the launcher
// uses a codex thread id): the delegation's wc_session_id claim binds against the
// web-console session id carried on WebConsoleSessionID, never the harness id. If
// the binding were ever (re)based on HarnessSession, this test would fail.
func TestAttachRunAttributesVerifiedDelegation(t *testing.T) {
	ctx := context.Background()
	token := delegationToken(t, delegationSession)
	ownerExpiry := time.Now().Add(15 * time.Minute).Truncate(time.Second)
	owner := humanPrincipal(ownerExpiry)
	owner.Scopes = []string{"agent-manager:read", "agent-manager:write"}
	orch, cleanup := newDelegationOrchestrator(t, owner, token)
	t.Cleanup(cleanup)

	result, err := orch.AttachRun(ctx, AttachRunRequest{
		HarnessKind: "codex", HarnessSession: "codex-thread-abc",
		WebConsoleSessionID:  delegationSession,
		HumanOwnerDelegation: token,
	})
	if err != nil {
		t.Fatalf("attach run: %v", err)
	}
	if result.Run.OwnerSubject != "operator-subject" {
		t.Fatalf("owner = %q, want operator-subject", result.Run.OwnerSubject)
	}
	if len(result.Run.OwnerScopes) == 0 {
		t.Fatal("verified delegation attached no owner scopes")
	}
	if result.Run.ExecutionMode != domain.ExecutionModeAttached {
		t.Fatalf("execution mode = %q", result.Run.ExecutionMode)
	}
	claims := assertAttachedIdentity(t, result, "operator-subject", owner.Scopes)
	if claims.ExpiresAt != ownerExpiry.Unix() || result.ExpiresAt.After(ownerExpiry) {
		t.Fatalf("attached token expiry = %s (claim %d), want no later than delegation expiry %s", result.ExpiresAt, claims.ExpiresAt, ownerExpiry)
	}
	grants := orch.standingGrants.(*fakeStandingGrants).grants
	if len(grants) != 1 {
		t.Fatalf("verified delegation minted %d session grants, want one", len(grants))
	}
	grant := grants[0]
	if grant.Subject != owner.Subject || grant.SessionID != delegationSession || grant.SessionID == "codex-thread-abc" {
		t.Fatalf("session grant binding = (%q, %q), want operator and Web Console session %q", grant.Subject, grant.SessionID, delegationSession)
	}
	if grant.ExpiresAt.After(time.Unix(ownerExpiry.Unix(), 0)) || grant.ExpiresAt.After(time.Now().Add(MaxSessionGrantLifetime)) {
		t.Fatalf("session grant expiry %s exceeds verified delegation or %s cap", grant.ExpiresAt, MaxSessionGrantLifetime)
	}
}

func TestAttachRunDoesNotMintSessionGrantFromInContextHumanFallback(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	ownerExpiry := now.Add(30 * time.Minute)
	owner := humanPrincipal(ownerExpiry)
	orch, cleanup := newDelegationOrchestratorAt(t, coreidentity.Principal{}, "", now)
	t.Cleanup(cleanup)
	ctx := coreidentity.WithPrincipal(context.Background(), owner)
	result, err := orch.AttachRun(ctx, AttachRunRequest{
		HarnessKind: "codex", HarnessSession: "codex-thread-fallback",
		WebConsoleSessionID: delegationSession, HumanOwnerDelegation: "invalid-delegation",
	})
	if err != nil {
		t.Fatalf("attach run with in-context human fallback: %v", err)
	}
	if result.Run.OwnerSubject != owner.Subject || len(result.Run.OwnerScopes) == 0 {
		t.Fatalf("in-context human attribution was lost: owner=%q scopes=%v", result.Run.OwnerSubject, result.Run.OwnerScopes)
	}
	if grants := orch.standingGrants.(*fakeStandingGrants).grants; len(grants) != 0 {
		t.Fatalf("unverified delegation minted session grants: %+v", grants)
	}
}

func TestAttachRunSessionGrantCannotOutliveDelegationAfterClockAdvance(t *testing.T) {
	baseNow := time.Now().Truncate(time.Second)
	ownerExpiry := baseNow.Add(3 * time.Minute)
	owner := humanPrincipal(ownerExpiry)
	token := delegationToken(t, delegationSession)
	repos, events, cleanup := testutil.SetupTestRepos(t)
	defer cleanup()
	clockReads := 0
	clock := func() time.Time {
		clockReads++
		if clockReads <= 3 {
			return baseNow
		}
		// Model a delayed grant persistence clock read. A relative-TTL
		// calculation made during attach would now mint beyond the proof.
		return baseNow.Add(2 * time.Second)
	}
	orch := New(nil, nil, repos.Runs,
		WithEvents(events), WithIdentitySecret([]byte(attachedIdentitySecret)),
		WithOwnerIdentity(fakeDelegationVerifier{validToken: token, principal: owner}),
		WithStandingGrants(&fakeStandingGrants{}), WithClock(clock),
	)
	result, err := orch.AttachRun(context.Background(), AttachRunRequest{
		HarnessKind: "codex", HarnessSession: "codex-thread-delayed-grant",
		WebConsoleSessionID: delegationSession, HumanOwnerDelegation: token,
	})
	if err != nil {
		t.Fatalf("AttachRun: %v", err)
	}
	grants := orch.standingGrants.(*fakeStandingGrants).grants
	if len(grants) != 1 {
		t.Fatalf("verified delegation minted %d grants, want one", len(grants))
	}
	grant := grants[0]
	wantDeadline := time.Unix(ownerExpiry.Unix(), 0)
	if grant.ExpiresAt.After(wantDeadline) || grant.Grant.ExpiresAt.After(wantDeadline) {
		t.Fatalf("grant expires after proof deadline: grant=%s operation=%s proof=%s (attached=%s)", grant.ExpiresAt, grant.Grant.ExpiresAt, wantDeadline, result.ExpiresAt)
	}
}

func TestAttachRunVerifiesRealDelegationJWTAndSessionBinding(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Unix(1_800_000_000, 0).UTC()
	const issuer = "scenario-authenticator"
	const audience = "scenario-authenticator:agent-manager"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/jwks" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"keys":[{"kid":"attached-test","kty":"RSA","alg":"RS256","n":%q,"e":%q}]}`,
			base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
			base64.RawURLEncoding.EncodeToString([]byte{1, 0, 1}))
	}))
	defer server.Close()
	verifier := authn.NewJWTVerifier(authn.JWTConfig{
		Source: coreidentity.SourceScenarioAuthenticator, Issuer: issuer, Audience: audience,
		JWKSURL: server.URL + "/jwks", Now: func() time.Time { return now },
	})

	cases := []struct {
		name      string
		audience  string
		session   string
		expiresAt time.Time
		wantOwner bool
	}{
		{name: "valid signature and bound human", audience: audience, session: delegationSession, expiresAt: now.Add(5 * time.Minute), wantOwner: true},
		{name: "wrong audience", audience: "scenario-authenticator:another-service", session: delegationSession, expiresAt: now.Add(5 * time.Minute)},
		{name: "expired signature", audience: audience, session: delegationSession, expiresAt: now.Add(-time.Second)},
		{name: "valid signature bound to another web session", audience: audience, session: "22222222-2222-2222-2222-222222222222", expiresAt: now.Add(5 * time.Minute)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			token := signAttachedDelegationJWT(t, key, map[string]any{
				"user_id": "owner-canonical", "actor_kind": "human",
				"scope": []string{"agent-manager:read", "agent-manager:write"},
				"iss":   issuer, "aud": tc.audience, "exp": tc.expiresAt.Unix(),
				"wc_session_id": tc.session,
			})
			repos, events, cleanup := testutil.SetupTestRepos(t)
			defer cleanup()
			orch := New(nil, nil, repos.Runs,
				WithEvents(events), WithIdentitySecret([]byte(attachedIdentitySecret)),
				WithOwnerIdentity(verifier), WithStandingGrants(&fakeStandingGrants{}),
				WithClock(func() time.Time { return now }),
			)
			result, err := orch.AttachRun(context.Background(), AttachRunRequest{
				HarnessKind: "codex", HarnessSession: "codex-thread-real-jwt",
				WebConsoleSessionID: delegationSession, HumanOwnerDelegation: token,
			})
			if err != nil {
				t.Fatalf("AttachRun: %v", err)
			}
			if tc.wantOwner {
				if result.Run.OwnerSubject != "owner-canonical" || !slices.Equal(result.Run.OwnerScopes, []string{"agent-manager:read", "agent-manager:write"}) {
					t.Fatalf("real JWT owner=%q scopes=%v", result.Run.OwnerSubject, result.Run.OwnerScopes)
				}
				assertAttachedIdentity(t, result, "owner-canonical", []string{"agent-manager:read", "agent-manager:write"})
				return
			}
			if result.Run.OwnerSubject != "" || len(result.Run.OwnerScopes) != 0 {
				t.Fatalf("rejected real JWT attributed owner=%q scopes=%v", result.Run.OwnerSubject, result.Run.OwnerScopes)
			}
			assertAttachedIdentity(t, result, "", nil)
		})
	}
}

func signAttachedDelegationJWT(t *testing.T, key *rsa.PrivateKey, claims map[string]any) string {
	t.Helper()
	header, err := json.Marshal(map[string]string{"alg": "RS256", "kid": "attached-test", "typ": "JWT"})
	if err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		t.Fatal(err)
	}
	input := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(payload)
	digest := sha256.Sum256([]byte(input))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	return input + "." + base64.RawURLEncoding.EncodeToString(signature)
}

func TestAttachRunCapsInContextHumanIdentityToOwnerExpiry(t *testing.T) {
	ownerExpiry := time.Now().Add(20 * time.Minute).Truncate(time.Second)
	owner := humanPrincipal(ownerExpiry)
	owner.Scopes = []string{"agent-manager:read"}
	orch, cleanup := newDelegationOrchestrator(t, owner, "")
	t.Cleanup(cleanup)
	ctx := coreidentity.WithPrincipal(context.Background(), owner)
	result, err := orch.AttachRun(ctx, AttachRunRequest{
		HarnessKind: "codex", HarnessSession: "codex-thread-context-human",
	})
	if err != nil {
		t.Fatalf("attach run with verified in-context human: %v", err)
	}
	claims := assertAttachedIdentity(t, result, owner.Subject, owner.Scopes)
	if claims.ExpiresAt != ownerExpiry.Unix() || result.ExpiresAt.After(ownerExpiry) {
		t.Fatalf("attached token expiry = %s (claim %d), want no later than human proof expiry %s", result.ExpiresAt, claims.ExpiresAt, ownerExpiry)
	}
}

func TestAttachRunRejectsUnusableDelegationExpiry(t *testing.T) {
	validToken := delegationToken(t, delegationSession)
	baseNow := time.Now().Add(10 * time.Minute).Truncate(time.Second).Add(100 * time.Millisecond)
	cases := []struct {
		name      string
		principal coreidentity.Principal
		now       time.Time
	}{
		{name: "unverified", principal: coreidentity.Principal{Kind: coreidentity.ActorHuman, Subject: "operator-subject", Scopes: []string{"agent-manager:write"}, ExpiresAt: baseNow.Add(time.Hour)}},
		{name: "expired", principal: humanPrincipal(time.Now().Add(-time.Minute))},
		{name: "missing expiry", principal: humanPrincipal(time.Time{})},
		{
			name:      "future but no whole-second token lifetime",
			principal: humanPrincipal(time.Unix(baseNow.Unix(), 0).Add(900 * time.Millisecond)),
			now:       baseNow,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			orch, cleanup := newDelegationOrchestratorAt(t, tc.principal, validToken, tc.now)
			t.Cleanup(cleanup)
			result, err := orch.AttachRun(context.Background(), AttachRunRequest{
				HarnessKind: "codex", HarnessSession: "codex-thread-expiry",
				WebConsoleSessionID: delegationSession, HumanOwnerDelegation: validToken,
			})
			if err != nil {
				t.Fatalf("AttachRun: %v", err)
			}
			if result.Run.OwnerSubject != "" || len(result.Run.OwnerScopes) != 0 {
				t.Fatalf("unusable proof attributed owner=%q scopes=%v; want no owner and no scopes", result.Run.OwnerSubject, result.Run.OwnerScopes)
			}
			assertAttachedIdentity(t, result, "", nil)
			if result.ExpiresAt.Before(time.Now().Add(23 * time.Hour)) {
				t.Fatalf("ownerless identity expiry = %s; want the existing 24-hour attached lifetime", result.ExpiresAt)
			}
		})
	}
}

func TestAttachRunInContextHumanWithoutUsableExpiryGetsNoScopes(t *testing.T) {
	baseNow := time.Now().Add(10 * time.Minute).Truncate(time.Second).Add(100 * time.Millisecond)
	cases := []struct {
		name    string
		owner   coreidentity.Principal
		now     time.Time
		wantSub string
	}{
		{name: "unverified", owner: coreidentity.Principal{Kind: coreidentity.ActorHuman, Subject: "operator-subject", Scopes: []string{"agent-manager:write"}, ExpiresAt: baseNow.Add(time.Hour)}},
		{name: "expired", owner: humanPrincipal(time.Now().Add(-time.Minute))},
		{name: "missing expiry", owner: humanPrincipal(time.Time{}), wantSub: "operator-subject"},
		{
			name:    "future but no whole-second token lifetime",
			owner:   humanPrincipal(time.Unix(baseNow.Unix(), 0).Add(900 * time.Millisecond)),
			now:     baseNow,
			wantSub: "operator-subject",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			orch, cleanup := newDelegationOrchestratorAt(t, coreidentity.Principal{}, "", tc.now)
			t.Cleanup(cleanup)
			ctx := coreidentity.WithPrincipal(context.Background(), tc.owner)
			result, err := orch.AttachRun(ctx, AttachRunRequest{HarnessKind: "codex", HarnessSession: "codex-context-expiry"})
			if err != nil {
				t.Fatalf("AttachRun: %v", err)
			}
			if result.Run.OwnerSubject != tc.wantSub || len(result.Run.OwnerScopes) != 0 {
				t.Fatalf("in-context proof owner=%q scopes=%v; want owner=%q and no scopes", result.Run.OwnerSubject, result.Run.OwnerScopes, tc.wantSub)
			}
			assertAttachedIdentity(t, result, tc.wantSub, nil)
			if result.ExpiresAt.Before(time.Now().Add(23 * time.Hour)) {
				t.Fatalf("scope-free identity expiry = %s; want the existing 24-hour lifetime", result.ExpiresAt)
			}
		})
	}
}

// Every unattributed path leaves the run with no owner and zero scope — the
// laundering guard. A present-but-unverifiable delegation conjures nothing.
func TestAttachRunRefusesUnverifiableDelegation(t *testing.T) {
	ctx := context.Background()
	validToken := delegationToken(t, delegationSession)

	cases := []struct {
		name       string
		delegation string
	}{
		{"absent", ""},
		{"bare non-JWT string", "not-a-jwt"},
		{"wrong key / forged (verifier rejects)", delegationToken(t, delegationSession) + "x"},
		{"wrong wc_session_id (replay onto another session)", delegationToken(t, "22222222-2222-2222-2222-222222222222")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// The verifier accepts only validToken; the wrong-session case uses a
			// DIFFERENT valid-looking token so it passes authenticity but fails the
			// session binding — unless it equals validToken it also fails Verify, so
			// register the wrong-session token as "valid" to isolate the binding check.
			accepted := validToken
			if tc.name == "wrong wc_session_id (replay onto another session)" {
				accepted = tc.delegation
			}
			orch, cleanup := newDelegationOrchestrator(t, humanPrincipal(time.Now().Add(time.Hour)), accepted)
			t.Cleanup(cleanup)
			// WebConsoleSessionID is the legitimate web-console session; the replay
			// case presents a delegation bound to a DIFFERENT session, so the binding
			// check refuses it.
			result, err := orch.AttachRun(ctx, AttachRunRequest{
				HarnessKind: "codex", HarnessSession: "codex-thread-abc",
				WebConsoleSessionID:  delegationSession,
				HumanOwnerDelegation: tc.delegation,
			})
			if err != nil {
				t.Fatalf("attach run: %v", err)
			}
			if result.Run.OwnerSubject != "" {
				t.Fatalf("owner = %q, want empty (unattributed)", result.Run.OwnerSubject)
			}
			if len(result.Run.OwnerScopes) != 0 {
				t.Fatalf("owner scopes = %v, want none", result.Run.OwnerScopes)
			}
			if result.Run.ExecutionMode != domain.ExecutionModeAttached {
				t.Fatalf("execution mode = %q", result.Run.ExecutionMode)
			}
			assertAttachedIdentity(t, result, "", nil)
		})
	}
}

// A non-human verified principal (service/agent) cannot attribute an attached run.
func TestAttachRunRefusesNonHumanDelegation(t *testing.T) {
	ctx := context.Background()
	token := delegationToken(t, delegationSession)
	service := coreidentity.Principal{Kind: coreidentity.ActorService, Subject: "some-service", Verified: true}
	orch, cleanup := newDelegationOrchestrator(t, service, token)
	t.Cleanup(cleanup)

	result, err := orch.AttachRun(ctx, AttachRunRequest{
		HarnessKind: "codex", HarnessSession: "codex-thread-abc",
		WebConsoleSessionID:  delegationSession,
		HumanOwnerDelegation: token,
	})
	if err != nil {
		t.Fatalf("attach run: %v", err)
	}
	if result.Run.OwnerSubject != "" {
		t.Fatalf("service delegation attributed owner %q; want empty", result.Run.OwnerSubject)
	}
	assertAttachedIdentity(t, result, "", nil)
}
