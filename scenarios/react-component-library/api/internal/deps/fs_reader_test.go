package deps

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFSPackageJSONReader_PrefersUIManifestAndFallsBackToRoot(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "ui-scenario", "ui"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "ui-scenario", "ui", "package.json"), []byte(`{"name":"ui"}`), 0o600))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "root-scenario"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "root-scenario", "package.json"), []byte(`{"name":"root"}`), 0o600))
	reader := NewFSPackageJSONReader(root)
	ui, err := reader.Read(context.Background(), "ui-scenario")
	require.NoError(t, err)
	require.JSONEq(t, `{"name":"ui"}`, string(ui))
	rootManifest, err := reader.Read(context.Background(), "root-scenario")
	require.NoError(t, err)
	require.JSONEq(t, `{"name":"root"}`, string(rootManifest))
}

// TestFSPackageJSONReader_ResolvesTemplateScenarioKey covers the template
// adoption key form "../templates/scenarios/<id>": it must resolve next to
// the scenarios root (so reapply against a vendored template copy can run
// dependency validation) while still rejecting traversal inside the id.
func TestFSPackageJSONReader_ResolvesTemplateScenarioKey(t *testing.T) {
	repoRoot := t.TempDir()
	scenariosRoot := filepath.Join(repoRoot, "scenarios")
	templateUI := filepath.Join(repoRoot, "templates", "scenarios", "react-vite", "ui")
	require.NoError(t, os.MkdirAll(scenariosRoot, 0o755))
	require.NoError(t, os.MkdirAll(templateUI, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(templateUI, "package.json"), []byte(`{"name":"template"}`), 0o600))

	reader := NewFSPackageJSONReader(scenariosRoot)

	got, err := reader.Read(context.Background(), "../templates/scenarios/react-vite")
	require.NoError(t, err)
	require.JSONEq(t, `{"name":"template"}`, string(got))

	_, err = reader.Read(context.Background(), "../templates/scenarios/../../secrets")
	require.ErrorContains(t, err, "invalid scenario name")
	_, err = reader.Read(context.Background(), "../other")
	require.ErrorContains(t, err, "invalid scenario name")
}
