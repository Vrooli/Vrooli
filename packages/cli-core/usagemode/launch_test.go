package usagemode

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func install(t *testing.T, runner string, s RunnerSettings) {
	t.Helper()
	useHome(t)
	m := mode("um-1", Scope{}, t0.Add(-time.Hour), tp(t0.Add(time.Hour)))
	m.Runners = map[string]RunnerSettings{runner: s}
	if err := Write(File{Modes: []Mode{m}}); err != nil {
		t.Fatal(err)
	}
}

var claudeSettings = RunnerSettings{
	ModelMap: map[string]string{"opus": "claude-sonnet-5", "sonnet": "claude-haiku-5-5"}, SessionModel: "claude-sonnet-5",
	Effort: "low", ContextLimitTokens: 150000,
}

func TestNoModeIsByteIdentical(t *testing.T) {
	useHome(t)
	args, env := []string{"exec", "-m", "x", "hi"}, []string{"A=1"}
	for _, runner := range []string{"codex", "claude", "grok"} {
		p := LaunchPlan(runner, args, env, t0)
		if p.Applied || p.ModeID != "" || p.Env != nil || !reflect.DeepEqual(p.Args, args) {
			t.Fatalf("%s: %+v", runner, p)
		}
	}
}

func TestClaudeEnv(t *testing.T) {
	install(t, "claude", claudeSettings)
	p := LaunchPlan("claude", []string{"-p", "hi"}, []string{"PATH=/bin"}, t0)
	want := []string{
		"ANTHROPIC_MODEL=claude-sonnet-5", "ANTHROPIC_DEFAULT_OPUS_MODEL=claude-sonnet-5", "ANTHROPIC_DEFAULT_SONNET_MODEL=claude-haiku-5-5",
		"CLAUDE_CODE_EFFORT_LEVEL=low", "CLAUDE_CODE_AUTO_COMPACT_WINDOW=150000", "VROOLI_USAGE_MODE=um-1",
	}
	if !p.Applied || p.ModeID != "um-1" || !reflect.DeepEqual(p.Env, want) || !reflect.DeepEqual(p.Args, []string{"-p", "hi"}) {
		t.Fatalf("%+v", p)
	}
	// claude-code resource name reaches the same mode.
	if !LaunchPlan("claude-code", nil, nil, t0).Applied {
		t.Fatal("claude-code runner name")
	}
}

func TestClaudeSkips(t *testing.T) {
	install(t, "claude", claudeSettings)
	for _, flag := range []string{"--resume", "-r", "--continue", "-c", "--resume=abc"} {
		if p := LaunchPlan("claude", []string{flag, "id"}, nil, t0); p.Applied || p.Env != nil {
			t.Errorf("%s must keep the session settings: %+v", flag, p)
		}
	}
	p := LaunchPlan("claude", nil, []string{"ANTHROPIC_MODEL=mine", "CLAUDE_CODE_EFFORT_LEVEL=high"}, t0)
	joined := strings.Join(p.Env, " ")
	if strings.Contains(joined, "ANTHROPIC_MODEL=") || strings.Contains(joined, "ANTHROPIC_DEFAULT_") || strings.Contains(joined, "EFFORT") {
		t.Fatalf("user env must win per variable: %v", p.Env)
	}
	small := claudeSettings
	small.ContextLimitTokens = 50000
	install(t, "claude", small)
	if !strings.Contains(strings.Join(LaunchPlan("claude", nil, nil, t0).Env, " "), "CLAUDE_CODE_AUTO_COMPACT_WINDOW=100000") {
		t.Fatal("limits under Claude's 100k minimum are raised to the minimum, not dropped")
	}
	if got, clamped := ClaudeContextLimit(50000); got != 100000 || !clamped {
		t.Fatalf("ClaudeContextLimit(50000) = %d, %v", got, clamped)
	}
	if got, clamped := ClaudeContextLimit(0); got != 0 || clamped {
		t.Fatalf("zero must stay unset: %d, %v", got, clamped)
	}
}

func TestClaudeExplicitModelWins(t *testing.T) {
	install(t, "claude", claudeSettings)
	for _, args := range [][]string{{"--model", "opus"}, {"--model=opus", "-p", "hi"}} {
		p := LaunchPlan("claude", args, nil, t0)
		if p.Applied || p.Env != nil || !reflect.DeepEqual(p.Args, args) {
			t.Fatalf("%v: %+v", args, p)
		}
	}
}

func TestExpiredModeIsInert(t *testing.T) {
	install(t, "claude", claudeSettings)
	if LaunchPlan("claude", nil, nil, t0.Add(time.Hour)).Applied {
		t.Fatal("expired mode applied")
	}
}

var codexSettings = RunnerSettings{SessionModel: "gpt-6-luna", Effort: "low", ContextLimitTokens: 200000}

func TestCodexArgs(t *testing.T) {
	prefix := []string{"-c", "model=gpt-6-luna", "-c", "model_reasoning_effort=low", "-c", "model_auto_compact_token_limit=200000"}
	cases := []struct {
		name string
		args []string
		want []string // nil means unchanged
	}{
		{"interactive", nil, prefix},
		{"prompt", []string{"fix it"}, prefix},
		{"flags then prompt", []string{"-C", "/tmp", "--search", "fix"}, prefix},
		{"exec", []string{"exec", "do"}, prefix},
		{"exec alias", []string{"e", "do"}, prefix},
		{"-m", []string{"-m", "gpt-6.1-terra", "exec", "x"}, nil},
		{"--model=", []string{"--model=gpt-6.1-terra"}, nil},
		{"-c model=", []string{"-c", "model=other", "exec"}, nil},
		{"--config model=", []string{"--config=model=other"}, nil},
		{"--oss", []string{"--oss", "exec", "x"}, nil},
		{"-p profile", []string{"-p", "work", "exec", "x"}, nil},
		{"--profile=", []string{"--profile=work"}, nil},
		{"-c model_provider=", []string{"-c", "model_provider=ollama", "exec", "x"}, nil},
		{"exec -o file resume", []string{"exec", "-o", "f", "resume", "id"}, nil},
		{"exec --output-last-message resume", []string{"exec", "--output-last-message", "f", "resume"}, nil},
		{"exec -o file prompt", []string{"exec", "-o", "f", "do it"}, prefix},
		{"-c other keeps model", []string{"-c", "sandbox_mode=x"}, prefix},
		{"-c explicit effort", []string{"-c", "model_reasoning_effort=high"}, append(append([]string{}, prefix[:2]...), prefix[4:]...)},
		{"resume", []string{"resume", "--last"}, nil},
		{"exec resume", []string{"exec", "resume", "id"}, nil},
		{"flag before resume", []string{"-C", "/x", "resume"}, nil},
		{"app-server", []string{"app-server"}, nil},
		{"login", []string{"login"}, nil},
		{"mcp", []string{"mcp", "list"}, nil},
		{"review", []string{"review"}, nil},
		{"fork", []string{"fork"}, nil},
	}
	install(t, "codex", codexSettings)
	for _, c := range cases {
		args := append([]string(nil), c.args...)
		p := LaunchPlan("codex", args, nil, t0)
		if !reflect.DeepEqual(args, c.args) {
			t.Errorf("%s: input mutated", c.name)
		}
		if c.want == nil {
			if p.Applied || !reflect.DeepEqual(p.Args, c.args) || p.Env != nil {
				t.Errorf("%s: want unchanged, got %+v", c.name, p)
			}
			continue
		}
		want := append(append([]string{}, c.want...), c.args...)
		if !p.Applied || !reflect.DeepEqual(p.Args, want) || !reflect.DeepEqual(p.Env, []string{"VROOLI_USAGE_MODE=um-1"}) {
			t.Errorf("%s: got %+v, want args %v", c.name, p, want)
		}
	}
}

func TestOtherRunnersUnchanged(t *testing.T) {
	install(t, "codex", codexSettings)
	args := []string{"run"}
	if p := LaunchPlan("opencode", args, nil, t0); p.Applied || !reflect.DeepEqual(p.Args, args) {
		t.Fatalf("%+v", p)
	}
}

func writeHarnessConfig(t *testing.T, envDir, name, body string) {
	t.Helper()
	dir := os.Getenv(envDir)
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestCallerANTHROPICModelSkipsAliasRemaps(t *testing.T) {
	install(t, "claude", claudeSettings)
	p := LaunchPlan("claude", []string{"-p", "hi"}, []string{"ANTHROPIC_MODEL=opus"}, t0)
	if p.Applied || p.Env != nil {
		t.Fatalf("a caller-set ANTHROPIC_MODEL is explicit; alias remaps would rewrite it: %+v", p)
	}
}

func TestSubagentModelIsNeverForced(t *testing.T) {
	install(t, "claude", claudeSettings)
	if strings.Contains(strings.Join(LaunchPlan("claude", nil, nil, t0).Env, " "), "CLAUDE_CODE_SUBAGENT_MODEL") {
		t.Fatal("forcing one sub-agent model would raise a Haiku sub-agent")
	}
}

func TestSessionModelNeverRaisesTheConfiguredDefault(t *testing.T) {
	settings := RunnerSettings{SessionModel: "claude-sonnet-5", ModelMap: map[string]string{"opus": "claude-sonnet-5", "claude-opus-5": "claude-sonnet-5"}}
	cases := []struct {
		name, config string
		wantModel    bool
	}{
		{"unset", ``, true},
		{"default is lowered by the map", `{"model":"opus"}`, true},
		{"default with context suffix", `{"model":"opus[1m]"}`, true},
		{"haiku default is lower already", `{"model":"haiku"}`, false},
		{"unmapped default", `{"model":"claude-haiku-5-5"}`, false},
		{"settings env model", `{"env":{"ANTHROPIC_MODEL":"haiku"}}`, false},
		{"malformed settings fail open to no change", `{`, false},
	}
	for _, c := range cases {
		install(t, "claude", settings)
		if c.config != "" {
			writeHarnessConfig(t, "CLAUDE_CONFIG_DIR", "settings.json", c.config)
		}
		got := strings.Contains(strings.Join(LaunchPlan("claude", nil, nil, t0).Env, " "), "ANTHROPIC_MODEL=")
		if got != c.wantModel {
			t.Errorf("claude %s: session model set = %v, want %v", c.name, got, c.wantModel)
		}
	}
	codex := RunnerSettings{SessionModel: "gpt-6-luna", ModelMap: map[string]string{"gpt-6.1-terra": "gpt-6-luna"}}
	codexCases := []struct {
		name, config string
		wantModel    bool
	}{
		{"unset", ``, true},
		{"mapped default", "model = \"gpt-6.1-terra\"\nmodel_reasoning_effort = \"high\"\n", true},
		{"already lower", "model = \"gpt-5.6-luna\"\n", false},
		{"model only inside a table is not the default", "[profiles.x]\nmodel = \"gpt-5.6-luna\"\n", true},
	}
	for _, c := range codexCases {
		install(t, "codex", codex)
		if c.config != "" {
			writeHarnessConfig(t, "CODEX_HOME", "config.toml", c.config)
		}
		got := strings.Contains(strings.Join(LaunchPlan("codex", nil, nil, t0).Args, " "), "model=gpt-6-luna")
		if got != c.wantModel {
			t.Errorf("codex %s: session model set = %v, want %v", c.name, got, c.wantModel)
		}
	}
}

func TestUnreadableHarnessConfigChangesNothing(t *testing.T) {
	install(t, "codex", RunnerSettings{SessionModel: "gpt-6-luna"})
	// A directory where config.toml should be is "exists but unreadable".
	if err := os.Mkdir(filepath.Join(os.Getenv("CODEX_HOME"), "config.toml"), 0o700); err != nil {
		t.Fatal(err)
	}
	if p := LaunchPlan("codex", nil, nil, t0); p.Applied {
		t.Fatalf("unreadable default must fail open to no change: %+v", p)
	}
}

func TestCodexExplicitModelSkipsEveryShimOverride(t *testing.T) {
	// Agent Manager launches pass -m (exempt roles too); the shim must add no
	// effort or context limit on top of the run's own settings.
	install(t, "codex", codexSettings)
	args := []string{"exec", "-m", "gpt-6.1-terra", "go"}
	if p := LaunchPlan("codex", args, nil, t0); p.Applied || !reflect.DeepEqual(p.Args, args) {
		t.Fatalf("%+v", p)
	}
}
