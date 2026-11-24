package permissions

import (
	"context"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/vrooli/internal/operatorstate"
	permissionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/permissions"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/operatorstateapi"
)

func TestRegistryTransportDerivesActorAndReturnsDurableWriteReceipt(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), operatorstate.StateFile)
	store := operatorstate.New(operatorstate.Config{RepoRoot: filepath.Dir(statePath), StatePath: func(context.Context) (string, error) { return statePath, nil }})
	handler := NewConnectHandler(operatorstateapi.New(store, nil))
	ctx := identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorHuman, Subject: "operator-a", Verified: true, Scopes: []string{"vrooli-onboarding:read", "vrooli-onboarding:write"}})

	listed, err := handler.ListPermissions(ctx, connect.NewRequest(&permissionsv1.ListPermissionsRequest{}))
	if err != nil {
		t.Fatalf("list permissions: %v", err)
	}
	if len(listed.Msg.GetPermissions()) != 1 || listed.Msg.GetPermissions()[0].GetId() != "onboarding/operator-state" {
		t.Fatalf("permission inventory = %+v", listed.Msg.GetPermissions())
	}
	filtered, err := handler.ListPermissions(ctx, connect.NewRequest(&permissionsv1.ListPermissionsRequest{Owner: "vrooli-onboarding"}))
	if err != nil || len(filtered.Msg.GetPermissions()) != 1 || filtered.Msg.GetPermissions()[0].GetOwner() != "vrooli-onboarding" {
		t.Fatalf("filtered permission inventory = %+v, err=%v", filtered, err)
	}
	initial, err := handler.ReadPermission(ctx, connect.NewRequest(&permissionsv1.ReadPermissionRequest{PermissionId: "onboarding/operator-state", Owner: "vrooli-onboarding"}))
	if err != nil {
		t.Fatalf("read operator state: %v", err)
	}
	request := &permissionsv1.WritePermissionRequest{
		PermissionId: "onboarding/operator-state", Owner: "vrooli-onboarding",
		ExpectedRevision: initial.Msg.GetRevision(), IdempotencyKey: "permission-write-1",
		ValueJson: []byte(`{"state":{"scenarios":{"demo":{"enabled":true}}},"update_mask":["scenarios.demo.enabled"]}`),
	}
	first, err := handler.WritePermission(ctx, connect.NewRequest(request))
	if err != nil {
		t.Fatalf("write permission: %v", err)
	}
	if got := first.Msg.GetReceipt(); got == nil || got.GetActorId() != "operator-a" || got.GetPermissionId() != request.GetPermissionId() || got.GetIdempotencyKey() != request.GetIdempotencyKey() {
		t.Fatalf("receipt does not bind server-derived actor and request: %+v", got)
	}
	replayed, err := handler.WritePermission(ctx, connect.NewRequest(request))
	if err != nil {
		t.Fatalf("replay permission write: %v", err)
	}
	if first.Msg.GetReceipt().GetId() != replayed.Msg.GetReceipt().GetId() || first.Msg.GetReceipt().GetResultRevision() != replayed.Msg.GetReceipt().GetResultRevision() {
		t.Fatalf("replay receipt = %+v, original = %+v", replayed.Msg.GetReceipt(), first.Msg.GetReceipt())
	}
	changed := *request
	changed.ValueJson = []byte(`{"state":{"scenarios":{"demo":{"enabled":false}}},"update_mask":["scenarios.demo.enabled"]}`)
	if _, err := handler.WritePermission(ctx, connect.NewRequest(&changed)); connect.CodeOf(err) != connect.CodeAlreadyExists {
		t.Fatalf("changed request reuse error = %v, code = %s", err, connect.CodeOf(err))
	}
}

func TestRegistryTransportRejectsMissingVerifiedPrincipal(t *testing.T) {
	handler := NewConnectHandler(operatorstateapi.New(nil, nil))
	if _, err := handler.ListPermissions(context.Background(), connect.NewRequest(&permissionsv1.ListPermissionsRequest{})); connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("unauthenticated inventory error = %v, code = %s", err, connect.CodeOf(err))
	}
}
