package server

import (
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

type fakeCommand struct {
	stdout  string
	stderr  string
	waitErr error
	killed  bool
}

func (f *fakeCommand) StdoutPipe() (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader(f.stdout)), nil
}

func (f *fakeCommand) StderrPipe() (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader(f.stderr)), nil
}

func (f *fakeCommand) Start() error { return nil }

func (f *fakeCommand) Wait() error { return f.waitErr }

func (f *fakeCommand) Kill() error {
	f.killed = true
	return nil
}

func (f *fakeCommand) Process() *os.Process { return nil }

type captureFactory struct {
	cmd      agentCommand
	command  string
	args     []string
	options  commandOptions
	buildErr error
}

func (c *captureFactory) build(_ context.Context, name string, args []string, opts commandOptions) (agentCommand, error) {
	c.command = name
	c.args = append([]string(nil), args...)
	c.options = opts
	if c.buildErr != nil {
		return nil, c.buildErr
	}
	return c.cmd, nil
}

func TestBuildAgentCommandArgsDefaults(t *testing.T) {
	tag, args := buildAgentCommandArgs(nil, "issue-123")
	if tag != "app-issue-tracker-issue-123" {
		t.Fatalf("unexpected tag %s", tag)
	}
	if len(args) == 0 || args[len(args)-1] != "-" {
		t.Fatalf("expected default args to include '-', got %v", args)
	}
}

func TestBuildClaudeEnvironmentSkipPermissions(t *testing.T) {
	settings := AgentSettings{MaxTurns: 5, AllowedTools: "Read", SkipPermissions: true}
	env := buildClaudeEnvironment([]string{"BASE=1"}, settings, "app-issue-tracker-100", "/tmp/transcript", "/tmp/last", 60)
	joined := strings.Join(env, ";")
	for _, key := range []string{"CODEX_AGENT_TAG=app-issue-tracker-100", "CODEX_SKIP_PERMISSIONS=true", "MAX_TURNS=5"} {
		if !strings.Contains(joined, key) {
			t.Fatalf("expected env to contain %s", key)
		}
	}
	if !strings.Contains(joined, "BASE=1") {
		t.Fatalf("expected base environment to be preserved")
	}
}

func TestExecuteClaudeCodeUsesCommandFactory(t *testing.T) {
	root := t.TempDir()
	resetAgentSettingsForTest()
	t.Cleanup(resetAgentSettingsForTest)

	settings := AgentSettings{
		MaxTurns:        10,
		AllowedTools:    "Read",
		TimeoutSeconds:  30,
		SkipPermissions: true,
		Provider:        "claude-code",
		CLICommand:      "resource-claude-code",
		Command:         []string{"run", "--tag", "{{TAG}}", "-"},
	}
	setAgentSettingsForTest(settings, root)

	cmd := &fakeCommand{stdout: "Investigation summary\nFinal response"}
	factory := &captureFactory{cmd: cmd}

	srv := &Server{
		config:         &Config{ScenarioRoot: root, VrooliRoot: root, WebsocketAllowedOrigins: []string{"*"}},
		commandFactory: factory.build,
	}

	start := time.Now()
	result, err := srv.executeClaudeCode(context.Background(), "prompt text", "issue-123", start, 15*time.Second)
	if err != nil {
		t.Fatalf("executeClaudeCode failed: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success, got %+v", result)
	}
	if result.LastMessage == "" {
		t.Fatalf("expected last message to be captured")
	}
	if factory.command != "resource-claude-code" {
		t.Fatalf("unexpected command %s", factory.command)
	}
	if len(factory.args) == 0 || factory.args[1] != "--tag" {
		t.Fatalf("expected tag flag in args, got %v", factory.args)
	}
	joinedEnv := strings.Join(factory.options.Env, ";")
	if !strings.Contains(joinedEnv, "CODEX_AGENT_TAG=app-issue-tracker-issue-123") {
		t.Fatalf("env missing CODEX_AGENT_TAG: %s", joinedEnv)
	}
	if factory.options.Dir != root {
		t.Fatalf("expected dir %s, got %s", root, factory.options.Dir)
	}
	if result.TranscriptPath == "" || result.LastMessagePath == "" {
		t.Fatalf("expected transcript paths to be set")
	}
	if _, err := os.Stat(result.TranscriptPath); err != nil {
		t.Fatalf("transcript file not created: %v", err)
	}
}

func TestExecuteClaudeCodeCommandFactoryError(t *testing.T) {
	root := t.TempDir()
	resetAgentSettingsForTest()
	t.Cleanup(resetAgentSettingsForTest)

	settings := AgentSettings{CLICommand: "resource-claude-code", Command: []string{"run"}, TimeoutSeconds: 30}
	setAgentSettingsForTest(settings, root)

	factory := &captureFactory{buildErr: errors.New("boom")}
	srv := &Server{
		config:         &Config{ScenarioRoot: root, VrooliRoot: root, WebsocketAllowedOrigins: []string{"*"}},
		commandFactory: factory.build,
	}

	if _, err := srv.executeClaudeCode(context.Background(), "prompt", "issue-1", time.Now(), 10*time.Second); err == nil {
		t.Fatalf("expected error from command factory")
	}
}
