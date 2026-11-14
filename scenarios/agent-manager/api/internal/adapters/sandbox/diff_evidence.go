package sandbox

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/google/uuid"
)

// IsVerifiedEmptyDiff is true only when the provider marked the response
// complete and its exact patch, file list, counters, archive state, and
// sandbox identity all describe an empty diff. Missing evidence is unknown.
func IsVerifiedEmptyDiff(diff *DiffResult, expectedSandboxID uuid.UUID) bool {
	if diff == nil || expectedSandboxID == uuid.Nil || diff.SandboxID != expectedSandboxID || !diff.EvidenceComplete {
		return false
	}
	if diff.Files == nil || len(diff.Files) != 0 || diff.UnifiedDiff != "" {
		return false
	}
	if diff.ArchiveState != "" && diff.ArchiveState != ArchiveStateComplete {
		return false
	}
	patchDigest := sha256.Sum256([]byte(diff.UnifiedDiff))
	if diff.PatchSHA256 != hex.EncodeToString(patchDigest[:]) {
		return false
	}
	return diff.Stats == (DiffStats{})
}

// IsVerifiedFileCount reports whether the response contains a complete file
// index whose count agrees with the provider's aggregate counters. Empty
// results use the stricter digest and zero-counter predicate above.
func IsVerifiedFileCount(diff *DiffResult, expectedSandboxID uuid.UUID) bool {
	if diff == nil || expectedSandboxID == uuid.Nil || diff.SandboxID != expectedSandboxID || !diff.EvidenceComplete {
		return false
	}
	if diff.Files == nil || (diff.ArchiveState != "" && diff.ArchiveState != ArchiveStateComplete) {
		return false
	}
	if diff.Stats.FilesChanged != len(diff.Files) {
		return false
	}
	if diff.Stats.FilesAdded < 0 || diff.Stats.FilesModified < 0 || diff.Stats.FilesDeleted < 0 {
		return false
	}
	// Every category is part of the verified total, so bound each before
	// summing. This also prevents malformed counters from overflowing int.
	if diff.Stats.FilesAdded > len(diff.Files) || diff.Stats.FilesModified > len(diff.Files) || diff.Stats.FilesDeleted > len(diff.Files) {
		return false
	}
	if diff.Stats.FilesChanged != diff.Stats.FilesAdded+diff.Stats.FilesModified+diff.Stats.FilesDeleted {
		return false
	}
	if len(diff.Files) == 0 {
		return IsVerifiedEmptyDiff(diff, expectedSandboxID)
	}
	return true
}
