// [REQ:GCT-OT-P1-002]
package stagedmessages

import (
	"fmt"
	"testing"

	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/advisory"
)

func capturedFor(before, after string) Captured {
	return Captured{Snapshot: &v1.StagedMessageSnapshot{Files: []*v1.StagedMessageFile{{Path: "file", OldBlobId: "base", NewBlobId: "staged", OldMode: "100644", NewMode: "100644", EditCount: int32(len(EditAtoms([]byte(before), []byte(after))))}}}, Bytes: map[string]FileBytes{"file": {[]byte(before), []byte(after)}}}
}

func TestExactRetainedIdentitySupportsOmittedDetailsWithoutClaimingInspection(t *testing.T) {
	for _, kind := range []string{"representative sample", "binary", "large content", "mode-only", "deletion"} {
		t.Run(kind, func(t *testing.T) {
			f := &v1.StagedMessageFile{Path: "file", OldBlobId: "base", NewBlobId: "candidate", OldMode: "100644", NewMode: "100644", OmissionReason: kind, EditCount: 7}
			if kind == "binary" {
				f.Binary = true
			}
			if kind == "mode-only" {
				f.NewBlobId = f.OldBlobId
				f.NewMode = "100755"
			}
			if kind == "deletion" {
				f.NewBlobId = ""
				f.NewMode = "000000"
			}
			c := Captured{Snapshot: &v1.StagedMessageSnapshot{Files: []*v1.StagedMessageFile{f}}}
			m := RetainedMessage{ID: "captured-source", Title: "Retained scoped explanation", Files: []RetainedFile{{Path: f.Path, BaseBlobID: f.OldBlobId, BlobID: f.NewBlobId, OldMode: f.OldMode, NewMode: f.NewMode, Captured: true}}}
			source, covered, ready := matchRetained(c, m)
			cv := coverage(c, covered)
			if !ready || !source.FullMatch || cv.Partial || cv.MatchedEdits != 7 || len(cv.CoveredPaths) != 1 || len(cv.Omissions) != 1 {
				t.Fatalf("exact identity lost or omitted interpretation hidden: source=%v coverage=%v", source, cv)
			}
			m.Files[0].BlobID = "different candidate"
			if _, _, ready := matchRetained(c, m); ready {
				t.Fatal("unavailable fragment bytes became a subset proof")
			}
			m.Files[0].BlobID = f.NewBlobId
			m.Files[0].Captured = false
			if _, _, ready := matchRetained(c, m); ready {
				t.Fatal("legacy unretained identities became retained capture proof")
			}
		})
	}
}

func TestLargeMixedRetainedSelectionRejectsUnstagedClaimsAndRetainsExactCoverage(t *testing.T) {
	c := Captured{Snapshot: &v1.StagedMessageSnapshot{}}
	m := RetainedMessage{ID: "large-capture", Title: "Change all captured settings"}
	for i := 0; i < MaxManifestFiles; i++ {
		path := fmt.Sprintf("path-%04d", i)
		f := &v1.StagedMessageFile{Path: path, OldBlobId: "base", NewBlobId: "candidate", OldMode: "100644", NewMode: "100644", OmissionReason: "detail sample bound"}
		c.Snapshot.Files = append(c.Snapshot.Files, f)
		m.Files = append(m.Files, RetainedFile{Path: path, BaseBlobID: "base", BlobID: "candidate", OldMode: "100644", NewMode: "100644", Captured: true})
	}
	m.Files[MaxManifestFiles-1].BlobID = "unstaged broader candidate"
	source, covered, ready := matchRetained(c, m)
	if ready || source.FullMatch || len(source.MatchedPaths) != MaxManifestFiles-1 {
		t.Fatalf("mixed retained match: ready=%v paths=%d", ready, len(source.MatchedPaths))
	}
	cv := coverage(c, covered)
	if !cv.Partial || len(cv.CoveredPaths) != MaxManifestFiles-1 || len(cv.UncoveredPaths) != 1 || len(cv.Omissions) != MaxManifestFiles {
		t.Fatalf("mixed coverage was truncated: covered=%d uncovered=%d omissions=%d", len(cv.CoveredPaths), len(cv.UncoveredPaths), len(cv.Omissions))
	}
	// An independently scoped source remains reusable for its exact subset.
	m.Files = m.Files[:1]
	_, covered, ready = matchRetained(c, m)
	if !ready || !coverage(c, covered).Partial {
		t.Fatal("bounded exact source subset should remain reusable with disclosure")
	}
}

func retainedFor(before, after string) RetainedMessage {
	return RetainedMessage{ID: "source@1", Revision: "1", Title: "Change selected setting", Body: "Retained explanation", Files: []RetainedFile{{Path: "file", BaseBlobID: "base", BlobID: "candidate", OldMode: "100644", NewMode: "100644", Before: []byte(before), After: []byte(after), Captured: true}}}
}

func TestPartialSourceNeverReusesUnstagedClaims(t *testing.T) {
	base := "a=old\nb=old\n"
	c := capturedFor(base, "a=new\nb=old\n")
	m := retainedFor(base, "a=new\nb=new\n")
	m.Title = "Change a and b"
	source, covered, reusable := matchRetained(c, m)
	if reusable || source.MatchedEdits != 1 || source.FullMatch || len(covered["file"]) != 1 {
		t.Fatalf("partial source %#v ready=%v", source, reusable)
	}
}

func TestRetainedMessageCanCoverSubsetOfStagedSelection(t *testing.T) {
	base := "a=old\nb=old\n"
	c := capturedFor(base, "a=new\nb=new\n")
	m := retainedFor(base, "a=new\nb=old\n")
	source, covered, ready := matchRetained(c, m)
	if !ready || !source.FullMatch {
		t.Fatalf("included source %#v", source)
	}
	s := suggestion(c, m, covered)
	if !s.Coverage.Partial || s.Coverage.MatchedEdits != 1 || s.Coverage.TotalEdits != 2 || len(s.Coverage.UncoveredPaths) != 1 {
		t.Fatalf("coverage %#v", s.Coverage)
	}
}

func TestSamePathUnrelatedEditHasNoContentCorrespondence(t *testing.T) {
	base := "a=old\nb=old\n"
	source, _, ready := matchRetained(capturedFor(base, "a=old\nb=new\n"), retainedFor(base, "a=new\nb=old\n"))
	if ready || source.MatchedEdits != 0 {
		t.Fatal("path became edit proof")
	}
}

func TestEditMatchingSurvivesAdjacentHunkMergingAndOffset(t *testing.T) {
	base := "a=old\nb=old\n"
	c := capturedFor(base, "header\na=new\nb=new\n")
	_, covered, ready := matchRetained(c, retainedFor(base, "a=new\nb=old\n"))
	if !ready || len(covered["file"]) != 1 {
		t.Fatal("adjacent/offset edit lost")
	}
}
