package inference

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"

	"ai-gateway/internal/providers"
	"ai-gateway/internal/rolecatalog"
)

func (r *ResourceRepository) Embed(ctx context.Context, request EmbeddingRequest) (EmbeddingResult, error) {
	role, definition, err := r.embeddingDefinition(request)
	if err != nil {
		return EmbeddingResult{}, err
	}
	var result EmbeddingResult
	for _, text := range request.Texts {
		item, err := r.embedText(ctx, request, role, definition, text)
		if err != nil {
			return EmbeddingResult{}, err
		}
		if result.Dimension != 0 && result.Dimension != len(item.vector) {
			return EmbeddingResult{}, errors.New("embedding_dimension_mismatch")
		}
		if result.Dimension == 0 {
			result.Dimension = len(item.vector)
		}
		result.Vectors = append(result.Vectors, item.vector)
		result.Provider, result.Model = item.provider, item.model
		result.InputTokens += item.inputTokens
		result.CostMicros += item.costMicros
		result.InputTokensProvenance = combineProvenance(result.InputTokensProvenance, inferencev1.UsageProvenance_USAGE_PROVENANCE_GATEWAY_ESTIMATE)
		result.CostProvenance = inferencev1.UsageProvenance_USAGE_PROVENANCE_UNKNOWN
	}
	return result, nil
}

func (r *ResourceRepository) embeddingDefinition(request EmbeddingRequest) (string, rolecatalog.InferenceRole, error) {
	role := strings.TrimSpace(request.Role)
	if request.Profile != sharedv1.Profile_PROFILE_LOCAL_ONLY {
		return "", rolecatalog.InferenceRole{}, fmt.Errorf("%w: typed embeddings require an explicit local-only profile", ErrUnavailable)
	}
	if err := validateEmbeddingPrivacy(request.PrivacyClass, request.PrivacyProvenance); err != nil {
		return "", rolecatalog.InferenceRole{}, fmt.Errorf("%w: invalid embedding privacy contract: %v", ErrUnavailable, err)
	}
	definition, ok := r.catalog.Roles[role]
	if !ok {
		return "", rolecatalog.InferenceRole{}, fmt.Errorf("%w: inference role %q is not declared", ErrUnavailable, role)
	}
	if len(request.Texts) == 0 {
		return "", rolecatalog.InferenceRole{}, errors.New("embedding texts are required")
	}
	return role, definition, nil
}

type embeddingItem struct {
	vector                  []float64
	provider, model         string
	inputTokens, costMicros int64
}

func (r *ResourceRepository) embedText(ctx context.Context, request EmbeddingRequest, role string, definition rolecatalog.InferenceRole, text string) (embeddingItem, error) {
	failures := make([]string, 0, len(definition.Candidates))
	for _, candidate := range orderedCandidates(definition.Candidates, r.adapters, request.Profile) {
		item, failure, ok := r.embedCandidate(ctx, request, definition, candidate, text)
		if !ok {
			failures = append(failures, failure)
			continue
		}
		return item, nil
	}
	return embeddingItem{}, fmt.Errorf("%w: no embedding candidate completed for %q (%s)", ErrUnavailable, role, strings.Join(failures, "; "))
}

func (r *ResourceRepository) embedCandidate(ctx context.Context, request EmbeddingRequest, definition rolecatalog.InferenceRole, candidate rolecatalog.Candidate, text string) (embeddingItem, string, bool) {
	adapter, available := r.adapters[strings.ToLower(strings.TrimSpace(candidate.Provider))]
	if !available {
		return embeddingItem{}, candidate.Provider + ": adapter unavailable", false
	}
	resolved, err := adapter.ResolveRole(ctx, candidate.ResourceRole)
	if err != nil {
		return embeddingItem{}, err.Error(), false
	}
	if !profileAdmitsRequest(request.Profile, request.PrivacyClass, resolved.Locality) {
		return embeddingItem{}, fmt.Sprintf("%s: resolved locality %q violates %s", candidate.Provider, resolved.Locality, request.Profile), false
	}
	executionRole, expectedArtifactDigest, _, err := r.authorizeLocalCandidate(ctx, adapter, candidate.ResourceRole, resolved)
	if err != nil {
		return embeddingItem{}, candidate.Provider + ": " + err.Error(), false
	}
	execution, err := adapter.Execute(ctx, providers.ExecutionRequest{
		Kind:                         sharedv1.RequestKind_REQUEST_KIND_TEXT_EMBEDDING,
		Role:                         candidate.ResourceRole,
		ExpectedModel:                executionRole.Model,
		ExpectedArtifactDigest:       expectedArtifactDigest,
		ExpectedRightsDecisionSHA256: executionRole.RightsDecisionSHA256,
		ExpectedRightsEvaluatedAt:    executionRole.RightsEvaluatedAt,
		ExpectedRightsReceiptSHA256:  executionRole.RightsReceiptSHA256,
		ExpectedPolicyFileSHA256:     executionRole.PolicyFileSHA256,
		Profile:                      request.Profile,
		InputText:                    text,
		Timeout:                      definition.Timeout(),
	})
	if err != nil {
		return embeddingItem{}, err.Error(), false
	}
	vector, inputTokens, costMicros, err := decodeEmbeddingResponse(candidate.Provider, execution.OutputText, text)
	if err != nil {
		return embeddingItem{}, err.Error(), false
	}
	return embeddingItem{vector: vector, provider: resolved.Provider, model: resolved.Model, inputTokens: inputTokens, costMicros: costMicros}, "", true
}

func decodeEmbeddingResponse(provider, raw, input string) ([]float64, int64, int64, error) {
	var response struct {
		Embedding  []float64   `json:"embedding"`
		Embeddings [][]float64 `json:"embeddings"`
	}
	if err := json.Unmarshal([]byte(raw), &response); err != nil {
		return nil, 0, 0, fmt.Errorf("decode %s embedding: %w", provider, err)
	}
	vector := response.Embedding
	if len(vector) == 0 && len(response.Embeddings) > 0 {
		vector = response.Embeddings[0]
	}
	if len(vector) == 0 {
		return nil, 0, 0, errors.New("embedding response contained no vector")
	}
	return vector, estimateTokens(input), 0, nil
}
