package main

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/repo-contract-go/repocontracttest"
	visualcapturesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/visual_captures"
	visualcapturesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/visual_captures/visual_captures_v1connect"
)

type visualCaptureRepoStore struct{ record RepoRecord }

func (s visualCaptureRepoStore) List(context.Context) ([]RepoRecord, error) {
	return []RepoRecord{s.record}, nil
}
func (s visualCaptureRepoStore) GetByID(_ context.Context, id int64) (*RepoRecord, error) {
	if id != s.record.ID {
		return nil, sql.ErrNoRows
	}
	record := s.record
	return &record, nil
}
func (s visualCaptureRepoStore) GetByPath(_ context.Context, path string) (*RepoRecord, error) {
	if path != s.record.Path {
		return nil, sql.ErrNoRows
	}
	record := s.record
	return &record, nil
}
func (s visualCaptureRepoStore) Upsert(_ context.Context, record RepoRecord) (RepoRecord, error) {
	return record, nil
}
func (visualCaptureRepoStore) Delete(context.Context, int64) error { return nil }
func (s visualCaptureRepoStore) GetActive(context.Context) (*RepoRecord, error) {
	record := s.record
	return &record, nil
}
func (visualCaptureRepoStore) SetActive(context.Context, int64) error       { return nil }
func (visualCaptureRepoStore) ClearActive(context.Context) error            { return nil }
func (visualCaptureRepoStore) TouchLastOpened(context.Context, int64) error { return nil }

func TestVisualCaptureConnectReadGroupPreservesScopeAndMetadata(t *testing.T) {
	repoPath := t.TempDir()
	repocontracttest.WriteRepoContract(t, repoPath, "scenarios")
	repocontracttest.WriteScenarioStub(t, repoPath, "scenarios", "demo")
	scenarioDir := filepath.Join(repoPath, "scenarios", "demo")
	if err := os.MkdirAll(scenarioDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(scenarioDir, "main.go"), []byte("package demo\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	storage := NewVisualCaptureStorage(testStorageResolver(t, t.TempDir()), OSFileIO{})
	capturedAt := time.Now().UTC().Add(-time.Hour).Truncate(time.Second)
	if err := storage.SaveSnapshotSet(42, SnapshotSetMeta{
		ID: "cap-1", ScenarioSlug: "demo", Role: SnapshotRoleCapture, TriggerType: "manual",
		Pages: []string{"/"}, ScreenshotCount: 1, VideoCount: 1, CreatedAt: capturedAt,
		Status: "complete", Presets: []CapturePreset{{Name: "Desktop Dark", Width: 1440, Height: 900, Theme: "dark"}},
	}, map[string][]byte{"home@1440x900_dark.png": []byte("png-bytes")}, map[string][]byte{"home.webm": []byte("video-bytes")}); err != nil {
		t.Fatalf("seed disposable capture metadata: %v", err)
	}

	git := &FakeGitRunner{IsRepository: true, GitAvailable: true, RepoRoot: repoPath}
	owner := &Server{
		repos:                NewRepoService(visualCaptureRepoStore{record: RepoRecord{ID: 42, Path: repoPath}}, git),
		visualCaptureStorage: storage,
	}
	path, handler := visualcapturesconnect.NewVisualCaptureServiceHandler(visualCaptureConnectServer{server: owner})
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	client := visualcapturesconnect.NewVisualCaptureServiceClient(srv.Client(), srv.URL)

	listed, err := client.ListCaptures(context.Background(), connect.NewRequest(&visualcapturesv1.ListCapturesRequest{RepositoryId: 42, ScenarioSlug: "demo"}))
	if err != nil {
		t.Fatalf("ListCaptures: %v", err)
	}
	if listed.Msg.GetTotal() != 1 || len(listed.Msg.GetSnapshots()) != 1 || listed.Msg.GetSnapshots()[0].GetId() != "cap-1" {
		t.Fatalf("list response lost exact snapshot membership: %#v", listed.Msg)
	}
	if listed.Msg.GetStaleness() == nil || !listed.Msg.GetStaleness().GetIsStale() {
		t.Fatalf("list response did not preserve staleness: %#v", listed.Msg.GetStaleness())
	}

	detail, err := client.GetCaptureDetail(context.Background(), connect.NewRequest(&visualcapturesv1.GetCaptureDetailRequest{RepositoryId: 42, ScenarioSlug: "demo", CaptureId: "cap-1"}))
	if err != nil {
		t.Fatalf("GetCaptureDetail: %v", err)
	}
	if detail.Msg.GetSnapshot().GetRole() != SnapshotRoleCapture || len(detail.Msg.GetScreenshots()) != 1 || detail.Msg.GetScreenshots()[0].GetViewportWidth() != 1440 || len(detail.Msg.GetVideos()) != 1 {
		t.Fatalf("detail response lost capture or file metadata: %#v", detail.Msg)
	}

	stats, err := client.GetStorageStats(context.Background(), connect.NewRequest(&visualcapturesv1.GetStorageStatsRequest{RepositoryId: 42}))
	if err != nil {
		t.Fatalf("GetStorageStats: %v", err)
	}
	wantStats, err := storage.GetStorageStats(42)
	if err != nil {
		t.Fatalf("read expected owner stats: %v", err)
	}
	if stats.Msg.GetSnapshotCount() != int32(wantStats.SnapshotCount) || stats.Msg.GetTotalSizeBytes() != wantStats.TotalSizeBytes || len(stats.Msg.GetPerScenario()) != len(wantStats.PerScenario) {
		t.Fatalf("storage stats = %#v", stats.Msg)
	}

	pathResult, err := client.GetScreenshotPath(context.Background(), connect.NewRequest(&visualcapturesv1.GetScreenshotPathRequest{RepositoryId: 42, ScenarioSlug: "demo", CaptureId: "cap-1", Filename: "home@1440x900_dark.png"}))
	if err != nil {
		t.Fatalf("GetScreenshotPath: %v", err)
	}
	if _, err := os.Stat(pathResult.Msg.GetPath()); err != nil {
		t.Fatalf("typed path does not resolve to saved screenshot: %v", err)
	}
	_, err = client.GetScreenshotPath(context.Background(), connect.NewRequest(&visualcapturesv1.GetScreenshotPathRequest{RepositoryId: 42, ScenarioSlug: "demo", CaptureId: "cap-1", Filename: "../outside.png"}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("path traversal code = %v, want InvalidArgument", connect.CodeOf(err))
	}
}
