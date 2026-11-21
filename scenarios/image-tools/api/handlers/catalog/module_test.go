package catalog

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	"image-tools/internal/capability"

	catalogv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/catalog"
)

func TestGetOperationCatalogPublishesEveryOperation(t *testing.T) {
	resp, err := (&Handler{}).GetOperationCatalog(context.Background(),
		connect.NewRequest(&catalogv1.GetOperationCatalogRequest{}))
	if err != nil {
		t.Fatalf("GetOperationCatalog: %v", err)
	}
	if got := len(resp.Msg.GetOperations()); got != len(capability.All()) {
		t.Errorf("operations = %d, want %d", got, len(capability.All()))
	}
	if resp.Msg.GetContractVersion() != capability.ContractVersion {
		t.Errorf("contract version = %q, want %q", resp.Msg.GetContractVersion(), capability.ContractVersion)
	}
	if resp.Msg.GetScenario() != capability.ScenarioSlug {
		t.Errorf("scenario = %q, want %q", resp.Msg.GetScenario(), capability.ScenarioSlug)
	}
}

func TestGetOperationCatalogFiltersByFamily(t *testing.T) {
	resp, err := (&Handler{}).GetOperationCatalog(context.Background(),
		connect.NewRequest(&catalogv1.GetOperationCatalogRequest{Family: "restore"}))
	if err != nil {
		t.Fatalf("GetOperationCatalog: %v", err)
	}
	if len(resp.Msg.GetOperations()) == 0 {
		t.Fatal("restore filter returned no operations")
	}
	for _, op := range resp.Msg.GetOperations() {
		if op.GetFamily() != "restore" {
			t.Errorf("filter returned %q of family %q", op.GetId(), op.GetFamily())
		}
	}
}

func TestGetOperationCatalogRejectsUnknownFamily(t *testing.T) {
	_, err := (&Handler{}).GetOperationCatalog(context.Background(),
		connect.NewRequest(&catalogv1.GetOperationCatalogRequest{Family: "not-a-family"}))
	if err == nil {
		t.Fatal("expected an error for an unknown family")
	}
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Errorf("code = %v, want InvalidArgument", connect.CodeOf(err))
	}
}
