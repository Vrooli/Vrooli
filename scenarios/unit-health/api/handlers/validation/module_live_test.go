package validation

import (
	"context"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	repocontract "github.com/vrooli/repo-contract-go"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	commonconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/unit-health/v1/validation"
	validationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/unit-health/v1/validation/validation_v1connect"
)

// [REQ:UH-INT-001]
func TestModuleMountsNativeAndSharedConnectContracts(t *testing.T) {
	repoRoot, err := repocontract.ResolveRepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	mod := Module(log.Default(), repoRoot, nil)
	router := mux.NewRouter()
	mod.Mount(router)
	server := httptest.NewServer(router)
	defer server.Close()
	native := validationconnect.NewValidationServiceClient(http.DefaultClient, server.URL)
	calibration, err := native.RunCalibration(context.Background(), connect.NewRequest(&validationv1.RunCalibrationRequest{Partition: "inventory"}))
	if err != nil {
		t.Fatal(err)
	}
	if calibration.Msg.GetPartition() != "inventory" {
		t.Fatalf("calibration partition = %q", calibration.Msg.GetPartition())
	}
	shared := commonconnect.NewScenarioValidationServiceClient(http.DefaultClient, server.URL)
	describe, err := shared.DescribeProvider(context.Background(), connect.NewRequest(&commonv1.DescribeProviderRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if describe.Msg.GetProvider() == "" || describe.Msg.GetContract() == "" {
		t.Fatalf("provider descriptor = %+v", describe.Msg)
	}
	if Schema() != "" || len(Endpoints) < 3 || ProtoFile == nil || ScenarioValidationProtoFile == nil {
		t.Fatal("module metadata is incomplete")
	}
}
