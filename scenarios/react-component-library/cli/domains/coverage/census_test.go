package coverage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildCensusCountsStoryCompositionAndSourceStyling(t *testing.T) {
	root := t.TempDir()
	versionDir := filepath.Join(root, "scenarios", "react-component-library", "library", "components", "Button", "versions", "1.0.0")
	require.NoError(t, os.MkdirAll(versionDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(versionDir, "story.json"), []byte(`{
  "args": {"fields": [{"default": {"$node": "button", "children": [{"$text": "Create"}]}}]},
  "stories": [
    {"id": "specimen", "composition": {"specimen": "x"}, "args": {"label": {"$text": "Create"}}},
    {"id": "plain", "args": {"label": {"$text": "button"}}}
  ]
}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(versionDir, "Button.tsx"), []byte(`export function Button() { return <button style={{ color: "red" }} /> }`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(versionDir, "story.tsx"), []byte(`export function Story() { return <button style={{ color: "red" }} /> }`), 0o644))

	report, err := buildCensus(root)
	require.NoError(t, err)
	require.Equal(t, 1, report.StoryContracts.CorruptCount)
	require.Equal(t, 1, report.StoryContracts.CorruptFiles)
	require.Equal(t, 1, report.Composition["specimen"])
	require.Equal(t, 1, report.Composition["none"])
	require.Equal(t, 1, report.Versions)
	require.Equal(t, 1, report.Styling.InlineStyleFiles)
	require.Equal(t, 0, report.Styling.MergeFiles)
}

func TestContainsAnyTestIDOnlyMatchesLibraryValues(t *testing.T) {
	ids := map[string]struct{}{"sidebar-shell": {}}
	require.True(t, containsAnyTestID(`const selector = "sidebar-shell"`, ids))
	require.False(t, containsAnyTestID(`const selector = "other-shell"`, ids))
}
