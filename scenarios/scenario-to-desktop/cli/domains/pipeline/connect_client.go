package pipeline

import (
	"context"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	pipelinev1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/pipeline"
	"github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/pipeline/pipelineconnect"
)

// pipelineRPC is the generated Connect contract consumed by the CLI. Keeping
// this narrow seam makes primitive-handler tests independent of HTTP transport.
type pipelineRPC interface {
	Get(context.Context, *connect.Request[pipelinev1.PipelineGetRequest]) (*connect.Response[pipelinev1.PipelineStatus], error)
	GetReleaseGate(context.Context, *connect.Request[pipelinev1.PipelineGetRequest]) (*connect.Response[pipelinev1.PipelineStatus], error)
	Resume(context.Context, *connect.Request[pipelinev1.PipelineResumeRequest]) (*connect.Response[pipelinev1.PipelineResumeResponse], error)
	Cancel(context.Context, *connect.Request[pipelinev1.PipelineCancelRequest]) (*connect.Response[pipelinev1.PipelineCancelResponse], error)
	List(context.Context, *connect.Request[pipelinev1.PipelineListRequest]) (*connect.Response[pipelinev1.PipelineListResponse], error)
	GetActive(context.Context, *connect.Request[pipelinev1.GetActivePipelineRequest]) (*connect.Response[pipelinev1.ActivePipelineResponse], error)
	CreateActive(context.Context, *connect.Request[pipelinev1.CreatePipelineRequest]) (*connect.Response[pipelinev1.CreatePipelineResponse], error)
	ResetActive(context.Context, *connect.Request[pipelinev1.ScenarioPipelineRequest]) (*connect.Response[pipelinev1.ResetPipelineResponse], error)
	GetHistory(context.Context, *connect.Request[pipelinev1.PipelineHistoryRequest]) (*connect.Response[pipelinev1.PipelineHistoryResponse], error)
	Run(context.Context, *connect.Request[pipelinev1.PipelineRunRequest]) (*connect.Response[pipelinev1.PipelineRunResponse], error)
	StartActive(context.Context, *connect.Request[pipelinev1.StartActivePipelineRequest]) (*connect.Response[pipelinev1.StartActivePipelineResponse], error)
}

func newPipelineRPC(app *cliapp.ScenarioApp) pipelineRPC {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(app)
	return pipelineconnect.NewPipelineServiceClient(httpClient, baseURL)
}
