package agentops

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// TargetKind mirrors the operating-mode target vocabulary (EXECUTION-MODES.md
// D1/D6). It is declared locally so the contracts package does not depend on
// the operating-mode engine; TestTargetVocabularyMatchesOperatingMode pins the
// two enums byte-identical.
type TargetKind string

const (
	TargetBacklogItem   TargetKind = "backlog-item"
	TargetInitiative    TargetKind = "initiative"
	TargetPlanExecution TargetKind = "plan-execution"
	TargetScenario      TargetKind = "scenario"
)

// AllTargetKinds is the canonical ordered target vocabulary.
var AllTargetKinds = []TargetKind{TargetBacklogItem, TargetInitiative, TargetPlanExecution, TargetScenario}

// IsValidTargetKind reports whether kind is a registered target kind.
func IsValidTargetKind(kind TargetKind) bool {
	switch kind {
	case TargetBacklogItem, TargetInitiative, TargetPlanExecution, TargetScenario:
		return true
	default:
		return false
	}
}

// CapabilityID is a closed-vocabulary target-adapter capability. An operation
// contract requires capabilities; a target adapter provides them; compilation
// fails closed on a mismatch.
type CapabilityID string

const (
	CapPlanRef             CapabilityID = "provides-plan-ref"
	CapPlanContext         CapabilityID = "provides-plan-context"
	CapSpecDocument        CapabilityID = "provides-spec-document"
	CapReviewArtifacts     CapabilityID = "provides-review-artifacts"
	CapEvidenceLedger      CapabilityID = "provides-evidence-ledger"
	CapMemberItems         CapabilityID = "provides-member-items"
	CapAcceptanceCriteria  CapabilityID = "provides-acceptance-criteria"
	CapClarificationThread CapabilityID = "provides-clarification-thread"
	CapExecutionWorkspace  CapabilityID = "provides-execution-workspace"
)

// AllCapabilityIDs is the closed capability vocabulary.
var AllCapabilityIDs = []CapabilityID{
	CapPlanRef, CapPlanContext, CapSpecDocument, CapReviewArtifacts,
	CapEvidenceLedger, CapMemberItems, CapAcceptanceCriteria,
	CapClarificationThread, CapExecutionWorkspace,
}

// IsValidCapabilityID reports whether id is a registered capability.
func IsValidCapabilityID(id CapabilityID) bool {
	for _, c := range AllCapabilityIDs {
		if c == id {
			return true
		}
	}
	return false
}

// TargetCapabilityDescriptor is the data shape (target-capability.schema.json)
// declaring which capabilities one target kind provides.
type TargetCapabilityDescriptor struct {
	Kind        string         `json:"kind"`
	TargetKind  TargetKind     `json:"target_kind"`
	Description string         `json:"description,omitempty"`
	Provides    []CapabilityID `json:"provides"`
}

// targetCapabilities is the SSOT of which capabilities each target adapter
// provides. backlog-item and initiative deliberately overlap on
// provides-review-artifacts and provides-evidence-ledger so a review operation
// can be shared across both when its inputs align.
var targetCapabilities = map[TargetKind][]CapabilityID{
	TargetBacklogItem: {
		CapSpecDocument, CapReviewArtifacts, CapEvidenceLedger,
		CapClarificationThread, CapExecutionWorkspace, CapPlanRef,
	},
	TargetInitiative: {
		CapPlanRef, CapPlanContext, CapMemberItems, CapAcceptanceCriteria,
		CapReviewArtifacts, CapEvidenceLedger, CapExecutionWorkspace,
	},
	TargetPlanExecution: {
		CapPlanContext, CapPlanRef, CapExecutionWorkspace,
	},
	// scenario is a plain repository workspace (a scenario directory): it provides
	// a place to run and a spec document to sync, but no plan/review/member state.
	TargetScenario: {
		CapExecutionWorkspace, CapSpecDocument,
	},
}

// TargetCapabilities returns the descriptor documents for every target kind,
// ordered by the canonical target vocabulary. These are the authored data the
// schema validates and the compatibility checker consults.
func TargetCapabilities() []TargetCapabilityDescriptor {
	out := make([]TargetCapabilityDescriptor, 0, len(AllTargetKinds))
	for _, kind := range AllTargetKinds {
		caps := append([]CapabilityID(nil), targetCapabilities[kind]...)
		out = append(out, TargetCapabilityDescriptor{
			Kind:       "agentops-target-capability",
			TargetKind: kind,
			Provides:   caps,
		})
	}
	return out
}

// ProvidedCapabilities returns the set a target kind provides.
func ProvidedCapabilities(kind TargetKind) (map[CapabilityID]bool, bool) {
	caps, ok := targetCapabilities[kind]
	if !ok {
		return nil, false
	}
	set := make(map[CapabilityID]bool, len(caps))
	for _, c := range caps {
		set[c] = true
	}
	return set, true
}

// ValidateTargetCapabilityDescriptor validates a descriptor document against
// the schema and the semantic rules JSON Schema cannot express: the target
// kind is registered, and every provided capability is in the closed
// vocabulary (the enum already blocks unknowns, but a Go-side check keeps the
// error typed and actionable and guards Go-constructed descriptors too).
func ValidateTargetCapabilityDescriptor(raw []byte) error {
	if err := ValidateDocument(SchemaTargetCapability, raw); err != nil {
		return err
	}
	var d TargetCapabilityDescriptor
	if err := json.Unmarshal(raw, &d); err != nil {
		return fmt.Errorf("decode target capability descriptor: %w", err)
	}
	if !IsValidTargetKind(d.TargetKind) {
		return fmt.Errorf("target capability descriptor names unknown target kind %q", d.TargetKind)
	}
	seen := map[CapabilityID]bool{}
	for _, c := range d.Provides {
		if !IsValidCapabilityID(c) {
			return fmt.Errorf("target %q declares unknown capability %q", d.TargetKind, c)
		}
		if seen[c] {
			return fmt.Errorf("target %q declares duplicate capability %q", d.TargetKind, c)
		}
		seen[c] = true
	}
	return nil
}

// CheckOperationTargetCompatibility reports whether a target kind provides
// every capability an operation requires. It fails CLOSED: a missing capability
// or an unknown target kind is a definitive, actionable error naming exactly
// what is missing — never an open-world "maybe". This is the compile-time gate
// that stops an operation from being bound to a target it cannot run on.
func CheckOperationTargetCompatibility(operation OperationID, required []CapabilityID, kind TargetKind) error {
	provided, ok := ProvidedCapabilities(kind)
	if !ok {
		return fmt.Errorf("operation %q cannot bind target kind %q: no capability descriptor is registered for it", operation, kind)
	}
	var missing []string
	for _, need := range required {
		if !IsValidCapabilityID(need) {
			return fmt.Errorf("operation %q requires unknown capability %q", operation, need)
		}
		if !provided[need] {
			missing = append(missing, string(need))
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return fmt.Errorf("operation %q is incompatible with target %q: missing capabilities [%s]", operation, kind, strings.Join(missing, ", "))
	}
	return nil
}

// CompatibleTargets returns the target kinds that can run an operation given
// its required capabilities, in canonical order. Used to prove an operation is
// shareable (e.g. review-round across backlog-item and initiative).
func CompatibleTargets(required []CapabilityID) []TargetKind {
	var out []TargetKind
	for _, kind := range AllTargetKinds {
		if CheckOperationTargetCompatibility("", required, kind) == nil {
			out = append(out, kind)
		}
	}
	return out
}
