package teamconfig

import "strings"

// Purpose and lifetime describe a team; they grant no authority and do not
// change scheduling or execution. An empty value means unspecified.
const (
	PurposeDomainStewardship = "domain-stewardship"
	PurposeDelivery          = "delivery"
	PurposeSupervision       = "supervision"
	LifetimeStanding         = "standing"
	LifetimeFinite           = "finite"
)

// ObjectiveDeclaration is the `objectivesServed` block on a team.json.
//
// It is an authored import and declaration input, not a current-state
// authority. The objective authority (internal/objectives) owns current
// objective state; this block only carries the team's declaration and its
// acknowledged revision for migration and drift comparison. Do not read it
// as the source of current coverage.
type ObjectiveDeclaration struct {
	ID       string `json:"id"`
	Role     string `json:"role,omitempty"`
	Coverage string `json:"coverage,omitempty"`
	// Note carries any qualifier the team wants recorded. It is never
	// validated; it exists so a team can say *why* its coverage is partial
	// without that reason having to live in prose the validator cannot see.
	Note string `json:"note,omitempty"`
	// AcknowledgedRevision is the objective revision this team last confirmed
	// its obligation list follows from. When it differs from the objective's
	// current revision — including when it is absent, which is the state every
	// team starts in — the team carries `objective_restatement_pending` until
	// its contrarian re-derives the obligations and records the new value.
	//
	// It is the actuator for the slowest loop in the target model: intent
	// changed, so the setpoint derived from it has to be re-derived. Nothing
	// else in the system fires on that event.
	AcknowledgedRevision string `json:"acknowledgedRevision,omitempty"`
}

// MetadataFindings validates independent descriptive fields without coupling
// team lifetime or purpose to runtime or authority.
func MetadataFindings(purpose, lifetime string, effortRefs []string) []ValidationFinding {
	var findings []ValidationFinding
	switch purpose {
	case "", PurposeDomainStewardship, PurposeDelivery, PurposeSupervision:
	default:
		findings = append(findings, ValidationFinding{Field: "purpose", Message: "purpose must be 'domain-stewardship', 'delivery', 'supervision', or empty"})
	}
	switch lifetime {
	case "", LifetimeStanding, LifetimeFinite:
	default:
		findings = append(findings, ValidationFinding{Field: "lifetime", Message: "lifetime must be 'standing', 'finite', or empty"})
	}
	seen := make(map[string]bool, len(effortRefs))
	for _, ref := range effortRefs {
		if strings.TrimSpace(ref) == "" || strings.TrimSpace(ref) != ref || strings.ContainsAny(ref, "\r\n\t") {
			findings = append(findings, ValidationFinding{Field: "effortRefs", Message: "effortRefs must contain nonempty references without surrounding whitespace or control characters"})
		} else if seen[ref] {
			findings = append(findings, ValidationFinding{Field: "effortRefs", Message: "effortRefs must not contain duplicate references"})
		}
		seen[ref] = true
	}
	return findings
}

// ValidateMetadata is shared by API and CLI mutation boundaries.
func ValidateMetadata(purpose, lifetime string, effortRefs []string) error {
	if findings := MetadataFindings(purpose, lifetime, effortRefs); len(findings) > 0 {
		return validationError(findings[0].Message)
	}
	return nil
}
