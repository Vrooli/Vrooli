package gates

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDependencyRankUsesDeclaredFoundationKindDuringRootMigration(t *testing.T) {
	root := t.TempDir()
	library := filepath.Join(root, "scenarios", "react-component-library", "library")
	foundation := filepath.Join(library, "components", "GestureTokens")
	hook := filepath.Join(library, "hooks", "useDrag")
	foundationVersion := filepath.Join(foundation, "versions", "1.0.0")
	hookVersion := filepath.Join(hook, "versions", "1.0.0")
	require.NoError(t, os.MkdirAll(foundationVersion, 0o755))
	require.NoError(t, os.MkdirAll(hookVersion, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(foundation, "component.json"), []byte(`{"libraryId":"react-component-library:GestureTokens","assetKind":"foundation"}`), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(hook, "component.json"), []byte(`{"libraryId":"react-component-library:useDrag","assetKind":"hook"}`), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(foundationVersion, "GestureTokens.ts"), []byte(`export const resolveGestureFeel = () => ({})`), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(hookVersion, "useDrag.ts"), []byte(`import { resolveGestureFeel } from "@vrooli/react-component-library/GestureTokens/1"; export const useDrag = resolveGestureFeel;`), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(hookVersion, "dependencies.json"), []byte(`{"libraryId":"react-component-library:useDrag","version":"1.0.0","dependencies":[{"libraryId":"react-component-library:GestureTokens","version":"1.0.0"}]}`), 0o600))

	result, err := ValidateDependencyRank(Scope{Root: root})
	require.NoError(t, err)
	require.Empty(t, result.Findings)
}
