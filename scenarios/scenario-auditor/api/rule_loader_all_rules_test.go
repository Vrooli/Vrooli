package main

import "testing"

func TestAllRuleImplementationsLoad(t *testing.T) {
	t.Setenv("VROOLI_ROOT", projectRootDir(t))

	rules, err := LoadRulesFromFiles()
	if err != nil {
		t.Fatalf("LoadRulesFromFiles failed: %v", err)
	}

	for id, info := range rules {
		if !info.Implementation.Valid {
			t.Fatalf("rule %s implementation invalid: %s", id, info.Implementation.Error)
		}
	}
}
