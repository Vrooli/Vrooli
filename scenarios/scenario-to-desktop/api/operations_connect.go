package main

import (
	"context"
	"fmt"
	"strings"

	"scenario-to-desktop-api/scenario"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
	"github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain/domainconnect"
	"google.golang.org/protobuf/types/known/emptypb"
)

type operationsConnectService struct {
	domainconnect.UnimplementedOperationsServiceHandler
	server          *Server
	scenarioHandler *scenario.Handler
}

func (s operationsConnectService) ListDesktopScenarioStatus(_ context.Context, _ *connect.Request[emptypb.Empty]) (*connect.Response[domainv1.DesktopScenarioStatusResponse], error) {
	if s.scenarioHandler == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("scenario handler is not configured"))
	}
	result, err := s.scenarioHandler.ListDesktopStatus()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	response := &domainv1.DesktopScenarioStatusResponse{}
	if result.Stats != nil {
		response.Stats = &domainv1.DesktopScenarioStats{Total: int32(result.Stats.Total), WithDesktop: int32(result.Stats.WithDesktop), Built: int32(result.Stats.Built), WebOnly: int32(result.Stats.WebOnly)}
	}
	for _, item := range result.Scenarios {
		response.Scenarios = append(response.Scenarios, desktopScenarioStatusToProto(item))
	}
	return connect.NewResponse(response), nil
}

func desktopScenarioStatusToProto(item scenario.ScenarioDesktopStatus) *domainv1.DesktopScenarioStatus {
	result := &domainv1.DesktopScenarioStatus{Name: item.Name, HasDesktop: item.HasDesktop, Platforms: item.Platforms, Built: item.Built}
	setDesktopScenarioOptionalFields(result, item)
	result.ConnectionConfig = connectionStatusToProto(item.ConnectionConfig)
	for _, artifact := range item.BuildArtifacts {
		result.BuildArtifacts = append(result.BuildArtifacts, buildArtifactStatusToProto(artifact))
	}
	return result
}

func setDesktopScenarioOptionalFields(target *domainv1.DesktopScenarioStatus, source scenario.ScenarioDesktopStatus) {
	setScenarioIdentityFields(target, source)
	setScenarioArtifactFields(target, source)
	setScenarioRecordFields(target, source)
}

func setScenarioIdentityFields(target *domainv1.DesktopScenarioStatus, source scenario.ScenarioDesktopStatus) {
	if source.DisplayName != "" {
		target.DisplayName = &source.DisplayName
	}
	if source.ServiceDisplay != "" {
		target.ServiceDisplayName = &source.ServiceDisplay
	}
	if source.ServiceDesc != "" {
		target.ServiceDescription = &source.ServiceDesc
	}
	if source.ServiceIconPath != "" {
		target.ServiceIconPath = &source.ServiceIconPath
	}
	if source.DesktopPath != "" {
		target.DesktopPath = &source.DesktopPath
	}
	if source.Version != "" {
		target.Version = &source.Version
	}
}

func setScenarioArtifactFields(target *domainv1.DesktopScenarioStatus, source scenario.ScenarioDesktopStatus) {
	if source.DistPath != "" {
		target.DistPath = &source.DistPath
	}
	if source.LastModified != "" {
		target.LastModified = &source.LastModified
	}
	if source.PackageSize != 0 {
		target.PackageSize = &source.PackageSize
	}
	if source.ArtifactsSource != "" {
		target.ArtifactsSource = &source.ArtifactsSource
	}
	if source.ArtifactsPath != "" {
		target.ArtifactsPath = &source.ArtifactsPath
	}
	if source.ArtifactsExpectedPath != "" {
		target.ArtifactsExpectedPath = &source.ArtifactsExpectedPath
	}
}

func setScenarioRecordFields(target *domainv1.DesktopScenarioStatus, source scenario.ScenarioDesktopStatus) {
	if source.RecordID != "" {
		target.RecordId = &source.RecordID
	}
	if source.RecordOutputPath != "" {
		target.RecordOutputPath = &source.RecordOutputPath
	}
	if source.RecordLocationMode != "" {
		target.RecordLocationMode = &source.RecordLocationMode
	}
	if source.RecordUpdatedAt != "" {
		target.RecordUpdatedAt = &source.RecordUpdatedAt
	}
}

func connectionStatusToProto(value *scenario.DesktopConnectionConfig) *domainv1.DesktopConnectionStatus {
	if value == nil {
		return nil
	}
	result := &domainv1.DesktopConnectionStatus{}
	if value.Mode != "" {
		result.Mode = &value.Mode
	}
	if value.Endpoint != "" {
		result.Endpoint = &value.Endpoint
	}
	return result
}

func buildArtifactStatusToProto(value scenario.DesktopBuildArtifact) *domainv1.DesktopBuildArtifactStatus {
	result := &domainv1.DesktopBuildArtifactStatus{Platform: value.Platform, FileName: value.FileName, SizeBytes: value.SizeBytes}
	if value.ModifiedAt != "" {
		result.ModifiedAt = &value.ModifiedAt
	}
	if value.AbsolutePath != "" {
		result.AbsolutePath = &value.AbsolutePath
	}
	if value.RelativePath != "" {
		result.RelativePath = &value.RelativePath
	}
	return result
}

func (s operationsConnectService) ProbeEndpoints(ctx context.Context, request *connect.Request[domainv1.ProbeEndpointsRequest]) (*connect.Response[domainv1.ProbeEndpointsResponse], error) {
	if s.server == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("server is not configured"))
	}
	result, err := s.server.probeEndpoints(ctx, probeEndpointsRequest{
		ProxyURL:  request.Msg.GetProxyUrl(),
		ServerURL: request.Msg.GetServerUrl(),
		APIURL:    request.Msg.GetApiUrl(),
		TimeoutMs: int(request.Msg.GetTimeoutMs()),
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	response := &domainv1.ProbeEndpointsResponse{Server: probeEndpointResultToProto(result.Server), Api: probeEndpointResultToProto(result.API)}
	if result.ProxyURL != "" {
		response.ProxyUrl = &result.ProxyURL
	}
	return connect.NewResponse(response), nil
}

func probeEndpointResultToProto(result probeEndpointResult) *domainv1.ProbeEndpointResult {
	response := &domainv1.ProbeEndpointResult{Status: result.Status}
	if result.StatusCode != nil {
		value := int32(*result.StatusCode)
		response.StatusCode = &value
	}
	if result.Message != "" {
		response.Message = &result.Message
	}
	return response
}

func (s operationsConnectService) GetProxyHints(_ context.Context, request *connect.Request[domainv1.ProxyHintsRequest]) (*connect.Response[domainv1.ProxyHintsResponse], error) {
	if s.server == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("server is not configured"))
	}
	result := &domainv1.ProxyHintsResponse{ScenarioName: request.Msg.GetScenarioName()}
	for _, hint := range s.server.collectProxyHints(request.Msg.GetScenarioName()) {
		result.Hints = append(result.Hints, &domainv1.ProxyHint{Url: hint.URL, Source: hint.Source, Confidence: hint.Confidence, Message: hint.Message})
	}
	return connect.NewResponse(result), nil
}

var _ domainconnect.OperationsServiceHandler = (*operationsConnectService)(nil)

func (operationsConnectService) ResolveScenarioPort(ctx context.Context, request *connect.Request[domainv1.ScenarioPortRequest]) (*connect.Response[domainv1.ScenarioPortResponse], error) {
	scenario, portName := strings.TrimSpace(request.Msg.GetScenarioName()), strings.TrimSpace(request.Msg.GetPortName())
	if scenario == "" || portName == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("scenario_name and port_name are required"))
	}
	port, err := discovery.ResolveScenarioPort(ctx, scenario, portName)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("resolve scenario port: %w", err))
	}
	return connect.NewResponse(&domainv1.ScenarioPortResponse{ScenarioName: scenario, PortName: portName, Host: "127.0.0.1", Port: int32(port), Url: fmt.Sprintf("http://127.0.0.1:%d", port)}), nil
}
