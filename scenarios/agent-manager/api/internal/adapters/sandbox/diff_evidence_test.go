package sandbox

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/google/uuid"
)

func TestIsVerifiedEmptyDiffRequiresCompleteConsistentEvidence(t *testing.T) {
	id := uuid.New()
	digest := sha256.Sum256(nil)
	base := DiffResult{
		SandboxID: id, Files: []FileChange{}, UnifiedDiff: "",
		PatchSHA256: hex.EncodeToString(digest[:]), EvidenceComplete: true,
	}
	tests := []struct {
		name   string
		mutate func(*DiffResult)
		want   bool
	}{
		{name: "live empty", want: true},
		{name: "captured empty archive", mutate: func(d *DiffResult) { d.ArchiveState = ArchiveStateComplete }, want: true},
		{name: "unavailable evidence", mutate: func(d *DiffResult) { d.EvidenceComplete = false }},
		{name: "uncaptured archive", mutate: func(d *DiffResult) { d.ArchiveState = ArchiveStateNotCaptured }},
		{name: "unknown archive state", mutate: func(d *DiffResult) { d.ArchiveState = "future-state" }},
		{name: "wrong sandbox", mutate: func(d *DiffResult) { d.SandboxID = uuid.New() }},
		{name: "missing file index", mutate: func(d *DiffResult) { d.Files = nil }},
		{name: "file present", mutate: func(d *DiffResult) { d.Files = []FileChange{{FilePath: "change.txt"}} }},
		{name: "patch present", mutate: func(d *DiffResult) { d.UnifiedDiff = "diff bytes" }},
		{name: "digest mismatch", mutate: func(d *DiffResult) { d.PatchSHA256 = "" }},
		{name: "nonzero counters", mutate: func(d *DiffResult) { d.Stats.FilesChanged = 1 }},
		{name: "nil result"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var diff *DiffResult
			if test.name != "nil result" {
				copy := base
				if test.mutate != nil {
					test.mutate(&copy)
				}
				diff = &copy
			}
			if got := IsVerifiedEmptyDiff(diff, id); got != test.want {
				t.Fatalf("IsVerifiedEmptyDiff() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestIsVerifiedFileCountRequiresCompleteConsistentEvidence(t *testing.T) {
	id := uuid.New()
	digest := sha256.Sum256([]byte("diff --git a/change.txt b/change.txt\n"))
	emptyDigest := sha256.Sum256(nil)
	base := DiffResult{
		SandboxID:        id,
		Files:            []FileChange{{FilePath: "change.txt"}},
		UnifiedDiff:      "diff --git a/change.txt b/change.txt\n",
		PatchSHA256:      hex.EncodeToString(digest[:]),
		EvidenceComplete: true,
		Stats:            DiffStats{FilesChanged: 1, FilesModified: 1},
	}
	tests := []struct {
		name   string
		mutate func(*DiffResult)
		want   bool
	}{
		{name: "complete live index", want: true},
		{name: "complete archive", mutate: func(d *DiffResult) { d.ArchiveState = ArchiveStateComplete }, want: true},
		{name: "strict verified empty index", mutate: func(d *DiffResult) {
			d.Files = []FileChange{}
			d.UnifiedDiff = ""
			d.PatchSHA256 = hex.EncodeToString(emptyDigest[:])
			d.Stats = DiffStats{}
		}, want: true},
		{name: "incomplete provider response", mutate: func(d *DiffResult) { d.EvidenceComplete = false }},
		{name: "wrong requested sandbox", mutate: func(d *DiffResult) { d.SandboxID = uuid.New() }},
		{name: "missing file index", mutate: func(d *DiffResult) { d.Files = nil }},
		{name: "counter disagrees with index", mutate: func(d *DiffResult) { d.Stats.FilesChanged = 2 }},
		{name: "breakdown disagrees with total", mutate: func(d *DiffResult) { d.Stats.FilesModified = 0 }},
		{name: "negative breakdown counter", mutate: func(d *DiffResult) {
			d.Stats.FilesAdded = -1
			d.Stats.FilesModified = 2
			d.Stats.FilesChanged = 1
		}},
		{name: "overflowing breakdown counters", mutate: func(d *DiffResult) {
			maxInt := int(^uint(0) >> 1)
			d.Stats.FilesAdded = maxInt
			d.Stats.FilesModified = maxInt
			d.Stats.FilesDeleted = 3
		}},
		{name: "uncaptured archive", mutate: func(d *DiffResult) { d.ArchiveState = ArchiveStateNotCaptured }},
		{name: "unknown archive state", mutate: func(d *DiffResult) { d.ArchiveState = "future-state" }},
		{name: "empty index requires strict digest proof", mutate: func(d *DiffResult) { d.Files = []FileChange{} }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			diff := base
			if test.mutate != nil {
				test.mutate(&diff)
			}
			if got := IsVerifiedFileCount(&diff, id); got != test.want {
				t.Fatalf("IsVerifiedFileCount() = %v, want %v", got, test.want)
			}
		})
	}
	if IsVerifiedFileCount(nil, id) {
		t.Fatal("nil diff must not verify a file count")
	}
}
