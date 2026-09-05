package harness

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"connectrpc.com/connect"
	harnessv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/harness"
)

const memoryHookID = "vrooli-memory-capture"

type hookCommandRunner func(home, runtime string, args []string) error

var runHookCommand hookCommandRunner = executeHookCommand

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
		args, err := hookCommandArgs(item, action)
		if err == nil {
			err = runHookCommand(home, item, args)
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

func hookCommandArgs(runtime, action string) ([]string, error) {
	if action != "install" && action != "remove" {
		return nil, fmt.Errorf("unsupported hook action %q", action)
	}
	event := "PreToolUse"
	hook := map[string]any{"type": "command", "command": "vrooli-memory hook --runtime " + runtime}
	if action == "remove" {
		return []string{"hooks", "remove", "--event", event, "--id", memoryHookID, "--scope", "global"}, nil
	}
	data, err := json.Marshal(hook)
	if err != nil {
		return nil, err
	}
	return []string{"hooks", "reconcile", "--event", event, "--id", memoryHookID, "--hook-json", string(data), "--scope", "global"}, nil
}

func executeHookCommand(home, runtime string, args []string) error {
	binary := "resource-" + runtime
	if override := strings.TrimSpace(os.Getenv("VROOLI_MEMORY_" + strings.ToUpper(strings.ReplaceAll(runtime, "-", "_")) + "_CLI")); override != "" {
		binary = override
	}
	cmd := exec.Command(binary, args...)
	env := append([]string(nil), os.Environ()...)
	env = append(env, "HOME="+home)
	if runtime == "grok" {
		env = append(env, "VROOLI_AGENT_HOOK_PATH="+filepath.Join(home, ".grok", "hooks", memoryHookID+".json"))
	}
	cmd.Env = env
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if message := strings.TrimSpace(stderr.String()); message != "" {
			return fmt.Errorf("%s: %w", message, err)
		}
		return err
	}
	return nil
}
