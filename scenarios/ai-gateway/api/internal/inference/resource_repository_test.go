package inference

import (
	"context"
	"errors"
	"testing"

	"ai-gateway/internal/providers"
	providermocks "ai-gateway/internal/providers/mocks"

	"github.com/stretchr/testify/require"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
)

func TestResourceRepositoryResolvesProviderAndModelThroughResourcePolicy(t *testing.T) {
	runner := &providermocks.FakeRunner{Results: map[string]providers.Result{
		"resource-ollama policy resolve --role classify.routing --json":                                                                     {Stdout: `{"role":"classify.routing","model":"qwen3.5:4b","capabilities":["generate","classify"],"sampling_support":{"temperature":"honored"}}`},
		"resource-ollama gateway generate --role classify.routing --json --prompt-stdin --temperature 0 --format {\"enum\":[\"real-bug\"]}": {Stdout: `{"response":"\"real-bug\"","eval_count":4}`},
	}}
	catalog := RoleCatalog{SchemaVersion: 1, Roles: map[string]InferenceRole{
		"classify.fast": {Sampling: deterministicRoleSampling(), Candidates: []Candidate{{Provider: "ollama", ResourceRole: "classify.routing", Reason: "test"}}},
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

func TestResourceRepositoryRemoteOnlySkipsLocalCandidates(t *testing.T) {
	runner := &providermocks.FakeRunner{
		Results: map[string]providers.Result{
			"resource-openrouter policy resolve --role extract.structured --json": {Stdout: `{"role":"extract.structured","model":"policy-selected"}`},
		},
		DefaultOK: true,
		Default:   providers.Result{Stdout: `{"choices":[{"message":{"content":"{\"type\":\"wait\"}"}}],"usage":{"prompt_tokens":12,"completion_tokens":3}}`},
	}
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
	result, err := repository.Run(context.Background(), ProviderRequest{
		Source: "browser screenshot", SchemaJSON: `{"type":"object"}`, Role: "extract.structured",
		Profile: sharedv1.Profile_PROFILE_REMOTE_ONLY,
	})
	require.NoError(t, err)
	require.Equal(t, "openrouter", result.Provider)
	require.Len(t, runner.Commands, 2)
	require.Equal(t, "resource-openrouter", runner.Commands[0].Name)
	require.Equal(t, "resource-openrouter", runner.Commands[1].Name)
}

func TestDecodeProviderResponseRejectsEmptyResponses(t *testing.T) {
	_, _, _, _, err := decodeProviderResponse(providers.ProviderOllama, `{"response":""}`, "prompt")
	require.Error(t, err)
	require.False(t, errors.Is(err, ErrUnavailable))
}

// deterministicRoleSampling mirrors what classify.fast / extract.structured /
// judge.default declare in the shipped catalog: pinned at 0 and closed to
// caller override.
func deterministicRoleSampling() *RoleSampling {
	temperature := 0.0
	return &RoleSampling{Temperature: &temperature}
}

func overridableRoleSampling(temperature float64) *RoleSampling {
	return &RoleSampling{Temperature: &temperature, Overridable: true}
}

func resolveStdout(role, model string, support providers.SamplingSupport) string {
	return `{"role":"` + role + `","model":"` + model + `","sampling_support":{"temperature":"` + string(support) + `"}}`
}

// The determinism invariant must hold on the hosted rung too. Before the role
// declared its own stance the repository pinned every call at 0 but never
// passed the flag to OpenRouter, so the documented invariant was false there.
func TestResourceRepositoryDeterministicRoleReachesOpenRouter(t *testing.T) {
	openRouterBody := `{"choices":[{"message":{"content":"{\"ok\":true}"}}],"usage":{"prompt_tokens":9,"completion_tokens":2}}`
	runner := &providermocks.FakeRunner{Results: map[string]providers.Result{
		"resource-openrouter policy resolve --role extract.structured --json": {Stdout: resolveStdout("extract.structured", "vendor/model", providers.SamplingHonored)},
		`resource-openrouter generate --role extract.structured --json --temperature 0 --response-format {"type":"json_schema","json_schema":{"name":"vrooli_typed_value","strict":true,"schema":{"type":"object"}}}`: {Stdout: openRouterBody},
	}}
	catalog := RoleCatalog{SchemaVersion: 1, Roles: map[string]InferenceRole{
		"extract.structured": {Sampling: deterministicRoleSampling(), Candidates: []Candidate{
			{Provider: "openrouter", ResourceRole: "extract.structured", Reason: "hosted"},
		}},
	}}
	repository, err := NewResourceRepository(catalog, []providers.Adapter{
		{Provider: providers.ProviderOpenRouter, CommandName: "resource-openrouter", Runner: runner},
	})
	require.NoError(t, err)
	result, err := repository.Run(context.Background(), ProviderRequest{Source: "s", SchemaJSON: `{"type":"object"}`, Role: "extract.structured"})
	require.NoError(t, err)
	require.NotNil(t, result.Applied.TemperatureSent)
	require.Equal(t, 0.0, *result.Applied.TemperatureSent)
	require.Equal(t, providers.SamplingHonored, result.Applied.TemperatureSupport)
}

// On the role-declared path the question is not "will this be honoured" but
// "will sending it break the call". A provider that ignores the control does not
// break, so the role's stated intent is still sent — and the caller still learns
// from temperature_support that it had no effect.
func TestResourceRepositorySendsRoleTemperatureToIgnoringCandidate(t *testing.T) {
	openRouterBody := `{"choices":[{"message":{"content":"{\"ok\":true}"}}],"usage":{"prompt_tokens":9,"completion_tokens":2}}`
	runner := &providermocks.FakeRunner{Results: map[string]providers.Result{
		"resource-openrouter policy resolve --role extract.structured --json": {Stdout: resolveStdout("extract.structured", "vendor/model", providers.SamplingIgnored)},
		`resource-openrouter generate --role extract.structured --json --temperature 0 --response-format {"type":"json_schema","json_schema":{"name":"vrooli_typed_value","strict":true,"schema":{"type":"object"}}}`: {Stdout: openRouterBody},
	}}
	catalog := RoleCatalog{SchemaVersion: 1, Roles: map[string]InferenceRole{
		"extract.structured": {Sampling: deterministicRoleSampling(), Candidates: []Candidate{
			{Provider: "openrouter", ResourceRole: "extract.structured", Reason: "hosted"},
		}},
	}}
	repository, err := NewResourceRepository(catalog, []providers.Adapter{
		{Provider: providers.ProviderOpenRouter, CommandName: "resource-openrouter", Runner: runner},
	})
	require.NoError(t, err)
	result, err := repository.Run(context.Background(), ProviderRequest{Source: "s", SchemaJSON: `{"type":"object"}`, Role: "extract.structured"})
	require.NoError(t, err)
	require.NotNil(t, result.Applied.TemperatureSent)
	require.Equal(t, providers.SamplingIgnored, result.Applied.TemperatureSupport,
		"the caller learns the control had no effect even though it was sent")
}

// An undeclared resource is best-effort, which means try it. Dropping the role's
// stated determinism because a resource has not published its declaration yet
// would silently make a deterministic role non-deterministic.
func TestResourceRepositorySendsRoleTemperatureToUndeclaredCandidate(t *testing.T) {
	runner := &providermocks.FakeRunner{Results: map[string]providers.Result{
		"resource-ollama policy resolve --role classify.routing --json":                                                             {Stdout: `{"role":"classify.routing","model":"qwen3.5:4b"}`},
		`resource-ollama gateway generate --role classify.routing --json --prompt-stdin --temperature 0 --format {"type":"object"}`: {Stdout: `{"response":"{\"ok\":true}","eval_count":2}`},
	}}
	catalog := RoleCatalog{SchemaVersion: 1, Roles: map[string]InferenceRole{
		"classify.fast": {Sampling: deterministicRoleSampling(), Candidates: []Candidate{
			{Provider: "ollama", ResourceRole: "classify.routing", Reason: "local"},
		}},
	}}
	repository, err := NewResourceRepository(catalog, []providers.Adapter{
		{Provider: providers.ProviderOllama, CommandName: "resource-ollama", Runner: runner},
	})
	require.NoError(t, err)
	result, err := repository.Run(context.Background(), ProviderRequest{Source: "s", SchemaJSON: `{"type":"object"}`, Role: "classify.fast"})
	require.NoError(t, err)
	require.NotNil(t, result.Applied.TemperatureSent)
	require.Equal(t, providers.SamplingUnknown, result.Applied.TemperatureSupport)
}

// A rejecting candidate is the one case where sending would break the call, so
// the role default is dropped and the candidate still serves the request.
func TestResourceRepositoryOmitsRoleTemperatureOnRejectingCandidate(t *testing.T) {
	openRouterBody := `{"choices":[{"message":{"content":"{\"ok\":true}"}}],"usage":{"prompt_tokens":9,"completion_tokens":2}}`
	runner := &providermocks.FakeRunner{Results: map[string]providers.Result{
		"resource-openrouter policy resolve --role write.diverse --json": {Stdout: resolveStdout("write.diverse", "vendor/rejecting", providers.SamplingRejected)},
		`resource-openrouter generate --role write.diverse --json --response-format {"type":"json_schema","json_schema":{"name":"vrooli_typed_value","strict":true,"schema":{"type":"object"}}}`: {Stdout: openRouterBody},
	}}
	catalog := RoleCatalog{SchemaVersion: 1, Roles: map[string]InferenceRole{
		"write.diverse": {Sampling: overridableRoleSampling(1.0), Candidates: []Candidate{
			{Provider: "openrouter", ResourceRole: "write.diverse", Reason: "hosted"},
		}},
	}}
	repository, err := NewResourceRepository(catalog, []providers.Adapter{
		{Provider: providers.ProviderOpenRouter, CommandName: "resource-openrouter", Runner: runner},
	})
	require.NoError(t, err)
	result, err := repository.Run(context.Background(), ProviderRequest{Source: "s", SchemaJSON: `{"type":"object"}`, Role: "write.diverse"})
	require.NoError(t, err)
	require.Nil(t, result.Applied.TemperatureSent, "sending to a rejecting provider would 400 the call")
	require.Equal(t, providers.SamplingRejected, result.Applied.TemperatureSupport)
}

// A caller-supplied control sent to a role that has not opened itself is a
// request defect. Walking further candidates could only repeat the answer, so
// the walk stops immediately.
func TestResourceRepositoryRejectsCallerTemperatureOnDeterministicRole(t *testing.T) {
	runner := &providermocks.FakeRunner{Results: map[string]providers.Result{
		"resource-ollama policy resolve --role classify.routing --json": {Stdout: resolveStdout("classify.routing", "qwen3.5:4b", providers.SamplingHonored)},
	}}
	catalog := RoleCatalog{SchemaVersion: 1, Roles: map[string]InferenceRole{
		"judge.default": {Sampling: deterministicRoleSampling(), Candidates: []Candidate{
			{Provider: "ollama", ResourceRole: "classify.routing", Reason: "local"},
		}},
	}}
	repository, err := NewResourceRepository(catalog, []providers.Adapter{
		{Provider: providers.ProviderOllama, CommandName: "resource-ollama", Runner: runner},
	})
	require.NoError(t, err)
	temperature := 1.2
	_, err = repository.Run(context.Background(), ProviderRequest{
		Source: "s", SchemaJSON: `{"type":"object"}`, Role: "judge.default", Temperature: &temperature,
	})
	require.ErrorIs(t, err, ErrRoleForbidsSampling)
	require.Contains(t, err.Error(), "judge.default")
}

// A caller-supplied control is a promise. When no candidate can keep it the
// gateway refuses by name instead of silently sampling some other way.
func TestResourceRepositoryReportsUnsupportedSamplingWhenNoCandidateHonors(t *testing.T) {
	runner := &providermocks.FakeRunner{Results: map[string]providers.Result{
		"resource-openrouter policy resolve --role write.default --json": {Stdout: resolveStdout("write.default", "vendor/model", providers.SamplingRejected)},
	}}
	catalog := RoleCatalog{SchemaVersion: 1, Roles: map[string]InferenceRole{
		"write.default": {Sampling: overridableRoleSampling(0.9), Candidates: []Candidate{
			{Provider: "openrouter", ResourceRole: "write.default", Reason: "hosted"},
		}},
	}}
	repository, err := NewResourceRepository(catalog, []providers.Adapter{
		{Provider: providers.ProviderOpenRouter, CommandName: "resource-openrouter", Runner: runner},
	})
	require.NoError(t, err)
	temperature := 1.2
	_, err = repository.Run(context.Background(), ProviderRequest{
		Source: "s", SchemaJSON: `{"type":"object"}`, Role: "write.default", Temperature: &temperature,
	})
	require.ErrorIs(t, err, ErrUnsupportedSampling)
	require.Contains(t, err.Error(), "rejected")
}

// The walk continues past a candidate that cannot honour the caller's control,
// and the skip is recorded by name rather than swallowed.
func TestResourceRepositorySkipsUnhonouringCandidateAndContinues(t *testing.T) {
	openRouterBody := `{"choices":[{"message":{"content":"{\"ok\":true}"}}],"usage":{"prompt_tokens":9,"completion_tokens":2}}`
	runner := &providermocks.FakeRunner{Results: map[string]providers.Result{
		"resource-openrouter policy resolve --role write.diverse --json":                                                                                                                         {Stdout: resolveStdout("write.diverse", "vendor/rejecting", providers.SamplingRejected)},
		"resource-ollama policy resolve --role write.default --json":                                                                                                                             {Stdout: resolveStdout("write.default", "gemma4:12b", providers.SamplingHonored)},
		`resource-ollama gateway generate --role write.default --json --prompt-stdin --temperature 1.2 --format {"type":"object"}`:                                                               {Stdout: `{"response":"{\"ok\":true}","eval_count":3}`},
		`resource-openrouter generate --role write.diverse --json --response-format {"type":"json_schema","json_schema":{"name":"vrooli_typed_value","strict":true,"schema":{"type":"object"}}}`: {Stdout: openRouterBody},
	}}
	catalog := RoleCatalog{SchemaVersion: 1, Roles: map[string]InferenceRole{
		"write.diverse": {Sampling: overridableRoleSampling(1.0), Candidates: []Candidate{
			{Provider: "openrouter", ResourceRole: "write.diverse", Reason: "hosted"},
			{Provider: "ollama", ResourceRole: "write.default", Reason: "local"},
		}},
	}}
	repository, err := NewResourceRepository(catalog, []providers.Adapter{
		{Provider: providers.ProviderOpenRouter, CommandName: "resource-openrouter", Runner: runner},
		{Provider: providers.ProviderOllama, CommandName: "resource-ollama", Runner: runner},
	})
	require.NoError(t, err)
	temperature := 1.2
	result, err := repository.Run(context.Background(), ProviderRequest{
		Source: "s", SchemaJSON: `{"type":"object"}`, Role: "write.diverse", Temperature: &temperature,
	})
	require.NoError(t, err)
	require.Equal(t, "ollama", result.Provider, "the honouring candidate serves the request")
	require.NotNil(t, result.Applied.TemperatureSent)
	require.Equal(t, 1.2, *result.Applied.TemperatureSent)
}

func TestResourceRepositoryReportsOutputCapSource(t *testing.T) {
	body := `{"response":"{\"ok\":true}","eval_count":3}`
	cases := []struct {
		name       string
		roleCap    string
		requestCap int32
		wantCap    int32
		wantSource OutputCapSource
		command    string
	}{
		{
			name:       "caller cap wins and is transmitted",
			roleCap:    `,"max_tokens":8192`,
			requestCap: 2048,
			wantCap:    2048,
			wantSource: OutputCapRequest,
			command:    `resource-ollama gateway generate --role write.default --json --prompt-stdin --max-tokens 2048 --temperature 0.9 --format {"type":"object"}`,
		},
		{
			name:       "role cap is reported but not re-sent",
			roleCap:    `,"max_tokens":8192`,
			wantCap:    8192,
			wantSource: OutputCapRolePolicy,
			command:    `resource-ollama gateway generate --role write.default --json --prompt-stdin --temperature 0.9 --format {"type":"object"}`,
		},
		{
			name:       "neither means nothing was imposed, which is not the same as unknown",
			wantCap:    0,
			wantSource: OutputCapNoneImposed,
			command:    `resource-ollama gateway generate --role write.default --json --prompt-stdin --temperature 0.9 --format {"type":"object"}`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			runner := &providermocks.FakeRunner{Results: map[string]providers.Result{
				"resource-ollama policy resolve --role write.default --json": {Stdout: `{"role":"write.default","model":"gemma4:12b","sampling_support":{"temperature":"honored"}` + tc.roleCap + `}`},
				tc.command: {Stdout: body},
			}}
			catalog := RoleCatalog{SchemaVersion: 1, Roles: map[string]InferenceRole{
				"write.default": {Sampling: overridableRoleSampling(0.9), Candidates: []Candidate{
					{Provider: "ollama", ResourceRole: "write.default", Reason: "local"},
				}},
			}}
			repository, err := NewResourceRepository(catalog, []providers.Adapter{
				{Provider: providers.ProviderOllama, CommandName: "resource-ollama", Runner: runner},
			})
			require.NoError(t, err)
			result, err := repository.Run(context.Background(), ProviderRequest{
				Source: "s", SchemaJSON: `{"type":"object"}`, Role: "write.default", MaxOutputTokens: tc.requestCap,
			})
			require.NoError(t, err)
			require.Equal(t, tc.wantCap, result.Applied.MaxOutputTokens)
			require.Equal(t, tc.wantSource, result.Applied.MaxOutputTokensSource)
		})
	}
}

func TestResourceRepositoryRefusesContextOverflowBeforeProviderDispatch(t *testing.T) {
	runner := &providermocks.FakeRunner{Results: map[string]providers.Result{
		"resource-ollama policy resolve --role write.default --json": {Stdout: `{"role":"write.default","model":"local-model","context_window":10,"sampling_support":{"temperature":"honored"}}`},
	}}
	catalog := RoleCatalog{SchemaVersion: 1, Roles: map[string]InferenceRole{
		"write.default": {Candidates: []Candidate{{Provider: "ollama", ResourceRole: "write.default", Reason: "local"}}},
	}}
	repository, err := NewResourceRepository(catalog, []providers.Adapter{{Provider: providers.ProviderOllama, CommandName: "resource-ollama", Runner: runner}})
	require.NoError(t, err)
	_, err = repository.Run(context.Background(), ProviderRequest{Source: "This prompt is intentionally longer than ten provider tokens.", Instruction: "write", SchemaJSON: `{"type":"object"}`, Role: "write.default", MaxOutputTokens: 8})
	require.ErrorIs(t, err, ErrContextOverflow)
	for _, command := range runner.Commands {
		require.NotContains(t, command.String(), "gateway generate", "overflow must be refused before provider dispatch")
	}
}
