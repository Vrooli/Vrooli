package intent

// Altitude identifies the intent ladder rung a claim came from.
type Altitude string

const (
	Outcome     Altitude = "outcome"
	Requirement Altitude = "requirement"
	Domain      Altitude = "domain"
	Code        Altitude = "code"
)

// RefKind classifies a normalized validation reference.
type RefKind string

const (
	RefCode   RefKind = "code"
	RefDoc    RefKind = "doc"
	RefManual RefKind = "manual"
)

// Ref is the normalized form of one requirement validation reference.
type Ref struct {
	Raw    string
	Path   string
	Member string
	Kind   RefKind
	Glob   bool
}

// CapabilityClaim is the single normalized unit shared by intent extractors
// and detectors.
type CapabilityClaim struct {
	ID         string
	Altitude   Altitude
	Text       string
	Anchor     string
	Refs       []Ref
	Provenance string
}

// Finding is a neutral intent finding. Producers map it into their native
// finding envelope at the edge.
type Finding struct {
	Code       string
	Severity   string
	Message    string
	Suggestion string
	Locations  []string
	ClaimID    string
	RelatedID  string
	Provenance string
}

const (
	CodePRDRefUnmatched   = "intent.prd_ref_unmatched"
	CodeOTOrphan          = "intent.ot_orphan"
	CodeRefMissing        = "intent.ref_missing"
	CodeReqUnownedDomain  = "intent.req_unowned_domain"
	CodeReqTransportOwned = "intent.req_transport_owned"
	CodeDomainUnrequired  = "intent.domain_unrequired"
	CodeOTNoDomain        = "intent.ot_no_domain"
	CodeVocabDrift        = "intent.vocab_drift"
	CodeSemanticGap       = "intent.semantic_coverage_gap"
	CodeResponsibility    = "intent.responsibility_mismatch"
)

// PRD template-contract codes. These are NOT part of the intent.* alignment
// invariant registry (docs/reference/intent-alignment.md) — they are the
// document-shape vocabulary absorbed from the retired PRD control-tower's template
// engine, emitted through business-health's `business` dimension. They live
// here because intent-go is the single parser of PRD.md and the checks are
// pure functions over its extraction.
const (
	CodeTemplateSections           = "prd_template_sections"
	CodeTemplateUnexpectedSections = "prd_template_unexpected_sections"
	CodeTemplateContent            = "prd_template_content"
	CodeOTIDFormat                 = "prd_ot_id_format"
)
