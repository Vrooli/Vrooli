package routing

import (
	"context"
	"fmt"
	"sort"
	"strings"

	aisearch "github.com/vrooli/ai-go/search"
	internalrouting "search-hub/internal/routing"
)

// fleetSemanticDescriptionIndex forwards the complete trusted caller envelope
// to the owner-bound shared store. It does not embed text or choose a recipe;
// those checks and admission happen inside FleetSemanticStore.
type fleetSemanticDescriptionIndex struct {
	store aisearch.FleetSemanticStore
}

var _ internalrouting.ProviderDescriptionIndex = (*fleetSemanticDescriptionIndex)(nil)

func newFleetSemanticDescriptionIndex(store aisearch.FleetSemanticStore) (*fleetSemanticDescriptionIndex, error) {
	if store == nil {
		return nil, fmt.Errorf("fleet semantic store is unavailable")
	}
	return &fleetSemanticDescriptionIndex{store: store}, nil
}

func (i *fleetSemanticDescriptionIndex) Shortlist(ctx context.Context, request internalrouting.ProviderDescriptionQuery, profiles []internalrouting.ProviderProfile) ([]internalrouting.ProviderProfile, internalrouting.DescriptionIndexResult) {
	result := internalrouting.DescriptionIndexResult{Total: len(profiles), Scores: make(map[string]float64)}
	if i == nil || i.store == nil || !validFleetSemanticDescriptionQuery(request) {
		result.Reason = "routing_identity_unavailable"
		return nil, result
	}
	queryResult, err := i.store.QueryActive(ctx, aisearch.FleetSemanticQueryRequest{
		OperationID: request.OperationID,
		TenantID:    request.TenantID,
		ArmID:       request.ArmID,
		RecipeID:    request.RecipeID,
		Text:        request.Text,
		Limit:       request.Limit,
		Attribution: aisearch.FleetSemanticAttribution{
			CallerID:              request.Attribution.CallerID,
			RightsEvidenceURI:     request.Attribution.RightsEvidenceURI,
			RightsEvidenceSHA256:  request.Attribution.RightsEvidenceSHA256,
			QualityEvidenceURI:    request.Attribution.QualityEvidenceURI,
			QualityEvidenceSHA256: request.Attribution.QualityEvidenceSHA256,
		},
	})
	if err != nil {
		result.Reason = "fleet_semantic_unavailable"
		return nil, result
	}
	profileByID := make(map[string]internalrouting.ProviderProfile, len(profiles))
	for _, profile := range profiles {
		profileByID[profile.ProviderID] = profile
	}
	best := make(map[string]float64, len(queryResult.Results))
	for _, hit := range queryResult.Results {
		providerID, ok := hit.Payload["provider_id"].(string)
		if !ok || strings.TrimSpace(providerID) == "" {
			continue
		}
		if _, exists := profileByID[providerID]; !exists {
			continue
		}
		if prior, exists := best[providerID]; !exists || hit.Score > prior {
			best[providerID] = hit.Score
		}
	}
	if len(best) == 0 {
		result.Reason = "fleet_semantic_no_results"
		return nil, result
	}
	type scoredProfile struct {
		profile internalrouting.ProviderProfile
		score   float64
	}
	scored := make([]scoredProfile, 0, len(best))
	for id, score := range best {
		result.Scores[id] = score
		scored = append(scored, scoredProfile{profile: profileByID[id], score: score})
	}
	sort.SliceStable(scored, func(a, b int) bool {
		if scored[a].score == scored[b].score {
			return scored[a].profile.ProviderID < scored[b].profile.ProviderID
		}
		return scored[a].score > scored[b].score
	})
	limit := request.Limit
	if limit <= 0 {
		result.Reason = "routing_identity_unavailable"
		return nil, result
	}
	if len(scored) > limit {
		scored = scored[:limit]
	}
	shortlist := make([]internalrouting.ProviderProfile, 0, len(scored))
	selected := make(map[string]struct{}, len(scored))
	for _, item := range scored {
		shortlist = append(shortlist, item.profile)
		selected[item.profile.ProviderID] = struct{}{}
	}
	for _, profile := range profiles {
		if _, ok := selected[profile.ProviderID]; !ok {
			result.Omitted = append(result.Omitted, profile.ProviderID)
		}
	}
	result.Available = true
	result.Returned = len(shortlist)
	return shortlist, result
}

func validFleetSemanticDescriptionQuery(request internalrouting.ProviderDescriptionQuery) bool {
	return strings.TrimSpace(request.OperationID) != "" &&
		strings.TrimSpace(request.TenantID) != "" &&
		strings.TrimSpace(request.ArmID) != "" &&
		strings.TrimSpace(request.RecipeID) != "" &&
		strings.TrimSpace(request.Text) != "" && request.Limit > 0 &&
		strings.TrimSpace(request.Attribution.CallerID) != "" &&
		strings.TrimSpace(request.Attribution.RightsEvidenceURI) != "" &&
		isSHA256Hex(request.Attribution.RightsEvidenceSHA256) &&
		strings.TrimSpace(request.Attribution.QualityEvidenceURI) != "" &&
		isSHA256Hex(request.Attribution.QualityEvidenceSHA256)
}

func isSHA256Hex(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, r := range value {
		if !(r >= '0' && r <= '9' || r >= 'a' && r <= 'f' || r >= 'A' && r <= 'F') {
			return false
		}
	}
	return true
}
