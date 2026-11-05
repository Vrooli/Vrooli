package cliutil

import (
	"context"
	"io"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/cli-core/usagemode"
)

func writeTestMode(t *testing.T, runner string, settings usagemode.RunnerSettings, expires time.Time) {
	t.Helper()
	t.Setenv("VROOLI_HOME", t.TempDir())
	t.Setenv("CODEX_HOME", t.TempDir()) // no configured default: the mode may set the session model
	t.Setenv("CLAUDE_CONFIG_DIR", t.TempDir())
	now := time.Now()
	err := usagemode.Write(usagemode.File{Modes: []usagemode.Mode{{
		ID: "um-test", StartedAt: now.Add(-time.Minute), ExpiresAt: &expires, Until: usagemode.Until{Kind: usagemode.UntilTime},
		Runners: map[string]usagemode.RunnerSettings{runner: settings},
	}}})
	if err != nil {
		t.Fatal(err)
	}
}

func TestApplyUsageModeNoModeIsIdentical(t *testing.T) {
	t.Setenv("VROOLI_HOME", t.TempDir())
	args, env := []string{"exec", "x"}, []string{"A=1", "B=2"}
	gotArgs, gotEnv := ApplyUsageMode("codex", args, env)
	if !reflect.DeepEqual(gotArgs, args) || !reflect.DeepEqual(gotEnv, env) {
		t.Fatalf("%v %v", gotArgs, gotEnv)
	}
}

func TestApplyUsageModeCodexAndClaude(t *testing.T) {
	writeTestMode(t, "codex", usagemode.RunnerSettings{SessionModel: "gpt-6-luna"}, time.Now().Add(time.Hour))
	args, env := ApplyUsageMode("codex", []string{"exec", "x"}, []string{"A=1"})
	if !reflect.DeepEqual(args, []string{"-c", "model=gpt-6-luna", "exec", "x"}) || !reflect.DeepEqual(env, []string{"A=1", "VROOLI_USAGE_MODE=um-test"}) {
		t.Fatalf("%v %v", args, env)
	}
	if args, _ := ApplyUsageMode("codex", []string{"-m", "gpt-6.1-terra"}, nil); strings.Contains(strings.Join(args, " "), "gpt-6-luna") {
		t.Fatalf("explicit model lost: %v", args)
	}
	writeTestMode(t, "claude", usagemode.RunnerSettings{SessionModel: "claude-sonnet-5"}, time.Now().Add(time.Hour))
	_, env = ApplyUsageMode("claude", []string{"-p", "x"}, []string{"ANTHROPIC_MODEL=mine", "A=1"})
	if !reflect.DeepEqual(env, []string{"ANTHROPIC_MODEL=mine", "A=1"}) {
		t.Fatalf("%v", env)
	}
}

func TestApplyUsageModeExpiredIsInert(t *testing.T) {
	writeTestMode(t, "codex", usagemode.RunnerSettings{SessionModel: "gpt-6-luna"}, time.Now().Add(-time.Second))
	args := []string{"exec"}
	if got, env := ApplyUsageMode("codex", args, []string{"A=1"}); !reflect.DeepEqual(got, args) || len(env) != 1 {
		t.Fatalf("%v %v", got, env)
	}
}

// The real launch path must reach the child with the mode applied.
func TestLaunchCodingAgentAppliesUsageMode(t *testing.T) {
	t.Setenv(AgentManagerIdentityTokenEnv, "")
	writeTestMode(t, "codex", usagemode.RunnerSettings{SessionModel: "gpt-6-luna"}, time.Now().Add(time.Hour))
	var gotArgs, gotEnv []string
	err := LaunchCodingAgent(context.Background(), AgentLaunchRequest{
		Agent: "codex", APIBase: "http://127.0.0.1:1", Args: []string{"exec", "hi"}, AttachTimeout: 50 * time.Millisecond,
		LookPath: func(string) (string, error) { return "/safe/fixture/codex", nil },
		RunChild: func(_ context.Context, _ string, args, environment []string, _ io.Reader, _, _ io.Writer) error {
			gotArgs, gotEnv = args, environment
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(gotArgs, []string{"-c", "model=gpt-6-luna", "exec", "hi"}) || environmentValue(gotEnv, usagemode.EnvMode) != "um-test" {
		t.Fatalf("child saw %v mode=%q", gotArgs, environmentValue(gotEnv, usagemode.EnvMode))
	}
}
