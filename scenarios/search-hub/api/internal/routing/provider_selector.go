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

func rerankProviderCandidates(ctx context.Context, query string, profiles []ProviderProfile, reranker Reranker) ([]ProviderProfile, error) {
	if reranker == nil {
		return nil, fmt.Errorf("cross-encoder unavailable")
	}
	hits := providerCandidates(profiles)
	var ranked []*routingv1.SearchHit
	var err error
	if preferred, ok := reranker.(interface {
		RerankWithPreference(context.Context, string, []*routingv1.SearchHit, string) ([]*routingv1.SearchHit, error)
	}); ok {
		ranked, err = preferred.RerankWithPreference(ctx, query, hits, aisearch.RerankPreferenceCrossEncoderRequired)
	} else {
		// Deterministic fakes and third-party adapters may only implement the
		// narrow Reranker seam. Production's shared adapter honors the required
		// preference above, so an LLM fallback cannot silently become a router.
		ranked, err = reranker.Rerank(ctx, query, hits)
	}
	if err != nil {
		return nil, err
	}
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
	if len(ordered) == 0 {
		return nil, fmt.Errorf("cross-encoder returned no provider choice")
	}
	return ordered, nil
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

func profileDescribesExternalFacts(profile ProviderProfile) bool {
	description := strings.ToLower(profile.Description)
	for _, marker := range []string{"external world", "external-world", "current events", "third-party", "software release", "outside-world factual"} {
		if strings.Contains(description, marker) {
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
