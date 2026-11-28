package codex

// A Codex TUI started with a per-session CODEX_HOME launches its own app-server
// daemon from that home's packages/app-server-daemon install: the app server
// itself, a detached updater loop, and a code-mode host. Nothing in Codex stops
// them when the TUI exits, so an archived or dismissed session kept ~60 MB to
// 1 GB of daemon alive indefinitely (the oldest for 17 days, 2026-11-21).
//
// This file is the one owner of that daemon's lifecycle on the Web Console
// side. Codex owns starting it: a TUI opened on the home starts the daemon on
// demand, so stopping it here never blocks reopen, unarchive, or recovery.
// A daemon is attributed to a session by the executable path it runs from
// (<state root>/codex/<session>/packages/app-server-daemon/...), never by its
// environment, and never to the operator's shared ~/.codex install.

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/envkit-go"
)

const (
	// homeDaemonInstallDir is where Codex installs the daemon inside a home.
	homeDaemonInstallDir = "packages/app-server-daemon"
	// DefaultDaemonStopGrace bounds the graceful phase (Codex's own stop plus
	// SIGTERM) before survivors are killed.
	DefaultDaemonStopGrace = 15 * time.Second
	daemonKillWait         = 3 * time.Second
	daemonPollInterval     = 100 * time.Millisecond
)

// DaemonProcess is one running process of a session home's daemon install.
type DaemonProcess struct {
	PID int
	// StartTicks is the process start time from the kernel. A pid that was
	// reused by another process no longer matches and is never signalled.
	StartTicks uint64
	// Cmdline is the process argv; Cmdline[0] is the install path it runs from.
	Cmdline []string
	// MemoryBytes is resident plus swapped memory when it was listed.
	MemoryBytes int64
}

// DaemonUsage is what a session's daemon processes hold, or what stopping
// them freed.
type DaemonUsage struct {
	Processes   int
	MemoryBytes int64
}

// managed reports whether this is the app-server itself, the process Codex's
// own `daemon stop` shuts down.
func (p DaemonProcess) managed() bool {
	for _, arg := range p.Cmdline {
		if arg == "--managed-daemon" {
			return true
		}
	}
	return false
}

// SessionDaemons lists the running daemon processes of every per-session Codex
// home under sessionStateRoot, grouped by session id. One process-table scan
// serves any number of sessions.
func SessionDaemons(sessionStateRoot string) (map[string][]DaemonProcess, error) {
	prefixes := sessionHomePrefixes(sessionStateRoot)
	procs, err := scanProcessesUnder(prefixes)
	if err != nil {
		return nil, err
	}
	out := map[string][]DaemonProcess{}
	for _, proc := range procs {
		if id := daemonSessionID(proc.Cmdline[0], prefixes); id != "" {
			out[id] = append(out[id], proc)
		}
	}
	return out, nil
}

// SessionDaemonUsage summarizes the daemons SessionDaemons found, by session.
func SessionDaemonUsage(daemons map[string][]DaemonProcess) map[string]DaemonUsage {
	out := make(map[string]DaemonUsage, len(daemons))
	for id, procs := range daemons {
		out[id] = usageOf(procs)
	}
	return out
}

// StopSessionDaemon stops the daemon of one session's home: Codex's own
// graceful stop first, SIGTERM for what it leaves (the updater loop, the
// code-mode host), then SIGKILL for anything still alive once grace has passed.
// It reports what was freed. A session with no daemon is a successful no-op.
// The caller decides that the session is not in use; the daemon of a session
// whose TUI is running must never be stopped.
func StopSessionDaemon(ctx context.Context, sessionStateRoot, sessionID string, grace time.Duration) (DaemonUsage, error) {
	if sessionID == "" || filepath.Base(sessionID) != sessionID {
		return DaemonUsage{}, os.ErrInvalid
	}
	all, err := SessionDaemons(sessionStateRoot)
	if err != nil {
		return DaemonUsage{}, err
	}
	return stopDaemonProcesses(ctx, SessionHomePath(sessionStateRoot, sessionID), all[sessionID], grace)
}

func stopDaemonProcesses(ctx context.Context, home string, procs []DaemonProcess, grace time.Duration) (DaemonUsage, error) {
	usage := usageOf(procs)
	if len(procs) == 0 {
		return usage, nil
	}
	if grace <= 0 {
		grace = DefaultDaemonStopGrace
	}
	deadline := time.Now().Add(grace)
	for _, proc := range procs {
		if proc.managed() {
			requestCodexStop(ctx, home, proc, deadline)
			break
		}
	}
	for _, proc := range procs {
		_ = terminateProcess(proc, false)
	}
	remaining := waitForExit(ctx, procs, deadline)
	if len(remaining) > 0 {
		for _, proc := range remaining {
			_ = terminateProcess(proc, true)
		}
		remaining = waitForExit(ctx, remaining, time.Now().Add(daemonKillWait))
	}
	if len(remaining) > 0 {
		pids := make([]string, len(remaining))
		for i, proc := range remaining {
			pids[i] = fmt.Sprint(proc.PID)
		}
		return usage, fmt.Errorf("daemon processes of %s still running after kill: %s", home, strings.Join(pids, ","))
	}
	return usage, nil
}

// requestCodexStop runs the installed Codex's own `app-server daemon stop`
// against the home, so the app server shuts down through its owner before any
// signal. It is best effort: a failure falls through to the signal phase.
func requestCodexStop(ctx context.Context, home string, managed DaemonProcess, deadline time.Time) {
	exe := managed.Cmdline[0]
	if _, err := os.Stat(exe); err != nil {
		return
	}
	stopCtx, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()
	cmd := exec.CommandContext(stopCtx, exe, "app-server", "daemon", "stop")
	cmd.Env = appServerEnvironment(os.Environ(), []string{"CODEX_HOME=" + home}, envkit.DefaultPlatform())
	_ = cmd.Run()
}

// waitForExit polls until every process has exited or the deadline passes, and
// returns the survivors.
func waitForExit(ctx context.Context, procs []DaemonProcess, deadline time.Time) []DaemonProcess {
	remaining := procs
	for {
		next := remaining[:0:0]
		for _, proc := range remaining {
			if processRunning(proc) {
				next = append(next, proc)
			}
		}
		remaining = next
		if len(remaining) == 0 || !time.Now().Before(deadline) || ctx.Err() != nil {
			return remaining
		}
		time.Sleep(daemonPollInterval)
	}
}

func usageOf(procs []DaemonProcess) DaemonUsage {
	usage := DaemonUsage{Processes: len(procs)}
	for _, proc := range procs {
		usage.MemoryBytes += proc.MemoryBytes
	}
	return usage
}

// sessionHomePrefixes are the "<root>/codex/" directories a daemon path can sit
// under: the configured root and, when it is reached through a symlink, its
// resolved form (a process records the path it was launched with).
func sessionHomePrefixes(sessionStateRoot string) []string {
	if sessionStateRoot == "" {
		return nil
	}
	prefixes := []string{filepath.Join(filepath.Clean(sessionStateRoot), "codex") + string(filepath.Separator)}
	if resolved, err := filepath.EvalSymlinks(sessionStateRoot); err == nil && resolved != filepath.Clean(sessionStateRoot) {
		prefixes = append(prefixes, filepath.Join(resolved, "codex")+string(filepath.Separator))
	}
	return prefixes
}

// daemonSessionID extracts the session id from an executable path of the form
// <prefix><session>/packages/app-server-daemon/..., or "" when it is not one.
func daemonSessionID(exe string, prefixes []string) string {
	for _, prefix := range prefixes {
		rest, ok := strings.CutPrefix(exe, prefix)
		if !ok {
			continue
		}
		id, tail, found := strings.Cut(rest, string(filepath.Separator))
		if !found || id == "" || id == "." || id == ".." {
			continue
		}
		if strings.HasPrefix(tail, homeDaemonInstallDir+string(filepath.Separator)) {
			return id
		}
	}
	return ""
}
