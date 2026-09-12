// Package finding is the single shape every validation result takes.
//
// Six shapes existed before this package: memberflow.Finding,
// memberflow.OperatingGraphFinding, teamcontract.ValidationFinding,
// heartbeat.ContractFinding, teamconfig.ValidationFinding, and
// store.TeamValidationFinding. Conversions between them lost fields in both
// directions, and teamcontract.ValidationFinding carried only Field and
// Message — no rule id and no severity — so a malformed team.json could never
// be catalogued, ranked, or rendered into a member's `# Contract Findings`.
//
// This package is a leaf on purpose. teamcontract imports no internal package,
// store imports teamcontract, and memberflow imports both, so a shared finding
// type cannot live in memberflow without inverting that order. The name
// `validation` was unavailable: prompt-manager/validation already holds
// IsValidHexColor and Slugify and is unrelated to rule validation.
package finding

// Severity ranks a finding. An error blocks a gating command; a warning is
// reported and does not.
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
)

// Kind separates what a finding was computed from. A declaration finding reads
// only checked-in files, so it can gate CI. A runtime finding reads live agent
// behavior recorded in the live team corpus, so it cannot: the tree can be correct
// while agents are still misbehaving. Sharing one exit code between them is why
// `graph topics` could not gate anything and why actual_writer_undeclared grew
// from 9 findings to 43 unchecked.
type Kind string

const (
	KindDeclaration Kind = "declaration"
	KindRuntime     Kind = "runtime"
)

// Finding is one validation result.
//
// The subject fields are flat rather than nested. Every rule addresses a
// different level — a team's graph, one member's topic declaration, a line in a
// plan-of-record document — and a nested subject would have forced every one of
// the several hundred construction sites through a wrapper that adds no
// meaning. A finding that names no subject at all cannot be acted on and should
// not exist.
type Finding struct {
	Rule     string   `json:"rule"`
	Severity Severity `json:"severity"`
	Kind     Kind     `json:"kind,omitempty"`

	Team     string `json:"team,omitempty"`
	Member   string `json:"member,omitempty"`
	Topic    string `json:"topic,omitempty"`
	Prefix   string `json:"prefix,omitempty"`
	Path     string `json:"path,omitempty"`
	Line     int    `json:"line,omitempty"`
	GraphID  string `json:"graph_id,omitempty"`
	NodeID   string `json:"node_id,omitempty"`
	Edge     string `json:"edge,omitempty"`
	OwnerKey string `json:"owner_key,omitempty"`

	// SourcePath is the file the finding was read out of, which is not always
	// the file an operator must edit to fix it (Path).
	SourcePath string `json:"source_path,omitempty"`

	Detail string `json:"detail"`

	// Advisory marks a finding produced by a heuristic that cannot fully
	// distinguish a real defect from a lookalike. Advisory findings are review
	// material for an operator sweep, not instructions: they are reported by
	// `graph topics` and withheld from surfaces that ask an agent to act.
	Advisory bool `json:"advisory,omitempty"`
}

// Subject renders the member identity as `team/member`, the form the topic
// family reports and groups by.
func (f Finding) Subject() string {
	if f.Team == "" {
		return f.Member
	}
	return f.Team + "/" + f.Member
}
