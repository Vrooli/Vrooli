package discovery

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	discoveryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ecosystem-manager/v1/discovery"
)

// newHandler builds a handler with no assembler — enough to exercise the
// validation and static paths without shelling out to the discovery sweep.
func newHandler() *ConnectHandler { return NewConnectHandler(Deps{}) }

func TestGetResourceEmptyNameRejected(t *testing.T) {
	_, err := newHandler().GetResource(
		context.Background(),
		connect.NewRequest(&discoveryv1.GetResourceRequest{Name: ""}),
	)
	if err == nil {
		t.Fatal("GetResource with empty name should error")
	}
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("GetResource empty name code = %v, want InvalidArgument", connect.CodeOf(err))
	}
}

func TestGetScenarioEmptyNameRejected(t *testing.T) {
	_, err := newHandler().GetScenario(
		context.Background(),
		connect.NewRequest(&discoveryv1.GetScenarioRequest{Name: ""}),
	)
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("GetScenario empty name code = %v, want InvalidArgument", connect.CodeOf(err))
	}
}

func TestListCategoriesReturnsGroupings(t *testing.T) { // [REQ:EM-CONN-001]
	resp, err := newHandler().ListCategories(
		context.Background(),
		connect.NewRequest(&discoveryv1.ListCategoriesRequest{}),
	)
	if err != nil {
		t.Fatalf("ListCategories: %v", err)
	}
	if len(resp.Msg.GetResourceCategories()) == 0 || len(resp.Msg.GetScenarioCategories()) == 0 {
		t.Fatal("expected non-empty category groupings")
	}
}

func TestListOperationsNilAssembler(t *testing.T) {
	resp, err := newHandler().ListOperations(
		context.Background(),
		connect.NewRequest(&discoveryv1.ListOperationsRequest{}),
	)
	if err != nil {
		t.Fatalf("ListOperations: %v", err)
	}
	if len(resp.Msg.GetOperations()) != 0 {
		t.Fatalf("expected no operations with nil assembler, got %d", len(resp.Msg.GetOperations()))
	}
}
