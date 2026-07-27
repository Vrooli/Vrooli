package deploytarget

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
)

type fakeRPC struct {
	saveRequest *domainv1.SaveDeployTargetRequest
	listCalls   int
}

func (f *fakeRPC) ListDeployTargets(context.Context, *connect.Request[domainv1.ListDeployTargetsRequest]) (*connect.Response[domainv1.ListDeployTargetsResponse], error) {
	f.listCalls++
	return connect.NewResponse(&domainv1.ListDeployTargetsResponse{}), nil
}

func (f *fakeRPC) GetDeployTarget(context.Context, *connect.Request[domainv1.DeployTargetNameRequest]) (*connect.Response[domainv1.GetDeployTargetResponse], error) {
	return connect.NewResponse(&domainv1.GetDeployTargetResponse{}), nil
}

func (f *fakeRPC) SaveDeployTarget(_ context.Context, request *connect.Request[domainv1.SaveDeployTargetRequest]) (*connect.Response[domainv1.SaveDeployTargetResponse], error) {
	f.saveRequest = request.Msg
	return connect.NewResponse(&domainv1.SaveDeployTargetResponse{Target: request.Msg.GetTarget()}), nil
}

func (f *fakeRPC) DeleteDeployTarget(context.Context, *connect.Request[domainv1.DeployTargetNameRequest]) (*connect.Response[domainv1.DeleteDeployTargetResponse], error) {
	return connect.NewResponse(&domainv1.DeleteDeployTargetResponse{Deleted: true}), nil
}

func (f *fakeRPC) TestDeployTarget(context.Context, *connect.Request[domainv1.TestDeployTargetRequest]) (*connect.Response[domainv1.TestDeployTargetResponse], error) {
	return connect.NewResponse(&domainv1.TestDeployTargetResponse{}), nil
}

func (f *fakeRPC) DiagnoseDeployTarget(context.Context, *connect.Request[domainv1.DeployTargetNameRequest]) (*connect.Response[domainv1.DiagnoseDeployTargetResponse], error) {
	return connect.NewResponse(&domainv1.DiagnoseDeployTargetResponse{}), nil
}

func TestSavePrimitiveUsesTypedDeployTargetRequest(t *testing.T) {
	fake := &fakeRPC{}
	commands := &Commands{rpc: fake}
	schema := cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "name", Required: true}}, Flags: []cliapp.Flag{{Name: "scenario", Required: true}, {Name: "profile", Required: true}, {Name: "label"}, {Name: "deployment-manager-profile-id"}}}
	modes := cliapptest.RunPrimitiveHandlerModes(t, commands.savePrimitive(), schema, []string{"release", "--scenario", "landing-page-business-suite", "--profile", "production", "--deployment-manager-profile-id", "profile-1"}, nil)
	if modes.HumanErr != nil || modes.JSONErr != nil {
		t.Fatalf("save primitive errors: human=%v json=%v", modes.HumanErr, modes.JSONErr)
	}
	if got := fake.saveRequest.GetTarget(); got.GetName() != "release" || got.GetScenarioName() != "landing-page-business-suite" || got.GetDeploymentManagerProfileId() != "profile-1" {
		t.Fatalf("saved target = %#v", got)
	}
}

func TestListPrimitiveUsesGeneratedClient(t *testing.T) {
	fake := &fakeRPC{}
	commands := &Commands{rpc: fake}
	modes := cliapptest.RunPrimitiveHandlerModes(t, commands.listPrimitive(), cliapp.ArgSchema{}, nil, nil)
	if modes.HumanErr != nil || modes.JSONErr != nil || fake.listCalls != 2 {
		t.Fatalf("list primitive = human=%v json=%v calls=%d", modes.HumanErr, modes.JSONErr, fake.listCalls)
	}
}
