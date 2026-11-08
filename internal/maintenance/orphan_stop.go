package maintenance

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"time"

	"github.com/vrooli/vrooli/internal/process"
	"github.com/vrooli/vrooli/internal/tuning"
)

var openOrphanHandleFn = process.OpenHandle

var errOrphanSkipped = errors.New("orphan candidate is absent or no longer eligible; no signal sent")

// stopOrphan pins before fresh qualification and never signals a numeric PID.
// The fresh complete snapshot rechecks current registry ownership and protected
// native trees, rather than inferring abandonment from an executable path.
func (c *Controller) stopOrphan(ctx context.Context, pid int) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	handle, err := openOrphanHandleFn(pid)
	if err != nil {
		if isMissingProcessError(err) {
			return errOrphanSkipped
		}
		return fmt.Errorf("pin orphan process: %w", err)
	}
	defer handle.Close()
	eligible, err := c.qualifyPinnedOrphan(ctx, pid, handle)
	if err != nil {
		return err
	}
	if !eligible {
		return errOrphanSkipped
	}
	if err := handle.Signal(false); err != nil && !isMissingProcessError(err) {
		return err
	}
	grace, cancel := context.WithTimeout(ctx, tuning.MaintenanceSettleDelay())
	err = process.AwaitExit(grace, handle)
	cancel()
	if err == nil {
		return nil
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	eligible, err = c.qualifyPinnedOrphan(ctx, pid, handle)
	if err != nil {
		return err
	}
	if !eligible {
		alive, err := handle.Alive()
		if err != nil {
			return err
		}
		if alive {
			return fmt.Errorf("process acquired an owner or protected scope after graceful stop; force refused")
		}
		return nil
	}
	if err := handle.Signal(true); err != nil && !isMissingProcessError(err) {
		return err
	}
	exit, cancelExit := context.WithTimeout(ctx, 2*time.Second)
	defer cancelExit()
	if err := process.AwaitExit(exit, handle); err != nil {
		return fmt.Errorf("orphan exit is unverified: %w", err)
	}
	return nil
}

func (c *Controller) qualifyPinnedOrphan(ctx context.Context, pid int, handle process.Handle) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	alive, err := handle.Alive()
	if err != nil || !alive {
		return false, err
	}
	if runtime.GOOS == "linux" {
		entry, ok := readProcessEntryFn(pid)
		if !ok {
			alive, err := handle.Alive()
			if err != nil || !alive {
				return false, err
			}
			return false, fmt.Errorf("live orphan process identity became unreadable")
		}
		if !looksLikeVrooliProcessFn(c.Root, c.Home, entry) {
			return false, nil
		}
		if entry.Cgroup == "" {
			return false, fmt.Errorf("native process scope is unreadable; cleanup refused")
		}
		if insideLifecycleOwnedScope(entry) {
			return false, nil
		}
	}
	if !c.stillVrooliOrphan(pid) {
		return false, nil
	}
	fresh, err := c.Snapshot()
	if err != nil {
		return false, fmt.Errorf("recheck current orphan ownership: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return false, err
	}
	for _, candidate := range fresh.Orphans {
		if candidate.PID == pid {
			// If this held incarnation exited during observation, the PID
			// may now name another process; it never inherits the signal.
			return handle.Alive()
		}
	}
	return false, nil
}
