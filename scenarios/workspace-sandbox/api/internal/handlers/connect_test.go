package handlers

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	workspacev1 "github.com/vrooli/vrooli/packages/proto/gen/go/workspace-sandbox/v1/workspace"
)

func TestConnectResolveWorkspaceValidatesSandboxID(t *testing.T) {
	_, err := NewConnectHandler(nil).ResolveWorkspace(context.Background(), connect.NewRequest(&workspacev1.ResolveWorkspaceRequest{SandboxId: "not-a-uuid"}))
	assertConnectCode(t, err, connect.CodeInvalidArgument)
}

func TestConnectCreateSandboxValidatesScope(t *testing.T) {
	_, err := NewConnectHandler(nil).CreateSandbox(context.Background(), connect.NewRequest(&workspacev1.CreateSandboxRequest{}))
	assertConnectCode(t, err, connect.CodeInvalidArgument)
}

func TestConnectGetSandboxDiffValidatesSandboxID(t *testing.T) {
	_, err := NewConnectHandler(nil).GetSandboxDiff(context.Background(), connect.NewRequest(&workspacev1.GetSandboxDiffRequest{SandboxId: "not-a-uuid"}))
	assertConnectCode(t, err, connect.CodeInvalidArgument)
}

func TestConnectPromoteSandboxRequiresConfirmation(t *testing.T) {
	_, err := NewConnectHandler(nil).PromoteSandbox(context.Background(), connect.NewRequest(&workspacev1.PromoteSandboxRequest{SandboxId: "not-a-uuid"}))
	assertConnectCode(t, err, connect.CodeFailedPrecondition)
}

func assertConnectCode(t *testing.T, err error, want connect.Code) {
	t.Helper()
	if err == nil {
		t.Fatal("expected an error")
	}
	got := connect.CodeOf(err)
	if got != want {
		t.Fatalf("connect code=%s, want %s; err=%v", got, want, err)
	}
}
