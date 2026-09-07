package main

import (
	"reflect"
	"testing"

	auditorv1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/auditor"
)

func TestAuditorFixRequestCanonicalizesSubjectSelections(t *testing.T) {
	req, err := auditorFixRequestFromProto([]string{" beta ", "alpha", "alpha"}, []string{" rule-2 ", "rule-1"})
	if err != nil {
		t.Fatalf("auditorFixRequestFromProto() error = %v", err)
	}
	if got, want := req.ScenarioNames, []string{"alpha", "beta"}; !sameStrings(got, want) {
		t.Fatalf("scenario names = %#v, want %#v", got, want)
	}
	if got, want := req.RuleIDs, []string{"rule-1", "rule-2"}; !sameStrings(got, want) {
		t.Fatalf("rule ids = %#v, want %#v", got, want)
	}
	context, err := auditorFixSubjectContext(req)
	if err != nil {
		t.Fatalf("auditorFixSubjectContext() error = %v", err)
	}
	if want := `{"scenario_names":["alpha","beta"],"rule_ids":["rule-1","rule-2"]}`; context != want {
		t.Fatalf("subject context = %q, want %q", context, want)
	}
}

func TestAuditorFixRequestRejectsBlankSelections(t *testing.T) {
	if _, err := auditorFixRequestFromProto([]string{"  "}, []string{"rule"}); err == nil {
		t.Fatal("expected blank scenario selection to be rejected")
	}
	if _, err := auditorFixRequestFromProto([]string{"scenario"}, []string{"  "}); err == nil {
		t.Fatal("expected blank rule selection to be rejected")
	}
}

func TestAuditorReadResponsesPreserveTypedShape(t *testing.T) {
	status := auditorJobStatusToProto(&AuditorJobStatus{
		ID: "job-1", Scenario: "demo", ScanType: "full", Status: "completed",
		StartedAt: "2026-09-07T00:00:00Z", TotalScenarios: 2, ProcessedScenarios: 2,
		Result: &AuditorCheckResult{
			CheckID: "check-1", Status: "passed", Statistics: map[string]int{"violations": 2, "files": 4},
			Violations: []AuditorViolation{{ID: "v-1", ScenarioName: "demo", Metadata: map[string]any{"line": 12}}},
		},
	})
	if status.GetResult() == nil || !reflect.DeepEqual(status.GetResult().GetStatistics(), []*auditorv1.IntCount{
		{Key: "files", Count: 4}, {Key: "violations", Count: 2},
	}) {
		t.Fatalf("typed job statistics = %#v", status.GetResult().GetStatistics())
	}
	if got := status.GetResult().GetViolations()[0].GetMetadata()[0].GetValue(); got != "12" {
		t.Fatalf("typed metadata value = %q, want JSON scalar 12", got)
	}
}

func TestAuditorRulesResponseToProtoSortsRuleIds(t *testing.T) {
	response := auditorRulesResponseToProto(&AuditorRulesListResponse{Rules: map[string]AuditorRule{
		"z": {ID: "z", Name: "Z"},
		"a": {ID: "a", Name: "A"},
	}})
	if got, want := []string{response.GetRules()[0].GetId(), response.GetRules()[1].GetId()}, []string{"a", "z"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("typed rule order = %#v, want %#v", got, want)
	}
}

func sameStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
