package notes_test

import (
	"testing"

	"{{SCENARIO_ID}}/internal/notes"
	"{{SCENARIO_ID}}/internal/testutil/modeltest"

	"github.com/stretchr/testify/require"
)

func TestAttachmentUploadWorkflow_TransitionMatrix(t *testing.T) {
	artifact := modeltest.LoadFormalArtifact(t, "attachment_upload_workflow.formal.generated.json")
	modeltest.AssertFormalTransitionsReplay(
		t,
		artifact,
		notes.AllAttachmentUploadStatuses(),
		notes.AllAttachmentUploadEvents(),
		func(status notes.AttachmentUploadStatus, event notes.AttachmentUploadEvent) (notes.AttachmentUploadStatus, error) {
			next, err := notes.TransitionAttachmentUpload(notes.AttachmentUploadState{Status: status}, event)
			return next.Status, err
		},
	)
}

func TestAttachmentUploadWorkflow_ReplaysTraces(t *testing.T) {
	artifact := modeltest.LoadFormalArtifact(t, "attachment_upload_workflow.formal.generated.json")
	modeltest.AssertFormalTracesReplay(
		t,
		artifact,
		notes.AllAttachmentUploadStatuses(),
		notes.AllAttachmentUploadEvents(),
		func(status notes.AttachmentUploadStatus, event notes.AttachmentUploadEvent) (notes.AttachmentUploadStatus, error) {
			next, err := notes.TransitionAttachmentUpload(notes.AttachmentUploadState{Status: status}, event)
			return next.Status, err
		},
	)
}

func TestAttachmentUploadWorkflow_ReplaysFormalModelArtifacts(t *testing.T) {
	artifact := modeltest.LoadFormalArtifact(t, "attachment_upload_workflow.formal.generated.json")
	modeltest.AssertFormalArtifactFresh(t, artifact, modeltest.FormalArtifactExpectation{
		ContractPath:    notes.AttachmentUploadContractPath,
		ContractSHA256:  notes.AttachmentUploadContractSHA256,
		ModelPath:       notes.AttachmentUploadModelPath,
		ModelSHA256:     notes.AttachmentUploadModelSHA256,
		GeneratorPath:   notes.AttachmentUploadGeneratorPath,
		GeneratorSHA256: notes.AttachmentUploadGeneratorSHA256,
		Invariants:      notes.AttachmentUploadFormalExpectedInvariants(),
		GeneratedChecks: notes.AttachmentUploadFormalExpectedGeneratedChecks(),
	})
	transition := func(status notes.AttachmentUploadStatus, event notes.AttachmentUploadEvent) (notes.AttachmentUploadStatus, error) {
		next, err := notes.TransitionAttachmentUpload(notes.AttachmentUploadState{Status: status}, event)
		return next.Status, err
	}
	modeltest.AssertFormalTransitionsReplay(
		t,
		artifact,
		notes.AllAttachmentUploadStatuses(),
		notes.AllAttachmentUploadEvents(),
		transition,
	)
	modeltest.AssertFormalTracesReplay(
		t,
		artifact,
		notes.AllAttachmentUploadStatuses(),
		notes.AllAttachmentUploadEvents(),
		transition,
	)
}

func TestAttachmentUploadWorkflow_RejectsUnknownState(t *testing.T) {
	err := notes.CheckAttachmentUploadInvariants(notes.AttachmentUploadState{Status: "ghost"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown attachment upload status")
}
