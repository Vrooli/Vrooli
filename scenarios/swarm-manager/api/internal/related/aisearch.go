package related

import (
	"context"
	"fmt"

	"swarm-manager/internal/aisearch"
)

// AISearchSimilarity adapts the semantic service without making aisearch
// depend on the related package.
type AISearchSimilarity struct{ service *aisearch.Service }

func NewAISearchSimilarity(service *aisearch.Service) *AISearchSimilarity {
	return &AISearchSimilarity{service}
}
func (a *AISearchSimilarity) Similar(ctx context.Context, target TargetRef, limit int) ([]Entity, bool, error) {
	entity := aisearch.EntityBacklog
	if target.Kind == TargetGoal {
		entity = aisearch.EntityGoal
	}
	response, err := a.service.SimilarTo(ctx, aisearch.SimilarTarget{Entity: entity, BacklogKind: target.BacklogKind, Name: target.Name}, limit)
	if err != nil {
		return nil, true, err
	}
	if response.Degraded {
		return nil, true, nil
	}
	out := make([]Entity, 0, len(response.Results))
	for _, r := range response.Results {
		kind := EntityKind(r.Entity)
		key := r.ID
		title := key
		status := ""
		archived := false
		if r.Payload != nil {
			if v, ok := r.Payload["record_id"].(string); ok && v != "" {
				key = v
			}
			if v, ok := r.Payload["name"].(string); ok && v != "" {
				key = v
			}
			if v, ok := r.Payload["title"].(string); ok && v != "" {
				title = v
			}
			if v, ok := r.Payload["status"].(string); ok {
				status = v
			}
			if v, ok := r.Payload["outcome"].(string); ok && status == "" {
				status = v
			}
			if v, ok := r.Payload["archived"].(bool); ok {
				archived = v
			}
		}
		if kind != EntityBacklog && kind != EntityGoal && kind != EntityRecord {
			return nil, true, fmt.Errorf("unknown similar entity %q", r.Entity)
		}
		out = append(out, Entity{Kind: kind, Key: key, Title: title, Status: status, Archived: archived, Reasons: []string{fmt.Sprintf("%d%% similar", r.ScorePercent)}, ScorePercent: r.ScorePercent})
	}
	return out, false, nil
}
