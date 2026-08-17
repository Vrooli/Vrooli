package dispatch

import "testing"

func TestAllowUsesDerivedScopeBindings(t *testing.T) {
	if err := Allow(Job{Verb: "scenario test", Scenario: "demo"}, []string{"vrooli-bridge:write"}, DefaultManifest); err != nil {
		t.Fatalf("write scope should authorize scenario test: %v", err)
	}
	if err := Allow(Job{Verb: "scenario wait", Scenario: "demo"}, []string{"vrooli-bridge:write"}, DefaultManifest); err != nil {
		t.Fatalf("write scope should authorize scenario wait: %v", err)
	}
	if err := Allow(Job{Verb: "scenario status", Scenario: "demo"}, []string{"vrooli-bridge:read"}, DefaultManifest); err != nil {
		t.Fatalf("read scope should authorize scenario status: %v", err)
	}
	if err := Allow(Job{Verb: "scenario wait", Scenario: "demo"}, []string{"vrooli-bridge:read"}, DefaultManifest); err == nil {
		t.Fatal("read scope must not authorize a lifecycle wait")
	}
	if err := Allow(Job{Verb: "scenario test", Scenario: "demo"}, []string{"vrooli-bridge:read"}, DefaultManifest); err == nil {
		t.Fatal("read scope must not authorize a write verb")
	}
}

func TestAllowUsesProjectCatalogVocabulary(t *testing.T) {
	if err := Allow(Job{Verb: "scenario status"}, []string{"vrooli-bridge:read"}, DefaultManifest); err != nil {
		t.Fatalf("cataloged project read command should be admitted: %v", err)
	}
	if err := Allow(Job{Verb: "scenario stop-all"}, []string{"vrooli-bridge:read", "vrooli-bridge:write"}, DefaultManifest); err == nil {
		t.Fatal("cataloged destructive project command must require its destructive scope")
	}
	if err := Allow(Job{Verb: "scenario stop-all"}, []string{"vrooli-bridge:destructive"}, DefaultManifest); err != nil {
		t.Fatalf("destructive scope should admit cataloged destructive command: %v", err)
	}
}

func TestAllowProjectSetupBindingsUseTheExpectedEffects(t *testing.T) {
	if err := Allow(Job{Verb: "setup status"}, []string{"vrooli-bridge:read"}, DefaultManifest); err != nil {
		t.Fatalf("read scope should authorize project setup status: %v", err)
	}
	if err := Allow(Job{Verb: "setup"}, []string{"vrooli-bridge:read"}, DefaultManifest); err == nil {
		t.Fatal("read scope must not authorize project setup")
	}
	if err := Allow(Job{Verb: "setup"}, []string{"vrooli-bridge:write"}, DefaultManifest); err != nil {
		t.Fatalf("write scope should authorize project setup: %v", err)
	}
}

func TestAllowPreservesMinimouseCurrentVerbScopes(t *testing.T) {
	scopes := []string{
		"vrooli-bridge:write",
		"vrooli-bridge:session",
		"scenario status*",
		"scenario start*",
		"scenario wait*",
		"scenario logs*",
		"scenario test*",
	}
	for _, verb := range []string{"scenario status", "scenario start", "scenario wait", "scenario logs", "scenario test", "scenario stop"} {
		if err := Allow(Job{Verb: verb, Scenario: "web-search"}, scopes, DefaultManifest); err != nil {
			t.Fatalf("minimouse scope should continue to admit %q: %v", verb, err)
		}
	}
}

func TestAllowDeviceControlVerbsAreExplicitlyGoverned(t *testing.T) {
	if err := Allow(Job{Verb: "device-control observe", Scenario: "device-control"}, []string{"vrooli-bridge:read"}, DefaultManifest); err != nil {
		t.Fatalf("read scope should authorize observation: %v", err)
	}
	if err := Allow(Job{Verb: "device-control actuate", Scenario: "device-control"}, []string{"vrooli-bridge:read"}, DefaultManifest); err == nil {
		t.Fatal("read scope must not authorize device actuation")
	}
	if err := Allow(Job{Verb: "device-control actuate", Scenario: "device-control"}, []string{"vrooli-bridge:write"}, DefaultManifest); err != nil {
		t.Fatalf("write scope should authorize device actuation: %v", err)
	}
}
