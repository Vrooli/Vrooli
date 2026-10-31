// Package objectives is the canonical owner of objective definitions, their
// attachment to teams, ordering and acknowledgements.
//
// The operator statement stays prose in docs/director-swarm/strategy/OBJECTIVES.md;
// this package owns identity, class, meaning revision, attachments, ordering and
// acknowledgements. It never writes the statement.
package objectives

// Class is an objective's position in the operator's ends/means model.
type Class string

const (
	// ClassTerminal names an end the operator seeks.
	ClassTerminal Class = "terminal"
	// ClassInstrumental names a means toward a terminal end.
	ClassInstrumental Class = "instrumental"
)

// Team roles and coverage qualifiers. An empty role is a legitimate value: the
// source did not distinguish one, so asserting a role would invent intent.
const (
	RolePrimary     = "primary"
	RoleSupporting  = "supporting"
	CoverageFull    = "full"
	CoveragePartial = "partial"
)

// Objective is one operator objective with its persisted ordering and the
// revision of its current meaning.
type Objective struct {
	ID              string `json:"id"`
	Title           string `json:"title"`
	Class           Class  `json:"class"`
	EvidenceSource  string `json:"evidenceSource,omitempty"`
	HasEvidence     bool   `json:"hasEvidence"`
	GapMarker       string `json:"gapMarker,omitempty"`
	GlobalOrder     int    `json:"globalOrder"`
	MeaningRevision string `json:"meaningRevision"`
}

// Attachment links one team to one objective. Priority is the team's own
// ordering; AcknowledgedRevision records the objective meaning revision the
// team has confirmed. AttachmentRevision is the team-wide revision of its links
// and order, supplied by the service at read time rather than stored per row.
type Attachment struct {
	ObjectiveID          string `json:"objectiveId"`
	TeamID               string `json:"teamId"`
	Role                 string `json:"role,omitempty"`
	Coverage             string `json:"coverage,omitempty"`
	Note                 string `json:"note,omitempty"`
	Priority             int    `json:"priority"`
	AcknowledgedRevision string `json:"acknowledgedRevision,omitempty"`
	AttachmentRevision   string `json:"attachmentRevision,omitempty"`
	RestatementPending   bool   `json:"restatementPending"`
}

// Relation is a directed support edge from an instrumental objective toward
// another objective. The graph must stay acyclic.
type Relation struct {
	FromObjectiveID string `json:"fromObjectiveId"`
	ToObjectiveID   string `json:"toObjectiveId"`
}

// Finding is one domain validation observation.
type Finding struct {
	Rule        string `json:"rule"`
	Severity    string `json:"severity"`
	ObjectiveID string `json:"objectiveId,omitempty"`
	TeamID      string `json:"teamId,omitempty"`
	Detail      string `json:"detail"`
}

// ValidationResult is the tally of domain findings for the current state.
type ValidationResult struct {
	Findings []Finding `json:"findings"`
	Errors   int       `json:"errors"`
	Warnings int       `json:"warnings"`
}
