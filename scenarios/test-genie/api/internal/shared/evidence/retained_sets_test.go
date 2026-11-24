package evidence

import (
	"testing"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
)

func TestCloneRetainedSetsPreservesIndependentEvidence(t *testing.T) {
	input := []*scenariovalidationv1.RetainedEvidenceSet{
		nil,
		{ProducerReceiptId: "receipt-1", Artifacts: []*commonv1.EvidenceRef{{ArtifactId: "artifact-1", Checksum: "original"}}},
	}

	got := CloneRetainedSets(input)
	if len(got) != 1 || got[0] == input[1] || got[0].GetArtifacts()[0] == input[1].GetArtifacts()[0] {
		t.Fatalf("clone did not preserve only an independent retained set: %#v", got)
	}
	input[1].Artifacts[0].Checksum = "mutated source"
	if got[0].GetArtifacts()[0].GetChecksum() != "original" {
		t.Fatalf("nested evidence aliases source after clone: %q", got[0].GetArtifacts()[0].GetChecksum())
	}
}

func TestCloneRetainedSetsReturnsEmptySliceForNilInput(t *testing.T) {
	got := CloneRetainedSets(nil)
	if got == nil || len(got) != 0 {
		t.Fatalf("CloneRetainedSets(nil) = %#v, want non-nil empty slice", got)
	}
}
