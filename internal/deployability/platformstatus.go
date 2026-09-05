package deployability

import "strings"

const (
	platformstatusParameterA = 2
	platformstatusParameterB = 3
	platformstatusParameterC = 4
	platformstatusParameterD = 5
)

// PlatformStatus is the closed vocabulary a manifest may author for a single
// host OS. Every token below is authored somewhere in the repository or is
// synthesized by a loader; nothing else is accepted, because a token the
// resolver does not understand is a declaration nobody has read.
type PlatformStatus string

const (
	// StatusSupported is authored by resource manifests and scenario
	// platform_capabilities blocks, and synthesized for a plain `platforms`
	// list entry.
	StatusSupported PlatformStatus = "supported"
	// StatusBuildVerified is authored by scenario platform_capabilities for
	// code that cross-compiles and passes fixtures but has never run on real
	// hardware of that platform.
	StatusBuildVerified PlatformStatus = "build-verified"
	// StatusExperimental is authored by scenario platform_capabilities for a
	// wired implementation whose behaviour on that platform is unproven.
	StatusExperimental PlatformStatus = "experimental"
	// StatusUnqualified is authored by scenario platform_capabilities and
	// carries the same honesty as experimental: wired, unproven.
	StatusUnqualified PlatformStatus = "unqualified"
	// StatusPartial is authored by resource manifests for an implementation
	// with known functional limits on that platform.
	StatusPartial PlatformStatus = "partial"
	// StatusUnsupported is authored by resource manifests and synthesized by
	// the capability loader for an OS a manifest does not claim.
	StatusUnsupported PlatformStatus = "unsupported"
	// StatusNotImplemented is countable portability debt: the capability may
	// exist on the platform, but no provider is wired yet.
	StatusNotImplemented PlatformStatus = "not_implemented"
	// StatusNotApplicable is closed work: the capability does not exist on the
	// platform or the safeguard's host mechanism cannot apply there.
	StatusNotApplicable PlatformStatus = "not_applicable"
)

// Qualification ranks how much real-world proof a platform declaration
// carries. It is ordered: a caller may ask whether a declaration reaches at
// least a given rung of the ladder.
type Qualification string

const (
	// QualificationUndeclared is the zero rung: nothing is declared, so
	// nothing is proven.
	QualificationUndeclared Qualification = "undeclared"
	// QualificationIneligible means the declaration deliberately says the
	// platform is out of scope.
	QualificationIneligible Qualification = "ineligible"
	// QualificationDegraded means an implementation exists with known limits.
	QualificationDegraded Qualification = "degraded"
	// QualificationUnqualified means an implementation is wired but unproven.
	QualificationUnqualified Qualification = "unqualified"
	// QualificationBuildVerified means the implementation compiles and passes
	// fixtures for the platform but has never run on a real host.
	QualificationBuildVerified Qualification = "build_verified"
	// QualificationQualified is the top rung: proven on real hardware.
	QualificationQualified Qualification = "qualified"
)

var qualificationRanks = map[Qualification]int{
	QualificationUndeclared:    0,
	QualificationIneligible:    1,
	QualificationDegraded:      platformstatusParameterA,
	QualificationUnqualified:   platformstatusParameterB,
	QualificationBuildVerified: platformstatusParameterC,
	QualificationQualified:     platformstatusParameterD,
}

var qualificationReasons = map[Qualification]string{
	QualificationUndeclared:    "nothing is declared about this platform",
	QualificationIneligible:    "declared deliberately absent on this platform",
	QualificationDegraded:      "implemented with known functional limits on this platform",
	QualificationUnqualified:   "an implementation is wired; its quality on this platform is unproven",
	QualificationBuildVerified: "compiles for this platform and passes fixture tests; never run on a real host",
	QualificationQualified:     "runs on real hardware of this platform",
}

// Rank returns the ladder position, ascending with real-world proof.
func (q Qualification) Rank() int { return qualificationRanks[q] }

// Reason returns the one-line human explanation of this rung.
func (q Qualification) Reason() string { return qualificationReasons[q] }

// AtLeast reports whether this rung carries at least as much proof as floor,
// so a caller can ask "is this at least build-verified?".
func (q Qualification) AtLeast(floor Qualification) bool { return q.Rank() >= floor.Rank() }

var platformStatusQualifications = map[PlatformStatus]Qualification{
	StatusSupported:      QualificationQualified,
	StatusBuildVerified:  QualificationBuildVerified,
	StatusExperimental:   QualificationUnqualified,
	StatusUnqualified:    QualificationUnqualified,
	StatusPartial:        QualificationDegraded,
	StatusUnsupported:    QualificationIneligible,
	StatusNotImplemented: QualificationDegraded,
	StatusNotApplicable:  QualificationIneligible,
}

// PlatformStatuses returns the vocabulary in ladder order, most proven first.
func PlatformStatuses() []PlatformStatus {
	return []PlatformStatus{
		StatusSupported,
		StatusBuildVerified,
		StatusExperimental,
		StatusUnqualified,
		StatusPartial,
		StatusUnsupported,
		StatusNotImplemented,
		StatusNotApplicable,
	}
}

// Qualification maps the authored token onto the honesty ladder.
func (s PlatformStatus) Qualification() Qualification {
	if rank, ok := platformStatusQualifications[s]; ok {
		return rank
	}
	return QualificationUndeclared
}

// UnknownPlatformStatusError names the offending token so an operator can find
// and fix the manifest that authored it. An unrecognised token is never
// downgraded to a valid status.
type UnknownPlatformStatusError struct{ Token string }

func (e UnknownPlatformStatusError) Error() string {
	allowed := make([]string, 0, len(platformStatusQualifications))
	for _, status := range PlatformStatuses() {
		allowed = append(allowed, string(status))
	}
	return "platform status " + quote(e.Token) + " is not in the platform status vocabulary (" + strings.Join(allowed, ", ") + ")"
}

// ParsePlatformStatus normalizes an authored token onto the vocabulary. An
// empty or unrecognised token is an error, never a silent downgrade.
func ParsePlatformStatus(raw string) (PlatformStatus, error) {
	token := strings.ToLower(strings.TrimSpace(raw))
	status := PlatformStatus(token)
	if _, ok := platformStatusQualifications[status]; !ok {
		return "", UnknownPlatformStatusError{Token: raw}
	}
	return status, nil
}

func quote(value string) string { return `"` + value + `"` }
