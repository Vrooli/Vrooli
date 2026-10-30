package inference

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"ai-gateway/internal/providers"
	providermocks "ai-gateway/internal/providers/mocks"

	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
)

const decideRole = "decide.test"

// decisionWorld stands in for the decision-model resource and the typed
// classify path. Each resource role is scripted to resolve (at a locality),
// refuse resolution with a typed code, answer, or fail on dispatch.
type decisionWorld struct {
	runner   *providermocks.FakeRunner
	classify []ProviderRequest
	enum     *inferencev1.RunResponse
}

func newDecisionWorld() *decisionWorld {
	return &decisionWorld{runner: &providermocks.FakeRunner{Results: map[string]providers.Result{}, Errors: map[string]error{}}}
}

func resolveKey(role string) string {
	return "resource-ollaya policy resolve --role " + role + " --json"
}
func decideKey(role string) string {
	return "resource-ollaya gateway decide --role " + role + " --input-stdin --json --timeout 30s"
}

func (w *decisionWorld) resolves(role, locality string) *decisionWorld {
	w.runner.Results[resolveKey(role)] = providers.Result{Stdout: fmt.Sprintf(`{"role":%q,"model":"m-%s","locality":%q,"capabilities":["decide"]}`, role, role, locality)}
	return w
}

func (w *decisionWorld) refusesResolve(role, code string) *decisionWorld {
	w.runner.Errors[resolveKey(role)] = &providers.CommandError{Code: code, Command: resolveKey(role), ExitCode: 1, Stderr: "role cannot serve"}
	return w
}

func (w *decisionWorld) answers(role, output string) *decisionWorld {
	w.runner.Results[decideKey(role)] = providers.Result{Stdout: output}
	return w
}

func (w *decisionWorld) failsDispatch(role, code string) *decisionWorld {
	w.runner.Errors[decideKey(role)] = &providers.CommandError{Code: code, Command: decideKey(role), ExitCode: 1}
	return w
}

func (w *decisionWorld) runTyped(_ context.Context, request ProviderRequest) *inferencev1.RunResponse {
	w.classify = append(w.classify, request)
	if w.enum == nil {
		return &inferencev1.RunResponse{Usage: &inferencev1.Usage{}, Error: &inferencev1.InferenceError{Code: inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_UNAVAILABLE, Message: "no local model"}}
	}
	return w.enum
}

func (w *decisionWorld) repository(t *testing.T) *ResourceRepository {
	t.Helper()
	catalog := RoleCatalog{SchemaVersion: 1, Roles: map[string]InferenceRole{
		decideRole: {Kind: RoleKindDecision, TimeoutMS: 30000, Candidates: []Candidate{
			{Provider: "ollaya", ResourceRole: "decide.local", Reason: "local"},
			{Provider: "ollaya", ResourceRole: "decide.hosted", Reason: "hosted"},
			{Provider: "ai-gateway", ResourceRole: "classify.fast", Reason: "enum"},
		}},
		"classify.fast": {Candidates: []Candidate{{Provider: "ollama", ResourceRole: "classify.routing", Reason: "local"}}},
	}}
	repository, err := NewResourceRepository(catalog, []providers.Adapter{{Provider: providers.ProviderOllaya, CommandName: "resource-ollaya", Locality: "local", Runner: w.runner}})
	require.NoError(t, err)
	return repository
}

func (w *decisionWorld) decide(t *testing.T, request DecisionRequest) *inferencev1.DecideResponse {
	t.Helper()
	if request.Role == "" {
		request.Role = decideRole
	}
	return w.repository(t).Decide(context.Background(), request, w.runTyped)
}

func choice(id string, labels ...string) *inferencev1.DecisionQuestion {
	question := &inferencev1.DecisionQuestion{Id: id, Type: inferencev1.DecisionQuestionType_DECISION_QUESTION_TYPE_CHOICE, Instructions: "Which " + id + "?"}
	for _, label := range labels {
		question.Options = append(question.Options, &inferencev1.DecisionOption{Label: label, Description: "means " + label})
	}
	return question
}

func noul(id string) *inferencev1.DecisionQuestion {
	return &inferencev1.DecisionQuestion{Id: id, Type: inferencev1.DecisionQuestionType_DECISION_QUESTION_TYPE_NOUL, Instructions: id + " holds."}
}

func score(id string, levels ...string) *inferencev1.DecisionQuestion {
	return &inferencev1.DecisionQuestion{Id: id, Type: inferencev1.DecisionQuestionType_DECISION_QUESTION_TYPE_SCORE, Levels: levels}
}

// operatorMessage is the motivating case: one operator reply, a three-way
// choice and a yes/no.
func operatorMessage() DecisionRequest {
	return DecisionRequest{
		State:     "No, stop. That is not what I asked for; revert the file.",
		Questions: []*inferencev1.DecisionQuestion{choice("kind", "correction", "approval", "question"), noul("stop")},
	}
}

const answeredOutput = `{"upstream":"local","locality":"local","model":"laya:en","latency_ms":6,"usage":{"input_tokens":103,"output_tokens":0},"cost_usd":null,
"answers":{"stop":{"type":"noul","noul":0.82},"kind":{"type":"choice","choice":"correction","confidence":0.61,"probabilities":{"correction":0.74,"approval":0.11,"question":0.15}}}}`

func statuses(response *inferencev1.DecideResponse) []string {
	out := make([]string, 0, len(response.GetCandidates()))
	for _, row := range response.GetCandidates() {
		status := strings.TrimPrefix(row.GetStatus().String(), "DECISION_CANDIDATE_STATUS_")
		if row.GetReason() != "" {
			status += ":" + row.GetReason()
		}
		out = append(out, status)
	}
	return out
}

func TestDecideAnswersFromTheFirstServingDecisionModel(t *testing.T) {
	world := newDecisionWorld().resolves("decide.local", "local").answers("decide.local", answeredOutput)
	response := world.decide(t, operatorMessage())

	require.Nil(t, response.GetError())
	require.Equal(t, inferencev1.DecisionMethod_DECISION_METHOD_DECISION_MODEL, response.GetMethod())
	require.Equal(t, "ollaya", response.GetProvider())
	require.Equal(t, "laya:en", response.GetModel(), "the answering model, not the router alias")
	require.Equal(t, "decide.local", response.GetResourceRole())
	require.Equal(t, "local", response.GetLocality())
	require.Equal(t, []string{"ANSWERED", "NOT_ATTEMPTED", "NOT_ATTEMPTED"}, statuses(response))
	require.Empty(t, world.classify)

	require.Len(t, response.GetAnswers(), 2)
	kind, stop := response.GetAnswers()[0], response.GetAnswers()[1]
	require.Equal(t, "kind", kind.GetQuestionId(), "answers follow question order, not provider map order")
	require.Equal(t, "correction", kind.GetChoice())
	require.InDelta(t, 0.61, kind.GetConfidence(), 1e-9)
	require.Equal(t, []string{"correction", "approval", "question"}, []string{kind.GetProbabilities()[0].GetOption(), kind.GetProbabilities()[1].GetOption(), kind.GetProbabilities()[2].GetOption()})
	require.InDelta(t, 0.82, stop.GetNoul(), 1e-9)
	require.Nil(t, stop.Confidence, "a noul answer carries no confidence")

	usage := response.GetUsage()
	require.EqualValues(t, 103, usage.GetInputTokens())
	require.Equal(t, inferencev1.UsageProvenance_USAGE_PROVENANCE_PROVIDER_REPORTED, usage.GetInputTokensProvenance())
	require.Equal(t, inferencev1.UsageProvenance_USAGE_PROVENANCE_PROVIDER_REPORTED, usage.GetOutputTokensProvenance(), "a reported zero is an observed zero")
	require.Equal(t, inferencev1.UsageProvenance_USAGE_PROVENANCE_UNKNOWN, usage.GetCostProvenance())

	var body map[string]json.RawMessage
	require.NoError(t, json.Unmarshal([]byte(world.runner.Commands[1].Stdin), &body))
	require.JSONEq(t, `"No, stop. That is not what I asked for; revert the file."`, string(body["state"]))
}

func TestDecideSkipsUnconfiguredCandidatesAndFallsBackToTheEnumPath(t *testing.T) {
	world := newDecisionWorld().refusesResolve("decide.local", providers.CodeUnreachable).refusesResolve("decide.hosted", providers.CodeUnconfigured)
	world.enum = &inferencev1.RunResponse{
		Validated: true, ValueJson: `{"kind":"correction"}`, Provider: "ollama", Model: "qwen3.5:4b",
		Usage: &inferencev1.Usage{InputTokens: 300, OutputTokens: 9},
	}
	request := DecisionRequest{State: "Stop, revert it.", Questions: []*inferencev1.DecisionQuestion{choice("kind", "correction", "approval", "question")}, Profile: sharedv1.Profile_PROFILE_LOCAL_ONLY}
	response := world.decide(t, request)

	require.Nil(t, response.GetError())
	require.Equal(t, []string{"SKIPPED:unreachable", "SKIPPED:unconfigured", "ANSWERED"}, statuses(response))
	require.Equal(t, inferencev1.DecisionMethod_DECISION_METHOD_LLM_ENUM, response.GetMethod())
	require.Equal(t, "ollama", response.GetProvider())
	require.Equal(t, "correction", response.GetAnswers()[0].GetChoice())
	require.Nil(t, response.GetAnswers()[0].Confidence, "an LLM label is not a calibrated distribution")
	require.Empty(t, response.GetAnswers()[0].GetProbabilities())
	require.EqualValues(t, 300, response.GetUsage().GetInputTokens())

	require.Len(t, world.classify, 1)
	fallback := world.classify[0]
	require.Equal(t, "classify.fast", fallback.Role)
	require.Equal(t, sharedv1.Profile_PROFILE_LOCAL_ONLY, fallback.Profile, "the fallback inherits the caller's locality stance")
	require.JSONEq(t, `{"type":"object","properties":{"kind":{"type":"string","enum":["correction","approval","question"]}},"required":["kind"],"additionalProperties":false}`, fallback.SchemaJSON)
	require.Contains(t, fallback.Instruction, "correction: means correction")
}

func TestDecideRefusesHostedCandidatesUnderLocalOnly(t *testing.T) {
	world := newDecisionWorld().refusesResolve("decide.local", providers.CodeUnconfigured).resolves("decide.hosted", "remote").answers("decide.hosted", answeredOutput)
	request := operatorMessage()
	request.Profile = sharedv1.Profile_PROFILE_LOCAL_ONLY
	response := world.decide(t, request)

	require.Equal(t, []string{"SKIPPED:unconfigured", "SKIPPED:locality_forbidden", "SKIPPED:unsupported_question_types"}, statuses(response))
	require.Equal(t, inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_UNAVAILABLE, response.GetError().GetCode())
	for _, command := range world.runner.Commands {
		require.NotContains(t, command.String(), "gateway decide", "a forbidden candidate is never dispatched")
	}
	require.Empty(t, world.classify, "a noul question cannot be answered by the enum fallback")
}

func TestDecideRejectsAnswerSetsThatBreakTheClosedSet(t *testing.T) {
	cases := map[string]string{
		"missing answer":       `{"answers":{"kind":{"type":"choice","choice":"correction","confidence":0.6,"probabilities":{"correction":0.8,"approval":0.1,"question":0.1}}}}`,
		"label outside set":    `{"answers":{"stop":{"type":"noul","noul":0.8},"kind":{"type":"choice","choice":"refund","confidence":0.6,"probabilities":{"correction":0.8,"approval":0.1,"question":0.1}}}}`,
		"wrong answer type":    `{"answers":{"stop":{"type":"choice","choice":"yes"},"kind":{"type":"choice","choice":"correction","confidence":0.6,"probabilities":{"correction":0.8,"approval":0.1,"question":0.1}}}}`,
		"partial distribution": `{"answers":{"stop":{"type":"noul","noul":0.8},"kind":{"type":"choice","choice":"correction","confidence":0.6,"probabilities":{"correction":0.8,"approval":0.2}}}}`,
		"not a distribution":   `{"answers":{"stop":{"type":"noul","noul":0.8},"kind":{"type":"choice","choice":"correction","confidence":0.6,"probabilities":{"correction":0.8,"approval":0.8,"question":0.1}}}}`,
		"noul above one":       `{"answers":{"stop":{"type":"noul","noul":1.3},"kind":{"type":"choice","choice":"correction","confidence":0.6,"probabilities":{"correction":0.8,"approval":0.1,"question":0.1}}}}`,
		"not json":             `laya says hi`,
	}
	for name, output := range cases {
		t.Run(name, func(t *testing.T) {
			world := newDecisionWorld().resolves("decide.local", "local").answers("decide.local", output).
				resolves("decide.hosted", "remote").answers("decide.hosted", answeredOutput)
			response := world.decide(t, operatorMessage())
			require.Equal(t, []string{"FAILED:invalid_answers", "ANSWERED", "NOT_ATTEMPTED"}, statuses(response))
			require.Equal(t, "decide.hosted", response.GetResourceRole())
			require.Equal(t, "remote", response.GetLocality())
		})
	}
}

func TestDecideScoreAnswersAreBoundedAndConfidenceIsDerivedWhenAbsent(t *testing.T) {
	question := score("frustration", "calm", "annoyed", "angry")
	ok := `{"answers":{"frustration":{"type":"score","score":1.6,"probabilities":{"0":0.1,"1":0.2,"2":0.7}}}}`
	world := newDecisionWorld().resolves("decide.local", "local").answers("decide.local", ok)
	response := world.decide(t, DecisionRequest{State: "This is the third time!", Questions: []*inferencev1.DecisionQuestion{question}})
	require.Nil(t, response.GetError())
	answer := response.GetAnswers()[0]
	require.InDelta(t, 1.6, answer.GetScore(), 1e-9)
	require.InDelta(t, (3*0.7-1)/2, answer.GetConfidence(), 1e-9, "System One's normalized top probability")

	outOfRange := `{"answers":{"frustration":{"type":"score","score":2.4,"confidence":0.5,"probabilities":{"0":0.1,"1":0.2,"2":0.7}}}}`
	world = newDecisionWorld().resolves("decide.local", "local").answers("decide.local", outOfRange).refusesResolve("decide.hosted", providers.CodeUnconfigured)
	response = world.decide(t, DecisionRequest{State: "x", Questions: []*inferencev1.DecisionQuestion{question}})
	require.Equal(t, []string{"FAILED:invalid_answers", "SKIPPED:unconfigured", "SKIPPED:unsupported_question_types"}, statuses(response))
}

func TestDecideReportsContextOverflowOnlyWhenEveryDispatchOverflowed(t *testing.T) {
	world := newDecisionWorld().resolves("decide.local", "local").failsDispatch("decide.local", providers.CodeContextOverflow).
		resolves("decide.hosted", "remote").failsDispatch("decide.hosted", providers.CodeContextOverflow)
	response := world.decide(t, operatorMessage())
	require.Equal(t, inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_CONTEXT_OVERFLOW, response.GetError().GetCode())

	world = newDecisionWorld().resolves("decide.local", "local").failsDispatch("decide.local", providers.CodeContextOverflow).
		resolves("decide.hosted", "remote").failsDispatch("decide.hosted", providers.CodeRateLimited)
	response = world.decide(t, operatorMessage())
	require.Equal(t, []string{"FAILED:context_overflow", "FAILED:rate_limited", "SKIPPED:unsupported_question_types"}, statuses(response))
	require.Equal(t, inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_UNAVAILABLE, response.GetError().GetCode())
}

func TestDecisionAndTypedRolesDoNotCrossServe(t *testing.T) {
	world := newDecisionWorld()
	response := world.decide(t, DecisionRequest{Role: "classify.fast", State: "x", Questions: []*inferencev1.DecisionQuestion{noul("q")}})
	require.Equal(t, inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_INVALID_REQUEST, response.GetError().GetCode())

	_, err := world.repository(t).Run(context.Background(), ProviderRequest{Role: decideRole, Source: "x", SchemaJSON: `{"type":"string"}`})
	require.ErrorIs(t, err, ErrUnavailable)
	require.Contains(t, err.Error(), "use Decide")
}

func TestSystemOneBodyPreservesQuestionAndOptionOrder(t *testing.T) {
	body, err := systemOneBody(DecisionRequest{
		StateJSON: ` {"assistant":"Done.","operator":"No, revert."} `,
		Questions: []*inferencev1.DecisionQuestion{
			choice("zeta", "b", "a"), score("alpha", "low", "high"),
			{Id: "stop", Type: inferencev1.DecisionQuestionType_DECISION_QUESTION_TYPE_NOUL, TrueDescription: "wants a stop"},
		},
	})
	require.NoError(t, err)
	require.Equal(t, `{"state":{"assistant":"Done.","operator":"No, revert."},"questions":{"zeta":{"type":"choice","instructions":"Which zeta?","criteria":{"b":"means b","a":"means a"}},"alpha":{"type":"score","criteria":["low","high"]},"stop":{"type":"noul","criteria":{"true":"wants a stop"}}}}`, body)
}

func TestDecisionValidationRejectsMalformedRequests(t *testing.T) {
	many := make([]*inferencev1.DecisionQuestion, MaxDecisionQuestions+1)
	for i := range many {
		many[i] = noul(fmt.Sprintf("q%d", i))
	}
	levels := make([]string, MaxScoreLevels+1)
	for i := range levels {
		levels[i] = fmt.Sprintf("level %d", i)
	}
	questionCases := map[string][]*inferencev1.DecisionQuestion{
		"no questions":       nil,
		"too many questions": many,
		"repeated id":        {noul("q"), noul("q")},
		"id with spaces":     {noul("is it")},
		"one option":         {choice("kind", "only")},
		"repeated label":     {choice("kind", "a", "a")},
		"padded label":       {choice("kind", " a", "b")},
		"too many levels":    {score("s", levels...)},
		"choice with levels": {{Id: "k", Type: inferencev1.DecisionQuestionType_DECISION_QUESTION_TYPE_CHOICE, Options: choice("k", "a", "b").Options, Levels: []string{"x", "y"}}},
		"noul with options":  {{Id: "k", Type: inferencev1.DecisionQuestionType_DECISION_QUESTION_TYPE_NOUL, Options: choice("k", "a", "b").Options}},
		"unspecified type":   {{Id: "k"}},
	}
	for name, questions := range questionCases {
		var decisionErr *DecisionError
		require.ErrorAs(t, ValidateQuestions(questions), &decisionErr, name)
	}
	stateCases := map[string][2]string{
		"no state":      {"", ""},
		"both states":   {"text", `{"a":1}`},
		"scalar json":   {"", `"just a string"`},
		"invalid json":  {"", `{"a":`},
		"oversize text": {strings.Repeat("x", MaxDecisionStateBytes+1), ""},
	}
	for name, state := range stateCases {
		require.Error(t, ValidateState(state[0], state[1]), name)
	}
	require.NoError(t, ValidateState("", `[{"turn":1}]`))
}

func TestShippedCatalogDeclaresDecisionRolesWithGatedHostedCandidates(t *testing.T) {
	catalog, err := LoadCatalog(filepath.Join("..", "..", "..", "config", "inference-role-catalog.json"))
	require.NoError(t, err)
	for _, name := range []string{"decide.fast", "decide.accurate"} {
		role := catalog.Roles[name]
		require.True(t, role.IsDecision(), name)
		last := role.Candidates[len(role.Candidates)-1]
		require.Equal(t, ProviderGateway, last.Provider, "%s must end on the enum fallback", name)
		require.Equal(t, "classify.fast", last.ResourceRole)
	}
	require.Equal(t, "decide.local", catalog.Roles["decide.fast"].Candidates[0].ResourceRole, "decide.fast is local-first")
	require.False(t, catalog.Roles["classify.fast"].IsDecision())
}

func TestCatalogRejectsMixedRoleKinds(t *testing.T) {
	typed := InferenceRole{Candidates: []Candidate{{Provider: "ollama", ResourceRole: "r", Reason: "x"}}}
	cases := map[string]map[string]InferenceRole{
		"decision role on a generative provider": {"d": {Kind: RoleKindDecision, Candidates: []Candidate{{Provider: "ollama", ResourceRole: "r", Reason: "x"}}}},
		"typed role on a decision model":         {"t": {Candidates: []Candidate{{Provider: "ollaya", ResourceRole: "r", Reason: "x"}}}},
		"fallback to an undeclared role":         {"d": {Kind: RoleKindDecision, Candidates: []Candidate{{Provider: "ai-gateway", ResourceRole: "missing", Reason: "x"}}}},
		"fallback to another decision role": {
			"d": {Kind: RoleKindDecision, Candidates: []Candidate{{Provider: "ai-gateway", ResourceRole: "e", Reason: "x"}}},
			"e": {Kind: RoleKindDecision, Candidates: []Candidate{{Provider: "ollaya", ResourceRole: "r", Reason: "x"}}},
		},
		"sampling on a decision role": {"d": {Kind: RoleKindDecision, Sampling: deterministicRoleSampling(), Candidates: []Candidate{{Provider: "ollaya", ResourceRole: "r", Reason: "x"}}}},
		"unknown kind":                {"d": {Kind: "oracle", Candidates: typed.Candidates}},
	}
	for name, roles := range cases {
		require.Error(t, RoleCatalog{SchemaVersion: 1, Roles: roles}.Validate(), name)
	}
}
