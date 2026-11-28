// Package internal holds shared Go Code Graph CLI adapters.
package internal

import (
	"github.com/vrooli/cli-core/cliapp"
	graphconnect "github.com/vrooli/vrooli/packages/proto/gen/go/go-code-graph/v1/graph/graph_v1connect"
)

// NewGoCodeGraphClient centralizes Connect client construction for the graph
// and rewrite command groups.
func NewGoCodeGraphClient(core *cliapp.ScenarioApp) graphconnect.GoCodeGraphServiceClient {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return graphconnect.NewGoCodeGraphServiceClient(httpClient, baseURL)
}
