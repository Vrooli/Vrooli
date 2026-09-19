package cliutil

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestStaleCheckerRebuildInheritsFloor(t *testing.T) {
	t.Setenv("GOFLAGS", "-mod=mod")
	t.Setenv("GOMAXPROCS", "")
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "packages", "cli-core"), 0o755); err != nil {
		t.Fatal(err)
	}
	var captured []string
	checker := &StaleChecker{
		LookPathFunc:  func(string) (string, error) { return "/usr/bin/go", nil },
		CommandRunner: func(cmd *exec.Cmd) error { captured = cmd.Env; return nil },
		Logger:        func(string, ...interface{}) {},
		Reexec:        func(string, []string) error { return nil },
	}
	if !checker.autoRebuild(FreshnessSpec{SourceRoot: filepath.Join(root, "scenarios", "x", "cli"), ContextRoot: root}, "fp") {
		t.Fatal("autoRebuild returned false")
	}
	goflags := environmentValue(captured, "GOFLAGS")
	if !strings.HasPrefix(goflags, "-mod=mod ") || !strings.Contains(goflags, "-p=") || environmentValue(captured, "GOMAXPROCS") == "" {
		t.Fatalf("rebuild env lacks the floor: %v", captured)
	}
}

func TestAgentLaunchEnvCarriesFloor(t *testing.T) {
	t.Setenv(AgentManagerIdentityTokenEnv, "")
	t.Setenv("GOFLAGS", "-mod=mod")
	t.Setenv("GOMAXPROCS", "")
	var captured []string
	_, err := LaunchCodingAgentResult(context.Background(), AgentLaunchRequest{
		Agent:    "codex",
		APIBase:  "http://127.0.0.1:1",
		LookPath: func(string) (string, error) { return "/safe/fixture/codex", nil },
		RunChild: func(_ context.Context, _ string, _ []string, environment []string, _ io.Reader, _, _ io.Writer) error {
			captured = environment
			return nil
		},
		AttachTimeout: 50 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("LaunchCodingAgentResult() error = %v", err)
	}
	goflags := environmentValue(captured, "GOFLAGS")
	if !strings.HasPrefix(goflags, "-mod=mod ") || !strings.Contains(goflags, "-p=") || environmentValue(captured, "GOMAXPROCS") == "" {
		t.Fatalf("agent env lacks the floor: GOFLAGS=%q GOMAXPROCS=%q", goflags, environmentValue(captured, "GOMAXPROCS"))
	}
}
