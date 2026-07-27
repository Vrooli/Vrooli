package state

import (
	"context"
	"scenario-to-desktop/cli/internal/support"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
)

type fakeRPC struct {
	request *domainv1.LoadScenarioStateRequest
}

func (f *fakeRPC) LoadScenarioState(_ context.Context, req *connect.Request[domainv1.LoadScenarioStateRequest]) (*connect.Response[domainv1.StateResponse], error) {
	f.request = req.Msg
	return connect.NewResponse(&domainv1.StateResponse{Found: true}), nil
}

func TestGetPrimitiveUsesTypedScenarioRequest(t *testing.T) {
	fake := &fakeRPC{}
	schema := cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "scenario", Required: true}}}
	modes := cliapptest.RunPrimitiveHandlerModes(t, (&Commands{rpc: fake}).getPrimitive(), schema, []string{"demo"}, nil)
	if modes.HumanErr != nil || modes.JSONErr != nil {
		t.Fatalf("state primitive errors: human=%v json=%v", modes.HumanErr, modes.JSONErr)
	}
	if fake.request.GetScenarioName() != "demo" {
		t.Fatalf("scenario = %q, want demo", fake.request.GetScenarioName())
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
	if group.Name != "state" || len(group.Subcommands) != 1 || group.Subcommands[0].Name != "get" {
		t.Fatalf("unexpected group: %#v", group)
	}
}
