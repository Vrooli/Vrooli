package deploy

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/shared"
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
	connectErr := new(connect.Error)
	if !errors.As(err, &connectErr) {
		t.Fatalf("error is not a Connect error: %v", err)
	}
	assertRemediationEnvelope(t, connectErr)
}

func TestConnectServiceGetListOrderingAndMissingTargetErrors(t *testing.T) {
	t.Parallel()
	repo := NewTargetRepository(filepath.Join(t.TempDir(), "deploy-targets.json"))
	if err := repo.Save("zeta", &DeployTarget{Label: "Zeta", ScenarioName: "scenario-z", RemoteProfile: "prod", DeploymentManagerProfileID: "dm-z"}); err != nil {
		t.Fatal(err)
	}
	if err := repo.Save("alpha", &DeployTarget{Label: "Alpha", ScenarioName: "scenario-a", RemoteProfile: "stage"}); err != nil {
		t.Fatal(err)
	}
	service := NewConnectService(NewHandler(repo))
	listed, err := service.ListDeployTargets(context.Background(), connect.NewRequest(&domainv1.ListDeployTargetsRequest{}))
	if err != nil || len(listed.Msg.GetTargets()) != 2 || listed.Msg.GetTargets()[0].GetName() != "alpha" {
		t.Fatalf("ListDeployTargets() = %#v, %v", listed, err)
	}
	got, err := service.GetDeployTarget(context.Background(), connect.NewRequest(&domainv1.DeployTargetNameRequest{Name: "zeta"}))
	if err != nil || got.Msg.GetTarget().GetDeploymentManagerProfileId() != "dm-z" {
		t.Fatalf("GetDeployTarget() = %#v, %v", got, err)
	}
	for _, call := range []struct {
		name string
		err  error
	}{
		{"get", func() error {
			_, err := service.GetDeployTarget(context.Background(), connect.NewRequest(&domainv1.DeployTargetNameRequest{Name: "missing"}))
			return err
		}()},
		{"delete", func() error {
			_, err := service.DeleteDeployTarget(context.Background(), connect.NewRequest(&domainv1.DeployTargetNameRequest{Name: "missing"}))
			return err
		}()},
	} {
		if code := connect.CodeOf(call.err); code != connect.CodeNotFound {
			t.Errorf("%s missing target code = %v, want not found", call.name, code)
		}
	}
}

func TestConnectServiceDiagnosesAndRejectsIncompleteRemoteTargets(t *testing.T) {
	service := NewConnectService(NewHandler(NewTargetRepository(filepath.Join(t.TempDir(), "deploy-targets.json"))))
	if err := service.handler.repo.Save("incomplete", &DeployTarget{ScenarioName: "landing-page-business-suite"}); err != nil {
		t.Fatal(err)
	}
	_, err := service.DiagnoseDeployTarget(context.Background(), connect.NewRequest(&domainv1.DeployTargetNameRequest{Name: "incomplete"}))
	if code := connect.CodeOf(err); code != connect.CodeInvalidArgument {
		t.Fatalf("DiagnoseDeployTarget incomplete target code = %v, want invalid argument", code)
	}
	_, err = service.TestDeployTarget(context.Background(), connect.NewRequest(&domainv1.TestDeployTargetRequest{Name: "incomplete"}))
	if code := connect.CodeOf(err); code != connect.CodeInvalidArgument {
		t.Fatalf("TestDeployTarget incomplete target code = %v, want invalid argument", code)
	}
}

func TestDeployTargetProtoValidationAndDoctorHelpers(t *testing.T) {
	if value, err := deployTargetFromProto(&domainv1.DeployTarget{Name: "release", ScenarioName: "scenario", RemoteProfile: "prod"}); err != nil || value.ScenarioName != "scenario" {
		t.Fatalf("deployTargetFromProto() = %#v, %v", value, err)
	}
	if _, err := deployTargetFromProto(&domainv1.DeployTarget{Name: "release"}); err == nil {
		t.Fatal("expected incomplete target validation error")
	}
	if got := deployTargetToProto("missing", nil); got.GetName() != "missing" {
		t.Fatalf("nil target proto = %#v", got)
	}
	values := appendUnique([]string{"first"}, "first", " second ", "")
	if len(values) != 2 || values[1] != "second" {
		t.Fatalf("appendUnique = %#v", values)
	}
}

func assertRemediationEnvelope(t *testing.T, err *connect.Error) {
	t.Helper()
	for _, detail := range err.Details() {
		value, valueErr := detail.Value()
		if valueErr != nil {
			t.Fatalf("decode error detail: %v", valueErr)
		}
		if envelope, ok := value.(*sharedv1.ErrorEnvelope); ok {
			if envelope.GetCode() == "" || envelope.GetCategory() == "" || envelope.GetRecovery() == "" || envelope.GetRecoveryHint() == "" {
				t.Fatalf("incomplete remediation envelope: %#v", envelope)
			}
			return
		}
	}
	t.Fatal("Connect error did not carry a remediation envelope")
}
