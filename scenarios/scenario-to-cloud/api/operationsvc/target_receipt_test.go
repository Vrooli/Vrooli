package operationsvc

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/execplan"
	"scenario-to-cloud/operations"

	operationsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/operations"
	"github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/operations/operationsv1connect"
)

type targetReceiptReaderFunc func(context.Context, string, string, string) (*domain.CloudOperation, operations.TargetReceipt, error)

func (f targetReceiptReaderFunc) ReadTargetReceipt(ctx context.Context, deploymentID, operationID, step string) (*domain.CloudOperation, operations.TargetReceipt, error) {
	return f(ctx, deploymentID, operationID, step)
}

type targetAuthorizerFunc func(context.Context, string) *apierrors.Error

func (f targetAuthorizerFunc) AuthorizeTarget(ctx context.Context, deploymentID string) *apierrors.Error {
	return f(ctx, deploymentID)
}

func TestGetTargetReceiptAuthorizesBeforeFreshRead(t *testing.T) {
	planBytes, err := json.Marshal(execplan.Plan{Target: execplan.Target{
		MachineID: "machine-1", NodeID: "node-1", EnrollmentGeneration: 7, Transport: "bridge",
	}})
	if err != nil {
		t.Fatal(err)
	}
	op := &domain.CloudOperation{ID: "op-1", DeploymentID: "dep-1", PlanDigest: "sha256:plan", Plan: planBytes}
	var order []string
	reader := targetReceiptReaderFunc(func(_ context.Context, deploymentID, operationID, step string) (*domain.CloudOperation, operations.TargetReceipt, error) {
		order = append(order, "read")
		if deploymentID != "dep-1" || operationID != "op-1" || step != "release.activate" {
			t.Fatalf("read identity = %s/%s/%s", deploymentID, operationID, step)
		}
		return op, operations.TargetReceipt{Found: true, Outcome: operations.StepSucceeded, Fence: 12}, nil
	})
	authorizer := targetAuthorizerFunc(func(_ context.Context, deploymentID string) *apierrors.Error {
		order = append(order, "authorize")
		if deploymentID != "dep-1" {
			t.Fatalf("authorized deployment = %s", deploymentID)
		}
		return nil
	})
	svc := &Service{targetReceipts: reader, targetAuthorizer: authorizer}
	resp, err := svc.GetTargetReceipt(context.Background(), connect.NewRequest(&operationsv1.GetTargetReceiptRequest{
		DeploymentId: "dep-1", OperationId: "op-1", Step: "release.activate",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if len(order) != 2 || order[0] != "authorize" || order[1] != "read" {
		t.Fatalf("target read order = %v", order)
	}
	got := resp.Msg
	if got.GetDeploymentId() != "dep-1" || got.GetOperationId() != "op-1" || got.GetPlanDigest() != "sha256:plan" || got.GetStep() != "release.activate" {
		t.Fatalf("receipt envelope = %+v", got)
	}
	if target := got.GetTarget(); target.GetMachineId() != "machine-1" || target.GetNodeId() != "node-1" || target.GetEnrollmentGeneration() != 7 || target.GetTransport() != "bridge" {
		t.Fatalf("target = %+v", target)
	}
	if receipt := got.GetReceipt(); !receipt.GetFound() || receipt.GetOutcome() != "succeeded" || receipt.GetFence() != 12 {
		t.Fatalf("receipt = %+v", receipt)
	}
}

func TestGetTargetReceiptRefusesBeforeReachWhenTargetGrantFails(t *testing.T) {
	read := false
	svc := &Service{
		targetAuthorizer: targetAuthorizerFunc(func(context.Context, string) *apierrors.Error {
			return apierrors.New(apierrors.CodeForbiddenTarget, "not allowed")
		}),
		targetReceipts: targetReceiptReaderFunc(func(context.Context, string, string, string) (*domain.CloudOperation, operations.TargetReceipt, error) {
			read = true
			return nil, operations.TargetReceipt{}, nil
		}),
	}
	_, err := svc.GetTargetReceipt(context.Background(), connect.NewRequest(&operationsv1.GetTargetReceiptRequest{
		DeploymentId: "dep-secret", OperationId: "op-1", Step: "release.activate",
	}))
	if err == nil || read {
		t.Fatalf("unauthorized receipt read reached target: err=%v read=%v", err, read)
	}
}

func TestConnectGetTargetReceiptReturnsFreshTargetIdentity(t *testing.T) {
	planBytes, err := json.Marshal(execplan.Plan{Target: execplan.Target{
		MachineID: "machine-vps-1", NodeID: "node-vps-1", EnrollmentGeneration: 12, Transport: "bridge",
	}})
	if err != nil {
		t.Fatal(err)
	}
	reads := 0
	svc := &Service{
		targetAuthorizer: targetAuthorizerFunc(func(_ context.Context, deploymentID string) *apierrors.Error {
			if deploymentID != "dep-stage" {
				t.Fatalf("authorized deployment = %s", deploymentID)
			}
			return nil
		}),
		targetReceipts: targetReceiptReaderFunc(func(_ context.Context, deploymentID, operationID, step string) (*domain.CloudOperation, operations.TargetReceipt, error) {
			reads++
			if deploymentID != "dep-stage" || operationID != "op-1" || step != "deploy.activate" {
				t.Fatalf("target read = %s/%s/%s", deploymentID, operationID, step)
			}
			return &domain.CloudOperation{ID: operationID, DeploymentID: deploymentID, PlanDigest: "sha256:plan", Plan: planBytes}, operations.TargetReceipt{
				Found: true, Outcome: operations.StepSucceeded, Fence: 19, Detail: "activation confirmed",
			}, nil
		}),
	}
	mux := http.NewServeMux()
	path, handler := svc.Handler()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	defer server.Close()
	client := operationsv1connect.NewOperationsServiceClient(server.Client(), server.URL)
	resp, err := client.GetTargetReceipt(context.Background(), connect.NewRequest(&operationsv1.GetTargetReceiptRequest{
		DeploymentId: "dep-stage", OperationId: "op-1", Step: "deploy.activate",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if reads != 1 || resp.Msg.GetPlanDigest() != "sha256:plan" || resp.Msg.GetTarget().GetNodeId() != "node-vps-1" || resp.Msg.GetTarget().GetEnrollmentGeneration() != 12 || resp.Msg.GetReceipt().GetFence() != 19 {
		t.Fatalf("wire response did not preserve fresh owner identity: reads=%d response=%+v", reads, resp.Msg)
	}
}
