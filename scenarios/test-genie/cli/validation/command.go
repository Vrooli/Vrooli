// Package validation exposes Test Genie's durable validation-receipt lifecycle.
package validation

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/validation"
	validationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/validation/validation_v1connect"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/durationpb"
)

func Register(manifest []byte, apiClient *cliutil.APIClient) (cliapp.SubcommandGroup, error) {
	client := func() (validationconnect.ValidationServiceClient, error) {
		return newClient(apiClient)
	}
	return register(manifest, client)
}

var newClient = func(apiClient *cliutil.APIClient) (validationconnect.ValidationServiceClient, error) {
	if apiClient == nil || strings.TrimSpace(apiClient.BaseURL()) == "" {
		return nil, errors.New("test-genie API base URL is not configured")
	}
	return validationconnect.NewValidationServiceClient(http.DefaultClient, strings.TrimRight(apiClient.BaseURL(), "/")), nil
}

func register(manifest []byte, client clientFactory) (cliapp.SubcommandGroup, error) {
	return cliapp.LoadFromManifestPrimitives(manifest, "validation", map[string]cliapp.PrimitiveHandler{
		"ValidationService.CreateValidation": cliapp.ProtoMutation(createCall(client), func(_ cliapp.OperationContext, response *validationv1.CreateValidationResponse) cliapp.MutationReport {
			return receiptMutationReport("created", response.GetReceipt())
		}),
		"ValidationService.GetValidation":   cliapp.ProtoList(getCall(client), receiptList),
		"ValidationService.WaitValidation":  cliapp.ProtoOperational(waitCall(client), waitReport),
		"ValidationService.ListValidations": cliapp.ProtoList(listCall(client), listReport),
		"ValidationService.CancelValidationWait": cliapp.ProtoMutation(cancelWaitCall(client), func(_ cliapp.OperationContext, response *validationv1.CancelValidationWaitResponse) cliapp.MutationReport {
			return cliapp.MutationReport{Result: []string{fmt.Sprintf("Wait detached: %t", response.GetCancelled())}, Changes: receiptLines(response.GetReceipt())}
		}),
		"ValidationService.AbortValidationWork": cliapp.ProtoMutation(abortCall(client), func(_ cliapp.OperationContext, response *validationv1.AbortValidationWorkResponse) cliapp.MutationReport {
			return receiptMutationReport("abort requested for", response.GetReceipt())
		}),
		"ValidationService.ExplainValidation": cliapp.ProtoOperational(explainCall(client), explainReport),
		"ValidationService.ListValidationShadows": cliapp.ProtoList(listShadowsCall(client), shadowListReport),
	})
}

type clientFactory func() (validationconnect.ValidationServiceClient, error)

func createCall(factory clientFactory) func(cliapp.OperationContext) (*validationv1.CreateValidationResponse, error) {
	return func(operation cliapp.OperationContext) (*validationv1.CreateValidationResponse, error) {
		payload, err := os.ReadFile(operation.Flag("intent-file"))
		if err != nil {
			return nil, fmt.Errorf("read validation intent: %w", err)
		}
		var intent validationv1.ValidationIntent
		if err := protojson.Unmarshal(payload, &intent); err != nil {
			return nil, fmt.Errorf("decode validation intent: %w", err)
		}
		client, err := factory()
		if err != nil {
			return nil, err
		}
		response, err := client.CreateValidation(context.Background(), connect.NewRequest(&validationv1.CreateValidationRequest{Intent: &intent}))
		if err != nil {
			return nil, err
		}
		return response.Msg, nil
	}
}

func getCall(factory clientFactory) func(cliapp.OperationContext) (*validationv1.GetValidationResponse, error) {
	return func(operation cliapp.OperationContext) (*validationv1.GetValidationResponse, error) {
		client, err := factory()
		if err != nil {
			return nil, err
		}
		response, err := client.GetValidation(context.Background(), connect.NewRequest(&validationv1.GetValidationRequest{ReceiptId: operation.Positional("receipt_id")}))
		if err != nil {
			return nil, err
		}
		return response.Msg, nil
	}
}

func waitCall(factory clientFactory) func(cliapp.OperationContext) (*validationv1.WaitValidationResponse, error) {
	return func(operation cliapp.OperationContext) (*validationv1.WaitValidationResponse, error) {
		timeout, err := time.ParseDuration(operation.Flag("timeout"))
		if err != nil || timeout <= 0 {
			return nil, fmt.Errorf("timeout must be a positive duration")
		}
		after, err := strconv.ParseUint(operation.Flag("after-revision"), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid after-revision: %w", err)
		}
		client, err := factory()
		if err != nil {
			return nil, err
		}
		response, err := client.WaitValidation(context.Background(), connect.NewRequest(&validationv1.WaitValidationRequest{ReceiptId: operation.Positional("receipt_id"), WaitId: operation.Flag("wait-id"), Timeout: durationpb.New(timeout), AfterRevision: after}))
		if err != nil {
			return nil, err
		}
		return response.Msg, nil
	}
}

func listCall(factory clientFactory) func(cliapp.OperationContext) (*validationv1.ListValidationsResponse, error) {
	return func(operation cliapp.OperationContext) (*validationv1.ListValidationsResponse, error) {
		pageSize, err := strconv.ParseUint(operation.Flag("page-size"), 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid page-size: %w", err)
		}
		client, err := factory()
		if err != nil {
			return nil, err
		}
		response, err := client.ListValidations(context.Background(), connect.NewRequest(&validationv1.ListValidationsRequest{CallerScenario: operation.Flag("caller"), CallerExecutionId: operation.Flag("execution"), PlanId: operation.Flag("plan"), PageSize: uint32(pageSize), PageToken: operation.Flag("page-token")}))
		if err != nil {
			return nil, err
		}
		return response.Msg, nil
	}
}

func cancelWaitCall(factory clientFactory) func(cliapp.OperationContext) (*validationv1.CancelValidationWaitResponse, error) {
	return func(operation cliapp.OperationContext) (*validationv1.CancelValidationWaitResponse, error) {
		client, err := factory()
		if err != nil {
			return nil, err
		}
		response, err := client.CancelValidationWait(context.Background(), connect.NewRequest(&validationv1.CancelValidationWaitRequest{ReceiptId: operation.Positional("receipt_id"), WaitId: operation.Flag("wait-id")}))
		if err != nil {
			return nil, err
		}
		return response.Msg, nil
	}
}

func abortCall(factory clientFactory) func(cliapp.OperationContext) (*validationv1.AbortValidationWorkResponse, error) {
	return func(operation cliapp.OperationContext) (*validationv1.AbortValidationWorkResponse, error) {
		client, err := factory()
		if err != nil {
			return nil, err
		}
		response, err := client.AbortValidationWork(context.Background(), connect.NewRequest(&validationv1.AbortValidationWorkRequest{ReceiptId: operation.Positional("receipt_id"), Reason: operation.Flag("reason"), RequestedBy: operation.Flag("requested-by")}))
		if err != nil {
			return nil, err
		}
		return response.Msg, nil
	}
}

func explainCall(factory clientFactory) func(cliapp.OperationContext) (*validationv1.ExplainValidationResponse, error) {
	return func(operation cliapp.OperationContext) (*validationv1.ExplainValidationResponse, error) {
		client, err := factory()
		if err != nil {
			return nil, err
		}
		response, err := client.ExplainValidation(context.Background(), connect.NewRequest(&validationv1.ExplainValidationRequest{ReceiptId: operation.Positional("receipt_id")}))
		if err != nil {
			return nil, err
		}
		return response.Msg, nil
	}
}

func listShadowsCall(factory clientFactory) func(cliapp.OperationContext) (*validationv1.ListValidationShadowsResponse, error) {
	return func(operation cliapp.OperationContext) (*validationv1.ListValidationShadowsResponse, error) {
		pageSize, err := strconv.ParseUint(operation.Flag("page-size"), 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid page-size: %w", err)
		}
		client, err := factory()
		if err != nil {
			return nil, err
		}
		response, err := client.ListValidationShadows(context.Background(), connect.NewRequest(&validationv1.ListValidationShadowsRequest{PageSize: uint32(pageSize)}))
		if err != nil {
			return nil, err
		}
		return response.Msg, nil
	}
}

func shadowListReport(_ cliapp.OperationContext, response *validationv1.ListValidationShadowsResponse) cliapp.ListReport {
	results := make([]string, 0, len(response.GetComparisons()))
	for _, item := range response.GetComparisons() {
		results = append(results, fmt.Sprintf("%s  %s:%s  matched=%t reason=%s", item.GetComparisonId(), item.GetSourceKind(), item.GetSourceId(), item.GetMatched(), item.GetReasonCode()))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("%d shadow comparison(s)", len(results))}, ResultsHeading: "Comparisons", Results: results, ListShaped: true, ResultCount: len(results)}
}

func receiptMutationReport(verb string, receipt *validationv1.ValidationReceipt) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Validation %s %s", verb, receipt.GetReceiptId())}, Changes: receiptLines(receipt), NextCommand: []string{fmt.Sprintf("test-genie validation wait --wait-id observer-1 %s", receipt.GetReceiptId())}}
}

func receiptList(_ cliapp.OperationContext, response *validationv1.GetValidationResponse) cliapp.ListReport {
	return cliapp.ListReport{Summary: receiptLines(response.GetReceipt())}
}

func listReport(_ cliapp.OperationContext, response *validationv1.ListValidationsResponse) cliapp.ListReport {
	results := make([]string, 0, len(response.GetReceipts()))
	for _, receipt := range response.GetReceipts() {
		results = append(results, fmt.Sprintf("%s  %s  lineage=%s", receipt.GetReceiptId(), receipt.GetState(), receipt.GetLineageId()))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("%d validation receipt(s)", len(results))}, ResultsHeading: "Receipts", Results: results, ListShaped: true, ResultCount: len(results)}
}

func waitReport(_ cliapp.OperationContext, response *validationv1.WaitValidationResponse) cliapp.OperationalReport {
	return cliapp.OperationalReport{Status: receiptLines(response.GetReceipt()), NextSteps: []string{fmt.Sprintf("timed_out=%t wait_cancelled=%t", response.GetTimedOut(), response.GetWaitCancelled())}}
}

func explainReport(_ cliapp.OperationContext, response *validationv1.ExplainValidationResponse) cliapp.OperationalReport {
	return cliapp.OperationalReport{Status: receiptLines(response.GetReceipt()), Triage: []cliapp.TriageGroup{{Heading: "Decisions", Items: response.GetDecisions()}}, NextSteps: response.GetNextActions()}
}

func receiptLines(receipt *validationv1.ValidationReceipt) []string {
	if receipt == nil {
		return []string{"receipt unavailable"}
	}
	return []string{fmt.Sprintf("receipt=%s", receipt.GetReceiptId()), fmt.Sprintf("lineage=%s", receipt.GetLineageId()), fmt.Sprintf("state=%s revision=%d", receipt.GetState(), receipt.GetRevision()), fmt.Sprintf("reason=%s", receipt.GetReasonCode())}
}
