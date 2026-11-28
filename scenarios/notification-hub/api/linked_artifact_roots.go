package main

import (
	"path/filepath"
	"strings"

	repocontract "github.com/vrooli/repo-contract-go"
)

// linkedArtifactRoots lists the folders a decision's file links may open from.
// Research Desk decisions name their packets and evidence under the research
// home, so it is a default root beside the work folders and steward delegations.
func linkedArtifactRoots(repoRoot, home, researchDir, extra string) []string {
	roots := []string{}
	if repoRoot != "" {
		roots = append(roots, filepath.Join(repoRoot, ".vrooli", "work"))
		// Decision prose commonly names repository files relative to the repo
		// (for example, scenarios/notification-hub/README.md). Keep that source
		// available to the owner while the resolver still enforces repo containment.
		roots = append(roots, repoRoot)
	}
	if home != "" {
		if runtimeHome, err := repocontract.VrooliUserRoot(home); err == nil {
			roots = append(roots,
				filepath.Join(runtimeHome, "work"),
				filepath.Join(runtimeHome, "portfolio", "steward", "delegations"),
			)
		}
	}
	if strings.TrimSpace(researchDir) != "" {
		roots = append(roots, researchDir)
	}
	return append(roots, filepath.SplitList(extra)...)
}
