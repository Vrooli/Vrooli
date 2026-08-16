package routing

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	aisearch "github.com/vrooli/ai-go/search"
	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"
)

const defaultAutoExternalThreshold = 0.60

const defaultMaxFanoutWidth = aisearch.DefaultRouterMaxFanoutWidth

const (
	selectorLegCrossEncoder = "cross_encoder"
	selectorLegLLM          = "llm"
	selectorLegLexical      = "lexical"
)

func autoExternalThreshold() float64 {
	if raw := strings.TrimSpace(os.Getenv("SEARCH_HUB_AUTO_EXTERNAL_THRESHOLD")); raw != "" {
		if value, err := strconv.ParseFloat(raw, 64); err == nil && value >= 0 && value <= 1 {
			return value
		}
	}
	return defaultAutoExternalThreshold
}

func truncateForErr(value string) string {
	value = strings.Join(strings.Fields(value), " ")
	if len(value) > 120 {
		return value[:120] + "…"
	}
	return value
}

// ProviderProfile is the registry-derived text used by the routing ladder.
// The router never embeds provider-specific knowledge; registered identity,
// type, group, and description are the complete scoring input.
type ProviderProfile struct {
	ProviderID         string
	Type               string
	Group              string
	Description        string
	OmittedProviderIDs []string
}

func buildProfiles(active []*registryv1.ProviderDescriptor) []ProviderProfile {
	out := make([]ProviderProfile, 0, len(active))
	for _, provider := range active {
		if provider.GetEndpoint() == nil {
			continue
		}
		id := strings.TrimSpace(provider.GetProviderId())
		typ := strings.TrimSpace(provider.GetType())
		if id == "" || typ == "" {
			continue
		}
		out = append(out, ProviderProfile{
			ProviderID:  id,
			Type:        typ,
			Group:       strings.TrimSpace(provider.GetProviderGroup()),
			Description: strings.TrimSpace(provider.GetDescription()),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ProviderID < out[j].ProviderID })
	return out
}

// lexicalProviderShortlist is intentionally model-free. Provider identity is
// part of the scored text so two otherwise similar descriptions remain
// distinguishable, and stable id ordering makes a fallback reproducible.
func lexicalProviderShortlist(query string, profiles []ProviderProfile, limit int) []ProviderProfile {
	if limit <= 0 || limit > len(profiles) {
		limit = len(profiles)
	}
	scored := make([]scoredProfile, 0, len(profiles))
	for _, profile := range profiles {
		scored = append(scored, scoredProfile{profile: profile, score: lexicalDescriptionScore(query, profile)})
	}
	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].score == scored[j].score {
			return scored[i].profile.ProviderID < scored[j].profile.ProviderID
		}
		return scored[i].score > scored[j].score
	})
	out := make([]ProviderProfile, 0, limit)
	for _, item := range scored[:limit] {
		out = append(out, item.profile)
	}
	return out
}

// fuseProviderRankings combines the semantic ordering with the named lexical
// fallback using reciprocal rank fusion. The provider description index is
// intentionally allowed to disagree with lexical evidence: a provider that
// both signals select receives the strongest score, while an exact lexical
// owner is not discarded merely because a dense model is uncertain about a
// short query. This is generic evidence fusion, not a provider-specific rule.
func fuseProviderRankings(primary, secondary []ProviderProfile) []ProviderProfile {
	const rrfK = 60.0
	type rankedProfile struct {
		profile ProviderProfile
		score   float64
		order   int
	}
	byID := make(map[string]rankedProfile, len(primary)+len(secondary))
	for rank, profile := range primary {
		byID[profile.ProviderID] = rankedProfile{profile: profile, score: 1 / (rrfK + float64(rank+1)), order: rank}
	}
	for rank, profile := range secondary {
		item, ok := byID[profile.ProviderID]
		if !ok {
			item = rankedProfile{profile: profile, order: len(primary) + rank}
		}
		item.score += 1 / (rrfK + float64(rank+1))
		byID[profile.ProviderID] = item
	}
	result := make([]rankedProfile, 0, len(byID))
	for _, item := range byID {
		result = append(result, item)
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].score == result[j].score {
			if result[i].order == result[j].order {
				return result[i].profile.ProviderID < result[j].profile.ProviderID
			}
			return result[i].order < result[j].order
		}
		return result[i].score > result[j].score
	})
	out := make([]ProviderProfile, 0, len(result))
	for _, item := range result {
		out = append(out, item.profile)
	}
	return out
}

// boundedProviderEvidenceUnion keeps the selector's dense and lexical top
// windows available to the cross-encoder. The dense index still ranks every
// eligible provider before this function is called; this boundary only limits
// the expensive cross-provider judging step. A dense model is useful for
// paraphrase, while exact identifiers and declaration words are often the
// strongest signal for implementation queries. Dropping either window before
// the cross-encoder makes that signal unrecoverable.
func boundedProviderEvidenceUnion(semantic, lexical []ProviderProfile, width int) []ProviderProfile {
	if width <= 0 {
		width = defaultMaxFanoutWidth
	}
	if len(semantic) > width {
		semantic = semantic[:width]
	}
	if len(lexical) > width {
		lexical = lexical[:width]
	}
	return fuseProviderRankings(semantic, lexical)
}

// guardedSemanticProviderSelection treats the lexical shortlist as a safety
// floor for a semantic experiment. A dense candidate may replace one lexical
// leaf only when it is within the bounded fan-out window of both the all-leaf
// semantic ranking and the cross-encoder choices. When no replacement is admissible,
// the cross-encoder order of the lexical floor is retained and omitted lexical
// leaves are appended in lexical order. This makes the guarded arm equivalent
// to the incumbent whenever semantic evidence is not strong enough, instead
// of silently replacing a judged order with raw lexical order. The policy is
// generic and operates only on registry-derived rankings; it does not encode
// provider knowledge.
func guardedSemanticProviderSelection(query string, picked, semantic, lexical []ProviderProfile, width int, rerankScores map[string]float64) []ProviderProfile {
	if width <= 0 {
		width = defaultMaxFanoutWidth
	}
	if len(lexical) > width {
		lexical = lexical[:width]
	}
	if len(lexical) == 0 {
		return append([]ProviderProfile(nil), picked...)
	}
	lexicalIDs := make(map[string]struct{}, len(lexical))
	semanticRank := make(map[string]int, len(semantic))
	for rank, profile := range semantic {
		semanticRank[profile.ProviderID] = rank
	}
	for _, profile := range lexical {
		lexicalIDs[profile.ProviderID] = struct{}{}
	}
	const (
		maxSemanticRank = 1
		maxPickedRank   = 2
	)
	var replacement *ProviderProfile
	for rank, profile := range picked {
		if rank >= maxPickedRank {
			break
		}
		if _, alreadyCovered := lexicalIDs[profile.ProviderID]; alreadyCovered {
			continue
		}
		if denseRank, ok := semanticRank[profile.ProviderID]; ok && denseRank < maxSemanticRank {
			if !semanticCandidateBeatsLexicalFloor(query, profile, lexical, rerankScores) {
				continue
			}
			candidate := profile
			replacement = &candidate
			break
		}
	}
	ordered := make([]ProviderProfile, 0, len(lexical))
	seen := make(map[string]struct{}, len(lexical))
	for _, profile := range picked {
		if replacement != nil && profile.ProviderID == replacement.ProviderID {
			ordered = append(ordered, profile)
			seen[profile.ProviderID] = struct{}{}
			continue
		}
		if _, ok := lexicalIDs[profile.ProviderID]; !ok {
			continue
		}
		if _, alreadyAdded := seen[profile.ProviderID]; alreadyAdded {
			continue
		}
		if len(ordered) >= width {
			break
		}
		ordered = append(ordered, profile)
		seen[profile.ProviderID] = struct{}{}
	}
	for _, profile := range lexical {
		if len(ordered) >= width {
			break
		}
		if _, ok := seen[profile.ProviderID]; ok {
			continue
		}
		ordered = append(ordered, profile)
		seen[profile.ProviderID] = struct{}{}
	}
	return ordered
}

// semanticCandidateBeatsLexicalFloor makes the replacement gate evidence
// based rather than rank based alone. The cross-encoder scores every provider
// candidate pointwise, so a semantic candidate must score strictly above the
// complete lexical safety floor before it can displace any lexical leaf. The
// candidate must also be at least as lexically compatible as the weakest
// floor member. This prevents a semantically attractive but lexically
// unrelated provider (for example a memory corpus for a documentation query)
// from displacing a concrete lexical signal. The relative floor is query-sized
// rather than an absolute term-count threshold, so short and long queries share
// the same policy.
// A nil score map preserves the deterministic in-process test seam for
// rerankers that only return an ordering; production rerankers always provide
// the map through rerankProviderCandidates.
func semanticCandidateBeatsLexicalFloor(query string, candidate ProviderProfile, lexical []ProviderProfile, rerankScores map[string]float64) bool {
	if len(rerankScores) == 0 {
		return lexicalCompatibilityWithFloor(query, candidate, lexical)
	}
	candidateScore, ok := rerankScores[candidate.ProviderID]
	if !ok {
		return false
	}
	for _, profile := range lexical {
		if score, ok := rerankScores[profile.ProviderID]; !ok || candidateScore <= score {
			return false
		}
	}
	return lexicalCompatibilityWithFloor(query, candidate, lexical)
}

func lexicalCompatibilityWithFloor(query string, candidate ProviderProfile, lexical []ProviderProfile) bool {
	if strings.TrimSpace(query) == "" || len(lexical) == 0 {
		return true
	}
	floorScore := lexicalDescriptionScore(query, lexical[len(lexical)-1])
	candidateScore := lexicalDescriptionScore(query, candidate)
	if candidateScore >= floorScore {
		return true
	}
	// A one-term lexical deficit can still be a useful semantic discovery, but
	// only when it agrees with a corpus type already represented by the safety
	// floor. This keeps a cross-type semantic jump from erasing an exact code,
	// document, command, or record signal while allowing a better leaf within
	// the same broad corpus family to enter the bounded fan-out.
	if candidateScore+1 < floorScore || strings.TrimSpace(candidate.Type) == "" {
		return false
	}
	for _, profile := range lexical {
		if strings.EqualFold(strings.TrimSpace(candidate.Type), strings.TrimSpace(profile.Type)) {
			return true
		}
	}
	return false
}

func providerCandidates(profiles []ProviderProfile) []*routingv1.SearchHit {
	candidates := make([]*routingv1.SearchHit, 0, len(profiles))
	for _, profile := range profiles {
		candidates = append(candidates, &routingv1.SearchHit{
			Id:      profile.ProviderID,
			Type:    profile.Type,
			Title:   profile.Group,
			Path:    profile.ProviderID,
			Snippet: profile.Description,
		})
	}
	return candidates
}

func rerankProviderCandidates(ctx context.Context, query string, profiles []ProviderProfile, reranker Reranker) ([]ProviderProfile, string, map[string]float64, error) {
	if reranker == nil {
		return nil, "", nil, fmt.Errorf("cross-encoder unavailable")
	}
	hits := providerCandidates(profiles)
	rank := func(preference string) ([]*routingv1.SearchHit, error) {
		if preferred, ok := reranker.(interface {
			RerankWithPreference(context.Context, string, []*routingv1.SearchHit, string) ([]*routingv1.SearchHit, error)
		}); ok {
			return preferred.RerankWithPreference(ctx, query, hits, preference)
		}
		return reranker.Rerank(ctx, query, hits)
	}
	selectorLeg := func(fallback string) string {
		if named, ok := reranker.(interface{ ActiveName(context.Context) string }); ok {
			return normalizeSelectorLeg(named.ActiveName(ctx), fallback)
		}
		return fallback
	}
	var ranked []*routingv1.SearchHit
	var err error
	if preferred, ok := reranker.(interface {
		RerankWithPreference(context.Context, string, []*routingv1.SearchHit, string) ([]*routingv1.SearchHit, error)
	}); ok {
		ranked, err = preferred.RerankWithPreference(ctx, query, hits, aisearch.RerankPreferenceCrossEncoderRequired)
	} else {
		ranked, err = rank(aisearch.RerankPreferenceCrossEncoderRequired)
	}
	if err != nil {
		crossEncoderErr := err
		// The required call intentionally probes only the cross-encoder. A
		// second preferred call lets the shared chain select its LLM leg while
		// preserving a deterministic lexical escape hatch when both are down.
		ranked, err = rank(aisearch.RerankPreferenceCrossEncoderPreferred)
		if err != nil {
			return nil, "", nil, fmt.Errorf("cross-encoder: %v; llm: %w", crossEncoderErr, err)
		}
		selectorLeg = func(fallback string) string {
			if named, ok := reranker.(interface{ ActiveName(context.Context) string }); ok {
				return normalizeSelectorLeg(named.ActiveName(ctx), fallback)
			}
			return fallback
		}
		ordered := orderProviderProfiles(ranked, profiles)
		if len(ordered) == 0 {
			return nil, "", nil, fmt.Errorf("llm provider pick returned no provider choice")
		}
		return ordered, selectorLeg(selectorLegLLM), rerankScoresByProviderID(ranked), nil
	}
	ordered := orderProviderProfiles(ranked, profiles)
	if len(ordered) == 0 {
		return nil, "", nil, fmt.Errorf("cross-encoder provider pick returned no provider choice")
	}
	return ordered, selectorLeg(selectorLegCrossEncoder), rerankScoresByProviderID(ranked), nil
}

func rerankScoresByProviderID(ranked []*routingv1.SearchHit) map[string]float64 {
	scores := make(map[string]float64, len(ranked))
	for _, hit := range ranked {
		if hit == nil || strings.TrimSpace(hit.GetId()) == "" {
			continue
		}
		scores[hit.GetId()] = hit.GetRerankScore()
	}
	return scores
}

func orderProviderProfiles(ranked []*routingv1.SearchHit, profiles []ProviderProfile) []ProviderProfile {
	byID := make(map[string]ProviderProfile, len(profiles))
	for _, profile := range profiles {
		byID[profile.ProviderID] = profile
	}
	ordered := make([]ProviderProfile, 0, len(ranked))
	for _, hit := range ranked {
		if hit == nil {
			continue
		}
		if profile, ok := byID[hit.GetId()]; ok {
			ordered = append(ordered, profile)
		}
	}
	return ordered
}

func normalizeSelectorLeg(name, fallback string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	switch {
	case strings.HasPrefix(name, "cross"):
		return selectorLegCrossEncoder
	case strings.HasPrefix(name, "llm"):
		return selectorLegLLM
	default:
		return fallback
	}
}

func queryLooksWebShaped(query string) bool {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return false
	}
	for _, projectTerm := range []string{"project", "repository", "repo", "code", "configuration", "config", "scenario", "cli", "api", "retry logic", "where is", "which command"} {
		if strings.Contains(q, projectTerm) {
			return false
		}
	}
	for _, externalTerm := range []string{"latest", "newest", "current", "version", "release", "releases", "feature", "features", "current events", "vendor", "product", "news", "what changed"} {
		if strings.Contains(q, externalTerm) {
			return true
		}
	}
	return false
}

func strategyHasStage(strategy RetrievalStrategy, kind StageKind) bool {
	for _, stage := range strategy.Stages {
		if stage.Kind == kind {
			return true
		}
	}
	return false
}

func strategyIntParam(strategy RetrievalStrategy, kind StageKind, key string, fallback int) int {
	for _, stage := range strategy.Stages {
		if stage.Kind != kind || stage.Params == nil {
			continue
		}
		switch value := stage.Params[key].(type) {
		case float64:
			if value >= 1 && value <= 128 {
				return int(value)
			}
		case int:
			if value >= 1 && value <= 128 {
				return value
			}
		}
	}
	return fallback
}

func strategyStringParam(strategy RetrievalStrategy, kind StageKind, key, fallback string) string {
	for _, stage := range strategy.Stages {
		if stage.Kind != kind || stage.Params == nil {
			continue
		}
		if value, ok := stage.Params[key].(string); ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return fallback
}
