package harness

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseNativeWriteFiltersNonMemoryTools(t *testing.T) {
	_, ok := parseNativeWrite("claude-code", []byte(`{"tool_name":"Bash","tool_input":{"command":"echo hi"}}`))
	require.False(t, ok)
	write, ok := parseNativeWrite("claude-code", []byte(`{"tool_name":"memory","tool_input":{"content":"remember this","path":"MEMORY.md"}}`))
	require.True(t, ok)
	require.Equal(t, nativeWrite{Runtime: "claude-code", Body: "remember this", Path: "MEMORY.md"}, write)
}

func TestNativeHookFailureAlwaysReturnsSuccess(t *testing.T) {
	err := processNativeHook("grok", []byte(`{"tool_name":"remember","tool_input":{"text":"durable fact"}}`), func(nativeWrite) error { return errors.New("api unavailable") })
	require.NoError(t, err)
}

func TestNativeHookAllowsImplicitMemoryDestination(t *testing.T) {
	var captured nativeWrite
	err := processNativeHook("claude-code", []byte(`{"tool_name":"memory","tool_input":{"content":"durable fact"}}`), func(write nativeWrite) error {
		captured = write
		return nil
	})
	require.NoError(t, err)
	require.Equal(t, nativeWrite{Runtime: "claude-code", Body: "durable fact"}, captured)
}

func TestReconcileHooksPreservesClaudeEntriesAndIsReversible(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Setenv("VROOLI_MEMORY_HOOK_HOME", dir))
	t.Cleanup(func() { _ = os.Unsetenv("VROOLI_MEMORY_HOOK_HOME") })
	settings := filepath.Join(dir, ".claude", "settings.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(settings), 0o755))
	require.NoError(t, os.WriteFile(settings, []byte(`{"hooks":{"PreToolUse":[{"managedBy":"user","hooks":[]}]},"other":true}`), 0o600))
	_, err := reconcileHooks("install", "claude-code")
	require.NoError(t, err)
	raw, err := os.ReadFile(settings)
	require.NoError(t, err)
	require.Contains(t, string(raw), memoryHookID)
	require.Contains(t, string(raw), `"other": true`)
	_, err = reconcileHooks("remove", "claude-code")
	require.NoError(t, err)
	raw, err = os.ReadFile(settings)
	require.NoError(t, err)
	require.NotContains(t, string(raw), memoryHookID)
	require.Contains(t, string(raw), "managedBy")
}
