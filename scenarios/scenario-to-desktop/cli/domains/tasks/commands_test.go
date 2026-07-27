package tasks

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
)

type fakeRPC struct{ request *domainv1.ListTasksRequest }

func (f *fakeRPC) ListTasks(_ context.Context, req *connect.Request[domainv1.ListTasksRequest]) (*connect.Response[domainv1.ListTasksResponse], error) {
	f.request = req.Msg
	return connect.NewResponse(&domainv1.ListTasksResponse{}), nil
}

func TestListPrimitiveUsesTypedPipelineRequest(t *testing.T) {
	fake := &fakeRPC{}
	schema := cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "pipeline", Required: true}}}
	modes := cliapptest.RunPrimitiveHandlerModes(t, (&Commands{rpc: fake}).listPrimitive(), schema, []string{"pipe-1"}, nil)
	if modes.HumanErr != nil || modes.JSONErr != nil {
		t.Fatalf("tasks primitive errors: human=%v json=%v", modes.HumanErr, modes.JSONErr)
	}
	if fake.request.GetPipelineId() != "pipe-1" {
		t.Fatalf("pipeline = %q, want pipe-1", fake.request.GetPipelineId())
	}
}
