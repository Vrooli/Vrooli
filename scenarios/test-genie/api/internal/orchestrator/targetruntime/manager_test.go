package targetruntime

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func TestEnsureRunningStartsPathAwareScenarioAndResolvesPorts(t *testing.T) {
	home := t.TempDir()
	scenarioDir := filepath.Join(t.TempDir(), "demo")
	if err := os.MkdirAll(scenarioDir, 0o755); err != nil {
		t.Fatal(err)
	}

	var calls [][]string
	manager := New("demo", scenarioDir).
		WithHome(home).
		WithProbes(func(context.Context, int) bool { return true }, func(int) bool { return true }).
		WithCommandRunner(func(ctx context.Context, dir string, env map[string]string, logWriter io.Writer, name string, args ...string) error {
			calls = append(calls, append([]string{name}, args...))
			writeRecord(t, home, "demo", "start-ui", 3001)
			writeRecord(t, home, "demo", "start-api", 4001)
			return nil
		})

	lease, err := manager.EnsureRunning(context.Background(), Needs{UI: true, API: true}, io.Discard)
	if err != nil {
		t.Fatalf("EnsureRunning: %v", err)
	}
	if !lease.Started {
		t.Fatal("expected manager to own started runtime")
	}
	if lease.URLs.UI != "http://127.0.0.1:3001" || lease.URLs.API != "http://127.0.0.1:4001" {
		t.Fatalf("unexpected urls: %#v", lease.URLs)
	}
	want := []string{"vrooli", "scenario", "start", "demo", "--clean-stale", "--path", scenarioDir}
	if !reflect.DeepEqual(calls[0], want) {
		t.Fatalf("start command = %#v, want %#v", calls[0], want)
	}
}

func TestEnsureRunningPreservesLifecycleFailureOutput(t *testing.T) {
	manager := New("demo", t.TempDir()).WithCommandRunner(func(ctx context.Context, dir string, env map[string]string, logWriter io.Writer, name string, args ...string) error {
		_, _ = fmt.Fprintln(logWriter, "ui/src/features/notes/NotesCard.tsx(134,11): error TS2322: Property 'tableTestId' does not exist")
		return errors.New("exit status 2")
	})

	_, err := manager.EnsureRunning(context.Background(), Needs{UI: true}, io.Discard)
	if err == nil {
		t.Fatal("EnsureRunning() error = nil, want lifecycle failure")
	}
	for _, want := range []string{"start target scenario demo", "exit status 2", "lifecycle start output", "TS2322", "tableTestId"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("EnsureRunning() error = %q, want %q", err, want)
		}
	}
}

func TestEnsureRunningDoesNotStartAlreadyRunningScenario(t *testing.T) {
	home := t.TempDir()
	writeRecord(t, home, "demo", "start-ui", 3001)

	var called bool
	manager := New("demo", "").
		WithHome(home).
		WithProbes(func(context.Context, int) bool { return true }, func(int) bool { return true }).
		WithCommandRunner(func(ctx context.Context, dir string, env map[string]string, logWriter io.Writer, name string, args ...string) error {
			called = true
			return nil
		})

	lease, err := manager.EnsureRunning(context.Background(), Needs{UI: true}, io.Discard)
	if err != nil {
		t.Fatalf("EnsureRunning: %v", err)
	}
	if lease.Started {
		t.Fatal("already-running scenario should not be owned by Test Genie")
	}
	if called {
		t.Fatal("start command should not run for already-running scenario")
	}
}

func TestRestartWithEnvUsesPathAwareRestartAndEnvOverrides(t *testing.T) {
	scenarioDir := filepath.Join(t.TempDir(), "demo")
	var gotArgs []string
	var gotEnv map[string]string
	manager := New("demo", scenarioDir).WithCommandRunner(func(ctx context.Context, dir string, env map[string]string, logWriter io.Writer, name string, args ...string) error {
		gotArgs = append([]string{name}, args...)
		gotEnv = env
		return nil
	})

	err := manager.RestartWithEnv(context.Background(), map[string]string{"DATABASE_URL": "postgres://temp"}, io.Discard)
	if err != nil {
		t.Fatalf("RestartWithEnv: %v", err)
	}
	want := []string{"vrooli", "scenario", "restart", "demo", "--clean-stale", "--path", scenarioDir}
	if !reflect.DeepEqual(gotArgs, want) {
		t.Fatalf("restart command = %#v, want %#v", gotArgs, want)
	}
	if gotEnv["DATABASE_URL"] != "postgres://temp" {
		t.Fatalf("env override not passed: %#v", gotEnv)
	}
}

func TestCommandEnvironmentScrubsCallerRuntimePorts(t *testing.T) {
	t.Setenv("UI_PORT", "21223")
	t.Setenv("API_PORT", "15421")
	t.Setenv("CUSTOM_TOOL_URL", "http://localhost:4321")

	env := commandEnvironment(map[string]string{"DATABASE_URL": "postgres://temp"})
	for _, item := range env {
		if item == "UI_PORT=21223" || item == "API_PORT=15421" {
			t.Fatalf("caller runtime port leaked into command environment: %v", env)
		}
	}
	if !containsEnv(env, "CUSTOM_TOOL_URL=http://localhost:4321") {
		t.Fatalf("unrelated env should be preserved: %v", env)
	}
	if !containsEnv(env, "DATABASE_URL=postgres://temp") {
		t.Fatalf("override env missing: %v", env)
	}
}

func TestCleanupStopsOnlyOwnedRuntime(t *testing.T) {
	var calls int
	manager := New("demo", "").WithCommandRunner(func(ctx context.Context, dir string, env map[string]string, logWriter io.Writer, name string, args ...string) error {
		calls++
		return nil
	})
	if err := manager.Cleanup(context.Background(), Lease{}, io.Discard); err != nil {
		t.Fatalf("Cleanup unowned: %v", err)
	}
	if calls != 0 {
		t.Fatalf("unowned cleanup ran %d command(s)", calls)
	}
	if err := manager.Cleanup(context.Background(), Lease{Started: true}, io.Discard); err != nil {
		t.Fatalf("Cleanup owned: %v", err)
	}
	if calls != 1 {
		t.Fatalf("owned cleanup ran %d command(s), want 1", calls)
	}
}

func containsEnv(env []string, want string) bool {
	for _, item := range env {
		if item == want {
			return true
		}
	}
	return false
}

func writeRecord(t *testing.T, home, scenario, step string, port int) {
	t.Helper()
	dir := filepath.Join(home, ".vrooli", "processes", "scenarios", scenario)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := []byte(`{"pid":12345,"port":` + strconv.Itoa(port) + `,"step":"` + step + `"}`)
	if err := os.WriteFile(filepath.Join(dir, step+".json"), content, 0o644); err != nil {
		t.Fatal(err)
	}
}
