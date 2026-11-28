package gates

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateDeclarationCoverageReportsMissingDeclarationsInScope(t *testing.T) {
	root := t.TempDir()
	write := func(path, contents string) {
		t.Helper()
		full := filepath.Join(root, filepath.FromSlash(path))
		require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
		require.NoError(t, os.WriteFile(full, []byte(contents), 0o644))
	}
	write("scenarios/react-component-library/catalog/assets/buttons/action.json", `{"asset":{"id":"catalog.action"}}`)
	write("scenarios/react-component-library/library/components/action/component.json", `{"catalogId":"catalog.action","libraryId":"action"}`)
	write("scenarios/react-component-library/library/components/legacy/component.json", `{"catalogId":"catalog.removed","libraryId":"legacy"}`)
	write("scenarios/react-component-library/library/hooks/anonymous/component.json", `{"libraryId":"anonymous"}`)

	t.Run("full corpus reports unknown and absent catalog IDs", func(t *testing.T) {
		result, err := ValidateDeclarationCoverage(Scope{Root: root})
		require.NoError(t, err)
		require.Equal(t, 3, result.Inspected)
		require.Len(t, result.Findings, 2)
		require.Contains(t, result.Findings[0].Message, `"anonymous"`)
		require.Contains(t, result.Findings[1].Message, `"catalog.removed"`)
		for _, finding := range result.Findings {
			require.Equal(t, "catalog.declaration_coverage", finding.Code)
			require.Equal(t, "__corpus__.declaration-coverage", finding.AssetID)
		}
	})

	t.Run("partial scope filters unrelated implementations and accepts a known declaration", func(t *testing.T) {
		result, err := ValidateDeclarationCoverage(Scope{Root: root, Assets: []string{"catalog.action"}})
		require.NoError(t, err)
		require.Equal(t, 1, result.Inspected)
		require.Empty(t, result.Findings)
	})
}

func TestValidateDeclarationCoverageRejectsMalformedManifest(t *testing.T) {
	root := t.TempDir()
	manifest := filepath.Join(root, "scenarios/react-component-library/library/components/broken/component.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(manifest), 0o755))
	require.NoError(t, os.WriteFile(manifest, []byte(`{"catalogId":`), 0o644))

	_, err := ValidateDeclarationCoverage(Scope{Root: root})
	require.Error(t, err)
	require.Contains(t, err.Error(), "parse "+manifest)
}
