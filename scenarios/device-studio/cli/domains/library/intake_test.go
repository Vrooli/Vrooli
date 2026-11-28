package library

import (
	"strings"
	"testing"
)

func TestMaterialCorrectionResponseBindsExactChildParentAndReviewState(t *testing.T) {
	parentSHA := strings.Repeat("a", 64)
	valid := materialCorrectionCandidate{ID: "material-child", Revision: 1, RevisionSHA256: strings.Repeat("c", 64), OriginalSHA256: strings.Repeat("d", 64), ReviewStatus: "unreviewed", RenderStatus: "not_reviewed", ParentCandidateID: "material-parent", ParentRevision: 2, ParentSHA256: parentSHA, ImportMethod: "local_material_correction"}
	if err := validateMaterialCorrectionCandidate(valid, "material-child", "material-parent", 2, parentSHA); err != nil {
		t.Fatalf("valid child response rejected: %v", err)
	}
	cases := []struct {
		name   string
		mutate func(*materialCorrectionCandidate)
	}{
		{"child ID", func(c *materialCorrectionCandidate) { c.ID = "other-child" }},
		{"parent ID", func(c *materialCorrectionCandidate) { c.ParentCandidateID = "other-parent" }},
		{"parent revision", func(c *materialCorrectionCandidate) { c.ParentRevision++ }},
		{"parent digest", func(c *materialCorrectionCandidate) { c.ParentSHA256 = strings.Repeat("b", 64) }},
		{"promoted review state", func(c *materialCorrectionCandidate) { c.ReviewStatus = "reviewed" }},
		{"renderable state", func(c *materialCorrectionCandidate) { c.RenderStatus = "ready" }},
		{"wrong import method", func(c *materialCorrectionCandidate) { c.ImportMethod = "local_upload" }},
		{"noninitial revision", func(c *materialCorrectionCandidate) { c.Revision = 2 }},
		{"invalid child digest", func(c *materialCorrectionCandidate) { c.RevisionSHA256 = "not-a-digest" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			candidate := valid
			tc.mutate(&candidate)
			if err := validateMaterialCorrectionCandidate(candidate, "material-child", "material-parent", 2, parentSHA); err == nil {
				t.Fatal("mismatched correction response was accepted")
			}
		})
	}
}
