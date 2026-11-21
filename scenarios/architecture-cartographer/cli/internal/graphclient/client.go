// Package graphclient owns construction of the CLI's shared Connect graph client.
package graphclient

import (
	"github.com/vrooli/cli-core/cliapp"
	graphconnect "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/graph/graph_v1connect"
)

func New(core *cliapp.ScenarioApp) graphconnect.GraphServiceClient {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return graphconnect.NewGraphServiceClient(httpClient, baseURL)
}
