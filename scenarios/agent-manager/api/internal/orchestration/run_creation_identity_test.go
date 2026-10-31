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
	eventpb "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-events/v1/domain"
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
			principal := coreidentity.Principal{Kind: coreidentity.ActorHuman, Verified: true, Subject: "owner-a", Scopes: []string{"agent-manager:write", "agent-manager:supervise"}, ExpiresAt: now.Add(time.Hour)}
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
	principal := coreidentity.Principal{Kind: coreidentity.ActorHuman, Verified: true, Subject: "owner-a", Scopes: []string{"agent-manager:write", "agent-manager:supervise", "agent-manager:admin"}, ExpiresAt: ownerExpiry}
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

func TestCreateRunReplayRequiresExactOriginalRequest(t *testing.T) {
	profileID := uuid.New()
	base := CreateRunRequest{
		TaskID:         uuid.New(),
		IdempotencyKey: "create-run:test-key",
		AgentProfileID: &profileID,
		Until:          "finish the retained objective",
		Prompt:         "complete the requested work",
		WorkReferences: []*eventpb.WorkReference{{Kind: "effort", Id: "effort-1", Revision: "target-7"}},
		Environment:    map[string]string{"VROOLI_WORKLOAD": "epoch-3"},
	}
	digest, err := createRunRequestDigest(base)
	if err != nil {
		t.Fatal(err)
	}
	base.requestDigest = digest
	run := &domain.Run{ResolvedConfig: &domain.RunConfig{Admission: &domain.RunAdmission{CreateRequestDigest: digest}}}
	if err := validateRunIdentityReplay(run, base); err != nil {
		t.Fatalf("identical retry was rejected: %v", err)
	}

	changed := []struct {
		name string
		edit func(*CreateRunRequest)
	}{
		{name: "until", edit: func(req *CreateRunRequest) { req.Until = "stop after a different goal" }},
		{name: "work reference", edit: func(req *CreateRunRequest) {
			req.WorkReferences = []*eventpb.WorkReference{{Kind: "effort", Id: "effort-2", Revision: "target-7"}}
		}},
		{name: "profile", edit: func(req *CreateRunRequest) { other := uuid.New(); req.AgentProfileID = &other }},
		{name: "workload environment", edit: func(req *CreateRunRequest) { req.Environment = map[string]string{"VROOLI_WORKLOAD": "epoch-4"} }},
		{name: "prompt override", edit: func(req *CreateRunRequest) { req.Prompt = "do different work" }},
	}
	for _, tc := range changed {
		t.Run(tc.name, func(t *testing.T) {
			req := base
			req.requestDigest = ""
			req.WorkReferences = append([]*eventpb.WorkReference(nil), base.WorkReferences...)
			tc.edit(&req)
			if err := validateRunIdentityReplay(run, req); err == nil {
				t.Fatal("same-key replay accepted changed request terms")
			} else if coded, ok := err.(interface{ Code() domain.ErrorCode }); !ok || coded.Code() != domain.ErrCodeStateIdempotencyReplayConflict {
				t.Fatalf("changed same-key request code=%v; want exact owner replay-conflict code", err)
			}
		})
	}

	// Historical admissions have no request digest. They remain on the
	// previously established identity-only compatibility path.
	legacy := &domain.Run{ResolvedConfig: &domain.RunConfig{Admission: &domain.RunAdmission{}}}
	if err := validateRunIdentityReplay(legacy, CreateRunRequest{}); err != nil {
		t.Fatalf("historical admission lost replay compatibility: %v", err)
	}
}
