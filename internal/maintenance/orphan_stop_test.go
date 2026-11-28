package maintenance

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/process"
)

type orphanTestHandle struct {
	alive  bool
	closed bool
	signal func(bool) error
}

func (h *orphanTestHandle) Alive() (bool, error) { return h.alive, nil }
func (h *orphanTestHandle) Signal(force bool) error {
	if h.signal != nil {
		return h.signal(force)
	}
	h.alive = false
	return nil
}
func (h *orphanTestHandle) Close() error { h.closed = true; return nil }

func stubOrphanHandles(t *testing.T) {
	t.Helper()
	previous := openOrphanHandleFn
	t.Cleanup(func() { openOrphanHandleFn = previous })
	openOrphanHandleFn = func(pid int) (process.Handle, error) {
		h := &orphanTestHandle{alive: true}
		h.signal = func(force bool) error {
			if err := killProcessFn(pid, force); err != nil {
				return err
			}
			h.alive = false
			return nil
		}
		return h, nil
	}
}

func TestOrphanStopPinsBeforeQualificationAndNeverSignalsReplacement(t *testing.T) {
	root, home := t.TempDir(), t.TempDir()
	entry := processTableEntry{PID: 5200, PPID: 1, Cgroup: "/", Executable: filepath.Join(root, "scenarios", "fixture", "api", "server")}
	h := &orphanTestHandle{alive: true}
	previousOpen, previousRead, previousList := openOrphanHandleFn, readProcessEntryFn, listProcessTableFn
	t.Cleanup(func() {
		openOrphanHandleFn, readProcessEntryFn, listProcessTableFn = previousOpen, previousRead, previousList
	})
	pinned := false
	openOrphanHandleFn = func(int) (process.Handle, error) { pinned = true; return h, nil }
	readProcessEntryFn = func(int) (processTableEntry, bool) {
		if !pinned {
			t.Fatal("qualification preceded pin")
		}
		return entry, true
	}
	listProcessTableFn = func() (map[int]processTableEntry, error) {
		// The original exits while a replacement with the same image and PID
		// becomes visible. Its apparent orphan status cannot inherit the signal.
		h.alive = false
		return map[int]processTableEntry{entry.PID: entry}, nil
	}
	h.signal = func(bool) error { t.Fatal("signal reached a replacement PID"); return nil }
	if err := NewController(root, home).stopOrphan(context.Background(), entry.PID); !errors.Is(err, errOrphanSkipped) {
		t.Fatal(err)
	}
	if !h.closed {
		t.Fatal("pinned process handle leaked")
	}
}

func TestOrphanStopRefusesUnknownHandleAndNewlyOwnedService(t *testing.T) {
	root, home := t.TempDir(), t.TempDir()
	entry := processTableEntry{PID: 5200, PPID: 1, Cgroup: "/", Executable: filepath.Join(root, "scenarios", "fixture", "api", "server")}
	previousOpen, previousRead, previousList := openOrphanHandleFn, readProcessEntryFn, listProcessTableFn
	t.Cleanup(func() {
		openOrphanHandleFn, readProcessEntryFn, listProcessTableFn = previousOpen, previousRead, previousList
	})
	openOrphanHandleFn = func(int) (process.Handle, error) { return nil, errors.New("stable handles unsupported") }
	if err := NewController(root, home).stopOrphan(context.Background(), entry.PID); err == nil {
		t.Fatal("unknown handle permitted cleanup")
	}
	h := &orphanTestHandle{alive: true, signal: func(bool) error { t.Fatal("signal reached a newly owned service"); return nil }}
	openOrphanHandleFn = func(int) (process.Handle, error) { return h, nil }
	readProcessEntryFn = func(int) (processTableEntry, bool) { return entry, true }
	listProcessTableFn = func() (map[int]processTableEntry, error) {
		owned := entry
		owned.Cgroup = "/vrooli.slice/vrooli-services.slice/vrooli-service-fixture-start-api.scope"
		return map[int]processTableEntry{entry.PID: owned}, nil
	}
	if err := NewController(root, home).stopOrphan(context.Background(), entry.PID); !errors.Is(err, errOrphanSkipped) {
		t.Fatal(err)
	}
	if !h.closed {
		t.Fatal("handle leaked after owner adoption")
	}
}

func TestGenericOrphanCleanupLeavesNativeServiceTreesToTheirOwner(t *testing.T) {
	root, home := t.TempDir(), t.TempDir()
	for _, unit := range []string{"vrooli-services.slice/vrooli-service-device-control-start-api.scope", "vrooli-services.slice/vrooli-svc-resource-update-exact.service", "vrooli-operations.slice/vrooli-op-live-attempt.service"} {
		entry := processTableEntry{PID: 5200, PPID: 1, Executable: filepath.Join(root, "resources", "fixture", "daemon"), Cgroup: "/vrooli.slice/" + unit}
		if got := collectOrphans(root, home, map[int]processTableEntry{entry.PID: entry}, nil, nil); len(got) != 0 {
			t.Fatalf("native tree was classified orphaned: %+v", got)
		}
	}
}

func TestOrphanStopDoesNotAcknowledgeUnverifiedExit(t *testing.T) {
	root, home := t.TempDir(), t.TempDir()
	entry := processTableEntry{PID: 5200, PPID: 1, Cgroup: "/", Executable: filepath.Join(root, "scenarios", "fixture", "api", "server")}
	previousOpen, previousRead, previousList := openOrphanHandleFn, readProcessEntryFn, listProcessTableFn
	t.Cleanup(func() {
		openOrphanHandleFn, readProcessEntryFn, listProcessTableFn = previousOpen, previousRead, previousList
	})
	var forceSignals int
	h := &orphanTestHandle{alive: true, signal: func(force bool) error {
		if force {
			forceSignals++
		}
		return nil
	}}
	openOrphanHandleFn = func(int) (process.Handle, error) { return h, nil }
	readProcessEntryFn = func(int) (processTableEntry, bool) { return entry, true }
	listProcessTableFn = func() (map[int]processTableEntry, error) { return map[int]processTableEntry{entry.PID: entry}, nil }
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	if err := NewController(root, home).stopOrphan(ctx, entry.PID); err == nil {
		t.Fatal("live pinned process was reported stopped")
	}
	if forceSignals != 1 || !h.closed {
		t.Fatalf("force=%d handle_closed=%v", forceSignals, h.closed)
	}
}

func TestOrphanStopRefusesForceAfterNativeOwnerReappears(t *testing.T) {
	root, home := t.TempDir(), t.TempDir()
	entry := processTableEntry{PID: 5200, PPID: 1, Cgroup: "/", Executable: filepath.Join(root, "scenarios", "fixture", "api", "server")}
	previousOpen, previousRead, previousList := openOrphanHandleFn, readProcessEntryFn, listProcessTableFn
	t.Cleanup(func() {
		openOrphanHandleFn, readProcessEntryFn, listProcessTableFn = previousOpen, previousRead, previousList
	})
	var term, force int
	h := &orphanTestHandle{alive: true, signal: func(hard bool) error {
		if hard {
			force++
		} else {
			term++
		}
		return nil
	}}
	openOrphanHandleFn = func(int) (process.Handle, error) { return h, nil }
	readProcessEntryFn = func(int) (processTableEntry, bool) { return entry, true }
	listProcessTableFn = func() (map[int]processTableEntry, error) {
		current := entry
		if term > 0 {
			current.Cgroup = "/vrooli.slice/vrooli-services.slice/vrooli-service-fixture-api.scope"
		}
		return map[int]processTableEntry{entry.PID: current}, nil
	}
	if err := NewController(root, home).stopOrphan(context.Background(), entry.PID); err == nil {
		t.Fatal("a newly owned live process was acknowledged stopped")
	}
	if term != 1 || force != 0 || !h.closed {
		t.Fatalf("term=%d force=%d closed=%v", term, force, h.closed)
	}
}
