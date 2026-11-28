//go:build linux

package codex

import (
	"context"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"testing"
	"time"
)

// The test binary doubles as the fake Codex install: a symlink to it placed
// under a session home's packages/app-server-daemon tree runs as the daemon
// (argv[0] is the install path, exactly what the scanner attributes by) and,
// invoked as `app-server daemon stop`, plays Codex's own graceful stop.
func TestMain(m *testing.M) {
	args := os.Args[1:]
	switch {
	case len(args) >= 3 && args[0] == "app-server" && args[1] == "daemon" && args[2] == "stop":
		os.Exit(fakeOwnerStop())
	case len(args) >= 1 && args[0] == "app-server":
		fakeDaemon(args)
	}
	os.Exit(m.Run())
}

// fakeOwnerStop records that the owner path ran against CODEX_HOME and asks the
// daemon recorded in daemon.pid to exit, like `codex app-server daemon stop`.
func fakeOwnerStop() int {
	home := os.Getenv("CODEX_HOME")
	if home == "" {
		return 2
	}
	_ = os.WriteFile(filepath.Join(home, "owner-stop-requested"), nil, 0o600)
	if raw, err := os.ReadFile(filepath.Join(home, "daemon.pid")); err == nil {
		if pid, err := strconv.Atoi(string(raw)); err == nil {
			_ = syscall.Kill(pid, syscall.SIGTERM)
		}
	}
	return 0
}

func fakeDaemon(args []string) {
	for _, arg := range args {
		if arg == "--ignore-term" {
			signal.Ignore(syscall.SIGTERM)
		}
	}
	time.Sleep(10 * time.Minute)
	os.Exit(0)
}

type daemonFixture struct {
	root string
}

func newDaemonFixture(t *testing.T) daemonFixture {
	t.Helper()
	return daemonFixture{root: t.TempDir()}
}

func (f daemonFixture) installPath(sessionID string) string {
	return filepath.Join(SessionHomePath(f.root, sessionID), homeDaemonInstallDir, "releases", "0.0.0-test", "bin", "codex")
}

// start launches a fake daemon process from the session's install path. The
// first daemon of a home also records daemon.pid, as Codex does.
func (f daemonFixture) start(t *testing.T, sessionID string, extra ...string) *exec.Cmd {
	t.Helper()
	exe := f.installPath(sessionID)
	if err := os.MkdirAll(filepath.Dir(exe), 0o755); err != nil {
		t.Fatal(err)
	}
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(exe); err != nil {
		if err := os.Symlink(self, exe); err != nil {
			t.Fatal(err)
		}
	}
	cmd := exec.Command(exe, append([]string{"app-server", "--listen", "unix://", "--managed-daemon"}, extra...)...)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	// The goroutine reaps the child, as init does for a real daemon, so an
	// exited process never lingers as a zombie.
	reaped := make(chan struct{})
	go func() { _ = cmd.Wait(); close(reaped) }()
	t.Cleanup(func() { _ = cmd.Process.Kill(); <-reaped })
	pidFile := filepath.Join(SessionHomePath(f.root, sessionID), "daemon.pid")
	if err := os.WriteFile(pidFile, []byte(strconv.Itoa(cmd.Process.Pid)), 0o600); err != nil {
		t.Fatal(err)
	}
	waitFor(t, func() bool { return len(mustDaemons(t, f.root)[sessionID]) > 0 })
	return cmd
}

func mustDaemons(t *testing.T, root string) map[string][]DaemonProcess {
	t.Helper()
	got, err := SessionDaemons(root)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatal("condition not reached")
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestDaemonSessionIDAttributesOnlySessionHomeInstalls(t *testing.T) {
	prefixes := sessionHomePrefixes("/state/web-console/sessions")
	cases := []struct{ exe, want string }{
		{"/state/web-console/sessions/codex/abc/packages/app-server-daemon/releases/1/bin/codex", "abc"},
		{"/state/web-console/sessions/codex/abc/packages/app-server-daemon/current/bin/codex-code-mode-host", "abc"},
		// The operator's shared install and other roots are never attributed.
		{"/home/op/.codex/packages/app-server-daemon/releases/1/bin/codex", ""},
		{"/other/sessions/codex/abc/packages/app-server-daemon/releases/1/bin/codex", ""},
		// Only the daemon install, not any program under a session home.
		{"/state/web-console/sessions/codex/abc/tmp/arg0/codex-arg0X/codex", ""},
		{"/state/web-console/sessions/codex/abc/packages/standalone/codex", ""},
		{"/state/web-console/sessions/codex//packages/app-server-daemon/x", ""},
		{"/state/web-console/sessions/codex/../packages/app-server-daemon/x", ""},
		{"/state/web-console/sessions/codex/abc", ""},
	}
	for _, tc := range cases {
		if got := daemonSessionID(tc.exe, prefixes); got != tc.want {
			t.Errorf("daemonSessionID(%q) = %q, want %q", tc.exe, got, tc.want)
		}
	}
}

func TestSessionDaemonsGroupsBySessionAndMeasuresMemory(t *testing.T) {
	f := newDaemonFixture(t)
	f.start(t, "session-a")
	f.start(t, "session-b")
	got := mustDaemons(t, f.root)
	if len(got) != 2 || len(got["session-a"]) != 1 || len(got["session-b"]) != 1 {
		t.Fatalf("daemons by session = %+v", got)
	}
	usage := SessionDaemonUsage(got)
	if usage["session-a"].Processes != 1 || usage["session-a"].MemoryBytes <= 0 {
		t.Fatalf("usage = %+v", usage["session-a"])
	}
	if other := mustDaemons(t, t.TempDir()); len(other) != 0 {
		t.Fatalf("daemons under an unrelated root = %+v", other)
	}
}

func TestStopSessionDaemonUsesCodexOwnerStopThenLeavesOtherSessionsRunning(t *testing.T) {
	f := newDaemonFixture(t)
	archived := f.start(t, "archived")
	archivedProc := procOf(t, archived)
	active := f.start(t, "active")

	usage, err := StopSessionDaemon(context.Background(), f.root, "archived", 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if usage.Processes != 1 || usage.MemoryBytes <= 0 {
		t.Fatalf("freed usage = %+v", usage)
	}
	marker := filepath.Join(SessionHomePath(f.root, "archived"), "owner-stop-requested")
	if _, err := os.Stat(marker); err != nil {
		t.Fatalf("Codex's own stop was not run against the session home: %v", err)
	}
	if processRunning(archivedProc) {
		t.Fatal("archived session daemon is still running")
	}
	// The home itself, where the transcript lives, is untouched.
	if _, err := os.Stat(filepath.Join(SessionHomePath(f.root, "archived"), "daemon.pid")); err != nil {
		t.Fatalf("session home changed: %v", err)
	}
	if err := syscall.Kill(active.Process.Pid, 0); err != nil || len(mustDaemons(t, f.root)["active"]) != 1 {
		t.Fatalf("another session's daemon was disturbed: %v", err)
	}
}

func TestStopSessionDaemonKillsWhatIgnoresGracefulStop(t *testing.T) {
	f := newDaemonFixture(t)
	stubborn := f.start(t, "stubborn", "--ignore-term")
	stubbornProc := procOf(t, stubborn)
	start := time.Now()
	if _, err := StopSessionDaemon(context.Background(), f.root, "stubborn", 300*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	if processRunning(stubbornProc) {
		t.Fatal("daemon that ignores SIGTERM survived the forced stop")
	}
	if elapsed := time.Since(start); elapsed > 4*time.Second {
		t.Fatalf("forced stop took %s, grace was 300ms", elapsed)
	}
}

func TestStopSessionDaemonIsIdempotentAndRestartable(t *testing.T) {
	f := newDaemonFixture(t)
	if usage, err := StopSessionDaemon(context.Background(), f.root, "never-ran", time.Second); err != nil || usage.Processes != 0 {
		t.Fatalf("no daemon: usage=%+v err=%v", usage, err)
	}
	f.start(t, "reopened")
	for i := 0; i < 2; i++ {
		if _, err := StopSessionDaemon(context.Background(), f.root, "reopened", 5*time.Second); err != nil {
			t.Fatal(err)
		}
	}
	if got := mustDaemons(t, f.root)["reopened"]; len(got) != 0 {
		t.Fatalf("daemon survived stop: %+v", got)
	}
	// Reopen: Codex starts a daemon on the same home again; stopping left
	// nothing behind that prevents it, and it is discoverable and stoppable.
	f.start(t, "reopened")
	if got := mustDaemons(t, f.root)["reopened"]; len(got) != 1 {
		t.Fatalf("daemon not available after reopen: %+v", got)
	}
}

func TestStopSessionDaemonRejectsPathLikeSessionIDs(t *testing.T) {
	for _, id := range []string{"", "../x", "a/b"} {
		if _, err := StopSessionDaemon(context.Background(), t.TempDir(), id, time.Second); err == nil {
			t.Errorf("session id %q was accepted", id)
		}
	}
}

// procOf captures the identity of a running fake daemon, to be checked after
// the stop.
func procOf(t *testing.T, cmd *exec.Cmd) DaemonProcess {
	t.Helper()
	ticks, _, ok := procStat(cmd.Process.Pid)
	if !ok {
		t.Fatalf("process %d is not running", cmd.Process.Pid)
	}
	return DaemonProcess{PID: cmd.Process.Pid, StartTicks: ticks}
}
