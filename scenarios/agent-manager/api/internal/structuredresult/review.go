package structuredresult

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"

	"agent-manager/internal/domain"
)

// NormalizeReviewSpec checks the authored verdict contract. Candidate identity
// is supplied by the retained-input owner at dispatch, never by prompt bindings.
func NormalizeReviewSpec(in *domain.ResultSpec) (*domain.ResultSpec, error) {
	spec, err := NormalizeSpec(in)
	if err != nil {
		return nil, err
	}
	if spec == nil || spec.Kind != domain.ResultSpecKindJSONSchema || spec.ExtractionMode != domain.StructuredExtractionDeterministic {
		return nil, fmt.Errorf("review requires a deterministic JSON-schema result")
	}
	var schema struct {
		Type       string                    `json:"type"`
		Required   []string                  `json:"required"`
		Properties map[string]map[string]any `json:"properties"`
	}
	if err := json.Unmarshal(spec.Schema, &schema); err != nil {
		return nil, err
	}
	if schema.Type != "object" {
		return nil, fmt.Errorf("review result must be an object")
	}
	for name, kind := range map[string]string{"accepted": "boolean", "candidateSha256": "string"} {
		if !slices.Contains(schema.Required, name) || schema.Properties[name]["type"] != kind {
			return nil, fmt.Errorf("review result requires %s with type %s", name, kind)
		}
	}
	return spec, nil
}

// BindReviewCandidate narrows the existing result schema and returns a copy.
// Both acceptance and rejection must name the exact retained input. Ordinary
// structured-result validation and persistence then own the verdict receipt.
func BindReviewCandidate(in *domain.ResultSpec, digest string) (*domain.ResultSpec, error) {
	decoded, err := hex.DecodeString(digest)
	if err != nil || len(decoded) != 32 || hex.EncodeToString(decoded) != digest {
		return nil, fmt.Errorf("review candidate requires a canonical SHA-256 digest")
	}
	spec, err := NormalizeReviewSpec(in)
	if err != nil {
		return nil, err
	}
	var schema map[string]any
	if err := json.Unmarshal(spec.Schema, &schema); err != nil {
		return nil, err
	}
	property := schema["properties"].(map[string]any)["candidateSha256"].(map[string]any)
	if prior, exists := property["const"]; exists && prior != digest {
		return nil, fmt.Errorf("review result is already bound to a different candidate")
	}
	property["const"] = digest
	spec.Schema, err = json.Marshal(schema)
	if err != nil {
		return nil, err
	}
	return NormalizeSpec(spec)
}
