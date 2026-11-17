package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"

	agentharness "github.com/vrooli/agentharness"
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

func TestPolicyRunnerGuardModesAreReadOnly(t *testing.T) {
	dir := t.TempDir()
	var output bytes.Buffer
	if err := run([]string{"guard-mode", "enable", "--snapshot-dir", dir}, strings.NewReader(""), &output, &output); err == nil || !strings.Contains(err.Error(), "read-only") {
		t.Fatalf("enable result = %v, want read-only refusal", err)
	}
	output.Reset()
	if err := run([]string{"guard-mode", "list", "--snapshot-dir", dir}, strings.NewReader(""), &output, &output); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), `"active"`) {
		t.Fatalf("list result = %s", output.String())
	}
	output.Reset()
	if err := run([]string{"guard-mode", "audit", "--snapshot-dir", dir}, strings.NewReader(""), &output, &output); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), `"entries"`) {
		t.Fatalf("audit result = %s", output.String())
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
		{"opencode deny keeps the stable deny class", "opencode", deny, 20, true, "evaluator audit receipt unavailable"},
		{"grok deny keeps the stable deny class", "grok", deny, 20, true, "evaluator audit receipt unavailable"},
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

func TestHookRefusalCarriesOnlyProviderOwnedEvaluatorIdentity(t *testing.T) {
	decision := agentharness.Decision{Action: agentharness.ActionDeny, ProviderID: "scenario-dependency-analyzer", EvaluatorID: "scenario-dependency-analyzer:dependency-addition", Reason: "route dependency mutation"}
	var stdout, stderr bytes.Buffer
	err := answerHook("codex", decision, &stdout, &stderr)
	var denied exitError
	if !errors.As(err, &denied) || denied.code != codexDenyExit || strings.Contains(stderr.String(), "vrooli_guard_check") || strings.Contains(stdout.String(), "vrooli_guard_check") {
		t.Fatalf("hook refusal unexpectedly used transcript identity bridge: err=%v stdout=%q stderr=%q", err, stdout.String(), stderr.String())
	}
	decision.EvaluatorID = "other-provider:dependency-addition"
	stdout.Reset()
	stderr.Reset()
	err = answerHook("claude-code", decision, &stdout, &stderr)
	if err == nil || strings.Contains(stdout.String(), "vrooli_guard_check") {
		t.Fatalf("foreign evaluator identity was surfaced: err=%v stdout=%q", err, stdout.String())
	}
}

func TestHookCommandRefusalCarriesEvaluatorIdentityFromSnapshot(t *testing.T) {
	now := time.Now().UTC()
	dir := t.TempDir()
	store := agentharness.NewBundleStore(dir)
	snapshot := agentharness.ProviderSnapshot{
		ProviderID: "scenario-dependency-analyzer", Version: "test",
		Capabilities: []agentharness.ProviderCapability{{ID: "dependency-policy", IdealPosture: "deny unapproved package mutation", DeclaredMaturity: agentharness.MaturityEnforcing, SupportsAnalysis: true, SupportsEnforce: true}},
		Health:       agentharness.ProviderHealth{State: agentharness.HealthHealthy, CheckedAt: now.Add(-time.Minute), ExpiresAt: now.Add(time.Hour)},
		Readiness:    agentharness.ProviderReadiness{State: "ready", RollbackPlan: "test only"},
		Evidence:     agentharness.EvidenceClean, CapturedAt: now.Add(-time.Minute), ExpiresAt: now.Add(time.Hour),
		Rules: []agentharness.PolicyRule{{EvaluatorID: "scenario-dependency-analyzer:dependency-addition", Risk: agentharness.RiskDependencyAdd, Action: agentharness.ActionDeny, Reason: "unapproved package mutation"}},
	}
	if err := store.PublishProvider(snapshot); err != nil {
		t.Fatal(err)
	}
	input := `{"tool_name":"pnpm","cwd":"/repo","arguments":["add","example"]}`
	var stdout, stderr bytes.Buffer
	err := run([]string{"hook", "--runner", "codex", "--profile", "enforcing", "--snapshot-dir", dir}, strings.NewReader(input), &stdout, &stderr)
	var denied exitError
	if !errors.As(err, &denied) || denied.code != codexDenyExit || strings.Contains(stderr.String(), "vrooli_guard_check") {
		t.Fatalf("real hook boundary changed denial semantics or exposed a marker: err=%v stdout=%q stderr=%q", err, stdout.String(), stderr.String())
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
