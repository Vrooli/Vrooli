package smoketest

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
	"github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain/domainconnect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/shared"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ConnectService struct {
	domainconnect.UnimplementedSmokeTestServiceHandler
	service Service
	store   Store
	cancels CancelManager
}

var _ domainconnect.SmokeTestServiceHandler = (*ConnectService)(nil)

func NewConnectService(service Service, store Store, cancels CancelManager) *ConnectService {
	return &ConnectService{service: service, store: store, cancels: cancels}
}

func (s *ConnectService) StartSmokeTest(ctx context.Context, req *connect.Request[domainv1.SmokeTestStartRequest]) (*connect.Response[domainv1.SmokeTestStartResponse], error) {
	if req.Msg.GetScenarioName() == "" || req.Msg.GetArtifactPath() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("scenario_name and artifact_path are required"))
	}
	platform := platformString(req.Msg.GetPlatform())
	if platform == "" {
		platform = s.service.CurrentPlatform()
	}
	if platform == "" {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("unable to determine target platform"))
	}
	id := uuid.NewString()
	now := time.Now()
	s.store.Save(&Status{SmokeTestID: id, ScenarioName: req.Msg.GetScenarioName(), Platform: platform, Status: "running", ArtifactPath: req.Msg.GetArtifactPath(), StartedAt: now, Logs: []string{"Smoke test queued"}, CurrentState: StateInitializing})
	// Preserve request-scoped values (trace/auth) while deliberately detaching
	// cancellation: smoke-test lifetime is owned by the cancellation registry.
	runCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	s.cancels.SetCancel(id, cancel)
	go s.service.PerformSmokeTest(runCtx, id, req.Msg.GetScenarioName(), req.Msg.GetArtifactPath(), platform)
	return connect.NewResponse(&domainv1.SmokeTestStartResponse{SmokeTestId: id, ScenarioName: req.Msg.GetScenarioName(), Platform: platformProto(platform), Status: sharedv1.SmokeTestStatus_SMOKE_TEST_STATUS_RUNNING, ArtifactPath: &req.Msg.ArtifactPath, StartedAt: timestamppb.New(now), Logs: []string{"Smoke test queued"}}), nil
}

func (s *ConnectService) GetSmokeTest(_ context.Context, req *connect.Request[domainv1.SmokeTestStatusRequest]) (*connect.Response[sharedv1.SmokeTestStatusResponse], error) {
	status, ok := s.store.Get(req.Msg.GetSmokeTestId())
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("smoke test %q not found", req.Msg.GetSmokeTestId()))
	}
	return connect.NewResponse(StatusToProto(status)), nil
}

func (s *ConnectService) CancelSmokeTest(_ context.Context, req *connect.Request[domainv1.SmokeTestCancelRequest]) (*connect.Response[domainv1.SmokeTestCancelResponse], error) {
	if _, ok := s.store.Get(req.Msg.GetSmokeTestId()); !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("smoke test %q not found", req.Msg.GetSmokeTestId()))
	}
	if cancel := s.cancels.TakeCancel(req.Msg.GetSmokeTestId()); cancel != nil {
		cancel()
		return connect.NewResponse(&domainv1.SmokeTestCancelResponse{Status: "cancelling"}), nil
	}
	return connect.NewResponse(&domainv1.SmokeTestCancelResponse{Status: "already_terminal"}), nil
}

// StatusToProto is the canonical boundary mapping for smoke-test status.
// Pipeline stage details reuse it to keep every transport consistent.
func StatusToProto(v *Status) *sharedv1.SmokeTestStatusResponse {
	result := &sharedv1.SmokeTestStatusResponse{SmokeTestId: v.SmokeTestID, ScenarioName: v.ScenarioName, Platform: platformProto(v.Platform), Status: statusProto(v.Status), ArtifactPath: optional(v.ArtifactPath), StartedAt: timestamppb.New(v.StartedAt), Logs: v.Logs, Error: optional(v.Error), TelemetryUploaded: v.TelemetryUploaded, TelemetryUploadError: optional(v.TelemetryUploadError)}
	if v.CompletedAt != nil {
		result.CompletedAt = timestamppb.New(*v.CompletedAt)
	}
	if v.ScreenRecording != nil {
		result.ScreenRecording = &sharedv1.ScreenRecordingSummary{
			Recorded:  v.ScreenRecording.Recorded,
			CaptureId: optional(v.ScreenRecording.CaptureID),
			Error:     optional(v.ScreenRecording.Error),
		}
	}
	if v.EvidenceReview != nil {
		review := &sharedv1.EvidenceReview{
			SchemaVersion:    v.EvidenceReview.SchemaVersion,
			Capability:       v.EvidenceReview.Capability,
			PlanId:           v.EvidenceReview.PlanID,
			Profile:          v.EvidenceReview.Profile,
			Disposition:      v.EvidenceReview.Disposition,
			Reason:           optional(v.EvidenceReview.Reason),
			EventCount:       int32(v.EvidenceReview.EventCount),
			DeploymentMode:   optional(v.EvidenceReview.DeploymentMode),
			ProviderTier:     optional(v.EvidenceReview.ProviderTier),
			ServiceIdentity:  optional(v.EvidenceReview.ServiceIdentity),
			Readiness:        optional(v.EvidenceReview.Readiness),
			FallbackDecision: optional(v.EvidenceReview.FallbackDecision),
			SafeRouteClass:   optional(v.EvidenceReview.SafeRouteClass),
		}
		for _, chapter := range v.EvidenceReview.Chapters {
			review.Chapters = append(review.Chapters, &sharedv1.EvidenceChapter{
				Id:                 chapter.ID,
				Purpose:            chapter.Purpose,
				Action:             chapter.Action,
				Disposition:        chapter.Disposition,
				AssertionId:        optional(chapter.AssertionID),
				Expected:           optional(chapter.Expected),
				Observed:           optional(chapter.Observed),
				Error:              optional(chapter.Error),
				VideoStartOffsetMs: chapter.VideoStartOffsetMs,
				VideoEndOffsetMs:   chapter.VideoEndOffsetMs,
				EvidenceIds:        append([]string(nil), chapter.EvidenceIDs...),
			})
		}
		result.EvidenceReview = review
	}
	return result
}

func platformString(v sharedv1.Platform) string {
	switch v {
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

func platformProto(v string) sharedv1.Platform {
	switch v {
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

func statusProto(v string) sharedv1.SmokeTestStatus {
	if v == "passed" {
		return sharedv1.SmokeTestStatus_SMOKE_TEST_STATUS_PASSED
	}
	if v == "failed" {
		return sharedv1.SmokeTestStatus_SMOKE_TEST_STATUS_FAILED
	}
	return sharedv1.SmokeTestStatus_SMOKE_TEST_STATUS_RUNNING
}

func optional[T comparable](v T) *T {
	var zero T
	if v == zero {
		return nil
	}
	return &v
}
