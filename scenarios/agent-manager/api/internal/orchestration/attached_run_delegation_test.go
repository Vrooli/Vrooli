package orchestration

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/testutil"

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

func newDelegationOrchestrator(t *testing.T, verifier coreidentity.Principal, validToken string) (*Orchestrator, func()) {
	t.Helper()
	repos, events, cleanup := testutil.SetupTestRepos(t)
	orch := New(nil, nil, repos.Runs,
		WithEvents(events),
		WithIdentitySecret([]byte("attached-delegation-test-secret")),
		WithOwnerIdentity(fakeDelegationVerifier{validToken: validToken, principal: verifier}),
		WithStandingGrants(&fakeStandingGrants{}),
	)
	return orch, cleanup
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
	orch, cleanup := newDelegationOrchestrator(t, humanPrincipal(time.Now().Add(time.Hour)), token)
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
}

// Every unattributed path leaves the run with no owner and zero scope — the
// laundering guard. A present-but-unverifiable delegation conjures nothing.
func TestAttachRunRefusesUnverifiableDelegation(t *testing.T) {
	ctx := context.Background()
	validToken := delegationToken(t, delegationSession)

	cases := []struct {
		name      string
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
}
