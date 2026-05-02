// Package memberflow defines and validates the per-member message-flow data
// layer (topics.json) that lives at:
//
//	scenarios/prompt-manager/store/teams/<team>/members/<member>/topics.json
//
// The schema declares, structurally, how a single team member produces and
// consumes work via topic-prefixed channels. It is the machine-readable
// counterpart to prose router skills.
//
// Phase 0 of the agent-system migration ships only the type definitions and a
// schema validator (no loader, no endpoints). Phase 2 adds the loader, API
// endpoints, and graph computation; Phase 3 layers the validation rules on top.
//
// DOC: docs/agent-system/drafts/topics-schema.md
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

	// DrainedBySkill is the skill ID that owns the drain procedure. Required.
	// The missing_drain_skill validation rule (Phase 3) cross-checks against
	// the skill registry.
	DrainedBySkill string `json:"drained_by_skill"`

	// SourceTeam names the team whose member writes this prefix, when the flow
	// is cross-team. Empty / nil means same-team or external producer.
	SourceTeam *string `json:"source_team,omitempty"`
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
}

// Topics is the full per-member declaration. Marshalled to/from
// topics.json. All top-level fields are optional; an empty object {} is valid
// and means "this member has no flow declarations."
type Topics struct {
	Intake               []IntakeEntry `json:"intake,omitempty"`
	Output               []OutputEntry `json:"output,omitempty"`
	DecisionsOwned       []string      `json:"decisions_owned,omitempty"`
	DecisionsConsumed    []string      `json:"decisions_consumed,omitempty"`
	RaisesCapabilityGaps bool          `json:"raises_capability_gaps,omitempty"`
	ExternalProducers    []string      `json:"external_producers,omitempty"`
}

// IsEmpty reports whether the declaration has no content. An empty Topics is a
// positive declaration ("audited; no flow") rather than ambiguous absence.
func (t Topics) IsEmpty() bool {
	return len(t.Intake) == 0 &&
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
//   - empty intake.prefix
//   - empty intake.drained_by_skill
//   - invalid prefix syntax (e.g., bare "*")
//   - empty output.prefix
//   - unknown output.destination_kind
//   - destination_kind == por_file with empty destination_path
//   - destination_path set with destination_kind != por_file (warning-tier; allowed but unused)
//
// Cross-member errors (orphan_input, conflicting_drain, etc.) require the full
// graph and are computed by Phase 3's validation package, not by Validate.
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
	if strings.TrimSpace(e.DrainedBySkill) == "" {
		return errors.New("drained_by_skill is required")
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
// Semantics (matches docs/agent-system/drafts/topics-schema.md):
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
