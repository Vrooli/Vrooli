package eligibility

type Status string

const (
	Eligible         Status = "eligible"
	Ineligible       Status = "ineligible"
	NeedsInformation Status = "needs_information"
)

type EvidenceState string

const (
	DeclaredPresent EvidenceState = "declared-present"
	DeclaredAbsent  EvidenceState = "declared-absent-within-scope"
	Precautionary   EvidenceState = "precautionary"
	Unknown         EvidenceState = "unknown"
	Conflicting     EvidenceState = "conflicting"
)

type (
	Profile struct {
		Revision                              int64
		ExcludedGroups, Allergies, Appliances []string
	}
	Candidate struct {
		Revision                   int64
		Groups, RequiredAppliances []string
		AllergenEvidence           map[string]EvidenceState
		MethodIDs                  []string
	}
	Reason struct {
		Code, Rule, Reference, Message string
		ViableMethodIDs                []string
	}
	Decision struct {
		Status                          Status
		Reasons                         []Reason
		ProfileRevision, RecipeRevision int64
		EvaluatorVersion                string
	}
)
