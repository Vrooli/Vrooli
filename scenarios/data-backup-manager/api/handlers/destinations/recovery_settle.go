package destinations

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"data-backup-manager/internal/destinationreadiness"

	"github.com/vrooli/api-core/operatordecision"
)

// Settle reconciles Notification Hub with the recovery journals, and is safe to
// run on every sweep. A repair question is wanted only while its journal is
// still paused at exactly that repair step and the drive is still not mounted
// writable. Any other pending question is withdrawn with the reason in plain
// words: the drive is back, the recovery moved on or finished, the record is
// gone, or (orphaned, keyed by journal id) its destination was removed. A
// question published before subject keys existed is closed by its event id.
func (f *RecoveryDecisionFlow) Settle(ctx context.Context, orphaned map[string]string) error {
	if f == nil || f.asks == nil || f.recoveries == nil || f.readiness == nil || f.publisher == nil {
		return nil
	}
	asks, err := f.asks.ListAsks(ctx)
	if err != nil {
		return fmt.Errorf("list repair decisions: %w", err)
	}
	var errs []error
	for _, ask := range asks {
		if ask.State != operatordecision.PendingStatePending {
			continue
		}
		reason, wanted, err := f.repairStillWanted(ctx, ask, orphaned)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if wanted {
			if err := f.ledger.finishLegacy(ctx, ask); err != nil {
				errs = append(errs, err)
			}
			continue
		}
		if err := f.ledger.withdraw(ctx, ask, reason, true); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (f *RecoveryDecisionFlow) repairStillWanted(ctx context.Context, ask *operatordecision.PendingAsk, orphaned map[string]string) (reason string, wanted bool, err error) {
	recoveryID := ask.Extra["recovery_id"]
	if why, gone := orphaned[recoveryID]; gone {
		return why, false, nil
	}
	journal, err := f.recoveries.Load(ctx, recoveryID)
	if errors.Is(err, destinationreadiness.ErrRecoveryNotFound) {
		return "This repair question no longer matches a recovery on record, so it was closed.", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("load recovery %s: %w", recoveryID, err)
	}
	if journal.State == destinationreadiness.RecoveryCompleted {
		return "The backup drive recovery finished, so no repair approval is needed any more.", false, nil
	}
	index, convErr := strconv.Atoi(ask.Extra["step_index"])
	if convErr != nil || index < 0 || index >= len(journal.Steps) || journal.Current != index ||
		journal.Steps[index].Plan.ID != ask.Extra["plan_id"] || journal.Steps[index].Status != "blocked" {
		return "The drive recovery moved on, so this repair is no longer waiting for approval.", false, nil
	}
	inspection, inspectErr := f.readiness.InspectDevice(ctx, journal.Identity)
	if inspectErr != nil {
		return "", false, fmt.Errorf("inspect drive for recovery %s: %w", recoveryID, inspectErr)
	}
	if inspection.Mounted && !inspection.ReadOnly && !inspection.DeviceWriteProtected {
		return "The backup drive is mounted and writable again, so no repair is needed.", false, nil
	}
	return "", true, nil
}

// withdrawForJournal closes every pending repair question of one recovery. It
// is the immediate path for a recovery that was completed in DBM; Settle is the
// periodic backstop.
func (f *RecoveryDecisionFlow) withdrawForJournal(ctx context.Context, journalID, reason string) {
	if f == nil || f.asks == nil || journalID == "" {
		return
	}
	asks, err := f.asks.ListAsks(ctx)
	if err != nil {
		f.logger.Printf("repair decisions for recovery %s could not be listed: %v", journalID, err)
		return
	}
	for _, ask := range asks {
		if ask.State != operatordecision.PendingStatePending || ask.Extra["recovery_id"] != journalID {
			continue
		}
		if err := f.ledger.withdraw(ctx, ask, reason, true); err != nil {
			f.logger.Printf("repair decision for recovery %s could not be withdrawn: %v", journalID, err)
		}
	}
}

// answerToWithdrawn handles an answer to a repair question DBM had already
// withdrawn: no repair is ever run for it, and the operator is told once.
func (f *RecoveryDecisionFlow) answerToWithdrawn(ctx context.Context, ask *operatordecision.PendingAsk, res operatordecision.DecisionResolution) error {
	if res.State != operatordecision.StateAnswered {
		return nil
	}
	f.logger.Printf("volume recovery answer to a withdrawn question ignored recovery=%s", ask.Extra["recovery_id"])
	return f.ledger.lateAnswer(ctx, ask, recoveryNotification(
		"Backup drive repair question was already closed",
		"Your answer arrived after the repair question was closed because the drive's situation changed, so no repair was run. If the drive still needs one I will ask again.",
		"volume-recovery-failed", "normal", ask.Subject, ask.ID))
}
