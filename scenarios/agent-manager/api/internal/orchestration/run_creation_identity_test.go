package orchestration

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/identity"
	"agent-manager/internal/orchestration/phases"
	"github.com/google/uuid"
	coreidentity "github.com/vrooli/api-core/identity"
)

type createOwnerVerifier func(context.Context, string) (coreidentity.Principal, error)

func (f createOwnerVerifier) Verify(ctx context.Context, token string) (coreidentity.Principal, error) {
	return f(ctx, token)
}

func TestCreateRunIdentityRefusesInvalidAuthorityBeforeEffects(t *testing.T) {
	now := time.Now()
	const credential = "private-owner-credential"
	for _, variant := range []string{"missing", "wrong-owner", "wrong-scope", "wildcard", "expired", "expiry-boundary", "unavailable", "service", "unverified", "empty-owner-grant"} {
		t.Run(variant, func(t *testing.T) {
			req := CreateRunRequest{OwnerToken: credential, ExpectedOwnerSubject: "owner-a", RequestedScopes: []string{"agent-manager:supervise"}}
			principal := coreidentity.Principal{Kind: coreidentity.ActorHuman, Verified: true, Subject: "owner-a", Scopes: []string{"agent-manager:supervise"}, ExpiresAt: now.Add(time.Hour)}
			var verifyErr error
			switch variant {
			case "missing":
				req.OwnerToken = ""
			case "wrong-owner":
				principal.Subject = "owner-b"
			case "wrong-scope":
				req.RequestedScopes = []string{"agent-manager:admin"}
			case "wildcard":
				req.RequestedScopes = []string{"*"}
			case "expired":
				principal.ExpiresAt = now.Add(-time.Second)
			case "expiry-boundary":
				principal.ExpiresAt = now
			case "unavailable":
				verifyErr = errors.New(credential)
			case "service":
				principal.Kind = coreidentity.ActorService
			case "unverified":
				principal.Verified = false
			case "empty-owner-grant":
				principal.Scopes = nil
			}
			o := &Orchestrator{clock: func() time.Time { return now }, ownerIdentity: createOwnerVerifier(func(context.Context, string) (coreidentity.Principal, error) { return principal, verifyErr })}
			// No repositories: every refusal must precede reservation and dispatch.
			if run, err := o.CreateRun(context.Background(), req); run != nil || err == nil || strings.Contains(err.Error(), credential) {
				t.Fatal("invalid authority was accepted or leaked a credential")
			}
		})
	}
}

func TestCreateRunIdentityFutureWakesRetainNarrowingAndExpiry(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	ownerExpiry := now.Add(time.Hour)
	principal := coreidentity.Principal{Kind: coreidentity.ActorHuman, Verified: true, Subject: "owner-a", Scopes: []string{"agent-manager:supervise", "agent-manager:admin"}, ExpiresAt: ownerExpiry}
	o := &Orchestrator{clock: func() time.Time { return now }, ownerIdentity: createOwnerVerifier(func(context.Context, string) (coreidentity.Principal, error) { return principal, nil })}
	for _, scopes := range [][]string{{"agent-manager:supervise"}, {}} {
		for wake := 0; wake < 2; wake++ {
			req := CreateRunRequest{OwnerToken: "private-owner-credential", ExpectedOwnerSubject: principal.Subject, RequestedScopes: scopes}
			if err := o.resolveCreateRunIdentity(context.Background(), &req); err != nil {
				t.Fatal(err)
			}
			run := &domain.Run{ID: uuid.New(), TaskID: uuid.New(), OwnerSubject: req.OwnerSubject, OwnerScopes: req.OwnerScopes, RequestedScopes: req.RequestedScopes, OwnerExpiresAt: req.OwnerExpiresAt}
			body, err := json.Marshal(run)
			if err != nil || strings.Contains(string(body), req.OwnerToken) {
				t.Fatal("credential leaked into run")
			}
			var restored domain.Run
			if err := json.Unmarshal(body, &restored); err != nil {
				t.Fatal(err)
			}
			in := phases.GenerateIdentityTokenInput{Run: &restored, RequestedScopes: restored.RequestedScopes, Secret: []byte("test-secret"), Deps: phases.Deps{Clock: func() time.Time { return now.Add(time.Duration(wake) * time.Minute) }}}
			token := phases.GenerateIdentityToken(context.Background(), in)
			claims, err := identity.VerifyToken(token, in.Secret)
			if err != nil || claims.Subject != principal.Subject || !slices.Equal(claims.Scopes, scopes) || claims.ExpiresAt != ownerExpiry.Unix() {
				t.Fatal("future wake widened or lost its owner authority", err)
			}
			in.Deps.Clock = func() time.Time { return ownerExpiry }
			if phases.GenerateIdentityToken(context.Background(), in) != "" {
				t.Fatal("expired owner minted fresh authority")
			}
		}
	}
}

func TestCreateRunIdentityReplayRetainsOriginalAuthority(t *testing.T) {
	if err := validateRunIdentityReplay(&domain.Run{RequestedScopes: []string{}}, CreateRunRequest{}); err != nil {
		t.Fatal("historical observation-only replay lost compatibility", err)
	}
	run := &domain.Run{OwnerSubject: "owner-a", RequestedScopes: []string{"agent-manager:supervise"}}
	for _, req := range []CreateRunRequest{
		{OwnerSubject: "owner-b", RequestedScopes: run.RequestedScopes},
		{OwnerSubject: run.OwnerSubject},
		{OwnerSubject: run.OwnerSubject, RequestedScopes: []string{}},
		{OwnerSubject: run.OwnerSubject, RequestedScopes: []string{"agent-manager:admin"}},
	} {
		if validateRunIdentityReplay(run, req) == nil {
			t.Fatal("replay changed original authority")
		}
	}
	if err := validateRunIdentityReplay(run, CreateRunRequest{OwnerSubject: run.OwnerSubject, RequestedScopes: run.RequestedScopes}); err != nil {
		t.Fatal(err)
	}
}
