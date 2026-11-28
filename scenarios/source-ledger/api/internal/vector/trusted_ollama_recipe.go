package vector

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strings"
	"unicode/utf8"

	ollamapolicy "github.com/vrooli/ai-go/ollama/policy"
	aisearch "github.com/vrooli/ai-go/search"
)

const (
	sourceLedgerEmbeddingRole       = "embedding.default"
	sourceLedgerNomicModel          = "nomic-embed-text:latest"
	sourceLedgerNomicManifestSHA256 = "0a109f422b47e3a30ba2b10eca18548e944e8a23073ee3f3e947efcf3c45e59f"
	sourceLedgerNomicModelLayerSHA  = "970aa74c0a90ef7482477cf803618e776e173c007bf957f635f1015bfcfef0e6"
	sourceLedgerNomicLicenseSHA     = "c71d239df91726fc519c6eb72d318ec65820627232b2f796219e87dcf35d0ab4"
	sourceLedgerOllamaBuild         = "resource-ollama-gateway/0.1.0+ollama-server/0.30.10"
	sourceLedgerNomicDimensions     = 768
	sourceLedgerNomicContext        = 2048
	sourceLedgerMaxEmbeddingRunes   = 1800
	sourceLedgerNomicArtifact       = "registry.ollama.ai/library/" + sourceLedgerNomicModel
	sourceLedgerNomicRevision       = "ollama-manifest:sha256:" + sourceLedgerNomicManifestSHA256
)

type ollamaEmbeddingPolicyResolver interface {
	ResolveRole(context.Context, string) (ollamapolicy.ResolvedRole, error)
	ResolveModel(context.Context, string) (ollamapolicy.ResolvedRole, error)
}

type sourceLedgerTaskEmbedder interface {
	EmbedDocument(context.Context, string) ([]float64, error)
	EmbedQuery(context.Context, string) ([]float64, error)
}

// TrustedSourceLedgerOllamaRuntime implements the recipe and exact-recipe
// embedding owner seams using resource-ollama policy resolution and its
// rights-checking gateway. It never accepts model identity from an API request.
type TrustedSourceLedgerOllamaRuntime struct {
	db       *sql.DB
	resolver ollamaEmbeddingPolicyResolver
	encoder  sourceLedgerTaskEmbedder
}

func NewTrustedSourceLedgerOllamaRuntime(db *sql.DB, resolver ollamaEmbeddingPolicyResolver, encoder sourceLedgerTaskEmbedder) (*TrustedSourceLedgerOllamaRuntime, error) {
	if db == nil || resolver == nil || encoder == nil {
		return nil, errors.New("trusted Source Ledger embedding runtime requires database, Ollama policy resolver, and owner encoder")
	}
	return &TrustedSourceLedgerOllamaRuntime{db: db, resolver: resolver, encoder: encoder}, nil
}

func NewDefaultTrustedSourceLedgerOllamaRuntime(db *sql.DB) (*TrustedSourceLedgerOllamaRuntime, error) {
	if db == nil {
		return nil, errors.New("trusted Source Ledger embedding runtime requires database")
	}
	encoder, ok := aisearch.NewEmbedderWithPrefixes(sourceLedgerNomicModel, "search_query: ", "search_document: ").(aisearch.TaskEmbedder)
	if !ok {
		return nil, errors.New("resource-ollama embedding client does not expose task-aware document embedding")
	}
	return NewTrustedSourceLedgerOllamaRuntime(db, ollamapolicy.Resolver{}, encoder)
}

func (r *TrustedSourceLedgerOllamaRuntime) ResolveCurrentRecipe(ctx context.Context) (aisearch.EmbeddingRecipe, error) {
	resolved, err := r.resolver.ResolveRole(ctx, sourceLedgerEmbeddingRole)
	if err != nil {
		return aisearch.EmbeddingRecipe{}, fmt.Errorf("resolve Source Ledger embedding role through resource-ollama: %w", err)
	}
	return sourceLedgerRecipeFromResolved(resolved, true)
}

func (r *TrustedSourceLedgerOllamaRuntime) ResolveRecipe(ctx context.Context, recipeID string) (aisearch.EmbeddingRecipe, error) {
	var canonical []byte
	err := r.db.QueryRowContext(ctx, `SELECT recipe_canonical FROM vector_generations WHERE recipe_id=? ORDER BY created_at DESC LIMIT 1`, recipeID).Scan(&canonical)
	if errors.Is(err, sql.ErrNoRows) {
		current, err := r.ResolveCurrentRecipe(ctx)
		return requireRecipe(current, err, recipeID, "Source Ledger recipe is not registered by an existing generation and is not the current owner recipe")
	}
	if err != nil {
		return aisearch.EmbeddingRecipe{}, err
	}
	recipe, err := aisearch.EmbeddingRecipeFromCanonical(canonical, strings.TrimPrefix(recipeID, "sha256:"))
	if recipe, err = requireRecipe(recipe, err, recipeID, "persisted Source Ledger recipe identity is invalid"); err != nil {
		return aisearch.EmbeddingRecipe{}, err
	}
	input, err := recipe.Input()
	if err != nil {
		return aisearch.EmbeddingRecipe{}, err
	}
	resolved, err := r.resolver.ResolveModel(ctx, input.ModelID)
	if err != nil {
		return aisearch.EmbeddingRecipe{}, fmt.Errorf("re-resolve Source Ledger model rights/runtime identity: %w", err)
	}
	current, err := sourceLedgerRecipeFromResolved(resolved, false)
	if _, err := requireRecipe(current, err, recipeID, "Source Ledger persisted recipe no longer matches owner-resolved model rights/runtime identity"); err != nil {
		return aisearch.EmbeddingRecipe{}, err
	}
	return recipe, nil
}

func (r *TrustedSourceLedgerOllamaRuntime) EmbedDocument(ctx context.Context, recipeID, text string) ([]float64, error) {
	return r.embedForRecipe(ctx, recipeID, text, "document", r.encoder.EmbedDocument)
}

// EmbedQueryForRecipe pairs recall queries with the exact active generation's
// immutable recipe. It shares the generation's trusted runtime identity and
// query instruction rather than embedding through the mutable gateway default.
func (r *TrustedSourceLedgerOllamaRuntime) EmbedQueryForRecipe(ctx context.Context, recipeID, text string) ([]float64, error) {
	return r.embedForRecipe(ctx, recipeID, text, "query", r.encoder.EmbedQuery)
}

// embedForRecipe rejects oversized input before invoking the provider and
// returns only a normalized vector of the recipe's exact dimensions.
func (r *TrustedSourceLedgerOllamaRuntime) embedForRecipe(ctx context.Context, recipeID, text, kind string, embed func(context.Context, string) ([]float64, error)) ([]float64, error) {
	recipe, err := r.ResolveRecipe(ctx, recipeID)
	if err != nil {
		return nil, err
	}
	if utf8.RuneCountInString(text) > sourceLedgerMaxEmbeddingRunes {
		return nil, fmt.Errorf("Source Ledger %s embedding input exceeds %d runes", kind, sourceLedgerMaxEmbeddingRunes)
	}
	vector, err := embed(ctx, text)
	if err != nil {
		return nil, err
	}
	input, _ := recipe.Input()
	if len(vector) != input.Dimensions {
		return nil, fmt.Errorf("Source Ledger %s embedding returned %d dimensions; recipe requires %d", kind, len(vector), input.Dimensions)
	}
	return normalizeRecipeVector(vector)
}

func normalizeRecipeVector(vector []float64) ([]float64, error) {
	var normSquared float64
	for _, value := range vector {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return nil, errors.New("Source Ledger embedding contains a non-finite value")
		}
		normSquared += value * value
	}
	if normSquared == 0 || math.IsInf(normSquared, 0) {
		return nil, errors.New("Source Ledger embedding has an invalid L2 norm")
	}
	norm := math.Sqrt(normSquared)
	for index := range vector {
		vector[index] /= norm
	}
	return vector, nil
}

func sourceLedgerRecipeFromResolved(resolved ollamapolicy.ResolvedRole, requireRole bool) (aisearch.EmbeddingRecipe, error) {
	// A model-level resolution carries no role, or the generic "model" role.
	roleMatches := resolved.Role == sourceLedgerEmbeddingRole || !requireRole && (resolved.Role == "" || resolved.Role == "model")
	if !roleMatches || resolved.Model != sourceLedgerNomicModel || resolved.EmbeddingDimensions != sourceLedgerNomicDimensions || resolved.ContextWindowTokens != sourceLedgerNomicContext {
		return aisearch.EmbeddingRecipe{}, errors.New("Source Ledger embedding owner did not resolve the pinned 768-dimensional Nomic model/context")
	}
	if err := verifySourceLedgerNomicArtifact(resolved); err != nil {
		return aisearch.EmbeddingRecipe{}, err
	}
	return aisearch.NewEmbeddingRecipe(aisearch.EmbeddingRecipeInput{
		SchemaVersion: "1", ModelID: sourceLedgerNomicModel,
		ModelRevision:  sourceLedgerNomicRevision,
		ArtifactSHA256: sourceLedgerNomicManifestSHA256, TokenizerRevision: "gguf-tokenizer-layer:sha256:" + sourceLedgerNomicModelLayerSHA,
		Runtime: "ollama-resource-gateway", RuntimeBuild: sourceLedgerOllamaBuild,
		Precision: "f16", QueryInstruction: "search_query: ", DocumentInstruction: "search_document: ",
		Pooling: "gguf-mean", Normalization: "l2", Dimensions: sourceLedgerNomicDimensions,
		OutputDType: "float32", OutputQuantization: "none", Truncation: "reject-input-over-1800-runes;gguf-context-2048",
		ChunkerVersion: "source-ledger-row/v1", ComposerVersion: "nomic-task-prefix/v1+source-ledger-document/v1",
	})
}

// verifySourceLedgerNomicArtifact binds rights, license evidence and runtime
// identity to the pinned manifest. Observation times and obligations are not
// identity.
func verifySourceLedgerNomicArtifact(resolved ollamapolicy.ResolvedRole) error {
	rights, identity := resolved.Rights, resolved.RuntimeIdentity
	if rights == nil || identity == nil || !sourceLedgerSHA256(resolved.PolicySHA256) ||
		[...]string{rights.ArtifactID, rights.Revision, rights.SHA256, rights.LicenseID, rights.Classification} != [...]string{sourceLedgerNomicArtifact, sourceLedgerNomicRevision, sourceLedgerNomicManifestSHA256, "Apache-2.0", "unrestricted"} {
		return errors.New("Source Ledger Nomic rights are missing or do not bind the exact installed manifest")
	}
	evidence := rights.Evidence
	evidence.ObservedAt = ""
	if evidence != (ollamapolicy.RightsEvidence{URL: "https://ollama.com/library/" + sourceLedgerNomicModel, Revision: sourceLedgerNomicManifestSHA256, SHA256: sourceLedgerNomicLicenseSHA}) {
		return errors.New("Source Ledger Nomic license evidence does not match the verified model page and bundled license")
	}
	runtime := *identity
	runtime.ObservedAt = ""
	if identity.ObservedAt == "" || runtime != (ollamapolicy.RuntimeArtifactIdentity{
		Provider: "ollama", Model: sourceLedgerNomicModel, ArtifactID: sourceLedgerNomicArtifact, ArtifactRevision: sourceLedgerNomicRevision,
		DigestKind: "ollama_model_contents", Digest: sourceLedgerNomicManifestSHA256, Source: "ollama /api/tags",
	}) {
		return errors.New("Source Ledger Nomic runtime identity does not bind the installed manifest digest")
	}
	return nil
}

func sourceLedgerSHA256(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, char := range value {
		if !(char >= '0' && char <= '9' || char >= 'a' && char <= 'f') {
			return false
		}
	}
	return true
}
