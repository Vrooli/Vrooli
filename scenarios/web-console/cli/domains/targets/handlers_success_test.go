package targets

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/shared"
	targetsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/targets"
	targetsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/targets/targets_v1connect"
)

type targetsTestClient struct {
	targetsconnect.TargetCatalogServiceClient
}

func (targetsTestClient) List(context.Context, *connect.Request[targetsv1.ListRequest]) (*connect.Response[targetsv1.ListResponse], error) {
	return connect.NewResponse(&targetsv1.ListResponse{Targets: []*sharedv1.Target{{Id: "local", Label: "Local"}}}), nil
}
func (targetsTestClient) Get(context.Context, *connect.Request[targetsv1.GetRequest]) (*connect.Response[targetsv1.GetResponse], error) {
	return connect.NewResponse(&targetsv1.GetResponse{Target: &sharedv1.Target{Id: "local", Label: "Local"}}), nil
}
func (targetsTestClient) Doctor(context.Context, *connect.Request[targetsv1.DoctorRequest]) (*connect.Response[targetsv1.DoctorResponse], error) {
	return connect.NewResponse(&targetsv1.DoctorResponse{Target: &sharedv1.Target{Id: "local", Label: "Local"}, Summary: "ready"}), nil
}

func TestHandlersRenderSuccessfulResponses(t *testing.T) {
	h := &handlers{client: targetsTestClient{}}
	schema := cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "target-id"}}}
	ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{Schema: schema, Positionals: map[string]string{"target-id": "local"}, JSON: true})
	for _, call := range []func(cliapp.RunContext) error{h.list, h.get, h.doctor} {
		if err := call(ctx); err != nil {
			t.Fatal(err)
		}
	}
}
