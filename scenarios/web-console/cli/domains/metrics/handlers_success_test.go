package metrics

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"
	metricsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/metrics"
	metricsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/metrics/metrics_v1connect"
)

type metricsTestClient struct {
	metricsconnect.MetricsServiceClient
}

func (metricsTestClient) Get(context.Context, *connect.Request[metricsv1.GetRequest]) (*connect.Response[metricsv1.GetResponse], error) {
	return connect.NewResponse(&metricsv1.GetResponse{}), nil
}

func TestRunRendersResponse(t *testing.T) {
	h := &handlers{client: metricsTestClient{}}
	if err := h.run(cliapp.NewTestRunContext(cliapp.TestRunContextOptions{JSON: true})); err != nil {
		t.Fatal(err)
	}
}
