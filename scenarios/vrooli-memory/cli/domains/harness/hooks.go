package harness

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"connectrpc.com/connect"
	harnessv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/harness"
)

const memoryHookID = "vrooli-memory-capture"

type nativeWrite struct {
	Runtime, Path, Body string
}

// parseNativeWrite accepts the structured hook envelopes used by the current
// Go-native harness resources. It is intentionally conservative: an ordinary
// tool call is ignored, while a recognized memory-write call with no body is
// treated as a no-op rather than guessed into journal content.
func parseNativeWrite(runtime string, raw []byte) (nativeWrite, bool) {
	var envelope map[string]any
	if json.Unmarshal(raw, &envelope) != nil {
		return nativeWrite{}, false
	}
	name := firstString(envelope, "tool_name", "toolName", "name", "tool")
	if !memoryTool(name) {
		return nativeWrite{}, false
	}
	input := firstMap(envelope, "tool_input", "toolInput", "input", "arguments")
	if input == nil {
		input = envelope
	}
	body := firstString(input, "content", "text", "body", "memory", "file_text")
	if strings.TrimSpace(body) == "" {
		return nativeWrite{}, false
	}
	return nativeWrite{Runtime: runtime, Path: firstString(input, "source_path", "sourcePath", "path", "file_path", "filePath"), Body: body}, true
}

func memoryTool(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "memory", "remember", "memory_write", "save_memory", "write_memory", "native_memory_write":
		return true
	default:
		return false
	}
}

func firstString(m map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := m[key].(string); ok && strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func firstMap(m map[string]any, keys ...string) map[string]any {
	for _, key := range keys {
		if value, ok := m[key].(map[string]any); ok {
			return value
		}
	}
	return nil
}

func (h *handlers) hook(args []string) error {
	fs := flag.NewFlagSet("vrooli-memory hook", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	runtime := fs.String("runtime", "", "source harness runtime")
	if err := fs.Parse(args); err != nil {
		return nil // hook failures must never block the agent
	}
	raw, err := io.ReadAll(os.Stdin)
	if err != nil {
		return nil
	}
	return processNativeHook(*runtime, raw, func(write nativeWrite) error {
		_, err := h.client.CaptureWrite(context.Background(), connect.NewRequest(&harnessv1.CaptureWriteRequest{Runtime: write.Runtime, SourcePath: write.Path, Content: write.Body}))
		return err
	})
}

func processNativeHook(runtime string, raw []byte, capture func(nativeWrite) error) error {
	write, ok := parseNativeWrite(runtime, raw)
	if !ok {
		return nil
	}
	_ = capture(write)
	return nil // a capture outage must never block the native tool call
}

func (h *handlers) hooks(args []string) error {
	fs := flag.NewFlagSet("vrooli-memory hooks", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	action := fs.String("action", "install", "install or remove")
	runtime := fs.String("runtime", "all", "claude-code, grok, or all")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *action != "install" && *action != "remove" {
		return errors.New("--action must be install or remove")
	}
	result, err := reconcileHooks(*action, *runtime)
	if err != nil {
		return err
	}
	for _, line := range result {
		fmt.Fprintln(os.Stdout, line)
	}
	return nil
}

func reconcileHooks(action, runtime string) ([]string, error) {
	runtimes := []string{"claude-code", "grok"}
	if runtime != "all" {
		runtimes = []string{runtime}
	}
	for _, item := range runtimes {
		if item != "claude-code" && item != "grok" {
			return nil, fmt.Errorf("unsupported hook runtime %q", item)
		}
	}
	home := strings.TrimSpace(os.Getenv("VROOLI_MEMORY_HOOK_HOME"))
	if home == "" {
		var err error
		home, err = os.UserHomeDir()
		if err != nil {
			return nil, err
		}
	}
	var out []string
	for _, item := range runtimes {
		var err error
		if item == "claude-code" {
			err = reconcileClaude(filepath.Join(home, ".claude", "settings.json"), action)
		} else {
			err = reconcileGrok(filepath.Join(home, ".grok", "hooks", memoryHookID+".json"), action)
		}
		if err != nil {
			return nil, err
		}
		verb := "installed"
		if action == "remove" {
			verb = "removed"
		}
		out = append(out, fmt.Sprintf("%s hook %s", item, verb))
	}
	return out, nil
}

func reconcileClaude(path, action string) error {
	doc := map[string]any{}
	if raw, err := os.ReadFile(path); err == nil && len(raw) > 0 {
		if err := json.Unmarshal(raw, &doc); err != nil {
			return err
		}
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}
	hooks, _ := doc["hooks"].(map[string]any)
	if hooks == nil {
		hooks = map[string]any{}
	}
	entries, _ := hooks["PreToolUse"].([]any)
	filtered := make([]any, 0, len(entries)+1)
	for _, raw := range entries {
		entry, _ := raw.(map[string]any)
		if entry["managedBy"] == memoryHookID {
			continue
		}
		filtered = append(filtered, raw)
	}
	if action == "install" {
		filtered = append(filtered, map[string]any{"matcher": "*", "managedBy": memoryHookID, "hooks": []any{map[string]any{"type": "command", "command": "vrooli-memory hook --runtime claude-code"}}})
	}
	if len(filtered) == 0 {
		delete(hooks, "PreToolUse")
	} else {
		hooks["PreToolUse"] = filtered
	}
	if len(hooks) == 0 {
		delete(doc, "hooks")
	} else {
		doc["hooks"] = hooks
	}
	return writeJSON(path, doc)
}

func reconcileGrok(path, action string) error {
	if action == "remove" {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	return writeJSON(path, map[string]any{"managedBy": memoryHookID, "hooks": map[string]any{"PreToolUse": []any{map[string]any{"matcher": "*", "hooks": []any{map[string]any{"type": "command", "command": "vrooli-memory hook --runtime grok"}}}}}})
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(data, '\n'), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
