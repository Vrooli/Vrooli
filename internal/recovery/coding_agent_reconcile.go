package recovery

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	platform "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/coreactivation"
)

type CodingAgentReconciliation struct {
	Changed          []CodingAgentRecord `json:"changed"`
	Remaining        int                 `json:"remaining"`
	ExecutionEffects bool                `json:"execution_effects"`
	ExitProven       bool                `json:"exit_proven"`
}

// ReconcileUnverified changes stale execution claims into explicit uncertainty.
// Historical records did not retain a native identity. Neither age nor a
// deadline can supply exit proof; pending suppression remains in force and
// this operation never signals, relaunches, or releases an investigation.
func (b *CodingAgentBroker) ReconcileUnverified(ctx context.Context, limit int) (CodingAgentReconciliation, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	report := CodingAgentReconciliation{Changed: []CodingAgentRecord{}}
	if limit < 1 || limit > 128 {
		return report, fmt.Errorf("reconciliation limit must be between 1 and 128")
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	unlock, err := b.lockLedger(ctx)
	if err != nil {
		return report, err
	}
	defer unlock()
	if err := b.loadLocked(); err != nil {
		return report, err
	}
	if err := ctx.Err(); err != nil {
		return report, err
	}
	now := b.now().UTC()
	before := append([]CodingAgentRecord(nil), b.records...)
	for i := range b.records {
		r := &b.records[i]
		if r.Outcome != "running" && r.Outcome != "scheduled" {
			continue
		}
		if !r.DeadlineAt.IsZero() && now.Before(r.DeadlineAt) {
			continue
		}
		if len(report.Changed) == limit {
			report.Remaining++
			continue
		}
		r.Outcome = "unconfirmed"
		r.ReconciledAt = &now
		// FinishedAt intentionally stays unchanged: no stop/completion occurred.
		r.ReconciliationReason = "execution unverified: no retained native identity; elapsed time is not proof of exit"
		report.Changed = append(report.Changed, *r)
	}
	if len(report.Changed) > 0 {
		if err := b.persistLocked(); err != nil {
			b.records = before
			return CodingAgentReconciliation{}, fmt.Errorf("reconciliation acknowledgement unconfirmed; inspect the durable ledger before retry: %w", err)
		}
	}
	return report, nil
}

// ReconcileNative may make the first submission for an undispatched admission.
// It never replays a consumed/ambiguous dispatch. A durable revoke and native
// exclusion precede completion or input retirement, including unadmitted inputs.
func (b *CodingAgentBroker) ReconcileNative(ctx context.Context, limit int) (int, error) {
	if ctx == nil || limit < 1 || limit > 128 {
		return 0, fmt.Errorf("native reconciliation requires a context and limit 1–128")
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	unlock, err := b.lockLedger(ctx)
	if err != nil {
		return 0, err
	}
	defer unlock()
	if err := b.loadLocked(); err != nil {
		return 0, err
	}
	n := b.native
	if n == nil {
		n = nativeCodingOwner("")
	}
	count := 0
	for i := range b.records {
		if err := ctx.Err(); err != nil {
			return count, err
		}
		r := &b.records[i]
		if _, err := uuid.Parse(r.ID); err != nil {
			continue
		}
		e, err := b.readExecution(r.ID)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return count, err
		}
		if e.Retired {
			continue
		}
		if count == limit {
			return count, nil
		}
		count++
		live, err := b.observeNativeExecution(ctx, n, &e, *r)
		if err != nil {
			return count, err
		}
		if live {
			continue
		}
		if err := b.excludeNativeExecution(ctx, n, &e); err != nil {
			return count, err
		}
		if r.Outcome == "scheduled" || r.Outcome == "running" || r.Outcome == "unconfirmed" {
			r.Outcome = "failed"
			r.FinishedAt = b.now().UTC()
			r.Error = "native recovery tree excluded; repair success was not recorded"
			if err := b.persistLocked(); err != nil {
				return count, err
			}
		}
		if err := b.finishNativeRetirement(ctx, n, &e); err != nil {
			return count, err
		}
	}
	if count == limit {
		return count, nil
	}
	orphans, err := b.reconcileNativePreparations(ctx, n, limit-count)
	return count + orphans, err
}

func (b *CodingAgentBroker) observeNativeExecution(ctx context.Context, n *codingAgentNativeOwner, e *CodingAgentExecution, r CodingAgentRecord) (bool, error) {
	if e.ExitProven {
		return false, nil
	}
	ref, err := n.lookup(ctx, e.Intent)
	if err == nil {
		if e.Ref != nil {
			ref = *e.Ref
		}
		ob, err := n.inspect(ctx, ref)
		if err != nil {
			return false, err
		}
		before := b.now().Before(e.Deadline)
		if before && e.Fence != "revoked" && ob.State == platform.OperationRunning && ob.RootAlive {
			return true, nil
		}
		if before && e.Fence == "ready" && ref.RootPID == 0 && ob.State == platform.OperationOutcomeUnknown {
			return true, nil
		}
		e.Ref = &ref
		return false, nil
	}
	if !nativeLookupMayBeAbsent(err) {
		return false, err
	}
	if e.Fence != "ready" || !b.now().Before(e.Deadline) {
		return false, nil
	}
	if e.DispatchCommitted {
		return true, nil
	} // Preserve an uncertain in-flight start until its deadline.
	if b.native == nil {
		return false, fmt.Errorf("native recovery dispatch owner is unavailable")
	}
	if e.Ref != nil || r.Outcome != "scheduled" || !nativeCodingAdmissionMatches(r, *e) {
		return false, fmt.Errorf("native recovery admission and prepared dispatch conflict")
	}
	absent, err := n.absent(ctx, e.Intent)
	if err != nil {
		return false, err
	}
	if !absent {
		return true, nil
	}
	worker, err := b.workerPath(e.ID)
	if err != nil {
		return false, err
	}
	sha, err := coreactivation.ImageDigest(ctx, worker)
	if err != nil || sha != e.WorkerSHA {
		return false, fmt.Errorf("native recovery prepared image changed; preserving admission")
	}
	spec, err := b.nativeSpec(*e)
	if err != nil || e.Intent.ValidateSpec(spec) != nil {
		return false, fmt.Errorf("native recovery prepared specification changed")
	}
	if err := b.dispatchNative(ctx, e); err != nil {
		return false, err
	}
	return true, nil
}

func nativeLookupMayBeAbsent(err error) bool {
	return errors.Is(err, platform.ErrOperationNotFound) || errors.Is(err, platform.ErrOperationOutcomeLost)
}

func nativeCodingAdmissionMatches(r CodingAgentRecord, e CodingAgentExecution) bool {
	return r.ID == e.ID && r.Scenario == e.Request.Scenario && r.Requester == e.Request.Requester && r.Reason == e.Request.Reason && r.DeadlineAt.Equal(e.Deadline) && r.CreatedAt.Before(e.Deadline) && !e.Deadline.After(r.CreatedAt.Add(CodingAgentRecoveryTimeout))
}

func (b *CodingAgentBroker) excludeNativeExecution(ctx context.Context, n *codingAgentNativeOwner, e *CodingAgentExecution) error {
	if e.ExitProven {
		return nil
	}
	e.Fence = "revoked"
	if err := b.writeExecution(*e); err != nil {
		return err
	}
	ref, err := n.lookup(ctx, e.Intent)
	if err == nil {
		if e.Ref != nil {
			ref = *e.Ref
		}
		e.Ref = &ref
		if err := b.writeExecution(*e); err != nil {
			return err
		}
		stopped, err := n.stop(ctx, ref)
		if err != nil || !stopped.NativeQuiescent {
			return fmt.Errorf("native recovery exclusion unconfirmed")
		}
	} else {
		if !nativeLookupMayBeAbsent(err) {
			return err
		}
		if err := nativeCodingAbsence(ctx, n, *e); err != nil {
			return err
		}
	}
	e.ExitProven = true
	return b.writeExecution(*e)
}

func nativeCodingAbsence(ctx context.Context, n *codingAgentNativeOwner, e CodingAgentExecution) error {
	absent, err := n.absent(ctx, e.Intent)
	if err != nil || !absent {
		return fmt.Errorf("native recovery absence unconfirmed")
	}
	if e.Ref != nil && e.Ref.RootPID > 0 {
		released, err := n.released(ctx, *e.Ref, e.BootID+":"+e.Ref.RootStartTime)
		if err != nil || !released {
			return fmt.Errorf("native recovery captured root exit unconfirmed")
		}
	}
	return nil
}

func (b *CodingAgentBroker) finishNativeRetirement(ctx context.Context, n *codingAgentNativeOwner, e *CodingAgentExecution) error {
	if !e.ExitProven || e.Fence != "revoked" {
		return fmt.Errorf("native recovery retirement lacks exclusion proof")
	}
	if e.Ref != nil {
		if err := n.release(ctx, *e.Ref); err != nil {
			if absenceErr := nativeCodingAbsence(ctx, n, *e); absenceErr != nil {
				return err
			}
		}
	}
	if err := b.retireNativeWorker(ctx, *e); err != nil {
		return err
	}
	e.Retired = true
	return b.writeExecution(*e)
}

func boundedCodingInputEntries(dir string, maximum int) ([]os.DirEntry, error) {
	f, err := os.Open(dir)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	entries, err := f.ReadDir(maximum + 1)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	if len(entries) > maximum {
		return nil, fmt.Errorf("native recovery input inventory exceeds its bound")
	}
	return entries, nil
}

func (b *CodingAgentBroker) reconcileNativePreparations(ctx context.Context, n *codingAgentNativeOwner, limit int) (int, error) {
	parent := filepath.Join(filepath.Dir(b.path), "coding-agent-executions")
	entries, err := boundedCodingInputEntries(parent, codingAgentMaxRecords)
	if errors.Is(err, os.ErrNotExist) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	admitted := make(map[string]bool, len(b.records))
	for _, r := range b.records {
		admitted[r.ID] = true
	}
	count := 0
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return count, err
		}
		id, err := uuid.Parse(entry.Name())
		if err != nil || id.String() != entry.Name() || id == uuid.Nil || !entry.IsDir() {
			return count, fmt.Errorf("native recovery input inventory contains unexpected content")
		}
		if admitted[entry.Name()] {
			continue
		}
		e, err := b.readExecution(entry.Name())
		if err == nil && e.Retired {
			continue
		}
		if count == limit {
			return count, nil
		}
		count++
		if errors.Is(err, os.ErrNotExist) {
			if err := removeEmptyCodingPreparation(filepath.Join(parent, entry.Name())); err != nil {
				return count, err
			}
			continue
		}
		if err != nil {
			return count, err
		}
		// Ledger lock excludes a still-preparing API writer. No missing receipt
		// may authorize dispatch; revoke and exclude even a retained claimed job.
		if err := b.excludeNativeExecution(ctx, n, &e); err != nil {
			return count, err
		}
		if err := b.finishNativeRetirement(ctx, n, &e); err != nil {
			return count, err
		}
	}
	// Confirm prior removals even after a lost directory-sync acknowledgement.
	directory, err := os.Open(parent)
	if err != nil {
		return count, err
	}
	err = directory.Sync()
	closeErr := directory.Close()
	return count, errors.Join(err, closeErr)
}

func removeEmptyCodingPreparation(dir string) error {
	entries, err := boundedCodingInputEntries(dir, 64)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil || !strings.HasPrefix(entry.Name(), ".vrooli-owned-") || !info.Mode().IsRegular() || info.Size() > 512<<20 {
			return fmt.Errorf("native recovery preparation lacks identity; preserving unexpected inputs")
		}
	}
	for _, entry := range entries {
		if err := os.Remove(filepath.Join(dir, entry.Name())); err != nil {
			return err
		}
	}
	if err := os.Remove(dir); err != nil {
		return err
	}
	parent, err := os.Open(filepath.Dir(dir))
	if err != nil {
		return err
	}
	defer parent.Close()
	return parent.Sync()
}
