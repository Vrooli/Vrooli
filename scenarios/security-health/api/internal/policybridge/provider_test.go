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
	if err := Publish(context.Background(), nil, validation.Report{}, now); err == nil {
		t.Fatal("nil sink must be rejected")
	}
}
