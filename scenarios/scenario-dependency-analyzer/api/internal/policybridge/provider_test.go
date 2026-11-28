package policybridge

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestBuildSnapshotDeclaresRouteAndArgvEvidence(t *testing.T) {
	data, err := BuildSnapshot(time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got["provider_id"] != "scenario-dependency-analyzer" || got["evidence_state"] != "clean" {
		t.Fatalf("snapshot = %+v", got)
	}
	rules, ok := got["rules"].([]any)
	if !ok || len(rules) != 4 {
		t.Fatalf("rules = %#v", got["rules"])
	}
	for _, raw := range rules {
		rule, ok := raw.(map[string]any)
		if !ok {
			t.Fatalf("rule has unexpected shape: %#v", raw)
		}
		if rule["risk"] == "lifecycle_execution" && rule["action"] != "allow" {
			t.Errorf("lifecycle execution action = %v, want allow to preserve live behavior", rule["action"])
		}
	}
	for _, raw := range rules {
		rule, ok := raw.(map[string]any)
		if !ok || rule["evaluator_id"] == "" {
			t.Errorf("rule lacks owner-issued evaluator ID: %#v", raw)
		}
		id, _ := rule["evaluator_id"].(string)
		if ok && !strings.HasPrefix(id, "scenario-dependency-analyzer:") {
			t.Errorf("rule evaluator ID is not provider-owned: %#v", raw)
		}
	}
}

func TestReviewedPolicyEffectsDigest(t *testing.T) {
	data, err := BuildSnapshot(time.Unix(0, 0).UTC())
	if err != nil {
		t.Fatal(err)
	}
	digest, err := EffectsDigest(data)
	if err != nil {
		t.Fatal(err)
	}
	const reviewed = "eda876fae372acce0451e3d9c46c75ffe52b54c573cfd284110f82bfc0be650a"
	if digest != reviewed {
		t.Fatalf("policy effects digest = %s, want reviewed digest %s", digest, reviewed)
	}
}
