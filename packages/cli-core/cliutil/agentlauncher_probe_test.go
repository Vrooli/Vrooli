package cliutil

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestInformationalInvocationRecognisesProbes(t *testing.T) {
	for _, args := range [][]string{
		{"--version"}, {"-V"}, {"--help"}, {"-h"}, {"doctor"},
	} {
		if !informationalInvocation("codex", args) {
			t.Fatalf("%v asks the binary a question and exits; it is not a session", args)
		}
	}
}

// The set is closed on purpose: anything unrecognised stays governed, so a
// new agent flag is contained and attributed until someone decides otherwise.
func TestInformationalInvocationTreatsAnythingElseAsASession(t *testing.T) {
	for _, args := range [][]string{
		nil,
		{},
		{"--yolo"},
		{"--version", "--yolo"},
		{"exec", "--help", "prompt"},
		{"--dangerously-skip-permissions"},
	} {
		if informationalInvocation("codex", args) {
			t.Fatalf("%v must be treated as a session", args)
		}
	}
}

// A probe must not mint a lease, a scope or a ceiling. On 2026-09-05 a loop
// asking all five agents `--version` every 15 seconds produced roughly 29,000
// scopes and 29,000 lease rows a day.
func TestProbeRecordsNoLeaseAndNoScope(t *testing.T) {
	var gotArgs []string
	request := AgentLaunchRequest{
		Agent:    "codex",
		Args:     []string{"--version"},
		LookPath: func(string) (string, error) { return "/usr/bin/codex", nil },
		RunChild: func(_ context.Context, _ string, args, _ []string, _ io.Reader, _, _ io.Writer) error {
			gotArgs = args
			return nil
		},
	}
	result, err := LaunchCodingAgentResult(context.Background(), request)
	if err != nil {
		t.Fatalf("probe: %v", err)
	}
	if result.LeaseRecorded {
		t.Fatal("a probe must record no editor lease")
	}
	if result.Scope != "" {
		t.Fatalf("a probe must create no scope: %q", result.Scope)
	}
	if result.ContainmentMethod != "" {
		t.Fatalf("a probe needs no ceiling: %q", result.ContainmentMethod)
	}
	if !strings.Contains(result.AttachFailure, "not a session") {
		t.Fatalf("the result must say why it was ungoverned: %q", result.AttachFailure)
	}
	if len(gotArgs) != 1 || gotArgs[0] != "--version" {
		t.Fatalf("the probe's arguments must reach the binary unchanged: %v", gotArgs)
	}
}

// A probe still answers: the operator's `codex --version` must work exactly
// as before, and its exit status must be the binary's.
func TestProbeStillRunsTheBinaryAndReturnsItsError(t *testing.T) {
	sentinel := io.ErrUnexpectedEOF
	request := AgentLaunchRequest{
		Agent:    "grok",
		Args:     []string{"--help"},
		LookPath: func(string) (string, error) { return "/usr/bin/grok", nil },
		RunChild: func(context.Context, string, []string, []string, io.Reader, io.Writer, io.Writer) error {
			return sentinel
		},
	}
	if _, err := LaunchCodingAgentResult(context.Background(), request); err != sentinel {
		t.Fatalf("the binary's exit status must survive: got %v", err)
	}
}

// A launch reaches the agent through more than one launcher stage and exec
// keeps the pid, so on 2026-09-04 each claude session produced two leases
// 36 ms apart under two spellings of the agent name, and the first stage's
// scope stayed named in a lease after systemd reaped the empty cgroup.
func TestNestedLaunchInheritsTheSessionInsteadOfOpeningASecond(t *testing.T) {
	request := AgentLaunchRequest{
		Agent:       "claude",
		Args:        []string{"--dangerously-skip-permissions"},
		Environment: []string{AgentSessionEnv + "=vrooli-agent-claude-code-e1e56d07"},
		LookPath:    func(string) (string, error) { return "/usr/bin/claude", nil },
		RunChild:    func(context.Context, string, []string, []string, io.Reader, io.Writer, io.Writer) error { return nil },
	}
	result, err := LaunchCodingAgentResult(context.Background(), request)
	if err != nil {
		t.Fatalf("nested launch: %v", err)
	}
	if result.LeaseRecorded {
		t.Fatal("a nested launcher stage must record no second lease")
	}
	if result.Scope != "vrooli-agent-claude-code-e1e56d07" {
		t.Fatalf("the nested stage must report the inherited scope, not mint one: %q", result.Scope)
	}
	if !strings.Contains(result.AttachFailure, "inherited") {
		t.Fatalf("the result must say the session was inherited: %q", result.AttachFailure)
	}
}

// An empty marker is an unmarked environment: the session is opened normally.
func TestEmptyMarkerIsNotAnInheritedSession(t *testing.T) {
	if environmentValue([]string{AgentSessionEnv + "="}, AgentSessionEnv) != "" {
		t.Fatal("an empty marker must read as absent")
	}
	if environmentValue([]string{"OTHER=x"}, AgentSessionEnv) != "" {
		t.Fatal("an absent marker must read as empty")
	}
	if got := environmentValue([]string{"A=1", AgentSessionEnv + "=scope-1"}, AgentSessionEnv); got != "scope-1" {
		t.Fatalf("marker: got %q", got)
	}
}

// The first stage must publish the marker so the next stage can see it.
func TestFirstStagePublishesTheSessionMarkerToTheChild(t *testing.T) {
	var childEnv []string
	request := AgentLaunchRequest{
		Agent:       "codex",
		Args:        []string{"--yolo"},
		Environment: []string{"PATH=/usr/bin"},
		LookPath:    func(string) (string, error) { return "/usr/bin/codex", nil },
		RunChild: func(_ context.Context, _ string, _, environment []string, _ io.Reader, _, _ io.Writer) error {
			childEnv = environment
			return nil
		},
	}
	if _, err := LaunchCodingAgentResult(context.Background(), request); err != nil {
		t.Fatalf("launch: %v", err)
	}
	if marker := environmentValue(childEnv, AgentSessionEnv); marker == "" {
		t.Fatalf("the child's environment must carry the session marker: %v", childEnv)
	}
}

// The ceiling refuses silently and the agent reports the refusal in its own
// words: on 2026-09-04 Codex called a full task ceiling "your local database
// appears to be damaged" and sent the operator to delete intact state. The
// launcher says the real reason first.
func TestLaunchWarnsWhenTheCeilingIsAboutToRefuse(t *testing.T) {
	previous := DefaultCeilingDiagnostic
	t.Cleanup(func() { DefaultCeilingDiagnostic = previous })
	RegisterCeilingDiagnostic(func() string {
		return "vrooli-agents.slice holds 4059 of its 4096 tasks. The database is fine."
	})
	var stderr bytes.Buffer
	request := AgentLaunchRequest{
		Agent:    "codex",
		Args:     []string{"--yolo"},
		Stderr:   &stderr,
		LookPath: func(string) (string, error) { return "/usr/bin/codex", nil },
		RunChild: func(context.Context, string, []string, []string, io.Reader, io.Writer, io.Writer) error { return nil },
	}
	result, err := LaunchCodingAgentResult(context.Background(), request)
	if err != nil {
		t.Fatalf("launch: %v", err)
	}
	if !strings.Contains(stderr.String(), "4059 of its 4096 tasks") {
		t.Fatalf("the operator must see the real reason on stderr: %q", stderr.String())
	}
	if !strings.Contains(result.CeilingWarning, "4059") {
		t.Fatalf("the result must carry the warning: %q", result.CeilingWarning)
	}
}

// A ceiling with room, or one that cannot be read, says nothing: a launcher
// diagnostic must never become noise on every launch.
func TestLaunchIsSilentWhenTheCeilingHasRoom(t *testing.T) {
	previous := DefaultCeilingDiagnostic
	t.Cleanup(func() { DefaultCeilingDiagnostic = previous })
	for _, diagnostic := range []CeilingDiagnostic{nil, func() string { return "" }, func() string { return "   " }} {
		DefaultCeilingDiagnostic = diagnostic
		var stderr bytes.Buffer
		request := AgentLaunchRequest{
			Agent:    "codex",
			Args:     []string{"--yolo"},
			Stderr:   &stderr,
			LookPath: func(string) (string, error) { return "/usr/bin/codex", nil },
			RunChild: func(context.Context, string, []string, []string, io.Reader, io.Writer, io.Writer) error { return nil },
		}
		result, err := LaunchCodingAgentResult(context.Background(), request)
		if err != nil {
			t.Fatalf("launch: %v", err)
		}
		if stderr.Len() != 0 || result.CeilingWarning != "" {
			t.Fatalf("a ceiling with room must say nothing: %q / %q", stderr.String(), result.CeilingWarning)
		}
	}
}

// The warning never blocks the launch: attribution and diagnostics are
// observability, and the agent must still start.
func TestCeilingWarningDoesNotBlockTheLaunch(t *testing.T) {
	previous := DefaultCeilingDiagnostic
	t.Cleanup(func() { DefaultCeilingDiagnostic = previous })
	RegisterCeilingDiagnostic(func() string { return "the ceiling is full" })
	started := false
	request := AgentLaunchRequest{
		Agent:    "codex",
		Args:     []string{"--yolo"},
		Stderr:   io.Discard,
		LookPath: func(string) (string, error) { return "/usr/bin/codex", nil },
		RunChild: func(context.Context, string, []string, []string, io.Reader, io.Writer, io.Writer) error {
			started = true
			return nil
		},
	}
	if _, err := LaunchCodingAgentResult(context.Background(), request); err != nil {
		t.Fatalf("launch: %v", err)
	}
	if !started {
		t.Fatal("a full ceiling is a warning, never a refusal to start the agent")
	}
}

// Help for a known task subcommand must answer without opening a session. All
// downstream effects are fake; the installed provider binary is never executed.
func TestSubcommandHelpRoutingUsesOnlyInformationalPath(t *testing.T) {
	type invocationCase struct {
		agent string
		args  []string
		probe bool
	}
	cases := []invocationCase{
		{"codex", []string{"exec", "--help"}, true},
		{"codex", []string{"exec", "-h"}, true},
		{"codex", []string{"exec", "-h", "--help"}, false},
		{"opencode", []string{"run", "--help"}, true},
		{"opencode", []string{"run", "-h"}, true},
		{"claude", []string{"--help"}, true},
		{"grok", []string{"--help"}, true},
		{"codex", []string{"exec", "compute 6*7"}, false},
		{"opencode", []string{"run", "compute 6*7"}, false},
		{"codex", []string{"exec", "--", "--help"}, false},
		{"opencode", []string{"run", "--", "--help"}, false},
		{"codex", []string{"exec", "--help", "--", "prompt"}, false},
		{"codex", []string{"exec", "--help", "prompt"}, false},
		{"opencode", []string{"run", "prompt", "--help"}, false},
		{"codex", []string{"exec", "--version"}, false},
		{"opencode", []string{"exec", "--help"}, false},
		{"codex", []string{"run", "--help"}, false},
		{"claude", []string{"exec", "--help"}, false},
		{"codex", []string{"EXEC", "--help"}, false},
		{"opencode", []string{" run", "--help"}, false},
		{"codex", []string{"exec", " --help"}, false},
		{"codex", []string{"exec", "--help="}, false},
		{"codex", []string{"--", "--help"}, false},
		{"codex", []string{"--help "}, false},
		{"codex", []string{"exec"}, false},
		{"opencode", nil, false},
	}

	// Every registered harness is checked independently. A positional word is
	// not a portable subcommand, and repeated/mixed flags are not qualified probes.
	for _, agent := range []string{"codex", "claude", "opencode", "grok", "agy"} {
		qualified := agent != "agy"
		for _, flag := range []string{"--help", "-h", "--version"} {
			cases = append(cases, invocationCase{agent, []string{flag}, qualified})
		}
		for _, flag := range []string{"-v", "-V"} {
			allowed := flag == "-V" && agent == "codex" || flag == "-v" && (agent == "claude" || agent == "opencode")
			cases = append(cases, invocationCase{agent, []string{flag}, allowed})
		}
		cases = append(cases, invocationCase{agent, []string{"doctor"}, agent == "codex" || agent == "claude"})
		for _, args := range [][]string{
			{"doctor", "doctor"}, {"doctor", "--version"}, {"--version", "doctor"},
			{"doctor", "--help"}, {"--help", "doctor"}, {"doctor", "project"},
			{"project", "--help"}, {"--", "doctor"}, {"--help", "--help"},
			{"--help", "--version"}, {"--version", "--version"}, {"-v", "-V"},
		} {
			cases = append(cases, invocationCase{agent, args, false})
		}
	}
	for _, tc := range cases {
		t.Run(tc.agent+"/"+strings.Join(tc.args, "|"), func(t *testing.T) {
			isolateLauncherModelPolicy(t)
			t.Setenv("TMPDIR", t.TempDir())
			t.Setenv(AgentSessionEnv, "")
			recorder := &recordingLeaseRecorder{}
			withLeaseRecorder(t, recorder)
			container := withFakeContainer(t, false)
			attachments := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				attachments++
				w.WriteHeader(http.StatusServiceUnavailable)
			}))
			defer server.Close()

			// In OpenCode doctor is a possible project name, not a diagnostic command.
			// Use a real private directory while retaining fake provider execution.
			workingDir := t.TempDir()
			if err := os.Mkdir(filepath.Join(workingDir, "doctor"), 0700); err != nil {
				t.Fatal(err)
			}
			calls := 0
			result, err := LaunchCodingAgentResult(context.Background(), AgentLaunchRequest{
				Agent: tc.agent, Args: tc.args, Environment: []string{"PATH=/fixture"}, APIBase: server.URL, WorkingDir: workingDir,
				LookPath: func(string) (string, error) { return "/fixture/provider", nil },
				RunChild: func(_ context.Context, path string, args, env []string, _ io.Reader, _, _ io.Writer) error {
					calls++
					if path != "/fixture/provider" || !reflect.DeepEqual(args, tc.args) {
						t.Errorf("argv changed: %q %q", path, args)
					}
					if (environmentValue(env, AgentSessionEnv) == "") != tc.probe {
						t.Errorf("session marker probe=%v", tc.probe)
					}
					return nil
				},
			})
			if err != nil || calls != 1 {
				t.Fatalf("fake child result: %+v %v calls=%d", result, err, calls)
			}
			if tc.probe {
				if attachments != 0 || len(recorder.created) != 0 || len(recorder.stopped) != 0 || len(container.runs) != 0 || len(container.selfCalls) != 0 || result.Scope != "" || result.LeaseRecorded || !strings.Contains(result.AttachFailure, "not a session") {
					t.Fatalf("help opened session: result=%+v attach=%d leases=%d", result, attachments, len(recorder.created))
				}
			} else {
				if attachments != 1 || len(recorder.created) != 1 || len(recorder.stopped) != 1 || !result.LeaseRecorded || strings.Contains(result.AttachFailure, "not a session") {
					t.Fatalf("session escaped governance: result=%+v attach=%d leases=%d", result, attachments, len(recorder.created))
				}
			}
		})
	}
}

func TestProtectedSubcommandHelpStillRequiresExistingCustody(t *testing.T) {
	for _, tc := range []struct{ agent, command string }{{"codex", "exec"}, {"opencode", "run"}} {
		t.Run(tc.agent, func(t *testing.T) {
			isolateLauncherModelPolicy(t)
			t.Setenv("TMPDIR", t.TempDir())
			t.Setenv(AgentSessionEnv, "")
			called := false
			_, err := LaunchCodingAgentResult(context.Background(), AgentLaunchRequest{
				Agent: tc.agent, Args: []string{tc.command, "--help"}, RequireNativeIdentity: true,
				LookPath: func(string) (string, error) { return "/fixture/provider", nil },
				RunChild: func(context.Context, string, []string, []string, io.Reader, io.Writer, io.Writer) error {
					called = true
					return nil
				},
			})
			if err == nil || called {
				t.Fatalf("protected help escaped custody: %v called=%v", err, called)
			}
		})
	}
}
