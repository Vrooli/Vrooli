package main

import "testing"

func TestUILifecycleLaunchImplementationLoads(t *testing.T) {
	t.Setenv("VROOLI_ROOT", projectRootDir(t))

	rules, err := LoadRulesFromFiles()
	if err != nil {
		t.Fatalf("LoadRulesFromFiles failed: %v", err)
	}
	rule, ok := rules["ui_lifecycle_launch"]
	if !ok {
		t.Fatalf("ui_lifecycle_launch rule not loaded")
	}

	if !rule.Implementation.Valid {
		t.Fatalf("expected ui_lifecycle_launch implementation to be valid, got error: %s", rule.Implementation.Error)
	}
}
