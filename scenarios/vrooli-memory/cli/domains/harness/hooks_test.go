package harness

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
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
	previousRunner := runHookCommand
	runHookCommand = func(home, runtime string, args []string) error {
		require.Equal(t, "claude-code", runtime)
		path := filepath.Join(home, ".claude", "settings.json")
		doc := map[string]any{}
		if raw, err := os.ReadFile(path); err == nil {
			require.NoError(t, json.Unmarshal(raw, &doc))
		}
		hooks, _ := doc["hooks"].(map[string]any)
		if hooks == nil {
			hooks = map[string]any{}
		}
		entries, _ := hooks["PreToolUse"].([]any)
		if strings.Contains(strings.Join(args, " "), "hooks remove") {
			kept := make([]any, 0, len(entries))
			for _, raw := range entries {
				group, _ := raw.(map[string]any)
				inner, _ := group["hooks"].([]any)
				if len(inner) == 1 {
					entry, _ := inner[0].(map[string]any)
					if entry["_id"] == memoryHookID {
						continue
					}
				}
				kept = append(kept, raw)
			}
			entries = kept
		} else {
			entries = append(entries, map[string]any{
				"matcher": "*",
				"hooks": []any{map[string]any{
					"_id": memoryHookID, "managedBy": "vrooli",
					"type": "command", "command": "vrooli-memory hook --runtime claude-code",
				}},
			})
		}
		if len(entries) == 0 {
			delete(hooks, "PreToolUse")
		} else {
			hooks["PreToolUse"] = entries
		}
		if len(hooks) == 0 {
			delete(doc, "hooks")
		} else {
			doc["hooks"] = hooks
		}
		data, err := json.MarshalIndent(doc, "", "  ")
		require.NoError(t, err)
		return os.WriteFile(path, append(data, '\n'), 0o600)
	}
	t.Cleanup(func() { runHookCommand = previousRunner })
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
