package services

// DOC: docs/operations/RUNBOOK.md#disk-pressure

import (
	"fmt"
	"strings"
	"time"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/collectors"
)

// PressureBand is how severe the current disk pressure is.
//
// The band, not the raw percentage, is what decides whether anything happens:
// warning records, high asks for a cleanup preview, critical permits
// unattended safe-tier reclamation.
type PressureBand int

const (
	// BandNormal is below the warning boundary. Record the sample, alert nobody.
	BandNormal PressureBand = iota
	// BandWarning persists a violation and takes no remedial action.
	BandWarning
	// BandHigh requests a conservative cleanup preview. Nothing is deleted.
	BandHigh
	// BandCritical permits safe-tier reclamation with no operator present.
	BandCritical
)

func (b PressureBand) String() string {
	switch b {
	case BandWarning:
		return "warning"
	case BandHigh:
		return "high"
	case BandCritical:
		return "critical"
	default:
		return "normal"
	}
}

// Severity maps a band onto the severity vocabulary the alert service filters on.
func (b PressureBand) Severity() string {
	switch b {
	case BandWarning:
		return "medium"
	case BandHigh:
		return "high"
	case BandCritical:
		return "critical"
	default:
		return "low"
	}
}

// classifyBand maps a usage percentage to a band using only settings-provided
// boundaries. It is pure so the whole table can be walked in a test.
func classifyBand(usedPercent float64, settings Settings) PressureBand {
	switch {
	case usedPercent >= settings.DiskCriticalPercent:
		return BandCritical
	case usedPercent >= settings.DiskHighPercent:
		return BandHigh
	case usedPercent >= settings.DiskThreshold:
		return BandWarning
	default:
		return BandNormal
	}
}

// BandObservation is the evidence that caused a band transition. Storing it
// makes an escalation auditable after the fact instead of leaving an operator
// to infer why the system acted.
type BandObservation struct {
	From           PressureBand `json:"from"`
	To             PressureBand `json:"to"`
	UsedPercent    float64      `json:"used_percent"`
	AvailableBytes int64        `json:"available_bytes"`
	At             time.Time    `json:"at"`
	FastFill       bool         `json:"fast_fill"`
}

// bandTracker owns the escalation state machine: debounce, cooldown, and the
// fast-fill bypass.
//
// It exists because a level-only rule is unusable in practice. During the
// incident the disk sat above its threshold for days; a rule that emits on
// every tick would have produced thousands of identical records and trained
// the operator to ignore all of them.
type bandTracker struct {
	current PressureBand

	// pending is a band observed but not yet in effect, and pendingTicks is
	// how many consecutive times it has been seen.
	pending      PressureBand
	pendingTicks int

	// lastEmittedAt is when a record was last produced for the current band.
	lastEmittedAt time.Time
	hasEmitted    bool

	lastObservation *BandObservation
}

// bandDecision is the outcome of evaluating one sample.
type bandDecision struct {
	Band PressureBand
	// Emit reports whether this sample should produce a durable record.
	Emit bool
	// Transition is set when the effective band changed on this sample.
	Transition *BandObservation
}

// evaluate advances the state machine by one observation.
//
// Escalation is debounced, de-escalation is not: dropping below a boundary is
// good news and holding on to a stale high band would keep remediation armed
// after the pressure is gone.
func (t *bandTracker) evaluate(
	usage collectors.DiskUsage,
	previousPercent float64,
	hasPrevious bool,
	settings Settings,
	now time.Time,
) bandDecision {
	observed := classifyBand(usage.UsedPercent, settings)

	fastFill := hasPrevious &&
		observed > t.current &&
		usage.UsedPercent-previousPercent >= settings.DiskFastFillJumpPercent

	effective := t.resolveEffectiveBand(observed, settings, fastFill)
	decision := bandDecision{Band: effective}

	if effective != t.current {
		observation := &BandObservation{
			From:           t.current,
			To:             effective,
			UsedPercent:    usage.UsedPercent,
			AvailableBytes: usage.AvailableBytes,
			At:             now,
			FastFill:       fastFill,
		}
		escalating := effective > t.current

		t.current = effective
		t.lastObservation = observation
		// A new band starts its cooldown fresh, so the next band can escalate
		// immediately rather than inheriting the previous band's timer.
		t.hasEmitted = false
		decision.Transition = observation

		// Only escalation is worth a record. De-escalation is recorded as a
		// transition for the audit trail but raises no alert.
		decision.Emit = escalating && effective >= BandWarning
		if decision.Emit {
			t.markEmitted(now)
		}
		return decision
	}

	// Same band as last time: emit only once the cooldown has expired.
	if effective >= BandWarning && t.cooldownExpired(now, settings) {
		decision.Emit = true
		t.markEmitted(now)
	}
	return decision
}

// resolveEffectiveBand applies debounce to an observed band.
func (t *bandTracker) resolveEffectiveBand(observed PressureBand, settings Settings, fastFill bool) PressureBand {
	if observed == t.current {
		t.pendingTicks = 0
		t.pending = observed
		return t.current
	}

	// A fast fill or a de-escalation takes effect on the first observation.
	if fastFill || observed < t.current {
		t.pendingTicks = 0
		t.pending = observed
		return observed
	}

	if observed != t.pending {
		t.pending = observed
		t.pendingTicks = 1
	} else {
		t.pendingTicks++
	}

	if t.pendingTicks >= settings.DiskEscalationDebounceTicks {
		t.pendingTicks = 0
		return observed
	}
	return t.current
}

func (t *bandTracker) cooldownExpired(now time.Time, settings Settings) bool {
	if !t.hasEmitted {
		return true
	}
	cooldown := time.Duration(settings.DiskEscalationCooldownSeconds) * time.Second
	return now.Sub(t.lastEmittedAt) >= cooldown
}

func (t *bandTracker) markEmitted(now time.Time) {
	t.lastEmittedAt = now
	t.hasEmitted = true
}

// MarshalJSON renders a band by name.
//
// The operator surface is read by humans during an incident; "warning" is
// legible where the underlying 1 is not.
func (b PressureBand) MarshalJSON() ([]byte, error) {
	return []byte(`"` + b.String() + `"`), nil
}

// UnmarshalJSON accepts a band name, rejecting anything unrecognised rather
// than silently decoding to normal.
func (b *PressureBand) UnmarshalJSON(data []byte) error {
	name := strings.Trim(string(data), `"`)
	for _, candidate := range []PressureBand{BandNormal, BandWarning, BandHigh, BandCritical} {
		if candidate.String() == name {
			*b = candidate
			return nil
		}
	}
	return fmt.Errorf("unknown pressure band %q", name)
}
