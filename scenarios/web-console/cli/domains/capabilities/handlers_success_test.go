package capabilities

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"
	capabilitiesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/capabilities"
	capabilitiesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/capabilities/capabilities_v1connect"
)

type capabilitiesTestClient struct {
	capabilitiesconnect.CapabilitiesServiceClient
}

func (capabilitiesTestClient) Get(context.Context, *connect.Request[capabilitiesv1.GetRequest]) (*connect.Response[capabilitiesv1.GetResponse], error) {
	return connect.NewResponse(&capabilitiesv1.GetResponse{}), nil
}
func (capabilitiesTestClient) Liveness(context.Context, *connect.Request[capabilitiesv1.LivenessRequest]) (*connect.Response[capabilitiesv1.LivenessResponse], error) {
	return connect.NewResponse(&capabilitiesv1.LivenessResponse{}), nil
}

func TestRunRendersBothModes(t *testing.T) {
	h := &handlers{client: capabilitiesTestClient{}}
	schema := cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "liveness", Bool: true}}}
	for _, live := range []bool{false, true} {
		if err := h.run(cliapp.NewTestRunContext(cliapp.TestRunContextOptions{Schema: schema, BoolFlags: map[string]bool{"liveness": live}, JSON: true})); err != nil {
			t.Fatal(err)
		}
	}
}
