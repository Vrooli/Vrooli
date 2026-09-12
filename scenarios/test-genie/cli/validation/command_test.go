package validation

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/validation"
	validationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/validation/validation_v1connect"
)

type fakeValidationClient struct {
	validationconnect.ValidationServiceClient
	calls map[string]int
}

func (f *fakeValidationClient) receipt() *validationv1.ValidationReceipt {
	return &validationv1.ValidationReceipt{ReceiptId: "receipt-1", LineageId: "lineage-1", State: validationv1.ReceiptState_RECEIPT_STATE_RUNNING, ReasonCode: validationv1.ValidationReasonCode_VALIDATION_REASON_CODE_NONE, Revision: 2}
}

func (f *fakeValidationClient) CreateValidation(context.Context, *connect.Request[validationv1.CreateValidationRequest]) (*connect.Response[validationv1.CreateValidationResponse], error) {
	f.calls["create"]++
	return connect.NewResponse(&validationv1.CreateValidationResponse{Receipt: f.receipt()}), nil
}

func (f *fakeValidationClient) GetValidation(context.Context, *connect.Request[validationv1.GetValidationRequest]) (*connect.Response[validationv1.GetValidationResponse], error) {
	f.calls["get"]++
	return connect.NewResponse(&validationv1.GetValidationResponse{Receipt: f.receipt()}), nil
}

func (f *fakeValidationClient) WaitValidation(context.Context, *connect.Request[validationv1.WaitValidationRequest]) (*connect.Response[validationv1.WaitValidationResponse], error) {
	f.calls["wait"]++
	return connect.NewResponse(&validationv1.WaitValidationResponse{Receipt: f.receipt()}), nil
}

func (f *fakeValidationClient) ListValidations(context.Context, *connect.Request[validationv1.ListValidationsRequest]) (*connect.Response[validationv1.ListValidationsResponse], error) {
	f.calls["list"]++
	return connect.NewResponse(&validationv1.ListValidationsResponse{Receipts: []*validationv1.ValidationReceipt{f.receipt()}}), nil
}

func (f *fakeValidationClient) CancelValidationWait(context.Context, *connect.Request[validationv1.CancelValidationWaitRequest]) (*connect.Response[validationv1.CancelValidationWaitResponse], error) {
	f.calls["cancel-wait"]++
	return connect.NewResponse(&validationv1.CancelValidationWaitResponse{Cancelled: true, Receipt: f.receipt()}), nil
}

func (f *fakeValidationClient) AbortValidationWork(context.Context, *connect.Request[validationv1.AbortValidationWorkRequest]) (*connect.Response[validationv1.AbortValidationWorkResponse], error) {
	f.calls["abort"]++
	return connect.NewResponse(&validationv1.AbortValidationWorkResponse{Receipt: f.receipt()}), nil
}

func (f *fakeValidationClient) ExplainValidation(context.Context, *connect.Request[validationv1.ExplainValidationRequest]) (*connect.Response[validationv1.ExplainValidationResponse], error) {
	f.calls["explain"]++
	return connect.NewResponse(&validationv1.ExplainValidationResponse{Receipt: f.receipt(), Decisions: []string{"attached"}, NextActions: []string{"wait"}}), nil
}

func (f *fakeValidationClient) ListValidationShadows(context.Context, *connect.Request[validationv1.ListValidationShadowsRequest]) (*connect.Response[validationv1.ListValidationShadowsResponse], error) {
	f.calls["shadows-list"]++
	return connect.NewResponse(&validationv1.ListValidationShadowsResponse{Comparisons: []*validationv1.ValidationShadowComparison{{ComparisonId: "shadow-1", Matched: true}}}), nil
}

func TestValidationCommandsCoverEveryLifecycleRPC(t *testing.T) {
	manifest, err := os.ReadFile(filepath.Join("..", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	fake := &fakeValidationClient{calls: map[string]int{}}
	group, err := register(manifest, func() (validationconnect.ValidationServiceClient, error) { return fake, nil })
	if err != nil {
		t.Fatal(err)
	}
	intentPath := filepath.Join(t.TempDir(), "intent.json")
	if err := os.WriteFile(intentPath, []byte(`{"schemaVersion":1}`), 0o644); err != nil {
		t.Fatal(err)
	}
	arguments := map[string][]string{
		"create":       {"--intent-file", intentPath},
		"get":          {"receipt-1"},
		"wait":         {"--wait-id", "observer", "--timeout", "1s", "receipt-1"},
		"list":         {},
		"cancel-wait":  {"--wait-id", "observer", "receipt-1"},
		"abort":        {"--reason", "operator", "--requested-by", "test", "receipt-1"},
		"explain":      {"receipt-1"},
		"shadows-list": {},
	}
	for _, command := range group.Subcommands {
		var stdout, stderr bytes.Buffer
		ctx, err := cliapp.NewTestRunContextFromArgs(command.Args, append(arguments[command.Name], "--json"), nil, &stdout, &stderr)
		if err != nil {
			t.Fatalf("parse %s: %v", command.Name, err)
		}
		if err := command.RunCtx(ctx); err != nil {
			t.Fatalf("run %s: %v", command.Name, err)
		}
		if stdout.Len() == 0 {
			t.Errorf("%s emitted no response", command.Name)
		}
	}
	for name := range arguments {
		if fake.calls[name] != 1 {
			t.Errorf("%s RPC calls = %d, want 1", name, fake.calls[name])
		}
	}
}
