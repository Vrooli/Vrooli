package components

import "testing"

func TestParseStoryContractSchemaVersionTwoParsesHarnessAndDescription(t *testing.T) {
	contract, diagnostics := ParseStoryContract([]byte(`{
  "schemaVersion": 2,
  "kind": "component",
  "args": {"fields": []},
  "environment": {"fixtures": []},
  "stories": [{"id":"controlled","name":"Controlled","description":"Shows the selected value.","harness":"ControlledWithReadout","args":{}}]
}`))
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %v", diagnostics)
	}
	if got := contract.Stories[0]; got.Harness != "ControlledWithReadout" || got.Description != "Shows the selected value." {
		t.Fatalf("story fields = %#v", got)
	}
}

func TestParseStoryContractSchemaVersionThreeSupportsFileAndStoryFrames(t *testing.T) {
	contract, diagnostics := ParseStoryContract([]byte(`{
  "schemaVersion": 3,
  "kind": "component",
  "args": {"fields": []},
  "environment": {"fixtures": []},
  "frame": {"asset":"navigation.page","region":"navigation","fixture":"fixtures.user-directory"},
  "stories": [{"id":"primary","name":"Primary","frame":{"asset":"templates.collection-page","region":"collection","fixture":"fixtures.resource-collection"},"args":{}}]
}`))
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %v", diagnostics)
	}
	if got := EffectiveStoryFrame(contract, &contract.Stories[0]); got == nil || got.Asset != "templates.collection-page" {
		t.Fatalf("effective frame = %#v", got)
	}
}

type frameRegistry map[string]CatalogFrameAsset

func (r frameRegistry) LookupCatalogFrameAsset(id string) (CatalogFrameAsset, bool) {
	asset, ok := r[id]
	return asset, ok
}

func TestValidateStoryFramesReportsNamedCatalogDiagnostics(t *testing.T) {
	contract, parseDiagnostics := ParseStoryContract([]byte(`{
  "schemaVersion": 3,
  "kind": "component",
  "args": {"fields": []},
  "environment": {"fixtures": []},
  "frame": {"asset":"navigation.page","region":"missing","fixture":"fixtures.bad"},
  "stories": [{"id":"primary","name":"Primary","args":{}}]
}`))
	if len(parseDiagnostics) != 0 {
		t.Fatalf("unexpected parse diagnostics: %v", parseDiagnostics)
	}
	diagnostics := ValidateStoryFrames(contract, frameRegistry{
		"navigation.page": {ID: "navigation.page", Kind: "navigation", Targets: []string{"react-vite"}, Regions: []string{"navigation"}, Expects: []CatalogFramePort{{Capability: "data-source", TypeArguments: []string{"TRecord"}}}},
		"fixtures.bad":    {ID: "fixtures.bad", Kind: "fixture", FixtureSatisfies: &CatalogFramePort{Capability: "router-adapter", TypeArguments: []string{"Bad"}}},
	})
	if len(diagnostics) != 2 {
		t.Fatalf("diagnostics = %v", diagnostics)
	}
	if diagnostics[0].Rule != "frame_fixture_data_source" || diagnostics[1].Rule != "frame_region_exists" {
		t.Fatalf("diagnostic rules = %v", diagnostics)
	}
}

func TestSchemaVersionTwoStillParsesWithoutFrame(t *testing.T) {
	contract, diagnostics := ParseStoryContract([]byte(`{
  "schemaVersion": 2,
  "kind": "component",
  "args": {"fields": []},
  "environment": {"fixtures": []},
  "stories": [{"id":"primary","name":"Primary","args":{}}]
}`))
	if contract == nil || len(diagnostics) != 0 {
		t.Fatalf("contract=%#v diagnostics=%v", contract, diagnostics)
	}
}

func TestParseStoryContractRejectsInvalidHarnessExport(t *testing.T) {
	_, diagnostics := ParseStoryContract([]byte(`{
  "schemaVersion": 2,
  "kind": "component",
  "args": {"fields": []},
  "environment": {"fixtures": []},
  "stories": [{"id":"controlled","name":"Controlled","harness":"not-a-valid-export","args":{}}]
}`))
	if len(diagnostics) != 1 || diagnostics[0].Rule != "javascript_identifier" {
		t.Fatalf("diagnostics = %v", diagnostics)
	}
}

func TestParseStoryContractStillRejectsUnknownSchemaVersionTwoField(t *testing.T) {
	_, diagnostics := ParseStoryContract([]byte(`{
  "schemaVersion": 2,
  "kind": "component",
  "args": {"fields": []},
  "environment": {"fixtures": []},
  "stories": [{"id":"controlled","name":"Controlled","caption":"misspelled","args":{}}]
}`))
	if len(diagnostics) != 1 || diagnostics[0].Rule != "valid_json" {
		t.Fatalf("diagnostics = %v", diagnostics)
	}
}

func TestParseStoryContractValidatesOneAssetLevelSchema(t *testing.T) {
	contract, diagnostics := ParseStoryContract([]byte(`{
  "schemaVersion": 1,
  "kind": "component",
  "args": {"fields": [
    {"path":"tone","kind":"enum","required":true,"options":["success","warning"]},
    {"path":"count","kind":"number","default":1,"minimum":0}
  ]},
  "environment": {"fixtures":[{"key":"voiceInput","adapter":"voice-input","options":["idle"]}]},
  "stories":[{"id":"success","name":"Success","args":{"tone":"success"},"environment":{"voiceInput":"idle"},"expect":[{"kind":"text","value":"Ready"}]}]
}`))
	if contract == nil || len(diagnostics) != 0 {
		t.Fatalf("contract=%#v diagnostics=%v", contract, diagnostics)
	}
}

func TestParseStoryContractSupportsLayoutExpectations(t *testing.T) {
	contract, diagnostics := ParseStoryContract([]byte(`{
  "schemaVersion": 3,
  "kind": "component",
  "args": {"fields": []},
  "environment": {"fixtures": []},
  "stories": [{"id":"layout","name":"Layout","args":{},"expect":[{"kind":"layout","selector":"[data-shell]","minWidth":240,"minHeight":160,"noOverflow":true}]}]
}`))
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %v", diagnostics)
	}
	expectation := contract.Stories[0].Expect[0]
	if expectation.Kind != "layout" || expectation.MinWidth == nil || *expectation.MinWidth != 240 || !expectation.NoOverflow {
		t.Fatalf("layout expectation = %#v", expectation)
	}
}

func TestParseStoryContractRejectsUnsafeAndLegacyStyleInput(t *testing.T) {
	_, diagnostics := ParseStoryContract([]byte(`{
  "schemaVersion": 1,
  "kind": "component",
  "args": {"fields": [{"path":"__proto__.tone","kind":"select"}]},
  "environment": {"fixtures": []},
  "stories":[{"id":"bad","name":"Bad","args":{"__proto__":"pollution"},"controls":{}}]
}`))
	if len(diagnostics) == 0 {
		t.Fatal("expected diagnostics")
	}
}

func TestParseStoryContractRejectsUndeclaredFixtureAdapter(t *testing.T) {
	_, diagnostics := ParseStoryContract([]byte(`{
  "schemaVersion": 1,
  "kind": "hook",
  "args": {"fields": []},
  "environment": {"fixtures": [{"key":"state","adapter":"arbitrary-import","options":["idle"]}]},
  "stories": [{"id":"idle","name":"Idle","args":{},"environment":{"state":"idle"}}]
}`))
	if len(diagnostics) == 0 {
		t.Fatal("expected unsupported adapter diagnostic")
	}
}

func TestParseStoryContractRequiresSafeComponentInteractionLocator(t *testing.T) {
	_, diagnostics := ParseStoryContract([]byte(`{
  "schemaVersion": 1,
  "kind": "component",
  "args": {"fields": []},
  "environment": {"fixtures": []},
  "stories": [{"id":"open","name":"Open","args":{},"interactions":[{"kind":"click"}]}]
}`))
	if len(diagnostics) == 0 {
		t.Fatal("expected interaction target diagnostic")
	}
}

func TestStoryCoverageGapsNamesEveryUnstoriedEnumValue(t *testing.T) {
	contract, diagnostics := ParseStoryContract([]byte(`{
  "schemaVersion": 1,
  "kind": "component",
  "args": {"fields": [{"path":"state","kind":"enum","options":["idle","recording","error"]}]},
  "environment": {"fixtures": []},
  "stories": [
    {"id":"idle","name":"Idle","args":{"state":"idle"}},
    {"id":"recording","name":"Recording","args":{"state":"recording"}}
  ]
}`))
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %v", diagnostics)
	}
	gaps := StoryCoverageGaps(contract)
	if len(gaps) != 1 || gaps[0].Path != "state" || gaps[0].Value != `"error"` {
		t.Fatalf("coverage gaps = %#v", gaps)
	}
}
