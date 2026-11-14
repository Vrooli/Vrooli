package aisearch

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// EmbeddingRecipeInput describes the complete semantic and runtime recipe used
// to encode documents and queries. Values are copied into an immutable
// EmbeddingRecipe by NewEmbeddingRecipe.
type EmbeddingRecipeInput struct {
	SchemaVersion       string `json:"schema_version"`
	ModelID             string `json:"model_id"`
	ModelRevision       string `json:"model_revision"`
	ArtifactSHA256      string `json:"artifact_sha256"`
	TokenizerRevision   string `json:"tokenizer_revision"`
	Runtime             string `json:"runtime"`
	RuntimeBuild        string `json:"runtime_build"`
	Precision           string `json:"precision"`
	QueryInstruction    string `json:"query_instruction"`
	DocumentInstruction string `json:"document_instruction"`
	Pooling             string `json:"pooling"`
	Normalization       string `json:"normalization"`
	Dimensions          int    `json:"dimensions"`
	Projection          string `json:"projection"`
	OutputDType         string `json:"output_dtype"`
	OutputQuantization  string `json:"output_quantization"`
	Truncation          string `json:"truncation"`
	ChunkerVersion      string `json:"chunker_version"`
	ComposerVersion     string `json:"composer_version"`
}

// EmbeddingRecipe is an immutable, canonical embedding contract. Its fields
// are private so a caller cannot mutate the recipe after its identity has been
// computed. Use Input to obtain a detached copy.
type EmbeddingRecipe struct {
	canonical []byte
	digest    string
}

// NewEmbeddingRecipe validates and freezes input, returning a stable
// SHA-256 identity over the canonical JSON representation.
func NewEmbeddingRecipe(input EmbeddingRecipeInput) (EmbeddingRecipe, error) {
	input = normalizeEmbeddingRecipeInput(input)
	if err := validateEmbeddingRecipeInput(input); err != nil {
		return EmbeddingRecipe{}, err
	}
	canonical, err := json.Marshal(input)
	if err != nil {
		return EmbeddingRecipe{}, fmt.Errorf("encode embedding recipe: %w", err)
	}
	return embeddingRecipeFromCanonical(canonical)
}

// EmbeddingRecipeFromCanonical verifies persisted recipe bytes and their
// claimed digest before admitting them as an immutable recipe.
func EmbeddingRecipeFromCanonical(canonical []byte, digest string) (EmbeddingRecipe, error) {
	if len(canonical) == 0 {
		return EmbeddingRecipe{}, errors.New("embedding recipe canonical bytes are required")
	}
	var input EmbeddingRecipeInput
	decoder := json.NewDecoder(bytes.NewReader(canonical))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		return EmbeddingRecipe{}, fmt.Errorf("decode embedding recipe: %w", err)
	}
	if normalizeEmbeddingRecipeInput(input) != input {
		return EmbeddingRecipe{}, errors.New("embedding recipe contains non-normalized identity fields")
	}
	if err := validateEmbeddingRecipeInput(input); err != nil {
		return EmbeddingRecipe{}, err
	}
	canonicalAgain, err := json.Marshal(input)
	if err != nil {
		return EmbeddingRecipe{}, fmt.Errorf("re-encode embedding recipe: %w", err)
	}
	if !bytes.Equal(canonical, canonicalAgain) {
		return EmbeddingRecipe{}, errors.New("embedding recipe bytes are not canonical")
	}
	recipe, err := embeddingRecipeFromCanonical(canonical)
	if err != nil {
		return EmbeddingRecipe{}, err
	}
	if digest != recipe.digest {
		return EmbeddingRecipe{}, fmt.Errorf("embedding recipe digest mismatch: got %q, computed %q", digest, recipe.digest)
	}
	return recipe, nil
}

// ID returns the immutable recipe digest using the "sha256:" prefix.
func (r EmbeddingRecipe) ID() string { return "sha256:" + r.digest }

// Digest returns the raw lowercase SHA-256 hex digest.
func (r EmbeddingRecipe) Digest() string { return r.digest }

// Canonical returns a copy of the canonical JSON bytes.
func (r EmbeddingRecipe) Canonical() []byte { return bytes.Clone(r.canonical) }

// Input returns a detached copy of the recipe fields.
func (r EmbeddingRecipe) Input() (EmbeddingRecipeInput, error) {
	if len(r.canonical) == 0 {
		return EmbeddingRecipeInput{}, errors.New("embedding recipe is uninitialized")
	}
	var input EmbeddingRecipeInput
	if err := json.Unmarshal(r.canonical, &input); err != nil {
		return EmbeddingRecipeInput{}, fmt.Errorf("decode immutable embedding recipe: %w", err)
	}
	return input, nil
}

// Verify confirms the recipe's current bytes still match the identity captured
// at construction. It is useful at persistence and adapter boundaries.
func (r EmbeddingRecipe) Verify() error {
	if len(r.canonical) == 0 || r.digest == "" {
		return errors.New("embedding recipe is uninitialized")
	}
	hash := sha256.Sum256(r.canonical)
	if hex.EncodeToString(hash[:]) != r.digest {
		return errors.New("embedding recipe bytes changed after identity was computed")
	}
	return nil
}

func embeddingRecipeFromCanonical(canonical []byte) (EmbeddingRecipe, error) {
	hash := sha256.Sum256(canonical)
	return EmbeddingRecipe{canonical: bytes.Clone(canonical), digest: hex.EncodeToString(hash[:])}, nil
}

func normalizeEmbeddingRecipeInput(input EmbeddingRecipeInput) EmbeddingRecipeInput {
	input.SchemaVersion = strings.TrimSpace(input.SchemaVersion)
	input.ModelID = strings.TrimSpace(input.ModelID)
	input.ModelRevision = strings.TrimSpace(input.ModelRevision)
	input.ArtifactSHA256 = strings.ToLower(strings.TrimSpace(input.ArtifactSHA256))
	input.TokenizerRevision = strings.TrimSpace(input.TokenizerRevision)
	input.Runtime = strings.TrimSpace(input.Runtime)
	input.RuntimeBuild = strings.TrimSpace(input.RuntimeBuild)
	input.Precision = strings.ToLower(strings.TrimSpace(input.Precision))
	input.Pooling = strings.ToLower(strings.TrimSpace(input.Pooling))
	input.Normalization = strings.ToLower(strings.TrimSpace(input.Normalization))
	input.Projection = strings.TrimSpace(input.Projection)
	input.OutputDType = strings.ToLower(strings.TrimSpace(input.OutputDType))
	input.OutputQuantization = strings.ToLower(strings.TrimSpace(input.OutputQuantization))
	input.Truncation = strings.TrimSpace(input.Truncation)
	input.ChunkerVersion = strings.TrimSpace(input.ChunkerVersion)
	input.ComposerVersion = strings.TrimSpace(input.ComposerVersion)
	return input
}

func validateEmbeddingRecipeInput(input EmbeddingRecipeInput) error {
	for field, value := range map[string]string{
		"schema_version": input.SchemaVersion, "model_id": input.ModelID, "model_revision": input.ModelRevision,
		"artifact_sha256": input.ArtifactSHA256, "tokenizer_revision": input.TokenizerRevision,
		"runtime": input.Runtime, "runtime_build": input.RuntimeBuild, "precision": input.Precision,
		"pooling": input.Pooling, "normalization": input.Normalization,
		"output_dtype": input.OutputDType, "truncation": input.Truncation,
		"chunker_version": input.ChunkerVersion, "composer_version": input.ComposerVersion,
	} {
		if value == "" {
			return fmt.Errorf("embedding recipe %s is required", field)
		}
	}
	if len(input.ArtifactSHA256) != sha256.Size*2 {
		return errors.New("embedding recipe artifact_sha256 must be a 64-character SHA-256 hex digest")
	}
	if _, err := hex.DecodeString(input.ArtifactSHA256); err != nil {
		return fmt.Errorf("embedding recipe artifact_sha256 is invalid: %w", err)
	}
	if input.Dimensions <= 0 {
		return errors.New("embedding recipe dimensions must be positive")
	}
	if input.QueryInstruction == "" || input.DocumentInstruction == "" {
		return errors.New("embedding recipe query_instruction and document_instruction are required; use an explicit empty marker when none")
	}
	return nil
}
