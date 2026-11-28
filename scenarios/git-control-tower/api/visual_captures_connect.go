package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"
	visualcapturesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/visual_captures"
	visualcapturesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/visual_captures/visual_captures_v1connect"
)

var _ visualcapturesconnect.VisualCaptureServiceHandler = visualCaptureConnectServer{}

type visualCaptureConnectServer struct{ server *Server }

func (s visualCaptureConnectServer) ListCaptures(ctx context.Context, req *connect.Request[visualcapturesv1.ListCapturesRequest]) (*connect.Response[visualcapturesv1.ListCapturesResponse], error) {
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("visual capture list request is required"))
	}
	repo, err := s.resolveRepository(ctx, req.Msg.GetRepositoryId())
	if err != nil {
		return nil, err
	}
	slug, err := requiredCaptureScenarioSlug(req.Msg.GetScenarioSlug())
	if err != nil {
		return nil, err
	}
	store, err := s.storage()
	if err != nil {
		return nil, err
	}
	metas, err := store.ListSnapshotSets(repo.ID, slug)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	response := &visualcapturesv1.ListCapturesResponse{
		Snapshots: make([]*visualcapturesv1.SnapshotMeta, 0, len(metas)),
		Total:     int32(len(metas)),
	}
	for _, meta := range metas {
		response.Snapshots = append(response.Snapshots, snapshotMetaToProto(meta))
		if response.Staleness == nil && meta.Status == "complete" && meta.EffectiveRole() == SnapshotRoleCapture {
			response.Staleness = visualCaptureStalenessToProto(CheckCaptureStaleness(repo.Path, slug, meta.CreatedAt))
		}
	}
	return connect.NewResponse(response), nil
}

func (s visualCaptureConnectServer) GetCaptureDetail(ctx context.Context, req *connect.Request[visualcapturesv1.GetCaptureDetailRequest]) (*connect.Response[visualcapturesv1.GetCaptureDetailResponse], error) {
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("visual capture detail request is required"))
	}
	repo, err := s.resolveRepository(ctx, req.Msg.GetRepositoryId())
	if err != nil {
		return nil, err
	}
	slug, err := requiredCaptureScenarioSlug(req.Msg.GetScenarioSlug())
	if err != nil {
		return nil, err
	}
	id := strings.TrimSpace(req.Msg.GetCaptureId())
	if id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("capture_id is required"))
	}
	store, err := s.storage()
	if err != nil {
		return nil, err
	}
	detail, err := store.GetSnapshotSet(repo.ID, slug, id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	response := &visualcapturesv1.GetCaptureDetailResponse{Snapshot: snapshotMetaToProto(detail.SnapshotSetMeta)}
	for _, file := range detail.Screenshots {
		response.Screenshots = append(response.Screenshots, snapshotFileToProto(file))
	}
	for _, file := range detail.Videos {
		response.Videos = append(response.Videos, snapshotFileToProto(file))
	}
	return connect.NewResponse(response), nil
}

func (s visualCaptureConnectServer) GetStorageStats(ctx context.Context, req *connect.Request[visualcapturesv1.GetStorageStatsRequest]) (*connect.Response[visualcapturesv1.StorageStats], error) {
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("visual capture storage request is required"))
	}
	repo, err := s.resolveRepository(ctx, req.Msg.GetRepositoryId())
	if err != nil {
		return nil, err
	}
	store, err := s.storage()
	if err != nil {
		return nil, err
	}
	stats, err := store.GetStorageStats(repo.ID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	response := &visualcapturesv1.StorageStats{TotalSizeBytes: stats.TotalSizeBytes, SnapshotCount: int32(stats.SnapshotCount)}
	for _, item := range stats.PerScenario {
		response.PerScenario = append(response.PerScenario, &visualcapturesv1.ScenarioStorageBreakdown{ScenarioSlug: item.ScenarioSlug, SnapshotCount: int32(item.SnapshotCount), SizeBytes: item.SizeBytes})
	}
	return connect.NewResponse(response), nil
}

func (s visualCaptureConnectServer) GetScreenshotPath(ctx context.Context, req *connect.Request[visualcapturesv1.GetScreenshotPathRequest]) (*connect.Response[visualcapturesv1.GetScreenshotPathResponse], error) {
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("screenshot path request is required"))
	}
	repo, err := s.resolveRepository(ctx, req.Msg.GetRepositoryId())
	if err != nil {
		return nil, err
	}
	slug, err := requiredCaptureScenarioSlug(req.Msg.GetScenarioSlug())
	if err != nil {
		return nil, err
	}
	id := strings.TrimSpace(req.Msg.GetCaptureId())
	if id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("capture_id is required"))
	}
	filename := strings.TrimSpace(req.Msg.GetFilename())
	if filename == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("filename is required"))
	}
	store, err := s.storage()
	if err != nil {
		return nil, err
	}
	path, err := store.GetScreenshotFilePath(repo.ID, slug, id, filename)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "path traversal") {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("screenshot not found"))
	}
	return connect.NewResponse(&visualcapturesv1.GetScreenshotPathResponse{Path: path}), nil
}

func (s visualCaptureConnectServer) resolveRepository(ctx context.Context, id int64) (ResolvedRepo, error) {
	if s.server == nil {
		return ResolvedRepo{}, connect.NewError(connect.CodeUnavailable, fmt.Errorf("repository service is unavailable"))
	}
	var idValue string
	if id > 0 {
		idValue = fmt.Sprintf("%d", id)
	}
	repo, err := s.server.resolveRepoForConnect(ctx, idValue, "")
	if err != nil {
		return ResolvedRepo{}, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return repo, nil
}

func (s visualCaptureConnectServer) storage() (*VisualCaptureStorage, error) {
	if s.server == nil || s.server.visualCaptureStorage == nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("visual capture storage is unavailable"))
	}
	return s.server.visualCaptureStorage, nil
}

func requiredCaptureScenarioSlug(value string) (string, error) {
	slug := strings.TrimSpace(value)
	if slug == "" {
		return "", connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("scenario_slug is required"))
	}
	return slug, nil
}

func snapshotMetaToProto(v SnapshotSetMeta) *visualcapturesv1.SnapshotMeta {
	out := &visualcapturesv1.SnapshotMeta{
		Id: v.ID, ScenarioSlug: v.ScenarioSlug, Role: v.EffectiveRole(), CommitHash: v.CommitHash,
		TriggerType: v.TriggerType, Pages: append([]string(nil), v.Pages...), ScreenshotCount: int32(v.ScreenshotCount),
		VideoCount: int32(v.VideoCount), VideoStatus: v.VideoStatus, CreatedAt: v.CreatedAt.Format(time.RFC3339Nano),
		SizeBytes: v.SizeBytes, Status: v.Status, Error: v.Error, PageDiscoveryMethod: v.PageDiscoveryMethod,
	}
	for _, preset := range v.Presets {
		out.Presets = append(out.Presets, &visualcapturesv1.CapturePreset{Name: preset.Name, Width: int32(preset.Width), Height: int32(preset.Height), Theme: preset.Theme})
	}
	return out
}

func visualCaptureStalenessToProto(v *SnapshotStalenessInfo) *visualcapturesv1.SnapshotStalenessInfo {
	if v == nil {
		return nil
	}
	out := &visualcapturesv1.SnapshotStalenessInfo{IsStale: v.IsStale}
	if v.LastFileChange != nil {
		out.LastFileChange = v.LastFileChange.Format(time.RFC3339Nano)
	}
	if v.CaptureCreatedAt != nil {
		out.CaptureCreatedAt = v.CaptureCreatedAt.Format(time.RFC3339Nano)
	}
	return out
}

func snapshotFileToProto(v SnapshotFile) *visualcapturesv1.SnapshotFile {
	return &visualcapturesv1.SnapshotFile{Filename: v.Filename, PagePath: v.PagePath, PageLabel: v.PageLabel, Width: int32(v.Width), Height: int32(v.Height), ViewportWidth: int32(v.ViewportWidth), ViewportHeight: int32(v.ViewportHeight), Theme: v.Theme, SizeBytes: v.SizeBytes}
}
