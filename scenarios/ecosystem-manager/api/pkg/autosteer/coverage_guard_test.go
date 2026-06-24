package autosteer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/ecosystem-manager/api/pkg/skillmap"
	"github.com/vrooli/maturity-go/dimensions"
)

type coveragePolicy struct {
	KnownUncovered map[string]knownUncovered `json:"known_uncovered"`
}

type knownUncovered struct {
	Reason      string `json:"reason"`
	TrackingRef string `json:"tracking_ref"`
}

func loadCoveragePolicy(t *testing.T, dir string) coveragePolicy {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(dir, "coverage-policy.json"))
	if err != nil {
		t.Fatalf("read coverage-policy.json: %v", err)
	}
	var policy coveragePolicy
	if err := json.Unmarshal(raw, &policy); err != nil {
		t.Fatalf("parse coverage-policy.json: %v", err)
	}
	for dim, entry := range policy.KnownUncovered {
		if !dimensions.IsValid(dimensions.Dimension(dim)) {
			t.Fatalf("coverage policy references unknown dimension %q", dim)
		}
		if entry.Reason == "" || entry.TrackingRef == "" {
			t.Fatalf("coverage policy entry %q must include reason and tracking_ref", dim)
		}
	}
	return policy
}

func loadShippedProfiles(t *testing.T, dir string) []*AutoSteerProfile {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(dir, "metadata.json"))
	if err != nil {
		t.Fatalf("read metadata.json: %v", err)
	}
	var idx ProfileMetadataIndex
	if err := json.Unmarshal(raw, &idx); err != nil {
		t.Fatalf("parse metadata.json: %v", err)
	}
	out := make([]*AutoSteerProfile, 0, len(idx.Profiles))
	for _, entry := range idx.Profiles {
		profile := loadProfileJSON(t, dir, filepath.Dir(entry.File))
		profile.Name = entry.Name
		profile.Description = entry.Description
		profile.Tags = append([]string(nil), entry.Tags...)
		if err := ValidateProfile(profile); err != nil {
			t.Fatalf("profile %s invalid: %v", entry.ID, err)
		}
		out = append(out, profile)
	}
	return out
}

func loadCatalogDeclarations(t *testing.T) []skillmap.SkillDeclaration {
	t.Helper()
	packs := findPromptManagerPacksForCoverage(t)
	if packs == "" {
		t.Skip("prompt-manager skill store not reachable from this checkout; coverage guard skipped")
	}
	decls := make([]skillmap.SkillDeclaration, 0)
	err := filepath.WalkDir(packs, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || d.Name() != "skill.json" {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("%s: read: %v", path, err)
			return nil
		}
		var s struct {
			ID               string   `json:"id"`
			TargetDimensions []string `json:"targetDimensions"`
		}
		if err := json.Unmarshal(raw, &s); err != nil {
			t.Errorf("%s: invalid skill.json: %v", path, err)
			return nil
		}
		if len(s.TargetDimensions) > 0 {
			decls = append(decls, skillmap.SkillDeclaration{ID: s.ID, Dimensions: s.TargetDimensions})
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", packs, err)
	}
	if len(decls) == 0 {
		t.Fatalf("no targetDimensions declarations found under %s", packs)
	}
	return decls
}

func findPromptManagerPacksForCoverage(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for i := 0; i < 8; i++ {
		candidate := filepath.Join(dir, "prompt-manager", "store", "skills", "packs")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

func dimensionHasEligibleSkill(profile *AutoSteerProfile, resolver *skillmap.Resolver, dim dimensions.Dimension) bool {
	return len(resolver.EligibleSkills(dim, effectiveAllow(profile, resolver))) > 0
}

func TestCoverageGuardCatalogReachability(t *testing.T) {
	profiles := loadShippedProfiles(t, profilesDir(t))
	resolver := skillmap.NewResolverWithWarner(&skillmap.FakeCatalog{Declarations: loadCatalogDeclarations(t)}, func(string, ...any) {})
	reachable := make(map[dimensions.Dimension]struct{})
	for _, profile := range profiles {
		for _, dim := range relevantDimensions(profile) {
			reachable[dim] = struct{}{}
		}
	}
	for _, skillID := range resolver.AllSkills() {
		dims := resolver.DimensionsForSkill(skillID)
		hit := false
		for _, dim := range dims {
			if _, ok := reachable[dim]; ok {
				hit = true
				break
			}
		}
		if !hit {
			t.Errorf("skill %q declares valid dimensions %v but no shipped profile values any of them", skillID, dims)
		}
	}
}

func TestCoverageGuardWeightedDimensionCoverage(t *testing.T) {
	dir := profilesDir(t)
	profiles := loadShippedProfiles(t, dir)
	policy := loadCoveragePolicy(t, dir)
	resolver := skillmap.NewResolverWithWarner(&skillmap.FakeCatalog{Declarations: loadCatalogDeclarations(t)}, func(string, ...any) {})
	for _, profile := range profiles {
		for raw := range profile.Objective.DimensionWeights {
			dim := dimensions.Dimension(raw)
			if dimensionHasEligibleSkill(profile, resolver, dim) {
				continue
			}
			if _, known := policy.KnownUncovered[raw]; known {
				continue
			}
			t.Errorf("profile %q weights dimension %q but no eligible skill targets it and coverage-policy.json has no known_uncovered entry", profile.Name, dim)
		}
	}
}
