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
	saveRequest   *domainv1.SaveDeployTargetRequest
	deleteRequest *domainv1.DeployTargetNameRequest
	testRequest   *domainv1.TestDeployTargetRequest
	doctorRequest *domainv1.DeployTargetNameRequest
	listCalls     int
	target        *domainv1.DeployTarget
}

func (f *fakeRPC) ListDeployTargets(context.Context, *connect.Request[domainv1.ListDeployTargetsRequest]) (*connect.Response[domainv1.ListDeployTargetsResponse], error) {
	f.listCalls++
	return connect.NewResponse(&domainv1.ListDeployTargetsResponse{}), nil
}

func (f *fakeRPC) GetDeployTarget(context.Context, *connect.Request[domainv1.DeployTargetNameRequest]) (*connect.Response[domainv1.GetDeployTargetResponse], error) {
	return connect.NewResponse(&domainv1.GetDeployTargetResponse{Target: f.target}), nil
}

func (f *fakeRPC) SaveDeployTarget(_ context.Context, request *connect.Request[domainv1.SaveDeployTargetRequest]) (*connect.Response[domainv1.SaveDeployTargetResponse], error) {
	f.saveRequest = request.Msg
	return connect.NewResponse(&domainv1.SaveDeployTargetResponse{Target: request.Msg.GetTarget()}), nil
}

func (f *fakeRPC) DeleteDeployTarget(_ context.Context, request *connect.Request[domainv1.DeployTargetNameRequest]) (*connect.Response[domainv1.DeleteDeployTargetResponse], error) {
	f.deleteRequest = request.Msg
	return connect.NewResponse(&domainv1.DeleteDeployTargetResponse{Name: request.Msg.GetName(), Deleted: true}), nil
}

func (f *fakeRPC) TestDeployTarget(_ context.Context, request *connect.Request[domainv1.TestDeployTargetRequest]) (*connect.Response[domainv1.TestDeployTargetResponse], error) {
	f.testRequest = request.Msg
	return connect.NewResponse(&domainv1.TestDeployTargetResponse{Target: f.target, ServiceAuthChecked: request.Msg.GetRequireServiceAuth()}), nil
}

func (f *fakeRPC) DiagnoseDeployTarget(_ context.Context, request *connect.Request[domainv1.DeployTargetNameRequest]) (*connect.Response[domainv1.DiagnoseDeployTargetResponse], error) {
	f.doctorRequest = request.Msg
	return connect.NewResponse(&domainv1.DiagnoseDeployTargetResponse{Target: f.target, Ready: false, Checks: []*domainv1.DeployTargetReadinessCheck{{Name: "profile", Passed: false, Detail: "profile needs login"}}, NextSteps: []string{"log in"}}), nil
}

func TestTargetPrimitivesSendTypedRequestsAndRenderReadiness(t *testing.T) {
	target := &domainv1.DeployTarget{Name: "release", Label: "Release", ScenarioName: "calculator", RemoteProfile: "production"}
	fake := &fakeRPC{target: target}
	commands := &Commands{rpc: fake}
	nameSchema := cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "name", Required: true}}}
	for name, handler := range map[string]cliapp.PrimitiveHandler{
		"get":    commands.getPrimitive(),
		"remove": commands.deletePrimitive(),
		"doctor": commands.doctorPrimitive(),
	} {
		t.Run(name, func(t *testing.T) {
			modes := cliapptest.RunPrimitiveHandlerModes(t, handler, nameSchema, []string{"release"}, nil)
			if modes.HumanErr != nil || modes.JSONErr != nil {
				t.Fatalf("%s errors: human=%v json=%v", name, modes.HumanErr, modes.JSONErr)
			}
		})
	}
	testSchema := cliapp.ArgSchema{Positionals: nameSchema.Positionals, Flags: []cliapp.Flag{{Name: "require-service-auth", Bool: true}}}
	modes := cliapptest.RunPrimitiveHandlerModes(t, commands.testPrimitive(), testSchema, []string{"release", "--require-service-auth"}, nil)
	if modes.HumanErr != nil || modes.JSONErr != nil {
		t.Fatalf("test errors: human=%v json=%v", modes.HumanErr, modes.JSONErr)
	}
	if fake.deleteRequest.GetName() != "release" || fake.testRequest.GetName() != "release" || !fake.testRequest.GetRequireServiceAuth() || fake.doctorRequest.GetName() != "release" {
		t.Fatalf("typed requests were not preserved: %#v %#v %#v", fake.deleteRequest, fake.testRequest, fake.doctorRequest)
	}
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
