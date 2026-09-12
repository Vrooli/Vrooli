package registry

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverMaturityTargetsIncludesDescriptorAndCapabilityTargets(t *testing.T) {
	root := t.TempDir()
	for _, scenario := range []string{"descriptor-only", "capability-only", "registered-only", "unrelated"} {
		if err := os.MkdirAll(filepath.Join(root, "scenarios", scenario, ".vrooli"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "scenarios", "descriptor-only", ".vrooli", "search.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "scenarios", "capability-only", ".vrooli", "service.json"), []byte(`{"capabilities":["search"]}`), 0o644); err != nil {
		t.Fatal(err)
	}

	targets, err := DiscoverMaturityTargets(root)
	if err != nil {
		t.Fatalf("DiscoverMaturityTargets: %v", err)
	}
	if len(targets) != 2 || targets[0].Scenario != "capability-only" || targets[1].Scenario != "descriptor-only" {
		t.Fatalf("targets = %+v, want sorted descriptor/capability target set", targets)
	}
	if targets[0].ApplicabilityReason != "capability:search" || targets[1].ApplicabilityReason != "descriptor" {
		t.Fatalf("applicability reasons = %+v", targets)
	}
	merged := MergeRegisteredMaturityTargets(targets, []string{"registered-only", "descriptor-only"}, root)
	if len(merged) != 3 || merged[2].Scenario != "registered-only" || merged[2].ApplicabilityReason != "registered-provider" {
		t.Fatalf("merged targets = %+v, want registered target union", merged)
	}
}
