package deploy

import (
	"context"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
)

func TestConnectServicePersistsAndListsDeployTargets(t *testing.T) {
	t.Parallel()
	service := NewConnectService(NewHandler(NewTargetRepository(filepath.Join(t.TempDir(), "deploy-targets.json"))))

	saved, err := service.SaveDeployTarget(context.Background(), connect.NewRequest(&domainv1.SaveDeployTargetRequest{Target: &domainv1.DeployTarget{Name: "release", Label: "Release", ScenarioName: "landing-page-business-suite", RemoteProfile: "production"}}))
	if err != nil {
		t.Fatalf("save deploy target: %v", err)
	}
	if got := saved.Msg.GetTarget().GetRemoteProfile(); got != "production" {
		t.Fatalf("remote profile = %q, want production", got)
	}

	listed, err := service.ListDeployTargets(context.Background(), connect.NewRequest(&domainv1.ListDeployTargetsRequest{}))
	if err != nil {
		t.Fatalf("list deploy targets: %v", err)
	}
	if len(listed.Msg.GetTargets()) != 1 || listed.Msg.GetTargets()[0].GetName() != "release" {
		t.Fatalf("listed targets = %#v", listed.Msg.GetTargets())
	}

	deleted, err := service.DeleteDeployTarget(context.Background(), connect.NewRequest(&domainv1.DeployTargetNameRequest{Name: "release"}))
	if err != nil || !deleted.Msg.GetDeleted() {
		t.Fatalf("delete deploy target = (%#v, %v)", deleted, err)
	}
}

func TestConnectServiceRejectsIncompleteDeployTarget(t *testing.T) {
	t.Parallel()
	service := NewConnectService(NewHandler(NewTargetRepository(filepath.Join(t.TempDir(), "deploy-targets.json"))))
	_, err := service.SaveDeployTarget(context.Background(), connect.NewRequest(&domainv1.SaveDeployTargetRequest{Target: &domainv1.DeployTarget{Name: "release"}}))
	if err == nil {
		t.Fatal("expected invalid target error")
	}
	if got := connect.CodeOf(err); got != connect.CodeInvalidArgument {
		t.Fatalf("error code = %v, want invalid argument", got)
	}
}
