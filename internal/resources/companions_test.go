package resources

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/process"
	"github.com/vrooli/vrooli/internal/tuning"
)

func TestWhisperActivityEdgeHasStablePrivateControlOwner(t *testing.T) {
	c := ResourceCompanion{Name: "activity-edge", Command: "resource-whisper", Args: []string{"activity-edge", "--native"}}
	got := companionLaunchArgs("whisper", c, "/runtime/processes/resources/whisper", 4321)
	want := []string{"activity-edge", "--native", "--control-socket", "/runtime/processes/resources/whisper/activity-edge.control.sock"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Whisper activity-edge args=%v, want %v", got, want)
	}
	other := companionLaunchArgs("reranker", ResourceCompanion{Name: "activity-edge", Args: []string{"activity-edge"}}, "/tmp", 4321)
	if !reflect.DeepEqual(other, []string{"activity-edge", "--parent-pid", "4321"}) {
		t.Fatalf("non-Whisper companion args=%v", other)
	}
	ollama := companionLaunchArgs("ollama", ResourceCompanion{Name: "activity-edge", Args: []string{"activity-edge"}}, "/runtime/processes/resources/ollama", 4321)
	wantOllama := []string{"activity-edge", "--control-socket", "/runtime/processes/resources/ollama/activity-edge.control.sock"}
	if !reflect.DeepEqual(ollama, wantOllama) {
		t.Fatalf("Ollama activity-edge args=%v, want %v", ollama, wantOllama)
	}
	for resource, want := range map[string]bool{"whisper": true, "ollama": true, "reranker": false} {
		if got := hasStableActivityEdge(resource); got != want {
			t.Errorf("hasStableActivityEdge(%q)=%t, want %t", resource, got, want)
		}
	}
}

// TestStartCompanionsDormant proves the supervisor is a no-op (byte-identical)
// for a resource that declares no companions.
func TestStartCompanionsDormant(t *testing.T) {
	called := false
	defer withCompanionDir(t, func(string) (string, error) {
		called = true
		return "", nil
	})()
	startCompanions("plain-resource", nil, 0, io.Discard)
	stopCompanions("plain-resource", nil, io.Discard)
	if called {
		t.Error("no companions declared must not touch the companion dir")
	}
}

func TestCompanionStartRecoversUniqueOwnedProcessWithoutLaunching(t *testing.T) {
	dir := t.TempDir()
	defer withCompanionDir(t, func(string) (string, error) { return dir, nil })()
	previous := findExistingCompanion
	t.Cleanup(func() { findExistingCompanion = previous })
	findExistingCompanion = func(resource string, c ResourceCompanion, bin string, parent int) (int, error) {
		if resource != "fixture" || c.Name != "edge" || parent != 42 {
			t.Fatalf("wrong recovery identity %q %q %d", resource, c.Name, parent)
		}
		return os.Getpid(), nil
	}
	// Running this command would create an unrelated sleeper; recovery must not.
	if err := startCompanion("fixture", ResourceCompanion{Name: "edge", Command: "sleep", Args: []string{"30"}}, 0, 42); err != nil {
		t.Fatal(err)
	}
	for _, suffix := range []string{"pid", "state"} {
		if pid, ok := readCompanionPID(filepath.Join(dir, "edge."+suffix)); !ok || pid != os.Getpid() {
			t.Fatalf("recovered %s=%d valid=%t", suffix, pid, ok)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "edge.log")); !os.IsNotExist(err) {
		t.Fatal("recovery opened a new launch log", err)
	}
}

func TestCompanionStartRefusesAmbiguousRecoveryWithoutLaunching(t *testing.T) {
	dir := t.TempDir()
	defer withCompanionDir(t, func(string) (string, error) { return dir, nil })()
	previous := findExistingCompanion
	t.Cleanup(func() { findExistingCompanion = previous })
	findExistingCompanion = func(string, ResourceCompanion, string, int) (int, error) {
		return 0, fmt.Errorf("ambiguous companion scope")
	}
	if err := startCompanion("fixture", ResourceCompanion{Name: "edge", Command: "sleep", Args: []string{"30"}}, 0, 42); err == nil {
		t.Fatal("ambiguous recovery launched another process")
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 0 {
		t.Fatalf("ambiguous recovery changed ownership: %v %v", entries, err)
	}
}

// TestCompanionLifecycle proves start launches + tracks a detached process,
// start is idempotent while alive, and stop signals it and clears the pidfile.
func TestCompanionLifecycle(t *testing.T) {
	t.Setenv("VROOLI_AGENT_IDENTITY_TOKEN", "original-requester-secret")
	t.Setenv("OPENCODE_AGENT_TAG", "original-tag")
	t.Setenv("VROOLI_RUN_ID", "original-run")
	t.Setenv("RESOURCE_COMPANION_FIXTURE", "retained")
	dir := t.TempDir()
	defer withCompanionDir(t, func(string) (string, error) { return dir, nil })()

	c := ResourceCompanion{Name: "edge", Command: "sleep", Args: []string{"30"}}
	if err := startCompanion("whisper", c, 0); err != nil {
		t.Fatalf("start companion: %v", err)
	}
	pidPath := filepath.Join(dir, "edge.pid")
	pid, ok := readCompanionPID(pidPath)
	liveEnv, envErr := process.ReadEnvironment(pid)
	if envErr != nil {
		t.Fatal(envErr)
	}
	for _, key := range []string{"VROOLI_AGENT_IDENTITY_TOKEN", "OPENCODE_AGENT_TAG", "VROOLI_RUN_ID"} {
		if liveEnv[key] != "" {
			t.Errorf("detached companion inherited requester %s", key)
		}
	}
	if liveEnv["RESOURCE_COMPANION_FIXTURE"] != "retained" {
		t.Fatal("resource environment lost")
	}
	if !ok || !process.IsPIDRunning(pid) {
		t.Fatalf("companion should be running; pidfile ok=%v pid=%d", ok, pid)
	}

	// Idempotent: a second start while alive reuses the same process.
	if err := startCompanion("whisper", c, 0); err != nil {
		t.Fatalf("idempotent start: %v", err)
	}
	if pid2, _ := readCompanionPID(pidPath); pid2 != pid {
		t.Fatalf("idempotent start spawned a new process: %d -> %d", pid, pid2)
	}

	if err := stopCompanion("whisper", c); err != nil {
		t.Fatalf("stop companion: %v", err)
	}
	if _, err := os.Stat(pidPath); !os.IsNotExist(err) {
		t.Errorf("pidfile should be removed after stop, stat err = %v", err)
	}
	// Give the signal a moment to take effect.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && process.IsPIDRunning(pid) {
		time.Sleep(20 * time.Millisecond)
	}
	if process.IsPIDRunning(pid) {
		t.Errorf("companion pid %d still running after stop", pid)
	}
}

func TestCompanionCrashLoopCapWritesFailureMarker(t *testing.T) {
	dir := t.TempDir()
	defer withCompanionDir(t, func(string) (string, error) { return dir, nil })()

	c := ResourceCompanion{Name: "edge", Command: "sleep", Args: []string{"30"}}
	pidPath := filepath.Join(dir, "edge.pid")
	if err := os.WriteFile(pidPath, []byte("99999999"), 0o644); err != nil {
		t.Fatalf("write dead pidfile: %v", err)
	}

	if err := startCompanion("whisper", c, 1); err != nil {
		t.Fatalf("first respawn should be allowed: %v", err)
	}
	pid, ok := readCompanionPID(pidPath)
	if !ok || !process.IsPIDRunning(pid) {
		t.Fatalf("respawned companion should be running; ok=%v pid=%d", ok, pid)
	}
	defer func() { _ = terminateCompanion(pid) }()
	if err := os.WriteFile(pidPath, []byte("99999999"), 0o644); err != nil {
		t.Fatalf("restore stale pidfile: %v", err)
	}

	err := startCompanion("whisper", c, 1)
	if err == nil {
		t.Fatal("second respawn inside the crash window should be capped")
	}
	failed, failure := readCompanionFailure(dir, "edge")
	if !failed {
		t.Fatal("expected failed marker after crash-loop cap")
	}
	if !strings.Contains(failure, "crash-loop cap reached") {
		t.Fatalf("failure = %q, want crash-loop cap", failure)
	}
	statuses, err := companionStatuses("whisper", []ResourceCompanion{{Name: "edge", Port: 8090}})
	if err != nil {
		t.Fatalf("companionStatuses: %v", err)
	}
	if len(statuses) != 1 || !statuses[0].Failed || !strings.Contains(statuses[0].Failure, "crash-loop cap reached") {
		t.Fatalf("statuses = %#v", statuses)
	}
	if !strings.Contains(companionDownMessage("whisper", statuses), "crash-loop cap reached") {
		t.Fatalf("message missing crash-loop cap: %q", companionDownMessage("whisper", statuses))
	}
}

func TestCompanionCrashEvidenceSurvivesStatusReapingPidfile(t *testing.T) {
	dir := t.TempDir()
	defer withCompanionDir(t, func(string) (string, error) { return dir, nil })()

	c := ResourceCompanion{Name: "edge", Command: "sleep", Args: []string{"30"}}
	if err := startCompanion("whisper", c, 1); err != nil {
		t.Fatalf("initial companion start: %v", err)
	}
	pid, ok := readCompanionPID(filepath.Join(dir, "edge.pid"))
	if !ok {
		t.Fatal("initial companion did not write a pidfile")
	}
	if err := terminateCompanion(pid); err != nil {
		t.Fatalf("terminate companion: %v", err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && process.IsPIDRunning(pid) {
		time.Sleep(20 * time.Millisecond)
	}

	statuses, err := companionStatuses("whisper", []ResourceCompanion{{Name: "edge"}})
	if err != nil {
		t.Fatalf("status after crash: %v", err)
	}
	if len(statuses) != 1 || statuses[0].Alive {
		t.Fatalf("status after crash = %#v", statuses)
	}
	if _, err := os.Stat(filepath.Join(dir, "edge.pid")); !os.IsNotExist(err) {
		t.Fatalf("status should reap dead pidfile, stat error = %v", err)
	}

	if err := startCompanion("whisper", c, 1); err != nil {
		t.Fatalf("recovery after status reaping: %v", err)
	}
	recoveredPID, ok := readCompanionPID(filepath.Join(dir, "edge.pid"))
	if !ok || !process.IsPIDRunning(recoveredPID) {
		t.Fatalf("recovered companion should be running; ok=%v pid=%d", ok, recoveredPID)
	}
	defer func() { _ = terminateCompanion(recoveredPID) }()
	attempts, err := os.ReadFile(filepath.Join(dir, "edge.attempts"))
	if err != nil {
		t.Fatalf("read crash attempts: %v", err)
	}
	if got := strings.Count(strings.TrimSpace(string(attempts)), "\n") + 1; got != 1 {
		t.Fatalf("crash attempts = %d, want one durable attempt", got)
	}
}

// TestStopCompanionNoPidfile is a clean no-op when nothing is tracked.
func TestStopCompanionNoPidfile(t *testing.T) {
	dir := t.TempDir()
	defer withCompanionDir(t, func(string) (string, error) { return dir, nil })()
	if err := stopCompanion("whisper", ResourceCompanion{Name: "edge", Command: "sleep"}); err != nil {
		t.Errorf("stop with no pidfile should be a no-op, got %v", err)
	}
}

func TestCompanionStatusesReportAliveAndDeadPidfiles(t *testing.T) {
	dir := t.TempDir()
	defer withCompanionDir(t, func(string) (string, error) { return dir, nil })()

	if err := os.WriteFile(filepath.Join(dir, "alive.pid"), []byte(strconv.Itoa(os.Getpid())), 0o644); err != nil {
		t.Fatalf("write alive pidfile: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "dead.pid"), []byte("99999999"), 0o644); err != nil {
		t.Fatalf("write dead pidfile: %v", err)
	}

	statuses, err := companionStatuses("whisper", []ResourceCompanion{
		{Name: "alive", Port: 8090},
		{Name: "dead", Port: 8091},
	})
	if err != nil {
		t.Fatalf("companionStatuses: %v", err)
	}
	if len(statuses) != 2 {
		t.Fatalf("len(statuses) = %d, want 2", len(statuses))
	}
	if !statuses[0].Alive || statuses[0].PID != os.Getpid() {
		t.Fatalf("alive status = %#v", statuses[0])
	}
	if statuses[1].Alive || statuses[1].PID != 99999999 {
		t.Fatalf("dead status = %#v", statuses[1])
	}
}

func TestAppendCompanionLogBoundsActiveAndRetainedFiles(t *testing.T) {
	dir := t.TempDir()
	line := strings.Repeat("x", int(tuning.CompanionLogMaxBytes()/2)) + "\n"
	appendCompanionLog(dir, "edge", line)
	appendCompanionLog(dir, "edge", line)
	appendCompanionLog(dir, "edge", "latest-state\n")
	for _, suffix := range []string{"", ".1"} {
		path := filepath.Join(dir, "edge.log"+suffix)
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("stat %s: %v", path, err)
		}
		if info.Size() > tuning.CompanionLogMaxBytes() {
			t.Fatalf("%s size = %d, cap = %d", path, info.Size(), tuning.CompanionLogMaxBytes())
		}
	}
	data, err := os.ReadFile(filepath.Join(dir, "edge.log"))
	if err != nil || !strings.Contains(string(data), "latest-state") {
		t.Fatalf("active log = %q, err=%v; newest line was not retained", string(data), err)
	}
}

func withCompanionDir(t *testing.T, fn func(string) (string, error)) func() {
	t.Helper()
	prev := resolveCompanionDir
	resolveCompanionDir = fn
	return func() { resolveCompanionDir = prev }
}

func TestCompanionStopWaitsForPortReleaseBeforeDiscardingOwnership(t *testing.T) {
	dir := t.TempDir()
	defer withCompanionDir(t, func(string) (string, error) { return dir, nil })()
	cmd := exec.Command(os.Args[0], "-test.run=^TestCompanionDelayedStopFixture$")
	cmd.Env = append(os.Environ(), "VROOLI_COMPANION_DELAYED_STOP=1")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = cmd.Process.Kill(); _ = cmd.Wait() }()
	address, err := bufio.NewReader(stdout).ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	address = strings.TrimSpace(address)
	if err := os.WriteFile(filepath.Join(dir, "edge.pid"), []byte(strconv.Itoa(cmd.Process.Pid)), 0600); err != nil {
		t.Fatal(err)
	}
	if err := stopCompanion("fixture", ResourceCompanion{Name: "edge"}); err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("tcp", address)
	if err != nil {
		t.Fatalf("stop returned before public port release: %v", err)
	}
	listener.Close()
	if _, err := os.Stat(filepath.Join(dir, "edge.pid")); !os.IsNotExist(err) {
		t.Fatal("finished ownership not cleared", err)
	}
}

func TestCompanionDelayedStopFixture(t *testing.T) {
	if os.Getenv("VROOLI_COMPANION_DELAYED_STOP") != "1" {
		return
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		os.Exit(2)
	}
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	fmt.Fprintln(os.Stdout, listener.Addr().String())
	<-signals
	time.Sleep(200 * time.Millisecond)
	listener.Close()
	os.Exit(0)
}
