package notes_test

import (
	"testing"
	"{{SCENARIO_ID}}/internal/notes"
	"{{SCENARIO_ID}}/internal/testutil/modeltest"

	"github.com/stretchr/testify/require"
)

func TestAttachmentUploadWorkflow_TransitionMatrix(t *testing.T) {
	modeltest.AssertTransitionMatrix(
		t,
		notes.AllAttachmentUploadStatuses(),
		notes.AllAttachmentUploadEvents(),
		attachmentUploadMatrix(),
		func(status notes.AttachmentUploadStatus, event notes.AttachmentUploadEvent) (notes.AttachmentUploadStatus, error) {
			next, err := notes.TransitionAttachmentUpload(notes.AttachmentUploadState{Status: status}, event)
			return next.Status, err
		},
	)
}

func TestAttachmentUploadWorkflow_ReplaysTraces(t *testing.T) {
	modeltest.ReplayTraces(t, attachmentUploadTraces(), func(status notes.AttachmentUploadStatus, event notes.AttachmentUploadEvent) (notes.AttachmentUploadStatus, error) {
		next, err := notes.TransitionAttachmentUpload(notes.AttachmentUploadState{Status: status}, event)
		return next.Status, err
	})
}

func TestAttachmentUploadWorkflow_ConformsToSpec(t *testing.T) {
	spec := modeltest.LoadWorkflowSpec(t, "attachment_upload_workflow.spec.json")
	modeltest.AssertWorkflowSpecConformance(
		t,
		spec,
		notes.AllAttachmentUploadStatuses(),
		notes.AllAttachmentUploadEvents(),
		attachmentUploadMatrix(),
		attachmentUploadTraces(),
	)
}

func TestAttachmentUploadWorkflow_RejectsUnknownState(t *testing.T) {
	err := notes.CheckAttachmentUploadInvariants(notes.AttachmentUploadState{Status: "ghost"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown attachment upload status")
}

func attachmentUploadMatrix() []modeltest.MatrixRow[notes.AttachmentUploadStatus, notes.AttachmentUploadEvent] {
	return []modeltest.MatrixRow[notes.AttachmentUploadStatus, notes.AttachmentUploadEvent]{
		{From: notes.AttachmentUploadReceived, Event: notes.AttachmentUploadStoreBytes, To: notes.AttachmentUploadBytesStored},
		{From: notes.AttachmentUploadReceived, Event: notes.AttachmentUploadRecordMetadata, To: notes.AttachmentUploadReceived, WantErr: true},
		{From: notes.AttachmentUploadReceived, Event: notes.AttachmentUploadFail, To: notes.AttachmentUploadFailed},

		{From: notes.AttachmentUploadBytesStored, Event: notes.AttachmentUploadStoreBytes, To: notes.AttachmentUploadBytesStored, WantErr: true},
		{From: notes.AttachmentUploadBytesStored, Event: notes.AttachmentUploadRecordMetadata, To: notes.AttachmentUploadMetadataRecorded},
		{From: notes.AttachmentUploadBytesStored, Event: notes.AttachmentUploadFail, To: notes.AttachmentUploadFailed},

		{From: notes.AttachmentUploadMetadataRecorded, Event: notes.AttachmentUploadStoreBytes, To: notes.AttachmentUploadMetadataRecorded, WantErr: true},
		{From: notes.AttachmentUploadMetadataRecorded, Event: notes.AttachmentUploadRecordMetadata, To: notes.AttachmentUploadMetadataRecorded, WantErr: true},
		{From: notes.AttachmentUploadMetadataRecorded, Event: notes.AttachmentUploadFail, To: notes.AttachmentUploadMetadataRecorded, WantErr: true},

		{From: notes.AttachmentUploadFailed, Event: notes.AttachmentUploadStoreBytes, To: notes.AttachmentUploadFailed, WantErr: true},
		{From: notes.AttachmentUploadFailed, Event: notes.AttachmentUploadRecordMetadata, To: notes.AttachmentUploadFailed, WantErr: true},
		{From: notes.AttachmentUploadFailed, Event: notes.AttachmentUploadFail, To: notes.AttachmentUploadFailed, WantErr: true},
	}
}

func attachmentUploadTraces() []modeltest.Trace[notes.AttachmentUploadStatus, notes.AttachmentUploadEvent] {
	return []modeltest.Trace[notes.AttachmentUploadStatus, notes.AttachmentUploadEvent]{
		{
			Name:    "successful_upload",
			Initial: notes.AttachmentUploadReceived,
			Steps: []modeltest.TraceStep[notes.AttachmentUploadStatus, notes.AttachmentUploadEvent]{
				{Event: notes.AttachmentUploadStoreBytes, Want: notes.AttachmentUploadBytesStored},
				{Event: notes.AttachmentUploadRecordMetadata, Want: notes.AttachmentUploadMetadataRecorded},
			},
		},
		{
			Name:    "store_failure",
			Initial: notes.AttachmentUploadReceived,
			Steps: []modeltest.TraceStep[notes.AttachmentUploadStatus, notes.AttachmentUploadEvent]{
				{Event: notes.AttachmentUploadFail, Want: notes.AttachmentUploadFailed},
				{Event: notes.AttachmentUploadRecordMetadata, Want: notes.AttachmentUploadFailed, WantErr: true},
			},
		},
	}
}
