package main

import (
	"path/filepath"
	"testing"
)

// [REQ:ONB-SELECT-CLOSURE]
func TestV2ClosureEvidenceIncludesDiamondDependenciesAndTryStartProvenance(t *testing.T) {
	root := t.TempDir()
	writeFixtureFile(t, filepath.Join(root, "scenarios", "root", ".vrooli", "service.json"), `{
  "service":{"name":"root"},
  "dependencies":{"scenarios":{"left":{"required":true},"right":{"required":true}},"resources":{"database":{"required":true}}}
}`)
	writeFixtureFile(t, filepath.Join(root, "scenarios", "left", ".vrooli", "service.json"), `{
  "service":{"name":"left"},
  "dependencies":{"scenarios":{"shared":{"required":true}},"resources":{"cache":{"startup_policy":"try_start"}}}
}`)
	writeFixtureFile(t, filepath.Join(root, "scenarios", "right", ".vrooli", "service.json"), `{
  "service":{"name":"right"},
  "dependencies":{"scenarios":{"shared":{"required":true}}}
}`)
	writeFixtureFile(t, filepath.Join(root, "scenarios", "shared", ".vrooli", "service.json"), `{"service":{"name":"shared"}}`)

	result, err := resolveClosureForState(root, []ScenarioReadModel{{Name: "root", Enabled: true}}, OperatorState{})
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string]closureMember{}
	for _, member := range result.Scenarios {
		byName[member.Name] = member
	}
	shared := byName["shared"]
	if shared.State != "included" || len(shared.Provenance) != 2 {
		t.Fatalf("shared diamond member = %#v", shared)
	}
	if shared.Provenance[0].From != "left" || shared.Provenance[1].From != "right" {
		t.Fatalf("shared provenance = %#v", shared.Provenance)
	}

	resources := map[string]closureMember{}
	for _, member := range result.Resources {
		resources[member.Name] = member
	}
	if got := resources["database"]; !got.Required || got.Policy != "must_start" || got.State != "included" {
		t.Fatalf("required resource = %#v", got)
	}
	if got := resources["cache"]; got.Required || got.Policy != "try_start" || got.State != "included" {
		t.Fatalf("try-start resource = %#v", got)
	}
}

// [REQ:ONB-SELECT-CLOSURE]
func TestV2ClosureEvidenceRejectsCyclesBeforeReturningAClosure(t *testing.T) {
	root := t.TempDir()
	writeFixtureFile(t, filepath.Join(root, "scenarios", "first", ".vrooli", "service.json"), `{"service":{"name":"first"},"dependencies":{"scenarios":{"second":{"required":true}}}}`)
	writeFixtureFile(t, filepath.Join(root, "scenarios", "second", ".vrooli", "service.json"), `{"service":{"name":"second"},"dependencies":{"scenarios":{"first":{"required":true}}}}`)

	result, err := resolveClosureForState(root, []ScenarioReadModel{{Name: "first", Enabled: true}}, OperatorState{})
	if err == nil || result.Scenarios != nil {
		t.Fatalf("cycle result = %#v, err = %v; want no partial closure", result, err)
	}
}
