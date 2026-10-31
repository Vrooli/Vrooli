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
		Revision                    int64
		ExcludedGroups, Allergies   []string
		Appliances, Cookware, Tools []string
		DeviceFunctions             []string
	}
	Method struct {
		ID, Name, PreparationMode              string
		RequiredAppliances, RequiredCookware   []string
		RequiredTools, RequiredDeviceFunctions []string
	}
	Candidate struct {
		Revision         int64
		Groups           []string
		AllergenEvidence map[string]EvidenceState
		Methods          []Method
	}
	Reason struct {
		Code, Rule, Reference, Message, MethodID string
		ViableMethodIDs                          []string
	}
	MethodDecision struct {
		MethodID, MethodName string
		Status               Status
		Reasons              []Reason
	}
	Decision struct {
		Status                          Status
		Reasons                         []Reason
		Methods                         []MethodDecision
		ViableMethodIDs                 []string
		ProfileRevision, RecipeRevision int64
		EvaluatorVersion                string
	}
)
