package cliutil

import (
	"context"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/vrooli/repo-contract-go/cliinvoke"
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

func BenchmarkLookupScenarioPortPeerRecord(b *testing.B) {
	oldPeer, oldRegistry, oldRunner := peerRecordLookupFn, runtimeRegistryLookupFn, portLookupRunner
	b.Cleanup(func() {
		peerRecordLookupFn, runtimeRegistryLookupFn, portLookupRunner = oldPeer, oldRegistry, oldRunner
	})
	peerRecordLookupFn = func(string, string) ScenarioPortOutcome { return ScenarioPortOutcome{Port: "18443"} }
	runtimeRegistryLookupFn = func(context.Context, string, string) ScenarioPortOutcome {
		b.Fatal("registry should not run")
		return ScenarioPortOutcome{}
	}
	portLookupRunner = func(context.Context, string, string) ScenarioPortOutcome {
		b.Fatal("CLI should not run")
		return ScenarioPortOutcome{}
	}
	policy := PortCachePolicy{}
	for i := 0; i < b.N; i++ {
		if got := LookupScenarioPort(context.Background(), "agent-manager", "API_PORT", policy); got.Port != "18443" {
			b.Fatal(got)
		}
	}
}

func TestDetectPortFromVrooliPassesNoGlobalFlags(t *testing.T) {
	originalLookPath := lookPathFn
	resetPortDetectorCache()
	originalExec := runVrooliFn
	originalPeer, originalRegistry := peerRecordLookupFn, runtimeRegistryLookupFn
	t.Cleanup(func() {
		lookPathFn = originalLookPath
		runVrooliFn = originalExec
		peerRecordLookupFn, runtimeRegistryLookupFn = originalPeer, originalRegistry
		resetPortDetectorCache()
	})
	peerRecordLookupFn = func(string, string) ScenarioPortOutcome { return ScenarioPortOutcome{Err: os.ErrNotExist} }
	runtimeRegistryLookupFn = func(context.Context, string, string) ScenarioPortOutcome {
		return ScenarioPortOutcome{Err: os.ErrNotExist}
	}

	originalHome := userHomeFn
	userHomeFn = func() (string, error) { return t.TempDir(), nil }
	t.Cleanup(func() { userHomeFn = originalHome })
	lookPathFn = func(file string) (string, error) {
		if file != "vrooli" {
			t.Fatalf("expected vrooli lookup, got %s", file)
		}
		return "/custom/vrooli", nil
	}

	called := false
	runVrooliFn = func(ctx context.Context, inv cliinvoke.Invocation) cliinvoke.Result {
		name, args := inv.Binary, inv.Args
		called = true
		if name != "/custom/vrooli" {
			t.Fatalf("expected resolved vrooli path, got %s", name)
		}
		want := []string{"scenario", "port", "test-genie", "API_PORT", "--json"}
		if !reflect.DeepEqual(args, want) {
			t.Fatalf("args = %#v, want %#v", args, want)
		}
		if strings.HasPrefix(args[0], "--") {
			t.Fatalf("args = %#v start with a global flag", args)
		}
		return cliinvoke.Result{Stdout: []byte("{\"success\":true,\"port\":15422}\n")}
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
	originalExec := runVrooliFn
	t.Cleanup(func() {
		lookPathFn = originalLookPath
		runVrooliFn = originalExec
	})
	originalHome := userHomeFn
	userHomeFn = func() (string, error) { return t.TempDir(), nil }
	t.Cleanup(func() { userHomeFn = originalHome })
	lookPathFn = func(string) (string, error) { return "/custom/vrooli", nil }
	runVrooliFn = func(ctx context.Context, inv cliinvoke.Invocation) cliinvoke.Result {
		name, args := inv.Binary, inv.Args
		if name != "/custom/vrooli" {
			t.Fatalf("expected resolved vrooli path, got %s", name)
		}
		want := []string{"scenario", "status", "vrooli-bridge", "--json"}
		if !reflect.DeepEqual(args, want) {
			t.Fatalf("args = %#v, want %#v", args, want)
		}
		return cliinvoke.Result{Stdout: []byte("{\"scenario\":{\"status\":\"stopped\"}}\n")}
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
	originalExec := runVrooliFn
	t.Cleanup(func() {
		lookPathFn = originalLookPath
		runVrooliFn = originalExec
	})

	lookPathFn = func(file string) (string, error) {
		return "", errors.New("executable file not found in $PATH")
	}

	runVrooliFn = func(ctx context.Context, inv cliinvoke.Invocation) cliinvoke.Result {
		name := inv.Binary
		if name != "vrooli" {
			t.Fatalf("expected bare vrooli fallback, got %s", name)
		}
		return cliinvoke.Result{Stdout: []byte("API_PORT=18847\n")}
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
	originalExec := runVrooliFn
	originalPeer, originalRegistry := peerRecordLookupFn, runtimeRegistryLookupFn
	t.Cleanup(func() {
		lookPathFn = originalLookPath
		runVrooliFn = originalExec
		peerRecordLookupFn, runtimeRegistryLookupFn = originalPeer, originalRegistry
		resetPortDetectorCache()
	})
	peerRecordLookupFn = func(string, string) ScenarioPortOutcome { return ScenarioPortOutcome{Err: os.ErrNotExist} }
	runtimeRegistryLookupFn = func(context.Context, string, string) ScenarioPortOutcome {
		return ScenarioPortOutcome{Err: os.ErrNotExist}
	}
	lookPathFn = func(string) (string, error) { return "vrooli", nil }

	var gotTarget string
	runVrooliFn = func(ctx context.Context, inv cliinvoke.Invocation) cliinvoke.Result {
		args := inv.Args
		// args: scenario port <target> API_PORT --json
		gotTarget = args[len(args)-3]
		return cliinvoke.Result{Stdout: []byte("{\"port\":19001}\n")}
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
	originalExec := runVrooliFn
	originalPeer, originalRegistry := peerRecordLookupFn, runtimeRegistryLookupFn
	t.Cleanup(func() {
		lookPathFn = originalLookPath
		runVrooliFn = originalExec
		peerRecordLookupFn, runtimeRegistryLookupFn = originalPeer, originalRegistry
		resetPortDetectorCache()
	})
	peerRecordLookupFn = func(string, string) ScenarioPortOutcome { return ScenarioPortOutcome{Err: os.ErrNotExist} }
	runtimeRegistryLookupFn = func(context.Context, string, string) ScenarioPortOutcome {
		return ScenarioPortOutcome{Err: os.ErrNotExist}
	}
	lookPathFn = func(string) (string, error) { return "vrooli", nil }

	var targets []string
	runVrooliFn = func(ctx context.Context, inv cliinvoke.Invocation) cliinvoke.Result {
		args := inv.Args
		target := args[len(args)-3]
		targets = append(targets, target)
		if target == "swarm-manager@shadow" {
			// Simulate "not found": non-zero exit yields empty port.
			return cliinvoke.Result{Class: cliinvoke.Lifecycle, ExitCode: 1, Err: errors.New("exit status 1")}
		}
		return cliinvoke.Result{Stdout: []byte("{\"port\":20002}\n")}
	}

	port := DetectPortFromVrooli("swarm-manager", "API_PORT")()
	if len(targets) != 2 || targets[0] != "swarm-manager@shadow" || targets[1] != "swarm-manager" {
		t.Fatalf("expected shadow-then-live lookups, got %v", targets)
	}
	if port != "20002" {
		t.Fatalf("expected live fallback port 20002, got %q", port)
	}
}
