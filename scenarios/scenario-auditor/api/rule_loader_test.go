package main

import "testing"

func TestCompileGoRuleContentTypeHeaders(t *testing.T) {
	t.Setenv("VROOLI_ROOT", projectRootDir(t))

	rules, err := LoadRulesFromFiles()
	if err != nil {
		t.Fatalf("LoadRulesFromFiles failed: %v", err)
	}
	rule, ok := rules["content_type_headers"]
	if !ok {
		t.Fatalf("content_type_headers rule not loaded")
	}

	if !rule.Implementation.Valid {
		t.Fatalf("expected content_type_headers implementation to be valid, got error: %s", rule.Implementation.Error)
	}
}
