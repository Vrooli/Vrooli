package destinations

import (
	"context"
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"data-backup-manager/internal/destinationreadiness"
	"data-backup-manager/internal/destinations"

	"github.com/vrooli/api-core/lifecycle"
	"github.com/vrooli/api-core/operatordecision"
)

const volumeRecoverySweepInterval = time.Minute

// VolumeRecoveryWorker observes registered filesystem destinations and
// automatically remounts only after the control-plane check positively reports
// a consistent filesystem. Dirty, unknown, failed-check, or unsupported cases
// are surfaced and routed to the exact repair decision flow when possible.
type VolumeRecoveryWorker struct {
	destinations destinations.Service
	readiness    *destinationreadiness.Service
	recoveries   *destinationreadiness.FileRecoveryStore
	decisions    *RecoveryDecisionFlow
	logger       *log.Logger
	interval     time.Duration

	mu     sync.Mutex
	cancel context.CancelFunc
	done   chan struct{}
}

func newVolumeRecoveryWorker(dests destinations.Service, readiness *destinationreadiness.Service, recoveries *destinationreadiness.FileRecoveryStore, decisions *RecoveryDecisionFlow, logger *log.Logger) *VolumeRecoveryWorker {
	if logger == nil {
		logger = log.Default()
	}
	return &VolumeRecoveryWorker{
		destinations: dests, readiness: readiness, recoveries: recoveries,
		decisions: decisions, logger: logger, interval: volumeRecoverySweepInterval,
	}
}

// LifecycleHooks lets the shared control plane stop and resume the monitor
// around maintenance. No scanner or host action runs while the provider is
// quiesced.
func (w *VolumeRecoveryWorker) LifecycleHooks() lifecycle.Hooks {
	return lifecycle.Hooks{Quiesce: w.Stop, Resume: func(context.Context) error { return w.Start() }}
}

func (w *VolumeRecoveryWorker) Start() error {
	if w == nil {
		return nil
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.cancel != nil {
		return nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	w.cancel, w.done = cancel, done
	go func() {
		defer close(done)
		ticker := time.NewTicker(w.interval)
		defer ticker.Stop()
		w.reconcile(ctx)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				w.reconcile(ctx)
			}
		}
	}()
	return nil
}

func (w *VolumeRecoveryWorker) Stop(ctx context.Context) error {
	if w == nil {
		return nil
	}
	w.mu.Lock()
	cancel, done := w.cancel, w.done
	w.cancel, w.done = nil, nil
	w.mu.Unlock()
	if cancel == nil {
		return nil
	}
	cancel()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (w *VolumeRecoveryWorker) reconcile(ctx context.Context) {
	if w == nil || w.destinations == nil || w.readiness == nil {
		return
	}
	dests, err := w.destinations.ListDestinations(ctx, 100)
	if err != nil {
		w.logger.Printf("volume recovery scan could not list destinations: %v", err)
		return
	}
	for _, destination := range dests {
		if ctx.Err() != nil {
			return
		}
		if destination.BackendKind != destinations.BackendFilesystem {
			continue
		}
		if destination.DeviceIdentity == nil || !destination.DeviceIdentity.StrongIdentity() {
			// Without a stable identity, an absent path could be a retired test
			// destination, a temporarily unavailable local tree, or a removable
			// volume. Do not guess or emit repeated mount-failure alerts.
			w.logger.Printf("volume recovery skipped destination %s: no persisted stable volume identity", destination.ID)
			continue
		}
		if err := w.reconcileDestination(ctx, destination); err != nil {
			w.logger.Printf("volume recovery for destination %s: %v", destination.ID, err)
		}
	}
}

func (w *VolumeRecoveryWorker) reconcileDestination(ctx context.Context, destination destinations.Destination) error {
	inspection, err := w.readiness.InspectDevice(ctx, *destination.DeviceIdentity)
	if err != nil {
		w.notifyMountFailure(ctx, destination, err)
		return err
	}
	if inspection.Mounted {
		return w.recordMountedRecovery(ctx, destination, inspection)
	}
	root := filepath.Clean(destination.DeviceIdentity.Mountpoint)
	if strings.TrimSpace(destination.DeviceIdentity.Mountpoint) == "" {
		err := fmt.Errorf("registered backup volume has no intended mount root; bind its recovery identity before automatic remount")
		w.notifyMountFailure(ctx, destination, err)
		return err
	}
	plans, err := w.planRecoveryActions(ctx, destination, root)
	if err != nil {
		return err
	}
	journalID, done, err := w.prepareRecoveryEpisode(ctx, destination, inspection, plans)
	if err != nil || done {
		return err
	}
	mounted, checkDetail, err := w.runRecoveryCheckAndMount(ctx, destination, plans, journalID)
	if err != nil || mounted {
		return err
	}
	return w.pauseForFilesystemRepair(ctx, destination, plans, journalID, checkDetail)
}

type recoveryActionPlans struct {
	root      string
	check     destinationreadiness.Plan
	checkErr  error
	repair    destinationreadiness.Plan
	repairErr error
	mount     destinationreadiness.Plan
}

func (w *VolumeRecoveryWorker) recordMountedRecovery(ctx context.Context, destination destinations.Destination, inspection destinationreadiness.Inspection) error {
	if destination.DeviceIdentity.Mountpoint != "" && inspection.Identity.Mountpoint != "" && filepath.Clean(inspection.Identity.Mountpoint) != filepath.Clean(destination.DeviceIdentity.Mountpoint) {
		err := fmt.Errorf("registered backup volume is mounted at %s instead of its recorded recovery root %s", inspection.Identity.Mountpoint, destination.DeviceIdentity.Mountpoint)
		w.notifyMountFailure(ctx, destination, err)
		return err
	}
	if inspection.ReadOnly || inspection.DeviceWriteProtected {
		err := fmt.Errorf("registered backup volume is mounted read-only or write-protected")
		w.notifyMountFailure(ctx, destination, err)
		return err
	}
	if _, err := w.recoveries.ObserveEpisode(ctx, destination.ID, inspection.Identity, true, ""); err != nil {
		return fmt.Errorf("record mounted recovery observation: %w", err)
	}
	return nil
}

func (w *VolumeRecoveryWorker) planRecoveryActions(ctx context.Context, destination destinations.Destination, root string) (recoveryActionPlans, error) {
	identity := *destination.DeviceIdentity
	plans := recoveryActionPlans{root: root}
	plans.check, plans.checkErr = w.readiness.PlanPreparation(ctx, destinationreadiness.PlanInput{
		Location: root, Action: destinationreadiness.ActionCheckFilesystem, ExpectedDevice: identity,
	})
	plans.repair, plans.repairErr = w.readiness.PlanPreparation(ctx, destinationreadiness.PlanInput{
		Location: root, Action: destinationreadiness.ActionRepairFilesystem, ExpectedDevice: identity,
	})
	var err error
	plans.mount, err = w.readiness.PlanPreparation(ctx, destinationreadiness.PlanInput{
		Location: root, Action: destinationreadiness.ActionMountReadWrite, ExpectedDevice: identity,
	})
	if err != nil {
		w.notifyMountFailure(ctx, destination, err)
		return recoveryActionPlans{}, err
	}
	return plans, nil
}

func (w *VolumeRecoveryWorker) prepareRecoveryEpisode(ctx context.Context, destination destinations.Destination, inspection destinationreadiness.Inspection, plans recoveryActionPlans) (string, bool, error) {
	legacyID := operatordecision.CorrelationID("dbm-monitor", destination.ID, plans.repair.ID, plans.mount.ID)
	episode, err := w.recoveries.ObserveEpisode(ctx, destination.ID, inspection.Identity, false, legacyID)
	if err != nil {
		return "", false, fmt.Errorf("record unmounted recovery episode: %w", err)
	}
	journalID := episode.RecoveryID
	existing, err := w.recoveries.Load(ctx, journalID)
	if errors.Is(err, destinationreadiness.ErrRecoveryNotFound) {
		return journalID, false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("load recovery journal: %w", err)
	}
	if existing.State == destinationreadiness.RecoveryCompleted {
		episode, err = w.recoveries.AdvanceUnmountedEpisode(ctx, destination.ID, journalID)
		if err != nil {
			return "", false, fmt.Errorf("advance completed recovery episode: %w", err)
		}
		journalID = episode.RecoveryID
		if _, err := w.recoveries.Load(ctx, journalID); errors.Is(err, destinationreadiness.ErrRecoveryNotFound) {
			return journalID, false, nil
		} else if err != nil {
			return "", false, fmt.Errorf("load new recovery episode journal: %w", err)
		}
		return journalID, false, nil
	}
	return journalID, true, w.reconcilePausedRecovery(ctx, existing)
}

func (w *VolumeRecoveryWorker) reconcilePausedRecovery(ctx context.Context, existing destinationreadiness.RecoveryJournal) error {
	if !pausedAtRepairApproval(existing) {
		return fmt.Errorf("existing recovery journal is not paused at a repair decision; inspect it before retrying")
	}
	if w.decisions == nil {
		return fmt.Errorf("recovery remains paused; Notification Hub decision delivery is unavailable")
	}
	terminal, err := w.decisions.HasTerminalDecision(ctx, existing)
	if err != nil || terminal {
		return err
	}
	if err := w.decisions.RaiseRepairApproval(ctx, existing); err != nil {
		return fmt.Errorf("retry repair decision: %w", err)
	}
	return nil
}

func pausedAtRepairApproval(journal destinationreadiness.RecoveryJournal) bool {
	return journal.Current >= 0 && journal.Current < len(journal.Steps) &&
		journal.Steps[journal.Current].Plan.Action == destinationreadiness.ActionRepairFilesystem &&
		journal.Steps[journal.Current].Status == "blocked"
}

func (w *VolumeRecoveryWorker) runRecoveryCheckAndMount(ctx context.Context, destination destinations.Destination, plans recoveryActionPlans, journalID string) (bool, string, error) {
	var result destinationreadiness.ExecuteResult
	checkErr := plans.checkErr
	if checkErr == nil && plans.check.Supported {
		result, checkErr = w.readiness.ExecutePreparation(ctx, destinationreadiness.ExecuteInput{
			Plan: plans.check, Confirmation: plans.check.ConfirmationPhrase,
		})
	}
	if checkErr == nil && result.Consistent == "yes" {
		return w.mountAfterCleanCheck(ctx, destination, plans.mount, journalID)
	}
	detail := "Read-only filesystem check consistency: " + string(result.Consistent)
	if checkErr != nil {
		detail = "Read-only filesystem check failed: " + checkErr.Error()
	}
	return false, detail, nil
}

func (w *VolumeRecoveryWorker) mountAfterCleanCheck(ctx context.Context, destination destinations.Destination, plan destinationreadiness.Plan, journalID string) (bool, string, error) {
	if !plan.Supported {
		err := fmt.Errorf("filesystem is consistent, but this platform manages volume attachment automatically and no mounted drive letter is visible")
		w.notifyMountFailure(ctx, destination, err)
		return false, "", err
	}
	if _, err := w.readiness.ExecutePreparation(ctx, destinationreadiness.ExecuteInput{Plan: plan, Confirmation: plan.ConfirmationPhrase}); err != nil {
		w.notifyMountFailure(ctx, destination, err)
		return false, "", fmt.Errorf("safe remount failed: %w", err)
	}
	mounted, err := w.readiness.InspectDevice(ctx, *destination.DeviceIdentity)
	if err != nil {
		w.notifyMountFailure(ctx, destination, err)
		return false, "", fmt.Errorf("remount completed but volume state could not be verified: %w", err)
	}
	if !mounted.Mounted || mounted.ReadOnly || mounted.DeviceWriteProtected {
		err := fmt.Errorf("remount completed but the selected volume is not verified mounted read/write")
		w.notifyMountFailure(ctx, destination, err)
		return false, "", err
	}
	if w.decisions != nil {
		w.decisions.NotifyRecoveryCompleted(ctx, destinationreadiness.RecoveryJournal{ID: journalID, Location: destination.Location})
	}
	return true, "", nil
}

func (w *VolumeRecoveryWorker) pauseForFilesystemRepair(ctx context.Context, destination destinations.Destination, plans recoveryActionPlans, journalID, checkDetail string) error {
	w.notifyMountFailure(ctx, destination, fmt.Errorf("%s", checkDetail))
	if plans.repairErr != nil {
		return fmt.Errorf("filesystem is not confirmed clean and repair planning failed: %w", plans.repairErr)
	}
	if !plans.repair.Supported {
		return fmt.Errorf("filesystem is not confirmed clean; repair is unsupported: %s", plans.repair.UnsupportedReason)
	}
	steps := []destinationreadiness.Plan{plans.repair}
	if plans.mount.Supported {
		steps = append(steps, plans.mount)
	} else if runtime.GOOS != "windows" {
		return fmt.Errorf("automatic read-write mount is unsupported: %s", plans.mount.UnsupportedReason)
	}
	journal, err := destinationreadiness.NewRecoveryJournal(plans.root, plans.repair.Identity, steps, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("prepare approval-bound recovery journal: %w", err)
	}
	journal.ID = journalID
	journal.Current = 0
	journal.State = destinationreadiness.RecoveryPaused
	journal.LastError = checkDetail
	journal.Steps[0].Status = "blocked"
	if err := w.recoveries.Save(ctx, journal); err != nil {
		return fmt.Errorf("persist approval-bound recovery journal: %w", err)
	}
	if w.decisions == nil {
		return fmt.Errorf("Notification Hub decision delivery is unavailable; recovery remains paused")
	}
	if err := w.decisions.RaiseRepairApproval(ctx, journal); err != nil {
		return fmt.Errorf("raise repair approval: %w", err)
	}
	return nil
}

func (w *VolumeRecoveryWorker) notifyMountFailure(ctx context.Context, destination destinations.Destination, cause error) {
	if w.decisions == nil {
		return
	}
	w.decisions.NotifyMountFailure(ctx, destinationreadiness.RecoveryJournal{ID: destination.ID, Location: destination.Location}, cause)
}

// HasTerminalDecision reports whether this journal's exact repair ask was
// already answered. The monitor uses this to avoid reopening a declined ask on
// every sweep.
func (f *RecoveryDecisionFlow) HasTerminalDecision(ctx context.Context, journal destinationreadiness.RecoveryJournal) (bool, error) {
	if f == nil || f.asks == nil || journal.Current < 0 || journal.Current >= len(journal.Steps) {
		return false, nil
	}
	step := journal.Steps[journal.Current]
	id := operatordecision.CorrelationID("dbm-repair", journal.ID, step.Plan.ID, fmt.Sprint(journal.Current))
	ask, err := f.asks.Get(ctx, id)
	if err != nil || ask == nil {
		return false, err
	}
	return ask.State != operatordecision.PendingStatePending && ask.State != "", nil
}
