package gates

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"react-component-library/internal/components"
)

func TestValidateStoryArgsFieldsReportsOnlyHandAuthoredProjectionDrift(t *testing.T) {
	root := t.TempDir()
	library := filepath.Join(root, "scenarios", "react-component-library", "library", "components")
	writeAsset := func(name, manifest, story string) {
		t.Helper()
		versionDir := filepath.Join(library, name, "versions", "1.0.0")
		require.NoError(t, os.MkdirAll(versionDir, 0o700))
		require.NoError(t, os.WriteFile(filepath.Join(library, name, "component.json"), []byte(manifest), 0o600))
		if story != "" {
			require.NoError(t, os.WriteFile(filepath.Join(versionDir, "story.json"), []byte(story), 0o600))
		}
	}
	writeAsset("HandAuthored", `{"catalogId":"HandAuthored","latest":"1.0.0"}`, `{"schemaVersion":5,"kind":"component","args":{"fields":[{"path":"label","kind":"text"}]},"environment":{"fixtures":[]},"stories":[{"id":"default","name":"Default","args":{}}]}`)
	writeAsset("NoFields", `{"catalogId":"NoFields","latest":"1.0.0"}`, `{"schemaVersion":5,"kind":"component","args":{"fields":[]},"environment":{"fixtures":[]},"stories":[{"id":"default","name":"Default","args":{}}]}`)
	writeAsset("MissingStory", `{"catalogId":"MissingStory","latest":"1.0.0"}`, "")
	writeAsset("NoLatest", `{"catalogId":"NoLatest","latest":""}`, `{"schemaVersion":5,"kind":"component","args":{"fields":[{"path":"label","kind":"text"}]},"environment":{"fixtures":[]},"stories":[]}`)
	writeAsset("MalformedManifest", `{`, `{"schemaVersion":5,"kind":"component","args":{"fields":[{"path":"label","kind":"text"}]},"environment":{"fixtures":[]},"stories":[]}`)
	writeAsset("MalformedStory", `{"catalogId":"MalformedStory","latest":"1.0.0"}`, `{`)
	writeAsset("InvalidContract", `{"catalogId":"InvalidContract","latest":"1.0.0"}`, `{"schemaVersion":5,"kind":"component","args":{"fields":[{"path":"bad","kind":"unsupported"}]},"environment":{"fixtures":[]},"stories":[{"id":"default","name":"Default","args":{}}]}`)

	result, err := ValidateStoryArgsFields(Scope{Root: root})
	require.NoError(t, err)
	require.Equal(t, 2, result.Inspected, "valid stories with and without declared fields are inspected")
	require.Len(t, result.Findings, 1)
	require.Equal(t, "catalog.args_fields_hand_authored", result.Findings[0].Code)
	require.Equal(t, "HandAuthored", filepath.Base(filepath.Dir(filepath.Dir(filepath.Dir(result.Findings[0].File)))))
	require.Contains(t, result.Findings[0].Message, "declares 1 args.fields but source derivation produced 0")

	scoped, err := ValidateStoryArgsFields(Scope{Root: root, Assets: []string{"NoFields"}})
	require.NoError(t, err)
	require.Equal(t, 1, scoped.Inspected)
	require.Empty(t, scoped.Findings)
}

func TestStoryFieldsEquivalentIgnoresOrderButDetectsContractChanges(t *testing.T) {
	left := []components.StoryField{
		{Path: "tone", Kind: components.StoryFieldEnum, Options: []json.RawMessage{json.RawMessage(`"primary"`), json.RawMessage(`"quiet"`)}},
		{Path: "label", Kind: components.StoryFieldText, Required: true},
	}
	right := []components.StoryField{
		{Path: "label", Kind: components.StoryFieldText, Required: true},
		{Path: "tone", Kind: components.StoryFieldEnum, Options: []json.RawMessage{json.RawMessage(`"primary"`), json.RawMessage(`"quiet"`)}},
	}
	require.True(t, storyFieldsEquivalent(left, right))
	right[1].Options = []json.RawMessage{json.RawMessage(`"primary"`)}
	require.False(t, storyFieldsEquivalent(left, right))
	right[1].Options = []json.RawMessage{json.RawMessage(`"primary"`), json.RawMessage(`"quiet"`)}
	right[0].Required = false
	require.False(t, storyFieldsEquivalent(left, right))
}
