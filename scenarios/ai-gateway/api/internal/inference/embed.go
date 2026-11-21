package inference

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"

	"ai-gateway/internal/providers"
	"ai-gateway/internal/rolecatalog"
)

// Embed answers every text from one candidate. A candidate is resolved and
// authorized once per request, never once per text, and all texts are
// embedded by the same candidate so one response never mixes vector spaces.
// The texts of a request run with bounded concurrency; the resource still
// admits and verifies each call itself.
func (r *ResourceRepository) Embed(ctx context.Context, request EmbeddingRequest) (EmbeddingResult, error) {
	role, definition, err := r.embeddingDefinition(request)
	if err != nil {
		return EmbeddingResult{}, err
	}
	failures := make([]string, 0, len(definition.Candidates))
	for _, candidate := range orderedCandidates(definition.Candidates, r.adapters, request.Profile) {
		result, failure, ok := r.embedWithCandidate(ctx, request, definition, candidate)
		if ok {
			return result, nil
		}
		failures = append(failures, failure)
	}
	return EmbeddingResult{}, fmt.Errorf("%w: no embedding candidate completed for %q (%s)", ErrUnavailable, role, strings.Join(failures, "; "))
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
	inputTokens, costMicros int64
}

// embeddingCandidate is a candidate that passed locality and rights
// authorization, with everything the resource command is told to expect.
type embeddingCandidate struct {
	adapter  providers.Adapter
	resolved providers.ResolvedRole
	role     providers.ResolvedRole
	digest   string
}

// maxEmbeddingConcurrency bounds the resource commands one request runs at
// once. The resource's own host semaphore still caps what the daemon sees.
const maxEmbeddingConcurrency = 4

func (r *ResourceRepository) embedWithCandidate(ctx context.Context, request EmbeddingRequest, definition rolecatalog.InferenceRole, candidate rolecatalog.Candidate) (EmbeddingResult, string, bool) {
	prepared, cached, failure, ok := r.prepareEmbeddingCandidate(ctx, request, candidate)
	if !ok {
		return EmbeddingResult{}, failure, false
	}
	items, err := r.embedTexts(ctx, request, definition, candidate, prepared)
	if err != nil && cached {
		// A cached authorization can be stale: the policy, rights context or
		// model may have changed since it was made, and the resource refuses
		// a call whose expected evidence no longer matches. Drop it and run
		// the full authorization once before giving up on the candidate.
		r.forgetEmbeddingCandidate(candidate)
		if prepared, _, failure, ok = r.prepareEmbeddingCandidate(ctx, request, candidate); !ok {
			return EmbeddingResult{}, failure, false
		}
		items, err = r.embedTexts(ctx, request, definition, candidate, prepared)
	}
	if err != nil {
		r.forgetEmbeddingCandidate(candidate)
		return EmbeddingResult{}, err.Error(), false
	}
	result := EmbeddingResult{Provider: prepared.resolved.Provider, Model: prepared.resolved.Model}
	for _, item := range items {
		if result.Dimension != 0 && result.Dimension != len(item.vector) {
			return EmbeddingResult{}, "embedding_dimension_mismatch", false
		}
		result.Dimension = len(item.vector)
		result.Vectors = append(result.Vectors, item.vector)
		result.InputTokens += item.inputTokens
		result.CostMicros += item.costMicros
		result.InputTokensProvenance = combineProvenance(result.InputTokensProvenance, inferencev1.UsageProvenance_USAGE_PROVENANCE_GATEWAY_ESTIMATE)
		result.CostProvenance = inferencev1.UsageProvenance_USAGE_PROVENANCE_UNKNOWN
	}
	return result, "", true
}

// prepareEmbeddingCandidate resolves a candidate and authorizes it for local
// inference. The outcome is reused for authorizationTTL: the resource command
// re-verifies the exact policy file, model digest and rights evidence it is
// told to expect on every call, so a reused authorization can only make a call
// fail closed, never admit one the current policy would refuse.
func (r *ResourceRepository) prepareEmbeddingCandidate(ctx context.Context, request EmbeddingRequest, candidate rolecatalog.Candidate) (embeddingCandidate, bool, string, bool) {
	key := embeddingCacheKey(request, candidate)
	if entry, ok := r.cachedEmbeddingCandidate(key); ok {
		return entry, true, "", true
	}
	adapter, available := r.adapters[strings.ToLower(strings.TrimSpace(candidate.Provider))]
	if !available {
		return embeddingCandidate{}, false, candidate.Provider + ": adapter unavailable", false
	}
	resolved, err := adapter.ResolveRole(ctx, candidate.ResourceRole)
	if err != nil {
		return embeddingCandidate{}, false, err.Error(), false
	}
	if !profileAdmitsRequest(request.Profile, request.PrivacyClass, resolved.Locality) {
		return embeddingCandidate{}, false, fmt.Sprintf("%s: resolved locality %q violates %s", candidate.Provider, resolved.Locality, request.Profile), false
	}
	executionRole, digest, _, err := r.authorizeLocalCandidate(ctx, adapter, candidate.ResourceRole, resolved)
	if err != nil {
		return embeddingCandidate{}, false, candidate.Provider + ": " + err.Error(), false
	}
	entry := embeddingCandidate{adapter: adapter, resolved: resolved, role: executionRole, digest: digest}
	r.storeEmbeddingCandidate(key, entry)
	return entry, false, "", true
}

func (r *ResourceRepository) embedTexts(ctx context.Context, request EmbeddingRequest, definition rolecatalog.InferenceRole, candidate rolecatalog.Candidate, prepared embeddingCandidate) ([]embeddingItem, error) {
	items := make([]embeddingItem, len(request.Texts))
	workers := min(maxEmbeddingConcurrency, len(request.Texts))
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		firstErr error
		next     int
	)
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				mu.Lock()
				index := next
				next++
				failed := firstErr != nil
				mu.Unlock()
				if failed || index >= len(request.Texts) {
					return
				}
				item, err := r.embedOne(runCtx, request, definition, candidate, prepared, request.Texts[index])
				if err != nil {
					mu.Lock()
					if firstErr == nil {
						firstErr = err
						cancel()
					}
					mu.Unlock()
					return
				}
				items[index] = item
			}
		}()
	}
	wg.Wait()
	if firstErr != nil {
		return nil, firstErr
	}
	return items, nil
}

func (r *ResourceRepository) embedOne(ctx context.Context, request EmbeddingRequest, definition rolecatalog.InferenceRole, candidate rolecatalog.Candidate, prepared embeddingCandidate, text string) (embeddingItem, error) {
	execution, err := prepared.adapter.Execute(ctx, providers.ExecutionRequest{
		Kind:                         sharedv1.RequestKind_REQUEST_KIND_TEXT_EMBEDDING,
		Role:                         candidate.ResourceRole,
		ExpectedModel:                prepared.role.Model,
		ExpectedArtifactDigest:       prepared.digest,
		ExpectedRightsDecisionSHA256: prepared.role.RightsDecisionSHA256,
		ExpectedRightsEvaluatedAt:    prepared.role.RightsEvaluatedAt,
		ExpectedRightsReceiptSHA256:  prepared.role.RightsReceiptSHA256,
		ExpectedPolicyFileSHA256:     prepared.role.PolicyFileSHA256,
		Profile:                      request.Profile,
		InputText:                    text,
		Timeout:                      definition.Timeout(),
	})
	if err != nil {
		return embeddingItem{}, err
	}
	vector, inputTokens, costMicros, err := decodeEmbeddingResponse(candidate.Provider, execution.OutputText, text)
	if err != nil {
		return embeddingItem{}, err
	}
	return embeddingItem{vector: vector, inputTokens: inputTokens, costMicros: costMicros}, nil
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
