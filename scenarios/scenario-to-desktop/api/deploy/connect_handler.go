package deploy

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"connectrpc.com/connect"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
	"github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain/domainconnect"
	sharedenv "scenario-to-desktop-api/shared/env"
)

// ConnectService exposes the durable deploy-target lifecycle through the
// scenario's generated contract. Credentials are deliberately resolved inside
// the service and never cross the transport boundary.
type ConnectService struct {
	domainconnect.UnimplementedDeployTargetServiceHandler
	handler *Handler
}

var _ domainconnect.DeployTargetServiceHandler = (*ConnectService)(nil)

func NewConnectService(handler *Handler) *ConnectService { return &ConnectService{handler: handler} }

func (s *ConnectService) ListDeployTargets(_ context.Context, _ *connect.Request[domainv1.ListDeployTargetsRequest]) (*connect.Response[domainv1.ListDeployTargetsResponse], error) {
	targets, err := s.handler.repo.List()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("list deploy targets: %w", err))
	}
	names := make([]string, 0, len(targets))
	for name := range targets {
		names = append(names, name)
	}
	sort.Strings(names)
	response := &domainv1.ListDeployTargetsResponse{Targets: make([]*domainv1.DeployTarget, 0, len(names))}
	for _, name := range names {
		response.Targets = append(response.Targets, deployTargetToProto(name, targets[name]))
	}
	return connect.NewResponse(response), nil
}

func (s *ConnectService) GetDeployTarget(_ context.Context, req *connect.Request[domainv1.DeployTargetNameRequest]) (*connect.Response[domainv1.GetDeployTargetResponse], error) {
	target, err := s.handler.repo.Get(req.Msg.GetName())
	if err != nil {
		return nil, deployTargetError("get deploy target", err)
	}
	return connect.NewResponse(&domainv1.GetDeployTargetResponse{Target: deployTargetToProto(req.Msg.GetName(), target)}), nil
}

func (s *ConnectService) SaveDeployTarget(_ context.Context, req *connect.Request[domainv1.SaveDeployTargetRequest]) (*connect.Response[domainv1.SaveDeployTargetResponse], error) {
	target := req.Msg.GetTarget()
	if target == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("deploy target is required"))
	}
	value, err := deployTargetFromProto(target)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err := s.handler.repo.Save(target.GetName(), value); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("save deploy target: %w", err))
	}
	return connect.NewResponse(&domainv1.SaveDeployTargetResponse{Target: deployTargetToProto(target.GetName(), value)}), nil
}

func (s *ConnectService) DeleteDeployTarget(_ context.Context, req *connect.Request[domainv1.DeployTargetNameRequest]) (*connect.Response[domainv1.DeleteDeployTargetResponse], error) {
	if err := s.handler.repo.Delete(req.Msg.GetName()); err != nil {
		return nil, deployTargetError("delete deploy target", err)
	}
	return connect.NewResponse(&domainv1.DeleteDeployTargetResponse{Name: req.Msg.GetName(), Deleted: true}), nil
}

func (s *ConnectService) TestDeployTarget(ctx context.Context, req *connect.Request[domainv1.TestDeployTargetRequest]) (*connect.Response[domainv1.TestDeployTargetResponse], error) {
	target, err := s.handler.repo.Get(req.Msg.GetName())
	if err != nil {
		return nil, deployTargetError("get deploy target", err)
	}
	if target.ScenarioName == "" || target.RemoteProfile == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("target missing required fields"))
	}
	secret := sharedenv.ResolveSecretWithSource("LPBS_SERVICE_SECRET")
	if strings.TrimSpace(secret.Value) == "" {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("%s", missingScenarioToDesktopSecretMessage()))
	}
	client := NewLPBSClient(target.ScenarioName, secret.Value)
	if err := client.TestRemoteProfile(ctx, target.RemoteProfile); err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("remote profile %q session test failed: %w", target.RemoteProfile, err))
	}
	if req.Msg.GetRequireServiceAuth() {
		status, err := client.GetServiceAuthStatus(ctx)
		if err != nil {
			return nil, connect.NewError(connect.CodeUnavailable, err)
		}
		if status == nil || !status.ServiceAuthConfigured {
			return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("service auth is not configured in %s runtime", target.ScenarioName))
		}
	}
	return connect.NewResponse(&domainv1.TestDeployTargetResponse{Target: deployTargetToProto(req.Msg.GetName(), target), ServiceAuthChecked: req.Msg.GetRequireServiceAuth()}), nil
}

func (s *ConnectService) DiagnoseDeployTarget(ctx context.Context, req *connect.Request[domainv1.DeployTargetNameRequest]) (*connect.Response[domainv1.DiagnoseDeployTargetResponse], error) {
	target, err := s.handler.repo.Get(req.Msg.GetName())
	if err != nil {
		return nil, deployTargetError("get deploy target", err)
	}
	if target.ScenarioName == "" || target.RemoteProfile == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("target missing required fields"))
	}
	report := runDeployTargetDoctor(ctx, req.Msg.GetName(), target)
	response := &domainv1.DiagnoseDeployTargetResponse{Target: deployTargetToProto(req.Msg.GetName(), target), Ready: report.Ready, NextSteps: report.NextSteps}
	for _, check := range report.Checks {
		response.Checks = append(response.Checks, &domainv1.DeployTargetReadinessCheck{Name: check.Name, Required: check.Required, Passed: check.Passed, Blocked: check.Blocked, Detail: check.Detail})
	}
	return connect.NewResponse(response), nil
}

func deployTargetToProto(name string, target *DeployTarget) *domainv1.DeployTarget {
	if target == nil {
		return &domainv1.DeployTarget{Name: name}
	}
	result := &domainv1.DeployTarget{Name: name, Label: target.Label, ScenarioName: target.ScenarioName, RemoteProfile: target.RemoteProfile}
	if target.DeploymentManagerProfileID != "" {
		result.DeploymentManagerProfileId = &target.DeploymentManagerProfileID
	}
	return result
}

func deployTargetFromProto(target *domainv1.DeployTarget) (*DeployTarget, error) {
	if strings.TrimSpace(target.GetName()) == "" || strings.TrimSpace(target.GetScenarioName()) == "" || strings.TrimSpace(target.GetRemoteProfile()) == "" {
		return nil, fmt.Errorf("name, scenario_name, and remote_profile are required")
	}
	return &DeployTarget{Label: target.GetLabel(), ScenarioName: target.GetScenarioName(), RemoteProfile: target.GetRemoteProfile(), DeploymentManagerProfileID: target.GetDeploymentManagerProfileId()}, nil
}

func deployTargetError(operation string, err error) error {
	code := connect.CodeInternal
	if strings.Contains(err.Error(), "not found") {
		code = connect.CodeNotFound
	}
	return connect.NewError(code, fmt.Errorf("%s: %w", operation, err))
}
