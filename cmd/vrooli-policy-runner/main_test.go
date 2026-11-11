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

// TestHookVerdictFollowsTheCallingHarness proves the runner enforces the
// removal floor on its own (no provider has published anything) and reports
// the verdict the way each harness reads it: Codex blocks only on exit 2 with a
// stderr reason and a silent stdout; other runners keep the decision document
// and the stable exit classes.
func TestHookVerdictFollowsTheCallingHarness(t *testing.T) {
	const deny = `{"tool_name":"Bash","cwd":"/","tool_input":{"command":"rm -rf /"}}`
	const allow = `{"tool_name":"Bash","cwd":"/tmp","tool_input":{"command":"echo hi"}}`
	cases := []struct {
		name, runner, input string
		wantExit            int // 0 means the hook returns no exit error
		wantStdoutJSON      bool
		wantStderr          string
	}{
		{"codex deny blocks with exit 2 and a stderr reason", "codex", deny, 2, false, "Blocked by Vrooli agent policy: / is a filesystem root"},
		{"codex allow is silent", "codex", allow, 0, false, ""},
		{"opencode deny keeps the stable deny class", "opencode", deny, 20, true, ""},
		{"grok deny keeps the stable deny class", "grok", deny, 20, true, ""},
		{"opencode allow reports the decision", "opencode", allow, 0, true, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			err := run([]string{"hook", "--runner", tc.runner, "--snapshot-dir", t.TempDir()}, strings.NewReader(tc.input), &stdout, &stderr)
			var decision exitError
			switch {
			case tc.wantExit == 0 && err != nil:
				t.Fatalf("hook error = %v, want allow", err)
			case tc.wantExit != 0 && (!errors.As(err, &decision) || decision.code != tc.wantExit):
				t.Fatalf("hook error = %v, want exit %d", err, tc.wantExit)
			}
			if got := strings.Contains(stdout.String(), `"action"`); got != tc.wantStdoutJSON {
				t.Fatalf("stdout carries a decision = %t, want %t: %q", got, tc.wantStdoutJSON, stdout.String())
			}
			if !strings.Contains(stderr.String(), tc.wantStderr) || (tc.wantStderr == "" && stderr.Len() != 0) {
				t.Fatalf("stderr = %q, want %q", stderr.String(), tc.wantStderr)
			}
		})
	}
}

// A hook that crashes must allow: a Go panic exits 2, which Codex reads as block.
func TestHookPanicAllowsInsteadOfBlocking(t *testing.T) {
	var stderr bytes.Buffer
	err := errors.New("pending")
	func() {
		defer allowOnPanic(&stderr, &err)
		panic("boom")
	}()
	if err != nil || !strings.Contains(stderr.String(), "allowed after internal error: boom") {
		t.Fatalf("err = %v, stderr = %q", err, stderr.String())
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
