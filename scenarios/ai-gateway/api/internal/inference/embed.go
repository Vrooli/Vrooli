package inference

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"strings"

	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"

	"ai-gateway/internal/providers"
	"ai-gateway/internal/rolecatalog"
)

// Embed answers every text from one candidate. A candidate is resolved and
// authorized once per request, never once per text, and all texts are
// embedded by the same candidate so one response never mixes vector spaces.
// The texts travel in as few resource calls as the role's batch bounds allow;
// the resource still admits and verifies each call itself.
func (r *ResourceRepository) Embed(ctx context.Context, request EmbeddingRequest) (EmbeddingResult, error) {
	role, definition, err := r.embeddingDefinition(request)
	if err != nil {
		return EmbeddingResult{}, err
	}
	if definition.Embedding == nil {
		return EmbeddingResult{}, fmt.Errorf("%w: inference role %q declares no embedding contract", ErrUnavailable, role)
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

func (r *ResourceRepository) embedWithCandidate(ctx context.Context, request EmbeddingRequest, definition rolecatalog.InferenceRole, candidate rolecatalog.Candidate) (EmbeddingResult, string, bool) {
	mark := r.adapters[strings.ToLower(strings.TrimSpace(candidate.Provider))].FactMark()
	prepared, failure, ok := r.prepareEmbeddingCandidate(ctx, request, candidate)
	if !ok {
		return EmbeddingResult{}, failure, false
	}
	items, err := r.embedTexts(ctx, request, definition, candidate, prepared)
	if err != nil && retryableWithFreshFacts(ctx, err) && prepared.adapter.InvalidateFacts(mark) {
		// The authorization reused resource facts read before this request,
		// and the resource refuses a call whose expected evidence no longer
		// matches its current policy, rights context or model. Plan once more
		// from fresh facts before giving up on the candidate.
		if prepared, failure, ok = r.prepareEmbeddingCandidate(ctx, request, candidate); !ok {
			return EmbeddingResult{}, failure, false
		}
		items, err = r.embedTexts(ctx, request, definition, candidate, prepared)
	}
	if err != nil {
		prepared.adapter.InvalidateFacts(providers.AnyFact)
		log.Printf("embedding candidate failed provider=%s role=%s texts=%d: %s", candidate.Provider, candidate.ResourceRole, len(request.Texts), providers.FailureDetail(err))
		return EmbeddingResult{}, err.Error(), false
	}
	result := EmbeddingResult{Provider: prepared.resolved.Provider, Model: prepared.resolved.Model, Normalization: definition.Embedding.Normalization}
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
// inference, once per request. Resolution reads the resource's facts through
// the adapter's runner, which may reuse them for a bounded time: the resource
// command re-verifies the exact policy file, model digest and rights evidence
// it is told to expect on every call, so reused facts can only make a call
// fail closed, never admit one the current policy would refuse.
func (r *ResourceRepository) prepareEmbeddingCandidate(ctx context.Context, request EmbeddingRequest, candidate rolecatalog.Candidate) (embeddingCandidate, string, bool) {
	adapter, available := r.adapters[strings.ToLower(strings.TrimSpace(candidate.Provider))]
	if !available {
		return embeddingCandidate{}, candidate.Provider + ": adapter unavailable", false
	}
	resolved, err := adapter.ResolveRole(ctx, candidate.ResourceRole)
	if err != nil {
		return embeddingCandidate{}, err.Error(), false
	}
	if !profileAdmitsRequest(request.Profile, request.PrivacyClass, resolved.Locality) {
		return embeddingCandidate{}, fmt.Sprintf("%s: resolved locality %q violates %s", candidate.Provider, resolved.Locality, request.Profile), false
	}
	executionRole, digest, _, err := r.authorizeLocalCandidate(ctx, adapter, candidate.ResourceRole, resolved)
	if err != nil {
		return embeddingCandidate{}, candidate.Provider + ": " + err.Error(), false
	}
	return embeddingCandidate{adapter: adapter, resolved: resolved, role: executionRole, digest: digest}, "", true
}

// retryableWithFreshFacts reports whether a failed embedding call may be the
// resource refusing stale planned evidence. A cancelled request or a call that
// ran out of time is not retried: repeating it would only double the wait.
func retryableWithFreshFacts(ctx context.Context, err error) bool {
	if ctx.Err() != nil || errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return false
	}
	var cmdErr *providers.CommandError
	return !errors.As(err, &cmdErr) || cmdErr.Code != "timeout"
}

// embedTexts runs the request's texts through the candidate one resource call
// per batch, in order. Calls are sequential on purpose: the resource admits
// embedding calls through a shared host semaphore with one slot, so overlapping
// them would only queue processes ahead of every other caller on the machine.
func (r *ResourceRepository) embedTexts(ctx context.Context, request EmbeddingRequest, definition rolecatalog.InferenceRole, candidate rolecatalog.Candidate, prepared embeddingCandidate) ([]embeddingItem, error) {
	items := make([]embeddingItem, 0, len(request.Texts))
	for _, batch := range embeddingBatches(request.Texts, *definition.Embedding) {
		batchItems, err := r.embedBatch(ctx, request, definition, candidate, prepared, batch)
		if err != nil {
			return nil, err
		}
		items = append(items, batchItems...)
	}
	return items, nil
}

// embeddingBatches splits texts into consecutive batches that each fit the
// role's bounds on inputs and encoded request bytes. A text too large for any
// batch travels alone, and the resource rejects it with its own reason.
func embeddingBatches(texts []string, contract rolecatalog.RoleEmbedding) [][]string {
	const envelopeBytes = len(`{"inputs":[]}`)
	var (
		batches [][]string
		current []string
		bytes   = envelopeBytes
	)
	for _, text := range texts {
		encoded, _ := json.Marshal(text) // a string always encodes
		size := len(encoded) + 1         // the separating comma
		if len(current) > 0 && (len(current) >= contract.MaxBatchInputs || bytes+size > contract.MaxBatchBytes) {
			batches = append(batches, current)
			current, bytes = nil, envelopeBytes
		}
		current = append(current, text)
		bytes += size
	}
	if len(current) > 0 {
		batches = append(batches, current)
	}
	return batches
}

func (r *ResourceRepository) embedBatch(ctx context.Context, request EmbeddingRequest, definition rolecatalog.InferenceRole, candidate rolecatalog.Candidate, prepared embeddingCandidate, texts []string) ([]embeddingItem, error) {
	execution, err := executeTraced(ctx, prepared.adapter, prepared.resolved, providers.ExecutionRequest{
		Kind:                         sharedv1.RequestKind_REQUEST_KIND_TEXT_EMBEDDING,
		Role:                         candidate.ResourceRole,
		ExpectedModel:                prepared.role.Model,
		ExpectedArtifactDigest:       prepared.digest,
		ExpectedRightsDecisionSHA256: prepared.role.RightsDecisionSHA256,
		ExpectedRightsEvaluatedAt:    prepared.role.RightsEvaluatedAt,
		ExpectedRightsReceiptSHA256:  prepared.role.RightsReceiptSHA256,
		ExpectedPolicyFileSHA256:     prepared.role.PolicyFileSHA256,
		Profile:                      request.Profile,
		InputTexts:                   texts,
		Timeout:                      definition.Timeout(),
	})
	if err != nil {
		return nil, err
	}
	vectors, err := decodeEmbeddingResponse(candidate.Provider, execution.OutputText, len(texts), definition.Embedding.Normalization)
	if err != nil {
		return nil, err
	}
	items := make([]embeddingItem, len(texts))
	for index, vector := range vectors {
		items[index] = embeddingItem{vector: vector, inputTokens: estimateTokens(texts[index])}
	}
	return items, nil
}

// normalizationTolerance is how far a vector's length may sit from one before
// a role that declares l2 vectors is considered to have broken its contract.
const normalizationTolerance = 1e-3

// decodeEmbeddingResponse reads the one vector per text a resource call must
// return and checks each against the role's declared scale.
func decodeEmbeddingResponse(provider, raw string, want int, normalization string) ([][]float64, error) {
	var response struct {
		Embedding  []float64   `json:"embedding"`
		Embeddings [][]float64 `json:"embeddings"`
	}
	if err := json.Unmarshal([]byte(raw), &response); err != nil {
		return nil, fmt.Errorf("decode %s embedding: %w", provider, err)
	}
	vectors := response.Embeddings
	if len(vectors) == 0 && len(response.Embedding) > 0 {
		vectors = [][]float64{response.Embedding}
	}
	if len(vectors) != want {
		return nil, fmt.Errorf("%s returned %d embedding vectors for %d texts", provider, len(vectors), want)
	}
	for index, vector := range vectors {
		if len(vector) == 0 {
			return nil, fmt.Errorf("%s embedding %d contained no vector", provider, index)
		}
		if normalization == rolecatalog.NormalizationL2 {
			if length := vectorLength(vector); math.Abs(length-1) > normalizationTolerance {
				return nil, fmt.Errorf("embedding_not_normalized: %s returned a vector of length %.4f for a role that declares l2", provider, length)
			}
		}
	}
	return vectors, nil
}

func vectorLength(vector []float64) float64 {
	var sum float64
	for _, value := range vector {
		sum += value * value
	}
	return math.Sqrt(sum)
}
