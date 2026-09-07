package controls

import "testing"

func TestMergePreviewRequiresExactRevisionIdentity(t *testing.T) {
	preview := MergePreview{RepositoryID: "repo", SourceRef: "feature", Destination: "main", ExpectedBase: "base-1", SourceHead: "head-1", SubjectDigest: "sha256:s"}
	if err := preview.Validate(); err != nil {
		t.Fatal(err)
	}
	preview.ExpectedBase = ""
	if err := preview.Validate(); err == nil {
		t.Fatal("incomplete merge precondition accepted")
	}
}

func TestUndoRejectsMixedOrUnverifiedAttribution(t *testing.T) {
	undo := UndoPreparation{RepositoryID: "repo", SubjectDigest: "sha256:s", HumanOnly: true, Files: []UndoFile{{Path: "a.go", CurrentDigest: "sha256:c", PreimageDigest: "sha256:p", AttributionState: "mixed"}}}
	if err := undo.Validate(); err == nil {
		t.Fatal("mixed attribution accepted")
	}
	undo.Files[0].AttributionState = "verified"
	if err := undo.Validate(); err != nil {
		t.Fatal(err)
	}
}
