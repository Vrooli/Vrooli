package autosteer

import (
	"fmt"
	"sort"

	"github.com/ecosystem-manager/api/pkg/findings"
	"github.com/ecosystem-manager/api/pkg/skillmap"
	"github.com/vrooli/maturity-go/dimensions"
)

// SkillResolver is the subset of skillmap.Resolver the selector needs. It is an
// interface so selection can be unit-tested without a full catalog.
//
// seam: SkillResolver maps profile-valued dimensions to the effective allow-set,
// then resolves skills that can close a concrete dimension.
type SkillResolver = SkillCatalogResolver

var _ SkillResolver = (*skillmap.Resolver)(nil)

// Selection is the controller's SELECT-stage decision.
type Selection struct {
	// SkillID is the chosen skill, or "" when no eligible skill exists for any
	// open dimension (the caller treats this as "nothing actionable").
	SkillID string
	// Dimension is the heaviest open dimension the chosen skill targets.
	Dimension dimensions.Dimension
	// WeightedScore is that dimension's profile-weighted open score.
	WeightedScore float64
	// Rationale is a human-readable explanation for the decision trace.
	Rationale string
}

// Selector implements the controller's SELECT stage: greedy diagnosis. Open
// findings are bucketed by dimension and ranked by profile-weighted severity
// (which dimension matters most now); the chosen skill is the first eligible
// skill that targets the heaviest actionable dimension. Deterministic and fully
// explainable — see docs/concepts/CONTROL-MODEL.md "Selection Policy".
type Selector struct {
	resolver SkillResolver
}

// NewSelector creates a greedy Selector over a skill resolver.
func NewSelector(resolver SkillResolver) *Selector {
	return &Selector{resolver: resolver}
}

// weightedDimension is a dimension with its profile-weighted open score.
type weightedDimension struct {
	dim    dimensions.Dimension
	score  float64
	weight float64
	count  int
}

// rankDimensions orders open dimensions by profile-weighted score descending,
// with a deterministic alphabetical tiebreak. A dimension absent from the
// weights map defaults to weight 1.0.
func rankDimensions(state findings.FindingsState, profile *AutoSteerProfile) []weightedDimension {
	weights := map[string]float64{}
	if profile != nil {
		weights = profile.Objective.DimensionWeights
	}
	ranked := make([]weightedDimension, 0, len(state.DimensionScore))
	for dim, raw := range state.DimensionScore {
		if raw <= 0 {
			continue
		}
		w := 1.0
		if wv, ok := weights[string(dim)]; ok {
			w = wv
		}
		ranked = append(ranked, weightedDimension{
			dim:    dim,
			score:  raw * w,
			weight: w,
			count:  state.DimensionCount[dim],
		})
	}
	sort.Slice(ranked, func(i, j int) bool {
		if ranked[i].score != ranked[j].score {
			return ranked[i].score > ranked[j].score
		}
		return ranked[i].dim < ranked[j].dim
	})
	return ranked
}

// SelectNextSkill picks the skill that best closes the heaviest open dimension.
// It falls through to the next dimension when the heaviest has no eligible
// skill in the allow-set, and returns an empty SkillID when nothing is
// actionable.
func (s *Selector) SelectNextSkill(state findings.FindingsState, profile *AutoSteerProfile) Selection {
	ranked := rankDimensions(state, profile)
	if len(ranked) == 0 {
		return Selection{Rationale: "no open findings — nothing to select"}
	}

	allow := effectiveAllow(profile, s.resolver)

	skipped := make([]string, 0)
	for _, wd := range ranked {
		eligible := s.resolver.EligibleSkills(wd.dim, allow)
		if len(eligible) == 0 {
			skipped = append(skipped, string(wd.dim))
			continue
		}
		chosen := eligible[0]
		rationale := fmt.Sprintf(
			"dimension %q is the heaviest open cluster with an eligible skill (weighted score %.2f, %d findings, weight %.2f); selected %q",
			wd.dim, wd.score, wd.count, wd.weight, chosen,
		)
		if len(skipped) > 0 {
			rationale += fmt.Sprintf(" (skipped %v — no eligible skill in allow-set)", skipped)
		}
		if profile != nil && (len(profile.AllowedSkills) > 0 || len(profile.DeniedSkills) > 0) {
			rationale += fmt.Sprintf(" (effective allow-set size %d after profile restrictions)", len(allow))
		}
		return Selection{
			SkillID:       chosen,
			Dimension:     wd.dim,
			WeightedScore: wd.score,
			Rationale:     rationale,
		}
	}

	return Selection{
		Rationale: fmt.Sprintf("no eligible skill in allow-set for any open dimension %v", skipped),
	}
}
