package agentops

import (
	"encoding/json"
	"fmt"
)

// MemberItemStrategyName is how an initiative schedules work across its member
// items. This is domain workflow STRATEGY configuration — the replacement for
// the phase-less item-level operating mode (EXECUTION-MODES.md D7), not an
// operating mode and not a target kind.
type MemberItemStrategyName string

const (
	StrategyParallelItems    MemberItemStrategyName = "parallel-items"
	StrategySequentialItems  MemberItemStrategyName = "sequential-items"
	StrategyPrioritizedItems MemberItemStrategyName = "prioritized-items"
)

// AllMemberItemStrategies is the canonical ordered strategy vocabulary.
var AllMemberItemStrategies = []MemberItemStrategyName{
	StrategyParallelItems, StrategySequentialItems, StrategyPrioritizedItems,
}

func isValidStrategy(name MemberItemStrategyName) bool {
	for _, s := range AllMemberItemStrategies {
		if s == name {
			return true
		}
	}
	return false
}

// MemberItemStrategy is the typed shape of member-item-strategy.schema.json.
type MemberItemStrategy struct {
	Kind               string                 `json:"kind"`
	Strategy           MemberItemStrategyName `json:"strategy"`
	ItemOperation      OperationID            `json:"item_operation"`
	MaxConcurrency     int                    `json:"max_concurrency,omitempty"`
	StopOnFirstFailure bool                   `json:"stop_on_first_failure,omitempty"`
	ItemSelection      string                 `json:"item_selection,omitempty"`
}

// DefaultMemberItemStrategy is the explicit domain-workflow configuration that
// replaces the legacy item-level default: advance ready member items in
// parallel, running each through the execution-run operation.
func DefaultMemberItemStrategy() MemberItemStrategy {
	return MemberItemStrategy{
		Kind:          "agentops-member-item-strategy",
		Strategy:      StrategyParallelItems,
		ItemOperation: OpExecutionRun,
		ItemSelection: "ready-only",
	}
}

// ValidateMemberItemStrategy validates a strategy document against the schema
// and the semantic rules JSON Schema cannot express: the strategy name is
// known and the per-item operation is a registered operation contract.
func ValidateMemberItemStrategy(raw []byte) error {
	if err := ValidateDocument(SchemaMemberItemStrategy, raw); err != nil {
		return err
	}
	var s MemberItemStrategy
	if err := json.Unmarshal(raw, &s); err != nil {
		return fmt.Errorf("decode member-item strategy: %w", err)
	}
	if !isValidStrategy(s.Strategy) {
		return fmt.Errorf("member-item strategy %q is unknown", s.Strategy)
	}
	if !IsValidOperationID(s.ItemOperation) {
		return fmt.Errorf("member-item strategy names unregistered item operation %q", s.ItemOperation)
	}
	return nil
}
