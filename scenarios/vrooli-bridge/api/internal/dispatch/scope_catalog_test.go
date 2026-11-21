package dispatch

import "testing"

func TestAllowUsesDerivedScopeBindings(t *testing.T) {
	manifest, _, err := BuildManifest()
	if err != nil {
		t.Fatal(err)
	}
	if err := Allow(Job{Verb: "scenario test", Scenario: "demo"}, []string{"vrooli-bridge:write", "vrooli:write"}, manifest); err != nil {
		t.Fatalf("paired transport and namespace scopes should authorize scenario test: %v", err)
	}
	if err := Allow(Job{Verb: "scenario wait", Scenario: "demo"}, []string{"vrooli-bridge:read", "vrooli:read"}, manifest); err != nil {
		t.Fatalf("paired transport and namespace scopes should authorize scenario wait: %v", err)
	}
	if err := Allow(Job{Verb: "scenario status", Scenario: "demo"}, []string{"vrooli-bridge:read", "vrooli:read"}, manifest); err != nil {
		t.Fatalf("paired transport and namespace scopes should authorize scenario status: %v", err)
	}
	if err := Allow(Job{Verb: "scenario wait", Scenario: "demo"}, []string{"vrooli-bridge:write", "vrooli:read"}, manifest); err == nil {
		t.Fatal("write scope must not authorize a read-only lifecycle wait")
	}
	if err := Allow(Job{Verb: "scenario test", Scenario: "demo"}, []string{"vrooli-bridge:read", "vrooli:write"}, manifest); err == nil {
		t.Fatal("read scope must not authorize a write verb")
	}
}

func TestAllowUsesProjectCatalogVocabulary(t *testing.T) {
	manifest, _, err := BuildManifest()
	if err != nil {
		t.Fatal(err)
	}
	if err := Allow(Job{Verb: "scenario status"}, []string{"vrooli-bridge:read", "vrooli:read"}, manifest); err != nil {
		t.Fatalf("cataloged project read command should be admitted: %v", err)
	}
	if err := Allow(Job{Verb: "scenario start-all"}, []string{"vrooli-bridge:read", "vrooli:write"}, manifest); err == nil {
		t.Fatal("a write command must require its write effect scope")
	}
	if err := Allow(Job{Verb: "scenario start-all"}, []string{"vrooli-bridge:write", "vrooli:write"}, manifest); err != nil {
		t.Fatalf("write scope should admit cataloged project command: %v", err)
	}
}

func TestAllowProjectSetupBindingsUseTheExpectedEffects(t *testing.T) {
	manifest, _, err := BuildManifest()
	if err != nil {
		t.Fatal(err)
	}
	if err := Allow(Job{Verb: "scenario info"}, []string{"vrooli-bridge:read", "vrooli:read"}, manifest); err != nil {
		t.Fatalf("read scope should authorize project scenario info: %v", err)
	}
	if err := Allow(Job{Verb: "scenario setup"}, []string{"vrooli-bridge:read", "vrooli:write"}, manifest); err == nil {
		t.Fatal("read scope must not authorize project scenario setup")
	}
	if err := Allow(Job{Verb: "scenario setup"}, []string{"vrooli-bridge:write", "vrooli:write"}, manifest); err != nil {
		t.Fatalf("write scope should authorize project scenario setup: %v", err)
	}
}

func TestAllowDeviceControlVerbsAreExplicitlyGoverned(t *testing.T) {
	manifest, _, err := BuildManifest()
	if err != nil {
		t.Fatal(err)
	}
	if err := Allow(Job{Verb: "device-control device list", Scenario: "device-control"}, []string{"vrooli-bridge:read", "device-control:read"}, manifest); err != nil {
		t.Fatalf("read scope should authorize device inventory: %v", err)
	}
	if err := Allow(Job{Verb: "device-control device actuate", Scenario: "device-control"}, []string{"vrooli-bridge:read", "device-control:write"}, manifest); err == nil {
		t.Fatal("read scope must not authorize device actuation")
	}
	if err := Allow(Job{Verb: "device-control device actuate", Scenario: "device-control"}, []string{"vrooli-bridge:write", "device-control:write"}, manifest); err != nil {
		t.Fatalf("write scope should authorize device actuation: %v", err)
	}
}
