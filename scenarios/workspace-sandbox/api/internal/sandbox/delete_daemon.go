// Package sandbox.
//
// delete_daemon.go — deterministic per-sandbox fuse-overlayfs daemon
// teardown driven by Service.Delete. The Service.Delete path calls
// killDaemonsForSandbox(id) after the driver's Cleanup so the daemon
// is gone before the repo row is marked deleted. The background
// daemon reaper (daemon_reaper.go) stays as a safety net for
// API-crash paths only.
//
// I-MOUNT-1: Delete returns ⇒ no fuse-overlayfs daemon remains for
// that sandbox UUID. Pinned by delete_daemon_lifecycle_test.go.
//
// DOC: daemon-reaper seam — deterministic teardown.
package sandbox

import (
	"bytes"
	"context"
	"log"
	"strconv"
	"time"

	"github.com/google/uuid"
)

// DefaultDeleteScopeKillTimeout is the bounded wait for daemons to
// terminate during Delete. Kept short because Delete latency matters.
const DefaultDeleteScopeKillTimeout = 5 * time.Second

// killDaemonsForSandbox synchronously SIGTERMs (then SIGKILLs after
// termWait) every fuse-overlayfs daemon whose cmdline references id.
// Returns the number of daemons reaped. Best-effort: failures to kill
// individual daemons are logged but do not fail Delete — the
// background reaper's safety net catches anything left over.
func (s *Service) killDaemonsForSandbox(ctx context.Context, id uuid.UUID) int {
	procFS := s.procFS
	if procFS == nil {
		procFS = NewRealProcFS()
	}
	return s.killDaemonsForSandboxWith(ctx, id, procFS, DefaultDeleteScopeKillTimeout)
}

// killDaemonsForSandboxWith is the testable seam: callers pass a
// ProcFS implementation and a custom term-wait so unit tests can
// exercise the kill path without spawning real processes.
func (s *Service) killDaemonsForSandboxWith(ctx context.Context, id uuid.UUID, procFS ProcFS, termWait time.Duration) int {
	if s == nil {
		return 0
	}
	pids, err := procFS.List()
	if err != nil {
		log.Printf("delete-daemon: list /proc: %v", err)
		return 0
	}
	target := []byte(id.String())
	reaped := 0
	for _, pidStr := range pids {
		pid, parseErr := strconv.Atoi(pidStr)
		if parseErr != nil {
			continue
		}
		entry, openErr := procFS.Open(pidStr)
		if openErr != nil {
			continue
		}
		cmdline := entry.Cmdline()
		if !bytes.Contains(cmdline, []byte("fuse-overlayfs")) {
			continue
		}
		if !bytes.Contains(cmdline, target) {
			continue
		}
		if killErr := killDaemon(pid, termWait, s.clock); killErr != nil {
			log.Printf("delete-daemon: kill pid=%d uuid=%s: %v", pid, id, killErr)
			continue
		}
		reaped++
		log.Printf("delete-daemon: reaped pid=%d uuid=%s reason=delete-scope", pid, id)
		if s.metrics != nil {
			s.metrics.IncDaemonReaped("delete_scope")
		}
	}
	return reaped
}
