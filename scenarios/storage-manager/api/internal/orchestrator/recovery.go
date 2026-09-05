package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"github.com/vrooli/api-core/eventbus"
	"storage-manager/internal/cleanup"
	"storage-manager/internal/recoverylock"
)

// RecoveryRun is the server-owned lifecycle record returned to callers.
type RecoveryRun struct {
	ID              string
	Status          string
	Trigger         string
	Partition       string
	Action          PressureAction
	PlanID          string
	EstimatedBytes  int64
	ReclaimedBytes  int64
	TargetFreeBytes int64
	StoppedBecause  string
	Reason          string
	StartedAt       time.Time
	CompletedAt     time.Time
	Done            <-chan struct{}
}

type WriterSnapshot struct {
	ID           string    `json:"id"`
	ObservedAt   time.Time `json:"observed_at"`
	Root         string    `json:"root"`
	Mount        string    `json:"mount"`
	Bytes        int64     `json:"bytes"`
	DeltaBytes   int64     `json:"delta_bytes"`
	DeltaHours   float64   `json:"delta_hours"`
	BytesPerHour int64     `json:"bytes_per_hour"`
	Partial      bool      `json:"partial"`
	Hot          bool      `json:"hot"`
}

type recoveryLedger interface {
	SaveRecoveryRun(context.Context, RecoveryRun) error
}

type recoveryHistoryReader interface {
	ListRecoveryRuns(context.Context, int) ([]RecoveryRun, error)
}

// SetRecoveryLockPath configures the cross-process host lock. An empty path
// keeps unit tests hermetic; production wires the api-core-resolved state root.
func (s *Service) SetRecoveryLockPath(path string) {
	s.recoveryGate.Lock()
	s.recoveryLockPath = path
	s.recoveryGate.Unlock()
}

// ReconcileInterruptedRecoveryRuns closes durable runs that were left in
// flight by a process restart. Recovery workers are deliberately not resumed
// from an old process: their provider context and previews are no longer
// trustworthy. A restarted server must expose a terminal record, and WaitRecovery
// must be able to return it instead of leaving clients blocked forever.
func (s *Service) ReconcileInterruptedRecoveryRuns(ctx context.Context) error {
	reader, ok := s.store.(recoveryHistoryReader)
	if !ok {
		return nil
	}
	runs, err := reader.ListRecoveryRuns(ctx, 0)
	if err != nil {
		return err
	}
	for _, run := range runs {
		if run.Status != RecoveryRunning {
			continue
		}
		done := make(chan struct{})
		close(done)
		run.Status = RecoveryFailed
		run.Reason = "interrupted by storage-manager restart"
		run.StoppedBecause = "interrupted_by_restart"
		run.CompletedAt = s.now()
		run.Done = done
		s.persistRecovery(ctx, run)
		s.recoveryMu.Lock()
		s.recoveryRuns[run.ID] = &run
		s.recoveryMu.Unlock()
	}
	return nil
}

const (
	RecoveryRunning  = "running"
	RecoveryComplete = "complete"
	RecoveryFailed   = "failed"
	recoveryBatchCap = int64(2 * 1024 * 1024 * 1024)
	// Owner providers return bounded previews. The owner-side recovery path uses
	// an oldest-first bounded SQL query and exact-ID apply, so 200 items keeps
	// progress practical for large stale indexes without reopening a full scan.
	recoveryBatchItems  = 200
	recoveryR3Budget    = int64(50 * 1024 * 1024 * 1024)
	recoveryMaxDuration = 20 * time.Minute
	// A single provider must not consume the whole recovery window. Owner
	// providers and bounded filesystem census paths already have shorter
	// operation budgets; this is the recovery-level circuit breaker for a
	// provider that ignores or exceeds those bounds.
	recoveryProviderBudget = 45 * time.Second
	// A provider normally reclaims up to recoveryBatchCap per iteration. This
	// independent ceiling prevents a faulty provider or a failed statfs from
	// turning a small positive receipt into an unbounded hot loop.
	recoveryMaxBatches = 1024
)

// StartRecovery accepts a trigger and returns before provider measurement or
// deletion begins. The request context cannot own the worker lifetime.
func (s *Service) StartRecovery(ctx context.Context, trigger, partition string, usedPercent float64, availableBytes int64, targetFreePercent float64, dryRun bool) (RecoveryRun, error) {
	if partition == "" {
		return RecoveryRun{}, fmt.Errorf("partition is required")
	}
	// Manual callers may omit pressure measurements. Resolve them at the
	// authority that owns the recovery run; otherwise zero values make the
	// target calculation fall back to the 10 GiB floor and can report
	// target_met after reclaiming only a small batch on a much fuller disk.
	if usedPercent == 0 && availableBytes == 0 {
		measuredUsed, measuredAvailable, measureErr := measureRecoverySpace(partition)
		if measureErr != nil {
			return RecoveryRun{}, fmt.Errorf("measure recovery partition %q: %w", partition, measureErr)
		}
		usedPercent = measuredUsed
		availableBytes = measuredAvailable
	}
	if availableBytes < 0 {
		return RecoveryRun{}, fmt.Errorf("available bytes cannot be negative")
	}
	if math.IsNaN(usedPercent) || math.IsInf(usedPercent, 0) || usedPercent < 0 || usedPercent > 100 {
		return RecoveryRun{}, fmt.Errorf("used percent must be between 0 and 100")
	}
	if math.IsNaN(targetFreePercent) || math.IsInf(targetFreePercent, 0) {
		return RecoveryRun{}, fmt.Errorf("target free percent must be finite")
	}
	if targetFreePercent < 0 || targetFreePercent >= 100 {
		return RecoveryRun{}, fmt.Errorf("target free percent must be between 0 and 100")
	}
	sequence := atomic.AddUint64(&s.recoverySeq, 1)
	id := fmt.Sprintf("recovery-%s-%d", s.recoveryInstance, sequence)
	done := make(chan struct{})
	run := &RecoveryRun{ID: id, Status: RecoveryRunning, Trigger: trigger, Partition: partition, StartedAt: s.now(), Done: done}
	s.recoveryMu.Lock()
	s.recoveryRuns[id] = run
	s.recoveryMu.Unlock()
	s.persistRecovery(ctx, *run)
	s.recoveryGate.Lock()
	locked := s.recoveryBusy
	if !locked {
		s.recoveryBusy = true
	}
	s.recoveryGate.Unlock()
	if locked {
		run.Status = RecoveryFailed
		run.Reason = "lock held by another recovery run"
		run.StoppedBecause = "lock_held"
		run.CompletedAt = s.now()
		s.persistRecovery(ctx, *run)
		close(done)
		return cloneRecoveryRun(*run), nil
	}
	initial := cloneRecoveryRun(*run)

	go func() {
		defer func() {
			s.recoveryGate.Lock()
			s.recoveryBusy = false
			s.recoveryGate.Unlock()
		}()
		s.recoveryGate.Lock()
		lockPath := s.recoveryLockPath
		s.recoveryGate.Unlock()
		release, lockErr := recoverylock.AcquireFor(lockPath, run.ID)
		if lockErr != nil {
			s.recoveryMu.Lock()
			run.Status = RecoveryFailed
			run.Reason = lockErr.Error()
			if errors.Is(lockErr, recoverylock.ErrLockHeld) {
				run.StoppedBecause = "lock_held"
			}
			run.CompletedAt = s.now()
			s.persistRecovery(context.Background(), *run)
			close(done)
			s.recoveryMu.Unlock()
			return
		}
		defer release()
		s.publishEvent(context.Background(), "storage.recovery.started", map[string]any{
			"run_id": run.ID, "trigger": trigger, "partition": partition,
		})
		var outcome PressureOutcome
		var err error
		target := recoveryTargetForRun(trigger, partition, usedPercent, availableBytes, targetFreePercent)
		s.recoveryMu.Lock()
		run.TargetFreeBytes = target
		s.recoveryMu.Unlock()
		// The caller context is intentionally not the worker lifetime: a
		// disconnected pressure sender must not cancel a server-owned run. The
		// service context still cancels it during API shutdown.
		s.recoveryGate.Lock()
		serviceContext := s.recoveryContext
		s.recoveryGate.Unlock()
		if serviceContext == nil {
			serviceContext = context.Background()
		}
		recoveryCtx, recoveryCancel := context.WithTimeout(serviceContext, recoveryMaxDuration)
		outcome, err = s.executeRecovery(recoveryCtx, run.ID, trigger, partition, availableBytes, target, dryRun)
		timedOut := recoveryCtx.Err() == context.DeadlineExceeded
		recoveryCancel()
		if err == nil && timedOut {
			outcome.Reason = "recovery_timeout"
		}
		s.recoveryMu.Lock()
		defer s.recoveryMu.Unlock()
		if err != nil {
			run.Status = RecoveryFailed
			run.Reason = err.Error()
			run.StoppedBecause = "error"
		} else {
			run.Status = RecoveryComplete
			run.Action = outcome.Action
			run.PlanID = outcome.PlanID
			run.EstimatedBytes = outcome.EstimatedBytes
			run.ReclaimedBytes = outcome.ReclaimedBytes
			run.Reason = outcome.Reason
			run.StoppedBecause = outcome.Reason
		}
		run.CompletedAt = s.now()
		s.persistRecovery(context.Background(), *run)
		s.publishEvent(context.Background(), "storage.recovery.completed", map[string]any{
			"run_id": run.ID, "status": run.Status, "reclaimed_bytes": run.ReclaimedBytes,
			"action": string(run.Action), "reason": run.Reason,
		})
		// Every server-owned run gets one work record, including runs that
		// reached the target without deleting or stopped with an error. A
		// missing reclaim receipt must not make a recovery attempt invisible to
		// the learning loop.
		if s.journal != nil {
			journalCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			if journalErr := s.journal.AppendRecovery(journalCtx, *run); journalErr != nil {
				s.publishEvent(context.Background(), "storage.recovery.journal_failed", map[string]any{"run_id": run.ID, "reason": "journal_unreachable"})
			}
			cancel()
		}
		close(done)
	}()
	return initial, nil
}

func recoveryTargetForRun(trigger, partition string, usedPercent float64, availableBytes int64, requestedFreePercent float64) int64 {
	target := recoveryTargetBytes(usedPercent, availableBytes, requestedFreePercent)
	// RATE identifies a specific hot governed root. It must be serviced even
	// when the device has ample free space for the ordinary pressure target;
	// otherwise a healthy device suppresses the only remediation path for a
	// runaway writer. Requiring one byte above the observed free space makes
	// the loop perform at least one bounded provider batch.
	if strings.Contains(strings.ToUpper(strings.TrimSpace(trigger)), "RATE") && partition != string(filepath.Separator) {
		return availableBytes + 1
	}
	return target
}

// recordObservedRecovery keeps warning-band pressure in the typed recovery
// ledger without invoking the deletion controller. Warning is an observation
// and escalation signal; only an explicit RATE or FLOOR trigger authorizes a
// recovery run from that band.
func (s *Service) recordObservedRecovery(ctx context.Context, trigger, partition string) RecoveryRun {
	sequence := atomic.AddUint64(&s.recoverySeq, 1)
	now := s.now()
	id := fmt.Sprintf("recovery-%s-%d", s.recoveryInstance, sequence)
	done := make(chan struct{})
	close(done)
	run := RecoveryRun{
		ID: id, Status: RecoveryComplete, Trigger: trigger, Partition: partition,
		Action: ActionObserved, Reason: "warning pressure observed; no recovery actions",
		StoppedBecause: "observed", StartedAt: now, CompletedAt: now, Done: done,
	}
	s.recoveryMu.Lock()
	s.recoveryRuns[id] = &run
	s.recoveryMu.Unlock()
	s.persistRecovery(ctx, run)
	return cloneRecoveryRun(run)
}

// executeRecovery deliberately measures and applies one provider at a time.
// A pressure report must be able to reclaim the first eligible batch without
// waiting for an exhaustive census of every provider on the host.
func (s *Service) executeRecovery(ctx context.Context, runID, trigger, partition string, free, target int64, dryRun bool) (PressureOutcome, error) {
	pol, err := s.CurrentPolicy(ctx)
	if err != nil {
		return PressureOutcome{}, err
	}
	action := ActionApplied
	if dryRun {
		action = ActionPreviewed
	}
	out := PressureOutcome{Band: BandHigh, Action: action, RunID: runID}
	rungSpent := make(map[string]int64)
	batches := 0
	if free >= target {
		out.Reason = "target_met"
		return out, nil
	}
	metas := s.registry.List()
	sort.SliceStable(metas, func(i, j int) bool {
		leftRank, rightRank := recoveryTierRank(metas[i]), recoveryTierRank(metas[j])
		if leftRank != rightRank {
			return leftRank < rightRank
		}
		leftPriority, rightPriority := recoveryProviderPriority(metas[i].ID), recoveryProviderPriority(metas[j].ID)
		if leftPriority != rightPriority {
			return leftPriority < rightPriority
		}
		return metas[i].ID < metas[j].ID
	})
	providerIDs := make([]string, 0, len(metas))
	for _, meta := range metas {
		providerIDs = append(providerIDs, meta.ID)
	}
	s.audit(ctx, AuditEvent{Type: "recovery.provider_order", PlanID: out.PlanID, Message: strings.Join(providerIDs, ",")})
providerLoop:
	for _, meta := range metas {
		providerPolicy := pol.Providers[meta.ID]
		s.audit(ctx, AuditEvent{Type: "recovery.provider_considered", PlanID: out.PlanID, ProviderID: meta.ID, Message: fmt.Sprintf("tier=%s owner_budget=%t enabled=%t approval=%s", meta.SafetyTier, meta.OwnerBudget, providerPolicy.Enabled, providerPolicy.ApprovalMode)})
		autonomous := autonomousTierAllowed(meta.SafetyTier)
		standingApproval := s.hasCurrentHostStandingApproval(meta.ID, meta.SafetyTier, pol)
		if autonomous {
			// Default-disabled is an ordinary retention posture. It must not
			// suppress the explicitly autonomous R0/R1 pressure rung.
			providerPolicy.Enabled = true
			providerPolicy.ApprovalMode = cleanup.ApprovalModeNone
		}
		if !autonomous && !(meta.SafetyTier == cleanup.SafetyTierSafeWithOwner && meta.OwnerBudget) && !standingApproval {
			out.ProvidersWithheld = append(out.ProvidersWithheld, meta.ID+" (operator line)")
			s.audit(ctx, AuditEvent{Type: "recovery.provider_skipped", PlanID: out.PlanID, ProviderID: meta.ID, Message: "provider withheld above autonomous authority"})
			continue
		}
		ownerBudget := meta.SafetyTier == cleanup.SafetyTierSafeWithOwner && meta.OwnerBudget
		if (!providerPolicy.Enabled && !ownerBudget && !standingApproval) || (!autonomous && providerPolicy.ApprovalMode == cleanup.ApprovalModeDisabled) {
			s.audit(ctx, AuditEvent{Type: "recovery.provider_skipped", PlanID: out.PlanID, ProviderID: meta.ID, Message: "provider disabled or lacks autonomous authority"})
			continue
		}
		if ownerBudget {
			providerPolicy.Enabled = true
			providerPolicy.ApprovalMode = cleanup.ApprovalModeNone
		}
		if autonomousTierAllowed(meta.SafetyTier) {
			providerPolicy.ApprovalMode = cleanup.ApprovalModeNone
		}
		// A rate signal names a specific child of the governed temporary
		// namespace. Its purpose is to stop a runaway writer, so the ordinary
		// retention age must not shield the hot fixture. Restrict this override
		// to a strict child path; never let an arbitrary RATE request turn the
		// whole host or the parent namespace into an age-free deletion target.
		if rateRecoveryChild(trigger, partition) && (meta.SafetyTier == cleanup.SafetyTierSafe || meta.SafetyTier == cleanup.SafetyTierRegenerable) {
			providerPolicy.MinAge = 0
			providerPolicy.AllowFreshReclaim = true
		}
		if standingApproval {
			providerPolicy.Enabled = true
			providerPolicy.ApprovalMode = cleanup.ApprovalModeNone
		}
		provider, ok := s.registry.Get(meta.ID)
		if !ok {
			s.audit(ctx, AuditEvent{Type: "recovery.provider_skipped", PlanID: out.PlanID, ProviderID: meta.ID, Message: "provider missing from registry"})
			continue
		}
		for batchIndex := 0; ; batchIndex++ {
			if batches >= recoveryMaxBatches {
				out.Reason = "batch_limit"
				return out, nil
			}
			rung := rungFor(meta.SafetyTier)
			if recoveryRungBudget(rung) > 0 && rungSpent[rung] >= recoveryRungBudget(rung) {
				break
			}
			providerCtx, providerCancel := context.WithTimeout(ctx, recoveryProviderBudget)
			estimate, estimateErr := provider.Estimate(providerCtx, cleanup.EstimateRequest{Scope: cleanup.ObservationScope{RootPaths: []string{partition}, Now: s.now(), Recovery: true}, Policy: providerPolicy})
			if estimateErr != nil {
				message := "estimate failed: " + cleanup.Redact(estimateErr.Error())
				if providerCtx.Err() != nil {
					message = "provider timeout during estimate"
				}
				s.audit(ctx, AuditEvent{Type: "recovery.provider_skipped", PlanID: out.PlanID, ProviderID: meta.ID, Message: message, Redacted: true})
				providerCancel()
				break
			}
			if estimate.EstimatedBytes <= 0 {
				s.audit(ctx, AuditEvent{Type: "recovery.provider_skipped", PlanID: out.PlanID, ProviderID: meta.ID, Message: "estimate reported no eligible bytes"})
				providerCancel()
				break
			}
			if out.PlanID == "" {
				out.PlanID = runID + "-plan"
			}
			preview, previewErr := provider.Preview(providerCtx, cleanup.PreviewRequest{Scope: cleanup.ObservationScope{RootPaths: []string{partition}, Now: s.now(), Recovery: true}, Policy: providerPolicy, Estimate: estimate})
			if previewErr != nil {
				message := "preview failed: " + cleanup.Redact(previewErr.Error())
				if providerCtx.Err() != nil {
					message = "provider timeout during preview"
				}
				s.audit(ctx, AuditEvent{Type: "recovery.provider_skipped", PlanID: out.PlanID, ProviderID: meta.ID, Message: message, Redacted: true})
				providerCancel()
				break
			}
			if len(preview.Items) == 0 {
				s.audit(ctx, AuditEvent{Type: "recovery.provider_skipped", PlanID: out.PlanID, ProviderID: meta.ID, Message: "preview reported no items"})
				providerCancel()
				break
			}
			batch := boundedRecoveryPreview(preview, recoveryBatchCap)
			if len(batch.Items) == 0 {
				providerCancel()
				break
			}
			out.EstimatedBytes += sumRecoveryPreviewBytes(batch.Items)
			if dryRun {
				out.Action = ActionPreviewed
				out.Reason = "dry-run: no providers applied"
				providerCancel()
				break
			}
			plan := Plan{ID: out.PlanID, PolicyVersion: pol.Version, Providers: []ProviderPlan{{ProviderID: meta.ID, ProviderVersion: meta.Version, Estimate: estimate, Preview: batch, Policy: providerPolicy}}}
			applyStarted := time.Now()
			result, applied, applyErr := s.applyProvider(providerCtx, plan, plan.Providers[0], ApplyInput{PlanID: plan.ID, PolicyVersion: pol.Version, ApprovalMode: cleanup.ApprovalModeNone, IdempotencyKey: fmt.Sprintf("%s|%s|%d", runID, meta.ID, batchIndex)})
			if applyErr != nil {
				if providerCtx.Err() != nil {
					s.audit(ctx, AuditEvent{Type: "recovery.provider_skipped", PlanID: plan.ID, ProviderID: meta.ID, Message: "provider timeout during apply"})
					providerCancel()
					out.Reason = "provider_timeout"
					return out, nil
				}
				providerCancel()
				return out, fmt.Errorf("recovery provider %s apply: %w", meta.ID, applyErr)
			}
			providerCancel()
			if !applied {
				s.audit(ctx, AuditEvent{Type: "recovery.provider_skipped", PlanID: plan.ID, ProviderID: meta.ID, Message: "provider was not runnable"})
				break
			}
			batches++
			out.ProvidersApplied = appendUnique(out.ProvidersApplied, meta.ID)
			out.ReclaimedBytes += result.ReclaimedBytes
			freeBefore := free
			free += result.ReclaimedBytes
			if measured, measureErr := measureRecoveryFreeBytes(partition); measureErr == nil && measured >= 0 {
				free = measured
			}
			rungSpent[rung] += result.ReclaimedBytes
			filesRemoved := recoveryFilesRemoved(result, batch)
			s.saveRecoveryAction(ctx, runID, meta.ID, rung, result.ReclaimedBytes, freeBefore, free, filesRemoved, time.Since(applyStarted), result)
			if free >= target {
				out.Reason = "target_met"
				return out, nil
			}
			if result.ReclaimedBytes == 0 {
				s.audit(ctx, AuditEvent{Type: "recovery.provider_zero_progress", PlanID: plan.ID, ProviderID: meta.ID, Message: "provider applied without reclaim; advancing to next provider"})
				continue providerLoop
			}
		}
	}
	if out.Reason == "" {
		out.Reason = "budget_exhausted_or_operator_line"
	}
	return out, nil
}

// recoveryFilesRemoved preserves an honest item count for providers that
// report reclaimed bytes but do not echo item IDs (for example a delegated
// owner or a batch-oriented cache provider). Explicit applied IDs remain the
// source of truth; otherwise the preview batch minus explicit skips is the
// bounded count represented by the provider result.
func recoveryFilesRemoved(result cleanup.ApplyResult, batch cleanup.Preview) int {
	if len(result.AppliedItems) > 0 {
		return len(result.AppliedItems)
	}
	if result.ReclaimedBytes <= 0 {
		return 0
	}
	count := len(batch.Items) - len(result.SkippedItems)
	if count < 0 {
		return 0
	}
	return count
}

// hasCurrentHostStandingApproval prevents a policy copied from another host
// from authorizing a conditional provider here. The setter validates this for
// new records, but the execution boundary must also defend against legacy or
// externally loaded policy rows.
func (s *Service) hasCurrentHostStandingApproval(providerID string, tier cleanup.SafetyTier, pol Policy) bool {
	if tier != cleanup.SafetyTierConditional || s.hostID == nil {
		return false
	}
	current := strings.TrimSpace(s.hostID())
	approval := pol.StandingApprovals[providerID]
	return current != "" && strings.TrimSpace(approval.HostID) == current
}

func rateRecoveryChild(trigger, partition string) bool {
	if !strings.Contains(strings.ToUpper(trigger), "RATE") {
		return false
	}
	base := strings.TrimSpace(os.Getenv("VROOLI_HOME"))
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return false
		}
		base = filepath.Join(home, ".vrooli")
	}
	base = filepath.Clean(filepath.Join(base, "tmp", "go-work"))
	partition = filepath.Clean(strings.TrimSpace(partition))
	rel, err := filepath.Rel(base, partition)
	return err == nil && rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !strings.Contains(rel, string(filepath.Separator))
}

// recoveryTargetBytes derives the usable capacity from df-style inputs. The
// prior implementation multiplied available bytes by the used percentage,
// which understated capacity as pressure increased and could make a run stop
// below its promised free-space target. Keep a conservative fallback when a
// sender reports an invalid percentage.
func recoveryTargetBytes(usedPercent float64, availableBytes int64, requestedFreePercent float64) int64 {
	if availableBytes < 0 {
		return 0
	}
	if usedPercent <= 0 || usedPercent >= 100 {
		return availableBytes
	}
	total := float64(availableBytes) / (1 - usedPercent/100)
	freePercent := requestedFreePercent
	if freePercent == 0 {
		freePercent = 15
	}
	target := int64(total * freePercent / 100)
	const floorBytes = int64(10 * 1024 * 1024 * 1024)
	if target < floorBytes {
		target = floorBytes
	}
	return target
}

func recoveryRungBudget(rung string) int64 {
	if rung == "R3" {
		return recoveryR3Budget
	}
	return 0 // R0-R2 are bounded by their providers' declarations.
}

func boundedRecoveryPreview(preview cleanup.Preview, capBytes int64) cleanup.Preview {
	bounded := preview
	bounded.Items = nil
	var total int64
	for _, item := range preview.Items {
		if len(bounded.Items) >= recoveryBatchItems {
			break
		}
		if item.Bytes <= 0 {
			continue
		}
		if item.Bytes > capBytes {
			if total == 0 && bounded.AllowSingleOvershoot {
				bounded.Items = append(bounded.Items, item)
				total = item.Bytes
			}
			continue
		}
		if total > capBytes-item.Bytes {
			continue
		}
		bounded.Items = append(bounded.Items, item)
		total += item.Bytes
	}
	return bounded
}

func sumRecoveryPreviewBytes(items []cleanup.PreviewItem) int64 {
	var total int64
	for _, item := range items {
		total += item.Bytes
	}
	return total
}

func appendUnique(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func rungFor(tier cleanup.SafetyTier) string {
	if tier == cleanup.SafetyTierSafe {
		return "R0"
	}
	if tier == cleanup.SafetyTierSafeWithOwner {
		return "R2"
	}
	if tier == cleanup.SafetyTierConditional {
		return "R3"
	}
	return "R1"
}

func recoveryTierRank(meta cleanup.ProviderMetadata) int {
	switch {
	case meta.SafetyTier == cleanup.SafetyTierSafe:
		return 0
	case meta.SafetyTier == cleanup.SafetyTierRegenerable:
		return 1
	case meta.SafetyTier == cleanup.SafetyTierSafeWithOwner && meta.OwnerBudget:
		return 2
	case meta.SafetyTier == cleanup.SafetyTierConditional:
		return 4
	default:
		return 3
	}
}

func recoveryProviderPriority(id string) int {
	// This plan's orphan rule is owned by storage-manager and has a concrete,
	// bounded local action. Process it before delegated owner providers in the
	// same R2 rung so an unreachable owner cannot consume the recovery pass
	// without advancing to the named orphan target.
	if id == "orphaned-database" {
		return 0
	}
	return 1
}

func (s *Service) saveRecoveryAction(ctx context.Context, runID, providerID, rung string, reclaimed, before, after int64, filesRemoved int, duration time.Duration, result cleanup.ApplyResult) {
	if ledger, ok := s.store.(interface {
		SaveRecoveryActionMetrics(context.Context, string, string, string, int64, int64, int64, int, time.Duration, cleanup.ApplyResult) error
	}); ok {
		_ = ledger.SaveRecoveryActionMetrics(ctx, runID, providerID, rung, reclaimed, before, after, filesRemoved, duration, result)
	} else if ledger, ok := s.store.(interface {
		SaveRecoveryAction(context.Context, string, string, string, int64, int64, int64, cleanup.ApplyResult) error
	}); ok {
		// Compatibility for lightweight stores used by older integrations.
		_ = ledger.SaveRecoveryAction(ctx, runID, providerID, rung, reclaimed, before, after, result)
	}
	// Action events are emitted after the durable write so consumers never see
	// an event for a batch that cannot be found in the recovery ledger. Event
	// delivery is intentionally best-effort and cannot fail the run.
	s.publishEvent(ctx, "storage.recovery.action", map[string]any{
		"run_id": runID, "rung": rung, "provider_id": providerID,
		"bytes_reclaimed": reclaimed, "files_removed": filesRemoved,
		"duration_ms": duration.Milliseconds(),
		"free_before": before, "free_after": after,
	})
}

func (s *Service) persistRecovery(ctx context.Context, run RecoveryRun) {
	if ledger, ok := s.store.(recoveryLedger); ok {
		_ = ledger.SaveRecoveryRun(ctx, run)
	}
}

func (s *Service) publishEvent(_ context.Context, eventType string, payload map[string]any) {
	if s.events == nil {
		return
	}
	// Event delivery is an observation side effect. Bound it independently of
	// the recovery context so an unavailable events service cannot consume the
	// controller's deletion budget or keep a completed run from terminating.
	eventCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = s.events.PublishDomainEvent(eventCtx, eventbus.DomainEvent{Source: "storage-manager", EventType: eventType, Payload: payload, Occurred: s.now()})
}

func (s *Service) WaitRecovery(ctx context.Context, id string) (RecoveryRun, error) {
	run, ok := s.recovery(id)
	if !ok {
		return RecoveryRun{}, fmt.Errorf("recovery run %q not found", id)
	}
	select {
	case <-run.Done:
		latest, _ := s.recovery(id)
		return latest, nil
	case <-ctx.Done():
		return RecoveryRun{}, ctx.Err()
	}
}

func (s *Service) ListRecovery(limit int) []RecoveryRun {
	s.recoveryMu.Lock()
	defer s.recoveryMu.Unlock()
	out := make([]RecoveryRun, 0, len(s.recoveryRuns))
	for _, run := range s.recoveryRuns {
		out = append(out, cloneRecoveryRun(*run))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].StartedAt.After(out[j].StartedAt) })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}

// ListRecoveryContext prefers the durable ledger when available and falls
// back to the process-local view for memory-backed test services.
func (s *Service) ListRecoveryContext(ctx context.Context, limit int) ([]RecoveryRun, error) {
	if ledger, ok := s.store.(interface {
		ListRecoveryRuns(context.Context, int) ([]RecoveryRun, error)
	}); ok {
		return ledger.ListRecoveryRuns(ctx, limit)
	}
	return s.ListRecovery(limit), nil
}

func (s *Service) recovery(id string) (RecoveryRun, bool) {
	s.recoveryMu.Lock()
	defer s.recoveryMu.Unlock()
	run, ok := s.recoveryRuns[id]
	if !ok {
		return RecoveryRun{}, false
	}
	return cloneRecoveryRun(*run), true
}

func cloneRecoveryRun(run RecoveryRun) RecoveryRun {
	// Do not expose the internal completion channel as a transport field.
	return run
}
