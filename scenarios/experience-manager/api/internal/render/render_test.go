package render

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"experience-manager/internal/spec"
)

func TestRenderPageIsByteStable(t *testing.T) { // [REQ:EXPERIEN-P1-001]
	page := fixturePage()
	first := RenderPage("demo", page)
	second := RenderPage("demo", page)
	require.Equal(t, first, second)
	require.Contains(t, first, "data-testid=primary-action")
	require.Contains(t, first, "Claim annotations")
}

func TestRenderWritesWireframeArtifact(t *testing.T) {
	dir := writeScenario(t)
	result, err := Render(Request{ScenarioDir: dir, PageID: "home", Mode: ModeWireframe})
	require.NoError(t, err)
	require.Equal(t, "demo", result.Scenario)
	require.Equal(t, ModeWireframe, result.Mode)
	require.FileExists(t, result.ArtifactPath)
	require.True(t, filepath.IsAbs(result.ArtifactPath) || filepath.Clean(result.ArtifactPath) == result.ArtifactPath)
	data, err := os.ReadFile(result.ArtifactPath)
	require.NoError(t, err)
	require.Equal(t, result.HTML, string(data))
}

func TestImageModeDegradesToWireframe(t *testing.T) { // [REQ:EXPERIEN-P1-001]
	dir := writeScenario(t)
	result, err := Render(Request{ScenarioDir: dir, PageID: "home", Mode: ModeImage})
	require.NoError(t, err)
	require.Equal(t, ModeWireframe, result.Mode)
	require.Contains(t, result.DegradedReason, "image-tools unavailable")
}

func TestCompareVariantsWritesStableArtifact(t *testing.T) { // [REQ:EXPERIEN-P1-001]
	dir := writeScenario(t)
	variants := []Variant{
		{ID: "compact", Title: "Compact", Page: fixturePage()},
		{ID: "evidence", Title: "Evidence Forward", Page: fixturePageWithTitle("Evidence Forward")},
	}
	first, err := Compare(CompareRequest{ScenarioDir: dir, PageID: "home", Mode: ModeWireframe, Variants: variants})
	require.NoError(t, err)
	second, err := Compare(CompareRequest{ScenarioDir: dir, PageID: "home", Mode: ModeWireframe, Variants: variants})
	require.NoError(t, err)
	require.Equal(t, first.HTML, second.HTML)
	require.Contains(t, first.HTML, "data-variant=\"compact\"")
	require.Contains(t, first.HTML, "Evidence Forward")
	require.FileExists(t, first.ArtifactPath)
}

func fixturePage() spec.PageDocument {
	return spec.PageDocument{
		Page: spec.PageIdentity{
			ID:      "home",
			Title:   "Home",
			Routes:  []string{"/"},
			Purpose: "Home page demonstrates deterministic wireframe rendering.",
		},
		Priorities: []spec.Priority{{Statement: "Primary action is visibly first."}},
		States:     []spec.State{{ID: "default", Description: "Default state."}},
		Elements: []spec.Element{
			{ID: "main", Role: "region", Name: "Main", Description: "Main content region."},
			{ID: "primary-action", Role: "button", Name: "Create", Description: "Primary create action."},
		},
		Claims: []spec.Claim{
			{ID: "primary-present", Type: "element-present", Statement: "The primary action is present.", Tier: "machine", Elements: []string{"primary-action"}, States: []string{"default"}},
		},
		Bindings: spec.Bindings{Elements: map[string]spec.Binding{
			"main":           {TestID: "main"},
			"primary-action": {TestID: "primary-action"},
		}},
		Sketch: spec.Sketch{Regions: []spec.SketchRegion{{ID: "main-region", Elements: []string{"main", "primary-action"}}}},
	}
}

func fixturePageWithTitle(title string) spec.PageDocument {
	page := fixturePage()
	page.Page.Title = title
	return page
}

func writeScenario(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "demo")
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "experience", "pages"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "experience", "index.json"), []byte(`{
  "kind": "experience-index",
  "contract": {"kind": "scenario-experience", "schema": "scenario-experience-spec/v1"},
  "schemaVersion": "1.0.0",
  "scenario": "demo",
  "description": "Demo experience contract for rendering tests.",
  "pages": [{"id": "home", "path": "pages/home.json", "title": "Home", "status": "draft"}],
  "journeys": []
}
`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "experience", "pages", "home.json"), []byte(`{
  "kind": "experience-page",
  "schemaVersion": "1.0.0",
  "page": {
    "id": "home",
    "title": "Home",
    "routes": ["/"],
    "purpose": "Home page demonstrates deterministic wireframe rendering."
  },
  "priorities": [{"statement": "Primary action is visibly first."}],
  "states": [{"id": "default", "description": "Default state."}],
  "elements": [
    {"id": "main", "role": "region", "name": "Main", "description": "Main content region."},
    {"id": "primary-action", "role": "button", "name": "Create", "description": "Primary create action."}
  ],
  "claims": [{
    "id": "primary-present",
    "type": "element-present",
    "statement": "The primary action is present.",
    "tier": "machine",
    "elements": ["primary-action"],
    "states": ["default"]
  }],
  "bindings": {"elements": {
    "main": {"testid": "main"},
    "primary-action": {"testid": "primary-action"}
  }},
  "sketch": {"regions": [{"id": "main-region", "elements": ["main", "primary-action"]}]}
}
`), 0o644))
	return dir
}
