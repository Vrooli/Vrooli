package deliveryramp

import "strings"

// DeliveryFormat is orthogonal to target OS, runtime tier, and host provider.
// A source repository is a delivery representation, not an execution target.
type DeliveryFormat string

const DeliveryFormatSourceRepository DeliveryFormat = "source_repository"

type SourceGate string

const (
	SourceGatePinned        SourceGate = "source_pinned"
	SourceGateClosure       SourceGate = "closure_complete"
	SourceGatePolicy        SourceGate = "policy_passed"
	SourceGateIntegrity     SourceGate = "integrity_verified"
	SourceGateReproducible  SourceGate = "reproducible"
	SourceGateCleanBuild    SourceGate = "clean_build_verified"
	SourceGateDocumentation SourceGate = "documentation_verified"
	SourceGateGovernance    SourceGate = "governance_approved"
)

type SourceEvidence struct {
	CandidateDigest string
	SourceDigest    string
	ClosureDigest   string
	PolicyDigest    string
	ArchiveDigest   string
	Reproducible    bool
	CleanBuild      GateDisposition
	Documentation   GateDisposition
	Governance      GateDisposition
	Gates           map[SourceGate]GateDisposition
}

type SourceProfileVerdict struct {
	Format      DeliveryFormat
	Disposition GateDisposition
	Required    []SourceGate
	Passed      []SourceGate
	Failures    []string
}

func SourceEvidenceProfile() []SourceGate {
	return []SourceGate{SourceGatePinned, SourceGateClosure, SourceGatePolicy, SourceGateIntegrity, SourceGateReproducible, SourceGateCleanBuild, SourceGateDocumentation, SourceGateGovernance}
}

// EvaluateSourceEvidence is fail-closed: missing, stale, or unrelated
// evidence cannot be treated as a successful source delivery profile.
func EvaluateSourceEvidence(e SourceEvidence) SourceProfileVerdict {
	v := SourceProfileVerdict{Format: DeliveryFormatSourceRepository, Required: SourceEvidenceProfile(), Disposition: GateFailed}
	if strings.TrimSpace(e.CandidateDigest) == "" {
		v.Failures = append(v.Failures, "candidate digest is missing")
	}
	if strings.TrimSpace(e.SourceDigest) == "" {
		v.Failures = append(v.Failures, "source digest is missing")
	}
	if strings.TrimSpace(e.ClosureDigest) == "" {
		v.Failures = append(v.Failures, "closure digest is missing")
	}
	if strings.TrimSpace(e.PolicyDigest) == "" {
		v.Failures = append(v.Failures, "policy digest is missing")
	}
	if strings.TrimSpace(e.ArchiveDigest) == "" {
		v.Failures = append(v.Failures, "archive digest is missing")
	}
	if e.CandidateDigest != "" && e.ArchiveDigest != "" && e.CandidateDigest != e.ArchiveDigest {
		v.Failures = append(v.Failures, "candidate digest does not match archive digest")
	}
	for _, gate := range v.Required {
		if gate == SourceGateReproducible {
			if e.Reproducible {
				v.Passed = append(v.Passed, gate)
			} else {
				v.Failures = append(v.Failures, "reproducibility evidence is missing or failed")
			}
			continue
		}
		disposition, ok := e.Gates[gate]
		if gate == SourceGateCleanBuild {
			disposition, ok = e.CleanBuild, true
		}
		if gate == SourceGateDocumentation {
			disposition, ok = e.Documentation, true
		}
		if gate == SourceGateGovernance {
			disposition, ok = e.Governance, true
		}
		if !ok || disposition != GatePassed {
			v.Failures = append(v.Failures, string(gate)+" is not passed")
		} else {
			v.Passed = append(v.Passed, gate)
		}
	}
	if len(v.Failures) == 0 {
		v.Disposition = GatePassed
	}
	return v
}
