package orchestration

import (
	"context"
	"slices"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/identity"
	"agent-manager/internal/orchestration/phases"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/scopecatalog"
)

// TestChildUnderApprovedParentInheritsAttenuatedAuthorityEndToEnd proves the
// full Phase-4 inheritance chain in one place: a child spawned by a parent that
// HAS authority inherits the parent's authority, bounded and attenuated, all the
// way into the credential it would present downstream.
//
// The chain under test:
//  1. caller-auth — authenticateCreateRunCaller, presented the parent's run
//     identity token, sets the child OwnerSubject = parent subject and
//     OwnerScopes = (requested ∩ parent-claims). (create_run_caller.go runOffered)
//  2. mint — phases.GenerateIdentityToken turns the persisted child OwnerScopes
//     (further narrowed by the child profile ceiling and request) into the
//     child's own signed identity token scopes. (phases/identity.go)
//  3. authorize — the minted scopes are exactly what scopecatalog.Resolve would
//     admit downstream: the inherited in-scope operation is admitted, and an
//     operation the parent never held is refused.
//
// The narrower minting and caller-intersection properties are pinned by
// phases.TestGenerateIdentityToken* and TestAuth01VerifiedCallerIntersectsBothChannels;
// the child-never-broadens escalation is pinned by
// TestInvariantChildRunNeverBroadensParent; the intentional zero-scope of a bare
// local attach is pinned by TestAttachRunRefusesUnverifiableDelegation. This test
// is the join that proves authority actually reaches a usable child credential —
// not that any one link works in isolation.
func TestChildUnderApprovedParentInheritsAttenuatedAuthorityEndToEnd(t *testing.T) {
	ctx := context.Background()
	now := time.Now().Truncate(time.Second)
	secret := []byte("child-authority-inheritance-e2e")

	// A parent that genuinely HAS authority: an active run holding read +
	// orchestrate scopes (as it would after being admitted under a human session
	// or an approved standing-grant lineage that carried those downstream scopes).
	parent := &domain.Run{ID: uuid.New(), TaskID: uuid.New(), OwnerSubject: "owner-a", Status: domain.RunStatusRunning}
	parentToken, err := identity.GenerateToken(&identity.Claims{
		RunID: parent.ID, TaskID: parent.TaskID, Subject: parent.OwnerSubject,
		Scopes:   []string{"agent-manager:read", "agent-manager:orchestrate"},
		IssuedAt: now.Unix(), ExpiresAt: now.Add(time.Hour).Unix(),
	}, secret)
	if err != nil {
		t.Fatal(err)
	}
	parent.IdentityTokenHash = identity.HashToken(parentToken)

	runs := &auth01Runs{effects: &auth01Effects{}, parent: parent, tokenHash: parent.IdentityTokenHash}
	o := &Orchestrator{clock: func() time.Time { return now }, runs: runs, identitySecret: secret}

	// Step 1 — caller-auth. The child presents the parent's exact run identity and
	// narrows to a subset the parent holds (read). No human token is offered: this
	// is a pure delegated child spawn.
	req := CreateRunRequest{
		TaskID:           parent.TaskID,
		RunIdentityToken: parentToken,
		ParentRunID:      &parent.ID,
		RequestedScopes:  []string{"agent-manager:read"},
	}
	if err := o.authenticateCreateRunCaller(ctx, &req); err != nil {
		t.Fatalf("legitimate delegated child spawn refused at caller-auth: %v", err)
	}
	if req.OwnerSubject != "owner-a" {
		t.Fatalf("child lost the parent owner attribution: %q", req.OwnerSubject)
	}
	if req.caller == nil || req.caller.Kind != "run" || req.caller.RunID != parent.ID {
		t.Fatalf("child was not attributed to its exact parent: %+v", req.caller)
	}
	// OwnerScopes is the parent ceiling (the inherited authority the run persists);
	// RequestedScopes is the narrowing. Neither may carry a scope the parent lacked.
	if !slices.Equal(req.OwnerScopes, []string{"agent-manager:read", "agent-manager:orchestrate"}) {
		t.Fatalf("child owner ceiling is not the parent ceiling: %v", req.OwnerScopes)
	}
	if !slices.Equal(req.RequestedScopes, []string{"agent-manager:read"}) {
		t.Fatalf("child narrowing lost or broadened: %v", req.RequestedScopes)
	}

	// Step 2 — mint. The persisted child run carries the inherited authority; its
	// own identity token is minted from that ceiling, the profile ceiling, and the
	// request narrowing. The profile here would allow destructive, proving the
	// parent ceiling (not the profile) is the binding limit.
	child := &domain.Run{
		ID: uuid.New(), TaskID: req.TaskID, ParentRunID: &parent.ID,
		OwnerSubject: req.OwnerSubject, OwnerScopes: req.OwnerScopes,
		RequestedScopes: req.RequestedScopes, Status: domain.RunStatusRunning,
	}
	profile := &domain.AgentProfile{
		ID: uuid.New(), ProfileKey: "child",
		DeclaredScopes: []string{"agent-manager:read", "agent-manager:orchestrate", "agent-manager:destructive"},
	}
	childToken := phases.GenerateIdentityToken(ctx, phases.GenerateIdentityTokenInput{
		Deps:            phases.Deps{Clock: func() time.Time { return now }},
		Run:             child,
		Profile:         profile,
		Secret:          secret,
		AccountScopes:   child.OwnerScopes,
		RequestedScopes: child.RequestedScopes,
	})
	if childToken == "" {
		t.Fatal("child identity token was not minted")
	}
	claims, err := identity.VerifyToken(childToken, secret)
	if err != nil {
		t.Fatalf("child identity token does not verify: %v", err)
	}
	if claims.Subject != "owner-a" {
		t.Fatalf("child credential lost the inherited owner: %q", claims.Subject)
	}
	// The minted scope set is exactly the attenuated intersection: the one scope
	// the child narrowed to AND the parent held. Not the profile's wider ceiling.
	if !slices.Equal(claims.Scopes, []string{"agent-manager:read"}) {
		t.Fatalf("minted child scopes are not the attenuated intersection: %v", claims.Scopes)
	}

	// Step 3 — authorize. The inherited credential admits the in-scope operation
	// downstream, and refuses a scope the parent never held — proving the
	// inheritance is bounded, not merely present.
	if !scopecatalog.Resolve(claims.Scopes, "agent-manager:read") {
		t.Fatal("inherited child credential cannot perform its one authorized operation")
	}
	if scopecatalog.Resolve(claims.Scopes, "agent-manager:destructive") {
		t.Fatal("inherited child credential admitted an operation its parent never held")
	}
	if scopecatalog.Resolve(claims.Scopes, "agent-manager:orchestrate") {
		t.Fatal("child credential retained a parent scope it explicitly narrowed away")
	}
}
