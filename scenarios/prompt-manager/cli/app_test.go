package main

import (
	"io"
	"os"
	"strings"
	"testing"
)

func TestNewApp_APIPortEnvVars_DoNotIncludeGenericAPIPort(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}

	portEnvVars := app.core.APIBaseOptions().PortEnvVars
	if len(portEnvVars) == 0 {
		t.Fatal("expected APIPortEnvVars to be configured")
	}
	if portEnvVars[0] != "PROMPT_MANAGER_API_PORT" {
		t.Fatalf("first APIPortEnvVar = %q, want %q", portEnvVars[0], "PROMPT_MANAGER_API_PORT")
	}
	for _, key := range portEnvVars {
		if key == "API_PORT" {
			t.Fatalf("unexpected generic API_PORT in APIPortEnvVars: %v", portEnvVars)
		}
	}
}

func TestSkillHelpDoesNotRequireAPI(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}

	output := captureStdout(t, func() {
		if err := app.Run([]string{"skill", "--help"}); err != nil {
			t.Fatalf("Run() error = %v", err)
		}
	})
	if !strings.Contains(output, "prompt-manager skill <subcommand> [args]") {
		t.Fatalf("expected skill help output, got %q", output)
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v", err)
	}
	os.Stdout = w
	defer func() {
		os.Stdout = old
	}()

	done := make(chan string, 1)
	go func() {
		var builder strings.Builder
		_, _ = io.Copy(&builder, r)
		done <- builder.String()
	}()

	fn()
	_ = w.Close()
	return <-done
}
