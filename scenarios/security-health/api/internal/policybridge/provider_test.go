package policybridge

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"security-health/internal/validation"
)

func TestBuildSnapshotPreservesFindingStateAndProvenance(t *testing.T) {
	now := time.Now().UTC()
	data, err := BuildSnapshot(validation.Report{Scenario: "demo", PolicyMode: validation.RolloutGuarded, Findings: []validation.Finding{{RuleID: "osv.GHSA-demo"}}}, now)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got["provider_id"] != "security-health" || got["evidence_state"] != "finding" {
		t.Fatalf("snapshot = %+v", got)
	}
	if got["provenance"].(map[string]any)["finding_count"] != "1" {
		t.Fatalf("provenance = %+v", got["provenance"])
	}
	rules, ok := got["rules"].([]any)
	wantIDs := []string{"security-health:dependency-addition", "security-health:dependency-upgrade", "security-health:lifecycle-execution", "security-health:unknown-risk"}
	if !ok || len(rules) != len(wantIDs) {
		t.Fatalf("rules = %#v", got["rules"])
	}
	for i, raw := range rules {
		rule, ok := raw.(map[string]any)
		if !ok || rule["evaluator_id"] != wantIDs[i] || rule["action"] != "allow" {
			t.Errorf("rule[%d] = %#v, want stable ID %q and unchanged allow action", i, raw, wantIDs[i])
		}
	}
	if err := Publish(context.Background(), nil, validation.Report{}, now); err == nil {
		t.Fatal("nil sink must be rejected")
	}
}
