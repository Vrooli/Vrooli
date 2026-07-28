package flow

import (
	"testing"

	"vrooli-memory/internal/journal/flow/generated"
)

func TestJournalWriteFormalReplay(t *testing.T) {
	generated.RunReplay(t, func(s generated.JournalWriteStatus, e generated.JournalWriteEvent) (generated.JournalWriteStatus, error) {
		next, err := TransitionJournalWrite(JournalWriteState{Status: s}, e)
		return next.Status, err
	})
}
