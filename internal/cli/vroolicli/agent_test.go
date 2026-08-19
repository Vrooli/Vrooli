package vroolicli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAgentRunnerBinaryAllowlist(t *testing.T) {
	for _, test := range []struct {
		input string
		want  string
	}{
		{input: "claude", want: "claude"},
		{input: "claude-code", want: "claude"},
		{input: "codex", want: "codex"},
		{input: "opencode", want: "opencode"},
		{input: "grok", want: "grok"},
	} {
		got, err := agentRunnerBinary(test.input)
		if err != nil || got != test.want {
			t.Fatalf("agentRunnerBinary(%q) = %q, %v; want %q", test.input, got, err, test.want)
		}
	}
	if _, err := agentRunnerBinary("sh -c whoami"); err == nil {
		t.Fatal("agentRunnerBinary accepted a non-allowlisted executable")
	}
}

func TestRunAgentCommandUsesTypedRunnerArguments(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PATH", dir)

	for _, test := range []struct {
		runner string
		want   string
	}{
		{runner: "claude", want: "--dangerously-skip-permissions\n-p\nhello\n"},
		{runner: "codex", want: "--dangerously-skip-permissions\nhello\n"},
		{runner: "opencode", want: "run\n--dangerously-skip-permissions\nhello\n"},
		{runner: "grok", want: "--dangerously-skip-permissions\nhello\n"},
	} {
		t.Run(test.runner, func(t *testing.T) {
			argsPath := filepath.Join(dir, test.runner+"-args")
			script := filepath.Join(dir, test.runner)
			contents := "#!/bin/sh\nprintf '%s\\n' \"$@\" > \"" + argsPath + "\"\n"
			if err := os.WriteFile(script, []byte(contents), 0o700); err != nil {
				t.Fatal(err)
			}
			ctx := &CommandContext{
				Stdin:  strings.NewReader("operator input\n"),
				Stdout: &strings.Builder{},
				Stderr: &strings.Builder{},
			}
			if err := (&App{}).runAgentCommand(ctx, []string{"launch", "--runner", test.runner, "--arg=--dangerously-skip-permissions", "--prompt", "hello"}); err != nil {
				t.Fatalf("runAgentCommand: %v", err)
			}
			data, err := os.ReadFile(argsPath)
			if err != nil {
				t.Fatal(err)
			}
			if got := string(data); got != test.want {
				t.Fatalf("runner argv = %q, want %q", got, test.want)
			}
		})
	}
}
