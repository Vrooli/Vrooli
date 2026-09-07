package main

import (
	"git-control-tower/internal/provenance"
)

// EnrichBlame attaches only explicitly linked, visibility-safe evidence. A
// caller can disable enrichment by passing nil evidence; native blame remains
// usable with an unknown standing.
type EnrichedBlameFile struct {
	BlameFile
	Standing         provenance.Standing   `json:"standing"`
	DowngradeReasons []string              `json:"downgrade_reasons,omitempty"`
	Evidence         []provenance.Evidence `json:"evidence,omitempty"`
}

func JoinBlameEvidence(file BlameFile, complete bool, evidence []provenance.Evidence) EnrichedBlameFile {
	joined := provenance.JoinEvidence(provenance.JoinInput{
		NativeCommit: firstBlameCommit(file.Lines), NativeDigest: file.ContentDigest,
		Complete: complete, Evidence: evidence,
	})
	return EnrichedBlameFile{BlameFile: file, Standing: joined.Standing, DowngradeReasons: joined.Reasons, Evidence: joined.Evidence}
}

func firstBlameCommit(lines []BlameLine) string {
	if len(lines) == 0 {
		return ""
	}
	commit := lines[0].Commit
	for _, line := range lines[1:] {
		if line.Commit != commit {
			// A mixed-line file has no single native commit. Never promote
			// the first line's commit to a file-level standing.
			return ""
		}
	}
	return commit
}
