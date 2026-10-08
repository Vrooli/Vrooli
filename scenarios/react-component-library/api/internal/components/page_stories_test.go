package components

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v5"
	"github.com/stretchr/testify/require"
)

func TestPublishedPageStorySchemaRequiresConcreteRoute(t *testing.T) {
	raw, err := os.ReadFile("../../../../../.vrooli/schemas/story-contract.schema.json")
	require.NoError(t, err)
	compiler := jsonschema.NewCompiler()
	require.NoError(t, compiler.AddResource("story.json", bytes.NewReader(raw)))
	schema, err := compiler.Compile("story.json")
	require.NoError(t, err)
	raw, err = os.ReadFile("../../../ui/src/pages/CoveragePage.story.json")
	require.NoError(t, err)
	var contract map[string]any
	require.NoError(t, json.Unmarshal(raw, &contract))
	require.NoError(t, schema.Validate(contract))
	delete(contract, "route")
	require.Error(t, schema.Validate(contract))
}

func TestPageStoryContractRequiresRouteAndAPIStateShape(t *testing.T) {
	valid := `{"schemaVersion":5,"kind":"page","route":"/coverage","environment":{"fixtures":[]},"stories":[{"id":"error","name":"Unavailable","role":"boundary","states":["error"],"apiState":[{"path":"/api/coverage","method":"GET","status":503,"body":{"message":"unavailable"}}],"expect":[{"kind":"text","value":"Coverage unavailable"}]}]}`
	contract, diagnostics := ParseStoryContract([]byte(valid))
	require.Empty(t, StoryContractErrors(diagnostics))
	require.Equal(t, StoryKindPage, contract.Kind)
	for _, route := range []string{"", "https://other.test/coverage", "//other.test", "/assets/:id"} {
		copy := *contract
		copy.Route = route
		require.NotEmpty(t, StoryContractErrors(ValidateStoryContract(&copy)), route)
	}
	copy := *contract
	copy.Args.Fields = []StoryField{{Path: "report", Kind: StoryFieldObject}}
	require.NotEmpty(t, StoryContractErrors(ValidateStoryContract(&copy)))
	contract.Stories[0].APIState[0].Status = 0
	require.NotEmpty(t, StoryContractErrors(ValidateStoryContract(contract)))
}

func TestPageStoryDiscoveryReadsSourceAndRejectsInvalidContracts(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "ui", "src", "pages")
	require.NoError(t, os.MkdirAll(dir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Example.tsx"), []byte("export function Example() { return null }"), 0644))
	raw := []byte(`{"schemaVersion":5,"kind":"page","route":"/example","environment":{"fixtures":[]},"stories":[{"id":"default","name":"Default","role":"anatomy","expect":[{"kind":"text","value":"Example"}]}]}`)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Example.story.json"), raw, 0644))
	page, err := LoadPageStory(context.Background(), root, "page:Example")
	require.NoError(t, err)
	require.Equal(t, "workspace", page.Projection.Version)
	require.Equal(t, StoryKindPage, page.Projection.Kind)
	require.Equal(t, filepath.Join(dir, "Example.story.json"), page.Projection.SourcePath)
	require.NotEmpty(t, page.Revision)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "shared.ts"), []byte("export const color = 'blue'"), 0644))
	changed, err := LoadPageStory(context.Background(), root, "page:Example")
	require.NoError(t, err)
	require.NotEqual(t, page.Revision, changed.Revision)
	var invalid map[string]any
	require.NoError(t, json.Unmarshal(raw, &invalid))
	delete(invalid, "route")
	raw, err = json.Marshal(invalid)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Example.story.json"), raw, 0644))
	_, err = LoadPageStories(context.Background(), root)
	require.ErrorContains(t, err, "page route")
}

func TestPageStoryRouteOverrideRemainsConcreteAndPageOnly(t *testing.T) {
	contract, diagnostics := ParseStoryContract([]byte(`{"schemaVersion":5,"kind":"page","route":"/design","environment":{"fixtures":[]},"stories":[{"id":"detail","name":"Detail","route":"/design/demo/page","role":"anatomy","expect":[{"kind":"text","value":"Design"}]}]}`))
	require.Empty(t, StoryContractErrors(diagnostics))
	for _, route := range []string{"/design/:scenario", "//external.test", "https://external.test"} {
		contract.Stories[0].Route = route
		require.NotEmpty(t, StoryContractErrors(ValidateStoryContract(contract)), route)
	}
	contract.Stories[0].Route = "/design/demo/page"
	contract.Kind = "component"
	require.NotEmpty(t, StoryContractErrors(ValidateStoryContract(contract)))
}
