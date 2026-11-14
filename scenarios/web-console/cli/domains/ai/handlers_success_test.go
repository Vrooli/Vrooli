package ai

import (
	"context"
	"os"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	aiv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/ai"
	aiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/ai/ai_v1connect"
)

type testClient struct{ aiconnect.AIServiceClient }

func (testClient) Generate(context.Context, *connect.Request[aiv1.GenerateRequest]) (*connect.Response[aiv1.GenerateResponse], error) {
	return connect.NewResponse(&aiv1.GenerateResponse{Command: "ls", Provider: "test"}), nil
}

func (testClient) Suggest(context.Context, *connect.Request[aiv1.SuggestRequest]) (*connect.Response[aiv1.SuggestResponse], error) {
	return connect.NewResponse(&aiv1.SuggestResponse{Commands: []string{"ls", "pwd"}, Provider: "test"}), nil
}

func (testClient) GetConfig(context.Context, *connect.Request[aiv1.GetConfigRequest]) (*connect.Response[aiv1.GetConfigResponse], error) {
	return connect.NewResponse(&aiv1.GetConfigResponse{Providers: []*aiv1.ProviderConfig{{Name: "test", Enabled: true}}, Health: []*aiv1.ProviderHealth{{Name: "test", Available: true}}}), nil
}

func (testClient) UpdateConfig(context.Context, *connect.Request[aiv1.UpdateConfigRequest]) (*connect.Response[aiv1.UpdateConfigResponse], error) {
	return connect.NewResponse(&aiv1.UpdateConfigResponse{}), nil
}

func (testClient) GetHealth(context.Context, *connect.Request[aiv1.GetHealthRequest]) (*connect.Response[aiv1.GetHealthResponse], error) {
	return connect.NewResponse(&aiv1.GetHealthResponse{Health: []*aiv1.ProviderHealth{{Name: "test", Available: true}}}), nil
}

func TestHandlersRenderSuccessfulResponses(t *testing.T) {
	body := t.TempDir() + "/body.json"
	if err := os.WriteFile(body, []byte(`{"prompt":"list files","context":"repo","name":"test","enabled":true,"priority":2,"timeout_sec":10,"max_retries":3}`), 0o600); err != nil {
		t.Fatal(err)
	}
	schema := cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "body-file"}}}
	h := &handlers{client: testClient{}}
	for _, call := range []func(cliapp.RunContext) error{
		h.generate, h.suggest, h.configGet, h.configSet, h.health,
	} {
		ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{Schema: schema, Flags: map[string]string{"body-file": body}, JSON: true})
		if err := call(ctx); err != nil {
			t.Fatal(err)
		}
	}
}
