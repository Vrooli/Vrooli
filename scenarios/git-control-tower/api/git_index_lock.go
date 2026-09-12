package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// staleLockAge is the minimum age of an index.lock file before it is
// considered stale. A brief grace period avoids racing with a git process
// that just created the lock.
const staleLockAge = 5 * time.Second

// index.lock retry tuning. The per-repo mutex only serializes in-process
// writers; external git/IDE/pre-commit (vrooli hygiene) can create index.lock
// outside it, and the stale-lock cleaner will not touch a lock younger than
// staleLockAge. A short bounded retry covers that sub-staleLockAge window where
// a live external writer briefly holds the lock.
const indexLockMaxAttempts = 3

// indexLockBaseBackoff is a var (not const) so tests can shrink the wait.
var indexLockBaseBackoff = 150 * time.Millisecond

// outputIndicatesIndexLock reports whether git output/stderr shows a failure to
// acquire .git/index.lock because another process currently holds it. It matches
// the canonical message: "fatal: Unable to create '.../.git/index.lock': File exists."
func outputIndicatesIndexLock(out []byte) bool {
	s := string(out)
	if !strings.Contains(s, "index.lock") {
		return false
	}
	return strings.Contains(s, "File exists") || strings.Contains(s, "Unable to create")
}

// sleepBackoff waits before the next retry attempt with linear backoff plus
// jitter, aborting early if the context is cancelled.
func sleepBackoff(ctx context.Context, attempt int, base time.Duration) error {
	wait := time.Duration(attempt) * base
	// Jitter from the wall clock spreads retries across processes without a weak
	// RNG (math/rand trips gosec G404, and crypto-grade randomness is unwarranted
	// for backoff timing).
	if half := int64(base / 2); half > 0 {
		wait += time.Duration(time.Now().UnixNano() % (half + 1))
	}
	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// retryOnIndexLock runs build()+run() and, when the git output signals index.lock
// contention, retries with bounded backoff up to attempts times. Only lock
// contention is retried — genuine git failures return immediately. On exhaustion
// the last (output, error) is returned unchanged so callers surface the real error.
func retryOnIndexLock(
	ctx context.Context,
	attempts int,
	base time.Duration,
	build func() *exec.Cmd,
	run func(*exec.Cmd) ([]byte, error),
) ([]byte, error) {
	var out []byte
	var err error
	for attempt := 1; attempt <= attempts; attempt++ {
		out, err = run(build())
		if err == nil || !outputIndicatesIndexLock(out) {
			return out, err
		}
		if attempt == attempts {
			break
		}
		if sleepErr := sleepBackoff(ctx, attempt, base); sleepErr != nil {
			return out, err // context cancelled; surface the original error
		}
	}
	return out, err
}

// execWithIndexLockRetry is the production-tuned wrapper used by index-modifying
// git commands.
func execWithIndexLockRetry(
	ctx context.Context,
	build func() *exec.Cmd,
	run func(*exec.Cmd) ([]byte, error),
) ([]byte, error) {
	return retryOnIndexLock(ctx, indexLockMaxAttempts, indexLockBaseBackoff, build, run)
}

// runCombined runs a command capturing stdout+stderr together, matching the
// error semantics of exec.Cmd.CombinedOutput.
func runCombined(cmd *exec.Cmd) ([]byte, error) {
	return cmd.CombinedOutput()
}

// cleanStaleLock removes .git/index.lock when it exists but is no longer
// held by a running process. This recovers from crashed or killed git
// operations that left the lock behind, which would otherwise block all
// subsequent index-modifying git commands.
//
// The function is intentionally conservative:
//   - It only acts when the lock file is older than staleLockAge.
//   - On Linux it reads /proc/<pid>/cmdline to verify the owner is a git
//     process. If /proc is unavailable it falls back to age-only heuristics.
//   - It logs every removal so operators can diagnose recurring crashes.
//
// Returns true if a stale lock was removed.
func cleanStaleLock(repoDir string) bool {
	lockPath := filepath.Join(repoDir, ".git", "index.lock")

	info, err := os.Stat(lockPath)
	if err != nil {
		return false // No lock file — nothing to do.
	}

	age := time.Since(info.ModTime())
	if age < staleLockAge {
		return false // Lock is fresh; a git process likely just created it.
	}

	// Attempt to read the PID from the lock file. Git itself does not write
	// a PID into index.lock (it relies on OS-level file locking), so the
	// file is typically empty. If we find a PID, verify the process is gone.
	pid := readLockPID(lockPath)
	if pid > 0 && isGitProcess(pid) {
		return false // Lock is held by a live git process.
	}

	if err := os.Remove(lockPath); err != nil {
		log.Printf("git-index-lock: failed to remove stale lock %s: %v", lockPath, err)
		return false
	}

	log.Printf("git-index-lock: removed stale lock %s (age %s)", lockPath, age.Round(time.Second))
	return true
}

// readLockPID attempts to read a numeric PID from the lock file.
// Returns 0 when the file is empty or does not contain a valid integer.
func readLockPID(lockPath string) int {
	data, err := os.ReadFile(lockPath)
	if err != nil || len(data) == 0 {
		return 0
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0
	}
	return pid
}

// isGitProcess checks whether the process with the given PID is alive and
// running a git command. On Linux this inspects /proc/<pid>/cmdline.
//
// The cmdline file contains NUL-separated arguments. We check whether the
// first argument (the executable) ends with "/git" or equals "git" to avoid
// false positives from binaries that happen to contain "git" in their name
// (e.g. "git-control-tower.test").
func isGitProcess(pid int) bool {
	cmdlinePath := fmt.Sprintf("/proc/%d/cmdline", pid)
	data, err := os.ReadFile(cmdlinePath)
	if err != nil {
		return false // Process doesn't exist or is inaccessible.
	}
	// Extract the first NUL-separated field (the executable path).
	exe := string(data)
	if idx := strings.IndexByte(exe, 0); idx >= 0 {
		exe = exe[:idx]
	}
	base := filepath.Base(exe)
	return base == "git"
}
