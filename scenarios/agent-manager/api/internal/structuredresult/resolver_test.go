package structuredresult

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"agent-manager/internal/domain"
)

type fakeExtractor struct {
	response ExtractResponse
	err      error
	request  ExtractRequest
}

func (f *fakeExtractor) Extract(_ context.Context, request ExtractRequest) (ExtractResponse, error) {
	f.request = request
	return f.response, f.err
}

func selected(output string) *domain.RunResult {
	return &domain.RunResult{
		FinalOutput: output,
		Selection: domain.FinalOutputSelection{
			Status: domain.FinalOutputSelectionSelected, SelectedCandidateID: "candidate-1",
		},
	}
}

func objectSpec(extraction domain.StructuredExtractionMode) *domain.ResultSpec {
	return &domain.ResultSpec{
		Version: SpecVersionV1, Kind: domain.ResultSpecKindJSONSchema,
		Schema:         json.RawMessage(`{"type":"object","properties":{"answer":{"type":"string"}},"required":["answer"],"additionalProperties":false}`),
		ExtractionMode: extraction,
	}
}

func TestNormalizeSpecCanonicalizesAndDigestsClassification(t *testing.T) {
	spec, err := NormalizeSpec(ClassificationSpec("blocked", "complete"))
	if err != nil {
		t.Fatal(err)
	}
	if spec.SchemaDigest == "" || len(spec.ClassificationValues) != 0 {
		t.Fatalf("normalized spec = %+v", spec)
	}
	if got := string(spec.Schema); got != `{"enum":["blocked","complete"],"type":"string"}` {
		t.Fatalf("canonical schema = %s", got)
	}
	again, err := NormalizeSpec(spec)
	if err != nil || again.SchemaDigest != spec.SchemaDigest {
		t.Fatalf("normalization is not stable: %+v err=%v", again, err)
	}
}

func TestNormalizeSpecRejectsUnsupportedAndUnboundedInputs(t *testing.T) {
	tests := []struct {
		name string
		spec *domain.ResultSpec
		want string
	}{
		{"remote ref", &domain.ResultSpec{Kind: domain.ResultSpecKindJSONSchema, Schema: json.RawMessage(`{"$ref":"https://example.test/schema"}`)}, "unsupported"},
		{"unknown keyword", &domain.ResultSpec{Kind: domain.ResultSpecKindJSONSchema, Schema: json.RawMessage(`{"type":"string","format":"email"}`)}, "unsupported"},
		{"nested remote ref", &domain.ResultSpec{Kind: domain.ResultSpecKindJSONSchema, Schema: json.RawMessage(`{"type":"object","additionalProperties":{"$ref":"https://example.test/schema"}}`)}, "unsupported"},
		{"undiscriminated union", &domain.ResultSpec{Kind: domain.ResultSpecKindJSONSchema, Schema: json.RawMessage(`{"oneOf":[{"type":"string"},{"type":"number"}]}`)}, "discriminator"},
		{"empty classification", ClassificationSpec(), "at least one"},
		{"duplicate classification", ClassificationSpec("yes", "yes"), "duplicate"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NormalizeSpec(tt.spec)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestNormalizeSpecAcceptsDiscriminatedUnion(t *testing.T) {
	spec := &domain.ResultSpec{Kind: domain.ResultSpecKindJSONSchema, Schema: json.RawMessage(`{
		"oneOf":[
			{"type":"object","properties":{"kind":{"const":"done"},"value":{"type":"string"}},"required":["kind","value"]},
			{"type":"object","properties":{"kind":{"const":"blocked"},"reason":{"type":"string"}},"required":["kind","reason"]}
		]
	}`)}
	if _, err := NormalizeSpec(spec); err != nil {
		t.Fatalf("valid discriminated union rejected: %v", err)
	}
}

func TestResolverDeterministicRules(t *testing.T) { // [REQ:REQ-P2-001]
	resolver := Resolver{}
	tests := []struct {
		name, output string
		status       domain.StructuredResultStatus
		method, code string
	}{
		{"whole document", `{"answer":"yes"}`, domain.StructuredResultSuccess, "whole_document", ""},
		{"one fenced document", "The answer follows.\n```json\n{\"answer\":\"yes\"}\n```", domain.StructuredResultSuccess, "json_fence", ""},
		{"prose object is not guessed", `answer: {"answer":"yes"}`, domain.StructuredResultInvalid, "", "invalid_json"},
		{"duplicate fences are ambiguous", "```json\n{\"answer\":\"a\"}\n```\n```json\n{\"answer\":\"b\"}\n```", domain.StructuredResultAmbiguous, "", "multiple_json_candidates"},
		{"duplicate whole values are ambiguous", "{\"answer\":\"a\"}\n{\"answer\":\"b\"}", domain.StructuredResultAmbiguous, "", "multiple_json_candidates"},
		{"schema mismatch", `{"answer":3}`, domain.StructuredResultInvalid, "", "schema_mismatch"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolver.Resolve(context.Background(), objectSpec(domain.StructuredExtractionDeterministic), selected(tt.output))
			if got.Status != tt.status || got.Method != tt.method {
				t.Fatalf("result = %+v", got)
			}
			if tt.code != "" && (len(got.Diagnostics) != 1 || got.Diagnostics[0].Code != tt.code) {
				t.Fatalf("diagnostics = %+v", got.Diagnostics)
			}
			if got.Status == domain.StructuredResultSuccess && got.SourceCandidateID != "candidate-1" {
				t.Fatalf("source candidate = %q", got.SourceCandidateID)
			}
		})
	}
}

func TestResolverSelectionAndFallbackOutcomes(t *testing.T) { // [REQ:REQ-P2-001]
	spec := objectSpec(domain.StructuredExtractionConstrained)

	ambiguous := selected(`{"answer":"yes"}`)
	ambiguous.Selection.Status = domain.FinalOutputSelectionAmbiguous
	if got := (Resolver{}).Resolve(context.Background(), spec, ambiguous); got.Status != domain.StructuredResultAmbiguous {
		t.Fatalf("ambiguous = %+v", got)
	}
	unavailable := selected("")
	unavailable.Selection.Status = domain.FinalOutputSelectionUnavailable
	if got := (Resolver{}).Resolve(context.Background(), spec, unavailable); got.Status != domain.StructuredResultUnavailable {
		t.Fatalf("unavailable = %+v", got)
	}
	if got := (Resolver{}).Resolve(context.Background(), spec, selected("not json")); got.Status != domain.StructuredResultAbstained {
		t.Fatalf("missing extractor = %+v", got)
	}

	extractor := &fakeExtractor{response: ExtractResponse{
		Candidate: json.RawMessage(`{"answer":"recovered"}`), Provider: "fake", Model: "portable",
	}}
	got := (Resolver{Extractor: extractor}).Resolve(context.Background(), spec, selected("not json"))
	if got.Status != domain.StructuredResultSuccess || got.Method != "constrained_extractor" || got.Extractor.Provider != "fake" {
		t.Fatalf("fallback success = %+v", got)
	}
	if extractor.request.RoleRef != DefaultExtractRole || len(extractor.request.Source) == 0 || len(extractor.request.Schema) == 0 {
		t.Fatalf("extract request = %+v", extractor.request)
	}

	extractor.response.Candidate = json.RawMessage(`{"answer":42}`)
	if got := (Resolver{Extractor: extractor}).Resolve(context.Background(), spec, selected("not json")); got.Status != domain.StructuredResultInvalid {
		t.Fatalf("hallucinated invalid candidate = %+v", got)
	}
	extractor.err = errors.New("provider outage containing SECRET_TOKEN")
	got = (Resolver{Extractor: extractor}).Resolve(context.Background(), spec, selected("not json"))
	if got.Status != domain.StructuredResultAbstained || strings.Contains(got.Diagnostics[0].Message, "SECRET_TOKEN") {
		t.Fatalf("outage diagnostic leaked source error: %+v", got)
	}
}

func TestClassificationAbstainsInsteadOfFabricating(t *testing.T) {
	spec := ClassificationSpec("complete", "blocked")
	got := (Resolver{}).Resolve(context.Background(), spec, selected(`"maybe"`))
	if got.Status != domain.StructuredResultAbstained || len(got.Value) != 0 {
		t.Fatalf("classification = %+v", got)
	}
}

func TestResolverBoundsSource(t *testing.T) {
	got := (Resolver{}).Resolve(context.Background(), objectSpec(domain.StructuredExtractionDeterministic), selected(strings.Repeat("x", MaxSourceBytes+1)))
	if got.Status != domain.StructuredResultInvalid || got.Diagnostics[0].Code != "source_too_large" {
		t.Fatalf("oversize = %+v", got)
	}
}
