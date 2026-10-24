package bundle

import (
	"context"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	pipelinev1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/pipeline"
	"github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/pipeline/pipelineconnect"
)

type pipelineRPC interface {
	CleanBundle(context.Context, *connect.Request[pipelinev1.BundleCleanRequest]) (*connect.Response[pipelinev1.BundleCleanResponse], error)
}

func newPipelineRPC(app *cliapp.ScenarioApp) pipelineRPC {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(app)
	return pipelineconnect.NewPipelineServiceClient(httpClient, baseURL)
}
