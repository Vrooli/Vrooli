package system

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"connectrpc.com/connect"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
	"github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain/domainconnect"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ConnectService exposes the same system state that the REST handlers expose.
// It reads Handler's injected stores and wine service directly, so transport
// behavior cannot diverge from the underlying domain state.
type ConnectService struct {
	domainconnect.UnimplementedSystemServiceHandler
	handler *Handler
}

var _ domainconnect.SystemServiceHandler = (*ConnectService)(nil)

func NewConnectService(handler *Handler) *ConnectService { return &ConnectService{handler: handler} }

func (s *ConnectService) GetSystemStatus(_ context.Context, _ *connect.Request[domainv1.GetSystemStatusRequest]) (*connect.Response[domainv1.SystemStatusResponse], error) {
	building, completed, failed, total := s.buildCounts()
	return connect.NewResponse(&domainv1.SystemStatusResponse{
		Service:      &domainv1.SystemServiceInfo{Name: "scenario-to-desktop", Version: "1.0.0", Description: "Transform Vrooli scenarios into professional desktop applications", Status: "running"},
		Statistics:   &domainv1.SystemBuildStatistics{TotalBuilds: int64(total), ActiveBuilds: int64(building), CompletedBuilds: int64(completed), FailedBuilds: int64(failed)},
		Capabilities: []string{"desktop_app_generation", "cross_platform_packaging", "template_system", "build_automation"}, SupportedFrameworks: []string{"electron"}, SupportedTemplates: []string{"universal", "advanced", "multi_window", "kiosk"},
	}), nil
}

func (s *ConnectService) ListTemplates(_ context.Context, _ *connect.Request[domainv1.ListTemplatesRequest]) (*connect.Response[domainv1.ListTemplatesResponse], error) {
	templates := templateInfos()
	result := make([]*domainv1.TemplateInfo, 0, len(templates))
	for _, item := range templates {
		result = append(result, templateInfoToProto(item))
	}
	return connect.NewResponse(&domainv1.ListTemplatesResponse{Templates: result, Count: int32(len(result))}), nil
}

func (s *ConnectService) GetTemplate(_ context.Context, req *connect.Request[domainv1.GetTemplateRequest]) (*connect.Response[domainv1.TemplateConfigResponse], error) {
	filename, ok := templateFilename(req.Msg.GetType())
	if !ok {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid template type %q", req.Msg.GetType()))
	}
	data, err := os.ReadFile(filepath.Join(s.handler.templateDir, "advanced", filename))
	if os.IsNotExist(err) {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("template %q was not found", req.Msg.GetType()))
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	var config map[string]any
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("parse template configuration: %w", err))
	}
	value, err := structpb.NewStruct(config)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&domainv1.TemplateConfigResponse{Config: value}), nil
}

func (s *ConnectService) CheckWine(_ context.Context, _ *connect.Request[domainv1.CheckWineRequest]) (*connect.Response[domainv1.WineCheckResponse], error) {
	if s.handler.wineService == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("wine service is not configured"))
	}
	value := s.handler.wineService.CheckStatus()
	response := &domainv1.WineCheckResponse{Installed: value.Installed, Platform: value.Platform, RequiredFor: value.RequiredFor}
	if value.Version != "" {
		response.Version = stringPtr(value.Version)
	}
	if value.RecommendedMethod != "" {
		response.RecommendedMethod = stringPtr(value.RecommendedMethod)
	}
	for _, method := range value.InstallMethods {
		response.InstallMethods = append(response.InstallMethods, &domainv1.WineInstallMethod{Id: method.ID, Name: method.Name, Description: method.Description, RequiresSudo: method.RequiresSudo, Estimated: method.Estimated, Steps: method.Steps})
	}
	return connect.NewResponse(response), nil
}

func (s *ConnectService) InstallWine(_ context.Context, req *connect.Request[domainv1.InstallWineRequest]) (*connect.Response[domainv1.WineInstallResponse], error) {
	if s.handler.wineService == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("wine service is not configured"))
	}
	if _, ok := templateInstallMethod(req.Msg.GetMethod()); !ok {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid installation method %q", req.Msg.GetMethod()))
	}
	installID, err := s.handler.wineService.StartInstallation(req.Msg.GetMethod())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&domainv1.WineInstallResponse{InstallId: installID, Status: "pending", Method: req.Msg.GetMethod()}), nil
}

func (s *ConnectService) GetWineInstallStatus(_ context.Context, req *connect.Request[domainv1.GetWineInstallStatusRequest]) (*connect.Response[domainv1.WineInstallStatusResponse], error) {
	if s.handler.wineService == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("wine service is not configured"))
	}
	status, ok := s.handler.wineService.GetInstallStatus(req.Msg.GetInstallId())
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("installation %q was not found", req.Msg.GetInstallId()))
	}
	response := &domainv1.WineInstallStatusResponse{InstallId: status.InstallID, Status: status.Status, Method: status.Method, StartedAt: timestamppb.New(status.StartedAt), Log: status.Log, ErrorLog: status.ErrorLog}
	if status.CompletedAt != nil {
		response.CompletedAt = timestamppb.New(*status.CompletedAt)
	}
	return connect.NewResponse(response), nil
}

func (s *ConnectService) buildCounts() (building, completed, failed, total int) {
	if s.handler == nil || s.handler.builds == nil {
		return
	}
	statuses := s.handler.builds.Snapshot()
	total = len(statuses)
	for _, status := range statuses {
		switch status.Status {
		case "building":
			building++
		case "ready":
			completed++
		case "failed":
			failed++
		}
	}
	return
}

func templateFilename(templateType string) (string, bool) {
	value, ok := map[string]string{"universal": "universal-app.json", "basic": "universal-app.json", "advanced": "advanced-app.json", "multi_window": "multi-window.json", "kiosk": "kiosk-mode.json"}[templateType]
	return value, ok
}

func templateInstallMethod(method string) (string, bool) {
	switch method {
	case "flatpak", "flatpak-auto", "appimage", "skip":
		return method, true
	default:
		return "", false
	}
}

func templateInfoToProto(value TemplateInfo) *domainv1.TemplateInfo {
	return &domainv1.TemplateInfo{Name: value.Name, Description: value.Description, Type: value.Type, Framework: value.Framework, UseCases: value.UseCases, Features: value.Features, Complexity: value.Complexity, Examples: value.Examples}
}
func stringPtr(value string) *string { return &value }
