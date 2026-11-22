package main

import (
	"git-control-tower/internal/provenance"
)

func enrichedBlameFile(file BlameFile, evidence []provenance.Evidence) BlameFile {
	joined := JoinBlameEvidence(file, file.ContentDigest != "", evidence)
	file.Standing, file.DowngradeReasons, file.Evidence = joined.Standing, joined.DowngradeReasons, joined.Evidence
	return file
}
