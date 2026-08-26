package components

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestPublishedSchemasAgreeWithContractStructSurface(t *testing.T) {
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller did not return the test source path")
	}
	readSchema := func(name string) map[string]any {
		raw, err := os.ReadFile(filepath.Join(filepath.Dir(sourceFile), "../../../../../.vrooli/schemas", name))
		if err != nil {
			t.Fatalf("read published schema %s: %v", name, err)
		}
		var schema map[string]any
		if err := json.Unmarshal(raw, &schema); err != nil {
			t.Fatalf("parse published schema %s: %v", name, err)
		}
		return schema
	}
	storySchema := readSchema("story-contract.schema.json")
	properties := storySchema["properties"].(map[string]any)
	for _, field := range []string{"schemaVersion", "kind", "args", "environment", "composition", "stories"} {
		if _, ok := properties[field]; !ok {
			t.Fatalf("story contract field %q is absent from published schema", field)
		}
	}
	if properties["schemaVersion"].(map[string]any)["const"] != float64(5) {
		t.Fatal("published story schema does not pin schemaVersion to 5")
	}
	defs := storySchema["$defs"].(map[string]any)
	for _, removed := range []string{"nodeValue", "iconValue", "handlerValue", "rowKeyValue", "columnsValue", "filtersValue"} {
		if _, ok := defs[removed]; ok {
			t.Fatalf("removed structured tag definition %q remains in the published schema", removed)
		}
	}
	story := defs["story"].(map[string]any)
	if _, ok := story["properties"].(map[string]any)["frame"]; ok {
		t.Fatal("story-level frame remains in the published schema")
	}
	manifestSchema := readSchema("component-manifest.schema.json")
	manifestProperties := manifestSchema["properties"].(map[string]any)
	if _, ok := manifestProperties["requiredTokens"]; ok {
		t.Fatal("derived requiredTokens must not be an authored manifest field")
	}
}

func TestParseStoryContractAcceptsV4Composition(t *testing.T) {
	contract, diagnostics := ParseStoryContract([]byte(`{
  "schemaVersion": 5,
  "kind": "component",
  "args": {"fields": []},
  "environment": {"fixtures": []},
  "stories": [{"id":"default","name":"Default","description":"A specimen.","composition":{"specimen":{"module":"./story.tsx","export":"DefaultStory"},"fixture":{"asset":"fixtures.data","version":"1.0.0"},"frame":{"asset":"navigation.page","version":"1.0.0","region":"content","fixture":"fixtures.data"}},"args":{}}]
}`))
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %v", diagnostics)
	}
	composition := contract.Stories[0].Composition
	if composition == nil || composition.Specimen.Export != "DefaultStory" || composition.Frame.Version != "1.0.0" {
		t.Fatalf("composition = %#v", composition)
	}
}

func TestParseStoryContractDefaultsOmittedStoryArgs(t *testing.T) {
	contract, diagnostics := ParseStoryContract([]byte(`{
  "schemaVersion": 5,
  "kind": "component",
  "stories": [{"id":"static","name":"Static"}]
}`))
	if len(diagnostics) != 0 || string(contract.Stories[0].Args) != `{}` {
		t.Fatalf("contract=%#v diagnostics=%v", contract, diagnostics)
	}
}

func TestParseStoryContractRejectsUnsupportedSchemaVersion(t *testing.T) {
	contract, diagnostics := ParseStoryContract([]byte(`{
  "schemaVersion": 3,
  "kind": "component",
  "stories": [{"id":"legacy","name":"Legacy","args":{}}]
}`))
	if contract == nil || len(diagnostics) != 1 || diagnostics[0].Rule != "supported_version" {
		t.Fatalf("contract=%#v diagnostics=%v", contract, diagnostics)
	}
	if !strings.Contains(diagnostics[0].Detail, "schemaVersion must be 5") {
		t.Fatalf("rejection does not identify the required schema version: %q", diagnostics[0].Detail)
	}
}

func TestStoryCoverageGapsAcceptsMatrixCovers(t *testing.T) {
	contract, diagnostics := ParseStoryContract([]byte(`{
  "schemaVersion": 5,
  "kind": "component",
  "args": {"fields": [{"path":"style","kind":"enum","options":["default","primary","secondary","danger","success","warning","muted","ghost"]}]},
  "environment": {"fixtures": []},
  "stories": [{"id":"styles","name":"Style matrix","role":"axis","axis":"style","covers":{"style":["default","primary","secondary","danger","success","warning","muted","ghost"]},"args":{}}]
}`))
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %v", diagnostics)
	}
	if gaps := StoryCoverageGaps(contract); len(gaps) != 0 {
		t.Fatalf("matrix covers left enum gaps: %v", gaps)
	}
}

func TestParseStoryContractRejectsLegacyFields(t *testing.T) {
	_, diagnostics := ParseStoryContract([]byte(`{
  "schemaVersion": 5,
  "kind": "component",
  "stories": [{"id":"legacy","name":"Legacy","harness":"Local","args":{}}]
}`))
	if len(diagnostics) != 1 || diagnostics[0].Rule != "valid_json" {
		t.Fatalf("diagnostics=%v", diagnostics)
	}
}

func TestParseStoryContractNamesRemovedStructuredTags(t *testing.T) {
	_, diagnostics := ParseStoryContract([]byte(`{
  "schemaVersion": 5,
  "kind": "component",
  "args": {"fields": [{"path":"getRowKey","kind":"structured","default":{"$rowKey":"id"}}]},
  "environment": {"fixtures": []},
  "stories": [{"id":"legacy","name":"Legacy","args":{"getRowKey":{"$rowKey":"id"}}}]
}`))
	if len(diagnostics) != 2 || diagnostics[0].Rule != "removed_structured_tag" || diagnostics[1].Rule != "removed_structured_tag" {
		t.Fatalf("diagnostics=%v", diagnostics)
	}
	if !strings.Contains(diagnostics[0].Detail, "$rowKey") || !strings.Contains(diagnostics[0].Detail, "story.tsx") {
		t.Fatalf("diagnostic does not name tag and replacement: %v", diagnostics)
	}
}

func TestParseStoryContractRejectsUnpinnedCompositionReferences(t *testing.T) {
	_, diagnostics := ParseStoryContract([]byte(`{
  "schemaVersion": 5,
  "kind": "component",
  "composition": {"specimen":{"module":"./other.tsx","export":"not-valid"},"fixture":{"asset":"fixture.data","version":"latest"}},
  "stories": [{"id":"default","name":"Default","args":{}}]
}`))
	if len(diagnostics) != 4 {
		t.Fatalf("diagnostics = %v", diagnostics)
	}
}

func TestParseStoryContractRejectsTwoCompositionRenderers(t *testing.T) {
	_, diagnostics := ParseStoryContract([]byte(`{
  "schemaVersion": 5,
  "kind": "component",
  "stories": [{"id":"invalid","name":"Invalid","composition":{"specimen":{"module":"./story.tsx","export":"Local"},"harness":{"asset":"preview.showcase","version":"1.0.0","export":"Showcase"}},"args":{}}]
}`))
	if len(StoryContractErrors(diagnostics)) != 1 || diagnostics[0].Rule != "exclusive_renderer" {
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
  "schemaVersion": 5,
  "kind": "component",
  "stories": [{"id":"primary","name":"Primary","composition":{"frame":{"asset":"navigation.page","version":"1.0.0","region":"missing","fixture":"fixtures.bad"}},"args":{}}]
}`))
	if len(parseDiagnostics) != 0 {
		t.Fatalf("unexpected parse diagnostics: %v", parseDiagnostics)
	}
	diagnostics := ValidateStoryFrames(contract, frameRegistry{
		"navigation.page": {ID: "navigation.page", Kind: "navigation", Targets: []string{"react-vite"}, Regions: []string{"navigation"}, Expects: []CatalogFramePort{{Capability: "data-source", TypeArguments: []string{"TRecord"}}}},
		"fixtures.bad":    {ID: "fixtures.bad", Kind: "fixture", FixtureSatisfies: &CatalogFramePort{Capability: "router-adapter", TypeArguments: []string{"Bad"}}},
	})
	if len(diagnostics) != 2 || diagnostics[0].Rule != "frame_fixture_data_source" || diagnostics[1].Rule != "frame_region_exists" {
		t.Fatalf("diagnostics = %v", diagnostics)
	}
}

func TestValidateStoryCompositionRejectsNonFixtureCatalogAsset(t *testing.T) {
	contract, parseDiagnostics := ParseStoryContract([]byte(`{
  "schemaVersion": 5,
  "kind": "component",
  "stories": [{"id":"primary","name":"Primary","composition":{"fixture":{"asset":"fixtures.data","version":"1.0.0"}},"args":{}}]
}`))
	if len(parseDiagnostics) != 0 {
		t.Fatalf("unexpected parse diagnostics: %v", parseDiagnostics)
	}
	diagnostics := ValidateStoryFrames(contract, frameRegistry{"fixtures.data": {ID: "fixtures.data", Kind: "component"}})
	if len(diagnostics) != 1 || diagnostics[0].Rule != "fixture_asset_kind" {
		t.Fatalf("diagnostics = %v", diagnostics)
	}
}
