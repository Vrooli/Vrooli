// Package structuredresult owns the single typed-output pipeline layered on
// top of domain.RunResult. Provider extraction is advisory; local parsing and
// JSON Schema validation remain the only authority for successful values.
package structuredresult

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	"agent-manager/internal/domain"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

const (
	SpecVersionV1      = "result-spec/v1"
	DefaultExtractRole = "extract.structured"
	MaxSchemaBytes     = 32 * 1024
	MaxSchemaDepth     = 32
	MaxSourceBytes     = 64 * 1024
	MaxCandidateBytes  = 64 * 1024
	MaxDiagnosticBytes = 8 * 1024
)

// ExtractRequest is deliberately bounded and provider-neutral.
type ExtractRequest struct {
	Source  string
	Schema  json.RawMessage
	RoleRef string
	// Instruction states caller intent. It is deliberately separate from the
	// schema: ai-gateway treats schema descriptions as metadata only and never
	// as instruction, so intent must travel in its own field.
	Instruction string
}

// ExtractResponse contains an untrusted candidate and provenance. Candidate is
// always parsed and locally revalidated before it can become success.
type ExtractResponse struct {
	Candidate      json.RawMessage
	Provider       string
	Model          string
	PolicySnapshot *domain.ExecutionPolicySnapshot
	Abstained      bool
}

// Extractor is the optional constrained-extraction seam. Production adapters
// resolve RoleRef through portable role policy; tests may provide a fake.
type Extractor interface {
	Extract(context.Context, ExtractRequest) (ExtractResponse, error)
}

// Resolver evaluates structured results. A nil Extractor deterministically
// degrades constrained fallback to abstained rather than guessing.
type Resolver struct {
	Extractor Extractor
}

// ClassificationSpec is the convenience helper for the common enum case. It
// still returns the same ResultSpec contract used for arbitrary JSON Schema.
func ClassificationSpec(values ...string) *domain.ResultSpec {
	return &domain.ResultSpec{
		Version:              SpecVersionV1,
		Kind:                 domain.ResultSpecKindClassification,
		ClassificationValues: append([]string(nil), values...),
		ExtractionMode:       domain.StructuredExtractionConstrained,
		ExtractionRole:       DefaultExtractRole,
	}
}

// NormalizeSpec validates the supported schema subset, canonicalizes the
// schema bytes, and calculates the digest stored on the run. It performs no
// remote loading and rejects all reference keywords.
func NormalizeSpec(in *domain.ResultSpec) (*domain.ResultSpec, error) {
	if in == nil || in.Kind == "" || in.Kind == domain.ResultSpecKindNone {
		return nil, nil
	}
	out := *in
	if out.Version == "" {
		out.Version = SpecVersionV1
	}
	if out.Version != SpecVersionV1 {
		return nil, fmt.Errorf("unsupported result spec version %q", out.Version)
	}
	if out.ExtractionMode == "" {
		out.ExtractionMode = domain.StructuredExtractionDeterministic
	}
	if out.ExtractionMode != domain.StructuredExtractionDeterministic && out.ExtractionMode != domain.StructuredExtractionConstrained {
		return nil, fmt.Errorf("unsupported extraction mode %q", out.ExtractionMode)
	}
	if out.ExtractionMode == domain.StructuredExtractionConstrained {
		if strings.TrimSpace(out.ExtractionRole) == "" {
			out.ExtractionRole = DefaultExtractRole
		}
	} else {
		out.ExtractionRole = ""
	}
	if out.SchemaRepairAttempts != nil && (*out.SchemaRepairAttempts < 0 || *out.SchemaRepairAttempts > 1) {
		return nil, fmt.Errorf("schemaRepairAttempts must be zero or one")
	}

	var schemaValue any
	switch out.Kind {
	case domain.ResultSpecKindJSONSchema:
		if len(out.Schema) == 0 {
			return nil, errors.New("json_schema result spec requires schema")
		}
		if len(out.Schema) > MaxSchemaBytes {
			return nil, fmt.Errorf("result schema exceeds %d bytes", MaxSchemaBytes)
		}
		if len(out.ClassificationValues) != 0 {
			return nil, errors.New("classificationValues is only valid for classification result specs")
		}
		if err := decodeOne(out.Schema, &schemaValue); err != nil {
			return nil, fmt.Errorf("decode result schema: %w", err)
		}
	case domain.ResultSpecKindClassification:
		if len(out.Schema) != 0 && len(out.ClassificationValues) != 0 {
			return nil, errors.New("classification must provide schema or classificationValues, not both")
		}
		if len(out.Schema) != 0 {
			if len(out.Schema) > MaxSchemaBytes {
				return nil, fmt.Errorf("result schema exceeds %d bytes", MaxSchemaBytes)
			}
			if err := decodeOne(out.Schema, &schemaValue); err != nil {
				return nil, fmt.Errorf("decode classification schema: %w", err)
			}
		} else {
			values, err := normalizedEnum(out.ClassificationValues)
			if err != nil {
				return nil, err
			}
			schemaValue = map[string]any{"type": "string", "enum": values}
		}
	default:
		return nil, fmt.Errorf("unsupported result spec kind %q", out.Kind)
	}

	if err := validateSchemaSubset(schemaValue, 0); err != nil {
		return nil, err
	}
	canonical, err := json.Marshal(schemaValue)
	if err != nil {
		return nil, fmt.Errorf("canonicalize result schema: %w", err)
	}
	if len(canonical) > MaxSchemaBytes {
		return nil, fmt.Errorf("result schema exceeds %d bytes", MaxSchemaBytes)
	}
	if _, err := compileSchema(canonical); err != nil {
		return nil, fmt.Errorf("compile result schema: %w", err)
	}
	digest := sha256.Sum256(canonical)
	out.Schema = canonical
	out.SchemaDigest = "sha256:" + hex.EncodeToString(digest[:])
	out.ClassificationValues = nil
	return &out, nil
}

func normalizedEnum(values []string) ([]any, error) {
	if len(values) == 0 {
		return nil, errors.New("classification requires at least one value")
	}
	seen := make(map[string]struct{}, len(values))
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			return nil, errors.New("classification values must be non-empty")
		}
		if _, exists := seen[value]; exists {
			return nil, fmt.Errorf("duplicate classification value %q", value)
		}
		seen[value] = struct{}{}
		normalized = append(normalized, value)
	}
	sort.Strings(normalized)
	out := make([]any, len(normalized))
	for i := range normalized {
		out[i] = normalized[i]
	}
	return out, nil
}

var supportedKeywords = map[string]struct{}{
	"$schema": {}, "$id": {}, "title": {}, "description": {},
	"type": {}, "properties": {}, "required": {}, "additionalProperties": {},
	"items": {}, "enum": {}, "const": {}, "oneOf": {},
	"minLength": {}, "maxLength": {}, "pattern": {},
	"minItems": {}, "maxItems": {}, "uniqueItems": {},
	"minimum": {}, "maximum": {}, "exclusiveMinimum": {}, "exclusiveMaximum": {}, "multipleOf": {},
}

func validateSchemaSubset(value any, depth int) error {
	if depth > MaxSchemaDepth {
		return fmt.Errorf("result schema exceeds maximum depth %d", MaxSchemaDepth)
	}
	obj, ok := value.(map[string]any)
	if !ok {
		return errors.New("result schema nodes must be objects")
	}
	for key := range obj {
		if _, ok := supportedKeywords[key]; !ok {
			return fmt.Errorf("unsupported result schema keyword %q", key)
		}
	}
	if raw, ok := obj["type"]; ok {
		typeName, ok := raw.(string)
		if !ok {
			return errors.New("result schema type must be one string; use oneOf for unions")
		}
		switch typeName {
		case "object", "array", "string", "number", "integer", "boolean", "null":
		default:
			return fmt.Errorf("unsupported result schema type %q", typeName)
		}
	}
	if raw, ok := obj["required"]; ok {
		required, ok := raw.([]any)
		if !ok {
			return errors.New("result schema required must be an array")
		}
		for _, name := range required {
			if typed, ok := name.(string); !ok || strings.TrimSpace(typed) == "" {
				return errors.New("result schema required entries must be non-empty strings")
			}
		}
	}
	if raw, ok := obj["properties"]; ok {
		properties, ok := raw.(map[string]any)
		if !ok {
			return errors.New("result schema properties must be an object")
		}
		for name, child := range properties {
			if strings.TrimSpace(name) == "" {
				return errors.New("result schema property names must be non-empty")
			}
			if err := validateSchemaSubset(child, depth+1); err != nil {
				return fmt.Errorf("property %q: %w", name, err)
			}
		}
	}
	if raw, ok := obj["items"]; ok {
		if err := validateSchemaSubset(raw, depth+1); err != nil {
			return fmt.Errorf("items: %w", err)
		}
	}
	if raw, ok := obj["additionalProperties"]; ok {
		switch typed := raw.(type) {
		case bool:
		case map[string]any:
			if err := validateSchemaSubset(typed, depth+1); err != nil {
				return fmt.Errorf("additionalProperties: %w", err)
			}
		default:
			return errors.New("result schema additionalProperties must be boolean or a schema object")
		}
	}
	if raw, ok := obj["oneOf"]; ok {
		branches, ok := raw.([]any)
		if !ok || len(branches) == 0 {
			return errors.New("result schema oneOf must be a non-empty array")
		}
		for i, branch := range branches {
			if err := validateSchemaSubset(branch, depth+1); err != nil {
				return fmt.Errorf("oneOf[%d]: %w", i, err)
			}
		}
		if err := validateDiscriminatedOneOf(branches); err != nil {
			return err
		}
	}
	return nil
}

func validateDiscriminatedOneOf(branches []any) error {
	discriminator := ""
	for i, branch := range branches {
		obj, ok := branch.(map[string]any)
		if !ok {
			return fmt.Errorf("oneOf[%d] must be an object schema with a required const discriminator", i)
		}
		properties, _ := obj["properties"].(map[string]any)
		requiredRaw, _ := obj["required"].([]any)
		required := make(map[string]struct{}, len(requiredRaw))
		for _, raw := range requiredRaw {
			if name, ok := raw.(string); ok {
				required[name] = struct{}{}
			}
		}
		branchDiscriminator := ""
		for name, raw := range properties {
			property, _ := raw.(map[string]any)
			if _, hasConst := property["const"]; hasConst {
				if _, isRequired := required[name]; isRequired {
					branchDiscriminator = name
					break
				}
			}
		}
		if branchDiscriminator == "" {
			return fmt.Errorf("oneOf[%d] requires a const discriminator property listed in required", i)
		}
		if discriminator == "" {
			discriminator = branchDiscriminator
		} else if discriminator != branchDiscriminator {
			return errors.New("oneOf branches must use the same discriminator property")
		}
	}
	return nil
}

// Resolve applies the deterministic-first ladder and optional constrained
// extraction fallback. It never mutates final-output selection provenance.
func (r Resolver) Resolve(ctx context.Context, spec *domain.ResultSpec, result *domain.RunResult) *domain.StructuredResult {
	if spec == nil || spec.Kind == domain.ResultSpecKindNone {
		return nil
	}
	normalized, err := NormalizeSpec(spec)
	if err != nil {
		return outcome(spec, domain.StructuredResultUnavailable, "unsupported_schema", "requested schema is unsupported")
	}
	out := &domain.StructuredResult{SpecKind: normalized.Kind, SchemaDigest: normalized.SchemaDigest}
	if result == nil || result.Selection.Status == domain.FinalOutputSelectionUnavailable {
		return withDiagnostic(out, domain.StructuredResultUnavailable, "final_output_unavailable", "canonical final output is unavailable")
	}
	if result.Selection.Status == domain.FinalOutputSelectionAmbiguous {
		return withDiagnostic(out, domain.StructuredResultAmbiguous, "final_output_ambiguous", "canonical final output is ambiguous")
	}
	out.SourceCandidateID = result.Selection.SelectedCandidateID
	if len(result.FinalOutput) > MaxSourceBytes {
		return withDiagnostic(out, domain.StructuredResultInvalid, "source_too_large", fmt.Sprintf("final output exceeds %d bytes", MaxSourceBytes))
	}

	candidate, method, parseStatus, diagnostic := deterministicCandidate(result.FinalOutput)
	if parseStatus == domain.StructuredResultAmbiguous {
		return withDiagnostic(out, parseStatus, diagnostic.Code, diagnostic.Message)
	}
	if parseStatus == domain.StructuredResultSuccess {
		if valid, diagnostic := validateCandidate(normalized.Schema, candidate); valid {
			out.Status = domain.StructuredResultSuccess
			out.Method = method
			out.Value = candidate
			return out
		} else if normalized.ExtractionMode == domain.StructuredExtractionDeterministic {
			return withDiagnostic(out, domain.StructuredResultInvalid, diagnostic.Code, diagnostic.Message)
		}
	} else if normalized.ExtractionMode == domain.StructuredExtractionDeterministic {
		return withDiagnostic(out, domain.StructuredResultInvalid, diagnostic.Code, diagnostic.Message)
	}

	if r.Extractor == nil {
		return withDiagnostic(out, domain.StructuredResultAbstained, "extractor_unavailable", "constrained extractor is unavailable")
	}
	response, err := r.Extractor.Extract(ctx, ExtractRequest{
		Source: result.FinalOutput, Schema: normalized.Schema, RoleRef: normalized.ExtractionRole,
	})
	out.Method = "constrained_extractor"
	out.Extractor = &domain.StructuredExtractionProvenance{
		RoleRef: normalized.ExtractionRole, Provider: response.Provider, Model: response.Model, PolicySnapshot: response.PolicySnapshot,
	}
	if err != nil {
		return withDiagnostic(out, domain.StructuredResultAbstained, "extractor_unavailable", "constrained extractor did not return a candidate")
	}
	if response.Abstained || len(response.Candidate) == 0 {
		return withDiagnostic(out, domain.StructuredResultAbstained, "extractor_abstained", "constrained extractor abstained")
	}
	if len(response.Candidate) > MaxCandidateBytes {
		return withDiagnostic(out, domain.StructuredResultInvalid, "candidate_too_large", fmt.Sprintf("extracted candidate exceeds %d bytes", MaxCandidateBytes))
	}
	var compact bytes.Buffer
	if err := json.Compact(&compact, response.Candidate); err != nil {
		return withDiagnostic(out, domain.StructuredResultInvalid, "extractor_invalid_json", "constrained extractor returned invalid JSON")
	}
	extracted := json.RawMessage(compact.Bytes())
	if valid, diagnostic := validateCandidate(normalized.Schema, extracted); !valid {
		return withDiagnostic(out, domain.StructuredResultInvalid, diagnostic.Code, diagnostic.Message)
	}
	out.Status = domain.StructuredResultSuccess
	out.Value = extracted
	return out
}

func outcome(spec *domain.ResultSpec, status domain.StructuredResultStatus, code, message string) *domain.StructuredResult {
	out := &domain.StructuredResult{Status: status}
	if spec != nil {
		out.SpecKind = spec.Kind
		out.SchemaDigest = spec.SchemaDigest
	}
	return withDiagnostic(out, status, code, message)
}

func withDiagnostic(out *domain.StructuredResult, status domain.StructuredResultStatus, code, message string) *domain.StructuredResult {
	out.Status = status
	if len(message) > MaxDiagnosticBytes {
		message = message[:MaxDiagnosticBytes]
	}
	out.Diagnostics = []domain.StructuredDiagnostic{{Code: code, Message: message}}
	return out
}

func deterministicCandidate(source string) (json.RawMessage, string, domain.StructuredResultStatus, domain.StructuredDiagnostic) {
	trimmed := strings.TrimSpace(source)
	if trimmed == "" {
		return nil, "", domain.StructuredResultInvalid, domain.StructuredDiagnostic{Code: "empty_source", Message: "final output is empty"}
	}
	if len(trimmed) <= MaxCandidateBytes {
		var value any
		if err := decodeOne([]byte(trimmed), &value); err == nil {
			canonical, _ := json.Marshal(value)
			return canonical, "whole_document", domain.StructuredResultSuccess, domain.StructuredDiagnostic{}
		}
		if countJSONValues([]byte(trimmed)) > 1 {
			return nil, "", domain.StructuredResultAmbiguous, domain.StructuredDiagnostic{Code: "multiple_json_candidates", Message: "final output contains multiple JSON values"}
		}
	}
	fences := jsonFences(source)
	if len(fences) > 1 {
		return nil, "", domain.StructuredResultAmbiguous, domain.StructuredDiagnostic{Code: "multiple_json_candidates", Message: "final output contains multiple JSON fences"}
	}
	if len(fences) == 1 {
		if len(fences[0]) > MaxCandidateBytes {
			return nil, "", domain.StructuredResultInvalid, domain.StructuredDiagnostic{Code: "candidate_too_large", Message: fmt.Sprintf("JSON fence exceeds %d bytes", MaxCandidateBytes)}
		}
		var value any
		if err := decodeOne([]byte(fences[0]), &value); err != nil {
			return nil, "", domain.StructuredResultInvalid, domain.StructuredDiagnostic{Code: "invalid_json", Message: "JSON fence is not a single valid JSON value"}
		}
		canonical, _ := json.Marshal(value)
		return canonical, "json_fence", domain.StructuredResultSuccess, domain.StructuredDiagnostic{}
	}
	return nil, "", domain.StructuredResultInvalid, domain.StructuredDiagnostic{Code: "invalid_json", Message: "final output is neither a whole JSON document nor one JSON fence"}
}

func countJSONValues(raw []byte) int {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	count := 0
	for {
		var value any
		err := decoder.Decode(&value)
		if errors.Is(err, io.EOF) {
			return count
		}
		if err != nil {
			return 0
		}
		count++
	}
}

func jsonFences(source string) []string {
	var out []string
	rest := source
	for {
		start := strings.Index(rest, "```")
		if start < 0 {
			break
		}
		rest = rest[start+3:]
		lineEnd := strings.IndexByte(rest, '\n')
		if lineEnd < 0 {
			break
		}
		language := strings.TrimSpace(rest[:lineEnd])
		body := rest[lineEnd+1:]
		end := strings.Index(body, "```")
		if end < 0 {
			break
		}
		if language == "json" || language == "JSON" {
			out = append(out, strings.TrimSpace(body[:end]))
		}
		rest = body[end+3:]
	}
	return out
}

func validateCandidate(schemaBytes, candidate json.RawMessage) (bool, domain.StructuredDiagnostic) {
	schema, err := compileSchema(schemaBytes)
	if err != nil {
		return false, domain.StructuredDiagnostic{Code: "unsupported_schema", Message: "requested schema could not be compiled"}
	}
	var value any
	if err := decodeOne(candidate, &value); err != nil {
		return false, domain.StructuredDiagnostic{Code: "invalid_json", Message: "candidate is not a single valid JSON value"}
	}
	if err := schema.Validate(value); err != nil {
		path := ""
		if validationErr, ok := err.(*jsonschema.ValidationError); ok {
			path = validationErr.InstanceLocation
		}
		return false, domain.StructuredDiagnostic{Code: "schema_mismatch", Path: path, Message: "candidate does not satisfy the requested schema"}
	}
	return true, domain.StructuredDiagnostic{}
}

// ValidateValue applies the same canonical local schema authority used for
// structured Run results to workflow input/output snapshots.
func ValidateValue(schemaBytes, candidate json.RawMessage) error {
	if len(candidate) == 0 || len(candidate) > MaxCandidateBytes {
		return fmt.Errorf("candidate must be between 1 and %d bytes", MaxCandidateBytes)
	}
	valid, diagnostic := validateCandidate(schemaBytes, candidate)
	if valid {
		return nil
	}
	return fmt.Errorf("%s: %s", diagnostic.Code, diagnostic.Message)
}

func compileSchema(raw []byte) (*jsonschema.Schema, error) {
	compiler := jsonschema.NewCompiler()
	compiler.Draft = jsonschema.Draft2020
	const resource = "urn:agent-manager:result-spec"
	if err := compiler.AddResource(resource, bytes.NewReader(raw)); err != nil {
		return nil, err
	}
	return compiler.Compile(resource)
}

func decodeOne(raw []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("multiple JSON values")
		}
		return err
	}
	return nil
}
