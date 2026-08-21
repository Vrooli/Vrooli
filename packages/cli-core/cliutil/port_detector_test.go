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
	resetPortDetectorCache()
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

func TestDetectScenarioRuntimeStatusReadsStoppedLifecycleState(t *testing.T) {
	originalLookPath := lookPathFn
	resetPortDetectorCache()
	originalExec := execCommandContextFn
	t.Cleanup(func() {
		lookPathFn = originalLookPath
		execCommandContextFn = originalExec
	})
	lookPathFn = func(string) (string, error) { return "/custom/vrooli", nil }
	execCommandContextFn = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		if name != "/custom/vrooli" {
			t.Fatalf("expected resolved vrooli path, got %s", name)
		}
		want := []string{"--no-stale-check", "--json", "scenario", "status", "vrooli-bridge"}
		if !reflect.DeepEqual(args, want) {
			t.Fatalf("args = %#v, want %#v", args, want)
		}
		return exec.CommandContext(ctx, "bash", "-lc", "printf '{\"scenario\":{\"status\":\"stopped\"}}\\n'")
	}

	if got := DetectScenarioRuntimeStatus("vrooli-bridge")(); got != "stopped" {
		t.Fatalf("status = %q, want stopped", got)
	}
}

func TestRuntimeStatusFromJSONFallsBackToRuntime(t *testing.T) {
	if got := runtimeStatusFromJSON(`{"runtime":{"status":"running"}}`); got != "running" {
		t.Fatalf("status = %q, want running", got)
	}
	if got := runtimeStatusFromJSON("not-json"); got != "" {
		t.Fatalf("invalid JSON status = %q, want empty", got)
	}
}

func TestDetectPortFromVrooliFallsBackToBareCommand(t *testing.T) {
	originalLookPath := lookPathFn
	resetPortDetectorCache()
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

func TestDetectPortFromVrooliRoutesToShadowWhenShadowed(t *testing.T) {
	clearInstanceOverrides(t)
	t.Setenv(EnvShadowScenarios, "agent-manager")

	originalLookPath := lookPathFn
	resetPortDetectorCache()
	originalExec := execCommandContextFn
	t.Cleanup(func() {
		lookPathFn = originalLookPath
		execCommandContextFn = originalExec
	})
	lookPathFn = func(string) (string, error) { return "vrooli", nil }

	var gotTarget string
	execCommandContextFn = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		// args: --no-stale-check --json scenario port <target> API_PORT
		gotTarget = args[len(args)-2]
		return exec.CommandContext(ctx, "bash", "-lc", "printf '{\"port\":19001}\\n'")
	}

	port := DetectPortFromVrooli("agent-manager", "API_PORT")()
	if gotTarget != "agent-manager@shadow" {
		t.Fatalf("expected shadow target, got %q", gotTarget)
	}
	if port != "19001" {
		t.Fatalf("expected 19001, got %q", port)
	}
}

func TestDetectPortFromVrooliFallsBackToLiveWhenShadowMissing(t *testing.T) {
	clearInstanceOverrides(t)
	t.Setenv(EnvShadowScenarios, "swarm-manager")
	// Fresh warn-dedup so the fallback is exercisable.
	shadowFallbackWarned.Delete("swarm-manager")

	originalLookPath := lookPathFn
	resetPortDetectorCache()
	originalExec := execCommandContextFn
	t.Cleanup(func() {
		lookPathFn = originalLookPath
		execCommandContextFn = originalExec
	})
	lookPathFn = func(string) (string, error) { return "vrooli", nil }

	var targets []string
	execCommandContextFn = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		target := args[len(args)-2]
		targets = append(targets, target)
		if target == "swarm-manager@shadow" {
			// Simulate "not found": non-zero exit yields empty port.
			return exec.CommandContext(ctx, "bash", "-lc", "exit 1")
		}
		return exec.CommandContext(ctx, "bash", "-lc", "printf '{\"port\":20002}\\n'")
	}

	port := DetectPortFromVrooli("swarm-manager", "API_PORT")()
	if len(targets) != 2 || targets[0] != "swarm-manager@shadow" || targets[1] != "swarm-manager" {
		t.Fatalf("expected shadow-then-live lookups, got %v", targets)
	}
	if port != "20002" {
		t.Fatalf("expected live fallback port 20002, got %q", port)
	}
}
