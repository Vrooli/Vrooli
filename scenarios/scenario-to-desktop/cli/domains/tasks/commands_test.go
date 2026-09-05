package tasks

import (
	"context"
	"testing"

	"scenario-to-desktop/cli/internal/support"

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

func TestCommandRegistrationBuildsConnectClient(t *testing.T) {
	app, err := cliapp.NewStandardScenarioApp(cliapp.StandardScenarioOptions{Name: "scenario-to-desktop-test", Version: "test"})
	if err != nil {
		t.Fatalf("NewStandardScenarioApp() error: %v", err)
	}
	deps := support.Dependencies{Core: func() *cliapp.ScenarioApp { return app }}
	if New(deps).rpc == nil {
		t.Fatal("New() returned a nil RPC client")
	}
	group := Register(deps)
	if group.Name != "tasks" || len(group.Subcommands) != 1 || group.Subcommands[0].Name != "list" {
		t.Fatalf("unexpected group: %#v", group)
	}
}
