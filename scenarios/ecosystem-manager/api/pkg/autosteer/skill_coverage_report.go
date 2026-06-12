package autosteer

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/ecosystem-manager/api/pkg/skillmap"
	"github.com/vrooli/maturity-go/dimensions"
	"github.com/vrooli/maturity-go/ladder"
)

// CoveragePolicy declares dimensions that are intentionally uncovered because
// no steer skill exists yet. Every entry needs an external tracking reference so
// coverage gaps stay visible instead of becoming implicit controller stalls.
type CoveragePolicy struct {
	KnownUncovered map[string]KnownUncovered `json:"known_uncovered"`
}

// KnownUncovered explains one intentional skill coverage gap.
type KnownUncovered struct {
	Reason      string `json:"reason"`
	TrackingRef string `json:"tracking_ref"`
}

// CoverageDimensionGap describes a profile dimension that has no eligible skill.
type CoverageDimensionGap struct {
	Dimension   string `json:"dimension"`
	Reason      string `json:"reason,omitempty"`
	TrackingRef string `json:"tracking_ref,omitempty"`
}

// CoverageKnownEntry is a known-uncovered policy entry that applies to a report.
type CoverageKnownEntry struct {
	Dimension   string `json:"dimension"`
	Reason      string `json:"reason"`
	TrackingRef string `json:"tracking_ref"`
}

// CoverageReport is the doctor/preflight payload for a profile's steer-skill
// coverage. JSON consumers should rely on these fields rather than parsing CLI
// text.
type CoverageReport struct {
	ProfileID              string                 `json:"profile_id"`
	ProfileName            string                 `json:"profile_name"`
	Scenario               string                 `json:"scenario,omitempty"`
	EffectiveAllowSet      []string               `json:"effective_allow_set"`
	RelevantDimensions     []string               `json:"relevant_dimensions"`
	GatedUncovered         []CoverageDimensionGap `json:"gated_uncovered"`
	WeightedUnactionable   []CoverageDimensionGap `json:"weighted_unactionable"`
	ExcludedSkills         []string               `json:"excluded_skills"`
	KnownUncoveredInPlay   []CoverageKnownEntry   `json:"known_uncovered_in_play"`
	ReconciliationWarnings []string               `json:"reconciliation_warnings,omitempty"`
}

// CoverageReporter builds coverage reports against a live profile repository
// and skill catalog.
type CoverageReporter struct {
	profiles ProfileRepository
	catalog  skillmap.CatalogSource
	policy   CoveragePolicy
}

// NewCoverageReporter returns a coverage report builder.
func NewCoverageReporter(profiles ProfileRepository, catalog skillmap.CatalogSource, policy CoveragePolicy) *CoverageReporter {
	return &CoverageReporter{profiles: profiles, catalog: catalog, policy: policy}
}

// LoadCoveragePolicy reads and validates a coverage-policy.json file.
func LoadCoveragePolicy(path string) (CoveragePolicy, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return CoveragePolicy{}, fmt.Errorf("read coverage policy: %w", err)
	}
	var policy CoveragePolicy
	if err := json.Unmarshal(raw, &policy); err != nil {
		return CoveragePolicy{}, fmt.Errorf("parse coverage policy: %w", err)
	}
	for dim, entry := range policy.KnownUncovered {
		if !dimensions.IsValid(dimensions.Dimension(dim)) {
			return CoveragePolicy{}, fmt.Errorf("coverage policy references unknown dimension %q", dim)
		}
		if entry.Reason == "" || entry.TrackingRef == "" {
			return CoveragePolicy{}, fmt.Errorf("coverage policy entry %q must include reason and tracking_ref", dim)
		}
	}
	return policy, nil
}

// Report returns the coverage preflight report for one profile.
func (r *CoverageReporter) Report(profileID, scenario string) (CoverageReport, error) {
	if r == nil {
		return CoverageReport{}, fmt.Errorf("coverage reporter is not configured")
	}
	if r.profiles == nil {
		return CoverageReport{}, fmt.Errorf("profile repository is not configured")
	}
	profile, err := r.profiles.GetProfile(profileID)
	if err != nil {
		return CoverageReport{}, err
	}
	resolver := skillmap.NewResolver(r.catalog)

	reconciled := cloneProfile(profile)
	warnings := []string(nil)
	if err := ReconcileProfile(reconciled, resolver); err != nil {
		warnings = append(warnings, err.Error())
	}

	effective := effectiveAllow(profile, resolver)
	report := CoverageReport{
		ProfileID:              profile.ID,
		ProfileName:            profile.Name,
		Scenario:               scenario,
		EffectiveAllowSet:      effective,
		RelevantDimensions:     dimensionStrings(relevantDimensions(profile)),
		ExcludedSkills:         resolver.Excluded(),
		ReconciliationWarnings: warnings,
	}

	knownInPlay := make(map[string]KnownUncovered)
	for _, dim := range gatedDimensionsThroughProfile(profile) {
		if len(resolver.EligibleSkills(dim, effective)) > 0 {
			continue
		}
		gap := r.coverageGap(dim)
		report.GatedUncovered = append(report.GatedUncovered, gap)
		if gap.TrackingRef != "" {
			knownInPlay[string(dim)] = r.policy.KnownUncovered[string(dim)]
		}
	}
	for raw := range profile.Objective.DimensionWeights {
		dim := dimensions.Dimension(raw)
		if len(resolver.EligibleSkills(dim, effective)) > 0 {
			continue
		}
		gap := r.coverageGap(dim)
		report.WeightedUnactionable = append(report.WeightedUnactionable, gap)
		if gap.TrackingRef != "" {
			knownInPlay[string(dim)] = r.policy.KnownUncovered[string(dim)]
		}
	}
	sort.Slice(report.GatedUncovered, func(i, j int) bool {
		return report.GatedUncovered[i].Dimension < report.GatedUncovered[j].Dimension
	})
	sort.Slice(report.WeightedUnactionable, func(i, j int) bool {
		return report.WeightedUnactionable[i].Dimension < report.WeightedUnactionable[j].Dimension
	})
	for dim, entry := range knownInPlay {
		report.KnownUncoveredInPlay = append(report.KnownUncoveredInPlay, CoverageKnownEntry{
			Dimension:   dim,
			Reason:      entry.Reason,
			TrackingRef: entry.TrackingRef,
		})
	}
	sort.Slice(report.KnownUncoveredInPlay, func(i, j int) bool {
		return report.KnownUncoveredInPlay[i].Dimension < report.KnownUncoveredInPlay[j].Dimension
	})
	return report, nil
}

func (r *CoverageReporter) coverageGap(dim dimensions.Dimension) CoverageDimensionGap {
	gap := CoverageDimensionGap{Dimension: string(dim)}
	if entry, ok := r.policy.KnownUncovered[string(dim)]; ok {
		gap.Reason = entry.Reason
		gap.TrackingRef = entry.TrackingRef
	}
	return gap
}

func dimensionStrings(dims []dimensions.Dimension) []string {
	out := make([]string, 0, len(dims))
	for _, dim := range dims {
		out = append(out, string(dim))
	}
	sort.Strings(out)
	return out
}

func gatedDimensionsThroughProfile(profile *AutoSteerProfile) []dimensions.Dimension {
	if !profile.ladderEnabled() {
		return nil
	}
	top := ladder.RungR4
	if parsed, ok := ladder.ParseRung(profile.Ladder.TopRung); ok {
		top = parsed
	}
	seen := make(map[dimensions.Dimension]struct{})
	out := make([]dimensions.Dimension, 0)
	for _, rung := range ladder.Rungs() {
		for _, dim := range rung.Dimensions {
			if _, dup := seen[dim]; dup {
				continue
			}
			seen[dim] = struct{}{}
			out = append(out, dim)
		}
		if rung.ID == top {
			break
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func coverageSummary(profile *AutoSteerProfile, resolver SkillCatalogResolver) (allowCount, relevantCount int, uncovered []string) {
	if resolver == nil {
		return 0, len(relevantDimensions(profile)), nil
	}
	allow := effectiveAllow(profile, resolver)
	relevant := relevantDimensions(profile)
	for _, dim := range relevant {
		if len(resolver.EligibleSkills(dim, allow)) == 0 {
			uncovered = append(uncovered, string(dim))
		}
	}
	sort.Strings(uncovered)
	return len(allow), len(relevant), uncovered
}
