package inference

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/cli-core/cliapp"
	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	inferenceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference/inference_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
)

type decisionBindingCapture struct {
	inferenceconnect.UnimplementedInferenceServiceHandler
	decide      *inferencev1.DecideRequest
	decideBatch *inferencev1.DecideBatchRequest
}

func (c *decisionBindingCapture) Decide(_ context.Context, request *connect.Request[inferencev1.DecideRequest]) (*connect.Response[inferencev1.DecideResponse], error) {
	c.decide = request.Msg
	return connect.NewResponse(&inferencev1.DecideResponse{}), nil
}

func (c *decisionBindingCapture) DecideBatch(_ context.Context, request *connect.Request[inferencev1.DecideBatchRequest]) (*connect.Response[inferencev1.DecideBatchResponse], error) {
	c.decideBatch = request.Msg
	return connect.NewResponse(&inferencev1.DecideBatchResponse{}), nil
}

func TestDecideCLIProtoBindingCarriesPrivacyAndDecodesQuestionsFile(t *testing.T) {
	questions := filepath.Join(t.TempDir(), "questions.json")
	require.NoError(t, os.WriteFile(questions, []byte(`[{"id":"choice","type":"DECISION_QUESTION_TYPE_CHOICE","instructions":"Pick one","options":[{"label":"keep"},{"label":"drop"}]}]`), 0o600))
	service := &decisionBindingCapture{}
	core, group := inferenceBindingGroup(t, service)
	command := inferenceCommand(t, group, "decide")
	ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
		Core: core, Schema: command.Args, JSON: true, Stdout: &bytes.Buffer{},
		Flags: map[string]string{
			"role": "decide.fast", "state": "fixture state", "questions": questions,
			"profile": "remote-only", "privacy-class": "public", "privacy-provenance": "cli_test.fixture_source",
		},
	})
	require.NoError(t, command.RunCtx(ctx))
	require.NotNil(t, service.decide)
	require.Equal(t, sharedv1.Profile_PROFILE_REMOTE_ONLY, service.decide.GetProfile())
	require.Equal(t, sharedv1.PrivacyClass_PRIVACY_CLASS_PUBLIC, service.decide.GetPrivacyClass())
	require.Equal(t, "cli_test.fixture_source", service.decide.GetPrivacyProvenance())
	require.Equal(t, "fixture state", service.decide.GetState())
	require.Len(t, service.decide.GetQuestions(), 1)
	require.Equal(t, "choice", service.decide.GetQuestions()[0].GetId())
	require.Equal(t, inferencev1.DecisionQuestionType_DECISION_QUESTION_TYPE_CHOICE, service.decide.GetQuestions()[0].GetType())
	require.Equal(t, []string{"keep", "drop"}, []string{service.decide.GetQuestions()[0].GetOptions()[0].GetLabel(), service.decide.GetQuestions()[0].GetOptions()[1].GetLabel()})
}

func TestDecideBatchCLIProtoBindingCarriesPrivacyAndDecodesQuestionsAndItemsFiles(t *testing.T) {
	temp := t.TempDir()
	questions := filepath.Join(temp, "questions.json")
	items := filepath.Join(temp, "items.json")
	require.NoError(t, os.WriteFile(questions, []byte(`[{"id":"score","type":"DECISION_QUESTION_TYPE_SCORE","levels":["low","high"]}]`), 0o600))
	require.NoError(t, os.WriteFile(items, []byte(`[{"state":"first"},{"stateJson":"{\"count\":2}"}]`), 0o600))
	service := &decisionBindingCapture{}
	core, group := inferenceBindingGroup(t, service)
	command := inferenceCommand(t, group, "decide-batch")
	ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
		Core: core, Schema: command.Args, JSON: true, Stdout: &bytes.Buffer{},
		Flags: map[string]string{
			"role": "decide.fast", "questions": questions, "items": items,
			"profile": "local-only", "privacy-class": "secret", "privacy-provenance": "cli_test.sensitive_fixture",
		},
	})
	require.NoError(t, command.RunCtx(ctx))
	require.NotNil(t, service.decideBatch)
	require.Equal(t, sharedv1.Profile_PROFILE_LOCAL_ONLY, service.decideBatch.GetProfile())
	require.Equal(t, sharedv1.PrivacyClass_PRIVACY_CLASS_SECRET, service.decideBatch.GetPrivacyClass())
	require.Equal(t, "cli_test.sensitive_fixture", service.decideBatch.GetPrivacyProvenance())
	require.Len(t, service.decideBatch.GetQuestions(), 1)
	require.Equal(t, "score", service.decideBatch.GetQuestions()[0].GetId())
	require.Equal(t, []string{"low", "high"}, service.decideBatch.GetQuestions()[0].GetLevels())
	require.Len(t, service.decideBatch.GetItems(), 2)
	require.Equal(t, "first", service.decideBatch.GetItems()[0].GetState())
	require.Equal(t, `{"count":2}`, service.decideBatch.GetItems()[1].GetStateJson())
}

func inferenceBindingGroup(t *testing.T, service *decisionBindingCapture) (*cliapp.ScenarioApp, cliapp.SubcommandGroup) {
	t.Helper()
	servicePath, serviceHandler := inferenceconnect.NewInferenceServiceHandler(service)
	mux := http.NewServeMux()
	mux.Handle(servicePath, serviceHandler)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	t.Setenv("AI_GATEWAY_BINDING_TEST_CONFIG_DIR", t.TempDir())
	core, err := cliapp.NewScenarioApp(cliapp.ScenarioOptions{
		Name: "ai-gateway", APIPrefix: "/", DefaultAPIBase: server.URL,
		ConfigDirEnvVars: []string{"AI_GATEWAY_BINDING_TEST_CONFIG_DIR"},
	})
	require.NoError(t, err)
	manifest, err := os.ReadFile(filepath.Join("..", "..", "manifest.json"))
	require.NoError(t, err)
	group, err := Register(core, manifest)
	require.NoError(t, err)
	return core, group
}

func inferenceCommand(t *testing.T, group cliapp.SubcommandGroup, name string) cliapp.Command {
	t.Helper()
	for _, command := range group.Subcommands {
		if command.Name == name {
			return command
		}
	}
	t.Fatalf("inference command %q not found", name)
	return cliapp.Command{}
}
