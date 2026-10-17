package readiness

import (
	"encoding/json"
	"testing"
)

func validTestItem(id string, requirement CleanRequirement, impact GlobalImpact, acceptance string) Item {
	return Item{
		ID: id, Title: id, Category: "test", Owner: "deployment-manager", Applicability: "all",
		CleanRequirement: requirement, GlobalImpact: impact,
		Freshness:          FreshnessPolicy{Basis: "candidate_identity"},
		Producer:           &ProducerRoute{Binding: "deployment-manager.test.read"},
		Remediation:        Remediation{Skill: "test", Topic: "fixture"},
		AcceptanceCriteria: acceptance,
	}
}

func TestDefaultChecklistIsValidAndCoversReleaseRiskGroups(t *testing.T) {
	checklist := DefaultChecklist()
	if err := checklist.Validate(); err != nil {
		t.Fatalf("default checklist invalid: %v", err)
	}
	categories := map[string]bool{}
	for _, item := range checklist.Items {
		categories[item.Category] = true
	}
	for _, category := range []string{"commercial", "storage_architecture", "migration_lineage", "test_integrity", "artifact_provenance", "dependency_governance", "security", "compatibility", "recovery", "observability", "performance_capacity", "platform_delivery", "operations"} {
		if !categories[category] {
			t.Errorf("checklist missing category %q", category)
		}
	}
}

func TestChecklistRejectsDuplicateAndUnknownVocabulary(t *testing.T) {
	checklist := Checklist{Version: ChecklistVersion, Items: []Item{validTestItem("one", Required, CapabilityGap, "criterion"), validTestItem("one", Advisory, AdvisoryImpact, "criterion")}}
	if err := checklist.Validate(); err == nil {
		t.Fatal("expected duplicate item refusal")
	}
	checklist.Items[1].ID = "two"
	checklist.Items[1].CleanRequirement = "maybe"
	if err := checklist.Validate(); err == nil {
		t.Fatal("expected unknown clean_requirement refusal")
	}
}

func TestBuiltInPolicyProjectionAndMetadata(t *testing.T) {
	policy := DefaultChecklist()
	if len(policy.Items) < 30 {
		t.Fatalf("policy has %d items, want retained checklist plus readiness groups", len(policy.Items))
	}
	if err := CheckProjection(BuiltInPolicyJSON()); err != nil {
		t.Fatalf("built-in projection drifted: %v", err)
	}
	for _, item := range policy.Items {
		if item.Owner == "" || item.Applicability == "" || item.Freshness.Basis == "" || item.Acceptance.Given == "" || item.Acceptance.When == "" || item.Acceptance.Then == "" || item.Remediation.Skill == "" {
			t.Fatalf("item %q has incomplete decision metadata: %+v", item.ID, item)
		}
	}
}

func TestProjectionRejectsDriftAndInvalidOwner(t *testing.T) {
	var policy Checklist
	if err := json.Unmarshal(BuiltInPolicyJSON(), &policy); err != nil {
		t.Fatal(err)
	}
	policy.Items[0].Title = "drifted"
	drifted, _ := json.Marshal(policy)
	if err := CheckProjection(drifted); err == nil {
		t.Fatal("expected projection drift refusal")
	}
	policy = DefaultChecklist()
	policy.Items[0].Owner = "unknown-owner"
	if err := policy.Validate(); err == nil {
		t.Fatal("expected unknown owner refusal")
	}
}
