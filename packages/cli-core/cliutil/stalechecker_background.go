package cliutil

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// refreshInBackground starts the owner refresh for a stale CLI without waiting
// for it. It is best effort and silent (unless VROOLI_CLI_DEBUG_STALE is set):
// the invoking command is latency-bound and must run the current binary
// whatever happens here.
//
// Launches are deduplicated per CLI by an exclusive lock on a state file that
// the refresh process itself inherits, so the lock lasts exactly as long as the
// refresh runs and is released by the kernel if it dies. A launch is also
// skipped within RefreshCooldown of the previous one, so a refresh that keeps
// failing is not retried by every hook call.
func (c *StaleChecker) refreshInBackground(spec FreshnessSpec) {
	cmd, _, err := c.refreshCommand(spec)
	if err != nil {
		c.debugRefresh("background refresh unavailable: %v\n", err)
		return
	}

	stateDir := c.refreshStateDir()
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		c.debugRefresh("background refresh state dir unavailable: %v\n", err)
		return
	}
	base := filepath.Join(stateDir, c.refreshKey())

	lock, acquired, err := acquireRefreshLock(base + ".lock")
	if err != nil {
		c.debugRefresh("background refresh lock unavailable: %v\n", err)
		return
	}
	if !acquired {
		c.debugRefresh("background refresh already running\n")
		return
	}
	// The parent's descriptor is closed on return; the child's inherited copy
	// keeps the lock held.
	defer lock.Close()

	if last, ok := readRefreshStamp(lock); ok && time.Since(last) < c.refreshCooldown() {
		c.debugRefresh("background refresh skipped: last launch %s ago\n", time.Since(last).Round(time.Second))
		return
	}
	if err := writeRefreshStamp(lock, time.Now()); err != nil {
		c.debugRefresh("background refresh stamp unavailable: %v\n", err)
		return
	}

	logFile, err := os.OpenFile(base+".log", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		c.debugRefresh("background refresh log unavailable: %v\n", err)
		return
	}
	defer logFile.Close()

	cmd.Stdin = nil
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.ExtraFiles = []*os.File{lock}
	detachProcess(cmd)

	if err := c.backgroundStarter()(cmd); err != nil {
		c.debugRefresh("background refresh did not start: %v\n", err)
		return
	}
	c.debugRefresh("started background refresh of %s CLI (log %s)\n", c.appLabel(), base+".log")
}

func (c *StaleChecker) backgroundStarter() func(*exec.Cmd) error {
	if c.BackgroundStarter != nil {
		return c.BackgroundStarter
	}
	return func(cmd *exec.Cmd) error {
		if err := cmd.Start(); err != nil {
			return err
		}
		return cmd.Process.Release()
	}
}

func (c *StaleChecker) debugRefresh(format string, args ...interface{}) {
	if truthy(os.Getenv(debugStaleEnvVar)) {
		c.log("[%s] "+format, append([]interface{}{c.appLabel()}, args...)...)
	}
}

func (c *StaleChecker) refreshCooldown() time.Duration {
	if c.RefreshCooldown > 0 {
		return c.RefreshCooldown
	}
	return defaultRefreshCooldown
}

func (c *StaleChecker) refreshStateDir() string {
	if strings.TrimSpace(c.RefreshStateDir) != "" {
		return c.RefreshStateDir
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		return filepath.Join(home, ".vrooli", "cli-refresh")
	}
	return filepath.Join(os.TempDir(), "vrooli-cli-refresh")
}

// refreshKey names the lock and log of one installed CLI.
func (c *StaleChecker) refreshKey() string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '.':
			return r
		}
		return '_'
	}, c.InstallKind+"-"+c.InstallName)
}

// The lock file doubles as the record of the last launch: its content is the
// launch time, so an empty file means "never launched".
func readRefreshStamp(f *os.File) (time.Time, bool) {
	buf := make([]byte, 64)
	n, _ := f.ReadAt(buf, 0)
	stamp, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(string(buf[:n])))
	return stamp, err == nil
}

func writeRefreshStamp(f *os.File, at time.Time) error {
	if err := f.Truncate(0); err != nil {
		return err
	}
	_, err := fmt.Fprint(f, at.UTC().Format(time.RFC3339Nano))
	return err
}
