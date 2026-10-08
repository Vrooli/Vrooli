package build

import (
	"context"
	"testing"

	"scenario-to-desktop/cli/internal/support"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/shared"
)

type fakeRPC struct{ request *domainv1.BuildStatusRequest }

func (f *fakeRPC) GetBuild(_ context.Context, req *connect.Request[domainv1.BuildStatusRequest]) (*connect.Response[sharedv1.BuildStatusResponse], error) {
	f.request = req.Msg
	return connect.NewResponse(&sharedv1.BuildStatusResponse{}), nil
}

func TestGetPrimitiveUsesTypedBuildRequest(t *testing.T) {
	fake := &fakeRPC{}
	schema := cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "build", Required: true}}}
	modes := cliapptest.RunPrimitiveHandlerModes(t, (&Commands{rpc: fake}).getPrimitive(), schema, []string{"build-1"}, nil)
	if modes.HumanErr != nil || modes.JSONErr != nil {
		t.Fatalf("build primitive errors: human=%v json=%v", modes.HumanErr, modes.JSONErr)
	}
	if fake.request.GetBuildId() != "build-1" {
		t.Fatalf("build ID = %q, want build-1", fake.request.GetBuildId())
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
	if group.Name != "build" || len(group.Subcommands) != 1 || group.Subcommands[0].Name != "get" {
		t.Fatalf("unexpected group: %#v", group)
	}
}
