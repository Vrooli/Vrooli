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
// Escalation is graded: warning records the observation, high runs an estimate
// and preview without deleting anything, and critical applies safe-tier
// providers with no operator present.
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

	// Warning observes and records. Nothing is planned or executed, so it does
	// not consume the deduplication window either.
	if signal.Band == BandWarning {
		outcome.Action = ActionObserved
		return outcome, nil
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
		outcome.Action = ActionPreviewed
		_ = s.audit(ctx, AuditEvent{
			Type:    "pressure.previewed",
			PlanID:  plan.ID,
			Message: fmt.Sprintf("high pressure on %s: %d bytes reclaimable, nothing deleted", signal.Partition, plan.TotalBytes),
		})
		return outcome, nil
	}

	return s.applyAutonomously(ctx, signal, plan, outcome)
}

// applyAutonomously runs the safe-tier subset of a plan with no operator
// present. It is only reachable at the critical band.
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
		Message: fmt.Sprintf("critical pressure on %s: ran %v, reclaimed %d bytes, withheld %v", signal.Partition, outcome.ProvidersApplied, outcome.ReclaimedBytes, outcome.ProvidersWithheld),
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
