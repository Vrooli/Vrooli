package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func readStatus(t *testing.T, l *loop) loopStatus {
	t.Helper()
	data, err := os.ReadFile(l.status.path)
	if err != nil {
		t.Fatalf("status file: %v", err)
	}
	var status loopStatus
	if err := json.Unmarshal(data, &status); err != nil {
		t.Fatalf("status file is not JSON: %v\n%s", err, data)
	}
	return status
}

// D6: a CLI that rejects the loop's argv is not something a retry fixes. The
// loop attempts three times, writes the status file naming the class, and
// exits 3 so the scheduler escalates. No breaker slot is spent on the way.
// [REQ:AUTOHEAL-P0-014] [REQ:CLI-LOOP-002] [REQ:CLI-LOOP-003]
func TestUsageErrorIsNonHealableAndExits3(t *testing.T) {
	home := isolatedHome(t)
	config := testConfig(t)
	config.VrooliCmdPath = fakeVrooli(t, usageBody)
	l, slept := testLoop(t, config)
	l.state = stateDetect
	l.record.State = stateDetect.String()

	done := make(chan int, 1)
	go func() { done <- l.run(context.Background()) }()
	select {
	case code := <-done:
		if code != exitNonHealable {
			t.Fatalf("exit code = %d, want %d", code, exitNonHealable)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("loop kept retrying a usage error")
	}

	status := readStatus(t, l)
	if status.LastFailureClass != "usage" || status.ConsecutiveFailures != nonHealableExitThreshold || status.State != "exit" {
		t.Fatalf("status = %+v", status)
	}
	if status.ExitCode == nil || *status.ExitCode != exitNonHealable {
		t.Fatalf("status exit_code = %v, want %d", status.ExitCode, exitNonHealable)
	}
	if got := countCalls(vrooliCalls(t, config.VrooliCmdPath), "scenario start"); got != nonHealableExitThreshold {
		t.Fatalf("scenario start attempted %d times, want %d", got, nonHealableExitThreshold)
	}
	for _, d := range *slept {
		if d != config.TickInterval {
			t.Fatalf("non-healable retries must pause one tick interval, not back off: slept %v", *slept)
		}
	}
	if _, err := os.Stat(filepath.Join(home, ".vrooli", "state", "vrooli-autoheal", "recovery-floor.json")); !os.IsNotExist(err) {
		t.Fatalf("recovery floor state was touched: %v", err)
	}
}

// Healable failures back off from one minute, doubling to fifteen, and the
// first healthy tick resets the wait.
// [REQ:AUTOHEAL-P0-002] [REQ:CLI-LOOP-001]
func TestBackoffDoublesAndResets(t *testing.T) {
	isolatedHome(t)
	want := []time.Duration{time.Minute, 2 * time.Minute, 4 * time.Minute, 8 * time.Minute, 15 * time.Minute, 15 * time.Minute}
	backoff := time.Duration(0)
	for i, w := range want {
		if backoff = nextBackoff(backoff); backoff != w {
			t.Fatalf("step %d: backoff = %v, want %v", i, backoff, w)
		}
	}

	config := testConfig(t)
	config.VrooliCmdPath = fakeVrooli(t, "case \"$1 $2\" in \"scenario start\") echo 'build component api: exit status 1' >&2; exit 1;; esac\n"+usageBody)
	l, slept := testLoop(t, config)
	l.state = stateHeal
	l.healReason = "test"
	for i := 0; i < 3; i++ {
		l.step(context.Background())
		if l.state != stateHeal {
			t.Fatalf("attempt %d: state = %s, want heal", i, l.state)
		}
	}
	if got := *slept; len(got) != 2 || got[0] != time.Minute || got[1] != 2*time.Minute {
		t.Fatalf("slept %v, want [1m 2m] (no wait before the first attempt)", got)
	}
	if l.backoff != 4*time.Minute || l.record.LastFailureClass != "lifecycle" || l.record.ConsecutiveFailures != 0 {
		t.Fatalf("after three healable failures: backoff=%v class=%q nonhealable=%d", l.backoff, l.record.LastFailureClass, l.record.ConsecutiveFailures)
	}

	if !config.adoptPort(context.Background(), fakeAPI(t, "Vrooli Autoheal API")) {
		t.Fatal("adopt")
	}
	l.state = stateVerify
	l.step(context.Background())
	if l.state != stateHealthy || l.backoff != 0 {
		t.Fatalf("a healthy tick must reset the backoff: state=%s backoff=%v", l.state, l.backoff)
	}
}

// The status file is the escalation target's only evidence; it must carry the
// exit code before the process ends, and every tick must refresh it.
// [REQ:CLI-LOOP-003]
func TestStatusFileWrittenBeforeExit(t *testing.T) {
	isolatedHome(t)
	config := testConfig(t)
	l, _ := testLoop(t, config)
	if !config.adoptPort(context.Background(), fakeAPI(t, "Vrooli Autoheal API")) {
		t.Fatal("adopt")
	}
	l.state = stateVerify
	l.step(context.Background())
	status := readStatus(t, l)
	if status.LastTickStatus != "ok" || status.LastTickAt == nil || status.State != "healthy" || status.PID != os.Getpid() {
		t.Fatalf("heartbeat not written after a tick: %+v", status)
	}
	if status.ExitCode != nil {
		t.Fatal("a running loop must not carry an exit code")
	}

	l.exit(exitNonHealable, "three usage errors")
	status = readStatus(t, l)
	if status.ExitCode == nil || *status.ExitCode != exitNonHealable || status.State != "exit" || status.DegradedReason != "three usage errors" {
		t.Fatalf("exit not recorded before returning: %+v", status)
	}
	if l.run(context.Background()) != exitNonHealable {
		t.Fatal("run after exit must return the recorded code")
	}
}

// [REQ:CLI-LOOP-003]
func TestPreflightFailureDegradesAndCountsTowardExit(t *testing.T) {
	isolatedHome(t)
	config := testConfig(t)
	config.VrooliCmdPath = fakeVrooli(t, usageBody)
	l, _ := testLoop(t, config)

	code := l.run(context.Background())
	if code != exitNonHealable {
		t.Fatalf("exit code = %d, want %d", code, exitNonHealable)
	}
	status := readStatus(t, l)
	if status.LastFailureClass != "usage" || status.Preflight == nil || status.Preflight.OK {
		t.Fatalf("status = %+v", status)
	}
}

// [REQ:AUTOHEAL-P0-002] [REQ:AUTOHEAL-P1-009] [REQ:CLI-LOOP-001] [REQ:INFRA-SHUTDOWN-001]
func TestSignalDuringSleepExitsZero(t *testing.T) {
	isolatedHome(t)
	config := testConfig(t)
	l, _ := testLoop(t, config)
	if !config.adoptPort(context.Background(), fakeAPI(t, "Vrooli Autoheal API")) {
		t.Fatal("adopt")
	}
	ctx, cancel := context.WithCancel(context.Background())
	l.sleep = func(ctx context.Context, _ time.Duration) bool { cancel(); return false }
	l.state = stateHealthy
	if code := l.run(ctx); code != exitSignal {
		t.Fatalf("exit code = %d, want 0 on signal", code)
	}
	if status := readStatus(t, l); status.ExitCode == nil || *status.ExitCode != exitSignal || status.State != "exit" {
		t.Fatalf("status = %+v", status)
	}
}

// [REQ:CLI-LOOP-003]
func TestAliveButFailingTicksDegradeInsteadOfRestarting(t *testing.T) {
	isolatedHome(t)
	config := testConfig(t)
	config.VrooliCmdPath = fakeVrooli(t, usageBody)
	l, _ := testLoop(t, config)
	if !config.adoptPort(context.Background(), fakeAPI(t, "Vrooli Autoheal API")) {
		t.Fatal("adopt")
	}
	// Point ticks at a path the fake does not serve, while /health still
	// identifies autoheal.
	config.TickEndpoint = config.HealthEndpoint + "/missing"
	l.state = stateHealthy
	for i := 0; i < config.MaxFailures; i++ {
		l.step(context.Background())
	}
	if l.state != stateDegraded {
		t.Fatalf("state = %s, want degraded (a live API is never restarted for failing ticks)", l.state)
	}
	if countCalls(vrooliCalls(t, config.VrooliCmdPath), "scenario") != 0 {
		t.Fatal("the lifecycle was touched")
	}
}
