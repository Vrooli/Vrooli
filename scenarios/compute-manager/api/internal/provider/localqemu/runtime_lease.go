package localqemu

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const capacityHeartbeatInterval = 5 * time.Minute

func (a *Adapter) requireReady() error {
	if a == nil || !a.RequireCapacity {
		return nil
	}
	a.leaseMu.Lock()
	defer a.leaseMu.Unlock()
	if !a.ready {
		if a.readyErr != nil {
			return fmt.Errorf("local-QEMU host admission is not ready: %w", a.readyErr)
		}
		return errors.New("local-QEMU host admission recovery has not completed")
	}
	if a.Admission == nil {
		return errors.New("shared host capacity admission is unavailable")
	}
	return nil
}

// RecoverCapacity must complete before the local-QEMU provider serves requests.
// Every persisted guest is stopped/cleaned if its exact capacity receipt cannot
// be reattached; no unleased guest is left available to this provider.
func (a *Adapter) RecoverCapacity(ctx context.Context) error {
	if a == nil || !a.RequireCapacity {
		return nil
	}
	a.leaseMu.Lock()
	a.ready = false
	a.readyErr = nil
	a.leaseMu.Unlock()
	if a.Admission == nil {
		return a.setNotReady(errors.New("shared host capacity broker is unavailable"))
	}
	root, err := a.dataRoot(ctx)
	if err != nil {
		return a.setNotReady(err)
	}
	instancesRoot := filepath.Join(root, "local-qemu")
	if err := os.MkdirAll(instancesRoot, 0o700); err != nil {
		return a.setNotReady(fmt.Errorf("create local-QEMU owner directory: %w", err))
	}
	if err := a.lockManager(instancesRoot); err != nil {
		return a.setNotReady(err)
	}
	entries, err := os.ReadDir(instancesRoot)
	if errors.Is(err, os.ErrNotExist) {
		a.setReady()
		return nil
	}
	if err != nil {
		return a.setNotReady(err)
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		id := entry.Name()
		path, err := metadataPath(root, id)
		if err != nil {
			return a.setNotReady(err)
		}
		local, err := readOwnedInstance(root, id, path)
		if errors.Is(err, os.ErrNotExist) {
			workdir := filepath.Join(instancesRoot, id)
			receipt, found, lookupErr := a.Admission.FindOrphan(ctx, id, workdir)
			if lookupErr != nil {
				return a.setNotReady(fmt.Errorf("inspect uncommitted local-QEMU admission %s: %w", id, lookupErr))
			}
			if found {
				orphan := Instance{ID: id, Provider: "local-qemu", State: StatePlanned, Workdir: workdir,
					Image: filepath.Join(workdir, instanceDiskName), Capacity: &receipt}
				if err := a.stopAndClean(ctx, orphan, nil); err != nil {
					return a.setNotReady(fmt.Errorf("settle uncommitted local-QEMU admission %s: %w", id, err))
				}
				continue
			}
			contents, readErr := os.ReadDir(workdir)
			if readErr == nil && len(contents) == 0 {
				if removeErr := os.Remove(workdir); removeErr != nil {
					return a.setNotReady(fmt.Errorf("remove empty local-QEMU admission directory %s: %w", id, removeErr))
				}
				a.recordWrite(ctx)
				continue
			}
			return a.setNotReady(fmt.Errorf("local-QEMU directory %s has files but no persisted instance or capacity receipt; preserved for owner review", id))
		}
		if err != nil {
			return a.setNotReady(fmt.Errorf("recover local-QEMU instance %s: %w", id, err))
		}
		if local.State == StateStarting && local.PID <= 0 {
			pid, findErr := findQEMUProcessPID(id)
			if findErr != nil {
				return a.setNotReady(fmt.Errorf("recover local-QEMU process identity %s: %w", id, findErr))
			}
			if pid > 0 {
				local.PID, local.State, local.UpdatedAt = pid, StateRunning, a.now()
				if err := writeLocalInstance(path, local); err != nil {
					return a.setNotReady(fmt.Errorf("persist recovered QEMU process %s: %w", id, err))
				}
				a.recordWrite(ctx)
			}
		}
		alive := false
		if local.State == StateRunning {
			alive, err = verifyQEMUProcess(local.PID, local.ID)
			if err != nil {
				return a.setNotReady(fmt.Errorf("verify recovered local-QEMU process %s: %w", id, err))
			}
		}
		if local.State != StateRunning || !alive {
			var lease *QEMULease
			if local.Capacity != nil {
				lease, _ = a.Admission.Reattach(ctx, *local.Capacity)
			}
			if err := a.stopAndClean(ctx, local, lease); err != nil {
				return a.setNotReady(fmt.Errorf("settle stopped local-QEMU instance %s: %w", id, err))
			}
			continue
		}
		if local.Capacity == nil {
			return a.setNotReady(errors.Join(fmt.Errorf("local-QEMU instance %s has no persisted capacity receipt", id), a.stopAndClean(ctx, local, nil)))
		}
		lease, reattachErr := a.Admission.Reattach(ctx, *local.Capacity)
		if reattachErr != nil {
			cleanupErr := a.stopAndClean(ctx, local, lease)
			return a.setNotReady(errors.Join(fmt.Errorf("reattach local-QEMU capacity for %s: %w", id, reattachErr), cleanupErr))
		}
		if lease == nil || lease.Disk == nil || lease.Resources == nil {
			cleanupErr := a.stopAndClean(ctx, local, lease)
			return a.setNotReady(errors.Join(fmt.Errorf("local-QEMU capacity reattachment for %s returned an incomplete lease", id), cleanupErr))
		}
		a.keepLease(id, local, lease)
	}
	a.setReady()
	return nil
}

func verifyQEMUProcess(pid int, instanceID string) (bool, error) {
	if pid <= 0 {
		return false, nil
	}
	command, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(pid), "cmdline"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("verify local-QEMU process %d identity: %w", pid, err)
	}
	arguments := strings.Split(strings.TrimRight(string(command), "\x00"), "\x00")
	if !processArgsMatchQEMU(arguments, instanceID) {
		return false, fmt.Errorf("refusing to signal PID %d because it does not identify local-QEMU instance %s", pid, instanceID)
	}
	return true, nil
}

func processArgsMatchQEMU(arguments []string, instanceID string) bool {
	want := "vrooli-local-qemu:" + instanceID
	for i := 0; i+1 < len(arguments); i++ {
		if arguments[i] == "-name" && arguments[i+1] == want {
			return true
		}
	}
	return false
}

func findQEMUProcessPID(instanceID string) (int, error) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return 0, fmt.Errorf("read process table: %w", err)
	}
	found := 0
	for _, entry := range entries {
		pid, err := strconv.Atoi(entry.Name())
		if err != nil || pid <= 0 {
			continue
		}
		command, err := os.ReadFile(filepath.Join("/proc", entry.Name(), "cmdline"))
		if err != nil || len(command) == 0 {
			continue
		}
		arguments := strings.Split(strings.TrimRight(string(command), "\x00"), "\x00")
		if !processArgsMatchQEMU(arguments, instanceID) {
			continue
		}
		if found != 0 {
			return 0, fmt.Errorf("multiple QEMU processes claim instance %s", instanceID)
		}
		found = pid
	}
	return found, nil
}

// lockManager prevents two Compute Manager processes from concurrently
// reattaching the same renewable owner claims or stopping the same guest.
func (a *Adapter) lockManager(instancesRoot string) error {
	a.leaseMu.Lock()
	defer a.leaseMu.Unlock()
	if a.managerLock != nil {
		return nil
	}
	path := filepath.Join(instancesRoot, ".compute-manager.lock")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return fmt.Errorf("open local-QEMU owner lock: %w", err)
	}
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = file.Close()
		return fmt.Errorf("another Compute Manager process owns local-QEMU capacity recovery: %w", err)
	}
	a.managerLock = file
	return nil
}

// CloseCapacity relinquishes the single-owner process lock during service
// shutdown. Claims are intentionally not released: a restarting manager can
// reattach them after the old process identity is dead, while an extended
// outage naturally lapses the renewable lease and triggers guest cleanup.
func (a *Adapter) CloseCapacity() error {
	if a == nil {
		return nil
	}
	a.leaseMu.Lock()
	for _, item := range a.leases {
		item.cancel()
	}
	lock := a.managerLock
	a.managerLock = nil
	a.leaseMu.Unlock()
	if lock == nil {
		return nil
	}
	return lock.Close()
}

func (a *Adapter) setNotReady(err error) error {
	a.leaseMu.Lock()
	a.ready = false
	a.readyErr = err
	a.leaseMu.Unlock()
	return err
}

func (a *Adapter) setReady() {
	a.leaseMu.Lock()
	a.ready = true
	a.readyErr = nil
	a.leaseMu.Unlock()
}

func (a *Adapter) ensureLease(ctx context.Context, local Instance) error {
	a.leaseMu.Lock()
	if existing := a.leases[local.ID]; existing != nil {
		a.leaseMu.Unlock()
		return nil
	}
	a.leaseMu.Unlock()
	if local.Capacity == nil || a.Admission == nil {
		return fmt.Errorf("local-QEMU instance %s has no usable persisted capacity receipt", local.ID)
	}
	lease, err := a.Admission.Reattach(ctx, *local.Capacity)
	if err != nil {
		return err
	}
	if lease == nil || lease.Disk == nil || lease.Resources == nil {
		return errors.New("reattached local-QEMU capacity lease is incomplete")
	}
	a.keepLease(local.ID, local, lease)
	return nil
}

func (a *Adapter) keepLease(id string, local Instance, lease *QEMULease) {
	if lease == nil {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	item := &adapterLease{capacity: lease, instance: local, cancel: cancel}
	a.leaseMu.Lock()
	if a.leases == nil {
		a.leases = make(map[string]*adapterLease)
	}
	if previous := a.leases[id]; previous != nil {
		previous.cancel()
	}
	a.leases[id] = item
	a.leaseMu.Unlock()
	go a.heartbeatLease(ctx, id, item)
}

func (a *Adapter) heartbeatLease(ctx context.Context, id string, item *adapterLease) {
	ticker := time.NewTicker(capacityHeartbeatInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			heartbeatCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			err := item.capacity.Heartbeat(heartbeatCtx)
			cancel()
			if err != nil {
				cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
				_ = a.stopAndClean(cleanupCtx, item.instance, item.capacity)
				cleanupCancel()
				a.leaseMu.Lock()
				if a.leases[id] == item {
					delete(a.leases, id)
					a.ready = false
					a.readyErr = fmt.Errorf("QEMU instance %s lost host capacity admission: %w", id, err)
				}
				a.leaseMu.Unlock()
				return
			}
		}
	}
}

func (a *Adapter) releaseLease(id string, lease *QEMULease) error {
	a.leaseMu.Lock()
	item := a.leases[id]
	if item != nil {
		item.cancel()
		delete(a.leases, id)
	}
	a.leaseMu.Unlock()
	if lease == nil && item != nil {
		lease = item.capacity
	}
	if lease == nil {
		return nil
	}
	return lease.Release(context.Background())
}

func (a *Adapter) forgetLease(id string) {
	a.leaseMu.Lock()
	if item := a.leases[id]; item != nil {
		item.cancel()
		delete(a.leases, id)
	}
	a.leaseMu.Unlock()
}

func (a *Adapter) stopAndClean(ctx context.Context, local Instance, lease *QEMULease) error {
	if local.PID > 0 {
		alive := true
		// A supplied process controller is an explicit test seam. Production
		// uses the OS process path, which must prove the PID still names this
		// exact guest before signaling it.
		if a.QEMU.StopProcess == nil {
			var verifyErr error
			alive, verifyErr = verifyQEMUProcess(local.PID, local.ID)
			if verifyErr != nil {
				return verifyErr
			}
		}
		if alive {
			if err := a.QEMU.Destroy(ctx, local); err != nil {
				return fmt.Errorf("stop and reap QEMU instance %s: %w", local.ID, err)
			}
		}
	}
	if err := cleanupOwnedWorkdir(local.Workdir); err != nil {
		return fmt.Errorf("clean local-QEMU workdir %s: %w", local.Workdir, err)
	}
	if lease != nil {
		if err := lease.Release(ctx); err != nil {
			return fmt.Errorf("release capacity after QEMU cleanup %s: %w", local.ID, err)
		}
		if lease.Disk == nil && local.Capacity != nil && a.Admission != nil && local.Capacity.DiskLeaseID != "" {
			if err := a.Admission.Reconcile(ctx, *local.Capacity); err != nil {
				return fmt.Errorf("reconcile disk capacity after QEMU cleanup %s: %w", local.ID, err)
			}
		}
	} else if local.Capacity != nil && a.Admission != nil && local.Capacity.DiskLeaseID != "" {
		if err := a.Admission.Reconcile(ctx, *local.Capacity); err != nil {
			return fmt.Errorf("reconcile disk capacity after QEMU cleanup %s: %w", local.ID, err)
		}
	}
	a.recordWrite(ctx)
	return nil
}
