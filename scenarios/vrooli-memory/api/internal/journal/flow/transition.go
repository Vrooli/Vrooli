package flow

import (
	"fmt"

	"vrooli-memory/internal/journal/flow/generated"
)

type (
	JournalWriteStatus = generated.JournalWriteStatus
	JournalWriteEvent  = generated.JournalWriteEvent
)

const (
	JournalWriteReceived     = generated.JournalWriteReceived
	JournalWriteClassified   = generated.JournalWriteClassified
	JournalWriteEmbedded     = generated.JournalWriteEmbedded
	JournalWriteUnclassified = generated.JournalWriteUnclassified
	JournalWriteAppended     = generated.JournalWriteAppended
	JournalWriteQueued       = generated.JournalWriteQueued
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
	case JournalWriteReceived, JournalWriteClassified, JournalWriteEmbedded, JournalWriteUnclassified, JournalWriteAppended, JournalWriteQueued:
		return nil
	default:
		return fmt.Errorf("unknown journal write status %q", state.Status)
	}
}
