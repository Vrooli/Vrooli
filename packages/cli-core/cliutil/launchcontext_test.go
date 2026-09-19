package cliutil

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveLaunchContextRecoversFromInvalidCurrentDirectory(t *testing.T) {
	current, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root := validRepo(current)
	if root == "" {
		t.Fatalf("could not resolve repository root from %q", current)
	}

	pointer := filepath.Join(t.TempDir(), "source-root")
	if err := os.WriteFile(pointer, []byte(root+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(LaunchSourcePointerEnv, pointer)
	previous := currentWorkingDirectory
	currentWorkingDirectory = func() (string, error) {
		return "", errors.New("permission denied")
	}
	t.Cleanup(func() { currentWorkingDirectory = previous })

	context, err := ResolveLaunchContext(LaunchContextRequest{
		Environment: []string{LaunchSourcePointerEnv + "=" + pointer},
	})
	if err != nil {
		t.Fatalf("ResolveLaunchContext() error = %v", err)
	}
	if context.WorkingDir != root || context.ProjectRoot != root {
		t.Fatalf("context = %+v, want root %q", context, root)
	}
	if context.Source != "source-root-pointer" {
		t.Fatalf("context source = %q, want source-root-pointer", context.Source)
	}
}

func TestResolveLaunchContextRejectsInvalidExplicitDirectory(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing")
	_, err := ResolveLaunchContext(LaunchContextRequest{WorkingDir: missing})
	if err == nil || !strings.Contains(err.Error(), "resolve requested working directory") {
		t.Fatalf("error = %v, want actionable invalid-directory error", err)
	}
}

func TestPrepareLaunchEnvironmentPrependsPortableToolPath(t *testing.T) {
	toolDir := t.TempDir()
	prepared := PrepareLaunchEnvironment([]string{"PATH=/system/bin", "SAFE=value"}, LaunchContext{ToolDirs: []string{toolDir}})
	path := environmentValue(prepared, "PATH")
	if !strings.HasPrefix(path, toolDir+string(os.PathListSeparator)) {
		t.Fatalf("PATH = %q, want %q first", path, toolDir)
	}
	if environmentValue(prepared, "SAFE") != "value" {
		t.Fatalf("safe environment value was not preserved: %v", prepared)
	}
}

func TestResolveLaunchContextUsesExplicitRuntimeBinaryDirectory(t *testing.T) {
	workingDir := t.TempDir()
	toolDir := t.TempDir()
	binary := filepath.Join(toolDir, "vrooli")
	if err := os.WriteFile(binary, []byte("fixture"), 0o755); err != nil {
		t.Fatal(err)
	}

	context, err := ResolveLaunchContext(LaunchContextRequest{
		WorkingDir:  workingDir,
		Environment: []string{LaunchRuntimeBinaryEnv + "=" + binary, "PATH=/system/bin"},
	})
	if err != nil {
		t.Fatalf("ResolveLaunchContext() error = %v", err)
	}
	prepared := PrepareLaunchEnvironment([]string{"PATH=/system/bin"}, context)
	if got := environmentValue(prepared, "PATH"); !strings.HasPrefix(got, toolDir+string(os.PathListSeparator)) {
		t.Fatalf("PATH = %q, want runtime binary directory %q first", got, toolDir)
	}
}

func TestRunNativeChildUsesResolvedWorkingDirectory(t *testing.T) {
	workingDir := t.TempDir()
	var stdout bytes.Buffer
	environment := append(os.Environ(), "CLI_CORE_LAUNCH_CONTEXT_HELPER=1")
	run := runNativeChild(AgentLaunchRequest{WorkingDir: workingDir}, filepath.Base(os.Args[0]))
	if err := run(context.Background(), os.Args[0], []string{"-test.run=TestLaunchContextHelper"}, environment, nil, &stdout, io.Discard); err != nil {
		t.Fatalf("runNativeChild() error = %v", err)
	}
	output := strings.TrimSpace(stdout.String())
	if firstLine, _, ok := strings.Cut(output, "\n"); ok {
		output = firstLine
	}
	got := strings.TrimPrefix(output, "CWD=")
	if got != workingDir {
		t.Fatalf("child working directory = %q, want %q", got, workingDir)
	}
}

func TestLaunchContextHelper(t *testing.T) {
	if os.Getenv("CLI_CORE_LAUNCH_CONTEXT_HELPER") != "1" {
		return
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	_, _ = os.Stdout.WriteString("CWD=" + cwd + "\n")
}
