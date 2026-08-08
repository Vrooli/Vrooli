package inference

import (
	"context"
	"errors"
	"testing"

	"ai-gateway/internal/providers"
	providermocks "ai-gateway/internal/providers/mocks"
	"github.com/stretchr/testify/require"
)

func TestResourceRepositoryResolvesProviderAndModelThroughResourcePolicy(t *testing.T) {
	runner := &providermocks.FakeRunner{Results: map[string]providers.Result{
		"resource-ollama policy resolve --role classify.routing --json":                                                                     {Stdout: `{"role":"classify.routing","model":"qwen3.5:4b","capabilities":["generate","classify"]}`},
		"resource-ollama gateway generate --role classify.routing --json --prompt-stdin --temperature 0 --format {\"enum\":[\"real-bug\"]}": {Stdout: `{"response":"\"real-bug\"","eval_count":4}`},
	}}
	catalog := RoleCatalog{SchemaVersion: 1, Roles: map[string]InferenceRole{
		"classify.fast": {Candidates: []Candidate{{Provider: "ollama", ResourceRole: "classify.routing", Reason: "test"}}},
	}}
	repository, err := NewResourceRepository(catalog, []providers.Adapter{{Provider: providers.ProviderOllama, CommandName: "resource-ollama", Runner: runner}})
	require.NoError(t, err)
	result, err := repository.Run(context.Background(), ProviderRequest{Source: "failure", SchemaJSON: `{"enum":["real-bug"]}`, Instruction: "Classify.", Role: "classify.fast"})
	require.NoError(t, err)
	require.Equal(t, `"real-bug"`, result.ValueJSON)
	require.Equal(t, "ollama", result.Provider)
	require.Equal(t, "qwen3.5:4b", result.Model)
	require.NotZero(t, result.InputTokens)
	require.Equal(t, int64(4), result.OutputTokens)
}

func TestResourceRepositoryReturnsTypedUnavailableWhenCandidatesFail(t *testing.T) {
	runner := &providermocks.FakeRunner{}
	catalog := RoleCatalog{SchemaVersion: 1, Roles: map[string]InferenceRole{
		"extract.structured": {Candidates: []Candidate{
			{Provider: "ollama", ResourceRole: "chat.default", Reason: "local"},
			{Provider: "openrouter", ResourceRole: "extract.structured", Reason: "hosted"},
		}},
	}}
	repository, err := NewResourceRepository(catalog, []providers.Adapter{
		{Provider: providers.ProviderOllama, CommandName: "resource-ollama", Runner: runner},
		{Provider: providers.ProviderOpenRouter, CommandName: "resource-openrouter", Runner: runner},
	})
	require.NoError(t, err)
	_, err = repository.Run(context.Background(), ProviderRequest{Source: "source", SchemaJSON: `{"type":"object"}`, Role: "extract.structured"})
	require.Error(t, err)
	require.ErrorIs(t, err, ErrUnavailable)
}

func TestDecodeProviderResponseRejectsEmptyResponses(t *testing.T) {
	_, _, _, _, err := decodeProviderResponse(providers.ProviderOllama, `{"response":""}`, "prompt")
	require.Error(t, err)
	require.False(t, errors.Is(err, ErrUnavailable))
}
