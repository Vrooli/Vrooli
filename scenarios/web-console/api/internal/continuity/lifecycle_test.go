package continuity

import "testing"

func TestTransitionTable(t *testing.T) {
	tests := []struct {
		from  State
		event Event
		want  State
	}{
		{StateProvisioning, EventProcessStarted, StateLive},
		{StateProvisioning, EventProcessStartFailedHistory, StateRecoverable},
		{StateLive, EventArchiveRequested, StateArchivePending},
		{StateLive, EventProcessExited, StateExited},
		{StateArchivePending, EventArchiveUndone, StateLive},
		{StateArchivePending, EventGraceElapsed, StateArchived},
		{StateExited, EventResumableSourceFound, StateRecoverable},
		{StateExited, EventRecoverySucceeded, StateLive},
		{StateExited, EventArchiveRequested, StateArchived},
		{StateRecoverable, EventRecoverySucceeded, StateLive},
		{StateRecoverable, EventDismissedToArchive, StateArchived},
		{StateArchived, EventReopenSucceeded, StateLive},
		{StateArchived, EventPermanentDeleteConfirmed, StateDeletePending},
		{StateDeletePending, EventArtifactsPrunedReceipt, StateDeleted},
		{StateDeletePending, EventDeletionFailedPreserved, StateArchived},
	}
	for _, tt := range tests {
		got, err := Transition(tt.from, tt.event)
		if err != nil || got != tt.want {
			t.Errorf("Transition(%q, %q) = %q, %v; want %q", tt.from, tt.event, got, err, tt.want)
		}
	}
}

func TestTransitionRejectsDestructiveShortcuts(t *testing.T) {
	for _, tt := range []struct {
		from  State
		event Event
	}{
		{StateLive, EventArtifactsPrunedReceipt},
		{StateArchived, EventArtifactsPrunedReceipt},
		{StateDeleted, EventReopenSucceeded},
		{StateDeletePending, EventProcessStarted},
		{StateLive, EventResumableSourceFound},
	} {
		if _, err := Transition(tt.from, tt.event); err == nil {
			t.Errorf("Transition(%q, %q) unexpectedly succeeded", tt.from, tt.event)
		}
	}
}

func TestTransitionMatrixIsExhaustive(t *testing.T) {
	legal := map[struct {
		from  State
		event Event
	}]State{}
	for _, tt := range []struct {
		from  State
		event Event
		to    State
	}{
		{StateProvisioning, EventProcessStarted, StateLive},
		{StateProvisioning, EventProcessStartFailedHistory, StateRecoverable},
		{StateLive, EventArchiveRequested, StateArchivePending},
		{StateLive, EventProcessExited, StateExited},
		{StateArchivePending, EventArchiveUndone, StateLive},
		{StateArchivePending, EventGraceElapsed, StateArchived},
		{StateExited, EventResumableSourceFound, StateRecoverable},
		{StateExited, EventRecoverySucceeded, StateLive},
		{StateExited, EventArchiveRequested, StateArchived},
		{StateRecoverable, EventRecoverySucceeded, StateLive},
		{StateRecoverable, EventDismissedToArchive, StateArchived},
		{StateArchived, EventReopenSucceeded, StateLive},
		{StateArchived, EventPermanentDeleteConfirmed, StateDeletePending},
		{StateDeletePending, EventArtifactsPrunedReceipt, StateDeleted},
		{StateDeletePending, EventDeletionFailedPreserved, StateArchived},
	} {
		legal[struct {
			from  State
			event Event
		}{tt.from, tt.event}] = tt.to
	}

	for _, from := range States() {
		for _, event := range Events() {
			want, isLegal := legal[struct {
				from  State
				event Event
			}{from, event}]
			got, err := Transition(from, event)
			if isLegal {
				if err != nil || got != want {
					t.Errorf("legal transition %s --%s--> %s, got %s, %v", from, event, want, got, err)
				}
				if err := ValidateTransition(from, event, want); err != nil {
					t.Errorf("ValidateTransition(%s, %s, %s): %v", from, event, want, err)
				}
				continue
			}
			if err == nil {
				t.Errorf("illegal transition %s --%s--> %s", from, event, got)
			}
		}
	}
}

func TestValidateStateRejectsUnknownValues(t *testing.T) {
	if err := ValidateState(State("future-state")); err == nil {
		t.Fatal("unknown lifecycle state was accepted")
	}
}
