package build

import (
	"context"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
	"github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain/domainconnect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/shared"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ConnectService owns only transport mapping; build execution remains in Service.
type ConnectService struct {
	domainconnect.UnimplementedBuildServiceHandler
	handler *Handler
}

var _ domainconnect.BuildServiceHandler = (*ConnectService)(nil)

func NewConnectService(handler *Handler) *ConnectService { return &ConnectService{handler: handler} }

func (s *ConnectService) StartBuild(_ context.Context, req *connect.Request[domainv1.BuildRequest]) (*connect.Response[domainv1.BuildResponse], error) {
	if req.Msg.GetDesktopPath() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("desktop_path is required"))
	}
	platforms := platformsFromProto(req.Msg.GetPlatforms())
	if len(platforms) == 0 {
		platforms = []string{"linux"}
	}
	buildID := uuid.NewString()
	s.create(buildID, "", req.Msg.GetDesktopPath(), platforms)
	go s.handler.service.PerformDesktopBuild(buildID, &BuildRequest{DesktopPath: req.Msg.GetDesktopPath(), Platforms: platforms, Sign: req.Msg.GetSign(), Publish: req.Msg.GetPublish()})
	return connect.NewResponse(&domainv1.BuildResponse{BuildId: buildID, Status: "building", StatusUrl: "/vrooli.scenario_to_desktop.v1.domain.BuildService/GetBuild"}), nil
}

func (s *ConnectService) StartScenarioBuild(_ context.Context, req *connect.Request[domainv1.ScenarioBuildRequest]) (*connect.Response[domainv1.BuildResponse], error) {
	if req.Msg.GetScenarioName() == "" || req.Msg.GetDesktopPath() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("scenario_name and desktop_path are required"))
	}
	platforms := platformsFromProto(req.Msg.GetPlatforms())
	if len(platforms) == 0 {
		platforms = []string{"linux"}
	}
	buildID := uuid.NewString()
	s.create(buildID, req.Msg.GetScenarioName(), req.Msg.GetDesktopPath(), platforms)
	go s.handler.service.PerformScenarioDesktopBuild(buildID, req.Msg.GetScenarioName(), req.Msg.GetDesktopPath(), platforms, req.Msg.GetClean())
	return connect.NewResponse(&domainv1.BuildResponse{BuildId: buildID, Status: "building", StatusUrl: "/vrooli.scenario_to_desktop.v1.domain.BuildService/GetBuild"}), nil
}

func (s *ConnectService) GetBuild(_ context.Context, req *connect.Request[domainv1.BuildStatusRequest]) (*connect.Response[sharedv1.BuildStatusResponse], error) {
	status, ok := s.handler.store.Get(req.Msg.GetBuildId())
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("build %q not found", req.Msg.GetBuildId()))
	}
	return connect.NewResponse(StatusToProto(status)), nil
}

func (s *ConnectService) create(id, scenario, path string, platforms []string) {
	now := time.Now()
	results := make(map[string]*PlatformResult, len(platforms))
	for _, platform := range platforms {
		results[platform] = &PlatformResult{Platform: platform, Status: "building", StartedAt: &now}
	}
	s.handler.store.Save(&Status{BuildID: id, ScenarioName: scenario, Status: "building", RequestedPlatforms: platforms, PlatformResults: results, OutputPath: path, CreatedAt: now, BuildLog: []string{}, ErrorLog: []string{}, Artifacts: map[string]string{}, Metadata: map[string]interface{}{}})
}

// StatusToProto is the canonical boundary mapping for build status. Pipeline
// stage details reuse it to keep standalone and orchestrated views identical.
func StatusToProto(status *Status) *sharedv1.BuildStatusResponse {
	result := &sharedv1.BuildStatusResponse{BuildId: status.BuildID, ScenarioName: status.ScenarioName, Status: buildStatusProto(status.Status), RequestedPlatforms: platformsToProto(status.RequestedPlatforms), PlatformResults: map[string]*sharedv1.PlatformBuildResult{}, OutputPath: optional(status.OutputPath), CreatedAt: timestamppb.New(status.CreatedAt), ErrorLog: status.ErrorLog, BuildLog: status.BuildLog, Artifacts: status.Artifacts}
	if status.CompletedAt != nil {
		result.CompletedAt = timestamppb.New(*status.CompletedAt)
	}
	for platform, value := range status.PlatformResults {
		item := &sharedv1.PlatformBuildResult{Platform: platformProto(platform), Status: platformStatusProto(value.Status), ErrorLog: value.ErrorLog, Artifact: optional(value.Artifact), FileSize: optional(value.FileSize), SkipReason: optional(value.SkipReason)}
		if value.StartedAt != nil {
			item.StartedAt = timestamppb.New(*value.StartedAt)
		}
		if value.CompletedAt != nil {
			item.CompletedAt = timestamppb.New(*value.CompletedAt)
		}
		result.PlatformResults[platform] = item
	}
	return result
}

func platformsFromProto(values []sharedv1.Platform) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if v := platformString(value); v != "" {
			result = append(result, v)
		}
	}
	return result
}

func platformsToProto(values []string) []sharedv1.Platform {
	result := make([]sharedv1.Platform, 0, len(values))
	for _, value := range values {
		result = append(result, platformProto(value))
	}
	return result
}

func platformString(value sharedv1.Platform) string {
	switch value {
	case sharedv1.Platform_PLATFORM_WIN:
		return "win"
	case sharedv1.Platform_PLATFORM_MAC:
		return "mac"
	case sharedv1.Platform_PLATFORM_LINUX:
		return "linux"
	default:
		return ""
	}
}

func platformProto(value string) sharedv1.Platform {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if base, _, ok := strings.Cut(normalized, "-"); ok {
		normalized = base
	}
	switch normalized {
	case "win", "windows":
		return sharedv1.Platform_PLATFORM_WIN
	case "mac", "macos":
		return sharedv1.Platform_PLATFORM_MAC
	case "linux":
		return sharedv1.Platform_PLATFORM_LINUX
	default:
		return sharedv1.Platform_PLATFORM_UNSPECIFIED
	}
}

func buildStatusProto(value string) sharedv1.BuildStatus {
	switch value {
	case "building":
		return sharedv1.BuildStatus_BUILD_STATUS_BUILDING
	case "ready":
		return sharedv1.BuildStatus_BUILD_STATUS_READY
	case "partial":
		return sharedv1.BuildStatus_BUILD_STATUS_PARTIAL
	case "failed":
		return sharedv1.BuildStatus_BUILD_STATUS_FAILED
	default:
		return sharedv1.BuildStatus_BUILD_STATUS_UNSPECIFIED
	}
}

func platformStatusProto(value string) sharedv1.PlatformBuildStatus {
	switch value {
	case "building", "pending":
		return sharedv1.PlatformBuildStatus_PLATFORM_BUILD_STATUS_BUILDING
	case "ready":
		return sharedv1.PlatformBuildStatus_PLATFORM_BUILD_STATUS_READY
	case "failed":
		return sharedv1.PlatformBuildStatus_PLATFORM_BUILD_STATUS_FAILED
	case "skipped":
		return sharedv1.PlatformBuildStatus_PLATFORM_BUILD_STATUS_SKIPPED
	default:
		return sharedv1.PlatformBuildStatus_PLATFORM_BUILD_STATUS_UNSPECIFIED
	}
}

func optional[T comparable](value T) *T {
	var zero T
	if value == zero {
		return nil
	}
	return &value
}
