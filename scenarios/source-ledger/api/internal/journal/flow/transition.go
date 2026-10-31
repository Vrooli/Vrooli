package flow

import (
	"fmt"

	"source-ledger/internal/journal/flow/generated"
)

type (
	JournalWriteStatus = generated.JournalWriteStatus
	JournalWriteEvent  = generated.JournalWriteEvent
)

const (
	JournalWriteReceived = generated.JournalWriteReceived
	JournalWriteAppended = generated.JournalWriteAppended
)

type JournalWriteState struct{ Status JournalWriteStatus }

func InitialJournalWriteState() JournalWriteState {
	return JournalWriteState{Status: JournalWriteReceived}
}

func TransitionJournalWrite(state JournalWriteState, event JournalWriteEvent) (JournalWriteState, error) {
	if err := CheckJournalWriteInvariants(state); err != nil {
		return state, err
	}
	next, err := generated.TransitionJournalWriteStatus(state.Status, event)
	return JournalWriteState{Status: next}, err
}

func CheckJournalWriteInvariants(state JournalWriteState) error {
	switch state.Status {
	case JournalWriteReceived, JournalWriteAppended:
		return nil
	default:
		return fmt.Errorf("unknown journal write status %q", state.Status)
	}
}
