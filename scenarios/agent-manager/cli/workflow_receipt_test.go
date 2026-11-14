package main

import (
	"testing"
	"time"

	"agent-manager/internal/goalhome"
	scenariov1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestWorkflowReceiptReadsOwnerApplicabilityAndExactCaseOutcome(t *testing.T) {
	for _, tc := range []struct {
		name            string
		mutate          func(map[string]any)
		passed, current bool
	}{
		{"current passed case", func(map[string]any) {}, true, true},
		{"stale dependency", func(m map[string]any) {
			m["current_evidence"].(map[string]any)["scopes"].([]any)[0].(map[string]any)["state"] = "stale"
		}, true, false},
		{"historical receipt", func(m map[string]any) { delete(m, "current_evidence") }, true, false},
		{"failed browser", func(m map[string]any) {
			m["execution"].(map[string]any)["runs"].([]any)[0].(map[string]any)["success"] = false
		}, false, true},
		{"primary storage leak", func(m map[string]any) {
			m["execution"].(map[string]any)["isolation"].(map[string]any)["primary_root_writes_during_test_mode"] = 1
		}, false, true},
		{"test-root storage leak", func(m map[string]any) {
			m["execution"].(map[string]any)["isolation"].(map[string]any)["test_root_writes"] = 1
		}, false, true},
		{"missing isolation counters", func(m map[string]any) { delete(m["execution"].(map[string]any), "isolation") }, false, true},
		{"dry run", func(m map[string]any) {
			m["execution"].(map[string]any)["runs"].([]any)[0].(map[string]any)["dry_run"] = true
		}, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := map[string]any{"current_evidence": map[string]any{"scopes": []any{map[string]any{"cases": []any{"bas/cases/one.json"}, "state": "current", "reason": "unchanged"}}}, "execution": map[string]any{"isolation": map[string]any{"primary_during_test_mode_requests": 0, "primary_root_writes_during_test_mode": 0, "test_root_writes": 0}, "runs": []any{map[string]any{"asset_path": "bas/cases/one.json", "success": true}}}}
			tc.mutate(m)
			native, err := structpb.NewStruct(m)
			if err != nil {
				t.Fatal(err)
			}
			detail, err := anypb.New(native)
			if err != nil {
				t.Fatal(err)
			}
			run := &scenariov1.ValidationRun{RunId: "owner", Scenario: "demo", State: scenariov1.ValidationRunState_VALIDATION_RUN_STATE_SUCCEEDED, CompletedAt: timestamppb.New(time.Now()), TerminalResult: &scenariov1.ValidateScenarioResponse{NativeDetail: detail}}
			got, err := workflowJourneyReceipt(run)
			if err != nil || len(got.Cases) != 1 || got.Cases[0].Path != "bas/cases/one.json" || got.Cases[0].Passed != tc.passed || got.Cases[0].Current != tc.current {
				t.Fatalf("%+v err=%v", got, err)
			}
		})
	}
}

func TestEpochCheckDerivesGateFromReceiptInsteadOfStaleProse(t *testing.T) {
	text := "# E1\n- Scenario: demo\n## Exit gate\n- G1 journey: bas/cases/one.json @ workflow-health\n## Slice log\n2026-10-25T12:00:00Z | prior report | exit metric=partial | gate G1=fail\n2026-10-25T12:01:00Z | owner receipt | exit metric=linked | gate G1=receipt workflow-health:owner\n"
	file := writeFile(t, t.TempDir()+"/E1.md", text)
	app := &App{journeyReceiptReader: func(id string) (goalhome.JourneyReceipt, error) {
		return goalhome.JourneyReceipt{ID: id, Scenario: "demo", CompletedAt: time.Now(), Cases: []goalhome.JourneyCaseEvidence{{Path: "bas/cases/one.json", Passed: true, Current: true}}}, nil
	}}
	_, code := runEpochCheck(t, app, file, "--acceptance")
	if code != 0 {
		t.Fatalf("current receipt did not supersede stale manual result: %d", code)
	}
	app.journeyReceiptReader = func(id string) (goalhome.JourneyReceipt, error) {
		return goalhome.JourneyReceipt{ID: id, Scenario: "demo", CompletedAt: time.Now(), Cases: []goalhome.JourneyCaseEvidence{{Path: "bas/cases/one.json", Passed: true, Current: false, Reason: "shared dependency changed"}}}, nil
	}
	_, code = runEpochCheck(t, app, file, "--acceptance")
	if code != exitRefused {
		t.Fatalf("stale receipt accepted: %d", code)
	}
}
