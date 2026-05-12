package cliutil

import (
	"context"
	"os/exec"
	"reflect"
	"testing"
)

func TestSanitizePortOutput(t *testing.T) {
	cases := []struct {
		name   string
		input  string
		expect string
	}{
		{name: "numeric", input: "3000\n", expect: "3000"},
		{name: "with label", input: "api port: 4567", expect: "4567"},
		{name: "no digits", input: "not running", expect: ""},
		{name: "empty", input: "", expect: ""},
	}

	for _, tc := range cases {
		if got := sanitizePortOutput(tc.input); got != tc.expect {
			t.Fatalf("%s: expected %q, got %q", tc.name, tc.expect, got)
		}
	}
}

func TestDetectPortFromVrooliUsesNoStaleCheck(t *testing.T) {
	originalLookPath := lookPathFn
	originalExec := execCommandContextFn
	t.Cleanup(func() {
		lookPathFn = originalLookPath
		execCommandContextFn = originalExec
	})

	lookPathFn = func(file string) (string, error) {
		if file != "vrooli" {
			t.Fatalf("expected vrooli lookup, got %s", file)
		}
		return "/custom/vrooli", nil
	}

	called := false
	execCommandContextFn = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		called = true
		if name != "/custom/vrooli" {
			t.Fatalf("expected resolved vrooli path, got %s", name)
		}
		want := []string{"--no-stale-check", "--json", "scenario", "port", "test-genie", "API_PORT"}
		if !reflect.DeepEqual(args, want) {
			t.Fatalf("args = %#v, want %#v", args, want)
		}
		return exec.CommandContext(ctx, "bash", "-lc", "printf '{\"success\":true,\"port\":15422}\\n'")
	}

	port := DetectPortFromVrooli("test-genie", "API_PORT")()
	if !called {
		t.Fatal("expected port detector to execute vrooli")
	}
	if port != "15422" {
		t.Fatalf("expected 15422, got %q", port)
	}
}

func TestDetectPortFromVrooliFallsBackToBareCommand(t *testing.T) {
	originalLookPath := lookPathFn
	originalExec := execCommandContextFn
	t.Cleanup(func() {
		lookPathFn = originalLookPath
		execCommandContextFn = originalExec
	})

	lookPathFn = func(file string) (string, error) {
		return "", exec.ErrNotFound
	}

	execCommandContextFn = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		if name != "vrooli" {
			t.Fatalf("expected bare vrooli fallback, got %s", name)
		}
		return exec.CommandContext(ctx, "bash", "-lc", "printf 'API_PORT=18847\\n'")
	}

	port := DetectPortFromVrooli("scenario-auditor", "API_PORT")()
	if port != "18847" {
		t.Fatalf("expected 18847, got %q", port)
	}
}
