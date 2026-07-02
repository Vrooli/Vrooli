package flow

import (
	"testing"
	"{{SCENARIO_ID}}/internal/notes/flow/generated"
	"{{SCENARIO_ID}}/internal/testutil/modeltest"
)

func TestAttachmentUploadFormalReplay(t *testing.T) {
	artifact := modeltest.LoadFormalArtifact(t, "generated/artifact.json")
	modeltest.AssertFormalArtifactFresh(t, artifact, modeltest.FormalArtifactExpectation{
		ContractPath:    generated.AttachmentUploadContractPath,
		ContractSHA256:  generated.AttachmentUploadContractSHA256,
		ModelPath:       generated.AttachmentUploadModelPath,
		ModelSHA256:     generated.AttachmentUploadModelSHA256,
		GeneratorPath:   generated.AttachmentUploadGeneratorPath,
		GeneratorSHA256: generated.AttachmentUploadGeneratorSHA256,
		Invariants:      generated.AttachmentUploadFormalExpectedInvariants(),
		GeneratedChecks: generated.AttachmentUploadFormalExpectedGeneratedChecks(),
	})
	transition := func(s generated.AttachmentUploadStatus, e generated.AttachmentUploadEvent) (generated.AttachmentUploadStatus, error) {
		next, err := TransitionAttachmentUpload(AttachmentUploadState{Status: s}, e)
		return next.Status, err
	}
	modeltest.AssertFormalTransitionsReplay(t, artifact, generated.AllAttachmentUploadStatuses(), generated.AllAttachmentUploadEvents(), transition)
	modeltest.AssertFormalTracesReplay(t, artifact, generated.AllAttachmentUploadStatuses(), generated.AllAttachmentUploadEvents(), transition)
}
