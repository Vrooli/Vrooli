package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestReadEventNormalizesNativePayload(t *testing.T) {
	event, err := readEvent(strings.NewReader(`{"tool_name":"shell","command":"pnpm add example","cwd":"/workspace"}`), "codex")
	if err != nil {
		t.Fatal(err)
	}
	if event.Runner != "codex" || event.Tool != "shell" || event.Shell != "pnpm add example" {
		t.Fatalf("normalized event = %+v", event)
	}
}

// TestReadEventReadsNestedToolInput covers the PreToolUse shape Claude Code
// and Codex send: the command lives under tool_input, beside cwd and the
// session's permission mode.
func TestReadEventReadsNestedToolInput(t *testing.T) {
	event, err := readEvent(strings.NewReader(`{"session_id":"s1","cwd":"/repo","permission_mode":"default","tool_name":"Bash","tool_input":{"command":"rm -rf build"}}`), "claude-code")
	if err != nil {
		t.Fatal(err)
	}
	if event.Shell != "rm -rf build" || event.WorkingDirectory != "/repo" || event.Context["permission_mode"] != "default" {
		t.Fatalf("normalized event = %+v", event)
	}
	argv, err := readEvent(strings.NewReader(`{"tool_name":"shell","tool_input":{"command":["rm","-r","x"],"workdir":"/w"}}`), "codex")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(argv.Arguments, " ") != "rm -r x" || argv.WorkingDirectory != "/w" {
		t.Fatalf("argv event = %+v", argv)
	}
}

// TestHookDecidesRemovalWithoutProviders proves the runner enforces the
// removal floor on its own: a deletion of the filesystem root is denied with
// the deny exit class even when no provider has published anything.
func TestHookDecidesRemovalWithoutProviders(t *testing.T) {
	var output bytes.Buffer
	err := run([]string{"hook", "--runner", "codex", "--snapshot-dir", t.TempDir()}, strings.NewReader(`{"tool_name":"Bash","cwd":"/","tool_input":{"command":"rm -rf /"}}`), &output, &output)
	var decision exitError
	if !errors.As(err, &decision) || decision.code != 20 {
		t.Fatalf("hook error = %v, want deny exit 20; output %s", err, output.String())
	}
	if !strings.Contains(output.String(), `"risk": "filesystem_removal"`) {
		t.Fatalf("decision output = %s", output.String())
	}
}

func TestStatusDoesNotRequireProviderProcess(t *testing.T) {
	var output bytes.Buffer
	err := run([]string{"status", "--snapshot-dir", t.TempDir()}, strings.NewReader(""), &output, &output)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), `"status": "unavailable"`) {
		t.Fatalf("status output = %s", output.String())
	}
}
