package autosteer

import (
	"fmt"
	"sort"
	"strings"

	"github.com/vrooli/maturity-go/dimensions"
)

// SkillCatalogResolver is the selector's catalog view: dimension eligibility
// plus enough enumeration to derive a profile's effective allow-set.
type SkillCatalogResolver interface {
	EligibleSkills(dim dimensions.Dimension, allow []string) []string
	DimensionsForSkill(skillID string) []dimensions.Dimension
	AllSkills() []string
}

// relevantDimensions returns the dimensions a profile values (its explicit
// weight keys).
func relevantDimensions(profile *AutoSteerProfile) []dimensions.Dimension {
	if profile == nil {
		return nil
	}
	seen := make(map[dimensions.Dimension]struct{})
	for raw := range profile.Objective.DimensionWeights {
		dim := dimensions.Dimension(raw)
		if dimensions.IsValid(dim) {
			seen[dim] = struct{}{}
		}
	}
	out := make([]dimensions.Dimension, 0, len(seen))
	for dim := range seen {
		out = append(out, dim)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func derivedAllow(profile *AutoSteerProfile, resolver SkillCatalogResolver) []string {
	if resolver == nil {
		return nil
	}
	relevant := relevantDimensions(profile)
	if len(relevant) == 0 {
		return nil
	}
	relevantSet := make(map[dimensions.Dimension]struct{}, len(relevant))
	for _, dim := range relevant {
		relevantSet[dim] = struct{}{}
	}
	out := make([]string, 0)
	for _, skillID := range resolver.AllSkills() {
		for _, dim := range resolver.DimensionsForSkill(skillID) {
			if _, ok := relevantSet[dim]; ok {
				out = append(out, skillID)
				break
			}
		}
	}
	return out
}

// effectiveAllow derives the profile's selectable skills from the catalog and
// treats allowed_skills as an optional restriction mask and denied_skills as a
// subtractive mask. ValidateProfile requires real profiles to value at least one
// dimension; the explicit-mask fallback keeps narrow selector unit tests usable.
func effectiveAllow(profile *AutoSteerProfile, resolver SkillCatalogResolver) []string {
	if profile == nil {
		return nil
	}
	derived := derivedAllow(profile, resolver)
	if len(derived) == 0 && len(relevantDimensions(profile)) == 0 {
		return subtractSkills(normalizeSkillIDs(profile.AllowedSkills), profile.DeniedSkills)
	}
	effective := derived
	if len(profile.AllowedSkills) > 0 {
		effective = intersectSkills(derived, normalizeSkillIDs(profile.AllowedSkills))
	}
	return subtractSkills(effective, profile.DeniedSkills)
}

// ReconcileProfile checks catalog-aware invariants that pure profile validation
// cannot: restriction masks may only name skills that target relevant profile
// dimensions. It normalizes mask fields in place.
func ReconcileProfile(profile *AutoSteerProfile, resolver SkillCatalogResolver) error {
	if profile == nil {
		return fmt.Errorf("profile is required")
	}
	derived := derivedAllow(profile, resolver)
	derivedSet := make(map[string]struct{}, len(derived))
	for _, skillID := range derived {
		derivedSet[skillID] = struct{}{}
	}
	for _, field := range []struct {
		name   string
		values *[]string
	}{
		{name: "allowed_skills", values: &profile.AllowedSkills},
		{name: "denied_skills", values: &profile.DeniedSkills},
	} {
		normalized, err := normalizeSkillIDsStrict(*field.values, field.name)
		if err != nil {
			return err
		}
		*field.values = normalized
		for _, skillID := range normalized {
			if _, ok := derivedSet[skillID]; !ok {
				return fmt.Errorf("%s contains %q, but that skill targets no dimension valued by profile %q", field.name, skillID, profile.Name)
			}
		}
	}
	return nil
}

func normalizeSkillIDs(values []string) []string {
	out, _ := normalizeSkillIDsStrict(values, "skill list")
	return out
}

func normalizeSkillIDsStrict(values []string, field string) ([]string, error) {
	out := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, raw := range values {
		id := strings.TrimSpace(raw)
		if id == "" {
			return nil, fmt.Errorf("%s contains an empty skill id", field)
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out, nil
}

func intersectSkills(base, mask []string) []string {
	if len(base) == 0 || len(mask) == 0 {
		return nil
	}
	maskSet := make(map[string]struct{}, len(mask))
	for _, id := range mask {
		maskSet[id] = struct{}{}
	}
	out := make([]string, 0, len(base))
	for _, id := range base {
		if _, ok := maskSet[id]; ok {
			out = append(out, id)
		}
	}
	return out
}

func subtractSkills(base, denied []string) []string {
	if len(base) == 0 || len(denied) == 0 {
		return append([]string(nil), base...)
	}
	denySet := make(map[string]struct{}, len(denied))
	for _, id := range normalizeSkillIDs(denied) {
		denySet[id] = struct{}{}
	}
	out := make([]string, 0, len(base))
	for _, id := range base {
		if _, denied := denySet[id]; denied {
			continue
		}
		out = append(out, id)
	}
	return out
}
