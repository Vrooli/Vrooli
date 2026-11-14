// Package evidence owns cloud-side release evidence: the capability profile
// (cloud-launch-v1) drawn from the certification matrix, append-only evidence
// records bound to one release, target and operation, the per-cell
// disposition evaluation, the exact review identity a governance decision
// binds to, and the publication decision that re-checks Deployment Manager's
// approval immediately before an activation effect.
//
// Nothing in this package performs a remote effect. It is consumed by
// api/ramp (adapters), api/evidencesvc (Connect), the REST handlers and, on
// the wire, by deployment-manager.
package evidence

import (
	"strings"

	"scenario-to-cloud/certification"

	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
	evidencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/evidence"
)

// ProducerRef is the stable producer identity stamped on every record and
// receipt this service emits. Consumers compare it exactly.
const ProducerRef = "scenario-to-cloud"

// Disposition is the shared evidence vocabulary for one cell. The values are
// distinct answers: only Passed satisfies a required cell.
type Disposition string

// Disposition vocabulary.
const (
	DispositionPassed      Disposition = "passed"
	DispositionFailed      Disposition = "failed"
	DispositionSkipped     Disposition = "skipped"
	DispositionUnsupported Disposition = "unsupported"
	DispositionUnavailable Disposition = "unavailable"
	// DispositionMissing is never recorded; it is the evaluation answer when
	// no record exists for a cell.
	DispositionMissing Disposition = "missing"
)

// Valid reports whether d is a recordable disposition (missing is not).
func (d Disposition) Recordable() bool {
	switch d {
	case DispositionPassed, DispositionFailed, DispositionSkipped, DispositionUnsupported, DispositionUnavailable:
		return true
	}
	return false
}

// FromVerdict maps a certification receipt verdict onto the disposition
// vocabulary.
func FromVerdict(v certification.Verdict) Disposition {
	switch v {
	case certification.VerdictPassed:
		return DispositionPassed
	case certification.VerdictFailed:
		return DispositionFailed
	case certification.VerdictSkipped:
		return DispositionSkipped
	case certification.VerdictUnsupported:
		return DispositionUnsupported
	default:
		return DispositionUnavailable
	}
}

// FromRamp maps a delivery-ramp journey disposition. Degraded is a failed
// cell: a required cell either proved its assertion or it did not.
func FromRamp(d deliveryramp.Disposition) Disposition {
	switch d {
	case deliveryramp.DispositionPass:
		return DispositionPassed
	case deliveryramp.DispositionFailed, deliveryramp.DispositionDegraded:
		return DispositionFailed
	case deliveryramp.DispositionNotRun:
		return DispositionSkipped
	case deliveryramp.DispositionUnsupported:
		return DispositionUnsupported
	default:
		return DispositionUnavailable
	}
}

// ToRamp maps back onto the delivery-ramp vocabulary for JourneyResult.
func (d Disposition) ToRamp() deliveryramp.Disposition {
	switch d {
	case DispositionPassed:
		return deliveryramp.DispositionPass
	case DispositionFailed:
		return deliveryramp.DispositionFailed
	case DispositionSkipped:
		return deliveryramp.DispositionNotRun
	case DispositionUnsupported:
		return deliveryramp.DispositionUnsupported
	default:
		return deliveryramp.DispositionUnavailable
	}
}

// Proto maps onto the wire enum.
func (d Disposition) Proto() evidencev1.Disposition {
	switch d {
	case DispositionPassed:
		return evidencev1.Disposition_DISPOSITION_PASSED
	case DispositionFailed:
		return evidencev1.Disposition_DISPOSITION_FAILED
	case DispositionSkipped:
		return evidencev1.Disposition_DISPOSITION_SKIPPED
	case DispositionUnsupported:
		return evidencev1.Disposition_DISPOSITION_UNSUPPORTED
	case DispositionUnavailable:
		return evidencev1.Disposition_DISPOSITION_UNAVAILABLE
	case DispositionMissing:
		return evidencev1.Disposition_DISPOSITION_MISSING
	default:
		return evidencev1.Disposition_DISPOSITION_UNSPECIFIED
	}
}

// ParseDisposition accepts the wire spelling; unknown values are refused so
// a consumer never coerces a foreign word into a pass.
func ParseDisposition(value string) (Disposition, bool) {
	d := Disposition(strings.ToLower(strings.TrimSpace(value)))
	if d.Recordable() || d == DispositionMissing {
		return d, true
	}
	return "", false
}
