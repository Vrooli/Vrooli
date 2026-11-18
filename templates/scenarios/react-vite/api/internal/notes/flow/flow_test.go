package flow

import (
	"github.com/vrooli/vrooli/packages/proto/modeltest"
	"testing"
	"{{SCENARIO_ID}}/internal/notes/flow/generated"
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

func TestGeneratedAttachmentUploadTransitionSurface(t *testing.T) {
	statuses := generated.AllAttachmentUploadStatuses()
	events := generated.AllAttachmentUploadEvents()
	if len(statuses) != 4 || len(events) != 3 {
		t.Fatalf("generated domain size = %d statuses, %d events; want 4 and 3", len(statuses), len(events))
	}
	for _, status := range statuses {
		for _, event := range events {
			valid := generated.AttachmentUploadIsValidEvent(status, event)
			wantNext := generated.AttachmentUploadNextStatus(status, event)
			gotNext, err := generated.TransitionAttachmentUploadStatus(status, event)
			if gotNext != wantNext {
				t.Errorf("TransitionAttachmentUploadStatus(%q, %q) = %q, want %q", status, event, gotNext, wantNext)
			}
			if valid && err != nil {
				t.Errorf("TransitionAttachmentUploadStatus(%q, %q) unexpected error: %v", status, event, err)
			}
			if !valid && err == nil {
				t.Errorf("TransitionAttachmentUploadStatus(%q, %q) expected invalid-transition error", status, event)
			}
		}
	}
	if len(generated.AttachmentUploadFormalExpectedInvariants()) == 0 || len(generated.AttachmentUploadFormalExpectedGeneratedChecks()) == 0 {
		t.Fatal("generated formal metadata must retain its invariant and check declarations")
	}
}
