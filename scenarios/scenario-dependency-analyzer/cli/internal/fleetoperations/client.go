package fleetoperations

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/signal"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/analysis_operations"
	rpc "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/analysis_operations/analysis_operations_v1connect"
	"google.golang.org/protobuf/encoding/protojson"

	"scenario-dependency-analyzer/cli/internal/support"
)

func Client(core *cliapp.ScenarioApp) rpc.AnalysisOperationsServiceClient {
	httpClient, base := cliapp.NewConnectHTTPClientWithTimeout(core, 0)
	return rpc.NewAnalysisOperationsServiceClient(httpClient, base)
}

// StartAndObserve emits the replay key before admission, then attaches once.
// An interrupted observer can reuse that key, or wait on the returned receipt.
func StartAndObserve(core *cliapp.ScenarioApp, kind pb.AnalysisOperationKind, apply bool, callerID string, wait, jsonOutput bool) error {
	if callerID == "" {
		var random [16]byte
		if _, err := rand.Read(random[:]); err != nil {
			return err
		}
		callerID = hex.EncodeToString(random[:])
	}
	fmt.Fprintln(os.Stderr, "Client operation ID:", callerID)
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	client := Client(core)
	admissionCtx, release := context.WithTimeout(ctx, 45*time.Second)
	defer release()
	response, err := client.StartOperation(admissionCtx, connect.NewRequest(&pb.StartOperationRequest{ClientOperationId: callerID, Kind: kind, ApplyGraph: apply}))
	if err != nil {
		return fmt.Errorf("start failed; reuse --client-operation-id %s to recover the same request: %w", callerID, err)
	}
	op := response.Msg.GetOperation()
	if op == nil || op.GetId() == "" {
		return fmt.Errorf("server returned no operation receipt; recover with --client-operation-id %s", callerID)
	}
	fmt.Fprintf(os.Stderr, "Operation: %s (reused=%t)\n", op.GetId(), response.Msg.GetReused())
	if wait {
		result, err := client.WaitOperation(ctx, connect.NewRequest(&pb.WaitOperationRequest{OperationId: op.GetId()}))
		if err != nil {
			return fmt.Errorf("observer ended; reattach with scenario-dependency-analyzer analysis wait %s: %w", op.GetId(), err)
		}
		op = result.Msg.GetOperation()
	}
	return Print(op, jsonOutput, wait)
}

func Observe(core *cliapp.ScenarioApp, id string, wait, jsonOutput bool) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	client := Client(core)
	var op *pb.AnalysisOperation
	if wait {
		response, err := client.WaitOperation(ctx, connect.NewRequest(&pb.WaitOperationRequest{OperationId: id}))
		if err != nil {
			return err
		}
		op = response.Msg.GetOperation()
	} else {
		response, err := client.GetOperation(ctx, connect.NewRequest(&pb.GetOperationRequest{OperationId: id}))
		if err != nil {
			return err
		}
		op = response.Msg.GetOperation()
	}
	return Print(op, jsonOutput, wait)
}

func Print(op *pb.AnalysisOperation, jsonOutput, requireSuccess bool) error {
	if op == nil {
		return fmt.Errorf("server returned no operation receipt")
	}
	if jsonOutput {
		body, err := protojson.MarshalOptions{Multiline: true, Indent: "  "}.Marshal(op)
		if err != nil {
			return err
		}
		if err := support.PrintAPIJSON(body); err != nil {
			return err
		}
	} else {
		status := []string{fmt.Sprintf("Operation: %s", op.GetId()), fmt.Sprintf("State: %s", op.GetState()), fmt.Sprintf("Scenarios: %d/%d complete; %d failed", op.GetScenariosCompleted(), op.GetScenariosTotal(), op.GetScenariosFailed())}
		if op.GetError() != "" {
			status = append(status, op.GetError())
		}
		if graph := op.GetGraph(); graph != nil {
			status = append(status, fmt.Sprintf("Graph: %d edges; applied=%t", graph.GetEdges(), graph.GetApplied()))
		}
		if err := support.PrintOperational(false, cliapp.OperationalReport{Status: status, NextSteps: []string{"scenario-dependency-analyzer analysis wait " + op.GetId(), "scenario-dependency-analyzer analysis get " + op.GetId() + " --json"}}, nil); err != nil {
			return err
		}
	}
	if requireSuccess && op.GetState() != pb.AnalysisOperationState_ANALYSIS_OPERATION_STATE_SUCCEEDED {
		return fmt.Errorf("operation %s ended %s: %s", op.GetId(), op.GetState(), op.GetError())
	}
	return nil
}
