package catalogcoverage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestShapeCensusCountsOnlyLiveVersionFileSets(t *testing.T) {
	root := t.TempDir()
	for _, dir := range []string{
		"components/Button/versions/1.0.0",
		"components/Button/versions/1.0.1",
		".retired/old/versions/1.0.0",
	} {
		require.NoError(t, os.MkdirAll(filepath.Join(root, dir), 0o755))
	}
	require.NoError(t, os.WriteFile(filepath.Join(root, "components/Button/versions/1.0.0/Button.tsx"), nil, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "components/Button/versions/1.0.1/Button.tsx"), nil, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "components/Button/versions/1.0.1/story.tsx"), nil, 0o644))
	rows, err := ShapeCensus(root)
	require.NoError(t, err)
	require.Len(t, rows, 2)
	counts := []int{rows[0].Count, rows[1].Count}
	require.ElementsMatch(t, []int{1, 1}, counts)
}

func TestShapeCensusReportsLegacySupportLayouts(t *testing.T) {
	root := t.TempDir()
	flat := filepath.Join(root, "support", "code-block", "0.3.4")
	file := filepath.Join(root, "support", "inline-code", "0.3.5.tsx")
	require.NoError(t, os.MkdirAll(flat, 0o755))
	require.NoError(t, os.MkdirAll(filepath.Dir(file), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(flat, "useCodeCopy.ts"), nil, 0o644))
	require.NoError(t, os.WriteFile(file, nil, 0o644))
	rows, err := ShapeCensus(root)
	require.NoError(t, err)
	require.Contains(t, rows, ShapeRow{Kind: "support-flat", Shape: []string{"useCodeCopy.ts"}, Count: 1})
	require.Contains(t, rows, ShapeRow{Kind: "support-file", Shape: []string{"<Asset>/<version>.tsx"}, Count: 1})
}

func TestDuplicationCensusReportsOwnedMetadataDrift(t *testing.T) {
	root := t.TempDir()
	catalog := filepath.Join(root, "scenarios/react-component-library/catalog/assets/controls")
	manifest := filepath.Join(root, "scenarios/react-component-library/library/components/Button")
	require.NoError(t, os.MkdirAll(catalog, 0o755))
	require.NoError(t, os.MkdirAll(manifest, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(catalog, "button.json"), []byte(`{"asset":{"id":"controls.button","name":"Button","description":"desired","slot":"ui-primitive"}}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(manifest, "component.json"), []byte(`{"description":"observed","slot":"ui-primitive"}`), 0o644))
	rows, err := DuplicationCensus(root)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, "description", rows[0].Field)
}
