package recovery

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/vrooli/internal/coreactivation"
)

// BindNativeLease durably links the launcher-created lease before its child
// starts. The actual native cgroup, rather than a proposed sibling scope, owns
// this session. Registry unavailability may degrade observation, not custody.
func (b *CodingAgentBroker) BindNativeLease(ctx context.Context, id, lease string, registered bool) error {
	if ctx == nil {
		return fmt.Errorf("native recovery context is required")
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	unlock, err := b.lockLedger(ctx)
	if err != nil {
		return err
	}
	defer unlock()
	if err := b.loadLocked(); err != nil {
		return err
	}
	e, err := b.readExecution(id)
	if err != nil {
		return err
	}
	if e.Fence != "claimed" || e.Ref == nil || !b.now().Before(e.Deadline) || e.LeaseID != "" && e.LeaseID != lease {
		return fmt.Errorf("native recovery lease fence refused")
	}
	if !b.nativeRunningAdmission(e) {
		return fmt.Errorf("native recovery lease admission changed")
	}
	if lease == "" || len(lease) > 256 {
		return fmt.Errorf("invalid native recovery lease")
	}
	if err := verifyNativeCodingSelf(ctx, e, *e.Ref); err != nil {
		return err
	}
	e.LeaseID = lease
	e.LeaseRegistered = registered
	return b.writeExecution(e)
}

func (b *CodingAgentBroker) GuardNativeChild(ctx context.Context, id string) error {
	if ctx == nil {
		return fmt.Errorf("native recovery context is required")
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	unlock, err := b.lockLedger(ctx)
	if err != nil {
		return err
	}
	defer unlock()
	if err := b.loadLocked(); err != nil {
		return err
	}
	e, err := b.readExecution(id)
	if err != nil {
		return err
	}
	if e.Fence != "claimed" || e.Ref == nil || e.LeaseID == "" || !b.now().Before(e.Deadline) {
		return fmt.Errorf("native recovery child fence refused")
	}
	if !b.nativeRunningAdmission(e) {
		return fmt.Errorf("native recovery child admission changed")
	}
	return verifyNativeCodingSelf(ctx, e, *e.Ref)
}

func (b *CodingAgentBroker) nativeRunningAdmission(e CodingAgentExecution) bool {
	for _, r := range b.records {
		if r.Outcome == "running" && nativeCodingAdmissionMatches(r, e) {
			return true
		}
	}
	return false
}

// RunNativeExecution retains the deadline admitted by the original caller.
// It does not create another native job or extend time after caller loss.
func (b *CodingAgentBroker) RunNativeExecution(ctx context.Context, e CodingAgentExecution, run CodingAgentRunner) error {
	if ctx == nil || run == nil {
		return fmt.Errorf("native recovery runner is required")
	}
	ctx, cancel := context.WithDeadline(ctx, e.Deadline)
	defer cancel()
	b.run = run
	b.executeContext(ctx, e.ID, e.Request)
	return nil
}

// NativeExecutions reads only sidecars bound to existing ledger receipts.
func (b *CodingAgentBroker) NativeExecutions() ([]CodingAgentExecution, error) {
	records, err := b.Snapshot()
	if err != nil {
		return nil, err
	}
	out := []CodingAgentExecution{}
	for _, r := range records {
		if _, err := uuid.Parse(r.ID); err != nil {
			continue // Historical receipt identities cannot name native inputs.
		}
		e, err := b.readExecution(r.ID)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, nil
}

type CodingAgentOwnership struct {
	Record        CodingAgentRecord
	NativePending bool
}

// NativeOwnershipSnapshot projects only executions bound to their admission
// through the canonical owner evaluator. Historical receipts without a native
// sidecar still suppress duplicate coding cost; they do not claim a live writer.
// Receipt completion alone is not native exit proof.
func (b *CodingAgentBroker) NativeOwnershipSnapshot(ctx context.Context) ([]CodingAgentOwnership, error) {
	if ctx == nil {
		return nil, fmt.Errorf("native recovery ownership context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	// Observe the authoritative files under the existing cross-process owner
	// lock. Avoid the legacy process-cache mutex and its background lock wait;
	// an optional observer must honor its monitoring caller's cancellation.
	unlock, err := b.lockLedger(ctx)
	if err != nil {
		return nil, err
	}
	defer unlock()
	records, err := readCodingAgentRecords(b.path)
	if err != nil {
		return nil, err
	}
	out := make([]CodingAgentOwnership, 0)
	for _, record := range records {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if _, err := uuid.Parse(record.ID); err != nil {
			continue
		}
		execution, err := b.readExecution(record.ID)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		if !nativeCodingAdmissionMatches(record, execution) {
			return nil, fmt.Errorf("native recovery ownership does not match its admission")
		}
		out = append(out, CodingAgentOwnership{Record: record, NativePending: !execution.ExitProven})
	}
	return out, nil
}

// RevokeNativeExecution closes only this admitted execution's effect fence.
// It does not signal a process or turn cancellation into proof of exit;
// ReconcileNative must still exclude the exact tree before closing the receipt.
func (b *CodingAgentBroker) RevokeNativeExecution(ctx context.Context, id string) error {
	if ctx == nil {
		return fmt.Errorf("native recovery context is required")
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	unlock, err := b.lockLedger(ctx)
	if err != nil {
		return err
	}
	defer unlock()
	if err := b.loadLocked(); err != nil {
		return err
	}
	found := false
	for _, r := range b.records {
		if r.ID == id {
			found = true
		}
	}
	if !found {
		return fmt.Errorf("native recovery admission is missing")
	}
	e, err := b.readExecution(id)
	if err != nil {
		return err
	}
	e.Fence = "revoked"
	return b.writeExecution(e)
}

// RunNativeReconciliation performs bounded control-plane reconciliation while
// its API host is alive. Losing this observer never removes native limits or
// transfers job custody back to the API process.
func (b *CodingAgentBroker) RunNativeReconciliation(ctx context.Context, report func(error)) {
	if ctx == nil {
		return
	}
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		if ctx.Err() != nil {
			return
		}
		operation, cancel := context.WithTimeout(ctx, 10*time.Second)
		_, err := b.ReconcileNative(operation, 32)
		cancel()
		if err != nil && ctx.Err() == nil && report != nil {
			report(err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (b *CodingAgentBroker) retireNativeWorker(ctx context.Context, e CodingAgentExecution) error {
	if e.Fence != "revoked" || !e.ExitProven {
		return fmt.Errorf("native recovery inputs still in use")
	}
	p, err := b.workerPath(e.ID)
	if err != nil {
		return err
	}
	entries, err := boundedCodingInputEntries(filepath.Dir(p), 64)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil || !info.Mode().IsRegular() || info.Size() > 512<<20 {
			return fmt.Errorf("native recovery retirement contains nonregular or oversized input")
		}
		if entry.Name() != "worker" && entry.Name() != "execution.json" && !strings.HasPrefix(entry.Name(), ".vrooli-owned-") {
			return fmt.Errorf("native recovery retirement contains unexpected input; preserving it")
		}
	}
	sha, err := coreactivation.ImageDigest(ctx, p)
	if !errors.Is(err, os.ErrNotExist) {
		if err != nil || sha != e.WorkerSHA {
			return fmt.Errorf("native recovery retained image changed; preserving it")
		}
		if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".vrooli-owned-") {
			if err := os.Remove(filepath.Join(filepath.Dir(p), entry.Name())); err != nil && !errors.Is(err, os.ErrNotExist) {
				return err
			}
		}
	}
	dir, err := os.Open(filepath.Dir(p))
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}
