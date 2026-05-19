package executions

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/vrooli/browser-automation-studio/constants"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/internal/protoconv"
	"github.com/vrooli/browser-automation-studio/internal/typeconv"
	workflowservice "github.com/vrooli/browser-automation-studio/services/workflow"
	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
	basexecution "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/execution"
	bastimeline "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/timeline"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

const defaultSeedScenario = "browser-automation-studio"

// service implements apiconnect.ExecutionsServiceHandler.
type service struct {
	deps Deps
}

func (s *service) log() *logrus.Logger { return s.deps.Logger }

func parseExecutionID(raw string) (uuid.UUID, error) {
	id, err := uuid.Parse(strings.TrimSpace(raw))
	if err != nil {
		return uuid.Nil, errInvalidExecutionID
	}
	return id, nil
}

// ---------------------------------------------------------------------------
// ListExecutions
// ---------------------------------------------------------------------------

func (s *service) ListExecutions(
	ctx context.Context,
	req *connect.Request[basapi.ListExecutionsRequest],
) (*connect.Response[basapi.ListExecutionsResponse], error) {
	ctx, cancel := context.WithTimeout(ctx, constants.DefaultRequestTimeout)
	defer cancel()

	var workflowID *uuid.UUID
	if raw := strings.TrimSpace(req.Msg.GetWorkflowId()); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid workflow_id"))
		}
		workflowID = &id
	}
	var projectID *uuid.UUID
	if raw := strings.TrimSpace(req.Msg.GetProjectId()); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid project_id"))
		}
		projectID = &id
	}

	limit := int(req.Msg.GetLimit())
	offset := int(req.Msg.GetOffset())

	executions, err := s.deps.Executor.ListExecutions(ctx, workflowID, projectID, limit, offset)
	if err != nil {
		s.log().WithError(err).Error("list executions failed")
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	out := &basapi.ListExecutionsResponse{
		Executions: make([]*basexecution.Execution, 0, len(executions)),
	}
	if req.Msg.GetIncludeExportability() {
		out.Exportability = make(map[string]*basapi.ExecutionExportability, len(executions))
	}

	for idx := range executions {
		execIdx := executions[idx]
		if execIdx == nil {
			continue
		}
		pbExec, convErr := s.deps.Executor.HydrateExecutionProto(ctx, execIdx)
		if convErr != nil {
			s.log().WithError(convErr).WithField("execution_id", execIdx.ID).Error("hydrate execution proto failed")
			return nil, connect.NewError(connect.CodeInternal, convErr)
		}
		out.Executions = append(out.Executions, pbExec)
		if out.Exportability != nil {
			out.Exportability[execIdx.ID.String()] = computeExportability(execIdx.ResultPath, s.deps.RecordingsRoot, execIdx.ID.String())
		}
	}
	out.Total = int32(len(out.Executions))
	out.HasMore = limit > 0 && len(out.Executions) >= limit
	return connect.NewResponse(out), nil
}

// computeExportability mirrors the legacy lightweight filesystem probes used
// by the REST handler. Keeping it inline avoids a dependency on the parent
// handlers package.
func computeExportability(resultPath, recordingsRoot, executionID string) *basapi.ExecutionExportability {
	out := &basapi.ExecutionExportability{}
	if strings.TrimSpace(resultPath) == "" {
		return out
	}
	resultDir := filepath.Dir(resultPath)
	timelinePath := filepath.Join(resultDir, "timeline.proto.json")
	if info, err := os.Stat(timelinePath); err == nil && !info.IsDir() && info.Size() > 0 {
		out.HasTimeline = true
	}
	screenshotsDir := filepath.Join(resultDir, "screenshots")
	if entries, err := os.ReadDir(screenshotsDir); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				out.HasScreenshots = true
				break
			}
		}
	}
	if strings.TrimSpace(recordingsRoot) != "" && strings.TrimSpace(executionID) != "" {
		videoDir := filepath.Join(recordingsRoot, executionID, "artifacts", "videos")
		if entries, err := os.ReadDir(videoDir); err == nil {
			for _, entry := range entries {
				if !entry.IsDir() {
					out.HasRecordedVideo = true
					break
				}
			}
		}
	}
	out.IsExportable = out.HasTimeline || out.HasRecordedVideo
	return out
}

// ---------------------------------------------------------------------------
// GetExecution
// ---------------------------------------------------------------------------

func (s *service) GetExecution(
	ctx context.Context,
	req *connect.Request[basapi.GetExecutionRequest],
) (*connect.Response[basapi.GetExecutionResponse], error) {
	id, err := parseExecutionID(req.Msg.GetExecutionId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	ctx, cancel := context.WithTimeout(ctx, constants.DefaultRequestTimeout)
	defer cancel()

	execIdx, err := s.deps.Executor.GetExecution(ctx, id)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, errExecutionNotFound)
		}
		s.log().WithError(err).WithField("execution_id", id).Error("get execution failed")
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	pbExec, err := s.deps.Executor.HydrateExecutionProto(ctx, execIdx)
	if err != nil {
		s.log().WithError(err).WithField("execution_id", id).Error("hydrate execution proto failed")
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&basapi.GetExecutionResponse{Execution: pbExec}), nil
}

// ---------------------------------------------------------------------------
// GetExecutionTimeline
// ---------------------------------------------------------------------------

func (s *service) GetExecutionTimeline(
	ctx context.Context,
	req *connect.Request[basapi.GetExecutionTimelineRequest],
) (*connect.Response[bastimeline.ExecutionTimeline], error) {
	id, err := parseExecutionID(req.Msg.GetExecutionId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	ctx, cancel := context.WithTimeout(ctx, constants.ExtendedRequestTimeout)
	defer cancel()

	// Preferred: on-disk proto timeline.
	if pbTimeline, err := s.deps.Executor.GetExecutionTimelineProto(ctx, id); err == nil && pbTimeline != nil {
		return connect.NewResponse(pbTimeline), nil
	}

	// Fallback: legacy result.json conversion.
	timeline, err := s.deps.Executor.GetExecutionTimeline(ctx, id)
	if err != nil {
		s.log().WithError(err).WithField("execution_id", id).Error("get execution timeline failed")
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	pbTimeline, err := protoconv.TimelineToProto(timeline)
	if err != nil {
		s.log().WithError(err).WithField("execution_id", id).Error("convert timeline to proto failed")
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(pbTimeline), nil
}

// ---------------------------------------------------------------------------
// StopExecution
// ---------------------------------------------------------------------------

func (s *service) StopExecution(
	ctx context.Context,
	req *connect.Request[basapi.StopExecutionRequest],
) (*connect.Response[basapi.StopExecutionResponse], error) {
	id, err := parseExecutionID(req.Msg.GetExecutionId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	ctx, cancel := context.WithTimeout(ctx, constants.DefaultRequestTimeout)
	defer cancel()

	if err := s.deps.Executor.StopExecution(ctx, id); err != nil {
		s.log().WithError(err).WithField("execution_id", id).Error("stop execution failed")
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&basapi.StopExecutionResponse{Status: "stopped"}), nil
}

// ---------------------------------------------------------------------------
// ResumeExecution
// ---------------------------------------------------------------------------

func (s *service) ResumeExecution(
	ctx context.Context,
	req *connect.Request[basapi.ResumeExecutionRequest],
) (*connect.Response[basapi.ResumeExecutionResponse], error) {
	id, err := parseExecutionID(req.Msg.GetExecutionId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	ctx, cancel := context.WithTimeout(ctx, constants.ExtendedRequestTimeout)
	defer cancel()

	params := jsonObjectToMap(req.Msg.GetParameters())
	if resumeURL := strings.TrimSpace(req.Msg.GetResumeUrl()); resumeURL != "" {
		if params == nil {
			params = map[string]any{}
		}
		params["resume_url"] = resumeURL
	}

	newExec, err := s.deps.Executor.ResumeExecution(ctx, id, params)
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "cannot be resumed") || strings.Contains(errMsg, "not resumable") {
			return nil, connect.NewError(connect.CodeFailedPrecondition, err)
		}
		s.log().WithError(err).WithField("execution_id", id).Error("resume execution failed")
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	pbExec, err := protoconv.ExecutionToProto(newExec)
	if err != nil {
		s.log().WithError(err).WithField("execution_id", newExec.ID).Error("convert resumed execution to proto failed")
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&basapi.ResumeExecutionResponse{Execution: pbExec}), nil
}

func jsonObjectToMap(o *commonv1.JsonObject) map[string]any {
	if o == nil {
		return nil
	}
	return typeconv.JsonValueMapToAny(o.GetFields())
}

// ---------------------------------------------------------------------------
// Screenshots / artifact listings
// ---------------------------------------------------------------------------

func (s *service) GetExecutionScreenshots(
	ctx context.Context,
	req *connect.Request[basapi.GetExecutionScreenshotsRequest],
) (*connect.Response[basexecution.GetScreenshotsResponse], error) {
	id, err := parseExecutionID(req.Msg.GetExecutionId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	ctx, cancel := context.WithTimeout(ctx, constants.DefaultRequestTimeout)
	defer cancel()

	shots, err := s.deps.Executor.GetExecutionScreenshots(ctx, id)
	if err != nil {
		s.log().WithError(err).WithField("execution_id", id).Error("get execution screenshots failed")
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&basexecution.GetScreenshotsResponse{
		ExecutionId: id.String(),
		Screenshots: shots,
		Total:       int32(len(shots)),
	}), nil
}

func (s *service) GetExecutionRecordedVideos(
	ctx context.Context,
	req *connect.Request[basapi.GetExecutionArtifactsRequest],
) (*connect.Response[basapi.GetExecutionVideosResponse], error) {
	id, err := parseExecutionID(req.Msg.GetExecutionId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	ctx, cancel := context.WithTimeout(ctx, constants.DefaultRequestTimeout)
	defer cancel()

	videos, err := s.deps.Executor.GetExecutionVideoArtifacts(ctx, id)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, errExecutionNotFound)
		}
		s.log().WithError(err).WithField("execution_id", id).Error("get recorded videos failed")
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &basapi.GetExecutionVideosResponse{
		ExecutionId: id.String(),
		Videos:      make([]*basapi.ExecutionFileArtifact, 0, len(videos)),
	}
	for _, v := range videos {
		out.Videos = append(out.Videos, videoArtifactToProto(v))
	}
	return connect.NewResponse(out), nil
}

func (s *service) GetExecutionRecordedTraces(
	ctx context.Context,
	req *connect.Request[basapi.GetExecutionArtifactsRequest],
) (*connect.Response[basapi.GetExecutionTracesResponse], error) {
	id, err := parseExecutionID(req.Msg.GetExecutionId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	ctx, cancel := context.WithTimeout(ctx, constants.DefaultRequestTimeout)
	defer cancel()

	files, err := s.deps.Executor.GetExecutionTraceArtifacts(ctx, id)
	if err != nil {
		s.log().WithError(err).WithField("execution_id", id).Error("get recorded traces failed")
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &basapi.GetExecutionTracesResponse{
		ExecutionId: id.String(),
		Traces:      make([]*basapi.ExecutionFileArtifact, 0, len(files)),
	}
	for _, f := range files {
		out.Traces = append(out.Traces, fileArtifactToProto(f))
	}
	return connect.NewResponse(out), nil
}

func (s *service) GetExecutionRecordedHar(
	ctx context.Context,
	req *connect.Request[basapi.GetExecutionArtifactsRequest],
) (*connect.Response[basapi.GetExecutionHarResponse], error) {
	id, err := parseExecutionID(req.Msg.GetExecutionId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	ctx, cancel := context.WithTimeout(ctx, constants.DefaultRequestTimeout)
	defer cancel()

	files, err := s.deps.Executor.GetExecutionHarArtifacts(ctx, id)
	if err != nil {
		s.log().WithError(err).WithField("execution_id", id).Error("get recorded har failed")
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &basapi.GetExecutionHarResponse{
		ExecutionId: id.String(),
		HarFiles:    make([]*basapi.ExecutionFileArtifact, 0, len(files)),
	}
	for _, f := range files {
		out.HarFiles = append(out.HarFiles, fileArtifactToProto(f))
	}
	return connect.NewResponse(out), nil
}

func fileArtifactToProto(f workflowservice.ExecutionFileArtifact) *basapi.ExecutionFileArtifact {
	out := &basapi.ExecutionFileArtifact{
		ArtifactId:  f.ArtifactID,
		StorageUrl:  f.StorageURL,
		ContentType: f.ContentType,
		Label:       f.Label,
	}
	if f.SizeBytes != nil {
		v := *f.SizeBytes
		out.SizeBytes = &v
	}
	if len(f.Payload) > 0 {
		out.Payload = typeconv.ToJsonObject(f.Payload)
	}
	return out
}

func videoArtifactToProto(v workflowservice.ExecutionVideoArtifact) *basapi.ExecutionFileArtifact {
	return fileArtifactToProto(workflowservice.ExecutionFileArtifact(v))
}

// ---------------------------------------------------------------------------
// ScheduleExecutionSeedCleanup
// ---------------------------------------------------------------------------

func (s *service) ScheduleExecutionSeedCleanup(
	_ context.Context,
	req *connect.Request[basapi.ScheduleSeedCleanupRequest],
) (*connect.Response[basapi.ScheduleSeedCleanupResponse], error) {
	id, err := parseExecutionID(req.Msg.GetExecutionId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	token := strings.TrimSpace(req.Msg.GetCleanupToken())
	if token == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errMissingCleanupToken)
	}
	scenario := strings.TrimSpace(req.Msg.GetSeedScenario())
	if scenario == "" {
		scenario = defaultSeedScenario
	}
	if s.deps.SeedScheduler == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errSeedSchedulerMissing)
	}
	if err := s.deps.SeedScheduler.Schedule(id.String(), scenario, token); err != nil {
		s.log().WithError(err).WithField("execution_id", id).Error("schedule seed cleanup failed")
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&basapi.ScheduleSeedCleanupResponse{Status: "scheduled"}), nil
}
