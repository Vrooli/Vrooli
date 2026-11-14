package database

import (
	"context"
	"encoding/json"
	"fmt"

	"agent-manager/internal/domain"
	"agent-manager/internal/runmodel"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// NewRepositoriesWithTerminalReceipts is an explicit, inactive composition
// option. Production calls NewRepositories, so no receipts are captured there.
// This does not install a publisher, API, grant, credential or Work mapping.
// Do not mutate a repository's scope after construction.
func NewRepositoriesWithTerminalReceipts(db *DB, log *logrus.Logger, scope runmodel.TerminalReceiptScope) (*Repositories, error) {
	if err := scope.Validate(); err != nil {
		return nil, err
	}
	repos := NewRepositories(db, log)
	repos.Runs.(*runRepository).terminalReceiptScope = &scope
	return repos, nil
}

// captureTerminalReceiptTx joins the owning transaction, never a callback after
// commit. Failed capture rolls back both run and task changes. Imported rows and
// historical terminal mutations are intentionally not prospective evidence.
func (r *runRepository) captureTerminalReceiptTx(ctx context.Context, tx *Tx, before *domain.Run) error {
	if r.terminalReceiptScope == nil || before == nil || before.Status.IsTerminal() || !before.Status.IsValid() || isImportedTerminalSource(before) {
		return nil
	}
	after, err := r.getTx(ctx, tx, before.ID)
	if err != nil {
		return err
	}
	if after == nil {
		return fmt.Errorf("terminal receipt run disappeared inside transaction")
	}
	if !after.Status.IsTerminal() || isImportedTerminalSource(after) {
		return nil
	}
	if after.LifecycleVersion != before.LifecycleVersion+1 {
		return fmt.Errorf("terminal receipt lifecycle fence did not advance once")
	}
	scope := *r.terminalReceiptScope
	if err := scope.Validate(); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO run_terminal_receipt_source (singleton,source_instance,source_epoch) VALUES (1,?,?) ON CONFLICT (singleton) DO NOTHING`, scope.SourceInstance, scope.SourceEpoch); err != nil {
		return err
	}
	var instance, epoch string
	if err := tx.QueryRowContext(ctx, `SELECT source_instance,source_epoch FROM run_terminal_receipt_source WHERE singleton=1`).Scan(&instance, &epoch); err != nil {
		return err
	}
	if instance != scope.SourceInstance || epoch != scope.SourceEpoch {
		return fmt.Errorf("terminal receipt source binding mismatch")
	}
	receipt, err := runmodel.NewTerminalReceipt(scope, after)
	if err != nil {
		return err
	}
	// A duplicate transition cannot pass the owner CAS. An existing different
	// receipt is therefore a conflict, not a reason to ignore an insert failure.
	_, err = tx.ExecContext(ctx, `INSERT INTO run_terminal_receipts (run_id,lifecycle_version,source_instance,source_epoch,observation_json,observation_sha256,event_intent_json,event_intent_id) VALUES (?,?,?,?,?,?,?,?)`, after.ID, after.LifecycleVersion, scope.SourceInstance, scope.SourceEpoch, receipt.ObservationJSON, receipt.ObservationSHA256, receipt.EventIntentJSON, receipt.EventIntentID)
	return err
}

func isImportedTerminalSource(run *domain.Run) bool {
	return run.ExecutionMode.Normalized() == domain.ExecutionModeImported || run.ImportedAt != nil || run.ImportSourceHarness != "" || run.ImportSourceSessionID != ""
}

// TerminalReceipt returns the original bytes, never a current-row recreation.
// This is an internal persistence seam, not an authorized external reader. A
// future handler must enforce current source-bound read authority and freshness.
// Missing/disabled receipts remain errors; they cannot qualify old runs.
func (r *runRepository) TerminalReceipt(ctx context.Context, runID uuid.UUID, lifecycleVersion int64) (runmodel.TerminalReceipt, error) {
	if r.terminalReceiptScope == nil {
		return runmodel.TerminalReceipt{}, fmt.Errorf("terminal receipt capture is disabled")
	}
	if err := r.terminalReceiptScope.Validate(); err != nil {
		return runmodel.TerminalReceipt{}, err
	}
	if runID == uuid.Nil || lifecycleVersion < 1 {
		return runmodel.TerminalReceipt{}, fmt.Errorf("exact run and terminal lifecycle required")
	}
	var result runmodel.TerminalReceipt
	var instance, epoch string
	err := r.db.QueryRowContext(ctx, `SELECT source_instance,source_epoch,observation_json,observation_sha256,event_intent_json,event_intent_id FROM run_terminal_receipts WHERE run_id=? AND lifecycle_version=?`, runID, lifecycleVersion).Scan(&instance, &epoch, &result.ObservationJSON, &result.ObservationSHA256, &result.EventIntentJSON, &result.EventIntentID)
	if err != nil {
		return runmodel.TerminalReceipt{}, err
	}
	if instance != r.terminalReceiptScope.SourceInstance || epoch != r.terminalReceiptScope.SourceEpoch {
		return runmodel.TerminalReceipt{}, fmt.Errorf("terminal receipt source binding mismatch")
	}
	if runmodel.SHA256([]byte(result.ObservationJSON)) != result.ObservationSHA256 || runmodel.SHA256([]byte(result.EventIntentJSON)) != result.EventIntentID {
		return runmodel.TerminalReceipt{}, fmt.Errorf("terminal receipt integrity mismatch")
	}
	var observation runmodel.TerminalObservation
	if err := json.Unmarshal([]byte(result.ObservationJSON), &observation); err != nil {
		return runmodel.TerminalReceipt{}, err
	}
	if observation.TerminalReceiptScope != *r.terminalReceiptScope || observation.RunID != runID.String() || observation.LifecycleVersion != lifecycleVersion {
		return runmodel.TerminalReceipt{}, fmt.Errorf("terminal receipt identity mismatch")
	}
	result.PublicationState = "not_dispatched"
	return result, nil
}
