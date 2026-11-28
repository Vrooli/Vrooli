package workflows

import (
	"reflect"
	"testing"
	"time"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestWorkflowKeyForKind(t *testing.T) {
	tests := []struct {
		kind Kind
		want string
	}{
		{KindExtract, ExtractWorkflowKey},
		{KindAdopt, AdoptWorkflowKey},
	}
	for _, tt := range tests {
		got, err := workflowKeyForKind(tt.kind)
		if err != nil || got != tt.want {
			t.Fatalf("workflowKeyForKind(%q) = %q, %v; want %q, nil", tt.kind, got, err, tt.want)
		}
	}
	if _, err := workflowKeyForKind("unexpected"); err == nil {
		t.Fatal("unknown workflow kind was accepted")
	}
}

func TestNewAgentManagerDispatcherUsesBoundedHTTPClient(t *testing.T) {
	dispatcher := NewAgentManagerDispatcher()
	if dispatcher.Client == nil {
		t.Fatal("dispatcher client is nil")
	}
	if dispatcher.Client.Timeout != agentManagerHTTPTimeout {
		t.Fatalf("dispatcher client timeout = %s, want %s", dispatcher.Client.Timeout, agentManagerHTTPTimeout)
	}
	if dispatcher.Client.Timeout <= 0 || dispatcher.Client.Timeout < time.Minute {
		t.Fatalf("dispatcher client timeout = %s, want a bounded long-poll-safe timeout", dispatcher.Client.Timeout)
	}
}

func TestWorkflowErrorIsEmptyForSuccessfulExecution(t *testing.T) {
	if got := workflowError(&domainpb.WorkflowExecution{Status: domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_SUCCEEDED}); got != "" {
		t.Fatalf("successful execution must not be projected as an error: %q", got)
	}
}

func TestProjectAssistantResultDoesNotProjectBlockedAssistantAsSuccess(t *testing.T) {
	output, err := structpb.NewValue(map[string]any{
		"result": map[string]any{
			"summary":  "validation unavailable",
			"outcome":  "blocked",
			"evidence": []any{"sandbox unavailable"},
		},
	})
	if err != nil {
		t.Fatalf("build output: %v", err)
	}
	status, projectedError := projectAssistantResult(&domainpb.WorkflowExecution{
		Status: domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_SUCCEEDED,
		Output: output,
	}, StatusSucceeded)
	if status != StatusFailed {
		t.Fatalf("blocked assistant outcome projected as %q, want failed", status)
	}
	if projectedError != "assistant outcome: blocked" {
		t.Fatalf("projected error = %q, want typed outcome", projectedError)
	}
}

func TestProjectAssistantResultClearsErrorForCompletedAssistant(t *testing.T) {
	output, err := structpb.NewValue(map[string]any{
		"result": map[string]any{"summary": "linked", "outcome": "completed"},
	})
	if err != nil {
		t.Fatalf("build output: %v", err)
	}
	status, projectedError := projectAssistantResult(&domainpb.WorkflowExecution{
		Status: domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_SUCCEEDED,
		Output: output,
	}, StatusSucceeded)
	if status != StatusSucceeded || projectedError != "" {
		t.Fatalf("completed assistant projected as status=%q error=%q", status, projectedError)
	}
}

func TestWorkflowInputMatchesEachDeclaredKindSchema(t *testing.T) {
	extract := workflowInput(StartInput{Kind: KindExtract, AssetID: "asset", SourceScenario: "source", SourcePath: "Card.tsx", TargetScenario: "must-not-send", ConfirmOverwrite: true})
	if want := map[string]any{"kind": "extract", "assetId": "asset", "sourceScenario": "source", "sourcePath": "Card.tsx", "requestedVersion": ""}; !reflect.DeepEqual(extract, want) {
		t.Fatalf("extract input = %#v, want %#v", extract, want)
	}
	adopt := workflowInput(StartInput{Kind: KindAdopt, AssetID: "asset", SourceScenario: "must-not-send", TargetScenario: "target", SourcePath: "target/Card.tsx", ConfirmOverwrite: true})
	if _, sent := adopt["sourceScenario"]; sent {
		t.Fatalf("adopt input leaked extract-only sourceScenario: %#v", adopt)
	}
	assets, ok := adopt["assets"].([]any)
	if !ok || len(assets) != 1 {
		t.Fatalf("adopt input assets = %#v, want one asset", adopt["assets"])
	}
	asset, ok := assets[0].(map[string]any)
	if !ok || asset["targetScenario"] != "target" || asset["confirmOverwrite"] != true || asset["assetId"] != "asset" {
		t.Fatalf("adopt input = %#v", adopt)
	}
}
