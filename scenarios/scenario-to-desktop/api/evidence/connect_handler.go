// Package evidence exposes durable desktop-validation evidence over Connect.
package evidence

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"scenario-to-desktop-api/captures"
	"scenario-to-desktop-api/livedesktop"
	"scenario-to-desktop-api/shared/connecterrors"

	"connectrpc.com/connect"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
	"github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain/domainconnect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/shared"

	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	domainerrors "scenario-to-desktop-api/shared/errors"
)

type ConnectService struct {
	domainconnect.UnimplementedEvidenceServiceHandler
	desktops desktopService
	captures captureService
}

type desktopService interface {
	StartSession(context.Context, livedesktop.SessionConfig) (*livedesktop.Session, error)
	GetSession(string) (*livedesktop.Session, error)
	ListSessions() []*livedesktop.Session
	LaunchApp(string, string) error
	Heartbeat(string) error
	FindArtifact(string) (string, error)
	ExecuteAction(context.Context, string, string, json.RawMessage) (*livedesktop.ActionResult, error)
	StopSession(string) error
}

type captureService interface {
	Store() captures.Store
	DeleteCapture(string, string) error
	CleanAll(string) error
}

var _ domainconnect.EvidenceServiceHandler = (*ConnectService)(nil)

func NewConnectService(desktops desktopService, captureService captureService) *ConnectService {
	return &ConnectService{desktops: desktops, captures: captureService}
}

func (s *ConnectService) StartDesktopSession(ctx context.Context, req *connect.Request[domainv1.DesktopSessionRequest]) (*connect.Response[domainv1.DesktopSession], error) {
	if err := requireLocal(req.Msg.GetTarget()); err != nil {
		return nil, err
	}
	if req.Msg.GetScenarioName() == "" {
		return nil, evidenceError(connect.CodeInvalidArgument, domainerrors.CodeValidation, "scenario_name is required", nil, domainerrors.RecoveryFixInput)
	}
	session, err := s.desktops.StartSession(ctx, livedesktop.SessionConfig{ScenarioName: req.Msg.GetScenarioName(), AppPath: req.Msg.GetArtifactPath(), Platform: platformString(req.Msg.GetPlatform()), Width: int(req.Msg.GetWidth()), Height: int(req.Msg.GetHeight())})
	if err != nil {
		return nil, evidenceError(connect.CodeInternal, domainerrors.CodeInternal, "start desktop session", err, domainerrors.RecoveryRetry)
	}
	return connect.NewResponse(sessionToProto(session.View(), req.Msg.GetTarget())), nil
}

func (s *ConnectService) GetDesktopSession(_ context.Context, req *connect.Request[domainv1.DesktopSessionRef]) (*connect.Response[domainv1.DesktopSession], error) {
	session, err := s.desktops.GetSession(req.Msg.GetSessionId())
	if err != nil {
		return nil, evidenceError(connect.CodeNotFound, domainerrors.CodeNotFound, "desktop session not found", err, domainerrors.RecoveryFixInput)
	}
	return connect.NewResponse(sessionToProto(session.View(), nil)), nil
}

func (s *ConnectService) ListDesktopSessions(_ context.Context, req *connect.Request[domainv1.ListDesktopSessionsRequest]) (*connect.Response[domainv1.ListDesktopSessionsResponse], error) {
	response := &domainv1.ListDesktopSessionsResponse{}
	for _, session := range s.desktops.ListSessions() {
		view := session.View()
		if req.Msg.GetScenarioName() == "" || view.ScenarioName == req.Msg.GetScenarioName() {
			response.Sessions = append(response.Sessions, sessionToProto(view, nil))
		}
	}
	return connect.NewResponse(response), nil
}

func (s *ConnectService) LaunchDesktopArtifact(_ context.Context, req *connect.Request[domainv1.LaunchDesktopArtifactRequest]) (*connect.Response[domainv1.DesktopSession], error) {
	if err := s.desktops.LaunchApp(req.Msg.GetSessionId(), req.Msg.GetArtifactPath()); err != nil {
		return nil, evidenceError(connect.CodeFailedPrecondition, domainerrors.CodeBadRequest, "launch desktop artifact", err, domainerrors.RecoveryFixInput)
	}
	session, err := s.desktops.GetSession(req.Msg.GetSessionId())
	if err != nil {
		return nil, evidenceError(connect.CodeNotFound, domainerrors.CodeNotFound, "desktop session not found", err, domainerrors.RecoveryFixInput)
	}
	return connect.NewResponse(sessionToProto(session.View(), nil)), nil
}

func (s *ConnectService) HeartbeatDesktopSession(_ context.Context, req *connect.Request[domainv1.DesktopSessionRef]) (*connect.Response[domainv1.DesktopSession], error) {
	if err := s.desktops.Heartbeat(req.Msg.GetSessionId()); err != nil {
		return nil, evidenceError(connect.CodeNotFound, domainerrors.CodeNotFound, "desktop session not found", err, domainerrors.RecoveryFixInput)
	}
	session, err := s.desktops.GetSession(req.Msg.GetSessionId())
	if err != nil {
		return nil, evidenceError(connect.CodeNotFound, domainerrors.CodeNotFound, "desktop session not found", err, domainerrors.RecoveryFixInput)
	}
	return connect.NewResponse(sessionToProto(session.View(), nil)), nil
}

func (s *ConnectService) FindDesktopArtifact(_ context.Context, req *connect.Request[domainv1.FindDesktopArtifactRequest]) (*connect.Response[domainv1.FindDesktopArtifactResponse], error) {
	path, err := s.desktops.FindArtifact(req.Msg.GetScenarioName())
	if err != nil {
		return nil, evidenceError(connect.CodeNotFound, domainerrors.CodeNotFound, "desktop artifact not found", err, domainerrors.RecoveryFixInput)
	}
	return connect.NewResponse(&domainv1.FindDesktopArtifactResponse{ArtifactPath: path}), nil
}

func (s *ConnectService) CaptureScreenshot(ctx context.Context, req *connect.Request[domainv1.CaptureScreenshotRequest]) (*connect.Response[domainv1.CaptureScreenshotResponse], error) {
	result, err := s.desktops.ExecuteAction(ctx, req.Msg.GetSessionId(), "screenshot", nil)
	if err != nil {
		return nil, evidenceError(connect.CodeFailedPrecondition, domainerrors.CodeBadRequest, "capture screenshot", err, domainerrors.RecoveryFixInput)
	}
	id, _ := result.Data["capture_id"].(string)
	if id == "" {
		return nil, evidenceError(connect.CodeInternal, domainerrors.CodeInternal, "screenshot did not persist a durable capture", nil, domainerrors.RecoveryRetry)
	}
	session, err := s.desktops.GetSession(req.Msg.GetSessionId())
	if err != nil {
		return nil, evidenceError(connect.CodeNotFound, domainerrors.CodeNotFound, "desktop session not found", err, domainerrors.RecoveryFixInput)
	}
	all, err := s.captures.Store().List(session.View().ScenarioName)
	if err != nil {
		return nil, evidenceError(connect.CodeInternal, domainerrors.CodeInternal, "list evidence captures", err, domainerrors.RecoveryRetry)
	}
	for _, capture := range all {
		if capture.ID == id {
			return connect.NewResponse(&domainv1.CaptureScreenshotResponse{Capture: captureToProto(capture)}), nil
		}
	}
	return nil, evidenceError(connect.CodeInternal, domainerrors.CodeInternal, fmt.Sprintf("persisted capture %q not found", id), nil, domainerrors.RecoveryRetry)
}

func (s *ConnectService) ControlDesktop(ctx context.Context, req *connect.Request[domainv1.DesktopControlRequest]) (*connect.Response[domainv1.DesktopControlResponse], error) {
	paramsValue := map[string]any{}
	if req.Msg.GetParams() != nil {
		paramsValue = req.Msg.GetParams().AsMap()
	}
	params, err := json.Marshal(paramsValue)
	if err != nil {
		return nil, evidenceError(connect.CodeInvalidArgument, domainerrors.CodeValidation, "desktop action parameters are invalid", err, domainerrors.RecoveryFixInput)
	}
	result, err := s.desktops.ExecuteAction(ctx, req.Msg.GetSessionId(), req.Msg.GetAction(), params)
	if err != nil {
		return nil, evidenceError(connect.CodeFailedPrecondition, domainerrors.CodeBadRequest, "control desktop", err, domainerrors.RecoveryFixInput)
	}
	data := map[string]any{"status": result.Status, "message": result.Message, "data": result.Data}
	out, err := structpb.NewStruct(data)
	if err != nil {
		return nil, evidenceError(connect.CodeInternal, domainerrors.CodeInternal, "encode desktop action result", err, domainerrors.RecoveryRetry)
	}
	return connect.NewResponse(&domainv1.DesktopControlResponse{Result: out}), nil
}

func (s *ConnectService) StopDesktopSession(_ context.Context, req *connect.Request[domainv1.DesktopSessionRef]) (*connect.Response[domainv1.DesktopSession], error) {
	session, err := s.desktops.GetSession(req.Msg.GetSessionId())
	if err != nil {
		return nil, evidenceError(connect.CodeNotFound, domainerrors.CodeNotFound, "desktop session not found", err, domainerrors.RecoveryFixInput)
	}
	if err := s.desktops.StopSession(req.Msg.GetSessionId()); err != nil {
		return nil, evidenceError(connect.CodeFailedPrecondition, domainerrors.CodeBadRequest, "stop desktop session", err, domainerrors.RecoveryFixInput)
	}
	return connect.NewResponse(sessionToProto(session.View(), nil)), nil
}

func (s *ConnectService) ListEvidenceCaptures(_ context.Context, req *connect.Request[domainv1.ListEvidenceCapturesRequest]) (*connect.Response[domainv1.ListEvidenceCapturesResponse], error) {
	items, err := s.captures.Store().List(req.Msg.GetScenarioName())
	if err != nil {
		return nil, evidenceError(connect.CodeInternal, domainerrors.CodeInternal, "list evidence captures", err, domainerrors.RecoveryRetry)
	}
	response := &domainv1.ListEvidenceCapturesResponse{}
	for _, item := range items {
		response.Captures = append(response.Captures, captureToProto(item))
	}
	return connect.NewResponse(response), nil
}

func (s *ConnectService) GetEvidenceCapture(_ context.Context, req *connect.Request[domainv1.GetEvidenceCaptureRequest]) (*connect.Response[domainv1.GetEvidenceCaptureResponse], error) {
	items, err := s.captures.Store().List(req.Msg.GetScenarioName())
	if err != nil {
		return nil, evidenceError(connect.CodeInternal, domainerrors.CodeInternal, "list evidence captures", err, domainerrors.RecoveryRetry)
	}
	var found *captures.Capture
	for i := range items {
		if items[i].ID == req.Msg.GetCaptureId() {
			found = &items[i]
			break
		}
	}
	if found == nil {
		return nil, evidenceError(connect.CodeNotFound, domainerrors.CodeNotFound, "evidence capture not found", nil, domainerrors.RecoveryFixInput)
	}
	provider, ok := s.captures.(interface {
		CaptureFilePath(string, string) (string, error)
	})
	if !ok {
		return nil, evidenceError(connect.CodeUnimplemented, domainerrors.CodeNotImplemented, "evidence content is unavailable from this capture provider", nil, domainerrors.RecoveryNone)
	}
	path, err := provider.CaptureFilePath(found.ScenarioName, found.ID)
	if err != nil {
		return nil, evidenceError(connect.CodeNotFound, domainerrors.CodeNotFound, "evidence file not found", err, domainerrors.RecoveryFixInput)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, evidenceError(connect.CodeInternal, domainerrors.CodeInternal, "read evidence capture", err, domainerrors.RecoveryRetry)
	}
	return connect.NewResponse(&domainv1.GetEvidenceCaptureResponse{Capture: captureToProto(*found), Content: content}), nil
}

func (s *ConnectService) GetEvidenceCapturesSummary(_ context.Context, req *connect.Request[domainv1.ListEvidenceCapturesRequest]) (*connect.Response[domainv1.EvidenceCapturesSummary], error) {
	summary, err := s.captures.Store().Summary(req.Msg.GetScenarioName())
	if err != nil {
		return nil, evidenceError(connect.CodeInternal, domainerrors.CodeInternal, "summarize evidence captures", err, domainerrors.RecoveryRetry)
	}
	return connect.NewResponse(&domainv1.EvidenceCapturesSummary{Count: int32(summary.Count), TotalBytes: summary.TotalBytes}), nil
}

func (s *ConnectService) DeleteEvidenceCapture(_ context.Context, req *connect.Request[domainv1.EvidenceCaptureRef]) (*connect.Response[emptypb.Empty], error) {
	if err := s.captures.DeleteCapture(req.Msg.GetScenarioName(), req.Msg.GetCaptureId()); err != nil {
		return nil, evidenceError(connect.CodeNotFound, domainerrors.CodeNotFound, "evidence capture not found", err, domainerrors.RecoveryFixInput)
	}
	return connect.NewResponse(&emptypb.Empty{}), nil
}

func (s *ConnectService) DeleteAllEvidenceCaptures(_ context.Context, req *connect.Request[domainv1.ListEvidenceCapturesRequest]) (*connect.Response[emptypb.Empty], error) {
	if err := s.captures.CleanAll(req.Msg.GetScenarioName()); err != nil {
		return nil, evidenceError(connect.CodeInternal, domainerrors.CodeInternal, "delete evidence captures", err, domainerrors.RecoveryRetry)
	}
	return connect.NewResponse(&emptypb.Empty{}), nil
}

func requireLocal(target *domainv1.EvidenceTarget) error {
	if target != nil && target.GetKind() == domainv1.EvidenceTarget_KIND_BRIDGE_NODE {
		return evidenceError(connect.CodeUnimplemented, domainerrors.CodeNotImplemented, "bridge-node desktop execution must be dispatched through Bridge's allowlisted job service", nil, domainerrors.RecoveryNone)
	}
	return nil
}

func evidenceError(code connect.Code, semanticCode domainerrors.ErrorCode, message string, cause error, recovery domainerrors.RecoveryAction) error {
	err := domainerrors.New(semanticCode, message).WithCause(cause).WithRecovery(recovery, "Review the desktop session and request, then retry the operation").InDomain("evidence")
	return connecterrors.WithEnvelope(connect.NewError(code, err))
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
	switch strings.ToLower(value) {
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

func sessionToProto(value livedesktop.SessionView, target *domainv1.EvidenceTarget) *domainv1.DesktopSession {
	result := &domainv1.DesktopSession{SessionId: value.ID, ScenarioName: value.ScenarioName, Platform: platformProto(value.Platform), State: sessionStateProto(value.State), Width: int32(value.Width), Height: int32(value.Height), CreatedAt: timestamppb.New(value.CreatedAt), LastHeartbeatAt: timestamppb.New(value.LastHeartbeat), Error: optional(value.Error), AppRunning: value.AppRunning, Target: target, VncPort: int32(value.VNCPort), WebsocketPort: int32(value.WSPort), Recording: value.IsRecording, NetworkMode: networkModeProto(value.NetworkMode), DarkMode: value.DarkMode, Locale: optional(value.Locale), Metrics: metricsToProto(value.Metrics)}
	if value.BandwidthKbps != 0 {
		bandwidth := int32(value.BandwidthKbps)
		result.BandwidthKbps = &bandwidth
	}
	return result
}

func sessionStateProto(value livedesktop.SessionState) domainv1.DesktopSessionState {
	switch value {
	case livedesktop.StateCreating:
		return domainv1.DesktopSessionState_DESKTOP_SESSION_STATE_CREATING
	case livedesktop.StateRunning:
		return domainv1.DesktopSessionState_DESKTOP_SESSION_STATE_RUNNING
	case livedesktop.StateStopping:
		return domainv1.DesktopSessionState_DESKTOP_SESSION_STATE_STOPPING
	case livedesktop.StateStopped:
		return domainv1.DesktopSessionState_DESKTOP_SESSION_STATE_STOPPED
	case livedesktop.StateError:
		return domainv1.DesktopSessionState_DESKTOP_SESSION_STATE_ERROR
	default:
		return domainv1.DesktopSessionState_DESKTOP_SESSION_STATE_UNSPECIFIED
	}
}

func networkModeProto(value string) domainv1.DesktopNetworkMode {
	switch strings.ToLower(value) {
	case "", "normal":
		return domainv1.DesktopNetworkMode_DESKTOP_NETWORK_MODE_NORMAL
	case "offline":
		return domainv1.DesktopNetworkMode_DESKTOP_NETWORK_MODE_OFFLINE
	case "slow":
		return domainv1.DesktopNetworkMode_DESKTOP_NETWORK_MODE_SLOW
	default:
		return domainv1.DesktopNetworkMode_DESKTOP_NETWORK_MODE_UNSPECIFIED
	}
}

func metricsToProto(value *livedesktop.MetricsView) *domainv1.DesktopSessionMetrics {
	if value == nil {
		return nil
	}
	result := &domainv1.DesktopSessionMetrics{SplashDetected: value.SplashDetected, ReadyDetected: value.ReadyDetected, SampleCount: int32(value.SampleCount)}
	for _, role := range value.ProcessRoles {
		result.ProcessRoles = append(result.ProcessRoles, &domainv1.DesktopProcessRoleMetric{
			Role: string(role.Role), Available: role.Available, Unsupported: role.Unsupported,
			ProcessCount: int32(role.ProcessCount), CpuPercent: role.CPUPercent, PeakCpuPercent: role.PeakCPU,
			RssBytes: role.RSSBytes, PeakRssBytes: role.PeakRSSBytes, Threads: int32(role.Threads),
			DurationMs: role.DurationMs, SampleCount: int32(role.SampleCount),
		})
	}
	if value.SplashDurationMs != nil {
		result.SplashDurationMs = value.SplashDurationMs
	}
	if value.ReadyDurationMs != nil {
		result.ReadyDurationMs = value.ReadyDurationMs
	}
	if value.PerformanceStatus != "" {
		result.PerformanceStatus = &value.PerformanceStatus
	}
	if value.PerformanceReason != "" {
		result.PerformanceReason = &value.PerformanceReason
	}
	if value.ProtocolStartupDurationMs != nil {
		result.ProtocolStartupDurationMs = value.ProtocolStartupDurationMs
	}
	if value.DemoStartupDurationMs != nil {
		result.DemoStartupDurationMs = value.DemoStartupDurationMs
	}
	if value.ProtocolTraceRef != "" {
		result.ProtocolTraceRef = &value.ProtocolTraceRef
	}
	if value.DemoTraceRef != "" {
		result.DemoTraceRef = &value.DemoTraceRef
	}
	if value.CurrentCPU != nil {
		result.CurrentCpuPercent = value.CurrentCPU
	}
	if value.CurrentRSSMB != nil {
		result.CurrentRssMb = value.CurrentRSSMB
	}
	if value.PeakRSSMB != nil {
		result.PeakRssMb = value.PeakRSSMB
	}
	return result
}

func captureToProto(value captures.Capture) *domainv1.EvidenceCapture {
	result := &domainv1.EvidenceCapture{CaptureId: value.ID, ScenarioName: value.ScenarioName, Kind: string(value.Type), SourceSessionId: value.SourceSession, Filename: value.Filename, FileSizeBytes: value.FileSizeBytes, CreatedAt: timestamppb.New(value.CreatedAt)}
	result.Checksum = value.Checksum
	if value.Width != 0 {
		width := int32(value.Width)
		result.Width = &width
	}
	if value.Height != 0 {
		height := int32(value.Height)
		result.Height = &height
	}
	if value.DurationMs != 0 {
		duration := value.DurationMs
		result.DurationMs = &duration
	}
	return result
}

func optional(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
