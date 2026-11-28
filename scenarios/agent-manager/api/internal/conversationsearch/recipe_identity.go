package conversationsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	ollamapolicy "github.com/vrooli/ai-go/ollama/policy"
	aisearch "github.com/vrooli/ai-go/search"
)

const (
	// conversationComposerVersion names the embedding-input composition in
	// conversationEmbeddingComposer. Changing what the composer emits changes
	// every vector, so it must change this value (TestComposerVersionPinsOutput
	// fails until it does).
	conversationComposerVersion = "conversation-context-v1"
	// recipeUnobserved marks a recipe field the owner policy does not record.
	// The immutable recipe requires every field; saying "unobserved" binds the
	// identity to what is known without inventing the rest.
	recipeUnobserved = "unobserved"
	// sentinelScrollLimit bounds how many collection meta sentinels are read.
	// A seeded generation carries its ancestors' sentinels as well as its own.
	sentinelScrollLimit  = 16
	sentinelProbeTimeout = 5 * time.Second
	// qdrantMetaMarker is the payload key the shared engine stamps on a
	// collection's meta sentinel. It is that package's wire contract.
	qdrantMetaMarker = "__aisearch_meta__"
)

// mintConversationRecipe builds the immutable embedding recipe for the engine
// the owner policy resolved today. The artifact identity (model, manifest
// digest) is what the policy observed from the embedding runtime; runtime
// fields the policy does not record are marked unobserved rather than guessed.
func mintConversationRecipe(resolved ollamapolicy.ResolvedRole, spec aisearch.CollectionSpec, taskPrefix bool) (aisearch.EmbeddingRecipe, error) {
	identity := resolved.RuntimeIdentity
	switch {
	case identity == nil:
		return aisearch.EmbeddingRecipe{}, errors.New("the embedding policy reports no runtime artifact identity")
	case identity.Provider != "ollama" || identity.Model != resolved.Model || spec.Model != resolved.Model:
		return aisearch.EmbeddingRecipe{}, fmt.Errorf("runtime identity %s/%s does not describe the resolved model %q", identity.Provider, identity.Model, resolved.Model)
	case spec.DenseSize <= 0 || resolved.EmbeddingDimensions != spec.DenseSize:
		return aisearch.EmbeddingRecipe{}, fmt.Errorf("policy dimensions %d do not match the collection's %d", resolved.EmbeddingDimensions, spec.DenseSize)
	case strings.TrimSpace(identity.ArtifactRevision) == "":
		return aisearch.EmbeddingRecipe{}, errors.New("the embedding policy reports no artifact revision")
	}
	queryInstruction, documentInstruction := "none", "none"
	if taskPrefix {
		queryInstruction, documentInstruction = "search_query: ", "search_document: "
	}
	quantization := "none"
	if spec.Storage.ScalarQuantization {
		quantization = "scalar-int8"
	}
	return aisearch.NewEmbeddingRecipe(aisearch.EmbeddingRecipeInput{
		SchemaVersion: "1", ModelID: resolved.Model, ModelRevision: identity.ArtifactRevision,
		ArtifactSHA256: identity.Digest,
		// The tokenizer ships inside the model artifact, so the manifest digest
		// already covers it.
		TokenizerRevision: "embedded-in-artifact:" + identity.ArtifactRevision,
		Runtime:           "ollama-resource-gateway", RuntimeBuild: "policy-schema:" + strings.TrimSpace(resolved.SchemaVersion),
		Precision: recipeUnobserved, QueryInstruction: queryInstruction, DocumentInstruction: documentInstruction,
		Pooling: recipeUnobserved, Normalization: recipeUnobserved, Dimensions: spec.DenseSize,
		OutputDType: "float32", OutputQuantization: quantization,
		Truncation:     fmt.Sprintf("composer-max-%d-bytes;model-context-%d-tokens", maxEmbeddingInputBytes, resolved.ContextWindowTokens),
		ChunkerVersion: DefaultRecipeVersion, ComposerVersion: conversationComposerVersion,
	})
}

// servingSentinel is what a collection's own meta sentinel recorded when the
// collection was created: independent evidence of the identity of its vectors.
type servingSentinel struct {
	Model        string
	Role         string
	DenseSize    int
	PolicySchema string
	RecipeID     string
}

// readServingSentinels reads every meta sentinel the collection holds.
func readServingSentinels(parent context.Context, baseURL, apiKey, collection string) ([]servingSentinel, error) {
	baseURL, collection = strings.TrimRight(strings.TrimSpace(baseURL), "/"), strings.TrimSpace(collection)
	if baseURL == "" || collection == "" {
		return nil, errors.New("vector store address and collection are required")
	}
	ctx, cancel := context.WithTimeout(parent, sentinelProbeTimeout)
	defer cancel()
	body, err := json.Marshal(map[string]any{
		"limit": sentinelScrollLimit, "with_payload": true, "with_vector": false,
		"filter": map[string]any{"must": []map[string]any{{"key": qdrantMetaMarker, "match": map[string]any{"value": true}}}},
	})
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/collections/"+url.PathEscape(collection)+"/points/scroll", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	if apiKey = strings.TrimSpace(apiKey); apiKey != "" {
		request.Header.Set("api-key", apiKey)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("read collection meta sentinels: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("read collection meta sentinels: vector store returned status %d", response.StatusCode)
	}
	var payload struct {
		Result struct {
			Points []struct {
				Payload map[string]any `json:"payload"`
			} `json:"points"`
		} `json:"result"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode collection meta sentinels: %w", err)
	}
	sentinels := make([]servingSentinel, 0, len(payload.Result.Points))
	for _, point := range payload.Result.Points {
		size, _ := point.Payload["dense_size"].(float64)
		sentinels = append(sentinels, servingSentinel{
			Model: sentinelString(point.Payload, "model"), Role: sentinelString(point.Payload, "role"),
			DenseSize: int(size), PolicySchema: sentinelString(point.Payload, "policy_schema_version"),
			RecipeID: sentinelString(point.Payload, "embedding_recipe_id"),
		})
	}
	return sentinels, nil
}

func sentinelString(payload map[string]any, key string) string {
	value, _ := payload[key].(string)
	return strings.TrimSpace(value)
}

// sameEmbeddingModel compares model references the way the shared engine does:
// an untagged name is the :latest tag.
func sameEmbeddingModel(a, b string) bool {
	normalize := func(model string) string {
		model = strings.ToLower(strings.TrimSpace(model))
		if model != "" && !strings.Contains(model, ":") {
			model += ":latest"
		}
		return model
	}
	return normalize(a) != "" && normalize(a) == normalize(b)
}

// legacyAdoptionBlocker returns why a recipe-less serving generation cannot be
// stamped with recipe, or "" when it can. Adoption asserts that the generation's
// vectors were produced by the recipe's model, dimensions and chunker; the
// catalog row and every sentinel in the serving collection must say the same,
// so a model swap or a mixed collection is declined rather than relabelled.
func legacyAdoptionBlocker(metadata aisearch.GenerationMetadata, recipe aisearch.EmbeddingRecipe, sentinels []servingSentinel) string {
	input, err := recipe.Input()
	if err != nil {
		return "the minted recipe is unreadable: " + err.Error()
	}
	switch {
	case strings.TrimSpace(metadata.RecipeID) != "" || strings.TrimSpace(metadata.RecipeCanonical) != "":
		return "the generation already carries a recipe identity"
	case !sameEmbeddingModel(metadata.Model, input.ModelID):
		return fmt.Sprintf("generation model %q differs from the policy's %q", metadata.Model, input.ModelID)
	case metadata.EmbeddingDimensions != input.Dimensions:
		return fmt.Sprintf("generation dimensions %d differ from the policy's %d", metadata.EmbeddingDimensions, input.Dimensions)
	case metadata.ChunkPolicy != input.ChunkerVersion:
		return fmt.Sprintf("generation chunk policy %q differs from the current %q", metadata.ChunkPolicy, input.ChunkerVersion)
	case len(sentinels) == 0:
		return "the serving collection has no meta sentinel to confirm the identity of its vectors"
	}
	for _, sentinel := range sentinels {
		switch {
		case !sameEmbeddingModel(sentinel.Model, input.ModelID):
			return fmt.Sprintf("the serving collection was built with model %q, not the policy's %q", sentinel.Model, input.ModelID)
		case sentinel.DenseSize != input.Dimensions:
			return fmt.Sprintf("the serving collection holds %d-dimension vectors, not the policy's %d", sentinel.DenseSize, input.Dimensions)
		case sentinel.Role != metadata.EmbeddingRole || sentinel.PolicySchema != metadata.PolicySchemaVersion:
			return fmt.Sprintf("the serving collection recorded role %q / policy schema %q, not %q / %q", sentinel.Role, sentinel.PolicySchema, metadata.EmbeddingRole, metadata.PolicySchemaVersion)
		case sentinel.RecipeID != "" && sentinel.RecipeID != recipe.ID():
			return fmt.Sprintf("the serving collection already records a different recipe %q", sentinel.RecipeID)
		}
	}
	return ""
}

// adoptLegacyRecipe stamps a recipe-less serving generation with recipe after
// the serving collection's own sentinels confirm its vectors. It returns the
// reason adoption was declined ("" when adopted). The catalog write is the
// only mutation: vectors are not touched and nothing is re-embedded.
func adoptLegacyRecipe(ctx context.Context, catalog aisearch.GenerationCatalog, record *aisearch.GenerationRecord, recipe aisearch.EmbeddingRecipe, sentinels []servingSentinel) (string, error) {
	if reason := legacyAdoptionBlocker(record.Metadata, recipe, sentinels); reason != "" {
		return reason, nil
	}
	adopted := *record
	adopted.Metadata.RecipeID = recipe.ID()
	adopted.Metadata.RecipeCanonical = string(recipe.Canonical())
	adopted.UpdatedAt = time.Now().UTC()
	if err := catalog.SaveGenerationRecord(ctx, adopted); err != nil {
		return "", fmt.Errorf("record the adopted recipe on generation %s: %w", record.Metadata.ID, err)
	}
	*record = adopted
	return "", nil
}
