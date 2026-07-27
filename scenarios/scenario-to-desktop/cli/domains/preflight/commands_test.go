package preflight

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
)

type fakeRPC struct {
	request *domainv1.GetPreflightJobRequest
}

func (f *fakeRPC) GetPreflightJob(_ context.Context, req *connect.Request[domainv1.GetPreflightJobRequest]) (*connect.Response[domainv1.JobStatusResponse], error) {
	f.request = req.Msg
	return connect.NewResponse(&domainv1.JobStatusResponse{}), nil
}

func TestGetPrimitiveUsesTypedJobRequest(t *testing.T) {
	fake := &fakeRPC{}
	schema := cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "job", Required: true}}}
	modes := cliapptest.RunPrimitiveHandlerModes(t, (&Commands{rpc: fake}).getPrimitive(), schema, []string{"job-1"}, nil)
	if modes.HumanErr != nil || modes.JSONErr != nil {
		t.Fatalf("preflight primitive errors: human=%v json=%v", modes.HumanErr, modes.JSONErr)
	}
	if fake.request.GetJobId() != "job-1" {
		t.Fatalf("job ID = %q, want job-1", fake.request.GetJobId())
	}
}
