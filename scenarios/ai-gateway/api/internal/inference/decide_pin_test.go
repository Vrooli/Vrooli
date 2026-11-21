package inference

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/ai-provider-go/policy"

	"ai-gateway/internal/providers"
	providermocks "ai-gateway/internal/providers/mocks"
	"ai-gateway/internal/rolecatalog"

	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
)

const bakeoffRole = "decide.bakeoff"

// bakeoff is a decision role with several local candidates, the shape of
// decide.candidates, over a runner that serves each one as a fully qualified
// Ollaya role (Apache-style behavioral-restriction rights, exact manifest
// identity, declared limits) unless a test scripts it otherwise.
type bakeoff struct {
	runner  *providermocks.FakeRunner
	service *Service
	enum    int
}

func newBakeoff(t *testing.T, roles ...string) *bakeoff {
	t.Helper()
	world := &bakeoff{runner: &providermocks.FakeRunner{Results: map[string]providers.Result{}, Errors: map[string]error{}, DefaultOK: true}}
	world.runner.Default = providers.Result{Stdout: answeredOutput}
	candidates := make([]rolecatalog.Candidate, 0, len(roles)+1)
	for _, role := range roles {
		world.qualifies(role, &providers.DecisionLimits{ContextTokens: 8192, MaxOptions: 255})
		candidates = append(candidates, rolecatalog.Candidate{Provider: "ollaya", ResourceRole: role, Reason: "bake-off candidate"})
	}
	candidates = append(candidates, rolecatalog.Candidate{Provider: "ai-gateway", ResourceRole: "classify.fast", Reason: "enum baseline"})
	catalog := rolecatalog.RoleCatalog{SchemaVersion: 1, Roles: map[string]rolecatalog.InferenceRole{
		bakeoffRole:     {Kind: rolecatalog.RoleKindDecision, TimeoutMS: 30000, Candidates: candidates},
		"classify.fast": {Candidates: []rolecatalog.Candidate{{Provider: "ollama", ResourceRole: "classify.routing", Reason: "local"}}},
	}}
	repository, err := NewResourceRepositoryWithRightsContextSource(catalog, []providers.Adapter{
		{Provider: providers.ProviderOllaya, Locality: "local", CommandName: "resource-ollaya", Runner: world.runner},
	}, fixtureRightsContext)
	require.NoError(t, err)
	world.service = NewService(repository)
	return world
}

// qualifies scripts one Ollaya role as resolvable with verified rights.
func (w *bakeoff) qualifies(role string, limits *providers.DecisionLimits) {
	model := "m-" + role
	digest := strings.Repeat("d", 64)
	rights := fixtureArtifactRights(model)
	rights.Classification = policy.RightsBehavioralRestrictions
	rights.Obligations = []string{"preserve the Apache-2.0 copyright and license notice"}
	w.resolvesWith(role, model, digest, rights, limits)
}

// unknownRights scripts one Ollaya role whose rights are an explicit unknown.
func (w *bakeoff) unknownRights(role string) {
	model := "m-" + role
	rights := fixtureArtifactRights(model)
	rights.Classification = policy.RightsUnverified
	rights.UnverifiedReason = "training_data_includes_non_commercial_sources_scope_not_reviewed"
	w.resolvesWith(role, model, strings.Repeat("d", 64), rights, &providers.DecisionLimits{ContextTokens: 512, MaxOptions: 255})
}

func (w *bakeoff) resolvesWith(role, model, digest string, rights policy.ArtifactRights, limits *providers.DecisionLimits) {
	identity := policy.RuntimeArtifactIdentity{
		Provider: providers.ProviderOllaya, Model: model, ArtifactID: rights.ArtifactID,
		ArtifactRevision: rights.Revision, DigestKind: policy.RuntimeDigestOllayaManifest,
		Digest: digest, Source: policy.RuntimeSourceOllayaTags, ObservedAt: "2026-11-11",
	}
	payload, err := json.Marshal(map[string]any{
		"role": role, "model": model, "canonical_model": model, "locality": "local", "capabilities": []string{"decide"},
		"schema_version": "1", "policy_sha256": strings.Repeat("f", 64),
		"rights": rights, "runtime_identity": identity, "observed_runtime_digest": digest, "expected_artifact_digest": digest,
		"limits": limits, "placement": map[string]any{"device": "gpu", "vram_bytes": 1 << 30},
	})
	if err != nil {
		panic(err)
	}
	w.runner.Results[resolveKey(role)] = providers.Result{Stdout: string(payload)}
}

func (w *bakeoff) dispatched() []string {
	var roles []string
	for _, command := range w.runner.Commands {
		text := command.String()
		if strings.Contains(text, "gateway decide --role ") {
			roles = append(roles, strings.Fields(strings.SplitN(text, "--role ", 2)[1])[0])
		}
	}
	return roles
}

func (w *bakeoff) resolved() []string {
	var roles []string
	for _, command := range w.runner.Commands {
		text := command.String()
		if strings.Contains(text, "policy resolve --role ") {
			roles = append(roles, strings.Fields(strings.SplitN(text, "--role ", 2)[1])[0])
		}
	}
	return roles
}

func pinned(pin string) DecisionRequest {
	request := operatorMessage()
	request.Role, request.PinResourceRole = bakeoffRole, pin
	request.Profile = sharedv1.Profile_PROFILE_LOCAL_ONLY
	request.PrivacyClass = sharedv1.PrivacyClass_PRIVACY_CLASS_CONFIDENTIAL
	return request
}

func TestPinnedRequestIsAnsweredByExactlyThatCandidate(t *testing.T) {
	world := newBakeoff(t, "decide.local", "decide.local.kev", "decide.local.pplx")
	response := world.service.Decide(context.Background(), pinned("decide.local.kev"))

	require.Nil(t, response.GetError(), "%+v", response.GetError())
	require.Equal(t, "decide.local.kev", response.GetResourceRole())
	require.Equal(t, []string{"ANSWERED"}, statuses(response), "the walk is exactly the pinned candidate")
	require.Equal(t, []string{"decide.local.kev"}, world.dispatched())
	require.NotContains(t, world.resolved(), "decide.local", "an unpinned candidate is never even resolved")
	require.NotContains(t, world.resolved(), "decide.local.pplx")
}

func TestPinnedRequestNeverFallsBackWhenThePinCannotServe(t *testing.T) {
	world := newBakeoff(t, "decide.local", "decide.local.kev")
	world.runner.Errors[resolveKey("decide.local.kev")] = &providers.CommandError{Code: providers.CodeInsufficientVRAM, Command: resolveKey("decide.local.kev"), ExitCode: 1, Stderr: "kev:4b needs 6.0 GiB of VRAM"}
	response := world.service.Decide(context.Background(), pinned("decide.local.kev"))

	require.Equal(t, inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_UNAVAILABLE, response.GetError().GetCode())
	require.Equal(t, "pin_resource_role", response.GetError().GetConstruct())
	require.Contains(t, response.GetError().GetMessage(), "no other candidate was tried")
	require.Equal(t, []string{"SKIPPED:insufficient_vram"}, statuses(response), "the typed reason is the answer; nothing else is tried")
	require.Empty(t, world.dispatched(), "no candidate, local or enum, answers a pinned request that its pin cannot serve")
	require.NotContains(t, world.resolved(), "decide.local")
}

func TestPinMustBeADeclaredCandidateOfTheRole(t *testing.T) {
	world := newBakeoff(t, "decide.local", "decide.local.kev")
	for _, pin := range []string{"decide.local.pplx", "decide.local.KEV", "decide.jev.edenai"} {
		response := world.service.Decide(context.Background(), pinned(pin))
		require.Equal(t, inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_INVALID_REQUEST, response.GetError().GetCode(), pin)
		require.Equal(t, "pin_resource_role", response.GetError().GetConstruct())
		require.Contains(t, response.GetError().GetMessage(), "decide.local.kev", "the refusal names what the role does declare")
	}
	require.Empty(t, world.runner.Commands, "an undeclared pin reaches no resource")

	response := world.service.Decide(context.Background(), pinned("decide.local.kev; rm -rf /"))
	require.Equal(t, "pin_resource_role", response.GetError().GetConstruct())
	require.Empty(t, world.runner.Commands)
}

func TestPinnedRequestStillEnforcesLocalityPrivacyAndRights(t *testing.T) {
	t.Run("locality", func(t *testing.T) {
		world := newBakeoff(t, "decide.local.kev")
		request := pinned("decide.local.kev")
		request.Profile = sharedv1.Profile_PROFILE_REMOTE_ONLY
		response := world.service.Decide(context.Background(), request)
		require.Equal(t, []string{"SKIPPED:locality_forbidden"}, statuses(response))
		require.Empty(t, world.dispatched())
	})
	t.Run("unknown rights", func(t *testing.T) {
		world := newBakeoff(t, "decide.local.kev")
		world.unknownRights("decide.local.kev")
		response := world.service.Decide(context.Background(), pinned("decide.local.kev"))
		require.Equal(t, []string{"SKIPPED:unconfigured"}, statuses(response))
		require.Contains(t, response.GetCandidates()[0].GetDetail(), "rights")
		require.Empty(t, world.dispatched(), "a pin never lifts the rights gate")
	})
}

func TestPinnedGatewayCandidateRunsOnlyTheEnumPath(t *testing.T) {
	world := newBakeoff(t, "decide.local.kev")
	repository := world.service.repository.(*ResourceRepository)
	var classified int
	request := pinned("classify.fast")
	request.Questions = []*inferencev1.DecisionQuestion{choice("kind", "correction", "approval", "question")}
	response := repository.Decide(context.Background(), request, func(context.Context, ProviderRequest) *inferencev1.RunResponse {
		classified++
		return &inferencev1.RunResponse{Validated: true, ValueJson: `{"kind":"correction"}`, Provider: "ollama", Model: "q", Usage: &inferencev1.Usage{}}
	})
	require.Nil(t, response.GetError())
	require.Equal(t, inferencev1.DecisionMethod_DECISION_METHOD_LLM_ENUM, response.GetMethod())
	require.Equal(t, 1, classified)
	require.Empty(t, world.resolved(), "pinning the enum baseline resolves no Ollaya candidate")
}

func TestPinAppliesToEveryItemOfABatch(t *testing.T) {
	world := newBakeoff(t, "decide.local", "decide.local.kev")
	template := pinned("decide.local.kev")
	template.State = ""
	response := world.service.DecideBatch(context.Background(), template, []DecisionRequest{{State: "one"}, {State: "two"}})
	require.Len(t, response.GetResults(), 2)
	for _, result := range response.GetResults() {
		require.Nil(t, result.GetError(), "%+v", result.GetError())
		require.Equal(t, "decide.local.kev", result.GetResourceRole())
	}
	require.Equal(t, []string{"decide.local.kev", "decide.local.kev"}, world.dispatched())

	bad := world.service.DecideBatch(context.Background(), pinned("nope; drop"), []DecisionRequest{{State: "x"}})
	require.Len(t, bad.GetResults(), 1)
	require.Equal(t, "pin_resource_role", bad.GetResults()[0].GetError().GetConstruct())
}

func TestUnpinnedRequestKeepsTheOrderedFallbackWalk(t *testing.T) {
	world := newBakeoff(t, "decide.local", "decide.local.kev")
	world.runner.Errors[resolveKey("decide.local")] = &providers.CommandError{Code: providers.CodeUnconfigured, Command: resolveKey("decide.local"), ExitCode: 1}
	request := pinned("")
	response := world.service.Decide(context.Background(), request)
	require.Nil(t, response.GetError(), "%+v", response.GetError())
	require.Equal(t, []string{"SKIPPED:unconfigured", "ANSWERED", "NOT_ATTEMPTED"}, statuses(response))
	require.Equal(t, "decide.local.kev", response.GetResourceRole())
}

// A model's declared bounds are data: a request outside them is skipped before
// anything is dispatched, and the ladder continues.
func TestRequestOutsideADeclaredLimitIsSkippedBeforeDispatch(t *testing.T) {
	world := newBakeoff(t, "decide.local", "decide.local.kev")
	world.qualifies("decide.local", &providers.DecisionLimits{ContextTokens: 512, MaxOptions: 2})
	request := pinned("")
	request.Questions = []*inferencev1.DecisionQuestion{choice("kind", "correction", "approval", "question"), noul("stop")}
	response := world.service.Decide(context.Background(), request)

	require.Nil(t, response.GetError(), "%+v", response.GetError())
	require.Equal(t, []string{"SKIPPED:limits_exceeded", "ANSWERED", "NOT_ATTEMPTED"}, statuses(response))
	require.Contains(t, response.GetCandidates()[0].GetDetail(), `"kind" has 3 options; the model accepts 2`)
	require.Equal(t, []string{"decide.local.kev"}, world.dispatched())
}

func TestLimitsViolationCoversEveryDeclaredBound(t *testing.T) {
	limits := &providers.DecisionLimits{ContextTokens: 512, MaxOptions: 3, MaxQuestions: 2, ScoreLevels: 3}
	cases := map[string]struct {
		limits    *providers.DecisionLimits
		questions []*inferencev1.DecisionQuestion
		want      string
	}{
		"no declared limits":          {nil, []*inferencev1.DecisionQuestion{choice("a", "1", "2", "3", "4")}, ""},
		"within every bound":          {limits, []*inferencev1.DecisionQuestion{choice("a", "1", "2", "3"), score("s", "0", "1", "2")}, ""},
		"too many options":            {limits, []*inferencev1.DecisionQuestion{choice("a", "1", "2", "3", "4")}, `question "a" has 4 options`},
		"too many questions":          {limits, []*inferencev1.DecisionQuestion{noul("a"), noul("b"), noul("c")}, "3 questions; the model accepts 2"},
		"score levels differ":         {limits, []*inferencev1.DecisionQuestion{score("s", "0", "1")}, `question "s" has 2 score levels; the model serves exactly 3`},
		"unset bounds are not bounds": {&providers.DecisionLimits{ContextTokens: 512}, []*inferencev1.DecisionQuestion{choice("a", "1", "2", "3", "4", "5")}, ""},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := limitsViolation(tc.limits, tc.questions)
			if tc.want == "" {
				require.Empty(t, got)
				return
			}
			require.Contains(t, got, tc.want)
		})
	}
}

// Memory refusals from the resource are stable walk codes, not provider_failed.
func TestMemoryAndLimitRefusalsAreStableWalkReasons(t *testing.T) {
	for _, code := range []string{providers.CodeInsufficientVRAM, providers.CodeInsufficientMemory, providers.CodeLimitsExceeded} {
		t.Run(code, func(t *testing.T) {
			world := newBakeoff(t, "decide.local", "decide.local.kev")
			world.runner.ErrorMatch = func(command providers.Command) error {
				if strings.Contains(command.String(), "gateway decide --role decide.local ") {
					return &providers.CommandError{Code: code, Command: command.String(), ExitCode: 1}
				}
				return nil
			}
			response := world.service.Decide(context.Background(), pinned(""))
			require.Nil(t, response.GetError(), "%+v", response.GetError())
			require.Equal(t, []string{"FAILED:" + code, "ANSWERED", "NOT_ATTEMPTED"}, statuses(response))
		})
	}
}

func TestPinResourceRoleShapeIsValidatedBeforeAnyResourceIsReached(t *testing.T) {
	require.NoError(t, validatePinResourceRole(""))
	require.NoError(t, validatePinResourceRole("decide.local.credence"))
	require.NoError(t, validatePinResourceRole("  decide.local  "))
	for _, bad := range []string{strings.Repeat("a", 101), "a b", "a;b", "../etc", "ünï"} {
		require.Error(t, validatePinResourceRole(bad), bad)
	}
}
