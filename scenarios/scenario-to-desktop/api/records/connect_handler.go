package records

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"scenario-to-desktop-api/shared/validation"
	"strings"

	"connectrpc.com/connect"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
	"github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain/domainconnect"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ConnectService struct {
	domainconnect.UnimplementedDesktopRecordsServiceHandler
	handler *Handler
}

var _ domainconnect.DesktopRecordsServiceHandler = (*ConnectService)(nil)

func NewConnectService(handler *Handler) *ConnectService { return &ConnectService{handler: handler} }
func (s *ConnectService) require() error {
	if s == nil || s.handler == nil || s.handler.records == nil {
		return connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("desktop record store is not configured"))
	}
	return nil
}

func (s *ConnectService) ListDesktopRecords(_ context.Context, _ *connect.Request[emptypb.Empty]) (*connect.Response[domainv1.DesktopRecordsResponse], error) {
	if err := s.require(); err != nil {
		return nil, err
	}
	var results []RecordWithBuild
	for _, record := range s.handler.records.List() {
		item := RecordWithBuild{Record: record}
		if record != nil && record.BuildID != "" && s.handler.builds != nil {
			if build, ok := s.handler.builds.Get(record.BuildID); ok {
				item.Build, item.HasBuild, item.BuildState = build, true, build.Status
			}
		}
		if record != nil && s.handler.smokeTests != nil {
			if id, recording, ok := s.handler.smokeTests.GetByScenario(record.ScenarioName); ok {
				item.SmokeTestID, item.ScreenRecording = id, recording
			}
		}
		results = append(results, item)
	}
	return connect.NewResponse(&domainv1.DesktopRecordsResponse{Records: recordsToProto(results)}), nil
}

func (s *ConnectService) MoveDesktopRecord(_ context.Context, request *connect.Request[domainv1.MoveDesktopRecordRequest]) (*connect.Response[domainv1.MoveDesktopRecordResponse], error) {
	if err := s.require(); err != nil {
		return nil, err
	}
	record, ok := s.handler.records.Get(request.Msg.GetRecordId())
	if !ok || record == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("record not found"))
	}
	move := &MoveRequest{Target: request.Msg.GetTarget(), DestinationPath: request.Msg.GetDestinationPath()}
	source, destination, err := s.handler.resolveMovePaths(record, move)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err := performMove(source, destination); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	s.handler.updateRecordAfterMove(record, move, destination)
	return connect.NewResponse(&domainv1.MoveDesktopRecordResponse{RecordId: request.Msg.GetRecordId(), From: source, To: destination, Status: "moved"}), nil
}

func (s *ConnectService) DeleteDesktopScenario(_ context.Context, request *connect.Request[domainv1.DeleteDesktopScenarioRequest]) (*connect.Response[domainv1.DeleteDesktopScenarioResponse], error) {
	if err := s.require(); err != nil {
		return nil, err
	}
	scenario := request.Msg.GetScenarioName()
	if !validation.IsSafeScenarioName(scenario) {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid scenario name"))
	}
	if s.handler.outputPathFunc == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("delete not configured"))
	}
	path := s.handler.outputPathFunc(scenario)
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if !strings.Contains(absPath, filepath.Join("platforms", "electron")) {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("security violation: invalid path"))
	}
	_, statErr := os.Stat(path)
	if statErr == nil {
		if err := os.RemoveAll(path); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	} else if !os.IsNotExist(statErr) {
		return nil, connect.NewError(connect.CodeInternal, statErr)
	}
	removed := s.handler.records.DeleteByScenario(scenario)
	message := fmt.Sprintf("Desktop version of '%s' deleted successfully", scenario)
	if os.IsNotExist(statErr) {
		message = fmt.Sprintf("Desktop version of '%s' was already missing; cleaned up record state.", scenario)
	}
	return connect.NewResponse(&domainv1.DeleteDesktopScenarioResponse{Status: "success", ScenarioName: scenario, DeletedPath: path, RemovedRecords: int32(removed), Message: message}), nil
}

func recordsToProto(records []RecordWithBuild) []*domainv1.DesktopRecordWithBuild {
	result := make([]*domainv1.DesktopRecordWithBuild, 0, len(records))
	for _, item := range records {
		if item.Record == nil {
			continue
		}
		result = append(result, recordWithBuildToProto(item))
	}
	return result
}

func recordWithBuildToProto(item RecordWithBuild) *domainv1.DesktopRecordWithBuild {
	value := &domainv1.DesktopRecordWithBuild{Record: desktopRecordToProto(item.Record), HasBuild: item.HasBuild}
	if item.BuildState != "" {
		value.BuildState = stringPtr(item.BuildState)
	}
	if item.SmokeTestID != "" {
		value.SmokeTestId = stringPtr(item.SmokeTestID)
	}
	if item.Build != nil {
		value.BuildStatus = buildSummaryToProto(item.Build)
	}
	if item.ScreenRecording != nil {
		value.ScreenRecording = recordingToProto(item.ScreenRecording)
	}
	return value
}

func desktopRecordToProto(record *DesktopAppRecord) *domainv1.DesktopRecord {
	value := &domainv1.DesktopRecord{Id: record.ID, BuildId: record.BuildID, ScenarioName: record.ScenarioName, OutputPath: record.OutputPath, CreatedAt: timestamppb.New(record.CreatedAt), UpdatedAt: timestamppb.New(record.UpdatedAt)}
	setOptionalRecordFields(value, record)
	return value
}

func setOptionalRecordFields(target *domainv1.DesktopRecord, source *DesktopAppRecord) {
	if source.AppDisplayName != "" {
		target.AppDisplayName = stringPtr(source.AppDisplayName)
	}
	if source.TemplateType != "" {
		target.TemplateType = stringPtr(source.TemplateType)
	}
	if source.Framework != "" {
		target.Framework = stringPtr(source.Framework)
	}
	if source.LocationMode != "" {
		target.LocationMode = stringPtr(source.LocationMode)
	}
	if source.DestinationPath != "" {
		target.DestinationPath = stringPtr(source.DestinationPath)
	}
	if source.StagingPath != "" {
		target.StagingPath = stringPtr(source.StagingPath)
	}
	if source.CustomPath != "" {
		target.CustomPath = stringPtr(source.CustomPath)
	}
	if source.DeploymentMode != "" {
		target.DeploymentMode = stringPtr(source.DeploymentMode)
	}
	if source.Icon != "" {
		target.Icon = stringPtr(source.Icon)
	}
}

func buildSummaryToProto(value *BuildStatusView) *domainv1.DesktopBuildSummary {
	result := &domainv1.DesktopBuildSummary{Status: value.Status}
	if value.OutputPath != "" {
		result.OutputPath = stringPtr(value.OutputPath)
	}
	metadata := &domainv1.BuildMetadata{}
	if version, ok := value.Metadata["version"].(string); ok {
		metadata.Version = stringPtr(version)
	}
	if branch, ok := value.Metadata["git_branch"].(string); ok {
		metadata.GitBranch = stringPtr(branch)
	}
	if commit, ok := value.Metadata["git_commit_hash"].(string); ok {
		metadata.GitCommitHash = stringPtr(commit)
	}
	if dirty, ok := value.Metadata["git_dirty"].(bool); ok {
		metadata.GitDirty = boolPtr(dirty)
	}
	if metadata.Version != nil || metadata.GitBranch != nil || metadata.GitCommitHash != nil || metadata.GitDirty != nil {
		result.Metadata = metadata
	}
	return result
}

func recordingToProto(value *ScreenRecordingView) *domainv1.ScreenRecordingSummary {
	result := &domainv1.ScreenRecordingSummary{Recorded: value.Recorded}
	if value.DurationMS != 0 {
		result.DurationMs = int64Ptr(value.DurationMS)
	}
	if value.FileSizeBytes != 0 {
		result.FileSizeBytes = int64Ptr(value.FileSizeBytes)
	}
	if value.Error != "" {
		result.Error = stringPtr(value.Error)
	}
	return result
}

func stringPtr(value string) *string { return &value }
func boolPtr(value bool) *bool       { return &value }
func int64Ptr(value int64) *int64    { return &value }
