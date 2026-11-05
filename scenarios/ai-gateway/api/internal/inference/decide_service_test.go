package inference

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"ai-gateway/internal/providers"

	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
)

// serviceWorld wires the real Service over a resource repository whose
// classify.fast candidate is a scripted Ollama, so the enum fallback runs the
// production typed path: schema gate, provider walk and local validation.
func serviceWorld(t *testing.T, world *decisionWorld, ollamaValue string) *Service {
	t.Helper()
	world.runner.DefaultOK = true
	world.runner.Default = providers.Result{Stdout: `{"response":` + ollamaValue + `,"prompt_eval_count":250,"eval_count":7}`}
	world.runner.Results["resource-ollama policy resolve --role classify.routing --json"] = providers.Result{Stdout: resolveStdout("classify.routing", "qwen3.5:4b", providers.SamplingHonored)}
	catalog := RoleCatalog{SchemaVersion: 1, Roles: map[string]InferenceRole{
		decideRole: {Kind: RoleKindDecision, TimeoutMS: 30000, Candidates: []Candidate{
			{Provider: "ollaya", ResourceRole: "decide.local", Reason: "local"},
			{Provider: "ai-gateway", ResourceRole: "classify.fast", Reason: "enum"},
		}},
		"classify.fast": {Sampling: deterministicRoleSampling(), Candidates: []Candidate{{Provider: "ollama", ResourceRole: "classify.routing", Reason: "local"}}},
	}}
	repository, err := NewResourceRepository(catalog, []providers.Adapter{
		{Provider: providers.ProviderOllaya, CommandName: "resource-ollaya", Locality: "local", Runner: world.runner},
		{Provider: providers.ProviderOllama, CommandName: "resource-ollama", Locality: "local", Runner: world.runner},
	})
	require.NoError(t, err)
	return NewService(repository)
}

func TestDecideFallbackRunsTheProductionClassifyPath(t *testing.T) {
	world := newDecisionWorld().refusesResolve("decide.local", providers.CodeUnreachable)
	service := serviceWorld(t, world, `"{\"kind\":\"approval\"}"`)
	response := service.Decide(context.Background(), DecisionRequest{
		Role: decideRole, State: "Looks great, ship it.",
		Questions: []*inferencev1.DecisionQuestion{choice("kind", "correction", "approval", "question")},
	})
	require.Nil(t, response.GetError())
	require.Equal(t, inferencev1.DecisionMethod_DECISION_METHOD_LLM_ENUM, response.GetMethod())
	require.Equal(t, "approval", response.GetAnswers()[0].GetChoice())
	require.Equal(t, "qwen3.5:4b", response.GetModel())
	require.EqualValues(t, 250, response.GetUsage().GetInputTokens())

	var generate string
	for _, command := range world.runner.Commands {
		if strings.Contains(command.String(), "gateway generate") {
			generate = command.String()
		}
	}
	require.Contains(t, generate, `"enum":["correction","approval","question"]`, "the enum grammar reaches the local model")
}

func TestDecideFallbackRejectsALabelOutsideTheEnum(t *testing.T) {
	world := newDecisionWorld().refusesResolve("decide.local", providers.CodeUnreachable)
	service := serviceWorld(t, world, `"{\"kind\":\"refund\"}"`)
	response := service.Decide(context.Background(), DecisionRequest{
		Role: decideRole, State: "x", Questions: []*inferencev1.DecisionQuestion{choice("kind", "correction", "approval")},
	})
	require.Equal(t, inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_UNAVAILABLE, response.GetError().GetCode())
	require.Equal(t, []string{"SKIPPED:unreachable", "FAILED:provider_failed"}, statuses(response))
	require.Contains(t, response.GetCandidates()[1].GetDetail(), "VALIDATION_FAILED")
	require.EqualValues(t, 500, response.GetUsage().GetInputTokens(), "both repair attempts are billed")
}

func TestDecideBatchPreservesOrderAndIsolatesBadItems(t *testing.T) {
	world := newDecisionWorld().resolves("decide.local", "local").answers("decide.local", answeredOutput)
	service := serviceWorld(t, world, `"{}"`)
	template := operatorMessage()
	template.Role = decideRole
	response := service.DecideBatch(context.Background(), template, []DecisionRequest{
		{State: "Stop."}, {State: "", StateJSON: `"scalar"`}, {StateJSON: `{"operator":"revert it"}`},
	})
	require.Len(t, response.GetResults(), 3)
	require.Nil(t, response.GetResults()[0].GetError())
	require.Equal(t, "state_json", response.GetResults()[1].GetError().GetConstruct(), "a bad item fails alone")
	require.Nil(t, response.GetResults()[2].GetError())
	require.EqualValues(t, 206, response.GetUsage().GetInputTokens(), "usage aggregates answered items only")

	refused := service.DecideBatch(context.Background(), DecisionRequest{Role: decideRole}, []DecisionRequest{{State: "a"}, {State: "b"}})
	require.Len(t, refused.GetResults(), 1, "a bad question set fails the batch once")
	require.Equal(t, "questions", refused.GetResults()[0].GetError().GetConstruct())

	over := make([]DecisionRequest, MaxBatchSize+1)
	require.Equal(t, "items", service.DecideBatch(context.Background(), template, over).GetResults()[0].GetError().GetConstruct())
}

func TestDecideWithoutADecisionRepositoryIsUnavailable(t *testing.T) {
	response := NewService(nil).Decide(context.Background(), DecisionRequest{Role: "decide.fast", State: "x", Questions: []*inferencev1.DecisionQuestion{noul("q")}})
	require.Equal(t, inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_UNAVAILABLE, response.GetError().GetCode())
	response = NewService(UnavailableRepository{Reason: "catalog is malformed"}).Decide(context.Background(), DecisionRequest{Role: "decide.fast", State: "x", Questions: []*inferencev1.DecisionQuestion{noul("q")}})
	require.Equal(t, "catalog is malformed", response.GetError().GetMessage())
}
