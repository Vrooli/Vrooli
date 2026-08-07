package harness

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPromptBlockIsIdempotentAndDoesNotNameMemoryCommand(t *testing.T) { // [REQ:VMEM-P1-007]
	path := filepath.Join(t.TempDir(), "AGENTS.md")
	require.NoError(t, os.WriteFile(path, []byte("# Existing\n"), 0o600))
	require.NoError(t, InstallPromptBlock(path))
	require.NoError(t, InstallPromptBlock(path))
	b, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, 1, strings.Count(string(b), promptStart))
	require.NotContains(t, strings.ToLower(PromptBlock()), "vrooli-memory ")
}

func TestPromptBlockMismatchIsRepairedToTheCanonicalBlock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "CLAUDE.md")
	content := "# Existing\n" + promptStart + "\n## stale wording\n" + promptEnd + "\n"
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	require.NoError(t, InstallPromptBlock(path))
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, "# Existing\n"+PromptBlock()+"\n", string(data))
}

func TestPromptTargetsAreSeparateFromProjectionFiles(t *testing.T) {
	root := t.TempDir()
	claude, err := PromptTarget("claude-code", root)
	require.NoError(t, err)
	codex, err := PromptTarget("codex", root)
	require.NoError(t, err)
	require.Equal(t, filepath.Join(root, "CLAUDE.md"), claude)
	require.Equal(t, filepath.Join(root, "AGENTS.md"), codex)
	_, err = PromptTarget("grok", root)
	require.Error(t, err)
}
