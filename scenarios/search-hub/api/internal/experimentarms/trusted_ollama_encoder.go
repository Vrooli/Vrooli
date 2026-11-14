package experimentarms

import (
	"context"
	"errors"
	"fmt"
	"math"
	"unicode/utf8"

	ollamapolicy "github.com/vrooli/ai-go/ollama/policy"
	aisearch "github.com/vrooli/ai-go/search"
	"search-hub/internal/routing"
)

const (
	ollamaEmbeddingRole       = "embedding.default"
	ollamaNomicModel          = "nomic-embed-text:latest"
	ollamaNomicManifestSHA256 = "0a109f422b47e3a30ba2b10eca18548e944e8a23073ee3f3e947efcf3c45e59f"
	ollamaNomicModelLayerSHA  = "970aa74c0a90ef7482477cf803618e776e173c007bf957f635f1015bfcfef0e6"
	ollamaNomicLicenseSHA     = "c71d239df91726fc519c6eb72d318ec65820627232b2f796219e87dcf35d0ab4"
	ollamaManagedBuild        = "resource-ollama-gateway/0.1.0+ollama-server/0.30.10"
	ollamaNomicDimensions     = 768
	ollamaNomicContextTokens  = 2048
	ollamaNomicMaxInputRunes  = 1800
	qdrantPeakFixedBytes      = int64(64 << 20)
	qdrantPeakPayloadBound    = int64(4 << 10)
	qdrantPeakSafetyFactor    = int64(4)
)

// TrustedOllamaFleetDescriptionEncoder is the exact Search Hub embedding
// implementation for the digest-pinned Ollama Nomic model. The public
// resource-ollama gateway rechecks the current tag digest and rights before
// every provider call; this object freezes those same owner facts into the
// generation recipe and applies the declared output normalization.
type TrustedOllamaFleetDescriptionEncoder struct {
	recipe   aisearch.EmbeddingRecipe
	embedder aisearch.Embedder
	task     aisearch.TaskEmbedder
}

var _ FleetDescriptionEncoder = (*TrustedOllamaFleetDescriptionEncoder)(nil)

// NewTrustedOllamaFleetDescriptionEncoder resolves embedding.default through
// the resource owner. It does not make an inference request. Construction
// fails closed unless rights, runtime identity, exact artifact digest, and
// dimensions all match the locally installed Nomic manifest evidence.
func NewTrustedOllamaFleetDescriptionEncoder(ctx context.Context, resolver ollamapolicy.Resolver) (*TrustedOllamaFleetDescriptionEncoder, error) {
	resolved, err := resolver.ResolveRole(ctx, ollamaEmbeddingRole)
	if err != nil {
		return nil, fmt.Errorf("resolve trusted Ollama embedding role: %w", err)
	}
	return newTrustedOllamaFleetDescriptionEncoder(resolved, aisearch.NewEmbedderWithPrefixes(resolved.Model, "search_query: ", "search_document: "))
}

func newTrustedOllamaFleetDescriptionEncoder(resolved ollamapolicy.ResolvedRole, embedder aisearch.Embedder) (*TrustedOllamaFleetDescriptionEncoder, error) {
	if embedder == nil {
		return nil, errors.New("trusted Ollama fleet encoder requires the owner embedding client")
	}
	task, ok := embedder.(aisearch.TaskEmbedder)
	if !ok {
		return nil, errors.New("trusted Ollama fleet encoder requires query and document embedding methods")
	}
	if resolved.Role != ollamaEmbeddingRole || resolved.Model != ollamaNomicModel || resolved.EmbeddingDimensions != ollamaNomicDimensions {
		return nil, errors.New("Ollama embedding role does not resolve to the expected Nomic model and 768-dimensional output")
	}
	if !isFleetDigest(resolved.PolicySHA256) {
		return nil, errors.New("Ollama embedding policy response is missing its exact policy digest")
	}
	rights, identity := resolved.Rights, resolved.RuntimeIdentity
	if rights == nil || identity == nil || rights.ArtifactID != "registry.ollama.ai/library/"+ollamaNomicModel || rights.Revision != "ollama-manifest:sha256:"+ollamaNomicManifestSHA256 || rights.SHA256 != ollamaNomicManifestSHA256 || rights.LicenseID != "Apache-2.0" || rights.Classification != "unrestricted" {
		return nil, errors.New("Ollama Nomic model rights are missing or do not bind the exact Apache-licensed installed manifest")
	}
	if rights.Evidence.URL != "https://ollama.com/library/nomic-embed-text:latest" || rights.Evidence.Revision != ollamaNomicManifestSHA256 || rights.Evidence.SHA256 != ollamaNomicLicenseSHA {
		return nil, errors.New("Ollama Nomic rights evidence does not match the verified model page and bundled license layer")
	}
	if identity.Provider != "ollama" || identity.Model != ollamaNomicModel || identity.ArtifactID != rights.ArtifactID || identity.ArtifactRevision != rights.Revision || identity.DigestKind != "ollama_model_contents" || identity.Digest != ollamaNomicManifestSHA256 || identity.Source != "ollama /api/tags" || identity.ObservedAt == "" {
		return nil, errors.New("Ollama Nomic runtime identity is not bound to the expected installed manifest digest")
	}
	recipe, err := aisearch.NewEmbeddingRecipe(aisearch.EmbeddingRecipeInput{
		SchemaVersion: "1", ModelID: ollamaNomicModel,
		ModelRevision:     "ollama-manifest:sha256:" + ollamaNomicManifestSHA256,
		ArtifactSHA256:    ollamaNomicManifestSHA256,
		TokenizerRevision: "gguf-tokenizer-layer:sha256:" + ollamaNomicModelLayerSHA,
		Runtime:           "ollama-resource-gateway", RuntimeBuild: ollamaManagedBuild,
		Precision: "f16", QueryInstruction: "search_query: ", DocumentInstruction: "search_document: ",
		Pooling: "gguf-mean", Normalization: "l2", Dimensions: ollamaNomicDimensions,
		OutputDType: "float32", OutputQuantization: "none",
		Truncation:      "reject-input-over-1800-runes;gguf-context-2048",
		ChunkerVersion:  routing.ProviderDescriptionFleetRecipeVersion,
		ComposerVersion: "nomic-task-prefix/v1+provider-profile-identity/v5",
	})
	if err != nil {
		return nil, fmt.Errorf("construct immutable Ollama embedding recipe: %w", err)
	}
	return &TrustedOllamaFleetDescriptionEncoder{recipe: recipe, embedder: embedder, task: task}, nil
}

func (e *TrustedOllamaFleetDescriptionEncoder) Recipe() aisearch.EmbeddingRecipe { return e.recipe }

func (e *TrustedOllamaFleetDescriptionEncoder) Available(ctx context.Context) bool {
	return e != nil && e.embedder != nil && e.embedder.Available(ctx)
}

func (e *TrustedOllamaFleetDescriptionEncoder) Embed(ctx context.Context, text string) ([]float64, error) {
	return e.embed(ctx, text, e.embedder.Embed)
}

func (e *TrustedOllamaFleetDescriptionEncoder) EmbedQuery(ctx context.Context, text string) ([]float64, error) {
	return e.embed(ctx, text, e.task.EmbedQuery)
}

func (e *TrustedOllamaFleetDescriptionEncoder) EmbedDocument(ctx context.Context, text string) ([]float64, error) {
	return e.embed(ctx, text, e.task.EmbedDocument)
}

func (e *TrustedOllamaFleetDescriptionEncoder) embed(ctx context.Context, text string, call func(context.Context, string) ([]float64, error)) ([]float64, error) {
	if e == nil || e.embedder == nil || e.task == nil || call == nil {
		return nil, errors.New("trusted Ollama fleet encoder is unavailable")
	}
	if utf8.RuneCountInString(text) > ollamaNomicMaxInputRunes {
		return nil, fmt.Errorf("provider-description input exceeds the %d-rune cap for the 2048-token Nomic context", ollamaNomicMaxInputRunes)
	}
	vector, err := call(ctx, text)
	if err != nil {
		return nil, err
	}
	if len(vector) != ollamaNomicDimensions {
		return nil, fmt.Errorf("Ollama Nomic embedding returned %d dimensions; recipe requires %d", len(vector), ollamaNomicDimensions)
	}
	var normSquared float64
	for _, value := range vector {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return nil, errors.New("Ollama Nomic embedding contains a non-finite value")
		}
		normSquared += value * value
	}
	if normSquared == 0 || math.IsInf(normSquared, 0) {
		return nil, errors.New("Ollama Nomic embedding has an invalid L2 norm")
	}
	norm := math.Sqrt(normSquared)
	for index := range vector {
		vector[index] /= norm
	}
	return vector, nil
}

// ProviderDescriptionFleetPeakEstimator computes a conservative generation
// peak from the source-owned point count and immutable recipe dimensions. The
// per-point payload allowance and safety multiplier cover serialized owner
// payload plus index/WAL amplification; the fixed floor covers collection and
// segment metadata. Admission remains the shared capacity owner's final gate.
type ProviderDescriptionFleetPeakEstimator struct{}

func (ProviderDescriptionFleetPeakEstimator) EstimateFleetStagePeakBytes(_ context.Context, _ string, recipe aisearch.EmbeddingRecipe, pointCount uint64) (int64, error) {
	if err := recipe.Verify(); err != nil {
		return 0, fmt.Errorf("verify generation recipe before disk estimate: %w", err)
	}
	input, err := recipe.Input()
	if err != nil {
		return 0, err
	}
	if input.Dimensions <= 0 || pointCount > math.MaxInt64 {
		return 0, errors.New("generation disk estimate has invalid dimensions or point count")
	}
	vectorBytesPerPoint := int64(input.Dimensions) * 4
	if vectorBytesPerPoint > math.MaxInt64-qdrantPeakPayloadBound {
		return 0, errors.New("generation disk estimate overflows per-point size")
	}
	perPoint := (vectorBytesPerPoint + qdrantPeakPayloadBound) * qdrantPeakSafetyFactor
	count := int64(pointCount)
	if count > (math.MaxInt64-qdrantPeakFixedBytes)/perPoint {
		return 0, errors.New("generation disk estimate overflows capacity bytes")
	}
	return qdrantPeakFixedBytes + count*perPoint, nil
}

func isFleetDigest(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, r := range value {
		if !(r >= '0' && r <= '9' || r >= 'a' && r <= 'f') {
			return false
		}
	}
	return true
}
