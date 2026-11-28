package components

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func sourceRootWithComponents(t *testing.T, root string) string {
	t.Helper()
	componentsRoot := filepath.Join(root, "components")
	require.NoError(t, os.MkdirAll(componentsRoot, 0o700))
	return root
}

func TestDefaultSourceRootHonorsExplicitAndScenarioConfiguration(t *testing.T) {
	t.Run("absolute component source override", func(t *testing.T) {
		want := sourceRootWithComponents(t, t.TempDir())
		t.Setenv("COMPONENT_SOURCE_ROOT", "  "+want+"  ")
		t.Setenv("SCENARIO_PATH", "")
		got, err := DefaultSourceRoot()
		require.NoError(t, err)
		require.Equal(t, want, got)
	})

	t.Run("relative override resolves inside configured scenario", func(t *testing.T) {
		scenario := t.TempDir()
		want := sourceRootWithComponents(t, filepath.Join(scenario, "library"))
		t.Setenv("COMPONENT_SOURCE_ROOT", "library")
		t.Setenv("SCENARIO_PATH", scenario)
		got, err := DefaultSourceRoot()
		require.NoError(t, err)
		require.Equal(t, want, got)
	})

	t.Run("scenario default uses its library when present", func(t *testing.T) {
		scenario := t.TempDir()
		want := sourceRootWithComponents(t, filepath.Join(scenario, "library"))
		t.Setenv("COMPONENT_SOURCE_ROOT", "")
		t.Setenv("SCENARIO_PATH", scenario)
		got, err := DefaultSourceRoot()
		require.NoError(t, err)
		require.Equal(t, want, got)
	})

	t.Run("runtime default resolves repository library", func(t *testing.T) {
		t.Setenv("COMPONENT_SOURCE_ROOT", "")
		t.Setenv("SCENARIO_PATH", "")
		got, err := DefaultSourceRoot()
		require.NoError(t, err)
		require.DirExists(t, filepath.Join(got, "components"))
		require.True(t, filepath.IsAbs(got))
	})
}

func TestResolveConfiguredSourceRootUsesExistingRelativeDirectories(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err)
	tempRoot := sourceRootWithComponents(t, t.TempDir())
	relative, err := filepath.Rel(cwd, tempRoot)
	require.NoError(t, err)
	t.Setenv("SCENARIO_PATH", "")
	require.Equal(t, tempRoot, resolveConfiguredSourceRoot(relative))
	require.Equal(t, tempRoot, existingSourceRoot(relative))
	fallback, err := filepath.Abs(filepath.Join("..", "missing-components-root"))
	require.NoError(t, err)
	require.Equal(t, filepath.Clean(fallback), resolveConfiguredSourceRoot("missing-components-root"))
}

func TestModuleWithRootReturnsMountedComponentsModule(t *testing.T) {
	root := sourceRootWithComponents(t, t.TempDir())
	module := ModuleWithRoot(nil, nil, root, nil)
	require.Equal(t, "components", module.Name)
	require.NotEmpty(t, module.Endpoints)
}

func TestEnsureComponentsDirectoryRejectsFilesystemRootAsAnEscapingChild(t *testing.T) {
	require.Error(t, ensureComponentsDirectory(string(filepath.Separator)))
}
