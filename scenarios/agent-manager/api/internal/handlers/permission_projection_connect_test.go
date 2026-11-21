package handlers

import (
	"context"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"connectrpc.com/connect"
	"github.com/vrooli/api-core/identity"
	requestidentity "github.com/vrooli/api-core/identity"
	permissionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/permissions"
)

func projectionCtx(kind identity.ActorKind, subject, owner string, scopes ...string) context.Context {
	return identity.WithPrincipal(context.Background(), identity.Principal{Kind: kind, Subject: subject, Owner: owner, Verified: true, Scopes: scopes})
}

func TestPermissionProjectionConnectListsOwnerScopedTypedPage(t *testing.T) {
	admin := &fakeGrantAdmin{grants: []*domain.StandingGrant{
		{ID: "owner-grant", Subject: "worker", ApprovedBy: "owner", State: domain.StandingGrantApproved, ApprovedAt: time.Now().Add(-time.Hour), ExpiresAt: time.Now().Add(time.Hour), Grant: &requestidentity.OperationGrant{ID: "owner-grant", Approver: "owner", Operations: []string{"run.create"}, Targets: []string{"agent-manager:*"}}},
		{ID: "other-grant", Subject: "other", ApprovedBy: "other-owner", State: domain.StandingGrantApproved, ApprovedAt: time.Now(), ExpiresAt: time.Now().Add(time.Hour), Grant: &requestidentity.OperationGrant{ID: "other-grant", Approver: "other-owner", Operations: []string{"run.create"}, Targets: []string{"agent-manager:*"}}},
	}}
	h, err := NewPermissionProjectionHandler(admin)
	if err != nil {
		t.Fatal(err)
	}
	response, err := h.ListPermissionProjection(projectionCtx(identity.ActorAgent, "run", "owner"), connect.NewRequest(&permissionsv1.ListPermissionProjectionRequest{Limit: 64}))
	if err != nil {
		t.Fatal(err)
	}
	if response.Msg.GetOwner() != "agent-manager" || response.Msg.GetSourceRevision() == "" || len(response.Msg.GetGrants()) != 1 || response.Msg.GetGrants()[0].GetSubject() != "worker" || response.Msg.GetGrants()[0].GetGrantRef() == "owner-grant" {
		t.Fatalf("unexpected projected page: %+v", response.Msg)
	}
}

func TestPermissionProjectionConnectRejectsMissingOrWeakPrincipal(t *testing.T) {
	h, err := NewPermissionProjectionHandler(&fakeGrantAdmin{})
	if err != nil {
		t.Fatal(err)
	}
	for _, ctx := range []context.Context{context.Background(), identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorHuman, Subject: "human", Verified: true})} {
		if _, err := h.ListPermissionProjection(ctx, connect.NewRequest(&permissionsv1.ListPermissionProjectionRequest{})); connect.CodeOf(err) != connect.CodePermissionDenied && connect.CodeOf(err) != connect.CodeUnauthenticated {
			t.Fatalf("weak principal was not refused: %v", err)
		}
	}
}

func TestPermissionProjectionConnectRefusesStaleReadAndCursorRevision(t *testing.T) {
	admin := &fakeGrantAdmin{grants: []*domain.StandingGrant{
		{ID: "g1", Subject: "w1", ApprovedBy: "owner", State: domain.StandingGrantApproved, ApprovedAt: time.Now(), ExpiresAt: time.Now().Add(time.Hour), Grant: &requestidentity.OperationGrant{ID: "g1", Approver: "owner", Operations: []string{"run.create"}, Targets: []string{"agent-manager:*"}}},
		{ID: "g2", Subject: "w2", ApprovedBy: "owner", State: domain.StandingGrantApproved, ApprovedAt: time.Now(), ExpiresAt: time.Now().Add(time.Hour), Grant: &requestidentity.OperationGrant{ID: "g2", Approver: "owner", Operations: []string{"run.create"}, Targets: []string{"agent-manager:*"}}},
	}}
	h, err := NewPermissionProjectionHandler(admin)
	if err != nil {
		t.Fatal(err)
	}
	ctx := projectionCtx(identity.ActorAgent, "run", "owner")
	first, err := h.ListPermissionProjection(ctx, connect.NewRequest(&permissionsv1.ListPermissionProjectionRequest{Limit: 1}))
	if err != nil {
		t.Fatal(err)
	}
	if first.Msg.GetNextCursor() == "" {
		t.Fatal("expected next page")
	}
	admin.grants[0].Subject = "changed"
	if _, err := h.ListPermissionProjection(ctx, connect.NewRequest(&permissionsv1.ListPermissionProjectionRequest{Limit: 1, Cursor: first.Msg.GetNextCursor()})); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("stale cursor accepted: %v", err)
	}
	if _, err := h.ReadPermissionProjection(ctx, connect.NewRequest(&permissionsv1.ReadPermissionProjectionRequest{Kind: "grant", Reference: first.Msg.GetGrants()[0].GetGrantRef(), ExpectedSourceRevision: first.Msg.GetSourceRevision()})); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("stale read accepted: %v", err)
	}
}

func TestPermissionProjectionConnectReadsTypedScopeAndOpaqueGrantReference(t *testing.T) {
	admin := &fakeGrantAdmin{grants: []*domain.StandingGrant{{ID: "visible-internal-id", Subject: "worker", ApprovedBy: "owner", State: domain.StandingGrantRevoked, ApprovedAt: time.Now().Add(-time.Hour), ExpiresAt: time.Now().Add(time.Hour), Grant: &requestidentity.OperationGrant{ID: "visible-internal-id", Approver: "owner", Operations: []string{"run.create"}, Targets: []string{"agent-manager:*"}}}}}
	h, err := NewPermissionProjectionHandler(admin)
	if err != nil {
		t.Fatal(err)
	}
	ctx := projectionCtx(identity.ActorAgent, "run", "owner")
	page, err := h.ListPermissionProjection(ctx, connect.NewRequest(&permissionsv1.ListPermissionProjectionRequest{Limit: 64}))
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Msg.GetScopeCatalog()) == 0 || len(page.Msg.GetGrants()) != 1 {
		t.Fatalf("scope/grant projection missing: %+v", page.Msg)
	}
	grant := page.Msg.GetGrants()[0]
	if grant.GetGrantRef() == "visible-internal-id" || grant.GetState() != domain.StandingGrantRevoked {
		t.Fatalf("unsafe or inaccurate grant projection: %+v", grant)
	}
	grantResult, err := h.ReadPermissionProjection(ctx, connect.NewRequest(&permissionsv1.ReadPermissionProjectionRequest{Kind: "grant", Reference: grant.GetGrantRef(), ExpectedSourceRevision: page.Msg.GetSourceRevision()}))
	if err != nil || grantResult.Msg.GetGrant().GetGrantRef() != grant.GetGrantRef() || grantResult.Msg.GetGrant().GetState() != domain.StandingGrantRevoked {
		t.Fatalf("grant read: %+v %v", grantResult, err)
	}
	scope := page.Msg.GetScopeCatalog()[0]
	scopeResult, err := h.ReadPermissionProjection(ctx, connect.NewRequest(&permissionsv1.ReadPermissionProjectionRequest{Kind: "scope", Reference: scope.GetScope(), ExpectedSourceRevision: page.Msg.GetSourceRevision()}))
	if err != nil || scopeResult.Msg.GetScope().GetScope() != scope.GetScope() {
		t.Fatalf("scope read: %+v %v", scopeResult, err)
	}
}
