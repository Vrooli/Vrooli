package experimentarms

import (
	"context"
	"errors"
	"math"
	"strings"
	"testing"

	ollamapolicy "github.com/vrooli/ai-go/ollama/policy"
	aisearch "github.com/vrooli/ai-go/search"
)

const trustedNomicModel = "nomic-embed-text:latest"
const trustedNomicDimensions = 768
const trustedNomicContextTokens = 2048

type trustedEmbeddingFake struct {
	calls int
	value float64
	size  int
	err   error
}

func (f *trustedEmbeddingFake) Embed(context.Context, string) ([]float64, error) {
	return f.vector()
}
func (*trustedEmbeddingFake) Available(context.Context) bool { return true }
func (f *trustedEmbeddingFake) EmbedQuery(context.Context, string) ([]float64, error) {
	return f.vector()
}
func (f *trustedEmbeddingFake) EmbedDocument(context.Context, string) ([]float64, error) {
	return f.vector()
}
func (f *trustedEmbeddingFake) vector() ([]float64, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	vector := make([]float64, f.size)
	for i := range vector {
		vector[i] = f.value
	}
	return vector, nil
}

func TestTrustedOllamaFleetEncoderBindsRecipeAndNormalizesWithoutProviderInference(t *testing.T) {
	fake := &trustedEmbeddingFake{value: 2, size: trustedNomicDimensions}
	encoder, err := newTrustedOllamaFleetDescriptionEncoder(trustedNomicRole(), fake)
	if err != nil {
		t.Fatal(err)
	}
	input, err := encoder.Recipe().Input()
	if err != nil {
		t.Fatal(err)
	}
	if input.ArtifactSHA256 != ollamaNomicManifestSHA256 || input.TokenizerRevision != "gguf-tokenizer-layer:sha256:"+ollamaNomicModelLayerSHA || input.Dimensions != trustedNomicDimensions || input.Precision != "f16" || input.Pooling != "gguf-mean" || input.Normalization != "l2" || input.QueryInstruction != "search_query: " || input.DocumentInstruction != "search_document: " || input.Truncation != "reject-input-over-1800-runes;gguf-context-2048" {
		t.Fatalf("recipe lost exact model/runtime behavior: %+v", input)
	}
	vector, err := encoder.EmbedDocument(context.Background(), "hello")
	if err != nil {
		t.Fatal(err)
	}
	if fake.calls != 1 || len(vector) != trustedNomicDimensions {
		t.Fatalf("calls=%d vector-dimensions=%d", fake.calls, len(vector))
	}
	var norm float64
	for _, value := range vector {
		norm += value * value
	}
	if math.Abs(norm-1) > 1e-12 {
		t.Fatalf("L2-normalized vector has squared norm %g", norm)
	}
}

func TestTrustedOllamaFleetEncoderRejectsIdentityDriftAndOversizedInput(t *testing.T) {
	for name, mutate := range map[string]func(*ollamapolicy.ResolvedRole){
		"model digest drift": func(role *ollamapolicy.ResolvedRole) { role.RuntimeIdentity.Digest = strings.Repeat("f", 64) },
		"rights missing":     func(role *ollamapolicy.ResolvedRole) { role.Rights = nil },
		"wrong license":      func(role *ollamapolicy.ResolvedRole) { role.Rights.LicenseID = "unknown" },
	} {
		t.Run(name, func(t *testing.T) {
			resolved := trustedNomicRole()
			mutate(&resolved)
			if _, err := newTrustedOllamaFleetDescriptionEncoder(resolved, &trustedEmbeddingFake{size: trustedNomicDimensions}); err == nil {
				t.Fatal("untrusted model identity was accepted")
			}
		})
	}
	fake := &trustedEmbeddingFake{size: trustedNomicDimensions}
	encoder, err := newTrustedOllamaFleetDescriptionEncoder(trustedNomicRole(), fake)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := encoder.EmbedQuery(context.Background(), strings.Repeat("x", ollamaMaxInputRunes+1)); err == nil || fake.calls != 0 {
		t.Fatalf("oversized input reached embedder: err=%v calls=%d", err, fake.calls)
	}
	badVector := &trustedEmbeddingFake{size: 2}
	badEncoder, err := newTrustedOllamaFleetDescriptionEncoder(trustedNomicRole(), badVector)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := badEncoder.Embed(context.Background(), "x"); err == nil {
		t.Fatal("dimension mismatch was accepted")
	}
	badVector.err = errors.New("owner gateway unavailable")
	if _, err := badEncoder.Embed(context.Background(), "x"); err == nil {
		t.Fatal("provider error was ignored")
	}
}

func TestTrustedOllamaFleetEncoderUsesPolicyResolvedModelIdentity(t *testing.T) {
	resolved := trustedNomicRole()
	resolved.Model = "nomic-embed-text-alias:latest"
	resolved.Rights.ArtifactID = "registry.ollama.ai/library/" + resolved.Model
	resolved.Rights.Evidence.URL = "https://ollama.com/library/" + resolved.Model
	resolved.RuntimeIdentity.Model = resolved.Model
	resolved.RuntimeIdentity.ArtifactID = resolved.Rights.ArtifactID
	encoder, err := newTrustedOllamaFleetDescriptionEncoder(resolved, &trustedEmbeddingFake{size: trustedNomicDimensions})
	if err != nil {
		t.Fatal(err)
	}
	input, err := encoder.Recipe().Input()
	if err != nil {
		t.Fatal(err)
	}
	if input.ModelID != resolved.Model || input.ArtifactSHA256 != ollamaNomicManifestSHA256 {
		t.Fatalf("recipe model identity = %q / %q; want policy-selected alias bound to pinned digest", input.ModelID, input.ArtifactSHA256)
	}
}

func TestProviderDescriptionFleetPeakEstimatorUsesRecipeAndOwnerPointCount(t *testing.T) {
	recipe, err := aisearch.NewEmbeddingRecipe(aisearch.EmbeddingRecipeInput{
		SchemaVersion: "1", ModelID: "test", ModelRevision: "rev", ArtifactSHA256: strings.Repeat("a", 64),
		TokenizerRevision: "tok", Runtime: "runtime", RuntimeBuild: "build", Precision: "f16",
		QueryInstruction: "none", DocumentInstruction: "none", Pooling: "mean", Normalization: "l2",
		Dimensions: 768, OutputDType: "float32", Truncation: "reject", ChunkerVersion: "chunker", ComposerVersion: "composer",
	})
	if err != nil {
		t.Fatal(err)
	}
	estimator := ProviderDescriptionFleetPeakEstimator{}
	got, err := estimator.EstimateFleetStagePeakBytes(context.Background(), "arm", recipe, 3)
	if err != nil {
		t.Fatal(err)
	}
	want := qdrantPeakFixedBytes + 3*((768*4+qdrantPeakPayloadBound)*qdrantPeakSafetyFactor)
	if got != want {
		t.Fatalf("peak=%d, want source-count/recipe-derived %d", got, want)
	}
	if _, err := estimator.EstimateFleetStagePeakBytes(context.Background(), "arm", recipe, math.MaxUint64); err == nil {
		t.Fatal("overflowing source count was accepted")
	}
	if _, err := estimator.EstimateFleetStagePeakBytes(context.Background(), "arm", aisearch.EmbeddingRecipe{}, 1); err == nil {
		t.Fatal("uninitialized recipe was accepted")
	}
}

func trustedNomicRole() ollamapolicy.ResolvedRole {
	artifactID := "registry.ollama.ai/library/" + trustedNomicModel
	revision := "ollama-manifest:sha256:" + ollamaNomicManifestSHA256
	rights := &ollamapolicy.ArtifactRights{
		ArtifactID: artifactID, Revision: revision, SHA256: ollamaNomicManifestSHA256,
		LicenseID: "Apache-2.0", Classification: "unrestricted",
		Evidence:    ollamapolicy.RightsEvidence{URL: "https://ollama.com/library/" + trustedNomicModel, Revision: ollamaNomicManifestSHA256, SHA256: ollamaNomicLicenseSHA, ObservedAt: "2026-11-12"},
		Obligations: []string{"retain the supplied Apache License 2.0 text and any supplied NOTICE when redistributing the artifact"},
	}
	identity := &ollamapolicy.RuntimeArtifactIdentity{
		Provider: "ollama", Model: trustedNomicModel, ArtifactID: artifactID,
		ArtifactRevision: revision, DigestKind: "ollama_model_contents",
		Digest: ollamaNomicManifestSHA256, Source: "ollama /api/tags", ObservedAt: "2026-11-12",
	}
	return ollamapolicy.ResolvedRole{
		SchemaVersion: "2026-06-10", PolicySHA256: strings.Repeat("a", 64), Role: ollamaEmbeddingRole,
		Source: "role", Model: trustedNomicModel, Capabilities: []string{"embedding"},
		EmbeddingDimensions: trustedNomicDimensions, ContextWindowTokens: trustedNomicContextTokens, Rights: rights, RuntimeIdentity: identity,
	}
}
