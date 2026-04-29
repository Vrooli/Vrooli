// Package sandbox.
//
// daemon_reaper.go — process-level orphan cleanup for fuse-overlayfs
// daemons whose sandbox UUID is no longer registered in the repository.
//
// Why this exists (2026-04-29):
//
// The filesystem orphan reconciler (orphan_reconciler.go) only walks
// directories. When a sandbox dir is removed but the daemon process
// keeps running (orphaned mount in flight, kernel held the mount past
// rmdir, etc.), the daemon survives indefinitely. We saw a daemon from
// April 26 still running after agent-manager had Delete()d the sandbox
// three days prior — drift the dir-walking reconciler can't see.
//
// This reconciler closes that gap.
//
// DOC: daemon-reaper seam. See docs/internal/SEAMS.md.
package sandbox

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"syscall"
	"time"

	"github.com/google/uuid"

	"workspace-sandbox/internal/types"
)

// upperDirUUIDPattern matches `--upperdir=<path>/<uuid>/...` arguments
// in fuse-overlayfs cmdlines. The UUID is the sandbox ID, since both
// project-overlay (BaseDir/<uuid>/upper) and home-overlay
// (HomeOverlayBaseDir/<uuid>/home-upper) layouts route their upper dir
// through the per-sandbox parent.
//
// Captures the UUID; tolerates `lowerdir=,upperdir=...` ordering by
// scanning the entire cmdline.
var upperDirUUIDPattern = regexp.MustCompile(
	`upperdir=[^,\x00]*/([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12})/`,
)

// DaemonReaperConfig tunes the reaper's safety windows.
type DaemonReaperConfig struct {
	// GracePeriod skips daemons younger than this. Prevents racing the
	// fuse-overlayfs spawn during sandbox creation: a daemon that was
	// started 200ms ago for a sandbox-creation request still in flight
	// looks orphaned because the sandbox row hasn't been committed yet.
	// Default: 30s.
	GracePeriod time.Duration

	// TermWait is how long to wait between SIGTERM and SIGKILL.
	// Default: 5s.
	TermWait time.Duration
}

// DefaultDaemonReaperConfig returns sensible defaults.
func DefaultDaemonReaperConfig() DaemonReaperConfig {
	return DaemonReaperConfig{
		GracePeriod: 30 * time.Second,
		TermWait:    5 * time.Second,
	}
}

// DaemonReapReport summarizes one reaper pass.
type DaemonReapReport struct {
	// Scanned is how many fuse-overlayfs processes were inspected.
	Scanned int
	// Reaped is how many were SIGTERM'd (and possibly SIGKILL'd).
	Reaped int
	// ReapedPIDs lists the PIDs that were killed.
	ReapedPIDs []int
	// SkippedYoung is how many were within the grace period.
	SkippedYoung int
	// SkippedAlive is how many were owned by a still-active sandbox.
	SkippedAlive int
	// Duration is how long the pass took.
	Duration time.Duration
}

// ReconcileStaleDaemons greps /proc for fuse-overlayfs processes whose
// upperdir UUID does not match any non-deleted sandbox in the repo, and
// kills them (SIGTERM, then SIGKILL after TermWait).
//
// Idempotent and best-effort: failures are logged, not surfaced.
//
// DOC: daemon-reaper seam.
func (s *Service) ReconcileStaleDaemons(ctx context.Context) DaemonReapReport {
	return s.reconcileStaleDaemonsWithConfig(ctx, DefaultDaemonReaperConfig(), NewRealProcFS())
}

// reconcileStaleDaemonsWithConfig is the testable seam: takes an
// fs.FS rooted at /proc plus the config, so unit tests can supply a
// synthetic fixture (testing/fstest.MapFS) without spawning processes.
func (s *Service) reconcileStaleDaemonsWithConfig(ctx context.Context, cfg DaemonReaperConfig, procFS interface {
	Open(string) (procEntry, error)
	List() ([]string, error)
},
) DaemonReapReport {
	start := time.Now()
	report := DaemonReapReport{}
	if s == nil || s.repo == nil {
		return report
	}

	pids, err := procFS.List()
	if err != nil {
		log.Printf("daemon-reaper: list /proc: %v", err)
		return report
	}

	now := time.Now()
	for _, pidStr := range pids {
		pid, err := strconv.Atoi(pidStr)
		if err != nil {
			continue // not a PID dir
		}
		entry, err := procFS.Open(pidStr)
		if err != nil {
			continue
		}
		cmdline := entry.Cmdline()
		if !bytes.Contains(cmdline, []byte("fuse-overlayfs")) {
			continue
		}
		report.Scanned++

		match := upperDirUUIDPattern.FindSubmatch(cmdline)
		if match == nil {
			// Daemon without an upperdir we can attribute — leave it alone.
			// (Could be operator-launched fuse-overlayfs unrelated to us.)
			continue
		}
		id, err := uuid.Parse(string(match[1]))
		if err != nil {
			continue
		}

		// Grace period: skip very young daemons (sandbox creation in flight).
		if started := entry.StartTime(); !started.IsZero() && now.Sub(started) < cfg.GracePeriod {
			report.SkippedYoung++
			continue
		}

		// Repo lookup: skip live sandboxes; reap deleted/missing.
		if !s.daemonOwnerIsOrphan(ctx, id) {
			report.SkippedAlive++
			continue
		}

		if err := killDaemon(pid, cfg.TermWait); err != nil {
			log.Printf("daemon-reaper: kill pid=%d uuid=%s: %v", pid, id, err)
			continue
		}
		report.Reaped++
		report.ReapedPIDs = append(report.ReapedPIDs, pid)
		log.Printf("daemon-reaper: reaped pid=%d uuid=%s reason=orphan",
			pid, id)
		s.logOrphanAuditEvent(ctx, id, "sandbox.daemon-reaped", map[string]interface{}{
			"pid":    pid,
			"reason": "fuse-overlayfs daemon outlived sandbox",
		})
	}

	report.Duration = time.Since(start)
	return report
}

// daemonOwnerIsOrphan returns true if the sandbox referenced by id is
// not in the repo or is marked deleted. Distinct from isOrphan() in
// orphan_reconciler.go because that function refuses to act on
// repo-error states (defensive); we want the same caution here.
func (s *Service) daemonOwnerIsOrphan(ctx context.Context, id uuid.UUID) bool {
	if s.repo == nil {
		return false
	}
	sb, err := s.repo.Get(ctx, id)
	if err == nil {
		return sb != nil && sb.Status == types.StatusDeleted
	}
	var notFound *types.NotFoundError
	return errors.As(err, &notFound)
}

// killDaemon sends SIGTERM, waits up to termWait, then SIGKILL.
// Returns nil if the process is gone after the operation.
func killDaemon(pid int, termWait time.Duration) error {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("find process: %w", err)
	}
	if err := proc.Signal(syscall.SIGTERM); err != nil {
		// ESRCH = already gone; treat as success.
		if errors.Is(err, syscall.ESRCH) || errors.Is(err, os.ErrProcessDone) {
			return nil
		}
		return fmt.Errorf("sigterm: %w", err)
	}
	deadline := time.Now().Add(termWait)
	for time.Now().Before(deadline) {
		if err := proc.Signal(syscall.Signal(0)); err != nil {
			return nil // process gone
		}
		time.Sleep(100 * time.Millisecond)
	}
	if err := proc.Signal(syscall.SIGKILL); err != nil {
		if errors.Is(err, syscall.ESRCH) || errors.Is(err, os.ErrProcessDone) {
			return nil
		}
		return fmt.Errorf("sigkill: %w", err)
	}
	return nil
}

// FormatDaemonReapReport renders a DaemonReapReport for log output.
func FormatDaemonReapReport(r DaemonReapReport) string {
	if r.Scanned == 0 {
		return "daemon-reaper: 0 fuse-overlayfs processes seen"
	}
	return fmt.Sprintf(
		"daemon-reaper: scanned=%d reaped=%d skipped-young=%d skipped-alive=%d (%v)",
		r.Scanned, r.Reaped, r.SkippedYoung, r.SkippedAlive, r.Duration.Round(time.Millisecond),
	)
}

// procEntry is the read-only view of a /proc/<pid> directory the
// reaper needs. Implementations: realProcEntry (production) and
// fixture-based (tests).
type procEntry interface {
	// Cmdline returns the contents of /proc/<pid>/cmdline. Args are
	// NUL-separated; we just need substring matching, not parsing.
	Cmdline() []byte

	// StartTime returns the process start time (best-effort). Zero
	// when unavailable.
	StartTime() time.Time
}

// realProcFS implements the fs interface against the host /proc.
type realProcFS struct{ root string }

// NewRealProcFS returns a /proc-rooted reaper FS.
func NewRealProcFS() interface {
	Open(string) (procEntry, error)
	List() ([]string, error)
} {
	return &realProcFS{root: "/proc"}
}

func (p *realProcFS) List() ([]string, error) {
	entries, err := os.ReadDir(p.root)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		out = append(out, e.Name())
	}
	return out, nil
}

func (p *realProcFS) Open(pid string) (procEntry, error) {
	return &realProcEntry{path: filepath.Join(p.root, pid)}, nil
}

type realProcEntry struct{ path string }

func (e *realProcEntry) Cmdline() []byte {
	data, _ := os.ReadFile(filepath.Join(e.path, "cmdline"))
	return data
}

func (e *realProcEntry) StartTime() time.Time {
	info, err := os.Stat(e.path)
	if err != nil {
		return time.Time{}
	}
	return info.ModTime()
}
