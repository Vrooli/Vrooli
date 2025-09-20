package main

import "testing"

func TestBuildRuleBucketsRespectsDisabledStates(t *testing.T) {
	originalStore := ruleStateStore
	defer func() { ruleStateStore = originalStore }()

	ruleStateStore = &RuleStateStore{states: make(map[string]bool)}

	rules := map[string]RuleInfo{
		"enabled_rule": {
			ID:      "enabled_rule",
			Targets: []string{"api"},
			Enabled: true,
		},
		"disabled_rule": {
			ID:      "disabled_rule",
			Targets: []string{"api"},
			Enabled: true,
		},
		"metadata_disabled": {
			ID:      "metadata_disabled",
			Targets: []string{"api"},
			Enabled: false,
		},
	}

	if err := ruleStateStore.SetState("disabled_rule", false); err != nil {
		t.Fatalf("SetState returned error: %v", err)
	}

	buckets, active := buildRuleBuckets(rules, nil)

	if _, ok := active["disabled_rule"]; ok {
		t.Fatalf("expected disabled_rule to be filtered out when disabled via state store")
	}

	if _, ok := active["metadata_disabled"]; ok {
		t.Fatalf("expected metadata_disabled to be filtered out due to metadata flag")
	}

	apiBucket := buckets["api"]
	if len(apiBucket) != 1 {
		t.Fatalf("expected exactly one rule in api bucket, got %d", len(apiBucket))
	}
	if apiBucket[0].ID != "enabled_rule" {
		t.Fatalf("expected enabled_rule to remain active, got %s", apiBucket[0].ID)
	}

	_, targetedActive := buildRuleBuckets(rules, []string{"disabled_rule"})
	if len(targetedActive) != 0 {
		t.Fatalf("expected no active targeted rules when the requested rule is disabled, got %d", len(targetedActive))
	}
}

