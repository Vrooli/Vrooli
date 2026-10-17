package vroolicli

import (
	"encoding/json"
	"github.com/vrooli/cli-core/cliutil"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/shell/shelltest"
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
			contents := shelltest.POSIXShebang() + "printf '%s\\n' \"$@\" > \"" + argsPath + "\"\n"
			if err := os.WriteFile(script, []byte(contents), 0o700); err != nil {
				t.Fatal(err)
			}
			ctx := &AppContext{
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

func TestRunAgentExecProtectedCommandAndTypedArguments(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	namespace, err := os.MkdirTemp("", "cmd-fixture-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(namespace)
	t.Setenv("TMPDIR", namespace)
	t.Setenv(cliutil.EnvIdentityToken, "")
	t.Setenv(cliutil.AgentSessionEnv, "")
	attach, detach := 0, 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/attach") {
			var body struct {
				HarnessKind string `json:"harness_kind"`
				Session     string `json:"harness_session_id"`
				PID         int    `json:"process_id"`
			}
			if json.NewDecoder(r.Body).Decode(&body) != nil || body.HarnessKind != "command-session" || body.Session == "" || body.PID != os.Getpid() {
				t.Error("dishonest command session binding")
			}
			attach++
			w.WriteHeader(201)
			io.WriteString(w, `{"run":{"id":"synthetic-owned-command"},"identity_token":"synthetic-command-leaf"}`)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/detach") {
			detach++
			w.WriteHeader(200)
			return
		}
		t.Error("unexpected route")
	}))
	defer server.Close()
	t.Setenv(cliutil.AgentManagerLauncherBaseEnv, server.URL)
	var out strings.Builder
	ctx := &AppContext{Stdin: strings.NewReader(""), Stdout: &out, Stderr: &strings.Builder{}}
	if err := (&App{}).runAgentCommand(ctx, []string{"exec", "--", "/bin/echo", "literal $(no-shell)", "a b"}); err != nil {
		t.Fatal(err)
	}
	if out.String() != "literal $(no-shell) a b\n" || attach != 1 || detach != 1 {
		t.Fatal("argv/cleanup", out.String(), attach, detach)
	}
	for _, args := range [][]string{{"exec"}, {"exec", "--task", "invalid", "--", "/bin/echo"}, {"exec", "--task", "00000000-0000-0000-0000-000000000000", "--", "/bin/echo"}} {
		if err := (&App{}).runAgentCommand(ctx, args); err == nil {
			t.Fatal("invalid command accepted")
		}
	}
	if attach != 1 || detach != 1 {
		t.Fatal("invalid arguments caused effects")
	}
}
