package autofix

import (
	"os"
	"path/filepath"
	"testing"

	"quality-health/internal/contracts"

	"github.com/stretchr/testify/require"
)

func TestPreviewAndApplyTSConfigStrictFix(t *testing.T) {
	root := t.TempDir()
	ui := filepath.Join(root, "ui")
	require.NoError(t, os.MkdirAll(ui, 0o755))
	path := filepath.Join(ui, "tsconfig.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"compilerOptions":{}}`), 0o644))

	preview, err := Preview(root, []string{contracts.RuleTSConfigStrict})
	require.NoError(t, err)
	require.Len(t, preview, 1)
	require.False(t, preview[0].Applied)
	require.Contains(t, preview[0].After, `"strict": true`)
	require.Contains(t, preview[0].After, "SAFETY-CRITICAL RULES")

	applied, err := Apply(root, []string{contracts.RuleTSConfigStrict})
	require.NoError(t, err)
	require.Len(t, applied, 1)
	require.True(t, applied[0].Applied)
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Contains(t, string(raw), `"noUncheckedIndexedAccess": true`)
	require.True(t, HasTSConfigProtectiveComments(string(raw)))
}
