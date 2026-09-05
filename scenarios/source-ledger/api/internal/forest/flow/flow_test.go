package flow

import (
	"testing"

	"source-ledger/internal/forest/flow/generated"
)

func TestForestCompactionFormalReplay(t *testing.T) {
	generated.RunReplay(t, func(s generated.ForestCompactionStatus, e generated.ForestCompactionEvent) (generated.ForestCompactionStatus, error) {
		next, err := TransitionForestCompaction(ForestCompactionState{Status: s}, e)
		return next.Status, err
	})
}
