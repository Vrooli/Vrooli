package spacedoc

// SchemaVersion is the contract version emitted in SpaceDefinition.SchemaVersion
// and required by .vrooli/schemas/space-definition.schema.json.
const SchemaVersion = "v1"

// Projection identifies one of the three readiness projections.
type Projection string

const (
	ProjectionAnswer   Projection = "answer"
	ProjectionValidate Projection = "validate"
	ProjectionGuide    Projection = "guide"
)

// Valid reports whether p is one of the three known projections.
func (p Projection) Valid() bool {
	switch p {
	case ProjectionAnswer, ProjectionValidate, ProjectionGuide:
		return true
	default:
		return false
	}
}

// DenominatorConfidence is how complete the denominator itself is believed to be.
type DenominatorConfidence string

const (
	ConfidenceAuthoritative DenominatorConfidence = "authoritative"
	ConfidencePartial       DenominatorConfidence = "partial"
	ConfidenceSketch        DenominatorConfidence = "sketch"
)

// CellStatus is the normalized live status of a denominator cell.
type CellStatus string

const (
	StatusNow     CellStatus = "now"
	StatusInReach CellStatus = "in_reach"
	StatusMissing CellStatus = "missing"
)

// Basis is the normalized epistemic-provenance axis (Answer projection only).
type Basis string

const (
	BasisDerived            Basis = "derived"
	BasisValidated          Basis = "validated"
	BasisDeclaredUnverified Basis = "declared_unverified"
	BasisContradicted       Basis = "contradicted"
	BasisAbsent             Basis = "absent"
)

// SpaceDefinition is the parsed denominator, matching space-definition/v1.
type SpaceDefinition struct {
	SchemaVersion         string                `json:"schema_version"`
	Projection            Projection            `json:"projection"`
	Owner                 string                `json:"owner"`
	DenominatorConfidence DenominatorConfidence `json:"denominator_confidence"`
	ConfidenceRationale   string                `json:"confidence_rationale,omitempty"`
	Source                string                `json:"source,omitempty"`
	Cells                 []Cell                `json:"cells"`
}

// Cell is one denominator grid row, normalized across the three space-doc shapes.
type Cell struct {
	ID          string     `json:"id"`
	Group       string     `json:"group,omitempty"`
	Question    string     `json:"question"`
	Owner       string     `json:"owner,omitempty"`
	Status      CellStatus `json:"status"`
	Basis       Basis      `json:"basis,omitempty"`
	Sufficiency string     `json:"sufficiency,omitempty"`
	Notes       []string   `json:"notes,omitempty"`
}
