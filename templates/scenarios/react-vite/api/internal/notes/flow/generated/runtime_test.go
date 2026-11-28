package generated

import "testing"

func TestAttachmentUploadTransitionTable(t *testing.T) {
	tests := []struct {
		name      string
		status    AttachmentUploadStatus
		event     AttachmentUploadEvent
		want      AttachmentUploadStatus
		wantValid bool
	}{
		{"received stores bytes", AttachmentUploadReceived, AttachmentUploadStoreBytes, AttachmentUploadBytesStored, true},
		{"received fails", AttachmentUploadReceived, AttachmentUploadFail, AttachmentUploadFailed, true},
		{"received cannot record metadata", AttachmentUploadReceived, AttachmentUploadRecordMetadata, AttachmentUploadReceived, false},
		{"bytes stored records metadata", AttachmentUploadBytesStored, AttachmentUploadRecordMetadata, AttachmentUploadMetadataRecorded, true},
		{"bytes stored can fail", AttachmentUploadBytesStored, AttachmentUploadFail, AttachmentUploadFailed, true},
		{"metadata recorded is terminal", AttachmentUploadMetadataRecorded, AttachmentUploadStoreBytes, AttachmentUploadMetadataRecorded, false},
		{"failed is terminal", AttachmentUploadFailed, AttachmentUploadRecordMetadata, AttachmentUploadFailed, false},
		{"unknown status is preserved", AttachmentUploadStatus("unknown"), AttachmentUploadStoreBytes, AttachmentUploadStatus("unknown"), false},
		{"unknown event is preserved", AttachmentUploadReceived, AttachmentUploadEvent("unknown"), AttachmentUploadReceived, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := TransitionAttachmentUploadStatus(tt.status, tt.event)
			if got != tt.want {
				t.Errorf("TransitionAttachmentUploadStatus() status = %q, want %q", got, tt.want)
			}
			if valid := err == nil; valid != tt.wantValid {
				t.Errorf("TransitionAttachmentUploadStatus() valid = %t, want %t (err=%v)", valid, tt.wantValid, err)
			}
		})
	}
}

func TestAttachmentUploadEnumerationsAreCompleteAndIsolated(t *testing.T) {
	statuses := AllAttachmentUploadStatuses()
	wantStatuses := []AttachmentUploadStatus{
		AttachmentUploadReceived,
		AttachmentUploadBytesStored,
		AttachmentUploadMetadataRecorded,
		AttachmentUploadFailed,
	}
	if len(statuses) != len(wantStatuses) {
		t.Fatalf("AllAttachmentUploadStatuses() length = %d, want %d", len(statuses), len(wantStatuses))
	}
	for i, want := range wantStatuses {
		if statuses[i] != want {
			t.Errorf("AllAttachmentUploadStatuses()[%d] = %q, want %q", i, statuses[i], want)
		}
	}
	statuses[0] = "mutated"
	if got := AllAttachmentUploadStatuses()[0]; got != AttachmentUploadReceived {
		t.Errorf("AllAttachmentUploadStatuses() exposed mutable backing storage: first status = %q", got)
	}

	events := AllAttachmentUploadEvents()
	wantEvents := []AttachmentUploadEvent{
		AttachmentUploadStoreBytes,
		AttachmentUploadRecordMetadata,
		AttachmentUploadFail,
	}
	if len(events) != len(wantEvents) {
		t.Fatalf("AllAttachmentUploadEvents() length = %d, want %d", len(events), len(wantEvents))
	}
	for i, want := range wantEvents {
		if events[i] != want {
			t.Errorf("AllAttachmentUploadEvents()[%d] = %q, want %q", i, events[i], want)
		}
	}
	events[0] = "mutated"
	if got := AllAttachmentUploadEvents()[0]; got != AttachmentUploadStoreBytes {
		t.Errorf("AllAttachmentUploadEvents() exposed mutable backing storage: first event = %q", got)
	}
}

func TestAttachmentUploadFormalMetadataIsIsolated(t *testing.T) {
	invariants := AttachmentUploadFormalExpectedInvariants()
	if len(invariants) == 0 {
		t.Fatal("AttachmentUploadFormalExpectedInvariants() returned no invariants")
	}
	invariants[0] = "mutated"
	if got := AttachmentUploadFormalExpectedInvariants()[0]; got != "TypeOK" {
		t.Errorf("AttachmentUploadFormalExpectedInvariants() exposed mutable backing storage: first invariant = %q", got)
	}

	checks := AttachmentUploadFormalExpectedGeneratedChecks()
	if len(checks) != 1 || checks[0] != "transitionTable" {
		t.Fatalf("AttachmentUploadFormalExpectedGeneratedChecks() = %v, want [transitionTable]", checks)
	}
	checks[0] = "mutated"
	if got := AttachmentUploadFormalExpectedGeneratedChecks()[0]; got != "transitionTable" {
		t.Errorf("AttachmentUploadFormalExpectedGeneratedChecks() exposed mutable backing storage: first check = %q", got)
	}
}
