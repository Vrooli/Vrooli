package operatorstate

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/vrooli/internal/operatorstate"
	operatorstatev1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/operatorstate"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/operatorstateapi"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

func TestPatchPreservesPermissionDeniedFromOwnerProvider(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), operatorstate.StateFile)
	store := operatorstate.New(operatorstate.Config{RepoRoot: filepath.Dir(statePath), StatePath: func(context.Context) (string, error) { return statePath, nil }})
	handler := NewConnectHandler(operatorstateapi.New(store, nil))
	ctx := identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorHuman, Subject: "operator-a", Verified: true})
	_, err := handler.PatchOperatorState(ctx, connect.NewRequest(&operatorstatev1.PatchOperatorStateRequest{
		State:      &operatorstatev1.OperatorState{Scenarios: map[string]*operatorstatev1.ScenarioChoice{"demo": {Enabled: boolPointer(true)}}},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"scenarios.demo.enabled"}}, ExpectedRevision: "revision-1",
	}))
	if err == nil || connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("patch error = %v, code = %s; want preserved permission_denied", err, connect.CodeOf(err))
	}
}

func TestPatchReturnsDurableReceiptAndReplaysExactRequest(t *testing.T) {
	stateRoot := filepath.Join(t.TempDir(), "state")
	_, current, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(current), "../../../../"))
	store := operatorstate.New(operatorstate.Config{RepoRoot: repoRoot, StateRoot: stateRoot, SchemaPath: filepath.Join(repoRoot, operatorstate.SchemaPath)})
	service := operatorstateapi.New(store, nil)
	initial, err := store.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	handler := NewConnectHandler(service)
	ctx := identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorHuman, Subject: "operator-a", Verified: true, Scopes: []string{"vrooli-onboarding:read", "vrooli-onboarding:write"}})
	makeRequest := func(value string) *connect.Request[operatorstatev1.PatchOperatorStateRequest] {
		enabled := value == "true"
		return connect.NewRequest(&operatorstatev1.PatchOperatorStateRequest{State: &operatorstatev1.OperatorState{Scenarios: map[string]*operatorstatev1.ScenarioChoice{"demo": {Enabled: &enabled}}}, UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"scenarios.demo.enabled"}}, ExpectedRevision: operatorstate.Revision(initial), RequestId: "stable-update-1"})
	}
	first, err := handler.PatchOperatorState(ctx, makeRequest("true"))
	if err != nil {
		t.Fatalf("first patch: %v", err)
	}
	if first.Msg.GetReceipt() == nil || first.Msg.GetReceipt().GetRequestId() != "stable-update-1" || first.Msg.GetReceipt().GetExpectedRevision() != operatorstate.Revision(initial) || first.Msg.GetReceipt().GetResultRevision() == operatorstate.Revision(initial) {
		t.Fatalf("receipt=%+v", first.Msg.GetReceipt())
	}
	replayed, err := handler.PatchOperatorState(ctx, makeRequest("true"))
	if err != nil {
		t.Fatalf("exact retry: %v", err)
	}
	if !proto.Equal(first.Msg.GetReceipt(), replayed.Msg.GetReceipt()) {
		t.Fatalf("retry receipt=%+v, original=%+v", replayed.Msg.GetReceipt(), first.Msg.GetReceipt())
	}
	if _, err := handler.PatchOperatorState(ctx, makeRequest("false")); connect.CodeOf(err) != connect.CodeAlreadyExists {
		t.Fatalf("changed payload request reuse error=%v code=%s", err, connect.CodeOf(err))
	}
	currentState, err := store.Load(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if currentState.Scenarios["demo"].Enabled == nil || !*currentState.Scenarios["demo"].Enabled || len(currentState.PermissionReceipts) != 1 {
		t.Fatalf("stored state after replay/conflict=%+v", currentState)
	}
	if _, err := os.Stat(filepath.Join(stateRoot, operatorstate.StateFile)); err != nil {
		t.Fatal(err)
	}
}

func boolPointer(value bool) *bool { return &value }
