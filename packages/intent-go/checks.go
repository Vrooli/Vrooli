package intent

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// FileSystem is the minimal file lookup seam used by ref-existence checks.
type FileSystem interface {
	Exists(path string) bool
	Glob(pattern string) ([]string, error)
}

type OSFileSystem struct{}

func (OSFileSystem) Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (OSFileSystem) Glob(pattern string) ([]string, error) {
	return filepath.Glob(pattern)
}

func CheckPRDRefResolves(outcomes, requirements []CapabilityClaim) []Finding {
	outcomeIDs := claimIDSet(outcomes, Outcome)
	var findings []Finding
	for _, req := range requirements {
		if req.Altitude != Requirement {
			continue
		}
		prdRef := RequirementPRDRef(req)
		if prdRef == "" || !strings.HasPrefix(strings.ToUpper(prdRef), "OT-") {
			continue
		}
		if _, ok := outcomeIDs[strings.ToUpper(prdRef)]; ok {
			continue
		}
		findings = append(findings, Finding{
			Code:       CodePRDRefUnmatched,
			Severity:   "warning",
			Message:    "Requirement " + req.ID + " references " + prdRef + ", which matches no operational target in PRD.md.",
			Suggestion: "Fix prd_ref to an existing operational target, or add the missing target to PRD.md.",
			Locations:  []string{req.Anchor, "PRD.md"},
			ClaimID:    req.ID,
			RelatedID:  prdRef,
			Provenance: "intent-go",
		})
	}
	return findings
}

func CheckOrphanOutcome(outcomes, requirements []CapabilityClaim) []Finding {
	refs := make(map[string]struct{})
	for _, req := range requirements {
		if prdRef := RequirementPRDRef(req); prdRef != "" {
			refs[strings.ToUpper(prdRef)] = struct{}{}
		}
	}
	var findings []Finding
	for _, outcome := range outcomes {
		if outcome.Altitude != Outcome {
			continue
		}
		if _, ok := refs[strings.ToUpper(outcome.ID)]; ok {
			continue
		}
		findings = append(findings, Finding{
			Code:       CodeOTOrphan,
			Severity:   "warning",
			Message:    "Operational target " + outcome.ID + " has no requirement pointing at it.",
			Suggestion: "Add a requirement with prd_ref " + outcome.ID + ", or remove the stale operational target.",
			Locations:  []string{outcome.Anchor},
			ClaimID:    outcome.ID,
			Provenance: "intent-go",
		})
	}
	return findings
}

func CheckRefExists(scenarioRoot string, requirements []CapabilityClaim) []Finding {
	return CheckRefExistsWithFS(scenarioRoot, requirements, OSFileSystem{})
}

func CheckRefExistsWithFS(scenarioRoot string, requirements []CapabilityClaim, fsys FileSystem) []Finding {
	if fsys == nil {
		fsys = OSFileSystem{}
	}
	var findings []Finding
	for _, req := range requirements {
		if req.Altitude != Requirement {
			continue
		}
		for _, ref := range req.Refs {
			if ref.Kind != RefCode {
				continue
			}
			if ref.Path == "" {
				if strings.TrimSpace(ref.Raw) != "" {
					findings = append(findings, Finding{
						Code:       CodeRefMissing,
						Severity:   "error",
						Message:    "Requirement " + req.ID + " validation ref is malformed (missing file path): " + ref.Raw,
						Suggestion: "Point validation.ref at an existing path relative to the scenario root, or remove the malformed entry.",
						Locations:  []string{req.Anchor},
						ClaimID:    req.ID,
						RelatedID:  ref.Raw,
						Provenance: "intent-go",
					})
				}
				continue
			}
			if pathExists(fsys, scenarioRoot, ref.Path, ref.Glob) {
				continue
			}
			findings = append(findings, Finding{
				Code:       CodeRefMissing,
				Severity:   "error",
				Message:    "Requirement " + req.ID + " validation references non-existent file: " + ref.Path,
				Suggestion: "Point validation.ref at an existing path relative to the scenario root, or remove the stale entry.",
				Locations:  []string{req.Anchor, ref.Path},
				ClaimID:    req.ID,
				RelatedID:  ref.Raw,
				Provenance: "intent-go",
			})
		}
	}
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].ClaimID == findings[j].ClaimID {
			return findings[i].RelatedID < findings[j].RelatedID
		}
		return findings[i].ClaimID < findings[j].ClaimID
	})
	return findings
}

func claimIDSet(claims []CapabilityClaim, altitude Altitude) map[string]struct{} {
	out := make(map[string]struct{})
	for _, claim := range claims {
		if claim.Altitude == altitude {
			out[strings.ToUpper(claim.ID)] = struct{}{}
		}
	}
	return out
}

func RequirementPRDRef(req CapabilityClaim) string {
	for _, ref := range req.Refs {
		if ref.Kind == RefDoc && strings.HasPrefix(strings.ToUpper(strings.TrimSpace(ref.Raw)), "OT-") {
			return strings.ToUpper(strings.TrimSpace(ref.Raw))
		}
	}
	return ""
}

func pathExists(fsys FileSystem, root, rel string, glob bool) bool {
	full := filepath.Join(root, filepath.FromSlash(rel))
	if glob {
		matches, err := fsys.Glob(full)
		if err == nil && len(matches) > 0 {
			return true
		}
		if fsys.Exists(full) {
			return true
		}
		return false
	}
	if fsys.Exists(full) {
		return true
	}
	for _, prefix := range []string{"api", "ui", "test"} {
		if fsys.Exists(filepath.Join(root, prefix, filepath.FromSlash(rel))) {
			return true
		}
	}
	return false
}
