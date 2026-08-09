package dispatch

import "testing"

func TestAllowUsesDerivedScopeBindings(t *testing.T) {
	if err := Allow(Job{Verb: "scenario test", Scenario: "demo"}, []string{"vrooli-bridge:write"}, DefaultManifest); err != nil {
		t.Fatalf("write scope should authorize scenario test: %v", err)
	}
	if err := Allow(Job{Verb: "scenario status", Scenario: "demo"}, []string{"vrooli-bridge:read"}, DefaultManifest); err != nil {
		t.Fatalf("read scope should authorize scenario status: %v", err)
	}
	if err := Allow(Job{Verb: "scenario test", Scenario: "demo"}, []string{"vrooli-bridge:read"}, DefaultManifest); err == nil {
		t.Fatal("read scope must not authorize a write verb")
	}
}
