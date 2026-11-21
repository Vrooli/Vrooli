package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"agent-manager/internal/goalhome"
	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	scenariov1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	providerconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
	"google.golang.org/protobuf/types/known/structpb"
)

func readWorkflowJourneyReceipt(id string) (goalhome.JourneyReceipt, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	base, err := discovery.ResolveScenarioURLDefault(ctx, "workflow-health")
	if err != nil {
		return goalhome.JourneyReceipt{}, err
	}
	client := providerconnect.NewDurableValidationRunServiceClient(http.DefaultClient, base)
	response, err := client.GetValidationRun(ctx, connect.NewRequest(&scenariov1.GetValidationRunRequest{RunId: id}))
	if err != nil {
		return goalhome.JourneyReceipt{}, err
	}
	return workflowJourneyReceipt(response.Msg.GetRun())
}

func workflowJourneyReceipt(run *scenariov1.ValidationRun) (goalhome.JourneyReceipt, error) {
	if run.GetState() != scenariov1.ValidationRunState_VALIDATION_RUN_STATE_SUCCEEDED || run.GetTerminalResult() == nil || run.GetCompletedAt() == nil {
		return goalhome.JourneyReceipt{}, fmt.Errorf("workflow run is not terminal with a retained result")
	}
	var native structpb.Struct
	if err := run.GetTerminalResult().GetNativeDetail().UnmarshalTo(&native); err != nil {
		return goalhome.JourneyReceipt{}, fmt.Errorf("owner native evidence unavailable: %w", err)
	}
	fields := native.AsMap()
	current, _ := fields["current_evidence"].(map[string]any)
	states := map[string]goalhome.JourneyCaseEvidence{}
	for _, raw := range anySlice(current["scopes"]) {
		scope, _ := raw.(map[string]any)
		for _, rawName := range anySlice(scope["cases"]) {
			name, _ := rawName.(string)
			_, dup := states[name]
			states[name] = goalhome.JourneyCaseEvidence{Path: name, Current: scope["state"] == "current" && !dup, Reason: fmt.Sprint(scope["reason"])}
			if dup {
				item := states[name]
				item.Reason = "ambiguous owner evidence scope"
				states[name] = item
			}
		}
	}
	execution, _ := fields["execution"].(map[string]any)
	isolation, _ := execution["isolation"].(map[string]any)
	leaked := false
	for _, key := range []string{"primary_during_test_mode_requests", "primary_root_writes_during_test_mode", "test_root_writes"} {
		value, known := isolation[key].(float64)
		if !known || value != 0 {
			leaked = true
		}
	}
	for _, key := range []string{"install_error", "heartbeat_error", "clear_error"} {
		if value, ok := isolation[key].(string); ok && strings.TrimSpace(value) != "" {
			leaked = true
		}
	}
	result := goalhome.JourneyReceipt{ID: run.GetRunId(), Scenario: run.GetScenario(), CompletedAt: run.GetCompletedAt().AsTime()}
	seen := map[string]bool{}
	for _, raw := range anySlice(execution["runs"]) {
		item, _ := raw.(map[string]any)
		name, _ := item["asset_path"].(string)
		evidence, ok := states[name]
		if !ok {
			evidence = goalhome.JourneyCaseEvidence{Path: name, Reason: "no qualified dependency snapshot; historical evidence remains unverified"}
		}
		evidence.Passed = item["success"] == true && item["skipped"] != true && item["refused"] != true && item["dry_run"] != true && !leaked && !seen[name]
		if !evidence.Passed {
			evidence.Reason = "owner case failed, was skipped/dry-run, duplicated, or lacks clean isolation"
		}
		seen[name] = true
		result.Cases = append(result.Cases, evidence)
	}
	return result, nil
}

func anySlice(value any) []any { items, _ := value.([]any); return items }
