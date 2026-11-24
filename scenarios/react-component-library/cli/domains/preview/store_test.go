package preview

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"
)

func TestFindPreviewAssetSourceNormalizesNamesAndPrefersComponents(t *testing.T) {
	root := t.TempDir()
	component := filepath.Join(root, "scenarios/react-component-library/library/components/Button/versions/1.0.0")
	hook := filepath.Join(root, "scenarios/react-component-library/library/hooks/Button/versions/1.0.0")
	require.NoError(t, os.MkdirAll(component, 0o755))
	require.NoError(t, os.MkdirAll(hook, 0o755))
	componentSource := filepath.Join(component, "source.tsx")
	hookSource := filepath.Join(hook, "source.tsx")
	require.NoError(t, os.WriteFile(componentSource, []byte("component"), 0o644))
	require.NoError(t, os.WriteFile(hookSource, []byte("hook"), 0o644))

	got, err := findPreviewAssetSource(root, "react-component-library:Button", " 1.0.0 ")
	require.NoError(t, err)
	require.Equal(t, componentSource, got)
}

func TestFindPreviewAssetSourceRejectsUnsafeOrMissingInputs(t *testing.T) {
	for _, tc := range []struct {
		id, version string
	}{
		{"", "1.0.0"}, {"Button", ""}, {"Button", "../secret"}, {"Button/../Other", "1.0.0"},
	} {
		_, err := findPreviewAssetSource(t.TempDir(), tc.id, tc.version)
		require.Error(t, err)
	}
	_, err := findPreviewAssetSource(t.TempDir(), "Button", "1.0.0")
	require.ErrorContains(t, err, "was not found")
}

func TestReadPreviewAssetDeclarations(t *testing.T) {
	path := filepath.Join(t.TempDir(), "asset.tsx")
	require.NoError(t, os.WriteFile(path, []byte("// no declaration\n"), 0o644))
	got, err := readPreviewAssetDeclarations(path)
	require.NoError(t, err)
	require.Empty(t, got)

	require.NoError(t, os.WriteFile(path, []byte("/**\n * @deps {\"zeta\":\"^2\",\"react\":\"^18\",\"skip\":\" \"}\n */\n"), 0o644))
	got, err = readPreviewAssetDeclarations(path)
	require.NoError(t, err)
	require.Equal(t, []previewStoreDeclaration{{Name: "react", Range: "^18"}, {Name: "zeta", Range: "^2"}}, got)

	require.NoError(t, os.WriteFile(path, []byte("* @deps {broken}\n"), 0o644))
	_, err = readPreviewAssetDeclarations(path)
	require.ErrorContains(t, err, "parse @deps")
	_, err = readPreviewAssetDeclarations(filepath.Join(t.TempDir(), "missing.tsx"))
	require.ErrorContains(t, err, "read preview asset")
}

func TestEnsurePreviewStoreSurfaceMergesDependenciesAndCreatesRequiredFiles(t *testing.T) {
	root := t.TempDir()
	declaration := previewStoreDeclaration{Name: "some-package", Range: "^3.1.0"}
	require.NoError(t, ensurePreviewStoreSurface(root, "preview-runtime", declaration))
	dir := filepath.Join(root, "scenarios/react-component-library/tools/preview-runtime")
	var manifest map[string]any
	raw, err := os.ReadFile(filepath.Join(dir, "package.json"))
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(raw, &manifest))
	require.Equal(t, "^3.1.0", manifest["dependencies"].(map[string]any)["some-package"])
	for _, name := range []string{"pnpm-lock.yaml", ".npmrc"} {
		_, err := os.Stat(filepath.Join(dir, name))
		require.NoError(t, err)
	}

	// Existing manifests and package-manager files are preserved while adding a dependency.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"custom","dependencies":{"existing":"1"}}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "pnpm-lock.yaml"), []byte("custom lock\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".npmrc"), []byte("custom config\n"), 0o644))
	require.NoError(t, ensurePreviewStoreSurface(root, "preview-runtime", previewStoreDeclaration{Name: "another", Range: "~1"}))
	raw, err = os.ReadFile(filepath.Join(dir, "package.json"))
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(raw, &manifest))
	require.Equal(t, "custom", manifest["name"])
	require.Equal(t, "1", manifest["dependencies"].(map[string]any)["existing"])
	require.Equal(t, "~1", manifest["dependencies"].(map[string]any)["another"])
	lock, err := os.ReadFile(filepath.Join(dir, "pnpm-lock.yaml"))
	require.NoError(t, err)
	require.Equal(t, "custom lock\n", string(lock))
	config, err := os.ReadFile(filepath.Join(dir, ".npmrc"))
	require.NoError(t, err)
	require.Equal(t, "custom config\n", string(config))
}

func TestEnsurePreviewStoreSurfaceRejectsMalformedManifest(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "scenarios/react-component-library/tools/preview-runtime")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "package.json"), []byte("{"), 0o644))
	require.ErrorContains(t, ensurePreviewStoreSurface(root, "preview-runtime", previewStoreDeclaration{Name: "x", Range: "1"}), "read governed preview surface manifest")
}

func TestPopulateStoreReportsNoDependenciesWithoutInstalling(t *testing.T) {
	core := cliapptest.NewTestApp(t, http.NewServeMux())
	ctx, output := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "component-id"}},
		Flags:       []cliapp.Flag{{Name: "version"}},
	}, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"component-id": "NotificationBadge"},
		Flags:       map[string]string{"version": "1.0.5"},
	})
	require.NoError(t, (&handlers{}).populateStore(ctx))
	require.Contains(t, output.String(), "No external runtime dependencies declared by NotificationBadge@1.0.5.")
}
