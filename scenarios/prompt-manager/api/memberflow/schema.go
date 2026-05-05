// Package memberflow defines and validates the per-member message-flow data
// layer (topics.json) that lives at:
//
//	scenarios/prompt-manager/store/teams/<team>/members/<member>/topics.json
//
// The schema declares, structurally, how a single team member produces and
// consumes work via topic-prefixed channels. It is the machine-readable
// counterpart to prose router skills and feeds the heartbeat builder's
// generated Inbox Flow section.
//
// DOC: docs/agent-system/TOPICS_SCHEMA.md, docs/agent-system/drafts/inbox-flow-refactor-plan.md
package memberflow

import (
	"errors"
	"fmt"
	"strings"
)

// DestinationKind enumerates the kinds of sinks an output entry can target.
type DestinationKind string

const (
	DestinationKnowledge     DestinationKind = "knowledge"
	DestinationDecision      DestinationKind = "decision"
	DestinationPORFile       DestinationKind = "por_file"
	DestinationCapabilityGap DestinationKind = "capability_gap"
	DestinationSkillProposal DestinationKind = "skill_proposal"
	DestinationBacklog       DestinationKind = "backlog"
)

func (k DestinationKind) Valid() bool {
	switch k {
	case DestinationKnowledge, DestinationDecision, DestinationPORFile,
		DestinationCapabilityGap, DestinationSkillProposal, DestinationBacklog:
		return true
	default:
		return false
	}
}

// IntakeEntry declares one topic-prefix this member drains.
type IntakeEntry struct {
	// Prefix is a topic-prefix string. Wildcard suffix `/*` indicates a prefix
	// match; a string without `/*` matches only that exact topic.
	Prefix string `json:"prefix"`

	// Taxonomy is the id of the signal taxonomy that owns this prefix. The
	// heartbeat builder resolves it to docs/<domain>/<id>.json (the JSON
	// sidecar) to render the Inbox Flow section. Required after migration.
	// During the inbox-flow refactor transition, an empty Taxonomy surfaces
	// as a `missing_taxonomy` warning rather than a hard error.
	Taxonomy string `json:"taxonomy,omitempty"`

	// ClassifierSkill is the optional, portable judgment skill the member
	// runs on each entry to assign signal_type, evidence_strength, and
	// honesty_flags. Optional: when the prefix carries a deterministic
	// signal_type (e.g., the topic itself names the type), no classifier is
	// needed.
	ClassifierSkill string `json:"classifier_skill,omitempty"`

	// SourceTeam names the team whose member writes this prefix, when the flow
	// is cross-team. Empty / nil means same-team or external producer.
	//
	// The literal "*" (SourceTeamWildcard) means "any team's members may
	// write" — used for universal-source intakes such as bug-inbox where
	// every agent on every team may report. The validator skips
	// orphan_input for "*" entries; a paired wildcard_source_misuse warning
	// fires when external_producers is empty (the producer-side anchor
	// must be documented).
	SourceTeam *string `json:"source_team,omitempty"`
}

// SourceTeamWildcard is the literal value of IntakeEntry.SourceTeam that
// declares a universal-source intake — any team's members may write the
// prefix. See IntakeEntry.SourceTeam for semantics. The same wildcard is
// accepted on RequiredReadEntry.SourceTeam and EvidenceConsumedEntry.SourceTeam.
const SourceTeamWildcard = "*"

// RequiredReadEntry declares one topic-prefix this member must read on every
// heartbeat for required-memory context. The heartbeat builder renders these
// prefixes into the agent prompt's "Required Memory" section so the agent
// always sees current state when it ticks.
//
// Where IntakeEntry says "I drain new entries on this prefix (and may
// classify them)", RequiredReadEntry says "I need this prefix's recent
// entries in my prompt every tick — I'm not draining or routing them." A
// single member commonly carries both: an intake[] for actionable signals
// and a required_read[] for always-on context.
//
// Read relationships used to live on team.json's per-member contract
// (under a member-level field that was invisible to the validator); they
// now live here so a single load surfaces every read relationship to the
// consumer-set used by ruleOrphanOutput (see consumer_set.go).
type RequiredReadEntry struct {
	// Prefix is a topic-prefix string. Wildcard suffix `/*` indicates a
	// prefix match; a string without `/*` matches only that exact topic.
	Prefix string `json:"prefix"`

	// SourceTeam names the team whose member writes this prefix when the
	// flow is cross-team. Empty / nil means same-team or external
	// producer. Mirrors IntakeEntry.SourceTeam semantics, including the
	// "*" (SourceTeamWildcard) value for universal-source reads.
	SourceTeam *string `json:"source_team,omitempty"`

	// Comment is freeform context for human readers (e.g., "needed to cite
	// most recent campaign-draft on every publish proposal"). Optional;
	// the validator does not interpret it.
	Comment string `json:"comment,omitempty"`
}

// EvidenceConsumedEntry declares one topic-prefix this member reads as
// evidence when authoring or contributing to specific decisions. Each entry
// names the decision-context ids that consume it.
//
// Where IntakeEntry says "I drain new entries", EvidenceConsumedEntry says
// "when authoring decision X, I cite entries from this prefix."
// ruleDanglingEvidenceDecision cross-checks each ForDecisions id against
// team.json::decisionContexts so typo'd or removed decision ids surface as
// findings rather than silent dead references.
type EvidenceConsumedEntry struct {
	// Prefix is a topic-prefix string. Wildcard suffix `/*` indicates a
	// prefix match; a string without `/*` matches only that exact topic.
	Prefix string `json:"prefix"`

	// SourceTeam names the team whose member writes this prefix when the
	// flow is cross-team. Empty / nil means same-team or external
	// producer. Mirrors IntakeEntry.SourceTeam semantics.
	SourceTeam *string `json:"source_team,omitempty"`

	// ForDecisions names the decision-context ids that cite this prefix
	// as evidence. Required and non-empty: an evidence relationship with
	// no consumer is not a relationship. ruleDanglingEvidenceDecision
	// validates each id resolves against some team's
	// team.json::decisionContexts.
	ForDecisions []string `json:"for_decisions"`
}

// OutputEntry declares one topic-prefix this member writes.
type OutputEntry struct {
	// Prefix is a topic-prefix string. Wildcard suffix `/*` indicates a prefix
	// match.
	Prefix string `json:"prefix"`

	// DestinationKind names the surface type written. Required.
	DestinationKind DestinationKind `json:"destination_kind"`

	// DestinationTeam names the team that consumes this output, when the flow
	// is cross-team. Empty / nil means same-team or non-team sink (e.g., a
	// PoR file or backlog).
	DestinationTeam *string `json:"destination_team,omitempty"`

	// DestinationPath is the markdown path under docs/ when DestinationKind ==
	// DestinationPORFile. Required for por_file destinations; ignored
	// otherwise. The dangling_por_sink validation rule (Phase 3) checks the
	// file exists.
	DestinationPath *string `json:"destination_path,omitempty"`

	// Schema names the destination front-matter schema declared on the
	// producer's taxonomy (taxonomy.schemas.<schema>). Optional but
	// encouraged for knowledge destinations: it lets the heartbeat builder
	// surface the expected front-matter shape and lets the validator catch
	// drift between topics.json and the taxonomy PoR.
	Schema string `json:"schema,omitempty"`
}

// Topics is the full per-member declaration. Marshalled to/from
// topics.json. All top-level fields are optional; an empty object {} is valid
// and means "this member has no flow declarations."
//
// Field ordering follows the consumption-side → production-side flow:
// intake/required_read/evidence_consumed describe what the member reads;
// output describes what it writes; decisions_owned/decisions_consumed
// describe its decision-graph position; raises_capability_gaps and
// external_producers are policy/anchor metadata.
type Topics struct {
	Intake               []IntakeEntry           `json:"intake,omitempty"`
	RequiredRead         []RequiredReadEntry     `json:"required_read,omitempty"`
	EvidenceConsumed     []EvidenceConsumedEntry `json:"evidence_consumed,omitempty"`
	Output               []OutputEntry           `json:"output,omitempty"`
	DecisionsOwned       []string                `json:"decisions_owned,omitempty"`
	DecisionsConsumed    []string                `json:"decisions_consumed,omitempty"`
	RaisesCapabilityGaps bool                    `json:"raises_capability_gaps,omitempty"`
	ExternalProducers    []string                `json:"external_producers,omitempty"`
}

// IsEmpty reports whether the declaration has no content. An empty Topics is a
// positive declaration ("audited; no flow") rather than ambiguous absence.
func (t Topics) IsEmpty() bool {
	return len(t.Intake) == 0 &&
		len(t.RequiredRead) == 0 &&
		len(t.EvidenceConsumed) == 0 &&
		len(t.Output) == 0 &&
		len(t.DecisionsOwned) == 0 &&
		len(t.DecisionsConsumed) == 0 &&
		!t.RaisesCapabilityGaps &&
		len(t.ExternalProducers) == 0
}

// Validate checks the Topics for shape errors that can be detected without
// cross-member context. Returns the first error encountered, or nil. Callers
// that want the full set of errors should call ValidateAll instead.
//
// Shape errors include:
//   - empty intake.prefix or malformed prefix
//   - empty output.prefix or unknown destination_kind
//   - destination_kind == por_file with empty destination_path
//
// Cross-member errors (orphan_input, conflicting_drain, unknown_taxonomy,
// missing_destination_schema, prose_topic_leak, etc.) require the full
// graph and are computed by Validate (validation.go), not by this method.
func (t Topics) Validate() error {
	errs := t.ValidateAll()
	if len(errs) == 0 {
		return nil
	}
	return errs[0]
}

// ValidateAll returns all shape errors detected. Used by tests; production
// callers typically want Validate() (first-error) for fast-fail.
func (t Topics) ValidateAll() []error {
	var errs []error

	for i, e := range t.Intake {
		if err := validateIntake(e); err != nil {
			errs = append(errs, fmt.Errorf("intake[%d]: %w", i, err))
		}
	}
	for i, e := range t.RequiredRead {
		if err := validateRequiredRead(e); err != nil {
			errs = append(errs, fmt.Errorf("required_read[%d]: %w", i, err))
		}
	}
	for i, e := range t.EvidenceConsumed {
		if err := validateEvidenceConsumed(e); err != nil {
			errs = append(errs, fmt.Errorf("evidence_consumed[%d]: %w", i, err))
		}
	}
	for i, e := range t.Output {
		if err := validateOutput(e); err != nil {
			errs = append(errs, fmt.Errorf("output[%d]: %w", i, err))
		}
	}
	return errs
}

func validateIntake(e IntakeEntry) error {
	if strings.TrimSpace(e.Prefix) == "" {
		return errors.New("prefix is required")
	}
	if !validPrefix(e.Prefix) {
		return fmt.Errorf("prefix %q is malformed", e.Prefix)
	}
	return nil
}

func validateRequiredRead(e RequiredReadEntry) error {
	if strings.TrimSpace(e.Prefix) == "" {
		return errors.New("prefix is required")
	}
	if !validPrefix(e.Prefix) {
		return fmt.Errorf("prefix %q is malformed", e.Prefix)
	}
	return nil
}

func validateEvidenceConsumed(e EvidenceConsumedEntry) error {
	if strings.TrimSpace(e.Prefix) == "" {
		return errors.New("prefix is required")
	}
	if !validPrefix(e.Prefix) {
		return fmt.Errorf("prefix %q is malformed", e.Prefix)
	}
	if len(e.ForDecisions) == 0 {
		return errors.New("for_decisions is required (non-empty list of decision-context ids)")
	}
	for i, id := range e.ForDecisions {
		if strings.TrimSpace(id) == "" {
			return fmt.Errorf("for_decisions[%d] is empty", i)
		}
	}
	return nil
}

func validateOutput(e OutputEntry) error {
	if strings.TrimSpace(e.Prefix) == "" {
		return errors.New("prefix is required")
	}
	if !validPrefix(e.Prefix) {
		return fmt.Errorf("prefix %q is malformed", e.Prefix)
	}
	if !e.DestinationKind.Valid() {
		return fmt.Errorf("destination_kind %q is not one of: knowledge, decision, por_file, capability_gap, skill_proposal, backlog", e.DestinationKind)
	}
	if e.DestinationKind == DestinationPORFile {
		if e.DestinationPath == nil || strings.TrimSpace(*e.DestinationPath) == "" {
			return errors.New("destination_path is required when destination_kind is por_file")
		}
	}
	return nil
}

// validPrefix enforces basic shape rules:
//   - non-empty
//   - no whitespace
//   - bare "*" is disallowed (would defeat overlap detection)
//   - "/*" suffix is the only allowed wildcard
//   - inner "*" (e.g. "foo/*/bar") is disallowed
func validPrefix(p string) bool {
	if p == "" || p == "*" {
		return false
	}
	if strings.ContainsAny(p, " \t\n") {
		return false
	}
	// strip a trailing /* if present
	core := strings.TrimSuffix(p, "/*")
	// the remainder must contain no '*' at all
	if strings.Contains(core, "*") {
		return false
	}
	return true
}

// Overlap reports whether two prefixes share at least one matchable topic.
//
// Semantics (matches docs/agent-system/TOPICS_SCHEMA.md):
//   - Equal prefixes always overlap.
//   - "foo/*" overlaps "foo/bar" (the wildcard is wider).
//   - "foo/bar/*" overlaps "foo/*" (the wildcard is wider).
//   - "foo/bar/*" does not overlap "foo/baz/*".
//   - Exact prefixes (no /*) overlap only with equal exact prefixes or with
//     wildcards whose stripped form is a prefix.
func Overlap(a, b string) bool {
	aWild := strings.HasSuffix(a, "/*")
	bWild := strings.HasSuffix(b, "/*")
	aCore := strings.TrimSuffix(a, "/*")
	bCore := strings.TrimSuffix(b, "/*")

	switch {
	case aCore == bCore:
		return true
	case aWild && (strings.HasPrefix(bCore, aCore+"/") || bCore == aCore):
		return true
	case bWild && (strings.HasPrefix(aCore, bCore+"/") || aCore == bCore):
		return true
	default:
		return false
	}
}
