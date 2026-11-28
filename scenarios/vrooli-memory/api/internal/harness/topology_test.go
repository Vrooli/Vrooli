package harness

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEnsureCuratedTopologyUsesProjectAgentsAsCanonicalAndRepairsLinks(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(t.TempDir(), "home")
	require.NoError(t, os.MkdirAll(filepath.Join(home, ".codex"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte("# Curated\n\n<!-- vrooli-memory:prompt-block:start -->\nprogrammatic text\n<!-- vrooli-memory:prompt-block:end -->\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root, "CLAUDE.md"), []byte("# Curated\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(home, ".codex", "AGENTS.md"), []byte("# Longer peer that must not replace canonical\n\nExtra\n"), 0o600))

	result, err := EnsureCuratedTopology(root, home)
	require.NoError(t, err)
	require.True(t, result.Changed)
	require.Equal(t, "# Curated\n", string(mustReadFile(t, filepath.Join(root, "AGENTS.md"))))
	for _, alias := range []string{filepath.Join(root, "CLAUDE.md"), filepath.Join(root, "GEMINI.md"), filepath.Join(home, ".codex", "AGENTS.md")} {
		require.Equal(t, filepath.Join(root, "AGENTS.md"), mustReadlink(t, alias))
	}
}

func TestDiscoverWorkspaceRootCanRecoverDeletedCanonical(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".vrooli"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "scenarios"), 0o755))
	working := filepath.Join(root, "scenarios", "vrooli-memory")
	require.NoError(t, os.MkdirAll(working, 0o755))
	t.Chdir(working)

	got, err := discoverWorkspaceRoot()
	require.NoError(t, err)
	require.Equal(t, root, got)
}

func TestEnsureCuratedTopologyRecoversMissingCanonicalWithoutImportingGeneratedProjection(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(t.TempDir(), "home")
	require.NoError(t, os.MkdirAll(filepath.Join(home, ".config", "opencode"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "CLAUDE.md"), []byte("# Recovered\n\nRule\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(home, ".config", "opencode", "AGENTS.md"), []byte(generatedHeader+"# Unified Vrooli Memory\n"), 0o600))

	_, err := EnsureCuratedTopology(root, home)
	require.NoError(t, err)
	require.Equal(t, "# Recovered\n\nRule\n", string(mustReadFile(t, filepath.Join(root, "AGENTS.md"))))
}

func TestEnsureCuratedTopologyRejectsNoSurvivingCandidate(t *testing.T) {
	_, err := EnsureCuratedTopology(t.TempDir(), t.TempDir())
	require.ErrorContains(t, err, "no surviving candidate")
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	return data
}

func mustReadlink(t *testing.T, path string) string {
	t.Helper()
	target, err := os.Readlink(path)
	require.NoError(t, err)
	return target
}
