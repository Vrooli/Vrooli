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
	"time"

	"github.com/vrooli/api-core/demand"
)

type demandFixture struct {
	acquires               []demand.AcquireRequest
	renews                 []string
	releases               []string
	releaseContextCanceled bool
	err                    error
}

func (f *demandFixture) Acquire(_ context.Context, req demand.AcquireRequest) (demand.Lease, error) {
	f.acquires = append(f.acquires, req)
	if f.err != nil {
		return demand.Lease{}, f.err
	}
	return demand.Lease{LeaseID: req.LeaseID, Scenario: req.Scenario, Kind: req.Kind, Status: "active", ExpiresAt: time.Now().Add(time.Minute)}, nil
}

func (f *demandFixture) Renew(_ context.Context, leaseID string, _ time.Duration) (demand.Lease, error) {
	f.renews = append(f.renews, leaseID)
	if f.err != nil {
		return demand.Lease{}, f.err
	}
	return demand.Lease{LeaseID: leaseID, Status: "active", ExpiresAt: time.Now().Add(time.Minute)}, nil
}

func (f *demandFixture) Release(ctx context.Context, leaseID, _ string) (demand.Lease, error) {
	f.releases = append(f.releases, leaseID)
	f.releaseContextCanceled = ctx.Err() != nil
	if f.err != nil {
		return demand.Lease{}, f.err
	}
	return demand.Lease{LeaseID: leaseID, Status: "released"}, nil
}

func TestCleanupReleasesJobDemandAfterSuiteCancellation(t *testing.T) {
	home := t.TempDir()
	fixture := &demandFixture{}
	manager := New("demo", t.TempDir()).
		WithHome(home).
		WithDemandLease(fixture, "test-genie:job:run-canceled", "run-canceled").
		WithProbes(func(context.Context, int) bool { return true }, func(int) bool { return true }).
		WithCommandRunner(func(context.Context, string, map[string]string, io.Writer, string, ...string) error {
			return errors.New("start failed")
		})

	// EnsureRunning releases on its own fresh context after the start failure;
	// use a canceled cleanup context for an already-live lease to exercise the
	// normal suite teardown path explicitly.
	writeRecord(t, home, "demo", "start-api", 4001)
	lease, err := manager.EnsureRunning(context.Background(), Needs{API: true}, io.Discard)
	if err != nil {
		t.Fatalf("EnsureRunning: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := manager.Cleanup(ctx, lease, io.Discard); err != nil {
		t.Fatalf("Cleanup: %v", err)
	}
	if fixture.releaseContextCanceled {
		t.Fatal("demand release inherited canceled suite context")
	}
}

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

func TestEnsureRunningOwnsAndReleasesJobDemand(t *testing.T) {
	home := t.TempDir()
	scenarioDir := filepath.Join(t.TempDir(), "demo")
	if err := os.MkdirAll(scenarioDir, 0o755); err != nil {
		t.Fatal(err)
	}
	var calls [][]string
	fixture := &demandFixture{}
	manager := New("demo", scenarioDir).
		WithHome(home).
		WithDemandLease(fixture, "test-genie:job:run-1", "run-1").
		WithProbes(func(context.Context, int) bool { return true }, func(int) bool { return true }).
		WithCommandRunner(func(ctx context.Context, dir string, env map[string]string, logWriter io.Writer, name string, args ...string) error {
			calls = append(calls, append([]string{name}, args...))
			if len(calls) == 1 {
				writeRecord(t, home, "demo", "start-api", 4001)
			}
			return nil
		})

	lease, err := manager.EnsureRunning(context.Background(), Needs{API: true}, io.Discard)
	if err != nil {
		t.Fatalf("EnsureRunning: %v", err)
	}
	if !lease.Started || len(fixture.acquires) != 1 {
		t.Fatalf("lease=%+v acquires=%d, want started run with one job hold", lease, len(fixture.acquires))
	}
	acquired := fixture.acquires[0]
	if acquired.Kind != demand.KindJob || acquired.ConsumerID != "test-genie:job:run-1" || acquired.RequestID != "run-1" {
		t.Fatalf("unexpected job demand: %+v", acquired)
	}
	wantStart := []string{"vrooli", "scenario", "start", "demo", "--clean-stale", "--path", scenarioDir, "--demand-managed"}
	if !reflect.DeepEqual(calls[0], wantStart) {
		t.Fatalf("start command=%#v, want %#v", calls[0], wantStart)
	}
	if err := manager.Cleanup(context.Background(), lease, io.Discard); err != nil {
		t.Fatalf("Cleanup: %v", err)
	}
	if len(fixture.releases) != 1 || fixture.releases[0] != acquired.LeaseID {
		t.Fatalf("released leases=%v, want %s", fixture.releases, acquired.LeaseID)
	}
	if len(calls) != 1 {
		t.Fatalf("cleanup calls=%#v: releasing job ownership must leave stopping to the demand reconciler", calls)
	}
	if err := manager.Cleanup(context.Background(), lease, io.Discard); err != nil {
		t.Fatalf("repeated Cleanup: %v", err)
	}
	if len(fixture.releases) != 1 || len(calls) != 1 {
		t.Fatalf("repeated cleanup changed ownership: releases=%v calls=%v", fixture.releases, calls)
	}
}

func TestEnsureRunningJobDemandProtectsAlreadyLiveTarget(t *testing.T) {
	home := t.TempDir()
	writeRecord(t, home, "demo", "start-api", 4001)
	fixture := &demandFixture{}
	var started bool
	manager := New("demo", "").
		WithHome(home).
		WithDemandLease(fixture, "test-genie:job:run-2", "run-2").
		WithProbes(func(context.Context, int) bool { return true }, func(int) bool { return true }).
		WithCommandRunner(func(context.Context, string, map[string]string, io.Writer, string, ...string) error {
			started = true
			return nil
		})

	lease, err := manager.EnsureRunning(context.Background(), Needs{API: true}, io.Discard)
	if err != nil {
		t.Fatalf("EnsureRunning: %v", err)
	}
	if lease.Started || len(fixture.acquires) != 1 {
		t.Fatalf("live target lease=%+v acquires=%d, want unowned runtime with job hold", lease, len(fixture.acquires))
	}
	if err := manager.Cleanup(context.Background(), lease, io.Discard); err != nil {
		t.Fatalf("Cleanup: %v", err)
	}
	if started || len(fixture.releases) != 1 {
		t.Fatalf("cleanup started=%t releases=%d, want no stop and one release", started, len(fixture.releases))
	}
}

func TestEnsureRunningDoesNotStartWhenJobDemandAcquireFails(t *testing.T) {
	fixture := &demandFixture{err: errors.New("runtime unavailable")}
	started := false
	manager := New("demo", t.TempDir()).
		WithDemandLease(fixture, "test-genie:job:run-3", "run-3").
		WithCommandRunner(func(context.Context, string, map[string]string, io.Writer, string, ...string) error {
			started = true
			return nil
		})
	if _, err := manager.EnsureRunning(context.Background(), Needs{API: true}, io.Discard); err == nil {
		t.Fatal("expected demand acquisition failure")
	}
	if started {
		t.Fatal("lifecycle start ran after demand acquisition failed")
	}
}

func TestEnsureRunningReleasesJobDemandWhenStartFails(t *testing.T) {
	fixture := &demandFixture{}
	manager := New("demo", t.TempDir()).
		WithDemandLease(fixture, "test-genie:job:run-4", "run-4").
		WithCommandRunner(func(context.Context, string, map[string]string, io.Writer, string, ...string) error {
			return errors.New("start failed")
		})
	if _, err := manager.EnsureRunning(context.Background(), Needs{API: true}, io.Discard); err == nil {
		t.Fatal("expected lifecycle start failure")
	}
	if len(fixture.acquires) != 1 || len(fixture.releases) != 1 || fixture.releases[0] != fixture.acquires[0].LeaseID {
		t.Fatalf("failed start demand cleanup: acquires=%v releases=%v", fixture.acquires, fixture.releases)
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

func TestEnsureRunningBoundsLifecycleStart(t *testing.T) {
	manager := New("demo", t.TempDir())
	manager.StartTimeout = 10 * time.Millisecond
	manager.WithCommandRunner(func(ctx context.Context, dir string, env map[string]string, logWriter io.Writer, name string, args ...string) error {
		<-ctx.Done()
		return ctx.Err()
	})

	_, err := manager.EnsureRunning(context.Background(), Needs{UI: true}, io.Discard)
	if err == nil {
		t.Fatal("EnsureRunning() error = nil, want lifecycle timeout")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("EnsureRunning() error = %v, want context deadline exceeded", err)
	}
	if !strings.Contains(err.Error(), "lifecycle start timed out") {
		t.Fatalf("EnsureRunning() error = %q, want timeout diagnosis", err)
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

// Renewal failure must end the context handed to target work, while cleanup
// remains idempotent and independent of the failed work context.
func TestLeaseRenewalLossCancelsTargetWorkAndReleasesOnce(t *testing.T) {
	failure := errors.New("job lease was revoked")
	client := &renewalFailureClient{failure: failure}
	hold, err := demand.AcquireLifetime(context.Background(), client, demand.AcquireRequest{
		LeaseID: "job-revoked", Scenario: "demo", Kind: demand.KindJob, TTL: 30 * time.Millisecond,
	}, "job finished")
	if err != nil {
		t.Fatal(err)
	}
	lease := Lease{demandHold: hold}
	defer hold.Close()
	select {
	case <-lease.Context(context.Background()).Done():
	case <-time.After(2 * time.Second):
		t.Fatal("target work outlived renewal loss")
	}
	if !errors.Is(context.Cause(lease.Context(context.Background())), failure) {
		t.Fatal("renewal cause was lost")
	}
	manager := New("demo", t.TempDir())
	for range 2 {
		if err := manager.Cleanup(lease.Context(context.Background()), lease, io.Discard); err != nil {
			t.Fatal(err)
		}
	}
	if client.releases != 1 || client.canceledRelease {
		t.Fatalf("release count=%d canceled=%v", client.releases, client.canceledRelease)
	}
}

type renewalFailureClient struct {
	failure         error
	releases        int
	canceledRelease bool
}

func (f *renewalFailureClient) Acquire(_ context.Context, req demand.AcquireRequest) (demand.Lease, error) {
	return demand.Lease{LeaseID: req.LeaseID, Status: "active", ExpiresAt: time.Now().Add(time.Minute)}, nil
}
func (f *renewalFailureClient) Renew(context.Context, string, time.Duration) (demand.Lease, error) {
	return demand.Lease{}, f.failure
}
func (f *renewalFailureClient) Release(ctx context.Context, id, _ string) (demand.Lease, error) {
	f.releases++
	f.canceledRelease = ctx.Err() != nil
	return demand.Lease{LeaseID: id}, nil
}
