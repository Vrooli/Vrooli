package inference

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
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
	for _, construct := range []string{"anyOf", "oneOf", "additionalProperties", "minLength", "format"} {
		t.Run(construct, func(t *testing.T) {
			_, err := gate.Parse(`{"type":"object","` + construct + `":{}}`)
			require.Error(t, err)
			var schemaErr *SchemaError
			require.ErrorAs(t, err, &schemaErr)
			require.Equal(t, construct, schemaErr.Construct)
		})
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
