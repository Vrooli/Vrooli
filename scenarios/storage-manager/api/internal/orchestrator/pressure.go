package orchestrator

// DOC: docs/reference/providers.md#autonomous-safe-tier-remediation

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"storage-manager/internal/cleanup"
)

// ErrInvalidPressureSignal marks a report the caller got wrong, as opposed to
// one storage-manager could not service.
//
// The distinction is what lets the transport choose an honest status code. A
// reporting safeguard that receives InvalidArgument should stop and fix its
// request; one that receives Internal should keep reporting, because the signal
// was fine and the fault is here.
var ErrInvalidPressureSignal = errors.New("invalid pressure signal")

// PressureBand is how severe a reporting safeguard judged disk pressure to be.
type PressureBand string

const (
	BandWarning  PressureBand = "warning"
	BandHigh     PressureBand = "high"
	BandCritical PressureBand = "critical"
)

// ParsePressureBand converts a reported band name, rejecting anything it does
// not recognise.
//
// An unknown band is an error, never a default. Defaulting an unrecognised
// value to critical would let a typo authorise unattended deletion; defaulting
// it to warning would let a typo silently disable remediation. Both are worse
// than refusing the request.
func ParsePressureBand(raw string) (PressureBand, error) {
	switch PressureBand(strings.ToLower(strings.TrimSpace(raw))) {
	case BandWarning:
		return BandWarning, nil
	case BandHigh:
		return BandHigh, nil
	case BandCritical:
		return BandCritical, nil
	default:
		return "", fmt.Errorf("%w: unknown pressure band %q: expected warning, high, or critical", ErrInvalidPressureSignal, raw)
	}
}

// PressureAction is what storage-manager did about a report.
type PressureAction string

const (
	ActionObserved     PressureAction = "observed"
	ActionPreviewed    PressureAction = "previewed"
	ActionApplied      PressureAction = "applied"
	ActionDeduplicated PressureAction = "deduplicated"
	ActionSuppressed   PressureAction = "suppressed"
)

// PressureSignal is an inbound report from a safeguard.
type PressureSignal struct {
	SourceScenario string
	Partition      string
	UsedPercent    float64
	Band           PressureBand
	AvailableBytes int64
}

// PressureOutcome is the result of handling a report.
type PressureOutcome struct {
	Band                   PressureBand
	Action                 PressureAction
	PlanID                 string
	EstimatedBytes         int64
	ReclaimedBytes         int64
	ProvidersApplied       []string
	ProvidersWithheld      []string
	Reason                 string
	BugReference           string
	AutonomousApplyEnabled bool
}

// autonomousTierAllowed is the single gate deciding whether a provider may run
// with no operator present.
//
// This function is correctness-critical and deliberately tiny. At the critical
// band the system deletes without asking, so the safety-tier classification
// becomes the only thing standing between remediation and silent data loss.
// Only SafetyTierSafe qualifies: safe-tier artifacts are reconstructible.
//
// SafetyTierSafeWithOwner is excluded on purpose. Those providers delegate
// deletion to another scenario and default to owner approval, so running them
// unattended would bypass an approval the owning scenario asked for. Making
// them autonomous is an operator decision, not a default.
//
// Every autonomous execution path must route through here. Do not add a second
// tier check elsewhere.
func autonomousTierAllowed(tier cleanup.SafetyTier) bool {
	return tier == cleanup.SafetyTierSafe
}

// pressureGuard collapses duplicate reports of the same pressure event.
//
// Two safeguards reach storage-manager independently by design, so duplicate
// and concurrent reports are expected rather than exceptional. The guard holds
// an in-flight marker for the whole execution, not just a timestamp, so two
// simultaneous reports cannot both pass the check before either finishes.
type pressureGuard struct {
	mu       sync.Mutex
	inFlight map[string]bool
	lastRun  map[string]time.Time
	window   time.Duration
}

func newPressureGuard(window time.Duration) *pressureGuard {
	return &pressureGuard{
		inFlight: make(map[string]bool),
		lastRun:  make(map[string]time.Time),
		window:   window,
	}
}

// acquire reports whether the caller may execute. The key deliberately excludes
// the source scenario: two different safeguards reporting the same partition at
// the same band are the same event, which is the whole point of the guard.
func (g *pressureGuard) acquire(partition string, band PressureBand, now time.Time) (bool, string) {
	key := partition + "|" + string(band)

	g.mu.Lock()
	defer g.mu.Unlock()

	if g.inFlight[key] {
		return false, "an execution for this partition and band is already in flight"
	}
	if last, ok := g.lastRun[key]; ok && now.Sub(last) < g.window {
		return false, fmt.Sprintf("an execution for this partition and band completed %s ago, within the %s deduplication window", now.Sub(last).Truncate(time.Second), g.window)
	}

	g.inFlight[key] = true
	return true, ""
}

// release clears the in-flight marker and starts the deduplication window.
func (g *pressureGuard) release(partition string, band PressureBand, now time.Time) {
	key := partition + "|" + string(band)

	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.inFlight, key)
	g.lastRun[key] = now
}

// ReportPressure handles an inbound pressure signal.
//
// Escalation is graded: warning plans and applies only the safe tier, high and
// critical apply only the provably safe tier with no operator present. High is
// an automatic remediation band because it is already an affirmative signal
// from the host-pressure observer; the safety boundary remains per-provider.
func (s *Service) ReportPressure(ctx context.Context, signal PressureSignal) (PressureOutcome, error) {
	if _, err := ParsePressureBand(string(signal.Band)); err != nil {
		return PressureOutcome{}, err
	}
	if strings.TrimSpace(signal.Partition) == "" {
		return PressureOutcome{}, fmt.Errorf("%w: partition is required", ErrInvalidPressureSignal)
	}

	outcome := PressureOutcome{
		Band:                   signal.Band,
		AutonomousApplyEnabled: s.AutonomousApplyEnabled(),
	}

	_ = s.audit(ctx, AuditEvent{
		Type:    "pressure.reported",
		Message: fmt.Sprintf("%s reported %s pressure on %s at %.1f%%", signal.SourceScenario, signal.Band, signal.Partition, signal.UsedPercent),
	})

	// Warning is the early counterweight: plan and run only the safe tier, while
	// recording an operator-visible escalation for the fastest unbounded owner.
	// Owner-delegated and conditional providers remain withheld by the same
	// autonomousTierAllowed gate used at critical pressure.
	if signal.Band == BandWarning {
		allowed, reason := s.pressure.acquire(signal.Partition, signal.Band, s.now())
		if !allowed {
			outcome.Action = ActionDeduplicated
			outcome.Reason = reason
			_ = s.audit(ctx, AuditEvent{Type: "pressure.deduplicated", Message: reason})
			return outcome, nil
		}
		defer func() { s.pressure.release(signal.Partition, signal.Band, s.now()) }()
		plan, err := s.Plan(ctx, cleanup.ObservationScope{Now: s.now()})
		if err != nil {
			return PressureOutcome{}, fmt.Errorf("plan for warning pressure: %w", err)
		}
		outcome.PlanID = plan.ID
		outcome.EstimatedBytes = plan.TotalBytes
		s.fileWarningBug(ctx, signal, &outcome)
		_ = s.audit(ctx, AuditEvent{Type: "pressure.warning_action", PlanID: plan.ID, Message: fmt.Sprintf("warning pressure on %s: safe-tier cleanup ran and growth review is required for the fastest unbounded owner", signal.Partition)})
		return s.applyAutonomously(ctx, signal, plan, outcome)
	}

	allowed, reason := s.pressure.acquire(signal.Partition, signal.Band, s.now())
	if !allowed {
		outcome.Action = ActionDeduplicated
		outcome.Reason = reason
		_ = s.audit(ctx, AuditEvent{Type: "pressure.deduplicated", Message: reason})
		return outcome, nil
	}
	defer func() { s.pressure.release(signal.Partition, signal.Band, s.now()) }()

	plan, err := s.Plan(ctx, cleanup.ObservationScope{Now: s.now()})
	if err != nil {
		return PressureOutcome{}, fmt.Errorf("plan for %s pressure: %w", signal.Band, err)
	}
	outcome.PlanID = plan.ID
	outcome.EstimatedBytes = plan.TotalBytes

	if signal.Band == BandHigh {
		_ = s.audit(ctx, AuditEvent{
			Type:    "pressure.high_action",
			PlanID:  plan.ID,
			Message: fmt.Sprintf("high pressure on %s: applying provably safe-tier cleanup", signal.Partition),
		})
	}

	return s.applyAutonomously(ctx, signal, plan, outcome)
}

// fileWarningBug reports the fastest growing unbounded entry at most once per
// owner per 24 hours. Reporting is advisory and never prevents safe-tier
// cleanup when Prompt Manager or growth storage is unavailable.
func (s *Service) fileWarningBug(ctx context.Context, signal PressureSignal, outcome *PressureOutcome) {
	s.warningMu.Lock()
	deps := s.warningDeps
	s.warningMu.Unlock()
	if deps.FastestUnbounded == nil || deps.FileBug == nil {
		return
	}
	target, ok, err := deps.FastestUnbounded(ctx)
	if err != nil {
		_ = s.audit(ctx, AuditEvent{Type: "pressure.bug_failed", Message: cleanup.Redact(err.Error()), Redacted: true})
		return
	}
	if !ok || target.OwnerID == "" || target.EntryName == "" || target.SlopeBytesPerHour <= 0 {
		return
	}
	ownerKey := target.OwnerKind + "/" + target.OwnerID
	now := s.now()
	s.warningMu.Lock()
	if s.warningBugBusy[ownerKey] || (!s.warningBugLast[ownerKey].IsZero() && now.Sub(s.warningBugLast[ownerKey]) < 24*time.Hour) {
		s.warningMu.Unlock()
		return
	}
	s.warningBugBusy[ownerKey] = true
	s.warningMu.Unlock()

	bug := WarningBugReport{
		Title:      fmt.Sprintf("Unbounded storage growth: %s/%s", ownerKey, target.EntryName),
		SignalType: "code-defect",
		Severity:   "major",
		Repro: []string{
			"Run `storage-manager storage growth --window=24h`.",
			fmt.Sprintf("Inspect owner `%s/%s` entry `%s`.", target.OwnerKind, target.OwnerID, target.EntryName),
			"Send a warning pressure signal to storage-manager.",
		},
		Expected:    fmt.Sprintf("The owner declares a retention ceiling for `%s`.", target.EntryName),
		Actual:      fmt.Sprintf("`%s` is unbounded and growing at %.2f bytes per hour from %d current bytes.", target.EntryName, target.SlopeBytesPerHour, target.CurrentBytes),
		Description: fmt.Sprintf("Warning pressure was reported for partition %s at %.1f%% used. The growth projection ranked this owner first among unbounded entries.", signal.Partition, signal.UsedPercent),
		Context: map[string]string{
			"scenario": target.OwnerID,
			"command":  "storage-manager storage growth",
		},
		HonestyFlags:   []string{"ai-generated-summary"},
		IdempotencyKey: fmt.Sprintf("storage-manager-warning|%s|%s", ownerKey, now.UTC().Format("2006-01-02")),
	}
	ref, err := deps.FileBug(ctx, bug)
	s.warningMu.Lock()
	delete(s.warningBugBusy, ownerKey)
	if err == nil {
		s.warningBugLast[ownerKey] = now
	}
	s.warningMu.Unlock()
	if err != nil {
		_ = s.audit(ctx, AuditEvent{Type: "pressure.bug_failed", Message: cleanup.Redact(err.Error()), Redacted: true})
		return
	}
	outcome.BugReference = ref
	_ = s.audit(ctx, AuditEvent{Type: "pressure.bug_filed", Message: fmt.Sprintf("warning pressure filed growth bug %s for %s", cleanup.Redact(ref), ownerKey), Redacted: true})
}

// applyAutonomously runs the safe-tier subset of a plan with no operator
// present. Warning and critical bands both reach this function; the tier gate
// is deliberately identical for both paths.
func (s *Service) applyAutonomously(ctx context.Context, signal PressureSignal, plan Plan, outcome PressureOutcome) (PressureOutcome, error) {
	// Partition the plan by what may run unattended, so the withheld set can
	// be reported rather than silently dropped.
	var runnable []ProviderPlan
	for _, pp := range plan.Providers {
		provider, ok := s.registry.Get(pp.ProviderID)
		if !ok {
			return PressureOutcome{}, fmt.Errorf("provider %q missing from registry", pp.ProviderID)
		}
		tier := provider.Metadata().SafetyTier
		if !autonomousTierAllowed(tier) {
			outcome.ProvidersWithheld = append(outcome.ProvidersWithheld, fmt.Sprintf("%s (%s)", pp.ProviderID, tier))
			continue
		}
		runnable = append(runnable, pp)
	}

	if !s.AutonomousApplyEnabled() {
		outcome.Action = ActionSuppressed
		outcome.Reason = "autonomous apply is disabled by the kill switch; the report was recorded and nothing was deleted"
		_ = s.audit(ctx, AuditEvent{
			Type:    "pressure.suppressed",
			PlanID:  plan.ID,
			Message: outcome.Reason,
		})
		return outcome, nil
	}

	if len(runnable) == 0 {
		outcome.Action = ActionApplied
		outcome.Reason = "no safe-tier provider was eligible to run"
		_ = s.audit(ctx, AuditEvent{Type: "pressure.applied", PlanID: plan.ID, Message: outcome.Reason})
		return outcome, nil
	}

	// Reclaimed bytes must come from what Apply actually removed, never from
	// the sum of estimates: two safeguards reporting the same event would
	// otherwise double-count space that was only freed once.
	for _, pp := range runnable {
		result, applied, err := s.applyProvider(ctx, plan, pp, ApplyInput{
			PlanID:         plan.ID,
			PolicyVersion:  plan.PolicyVersion,
			ApprovalMode:   pp.Policy.ApprovalMode,
			IdempotencyKey: autonomousIdempotencyKey(plan.ID, signal),
		})
		if err != nil {
			_ = s.audit(ctx, AuditEvent{Type: "pressure.apply_failed", PlanID: plan.ID, ProviderID: pp.ProviderID, Message: cleanup.Redact(err.Error()), Redacted: true})
			continue
		}
		if !applied {
			continue
		}
		outcome.ProvidersApplied = append(outcome.ProvidersApplied, pp.ProviderID)
		outcome.ReclaimedBytes += result.ReclaimedBytes
	}

	outcome.Action = ActionApplied
	_ = s.audit(ctx, AuditEvent{
		Type:    "pressure.applied",
		PlanID:  plan.ID,
		Message: fmt.Sprintf("%s pressure on %s: ran %v, reclaimed %d bytes, withheld %v", signal.Band, signal.Partition, outcome.ProvidersApplied, outcome.ReclaimedBytes, outcome.ProvidersWithheld),
	})
	return outcome, nil
}

// autonomousIdempotencyKey ties an autonomous apply to the plan and the event
// that triggered it, so a replay of the same event is recognised.
func autonomousIdempotencyKey(planID string, signal PressureSignal) string {
	return fmt.Sprintf("autonomous|%s|%s|%s", planID, signal.Partition, signal.Band)
}

// AutonomousApplyEnabled reports whether unattended apply is permitted.
func (s *Service) AutonomousApplyEnabled() bool {
	s.autonomousMu.RLock()
	defer s.autonomousMu.RUnlock()
	return s.autonomousApply
}

// SetAutonomousApplyEnabled is the kill switch. Disabling it stops all
// unattended deletion while leaving observation, planning, and preview intact,
// so an operator can silence remediation without going blind.
func (s *Service) SetAutonomousApplyEnabled(enabled bool) {
	s.autonomousMu.Lock()
	s.autonomousApply = enabled
	s.autonomousMu.Unlock()
}
