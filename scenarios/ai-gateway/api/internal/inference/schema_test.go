package inference

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
)

func TestSchemaGateSupportedSubset(t *testing.T) {
	gate := SchemaGate{}
	for _, test := range []struct {
		name   string
		schema string
		value  string
	}{
		{"type", `{"type":"string"}`, `"ok"`},
		{"enum", `{"enum":["a","b"]}`, `"b"`},
		{"const", `{"const":"a"}`, `"a"`},
		{"required", `{"type":"object","required":["name"],"properties":{"name":{"type":"string"}}}`, `{"name":"vrooli"}`},
		{"items", `{"type":"array","items":{"type":"integer"}}`, `[1,2]`},
		{"pattern", `{"type":"string","pattern":"^[a-z]+$"}`, `"vrooli"`},
		{"minimum-maximum", `{"type":"number","minimum":1,"maximum":3}`, `2`},
		{"exclusive-minimum", `{"type":"number","exclusiveMinimum":0}`, `0.1`},
		{"array-bounds", `{"type":"array","minItems":1,"maxItems":2}`, `[1]`},
		{"string-bounds", `{"type":"string","minLength":1,"maxLength":2}`, `"é"`},
		{"closed-object", `{"type":"object","properties":{"name":{"type":"string"}},"additionalProperties":false}`, `{"name":"ok"}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			schema, err := gate.Parse(test.schema)
			require.NoError(t, err)
			require.NoError(t, ValidateJSON(schema, []byte(test.value)))
		})
	}
}

func TestSchemaGateRejectsUnsupportedConstructs(t *testing.T) {
	gate := SchemaGate{}
	for _, construct := range []string{"anyOf", "oneOf", "format"} {
		t.Run(construct, func(t *testing.T) {
			_, err := gate.Parse(`{"type":"object","` + construct + `":{}}`)
			require.Error(t, err)
			var schemaErr *SchemaError
			require.ErrorAs(t, err, &schemaErr)
			require.Equal(t, construct, schemaErr.Construct)
		})
	}
}

func TestGeometryRecipeSchemaFixtureIsAcceptedAndEnforced(t *testing.T) {
	fixture, err := os.ReadFile(filepath.Join("testdata", "device-studio-geometry-recipe-schema-v1.json"))
	require.NoError(t, err)
	require.Equal(t, "20fe5ac7b117080ea3663ec91145f1b1c6149fbd7683fc7f3f3152a30c3abbfc", fmt.Sprintf("%x", sha256.Sum256(fixture)), "fixture is the retained exact Device Studio geometry.SchemaJSON contract")
	schema, err := (SchemaGate{}).Parse(string(fixture))
	require.NoError(t, err)
	valid := `{"version":"device_geometry_recipe.v1","profile":"detailed_device_v1","operation":"construct","envelope_mm":[80,150,12],"parts":[{"id":"housing","kind":"rounded_box","role":"housing","center_mm":[0,0,0],"size_mm":[80,150,12],"rotation_deg":[0,0,0],"bevel_mm":3,"material":"matte_polymer"},{"id":"display","kind":"box","role":"display","center_mm":[0,0,6],"size_mm":[60,90,1],"rotation_deg":[0,0,0],"bevel_mm":0,"material":"glass"}],"screen":{"part_id":"display","face":"front:+z","center_uv":[0.5,0.5],"size_uv":[0.7,0.5],"rotation_deg":0}}`
	require.NoError(t, ValidateJSON(schema, []byte(valid)))
	for _, invalid := range []string{
		`{"version":"device_geometry_recipe.v1","profile":"detailed_device_v1","operation":"construct","envelope_mm":[0,150,12],"parts":[{"id":"housing","kind":"rounded_box","role":"housing","center_mm":[0,0,0],"size_mm":[80,150,12],"rotation_deg":[0,0,0],"bevel_mm":3,"material":"matte_polymer"}],"screen":{"part_id":"housing","face":"front:+z","center_uv":[0.5,0.5],"size_uv":[0.7,0.5],"rotation_deg":0}}`,
		`{"version":"device_geometry_recipe.v1","profile":"detailed_device_v1","operation":"construct","envelope_mm":[80,150,12,1],"parts":[{"id":"housing","kind":"rounded_box","role":"housing","center_mm":[0,0,0],"size_mm":[80,150,12],"rotation_deg":[0,0,0],"bevel_mm":3,"material":"matte_polymer"}],"screen":{"part_id":"housing","face":"front:+z","center_uv":[0.5,0.5],"size_uv":[0.7,0.5],"rotation_deg":0}}`,
		`{"version":"device_geometry_recipe.v1","profile":"detailed_device_v1","operation":"construct","envelope_mm":[80,150,12],"parts":[],"screen":{"part_id":"housing","face":"front:+z","center_uv":[0.5,0.5],"size_uv":[0.7,0.5],"rotation_deg":0}}`,
		`{"version":"device_geometry_recipe.v1","profile":"detailed_device_v1","operation":"construct","envelope_mm":[80,150,12],"parts":[{"id":"","kind":"rounded_box","role":"housing","center_mm":[0,0,0],"size_mm":[80,150,12],"rotation_deg":[0,0,0],"bevel_mm":3,"material":"matte_polymer"}],"screen":{"part_id":"housing","face":"front:+z","center_uv":[0.5,0.5],"size_uv":[0.7,0.5],"rotation_deg":0}}`,
	} {
		require.Error(t, ValidateJSON(schema, []byte(invalid)))
	}
}

func TestSchemaGateBoundsEnforceTypesAndExclusiveEdges(t *testing.T) {
	cases := []struct {
		schema, value string
		valid         bool
	}{
		{`{"type":"number","exclusiveMinimum":0,"maximum":1}`, `0`, false},
		{`{"type":"number","exclusiveMinimum":0,"maximum":1}`, `0.0001`, true},
		{`{"type":"number","exclusiveMinimum":0,"maximum":1}`, `1`, true},
		{`{"type":"array","minItems":1,"maxItems":2,"items":{"type":"integer"}}`, `[]`, false},
		{`{"type":"array","minItems":1,"maxItems":2,"items":{"type":"integer"}}`, `[1,2,3]`, false},
		{`{"type":"array","minItems":1,"maxItems":2,"items":{"type":"integer"}}`, `["1"]`, false},
		{`{"type":"string","minLength":1,"maxLength":2}`, `""`, false},
		{`{"type":"string","minLength":1,"maxLength":2}`, `"abc"`, false},
		{`{"type":"string","minLength":1,"maxLength":2}`, `"é"`, true},
		{`{"type":"object","properties":{"name":{"type":"string"}},"additionalProperties":false}`, `{"name":"ok","extra":1}`, false},
	}
	for _, test := range cases {
		schema, err := (SchemaGate{}).Parse(test.schema)
		require.NoError(t, err)
		err = ValidateJSON(schema, []byte(test.value))
		if test.valid {
			require.NoError(t, err)
		} else {
			require.Error(t, err)
		}
	}
	for _, invalid := range []string{
		`{"type":"array","minItems":-1}`,
		`{"type":"array","minItems":1.5}`,
		`{"type":"array","minItems":2,"maxItems":1}`,
		`{"type":"object","additionalProperties":{}}`,
	} {
		_, err := (SchemaGate{}).Parse(invalid)
		require.Error(t, err)
	}
}

func TestServiceMarksInvalidProviderValue(t *testing.T) {
	service := NewService(fakeRepository{result: ProviderResult{ValueJSON: `{"kind":"unexpected"}`, Provider: "fake", Model: "test", InputTokens: 2, OutputTokens: 1}})
	response := service.Run(t.Context(), ProviderRequest{Source: "source", SchemaJSON: `{"type":"object","required":["kind"],"properties":{"kind":{"enum":["expected"]}}}`, Role: "extract.structured"})
	require.False(t, response.GetValidated())
	require.Equal(t, "INFERENCE_ERROR_CODE_VALIDATION_FAILED", response.GetError().GetCode().String())
}

func TestServicePassesInstructionAsASeparateProviderField(t *testing.T) {
	repository := &recordingRepository{result: ProviderResult{ValueJSON: `"ok"`, Provider: "fake", Model: "test", InputTokens: 3, OutputTokens: 1}}
	service := NewService(repository)
	response := service.Run(t.Context(), ProviderRequest{
		Source:      "source",
		SchemaJSON:  `{"type":"string"}`,
		Instruction: "Classify by root cause, not surface wording.",
		Role:        "classify.fast",
	})
	require.True(t, response.GetValidated())
	require.Equal(t, "Classify by root cause, not surface wording.", repository.request.Instruction)
}

func TestServiceBatchPreservesOrderAndAggregatesUsage(t *testing.T) {
	repository := &batchRecordingRepository{}
	service := NewService(repository)
	response := service.RunBatch(t.Context(), []ProviderRequest{
		{Source: "first", SchemaJSON: `{"type":"string"}`, Role: "classify.fast"},
		{Source: "second", SchemaJSON: `{"type":"string"}`, Role: "classify.fast"},
	})
	require.Len(t, response.GetResults(), 2)
	require.Equal(t, `"first"`, response.GetResults()[0].GetValueJson())
	require.Equal(t, `"second"`, response.GetResults()[1].GetValueJson())
	require.EqualValues(t, 11, response.GetUsage().GetInputTokens())
	require.EqualValues(t, 2, response.GetUsage().GetOutputTokens())
}

func TestServiceRepairRetainsUsageProvenanceAcrossAttempts(t *testing.T) {
	repository := &provenanceRepairRepository{}
	response := NewService(repository).Run(t.Context(), ProviderRequest{Source: "brief", SchemaJSON: `{"type":"string"}`, Role: "extract.structured"})
	require.True(t, response.GetValidated())
	require.Equal(t, 2, repository.calls)
	require.Equal(t, inferencev1.UsageProvenance_USAGE_PROVENANCE_MIXED, response.GetUsage().GetInputTokensProvenance())
	require.Equal(t, inferencev1.UsageProvenance_USAGE_PROVENANCE_PROVIDER_REPORTED, response.GetUsage().GetOutputTokensProvenance())
	require.Equal(t, inferencev1.UsageProvenance_USAGE_PROVENANCE_UNKNOWN, response.GetUsage().GetCostProvenance())
	// A reported zero remains distinct from absent cost after adding a known
	// zero to an attempt whose cost is unknown.
	require.Zero(t, response.GetUsage().GetCostMicros())
}

type provenanceRepairRepository struct{ calls int }

func (r *provenanceRepairRepository) Run(context.Context, ProviderRequest) (ProviderResult, error) {
	r.calls++
	if r.calls == 1 {
		return ProviderResult{ValueJSON: `42`, InputTokens: 2, OutputTokens: 0, InputTokensProvenance: inferencev1.UsageProvenance_USAGE_PROVENANCE_GATEWAY_ESTIMATE, OutputTokensProvenance: inferencev1.UsageProvenance_USAGE_PROVENANCE_PROVIDER_REPORTED, CostProvenance: inferencev1.UsageProvenance_USAGE_PROVENANCE_UNKNOWN}, nil
	}
	return ProviderResult{ValueJSON: `"repaired"`, InputTokens: 3, OutputTokens: 0, CostMicros: 0, InputTokensProvenance: inferencev1.UsageProvenance_USAGE_PROVENANCE_PROVIDER_REPORTED, OutputTokensProvenance: inferencev1.UsageProvenance_USAGE_PROVENANCE_PROVIDER_REPORTED, CostProvenance: inferencev1.UsageProvenance_USAGE_PROVENANCE_PROVIDER_REPORTED}, nil
}

func TestServiceBatchKeepsOtherItemsWhenOneFailsLocalValidation(t *testing.T) {
	service := NewService(mixedBatchRepository{})
	response := service.RunBatch(t.Context(), []ProviderRequest{
		{Source: "first", SchemaJSON: `{"type":"string"}`, Role: "classify.fast"},
		{Source: "bad", SchemaJSON: `{"type":"string"}`, Role: "classify.fast"},
		{Source: "third", SchemaJSON: `{"type":"string"}`, Role: "classify.fast"},
	})
	require.Len(t, response.GetResults(), 3)
	require.True(t, response.GetResults()[0].GetValidated())
	require.Equal(t, "INFERENCE_ERROR_CODE_VALIDATION_FAILED", response.GetResults()[1].GetError().GetCode().String())
	require.True(t, response.GetResults()[2].GetValidated())
}

func TestServiceBatchRejectsTooManyItems(t *testing.T) {
	service := NewService(fakeRepository{})
	requests := make([]ProviderRequest, MaxBatchSize+1)
	response := service.RunBatch(t.Context(), requests)
	require.Len(t, response.GetResults(), 1)
	require.Equal(t, "INFERENCE_ERROR_CODE_INVALID_REQUEST", response.GetResults()[0].GetError().GetCode().String())
}

type embeddingTestRepository struct {
	result EmbeddingResult
	role   string
	texts  []string
}

func (r *embeddingTestRepository) Run(context.Context, ProviderRequest) (ProviderResult, error) {
	return ProviderResult{}, nil
}

func (r *embeddingTestRepository) Embed(_ context.Context, role string, texts []string) (EmbeddingResult, error) {
	r.role, r.texts = role, append([]string(nil), texts...)
	return r.result, nil
}

func TestServiceEmbedPreservesTypedProviderEvidence(t *testing.T) {
	repository := &embeddingTestRepository{result: EmbeddingResult{Vectors: [][]float64{{1, 0}, {0, 1}}, Provider: "ollama", Model: "nomic", Dimension: 2, InputTokens: 8, CostMicros: 12}}
	response := NewService(repository).Embed(t.Context(), "embedding.default", []string{"one", "two"}, false)
	require.Nil(t, response.GetError())
	require.Equal(t, "embedding.default", repository.role)
	require.Equal(t, "ollama", response.GetProvider())
	require.EqualValues(t, 2, response.GetDimension())
	require.EqualValues(t, 12, response.GetUsage().GetCostMicros())
	require.Len(t, response.GetVectors(), 2)
}

func TestServiceEmbedRejectsSamplingAndDimensionMismatch(t *testing.T) {
	repository := &embeddingTestRepository{result: EmbeddingResult{Vectors: [][]float64{{1}, {1, 0}}, Provider: "fake", Model: "m", Dimension: 2}}
	service := NewService(repository)
	response := service.Embed(t.Context(), "embedding.default", []string{"one", "two"}, true)
	require.Equal(t, "INFERENCE_ERROR_CODE_INVALID_REQUEST", response.GetError().GetCode().String())
	response = service.Embed(t.Context(), "embedding.default", []string{"one", "two"}, false)
	require.Equal(t, "INFERENCE_ERROR_CODE_VALIDATION_FAILED", response.GetError().GetCode().String())
}

type fakeRepository struct {
	result ProviderResult
	err    error
}

func (f fakeRepository) Run(_ context.Context, _ ProviderRequest) (ProviderResult, error) {
	return f.result, f.err
}

type recordingRepository struct {
	result  ProviderResult
	request ProviderRequest
}

func (r *recordingRepository) Run(_ context.Context, request ProviderRequest) (ProviderResult, error) {
	r.request = request
	return r.result, nil
}

type batchRecordingRepository struct{}

func (batchRecordingRepository) Run(_ context.Context, request ProviderRequest) (ProviderResult, error) {
	return ProviderResult{ValueJSON: `"` + request.Source + `"`, InputTokens: int64(len(request.Source)), OutputTokens: 1}, nil
}

type mixedBatchRepository struct{}

func (mixedBatchRepository) Run(_ context.Context, request ProviderRequest) (ProviderResult, error) {
	if request.Source == "bad" {
		return ProviderResult{ValueJSON: `42`, InputTokens: 1, OutputTokens: 1}, nil
	}
	return ProviderResult{ValueJSON: `"` + request.Source + `"`, InputTokens: 1, OutputTokens: 1}, nil
}
