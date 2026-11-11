package inference

import (
	"context"
	"errors"
	"strings"
	"testing"

	"ai-gateway/internal/providers"
	providermocks "ai-gateway/internal/providers/mocks"
	"github.com/vrooli/ai-provider-go/policy"

	"github.com/stretchr/testify/require"
	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
)

func TestResourceRepositoryResolvesProviderAndModelThroughResourcePolicy(t *testing.T) {
	runner := &providermocks.FakeRunner{Results: map[string]providers.Result{
		"resource-ollama gateway generate --role classify.routing --json --prompt-stdin --temperature 0 --format {\"enum\":[\"real-bug\"]}": {Stdout: `{"response":"\"real-bug\"","eval_count":4}`},
	}}
	runner.DefaultOK = true
	runner.Default = providers.Result{Stdout: `{"response":"\"real-bug\"","eval_count":4}`}
	configureQualifiedLocalRole(runner, "classify.routing", "qwen3.5:4b", "local")
	catalog := RoleCatalog{SchemaVersion: 1, Roles: map[string]InferenceRole{
		"classify.fast": {Sampling: deterministicRoleSampling(), Candidates: []Candidate{{Provider: "ollama", ResourceRole: "classify.routing", Reason: "test"}}},
	}}
	repository, err := NewResourceRepositoryWithRightsContextSource(catalog, []providers.Adapter{{Provider: providers.ProviderOllama, Locality: "local", CommandName: "resource-ollama", Runner: runner}}, fixtureRightsContext)
	require.NoError(t, err)
	result, err := repository.Run(context.Background(), ProviderRequest{Source: "failure", SchemaJSON: `{"enum":["real-bug"]}`, Instruction: "Classify.", Role: "classify.fast"})
	require.NoError(t, err)
	require.Equal(t, `"real-bug"`, result.ValueJSON)
	require.Equal(t, "ollama", result.Provider)
	require.Equal(t, "qwen3.5:4b", result.Model)
	require.NotZero(t, result.InputTokens)
	require.Equal(t, int64(4), result.OutputTokens)
}

func TestTypedEmbeddingRequiresLocalOnlyAndCarriesItToResourceBoundary(t *testing.T) {
	for _, tc := range []struct {
		name     string
		locality string
		wantRun  bool
	}{
		{name: "local resource", locality: "local", wantRun: true},
		{name: "remote resource", locality: "remote", wantRun: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			digest := strings.Repeat("c", 64)
			runner := &providermocks.FakeRunner{}
			runner.DefaultOK = true
			runner.Default = providers.Result{Stdout: `{"embedding":[1,0]}`}
			configureQualifiedLocalRole(runner, "embed.resource", "embed-model", tc.locality)
			catalog := RoleCatalog{SchemaVersion: 1, Roles: map[string]InferenceRole{
				"embedding.default": {Candidates: []Candidate{{Provider: providers.ProviderOllama, ResourceRole: "embed.resource", Reason: "fixture"}}},
			}}
			repository, err := NewResourceRepositoryWithRightsContextSource(catalog, []providers.Adapter{{Provider: providers.ProviderOllama, CommandName: "resource-ollama", Locality: tc.locality, Runner: runner}}, fixtureRightsContext)
			require.NoError(t, err)
			request := EmbeddingRequest{Role: "embedding.default", Texts: []string{"hello"}, Profile: sharedv1.Profile_PROFILE_LOCAL_ONLY}
			_, err = repository.Embed(context.Background(), request)
			if tc.wantRun {
				require.NoError(t, err)
				gateway := findCommand(t, runner.Commands, "gateway embed")
				require.Contains(t, gateway.Args, "--require-loopback")
				require.Contains(t, gateway.Args, "--expected-model")
				require.Contains(t, gateway.Args, "--expected-artifact-digest")
				require.Contains(t, gateway.Args, "--expected-rights-decision-sha256")
				require.Contains(t, gateway.Args, "--expected-rights-evaluated-at")
			} else {
				require.ErrorIs(t, err, ErrUnavailable)
				for _, command := range runner.Commands {
					require.NotContains(t, command.String(), "gateway embed", "remote candidate must be refused before provider execution")
				}
			}
		})
	}
}

func TestTypedEmbeddingRejectsMissingOrChangedRuntimeDigestBeforeExecute(t *testing.T) {
	for _, tc := range []struct {
		name           string
		identityDigest string
		inventory      string
	}{
		{name: "missing identity", identityDigest: "", inventory: strings.Repeat("c", 64)},
		{name: "inventory changed after role resolution", identityDigest: strings.Repeat("c", 64), inventory: strings.Repeat("d", 64)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			roleResolution := `{"role":"embed.resource","model":"embed-model","locality":"local"}`
			if tc.identityDigest != "" {
				roleResolution = embeddingRoleResolution("local", tc.identityDigest)
			}
			runner := &providermocks.FakeRunner{Results: map[string]providers.Result{
				"resource-ollama policy resolve --role embed.resource --json": {Stdout: roleResolution},
				"resource-ollama models inventory --json":                     {Stdout: `{"models":[{"name":"embed-model","digest":"` + tc.inventory + `"}]}`},
			}}
			catalog := RoleCatalog{SchemaVersion: 1, Roles: map[string]InferenceRole{
				"embedding.default": {Candidates: []Candidate{{Provider: providers.ProviderOllama, ResourceRole: "embed.resource", Reason: "fixture"}}},
			}}
			repository, err := NewResourceRepositoryWithRightsContextSource(catalog, []providers.Adapter{{Provider: providers.ProviderOllama, Locality: "local", CommandName: "resource-ollama", Runner: runner}}, fixtureRightsContext)
			require.NoError(t, err)
			_, err = repository.Embed(context.Background(), EmbeddingRequest{
				Role: "embedding.default", Texts: []string{"hello"}, Profile: sharedv1.Profile_PROFILE_LOCAL_ONLY,
			})
			require.ErrorIs(t, err, ErrUnavailable)
			require.Len(t, runner.Commands, 2, "digest mismatch must fail before the gateway command")
		})
	}
}

func embeddingRoleResolution(locality, digest string) string {
	rights := fixtureArtifactRights("embed-model")
	identity := policy.RuntimeArtifactIdentity{
		Provider: providers.ProviderOllama, Model: "embed-model", ArtifactID: rights.ArtifactID,
		ArtifactRevision: rights.Revision, DigestKind: policy.RuntimeDigestOllamaModelContents,
		Digest: digest, Source: policy.RuntimeSourceOllamaTags, ObservedAt: "2026-11-11",
	}
	payload := struct {
		Role            string                         `json:"role"`
		Model           string                         `json:"model"`
		Locality        string                         `json:"locality"`
		Rights          policy.ArtifactRights          `json:"rights"`
		RuntimeIdentity policy.RuntimeArtifactIdentity `json:"runtime_identity"`
	}{"embed.resource", "embed-model", locality, rights, identity}
	data, _ := json.Marshal(payload)
	return string(data)
}

func configureQualifiedLocalRole(runner *providermocks.FakeRunner, role, model, locality string) {
	digest := strings.Repeat("c", 64)
	rights := fixtureArtifactRights(model)
	identity := policy.RuntimeArtifactIdentity{
		Provider: providers.ProviderOllama, Model: model, ArtifactID: rights.ArtifactID,
		ArtifactRevision: rights.Revision, DigestKind: policy.RuntimeDigestOllamaModelContents,
		Digest: digest, Source: policy.RuntimeSourceOllamaTags, ObservedAt: "2026-11-11",
	}
	payload := struct {
		Role            string                         `json:"role"`
		Model           string                         `json:"model"`
		Locality        string                         `json:"locality"`
		Rights          policy.ArtifactRights          `json:"rights"`
		RuntimeIdentity policy.RuntimeArtifactIdentity `json:"runtime_identity"`
	}{role, model, locality, rights, identity}
	data, _ := json.Marshal(payload)
	runner.Results["resource-ollama policy resolve --role "+role+" --json"] = providers.Result{Stdout: string(data)}
	runner.Results["resource-ollama models inventory --json"] = providers.Result{Stdout: `{"models":[{"name":"` + model + `","digest":"` + digest + `"}]}`}
}

func fixtureArtifactRights(model string) policy.ArtifactRights {
	return policy.ArtifactRights{
		ArtifactID: "fixture/" + model, Revision: "fixture-revision-1", SHA256: strings.Repeat("a", 64),
		LicenseID: "LicenseRef-fixture", Classification: policy.RightsUnrestricted,
		Evidence: policy.RightsEvidence{URL: "https://example.invalid/fixture-license", Revision: "fixture-license-v1", SHA256: strings.Repeat("b", 64), ObservedAt: "2026-11-10"},
	}
}

func fixtureRightsContext(context.Context) (policy.RightsContext, error) {
	return policy.RightsContext{ID: "fixture-context", Version: "fixture-v1", EntityScope: "fixture-entity", RevenueStatus: "pre_revenue", AsOf: "2026-11-10", Source: "test_fixture"}, nil
}

func findCommand(t *testing.T, commands []providers.Command, contains string) providers.Command {
	t.Helper()
	for _, command := range commands {
		if strings.Contains(command.String(), contains) {
			return command
		}
	}
	t.Fatalf("command containing %q not found in %+v", contains, commands)
	return providers.Command{}
}

func TestTypedEmbeddingRejectsMissingLocalOnlyContractBeforeResolve(t *testing.T) {
	runner := &providermocks.FakeRunner{DefaultOK: true}
	catalog := RoleCatalog{SchemaVersion: 1, Roles: map[string]InferenceRole{
		"embedding.default": {Candidates: []Candidate{{Provider: "ollama", ResourceRole: "embedding.default", Reason: "fixture"}}},
	}}
	repository, err := NewResourceRepository(catalog, []providers.Adapter{{Provider: "ollama", CommandName: "resource-ollama", Locality: "local", Runner: runner}})
	require.NoError(t, err)
	_, err = repository.Embed(context.Background(), EmbeddingRequest{Role: "embedding.default", Texts: []string{"hello"}})
	require.ErrorIs(t, err, ErrUnavailable)
	require.Empty(t, runner.Commands, "an absent transport contract must not reach role resolution or provider execution")
}

func TestMeteredCreditAccountingPreservesKnownZeroAndPendingAbsence(t *testing.T) {
	_, settled, err := decodeProviderResponse(providers.ProviderMetered, `{"content":"{}","credits_charged":0,"accounting_status":"settled","actual_cost_micros":0}`, "prompt")
	require.NoError(t, err)
	require.NotNil(t, settled.creditsCharged)
	require.Zero(t, *settled.creditsCharged)
	require.Equal(t, "settled", settled.accountingStatus)

	_, pending, err := decodeProviderResponse(providers.ProviderMetered, `{"content":"{}","credits_charged":0,"accounting_status":"pending_reconciliation"}`, "prompt")
	require.NoError(t, err)
	require.Nil(t, pending.creditsCharged)
	require.Equal(t, "pending_reconciliation", pending.accountingStatus)

	responseUsage := usage(ProviderResult{CreditsCharged: settled.creditsCharged, AccountingStatus: settled.accountingStatus})
	require.NotNil(t, responseUsage.CreditsCharged)
	require.Zero(t, *responseUsage.CreditsCharged)
	require.Equal(t, "settled", responseUsage.AccountingStatus)
}

func TestDeviceGeometryFundingRolesHaveOneProviderAndNoFallbackCandidates(t *testing.T) {
	catalog, err := LoadCatalog("../../../config/inference-role-catalog.json")
	require.NoError(t, err)
	for role, provider := range map[string]string{
		"device.geometry.byok":           providers.ProviderOpenRouter,
		"device.geometry.vrooli_credits": providers.ProviderMetered,
	} {
		definition, ok := catalog.Roles[role]
		require.True(t, ok, "missing explicit funding role %s", role)
		require.Len(t, definition.Candidates, 1, "funding role %s must not have fallback candidates", role)
		require.Equal(t, provider, definition.Candidates[0].Provider)
		require.Equal(t, "extract.structured", definition.Candidates[0].ResourceRole)
	}
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
		{Provider: providers.ProviderOllama, Locality: "local", CommandName: "resource-ollama", Runner: runner},
		{Provider: providers.ProviderOpenRouter, Locality: "remote", CommandName: "resource-openrouter", Runner: runner},
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
		{Provider: providers.ProviderOllama, Locality: "local", CommandName: "resource-ollama", Runner: runner},
		{Provider: providers.ProviderOpenRouter, Locality: "remote", CommandName: "resource-openrouter", Runner: runner},
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

func TestResourceRepositoryQualityFirstUsesSharedRemoteOrdering(t *testing.T) {
	runner := &providermocks.FakeRunner{
		Results: map[string]providers.Result{
			"resource-openrouter policy resolve --role extract.structured --json": {Stdout: resolveStdout("extract.structured", "hosted-fixture", providers.SamplingHonored)},
			`resource-openrouter generate --role extract.structured --json --temperature 0 --response-format {"type":"json_schema","json_schema":{"name":"vrooli_typed_value","strict":true,"schema":{"type":"object"}}}`: {Stdout: `{"choices":[{"message":{"content":"{\"source\":\"remote\"}"}}],"usage":{"prompt_tokens":3,"completion_tokens":2}}`},
		},
		DefaultOK: true,
	}
	catalog := RoleCatalog{SchemaVersion: 1, Roles: map[string]InferenceRole{
		"extract.structured": {Sampling: deterministicRoleSampling(), Candidates: []Candidate{
			{Provider: "ollama", ResourceRole: "chat.default", Reason: "catalog-first local candidate"},
			{Provider: "openrouter", ResourceRole: "extract.structured", Reason: "remote candidate ranked first for quality-first"},
		}},
	}}
	repository, err := NewResourceRepository(catalog, []providers.Adapter{
		{Provider: providers.ProviderOllama, Locality: "local", CommandName: "resource-ollama", Runner: runner},
		{Provider: providers.ProviderOpenRouter, Locality: "remote", CommandName: "resource-openrouter", Runner: runner},
	})
	require.NoError(t, err)
	result, err := repository.Run(context.Background(), ProviderRequest{
		Source: "source", SchemaJSON: `{"type":"object"}`, Role: "extract.structured",
		Profile: sharedv1.Profile_PROFILE_QUALITY_FIRST,
	})
	require.NoError(t, err)
	require.Equal(t, "openrouter", result.Provider)
	require.NotEmpty(t, runner.Commands)
	require.Equal(t, "resource-openrouter", runner.Commands[0].Name, "typed execution should use the shared quality-first locality order")
}

func TestResourceRepositorySecretPrivacyBlocksRemoteBeforeDispatch(t *testing.T) {
	runner := &providermocks.FakeRunner{DefaultOK: true}
	catalog := RoleCatalog{SchemaVersion: 1, Roles: map[string]InferenceRole{
		"extract.structured": {Candidates: []Candidate{{Provider: "openrouter", ResourceRole: "extract.structured", Reason: "remote"}}},
	}}
	repository, err := NewResourceRepository(catalog, []providers.Adapter{{Provider: providers.ProviderOpenRouter, Locality: "remote", CommandName: "resource-openrouter", Runner: runner}})
	require.NoError(t, err)
	_, err = repository.Run(context.Background(), ProviderRequest{
		Source: "private source", SchemaJSON: `{"type":"object"}`, Role: "extract.structured",
		Profile: sharedv1.Profile_PROFILE_LOCAL_FIRST, PrivacyClass: sharedv1.PrivacyClass_PRIVACY_CLASS_SECRET,
	})
	require.ErrorIs(t, err, ErrUnavailable)
	require.Empty(t, runner.Commands, "secret privacy must block the remote candidate before provider resolution")
}

func TestResourceRepositoryProviderCallLimitCountsFailedDispatchAndStopsCandidateFallback(t *testing.T) {
	failedGenerate := `resource-ollama gateway generate --role chat.one --json --prompt-stdin --require-loopback --temperature 0 --format {"type":"object"}`
	runner := &providermocks.FakeRunner{
		Results: map[string]providers.Result{
			"resource-ollama policy resolve --role chat.one --json": {Stdout: resolveStdout("chat.one", "local-one", providers.SamplingHonored)},
			"resource-ollama policy resolve --role chat.two --json": {Stdout: resolveStdout("chat.two", "local-two", providers.SamplingHonored)},
		},
		Errors:    map[string]error{failedGenerate: errors.New("transport failed")},
		DefaultOK: true,
		Default:   providers.Result{Stdout: `{"response":"{\"ok\":true}"}`},
	}
	catalog := RoleCatalog{SchemaVersion: 1, Roles: map[string]InferenceRole{
		"extract.structured": {Sampling: deterministicRoleSampling(), Candidates: []Candidate{
			{Provider: "ollama", ResourceRole: "chat.one", Reason: "first"},
			{Provider: "ollama", ResourceRole: "chat.two", Reason: "fallback"},
		}},
	}}
	repository, err := NewResourceRepository(catalog, []providers.Adapter{{Provider: providers.ProviderOllama, Locality: "local", CommandName: "resource-ollama", Runner: runner}})
	require.NoError(t, err)
	result, err := repository.Run(context.Background(), ProviderRequest{
		Source: "source", SchemaJSON: `{"type":"object"}`, Role: "extract.structured",
		Profile: sharedv1.Profile_PROFILE_LOCAL_ONLY, MaxProviderCalls: 1,
	})
	require.ErrorIs(t, err, ErrUnavailable)
	require.EqualValues(t, 1, result.ProviderCallsMade)
	var generateCalls int
	for _, command := range runner.Commands {
		if strings.Contains(command.String(), "gateway generate") {
			generateCalls++
		}
	}
	require.Equal(t, 1, generateCalls, "a failed actual dispatch consumes the total bound before fallback")
}

func TestDecodeProviderResponseRejectsEmptyResponses(t *testing.T) {
	_, _, err := decodeProviderResponse(providers.ProviderOllama, `{"response":""}`, "prompt")
	require.Error(t, err)
	require.False(t, errors.Is(err, ErrUnavailable))
}

func TestProviderUsageDistinguishesReportedZeroFromUnknown(t *testing.T) {
	t.Run("Ollama reports zero prompt and completion counts", func(t *testing.T) {
		_, got, err := decodeProviderResponse(providers.ProviderOllama, `{"response":"{}","prompt_eval_count":0,"eval_count":0}`, "prompt")
		require.NoError(t, err)
		require.Zero(t, got.input)
		require.Zero(t, got.output)
		require.Equal(t, inferencev1.UsageProvenance_USAGE_PROVENANCE_PROVIDER_REPORTED, got.inputProvenance)
		require.Equal(t, inferencev1.UsageProvenance_USAGE_PROVENANCE_PROVIDER_REPORTED, got.outputProvenance)
		require.Equal(t, inferencev1.UsageProvenance_USAGE_PROVENANCE_UNKNOWN, got.costProvenance)
	})
	t.Run("Ollama estimates missing prompt and keeps completion unknown", func(t *testing.T) {
		_, got, err := decodeProviderResponse(providers.ProviderOllama, `{"response":"{}"}`, "prompt")
		require.NoError(t, err)
		require.Equal(t, inferencev1.UsageProvenance_USAGE_PROVENANCE_GATEWAY_ESTIMATE, got.inputProvenance)
		require.Equal(t, inferencev1.UsageProvenance_USAGE_PROVENANCE_UNKNOWN, got.outputProvenance)
		require.Equal(t, inferencev1.UsageProvenance_USAGE_PROVENANCE_UNKNOWN, got.costProvenance)
	})
	t.Run("OpenRouter reported zeros remain reported", func(t *testing.T) {
		_, got, err := decodeProviderResponse(providers.ProviderOpenRouter, `{"choices":[{"message":{"content":"{}"}}],"usage":{"prompt_tokens":0,"completion_tokens":0,"cost":0}}`, "prompt")
		require.NoError(t, err)
		require.Zero(t, got.input)
		require.Zero(t, got.output)
		require.Zero(t, got.cost)
		require.Equal(t, inferencev1.UsageProvenance_USAGE_PROVENANCE_PROVIDER_REPORTED, got.inputProvenance)
		require.Equal(t, inferencev1.UsageProvenance_USAGE_PROVENANCE_PROVIDER_REPORTED, got.outputProvenance)
		require.Equal(t, inferencev1.UsageProvenance_USAGE_PROVENANCE_PROVIDER_REPORTED, got.costProvenance)
	})
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
		{Provider: providers.ProviderOpenRouter, Locality: "remote", CommandName: "resource-openrouter", Runner: runner},
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
		{Provider: providers.ProviderOpenRouter, Locality: "remote", CommandName: "resource-openrouter", Runner: runner},
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
		{Provider: providers.ProviderOllama, Locality: "local", CommandName: "resource-ollama", Runner: runner},
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
		{Provider: providers.ProviderOpenRouter, Locality: "remote", CommandName: "resource-openrouter", Runner: runner},
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
		{Provider: providers.ProviderOllama, Locality: "local", CommandName: "resource-ollama", Runner: runner},
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
		{Provider: providers.ProviderOpenRouter, Locality: "remote", CommandName: "resource-openrouter", Runner: runner},
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
		{Provider: providers.ProviderOpenRouter, Locality: "remote", CommandName: "resource-openrouter", Runner: runner},
		{Provider: providers.ProviderOllama, Locality: "local", CommandName: "resource-ollama", Runner: runner},
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
				{Provider: providers.ProviderOllama, Locality: "local", CommandName: "resource-ollama", Runner: runner},
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
	repository, err := NewResourceRepository(catalog, []providers.Adapter{{Provider: providers.ProviderOllama, Locality: "local", CommandName: "resource-ollama", Runner: runner}})
	require.NoError(t, err)
	_, err = repository.Run(context.Background(), ProviderRequest{Source: "This prompt is intentionally longer than ten provider tokens.", Instruction: "write", SchemaJSON: `{"type":"object"}`, Role: "write.default", MaxOutputTokens: 8})
	require.ErrorIs(t, err, ErrContextOverflow)
	for _, command := range runner.Commands {
		require.NotContains(t, command.String(), "gateway generate", "overflow must be refused before provider dispatch")
	}
}

func TestDecodeOpenAICompatibleProviderResponses(t *testing.T) {
	body := `{"choices":[{"message":{"content":"{\"ok\":true}"}}],"usage":{"prompt_tokens":4,"completion_tokens":2}}`
	for _, provider := range []string{providers.ProviderOpenRouter, providers.ProviderOrcaRouter, providers.ProviderAbliteration} {
		got, usage, err := decodeProviderResponse(provider, body, "prompt")
		if err != nil {
			t.Fatalf("%s decode: %v", provider, err)
		}
		if got != `{"ok":true}` || usage.input != 4 || usage.output != 2 {
			t.Fatalf("%s result = %q usage = %+v", provider, got, usage)
		}
	}
}
