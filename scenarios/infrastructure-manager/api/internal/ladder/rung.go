// Package ladder grades the host's device layer against the five-rung
// capability ladder and joins that grading onto the substrate projection's
// authored cells.
//
// The rungs are ordered by DEPENDENCY, not by preference: a device must be
// identified before it can be measured, measured before the measurement can be
// retained, retained before an operator can act on it, and acted upon before a
// forward-looking signal means anything. The ordering is enforced, not merely
// documented — a rung never reports covered while the rung below it is blind,
// because a "healthy" anticipation grade computed over a device nobody could
// identify is a confident statement about nothing.
//
// This package reads and grades. It has NO actuation path and must not grow
// one: the instrument has no controller letter, so a restart, reconcile or
// mutate verb here would be a contract violation rather than a feature.
package ladder

import "fmt"

// Rung names one step of the capability ladder. The tokens match the device
// graph owner's vocabulary exactly, so a rung never changes meaning in
// transit.
type Rung string

const (
	RungIdentity     Rung = "identity"
	RungTelemetry    Rung = "telemetry"
	RungEvidence     Rung = "evidence"
	RungControl      Rung = "control"
	RungAnticipation Rung = "anticipation"
)

// Rungs is the ladder in dependency order, lowest first.
var Rungs = []Rung{RungIdentity, RungTelemetry, RungEvidence, RungControl, RungAnticipation}

var rungIndex = func() map[Rung]int {
	index := make(map[Rung]int, len(Rungs))
	for position, rung := range Rungs {
		index[rung] = position
	}
	return index
}()

// Index returns the rung's position in the ladder, or -1 for a token outside
// the vocabulary. An unrecognised rung is never silently placed at the bottom,
// because the bottom is the rung everything else depends on.
func (r Rung) Index() int {
	if position, ok := rungIndex[r]; ok {
		return position
	}
	return -1
}

// Below returns the rung immediately beneath this one.
func (r Rung) Below() (Rung, bool) {
	position := r.Index()
	if position <= 0 {
		return "", false
	}
	return Rungs[position-1], true
}

// ParseRung normalises a token onto the vocabulary. An unrecognised token is
// an error, never a downgrade to a valid rung.
func ParseRung(token string) (Rung, error) {
	rung := Rung(token)
	if rung.Index() < 0 {
		return "", fmt.Errorf("rung %q is not in the ladder vocabulary %v", token, Rungs)
	}
	return rung, nil
}

// Observation is the outcome of grading one rung. The first four tokens are
// the device graph owner's; ObservationBlocked is this package's, and it is
// the whole point of the dependency ordering.
type Observation string

const (
	// ObservationMeasured means a real value was obtained from the host.
	ObservationMeasured Observation = "measured"
	// ObservationUnmeasurable means the rung applies and a value should exist,
	// but the host refused or could not produce it. Never zero, never healthy.
	ObservationUnmeasurable Observation = "unmeasurable"
	// ObservationUnavailable means the mechanism that would produce the value
	// is not present on this host at all.
	ObservationUnavailable Observation = "unavailable"
	// ObservationNotApplicable means the rung is meaningless for this device
	// class. It is not a gap and never blocks the rungs above it.
	ObservationNotApplicable Observation = "not_applicable"
	// ObservationBlocked means the rung was not graded because a rung beneath
	// it is blind. It is reported instead of a grade so a covered-looking
	// upper rung can never rest on an ungraded lower one.
	ObservationBlocked Observation = "blocked"
	// ObservationUnread means no source produced a grade for this rung at all.
	// It is distinct from unmeasurable — nothing was even attempted — and it
	// is what a cell carries while its join is unavailable.
	ObservationUnread Observation = "unread"
)

// Supports reports whether this observation lets the rungs above it be graded.
// Only a real measurement does, plus a rung that is meaningless for the class
// and therefore cannot be blind.
func (o Observation) Supports() bool {
	return o == ObservationMeasured || o == ObservationNotApplicable
}

// RungReading is one rung's grade with the evidence for it.
type RungReading struct {
	Rung Rung
	// Observation is the grade. Reason is mandatory for everything except a
	// plain measurement: an ungraded rung with no reason is indistinguishable
	// from one nobody looked at.
	Observation Observation
	Reason      string
	Mechanism   string
	Remediation string
	// BlockedBy names the lower rung whose blindness suppressed this grade. It
	// is set only for ObservationBlocked.
	BlockedBy Rung
}

// ApplyDependency enforces the ladder ordering over one device's readings.
//
// It walks the ladder from the bottom. Once a rung fails to support the rungs
// above it, every higher rung is replaced by ObservationBlocked naming the
// rung that blinded it — including rungs the owner graded as measured. That
// last part is the rule with teeth: a device whose identity could not be
// resolved may still emit a temperature, and reporting that temperature as a
// covered telemetry rung attributes a real reading to a device nobody can
// name. The reading is not wrong; the claim built on it is.
//
// Rungs with no reading at all are returned as ObservationUnread rather than
// omitted, so the ladder is always the full five rungs and a missing grade is
// visible instead of absent.
func ApplyDependency(readings map[Rung]RungReading) []RungReading {
	out := make([]RungReading, 0, len(Rungs))
	var (
		blindRung        Rung
		blindObservation Observation
		blinded          bool
	)
	for _, rung := range Rungs {
		reading, ok := readings[rung]
		if !ok {
			reading = RungReading{
				Rung:        rung,
				Observation: ObservationUnread,
				Reason:      "no source graded this rung",
			}
		}
		reading.Rung = rung
		if blinded {
			out = append(out, RungReading{
				Rung:        rung,
				Observation: ObservationBlocked,
				Reason:      fmt.Sprintf("the %s rung beneath it is %s, so this rung cannot be graded", blindRung, blindObservation),
				BlockedBy:   blindRung,
			})
			continue
		}
		out = append(out, reading)
		if !reading.Observation.Supports() {
			blinded, blindRung, blindObservation = true, rung, reading.Observation
		}
	}
	return out
}
