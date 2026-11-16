package destinationreadiness

import (
	"context"
	"fmt"
	"path/filepath"

	"data-backup-manager/internal/sysmounts"
)

// reconcileInterruptedMountStep accepts only mount transitions whose
// postcondition is visible in a fresh observation of the same physical volume.
// It deliberately cannot reconcile filesystem checks or repairs: neither has
// a postcondition this read-only observer can prove.
func (s *Service) reconcileInterruptedMountStep(ctx context.Context, journal RecoveryJournal, step RecoveryStep) (bool, string, error) {
	action := step.Plan.Action
	if action != ActionUnmount && action != ActionMountReadWrite {
		return false, "", nil
	}
	if s == nil || s.deviceInspector == nil {
		return false, "", fmt.Errorf("device inspection is unavailable; refusing to reconcile interrupted %s", action)
	}
	if !journal.Identity.StrongIdentity() || !step.Plan.Identity.StrongIdentity() ||
		!journal.Identity.MatchesDevice(step.Plan.Identity) ||
		filepath.Clean(step.Plan.TargetPath) != filepath.Clean(journal.Location) {
		return false, "", fmt.Errorf("interrupted %s step is not bound to the journal device and mount root", action)
	}
	observed, err := s.deviceInspector.InspectDevice(ctx, journal.Identity)
	if err != nil {
		return false, "", fmt.Errorf("inspect device before reconciling interrupted %s: %w", action, err)
	}
	if !observed.Identity.StrongIdentity() || !journal.Identity.MatchesDevice(observed.Identity) {
		return false, "", fmt.Errorf("device identity is not proven for interrupted %s; refusing reconciliation", action)
	}
	switch action {
	case ActionUnmount:
		if !observed.Mounted {
			return true, "fresh inspection confirmed the journaled device is unmounted", nil
		}
	case ActionMountReadWrite:
		if observed.Mounted && filepath.Clean(observed.Identity.Mountpoint) == filepath.Clean(journal.Location) &&
			!observed.ReadOnly && !observed.DeviceWriteProtected &&
			observed.FilesystemState == sysmounts.FilesystemStateClean {
			return true, "fresh inspection confirmed the journaled device is mounted read/write at its recovery root", nil
		}
	}
	return false, "", nil
}
