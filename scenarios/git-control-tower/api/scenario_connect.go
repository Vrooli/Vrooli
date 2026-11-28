package main

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	scenariosv1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/scenarios"
	scenariosconnect "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/scenarios/scenarios_v1connect"
	"google.golang.org/protobuf/proto"
)

var _ scenariosconnect.ScenarioServiceHandler = (*scenarioConnectServer)(nil)

type scenarioConnectServer struct{ server *Server }

func (s scenarioConnectServer) ListScenarios(ctx context.Context, req *connect.Request[scenariosv1.ListScenariosRequest]) (*connect.Response[scenariosv1.ListScenariosResponse], error) {
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("scenario list request is required"))
	}
	items, err := s.server.scenarioLocator.List(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to list scenarios"))
	}
	resp := &scenariosv1.ListScenariosResponse{Scenarios: make([]*scenariosv1.ScenarioInfo, 0, len(items))}
	for _, item := range items {
		info := &scenariosv1.ScenarioInfo{Name: item.Name, DisplayName: item.DisplayName, Description: item.Description, Status: item.Status, Tags: item.Tags, Runtime: item.Runtime}
		if item.HealthStatus != nil {
			info.HealthStatus = proto.String(*item.HealthStatus)
		}
		resp.Scenarios = append(resp.Scenarios, info)
	}
	return connect.NewResponse(resp), nil
}

func (s scenarioConnectServer) GetScenarioEnvelope(ctx context.Context, req *connect.Request[scenariosv1.GetScenarioEnvelopeRequest]) (*connect.Response[scenariosv1.ScenarioEnvelope], error) {
	if req == nil || req.Msg == nil || req.Msg.GetSlug() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("scenario slug is required"))
	}
	envelope, lookupErr := s.server.getScenarioEnvelope(ctx, req.Msg.GetSlug())
	if lookupErr != nil {
		if lookupErr.status == 404 {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("scenario not found"))
		}
		if lookupErr.status == 400 {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("scenario slug is required"))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to load scenario envelope"))
	}
	return connect.NewResponse(scenarioEnvelopeToProto(envelope)), nil
}

func (s scenarioConnectServer) GetScenarioIsolation(ctx context.Context, req *connect.Request[scenariosv1.GetScenarioIsolationRequest]) (*connect.Response[scenariosv1.ScenarioIsolation], error) {
	if req == nil || req.Msg == nil || req.Msg.GetSlug() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("scenario slug is required"))
	}
	return connect.NewResponse(scenarioIsolationToProto(s.server.getScenarioIsolation(ctx, req.Msg.GetSlug()))), nil
}

func scenarioEnvelopeToProto(envelope *ScenarioEnvelopeResponse) *scenariosv1.ScenarioEnvelope {
	deps := &scenariosv1.ScenarioEnvelopeDependencies{Scenarios: envelope.Dependencies.Scenarios, Resources: envelope.Dependencies.Resources}
	lifecycle := &scenariosv1.ScenarioEnvelopeLifecycle{}
	if envelope.Lifecycle.TestCommand != "" {
		lifecycle.TestCommand = proto.String(envelope.Lifecycle.TestCommand)
	}
	if envelope.Lifecycle.BuildCommand != "" {
		lifecycle.BuildCommand = proto.String(envelope.Lifecycle.BuildCommand)
	}
	return &scenariosv1.ScenarioEnvelope{Name: envelope.Name, DisplayName: envelope.DisplayName, Description: envelope.Description, Path: envelope.Path, Tags: envelope.Tags, Dependencies: deps, Lifecycle: lifecycle}
}

func scenarioIsolationToProto(value IsolationResponse) *scenariosv1.ScenarioIsolation {
	resp := &scenariosv1.ScenarioIsolation{Status: string(value.Status), Reasons: value.Reasons}
	for _, violation := range value.Violations {
		mapped := &scenariosv1.ScenarioIsolationViolation{RuleId: violation.RuleID, Severity: violation.Severity}
		if violation.File != "" {
			mapped.File = proto.String(violation.File)
		}
		if violation.Line != 0 {
			mapped.Line = proto.Uint32(violation.Line)
		}
		if violation.Excerpt != "" {
			mapped.Excerpt = proto.String(violation.Excerpt)
		}
		resp.Violations = append(resp.Violations, mapped)
	}
	return resp
}
