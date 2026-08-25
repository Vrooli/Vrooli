package lifecycle

import (
	"bytes"
	"fmt"
	"io"
	"runtime"
	"strings"
	"testing"
	"time"

	resourcecontrol "github.com/vrooli/vrooli/internal/resources/control"
	"github.com/vrooli/vrooli/internal/scenario"
)

// Characterization suite for the scenario start/restart wait contract refactor
// (docs/plans/scenario-lifecycle-start-wait-contract-plan.md, Phase 1 step 1).
//
// These tests pin the CURRENT observable behavior of startScenario and the
// bespoke wait loops — human-visible progress lines, step ordering, and the
// exact timeout/interval values — so the awaiter migration (Phase 1) and the
// startScenario decomposition (Phase 2) can prove "no observable change".
// They intentionally assert byte-for-byte output and exact sleep sequences.

// fakeAwaitClock is a deterministic now/sleep pair that records every sleep
// duration and advances virtual time by exactly the requested amount.
type fakeAwaitClock struct {
	now    time.Time
	sleeps []time.Duration
}

func newFakeAwaitClock() *fakeAwaitClock {
	return &fakeAwaitClock{now: time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)}
}

func (c *fakeAwaitClock) Now() time.Time { return c.now }

func (c *fakeAwaitClock) Sleep(d time.Duration) {
	c.sleeps = append(c.sleeps, d)
	c.now = c.now.Add(d)
}

func durationsEqual(got, want []time.Duration) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

// --- WaitForHealth (internal/lifecycle/health.go) ---

func waitForHealthFixture(critical bool, grace, timeout, interval int) scenario.Scenario {
	return scenario.Scenario{
		Slug: "charlie",
		Manifest: scenario.ServiceManifest{
			Lifecycle: scenario.Lifecycle{
				Health: &scenario.HealthConfig{
					Checks: []scenario.HealthCheck{{
						Name:     "api",
						Type:     "unsupported",
						Critical: critical,
					}},
					StartupGracePeriod: grace,
					Timeout:            timeout,
					Interval:           interval,
				},
			},
		},
	}
}

// TestCharacterizeWaitForHealthTiming pins the readiness-driven wait shape:
// evaluation happens immediately, startup grace is only a failure ceiling,
// and expiry is strictly-after (now == deadline gets one more evaluation).
func TestCharacterizeWaitForHealthTiming(t *testing.T) {
	clock := newFakeAwaitClock()
	runner := &Runner{deps: lifecycleDeps{now: clock.Now, sleep: clock.Sleep}}

	item := waitForHealthFixture(true, 25, 50, 10)
	status, err := runner.WaitForHealth(item, nil)
	if err == nil {
		t.Fatal("expected failing critical check to error")
	}
	if status != "unhealthy" {
		t.Fatalf("status = %q, want unhealthy", status)
	}
	// The health deadline is anchored at the first evaluation. Evaluations at
	// 0..50 all sleep (50 is NOT strictly after the deadline); the eval at 60
	// exits. StartupGracePeriod contributes no sleep.
	want := []time.Duration{
		10 * time.Millisecond, 10 * time.Millisecond, 10 * time.Millisecond,
		10 * time.Millisecond, 10 * time.Millisecond, 10 * time.Millisecond,
	}
	if !durationsEqual(clock.sleeps, want) {
		t.Fatalf("sleep sequence = %v, want %v", clock.sleeps, want)
	}
}

// TestCharacterizeWaitForHealthDegradedAfterTimeoutSucceeds pins the
// degraded-grace success path: a persistently degraded scenario (non-critical
// checks failing) converts deadline expiry into a nil-error "degraded" return.
func TestCharacterizeWaitForHealthDegradedAfterTimeoutSucceeds(t *testing.T) {
	clock := newFakeAwaitClock()
	runner := &Runner{deps: lifecycleDeps{now: clock.Now, sleep: clock.Sleep}}

	item := waitForHealthFixture(false, 0, 50, 10)
	status, err := runner.WaitForHealth(item, nil)
	if err != nil {
		t.Fatalf("degraded-after-timeout must succeed, got %v", err)
	}
	if status != "degraded" {
		t.Fatalf("status = %q, want degraded", status)
	}
	if len(clock.sleeps) == 0 {
		t.Fatal("expected interval sleeps before timeout")
	}
}

// TestCharacterizeWaitForHealthDefaultsAndIntervalCap pins the policy values:
// default deadline 30s when the manifest omits Timeout, and manifest intervals
// above 2s are capped to 2s.
func TestCharacterizeWaitForHealthDefaultsAndIntervalCap(t *testing.T) {
	clock := newFakeAwaitClock()
	runner := &Runner{deps: lifecycleDeps{now: clock.Now, sleep: clock.Sleep}}

	item := waitForHealthFixture(true, 0, 0, 5000)
	if _, err := runner.WaitForHealth(item, nil); err == nil {
		t.Fatal("expected failing critical check to error")
	}
	if len(clock.sleeps) == 0 {
		t.Fatal("expected sleeps")
	}
	for i, d := range clock.sleeps {
		if d != 2*time.Second {
			t.Fatalf("sleep[%d] = %v, want capped 2s interval", i, d)
		}
	}
	// Default 30s deadline with 2s polls: evals at 0,2,...,30 all sleep (30 is
	// not strictly after), eval at 32 exits ⇒ 16 sleeps.
	if len(clock.sleeps) != 16 {
		t.Fatalf("sleep count = %d, want 16 (30s default deadline, 2s interval, strict-after expiry)", len(clock.sleeps))
	}
}

// TestCharacterizeWaitForHealthNoChecksFastReturn pins the no-checks fast
// path: immediate "running", no sleeps, no error.
func TestCharacterizeWaitForHealthNoChecksFastReturn(t *testing.T) {
	clock := newFakeAwaitClock()
	runner := &Runner{deps: lifecycleDeps{now: clock.Now, sleep: clock.Sleep}}
	status, err := runner.WaitForHealth(scenario.Scenario{Slug: "nochecks"}, nil)
	if err != nil || status != "running" {
		t.Fatalf("got (%q, %v), want (running, nil)", status, err)
	}
	if len(clock.sleeps) != 0 {
		t.Fatalf("expected no sleeps, got %v", clock.sleeps)
	}
}

// --- isRegistryRuntimeHealthy retry (internal/lifecycle/health.go) ---

// TestCharacterizeRegistryHealthRetryPolicy pins the bounded probe retry: 3
// attempts with 1s sleeps BETWEEN attempts (2 sleeps total) before condemning
// a running instance.
func TestCharacterizeRegistryHealthRetryPolicy(t *testing.T) {
	clock := newFakeAwaitClock()
	runner := &Runner{deps: lifecycleDeps{
		now:          clock.Now,
		sleep:        clock.Sleep,
		isPIDRunning: func(int) bool { return true },
	}}

	item := waitForHealthFixture(true, 0, 0, 0)
	view := registryRuntimeView{Authoritative: true}
	if runner.isRegistryRuntimeHealthy(item, view) {
		t.Fatal("failing checks must report unhealthy")
	}
	want := []time.Duration{1 * time.Second, 1 * time.Second}
	if !durationsEqual(clock.sleeps, want) {
		t.Fatalf("sleep sequence = %v, want %v", clock.sleeps, want)
	}
}

// TestCharacterizeRegistryHealthOrphanSquatGuard pins the orphan-squat PID
// guard: a dead recorded owner (non-supervisor) short-circuits to unhealthy
// with zero probe attempts, while an unknown owner PID is NOT condemned.
func TestCharacterizeRegistryHealthOrphanSquatGuard(t *testing.T) {
	deadPID := 424242
	item := waitForHealthFixture(true, 0, 0, 0)

	clock := newFakeAwaitClock()
	runner := &Runner{deps: lifecycleDeps{
		now:          clock.Now,
		sleep:        clock.Sleep,
		isPIDRunning: func(int) bool { return false },
	}}
	view := registryRuntimeView{Authoritative: true}
	view.Instance.OwnerPID = &deadPID
	if runner.isRegistryRuntimeHealthy(item, view) {
		t.Fatal("dead owner pid must condemn the runtime")
	}
	if len(clock.sleeps) != 0 {
		t.Fatalf("orphan-squat guard must not probe/sleep, got %v", clock.sleeps)
	}

	// Unknown owner PID: not condemned by the guard; falls through to probes.
	clock2 := newFakeAwaitClock()
	runner2 := &Runner{deps: lifecycleDeps{
		now:          clock2.Now,
		sleep:        clock2.Sleep,
		isPIDRunning: func(int) bool { return false },
	}}
	view2 := registryRuntimeView{Authoritative: true}
	if runner2.isRegistryRuntimeHealthy(item, view2) {
		t.Fatal("failing checks must still report unhealthy")
	}
	if len(clock2.sleeps) != 2 {
		t.Fatalf("unknown owner must fall through to the 3-attempt probe, sleeps=%v", clock2.sleeps)
	}

	// No health checks: authoritative instance is healthy with zero probes.
	if !runner2.isRegistryRuntimeHealthy(scenario.Scenario{Slug: "nochecks"}, view2) {
		t.Fatal("no-checks scenario with authoritative registry must be healthy")
	}
	// Non-authoritative view is never healthy.
	if runner2.isRegistryRuntimeHealthy(scenario.Scenario{Slug: "nochecks"}, registryRuntimeView{}) {
		t.Fatal("non-authoritative view must be unhealthy")
	}
}

// --- waitForResourceDependencyReady (internal/lifecycle/dependencies.go) ---

// TestCharacterizeResourceDependencyReadyTiming pins the resource readiness
// wait: 500ms interval, 30s deadline, expiry when now >= deadline (inclusive),
// evaluation before every deadline check, and the exact not-ready error shape.
func TestCharacterizeResourceDependencyReadyTiming(t *testing.T) {
	clock := newFakeAwaitClock()
	statusCalls := 0
	runner := &Runner{deps: lifecycleDeps{
		now:   clock.Now,
		sleep: clock.Sleep,
		resourceStatus: func(name string, fast bool) (resourcecontrol.Status, error) {
			statusCalls++
			return resourcecontrol.Status{Running: false, Health: "down", StatusCode: "not_running"}, nil
		},
	}}

	_, err := runner.waitForResourceDependencyReady("postgres")
	if err == nil {
		t.Fatal("expected not-ready timeout error")
	}
	wantMsg := `resource dependency postgres is not ready after start (running=false health="down" status_code="not_running")`
	if err.Error() != wantMsg {
		t.Fatalf("error = %q, want %q", err.Error(), wantMsg)
	}
	// Inclusive expiry: evals at 0, 0.5, …, 29.5 all sleep; the eval at 30.0
	// sees now >= deadline and exits ⇒ 61 evaluations, 60 sleeps of 500ms.
	if statusCalls != 61 {
		t.Fatalf("status calls = %d, want 61", statusCalls)
	}
	if len(clock.sleeps) != 60 {
		t.Fatalf("sleep count = %d, want 60", len(clock.sleeps))
	}
	for i, d := range clock.sleeps {
		if d != 500*time.Millisecond {
			t.Fatalf("sleep[%d] = %v, want 500ms", i, d)
		}
	}
}

// TestCharacterizeResourceDependencyReadyStatusErrorWrapped pins the persistent
// status-error path: the LAST status error is wrapped in the timeout message.
func TestCharacterizeResourceDependencyReadyStatusErrorWrapped(t *testing.T) {
	clock := newFakeAwaitClock()
	probeErr := fmt.Errorf("controller unavailable")
	runner := &Runner{deps: lifecycleDeps{
		now:   clock.Now,
		sleep: clock.Sleep,
		resourceStatus: func(name string, fast bool) (resourcecontrol.Status, error) {
			return resourcecontrol.Status{}, probeErr
		},
	}}
	_, err := runner.waitForResourceDependencyReady("redis")
	if err == nil {
		t.Fatal("expected timeout error")
	}
	want := "status started resource dependency redis: controller unavailable"
	if err.Error() != want {
		t.Fatalf("error = %q, want %q", err.Error(), want)
	}
}

// --- full-start progress lines (startScenario / stop / restart) ---

// startCharacterizationRunner builds a real Runner over the alpha fixture with
// progress output captured. Quiet verbosity still emits progressf lines (they
// are the primary heartbeat) while suppressing [INFO] step headers, so the
// captured buffer holds exactly the progressf stream.
func startCharacterizationRunner(t *testing.T) (*Runner, *bytes.Buffer) {
	t.Helper()
	root := t.TempDir()
	home := t.TempDir()
	writeLifecycleFixture(t, root, "alpha")
	var out bytes.Buffer
	runner, err := NewRunner(root, home, &out, io.Discard)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	runner.Verbosity = VerbosityQuiet
	return runner, &out
}

// TestCharacterizeStartProgressLines pins the byte-exact human progress stream
// for the three canonical flows: fresh start, already-running reuse, restart.
func TestCharacterizeStartProgressLines(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("lifecycle process management currently targets linux")
	}
	runner, out := startCharacterizationRunner(t)
	cleanupRunner(t, runner, "alpha", StopOptions{})

	if _, err := runner.Start("alpha", StartOptions{}); err != nil {
		t.Fatalf("Start: %v", err)
	}
	wantFresh := "starting alpha...\n" +
		"running setup phase for alpha...\n" +
		"running develop phase for alpha...\n" +
		"waiting for alpha to become healthy...\n"
	if got := out.String(); got != wantFresh {
		t.Fatalf("fresh start progress =\n%q\nwant\n%q", got, wantFresh)
	}

	out.Reset()
	if _, err := runner.Start("alpha", StartOptions{}); err != nil {
		t.Fatalf("Start(reuse): %v", err)
	}
	wantReuse := "starting alpha...\n" +
		"alpha is already running\n"
	if got := out.String(); got != wantReuse {
		t.Fatalf("reuse start progress =\n%q\nwant\n%q", got, wantReuse)
	}

	out.Reset()
	if _, err := runner.Restart("alpha", StartOptions{}); err != nil {
		t.Fatalf("Restart: %v", err)
	}
	wantRestart := "stopping alpha...\n" +
		"starting alpha...\n" +
		"running develop phase for alpha...\n" +
		"waiting for alpha to become healthy...\n"
	if got := out.String(); got != wantRestart {
		t.Fatalf("restart progress =\n%q\nwant\n%q", got, wantRestart)
	}

	out.Reset()
	if err := runner.Stop("alpha", StopOptions{}); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if got := out.String(); got != "stopping alpha...\n" {
		t.Fatalf("stop progress = %q", got)
	}
}

// TestCharacterizeDependencyStartProgressLines pins the dependency-flow lines:
// the "starting dependency" transition (reason string is environment-dependent,
// so it is prefix-asserted), the dependency's own phase lines interleaved
// before the parent's, and the reuse line on a warm second start.
func TestCharacterizeDependencyStartProgressLines(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("lifecycle process management currently targets linux")
	}
	root := t.TempDir()
	home := t.TempDir()
	writeLifecycleFixture(t, root, "beta")
	alpha := lifecycleFixtureManifest("alpha")
	alpha.Dependencies.Scenarios = map[string]scenario.Dependency{
		"beta": {Required: true},
	}
	writeLifecycleFixtureManifest(t, root, alpha)

	var out bytes.Buffer
	runner, err := NewRunner(root, home, &out, io.Discard)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	runner.Verbosity = VerbosityQuiet
	cleanupRunner(t, runner, "alpha", StopOptions{})
	cleanupRunner(t, runner, "beta", StopOptions{})

	if _, err := runner.Start("alpha", StartOptions{}); err != nil {
		t.Fatalf("Start: %v", err)
	}
	lines := strings.Split(strings.TrimSuffix(out.String(), "\n"), "\n")
	wantSeq := []struct {
		prefix bool
		text   string
	}{
		{false, "starting alpha..."},
		{true, "alpha: starting dependency beta ("},
		{false, "running setup phase for beta..."},
		{false, "running develop phase for beta..."},
		{false, "waiting for beta to become healthy..."},
		{false, "running setup phase for alpha..."},
		{false, "running develop phase for alpha..."},
		{false, "waiting for alpha to become healthy..."},
	}
	if len(lines) != len(wantSeq) {
		t.Fatalf("line count = %d, want %d; lines=%q", len(lines), len(wantSeq), lines)
	}
	for i, want := range wantSeq {
		if want.prefix {
			if !strings.HasPrefix(lines[i], want.text) {
				t.Fatalf("line[%d] = %q, want prefix %q", i, lines[i], want.text)
			}
			continue
		}
		if lines[i] != want.text {
			t.Fatalf("line[%d] = %q, want %q", i, lines[i], want.text)
		}
	}

	out.Reset()
	if _, err := runner.Start("alpha", StartOptions{}); err != nil {
		t.Fatalf("Start(warm): %v", err)
	}
	wantWarm := "starting alpha...\n" +
		"alpha: dependency beta already running; reusing existing process\n" +
		"alpha is already running\n"
	if got := out.String(); got != wantWarm {
		t.Fatalf("warm start progress =\n%q\nwant\n%q", got, wantWarm)
	}
}
