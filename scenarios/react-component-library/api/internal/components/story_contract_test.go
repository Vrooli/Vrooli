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

func TestParseStoryContractDefaultsOmittedStoryArgs(t *testing.T) {
	contract, diagnostics := ParseStoryContract([]byte(`{
  "schemaVersion": 3,
  "kind": "component",
  "stories": [{"id":"static","name":"Static"}]
}`))
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %v", diagnostics)
	}
	if got := string(contract.Stories[0].Args); got != `{}` {
		t.Fatalf("normalized args = %s", got)
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

func TestParseStoryContractValidatesOptionalPinnedFrameVersion(t *testing.T) {
	contract, diagnostics := ParseStoryContract([]byte(`{
  "schemaVersion": 3,
  "kind": "component",
  "args": {"fields": []},
  "environment": {"fixtures": []},
  "frame": {"asset":"navigation.page","version":"1.2.0","region":"navigation","fixture":"fixtures.user-directory"},
  "stories": [{"id":"primary","name":"Primary","args":{}}]
}`))
	if len(diagnostics) != 0 || contract.Frame.Version != "1.2.0" {
		t.Fatalf("contract=%#v diagnostics=%v", contract, diagnostics)
	}

	_, diagnostics = ParseStoryContract([]byte(`{
  "schemaVersion": 3,
  "kind": "component",
  "args": {"fields": []},
  "environment": {"fixtures": []},
  "frame": {"asset":"navigation.page","version":"latest","region":"navigation","fixture":"fixtures.user-directory"},
  "stories": [{"id":"primary","name":"Primary","args":{}}]
}`))
	if len(diagnostics) != 1 || diagnostics[0].Rule != "frame_version" {
		t.Fatalf("diagnostics=%v", diagnostics)
	}
}

func TestParseStoryContractSupportsVersionedSharedHarness(t *testing.T) {
	contract, diagnostics := ParseStoryContract([]byte(`{
  "schemaVersion": 3,
  "kind": "component",
  "args": {"fields": []},
  "environment": {"fixtures": []},
  "stories": [{"id":"default","name":"Default","sharedHarness":{"asset":"preview.showcase","version":"1.0.0","export":"Showcase","config":{"title":"Example"}},"args":{}}]
}`))
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %v", diagnostics)
	}
	if got := contract.Stories[0].SharedHarness; got == nil || got.Asset != "preview.showcase" || got.Export != "Showcase" {
		t.Fatalf("shared harness = %#v", got)
	}
}

func TestParseStoryContractSchemaVersionFourResolvesExplicitCompositionRoles(t *testing.T) {
	contract, diagnostics := ParseStoryContract([]byte(`{
  "schemaVersion": 4,
  "kind": "component",
  "composition": {
    "specimen": {"module":"./story.tsx","export":"MetricCardStory"},
    "fixture": {"asset":"fixtures.resource-collection","version":"1.0.0","state":"ready"},
    "frame": {"asset":"navigation.page","version":"1.0.0","region":"content","fixture":"fixtures.user-directory"}
  },
  "stories": [{"id":"metric-card","name":"Metric card","args":{}}]
}`))
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %v", diagnostics)
	}
	composition := EffectiveStoryComposition(contract, &contract.Stories[0])
	if composition == nil || composition.Specimen.Export != "MetricCardStory" || composition.Fixture.Version != "1.0.0" {
		t.Fatalf("composition = %#v", composition)
	}
	if got := EffectiveStoryLocalHarness(contract, &contract.Stories[0]); got != "MetricCardStory" {
		t.Fatalf("local harness = %q", got)
	}
}

func TestParseStoryContractRejectsUnpinnedCompositionReferences(t *testing.T) {
	_, diagnostics := ParseStoryContract([]byte(`{
  "schemaVersion": 4,
  "kind": "component",
  "composition": {
    "specimen": {"module":"./other.tsx","export":"not-valid"},
    "fixture": {"asset":"fixture.data","version":"latest"}
  },
  "stories": [{"id":"default","name":"Default","args":{}}]
}`))
	if len(diagnostics) != 4 {
		t.Fatalf("diagnostics = %v", diagnostics)
	}
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity != StoryDiagnosticError {
			t.Fatalf("diagnostic severity = %#v", diagnostic)
		}
	}
}

func TestParseStoryContractWarnsForLegacyRawNodeWithoutBlocking(t *testing.T) {
	contract, diagnostics := ParseStoryContract([]byte(`{
  "schemaVersion": 1,
  "kind": "component",
  "args": {"fields": [{"path":"children","kind":"structured"}]},
  "environment": {"fixtures": []},
  "stories": [{"id":"legacy","name":"Legacy","args":{"children":{"$node":{"tag":"div"}}}}]
}`))
	if contract == nil || len(diagnostics) != 1 {
		t.Fatalf("contract=%#v diagnostics=%v", contract, diagnostics)
	}
	if diagnostics[0].Severity != StoryDiagnosticWarning || diagnostics[0].Rule != "legacy_raw_node" {
		t.Fatalf("diagnostics = %#v", diagnostics)
	}
}

func TestParseStoryContractRejectsLocalAndSharedHarnessTogether(t *testing.T) {
	_, diagnostics := ParseStoryContract([]byte(`{
  "schemaVersion": 3,
  "kind": "component",
  "args": {"fields": []},
  "environment": {"fixtures": []},
  "stories": [{"id":"invalid","name":"Invalid","harness":"Local","sharedHarness":{"asset":"preview.showcase","version":"1.0.0","export":"Showcase"},"args":{}}]
}`))
	if len(diagnostics) != 1 || diagnostics[0].Rule != "exclusive_harness" {
		t.Fatalf("diagnostics = %v", diagnostics)
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

func TestValidateStoryCompositionRejectsNonFixtureCatalogAsset(t *testing.T) {
	contract, parseDiagnostics := ParseStoryContract([]byte(`{
  "schemaVersion": 4,
  "kind": "component",
  "composition": {"fixture":{"asset":"fixtures.data","version":"1.0.0"}},
  "stories": [{"id":"primary","name":"Primary","args":{}}]
}`))
	if len(parseDiagnostics) != 0 {
		t.Fatalf("unexpected parse diagnostics: %v", parseDiagnostics)
	}
	diagnostics := ValidateStoryFrames(contract, frameRegistry{
		"fixtures.data": {ID: "fixtures.data", Kind: "component"},
	})
	if len(diagnostics) != 1 || diagnostics[0].Rule != "fixture_asset_kind" {
		t.Fatalf("diagnostics = %v", diagnostics)
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

func TestParseStoryContractSupportsCountExpectations(t *testing.T) {
	contract, diagnostics := ParseStoryContract([]byte(`{
  "schemaVersion": 3,
  "kind": "component",
  "args": {"fields": []},
  "environment": {"fixtures": []},
  "stories": [{"id":"single","name":"Single subject","args":{},"expect":[{"kind":"count","selector":"button","value":"1"}]}]
}`))
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %v", diagnostics)
	}
	if expectation := contract.Stories[0].Expect[0]; expectation.Kind != "count" || expectation.Selector != "button" || expectation.Value != "1" {
		t.Fatalf("count expectation = %#v", expectation)
	}
}

func TestParseStoryContractSupportsEvidenceReviewSets(t *testing.T) {
	contract, diagnostics := ParseStoryContract([]byte(`{
  "schemaVersion": 3,
  "kind": "component",
  "args": {"fields": []},
  "environment": {"fixtures": []},
  "stories": [{"id":"loading","name":"Loading","args":{},"evidence":{"reviewSet":"core","states":["loading","mobile"]}}]
}`))
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %v", diagnostics)
	}
	if evidence := contract.Stories[0].Evidence; evidence == nil || evidence.ReviewSet != "core" || len(evidence.States) != 2 {
		t.Fatalf("evidence = %#v", evidence)
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
