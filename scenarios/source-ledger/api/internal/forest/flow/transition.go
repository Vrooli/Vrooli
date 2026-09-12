package flow

import (
	"fmt"

	"source-ledger/internal/forest/flow/generated"
)

type (
	ForestCompactionStatus = generated.ForestCompactionStatus
	ForestCompactionEvent  = generated.ForestCompactionEvent
)

const (
	ForestCompactionIdle        = generated.ForestCompactionIdle
	ForestCompactionScoring     = generated.ForestCompactionScoring
	ForestCompactionSummarizing = generated.ForestCompactionSummarizing
	ForestCompactionWritten     = generated.ForestCompactionWritten
)

type ForestCompactionState struct{ Status ForestCompactionStatus }

func InitialForestCompactionState() ForestCompactionState {
	return ForestCompactionState{Status: ForestCompactionIdle}
}

func TransitionForestCompaction(state ForestCompactionState, event ForestCompactionEvent) (ForestCompactionState, error) {
	if err := CheckForestCompactionInvariants(state); err != nil {
		return state, err
	}
	next, err := generated.TransitionForestCompactionStatus(state.Status, event)
	return ForestCompactionState{Status: next}, err
}

func CheckForestCompactionInvariants(state ForestCompactionState) error {
	switch state.Status {
	case ForestCompactionIdle, ForestCompactionScoring, ForestCompactionSummarizing, ForestCompactionWritten:
		return nil
	default:
		return fmt.Errorf("unknown forest compaction status %q", state.Status)
	}
}
