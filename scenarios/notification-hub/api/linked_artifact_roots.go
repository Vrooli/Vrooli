package main

import (
	"path/filepath"
	"strings"
)

// linkedArtifactRoots lists the folders a decision's file links may open from.
// Research Desk decisions name their packets and evidence under the research
// home, so it is a default root beside the work folders and steward delegations.
func linkedArtifactRoots(repoRoot, home, researchDir, extra string) []string {
	roots := []string{}
	if repoRoot != "" {
		roots = append(roots, filepath.Join(repoRoot, ".vrooli", "work"))
	}
	if home != "" {
		roots = append(roots,
			filepath.Join(home, ".vrooli", "work"),
			filepath.Join(home, ".vrooli", "portfolio", "steward", "delegations"),
		)
	}
	if strings.TrimSpace(researchDir) != "" {
		roots = append(roots, researchDir)
	}
	return append(roots, filepath.SplitList(extra)...)
}
