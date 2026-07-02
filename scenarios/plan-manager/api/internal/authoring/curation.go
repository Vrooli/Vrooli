package authoring

import (
	"sort"
	"strings"

	planmodel "plan-manager/internal/planmodel"
)

const (
	contextSearchShortlistScore = 0.60
	contextHighConfidenceScore  = 0.80
	contextSkillShortlistCap    = 3
	contextDocShortlistCap      = 3
	contextActionShortlistCap   = 2
)

func curateContextBatch(candidates []ContextCandidate) ([]ContextCandidate, CurationStats) {
	out := append([]ContextCandidate(nil), candidates...)
	sort.SliceStable(out, func(i, j int) bool {
		return contextCandidateRankLess(out[i], out[j])
	})

	var stats CurationStats
	used := map[planmodel.RelevantContextKind]int{}
	for i := range out {
		out[i].Tier = contextTierLonglist
		out[i].HighConfidence = contextCandidateHighConfidence(out[i])
		eligible, reason := contextCandidateShortlistEligible(out[i])
		if !eligible {
			switch reason {
			case "topic":
				stats.OmittedTopicFiller++
			default:
				stats.OmittedBelowThreshold++
			}
			continue
		}
		if used[contextCapKind(out[i].Item.Kind)] >= contextKindCap(out[i].Item.Kind) {
			stats.OmittedByCap++
			continue
		}
		used[contextCapKind(out[i].Item.Kind)]++
		out[i].Tier = contextTierShortlist
	}
	return out, stats
}

func contextCandidateRankLess(a, b ContextCandidate) bool {
	aCorroborated := contextCorroborationCount(a) >= 2
	bCorroborated := contextCorroborationCount(b) >= 2
	if aCorroborated != bCorroborated {
		return aCorroborated
	}
	aTopic := contextOriginTopicOnly(a)
	bTopic := contextOriginTopicOnly(b)
	if aTopic != bTopic {
		return !aTopic
	}
	if a.Score != b.Score {
		return a.Score > b.Score
	}
	if a.Item.Kind != b.Item.Kind {
		return string(a.Item.Kind) < string(b.Item.Kind)
	}
	return strings.ToLower(firstNonEmpty(a.Item.Target, a.Item.Command, a.Item.Label)) <
		strings.ToLower(firstNonEmpty(b.Item.Target, b.Item.Command, b.Item.Label))
}

func contextCandidateShortlistEligible(candidate ContextCandidate) (bool, string) {
	if contextCorroborationCount(candidate) >= 2 {
		return true, ""
	}
	if contextOriginTopicOnly(candidate) {
		return false, "topic"
	}
	if strings.TrimSpace(candidate.Origin) == "" {
		return true, ""
	}
	if candidate.Score >= contextSearchShortlistScore {
		return true, ""
	}
	return false, "score"
}

func contextCandidateHighConfidence(candidate ContextCandidate) bool {
	if contextCorroborationCount(candidate) >= 2 {
		return true
	}
	return !contextOriginTopicOnly(candidate) && candidate.Score >= contextHighConfidenceScore
}

func contextCorroborationCount(candidate ContextCandidate) int {
	if len(candidate.Corroboration) == 0 {
		return 1
	}
	seen := map[string]struct{}{}
	for _, hit := range candidate.Corroboration {
		key := strings.TrimSpace(hit.Probe) + "\x00" + strings.TrimSpace(hit.Concept)
		if key == "\x00" {
			continue
		}
		seen[key] = struct{}{}
	}
	if len(seen) == 0 {
		return 1
	}
	return len(seen)
}

func contextOriginTopicOnly(candidate ContextCandidate) bool {
	return strings.EqualFold(strings.TrimSpace(candidate.Origin), "topic")
}

func contextKindCap(kind planmodel.RelevantContextKind) int {
	switch contextCapKind(kind) {
	case planmodel.RelevantContextSkill:
		return contextSkillShortlistCap
	case planmodel.RelevantContextDoc:
		return contextDocShortlistCap
	default:
		return contextActionShortlistCap
	}
}

func contextCapKind(kind planmodel.RelevantContextKind) planmodel.RelevantContextKind {
	switch kind {
	case planmodel.RelevantContextSkill, planmodel.RelevantContextDoc:
		return kind
	default:
		return planmodel.RelevantContextCommand
	}
}
