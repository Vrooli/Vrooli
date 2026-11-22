package main

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"connectrpc.com/connect"
	tidinessv1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/tidiness"
)

type connectTidinessClient struct {
	score       *TidinessScoreResponse
	issues      []TidinessIssue
	stale       *TidinessStalenessInfo
	scan        *TidinessLightScanResult
	detail      *TidinessScenarioDetail
	err         error
	calls       []string
	scanRequest TidinessLightScanRequest
}

func (f *connectTidinessClient) GetTidinessScore(context.Context, string) (*TidinessScoreResponse, error) {
	f.calls = append(f.calls, "score")
	return f.score, f.err
}
func (f *connectTidinessClient) GetIssues(context.Context, string, string, string, string, int) ([]TidinessIssue, error) {
	f.calls = append(f.calls, "issues")
	return f.issues, f.err
}
func (f *connectTidinessClient) GetStaleness(context.Context, string) (*TidinessStalenessInfo, error) {
	f.calls = append(f.calls, "staleness")
	return f.stale, f.err
}
func (f *connectTidinessClient) TriggerLightScan(_ context.Context, req TidinessLightScanRequest) (*TidinessLightScanResult, error) {
	f.calls = append(f.calls, "scan")
	f.scanRequest = req
	return f.scan, f.err
}
func (f *connectTidinessClient) GetScenarioDetail(context.Context, string) (*TidinessScenarioDetail, error) {
	f.calls = append(f.calls, "detail")
	return f.detail, f.err
}

func tidinessTestServer(client TidinessManagerAPI, status CapabilityStatus) *Server {
	caps := NewCapabilityRegistry([]CapabilityDef{{ID: "tidiness-manager", Name: "Tidiness Manager", DependencyKind: DependencyScenario, DependencySlug: "tidiness-manager"}}, map[string]StatusChecker{"tidiness-manager": &FakeStatusChecker{Status: status}}, time.Minute)
	git := NewFakeGitRunner()
	git.RepoRoot = "/home/matthalloran8/Vrooli"
	return &Server{capabilities: caps, tidinessClient: client, git: git}
}

func TestTidinessConnectPreservesOwnerResultsAndRepoScopedScan(t *testing.T) {
	started := time.Date(2026, 11, 21, 10, 0, 0, 0, time.UTC)
	client := &connectTidinessClient{
		score:  &TidinessScoreResponse{Scenario: "git-control-tower", Score: 87.5, Violations: 3, Breakdown: &TidinessBreakdown{LintIssues: 1, LongFiles: 2}, Metrics: &TidinessMetricsSummary{TotalFiles: 12, AvgFileLength: 42.5}},
		issues: []TidinessIssue{{ID: 9, Scenario: "git-control-tower", FilePath: "api/main.go", Severity: "warning", LineNumber: intPtr(8), ColumnNumber: intPtr(3), AgentNotes: "owner note"}},
		stale:  &TidinessStalenessInfo{IsStale: true, ModifiedFiles: 2, StaleReason: "files changed"},
		scan:   &TidinessLightScanResult{Scenario: "git-control-tower", StartedAt: started, CompletedAt: started.Add(time.Second), Duration: 1000, FileMetrics: []TidinessFileMetric{{Path: "api/main.go", Lines: 12, Extension: "go"}}, LongFiles: []TidinessLongFile{{Path: "api/large.go", Lines: 901, Threshold: 900}}},
		detail: &TidinessScenarioDetail{Scenario: "git-control-tower", LightIssues: 1, AIIssues: 2, LongFiles: 1, Files: []TidinessScenarioFileInfo{{Path: "api/main.go", Lines: 12, TotalIssues: 3, VisitCount: 4}}},
	}
	s := tidinessConnectServer{server: tidinessTestServer(client, StatusAvailable)}

	score, err := s.GetScore(context.Background(), connect.NewRequest(&tidinessv1.GetScoreRequest{ScenarioName: "git-control-tower"}))
	if err != nil || score.Msg.GetScore() != 87.5 || score.Msg.GetBreakdown().GetLintIssues() != 1 || score.Msg.GetMetrics().GetTotalFiles() != 12 {
		t.Fatalf("score = %#v, %v", score, err)
	}
	issues, err := s.ListIssues(context.Background(), connect.NewRequest(&tidinessv1.ListIssuesRequest{ScenarioName: "git-control-tower", Limit: 5000, File: "api/main.go"}))
	if err != nil || len(issues.Msg.GetIssues()) != 1 || issues.Msg.GetIssues()[0].GetId() != 9 || issues.Msg.GetIssues()[0].GetLineNumber() != 8 || issues.Msg.GetIssues()[0].GetAgentNotes() != "owner note" {
		t.Fatalf("issues = %#v, %v", issues, err)
	}
	if client.calls[len(client.calls)-1] != "issues" {
		t.Fatalf("owner call order = %v", client.calls)
	}
	stale, err := s.GetStaleness(context.Background(), connect.NewRequest(&tidinessv1.GetStalenessRequest{ScenarioName: "git-control-tower"}))
	if err != nil || !stale.Msg.GetIsStale() || stale.Msg.GetModifiedFiles() != 2 {
		t.Fatalf("staleness = %#v, %v", stale, err)
	}
	scan, err := s.TriggerLightScan(context.Background(), connect.NewRequest(&tidinessv1.TriggerLightScanRequest{ScenarioName: "git-control-tower", TimeoutSec: 60, Incremental: true}))
	if err != nil {
		t.Fatal(err)
	}
	expectedPath := filepath.Join("/home/matthalloran8/Vrooli", "scenarios", "git-control-tower")
	if client.scanRequest.ScenarioPath != expectedPath || client.scanRequest.TimeoutSec != 60 || !client.scanRequest.Incremental || scan.Msg.GetDurationMs() != 1000 || len(scan.Msg.GetLongFiles()) != 1 {
		t.Fatalf("scan request/result = %#v / %#v", client.scanRequest, scan.Msg)
	}
	detail, err := s.GetScenarioDetail(context.Background(), connect.NewRequest(&tidinessv1.GetScenarioDetailRequest{ScenarioName: "git-control-tower"}))
	if err != nil || detail.Msg.GetAiIssues() != 2 || detail.Msg.GetFiles()[0].GetVisitCount() != 4 {
		t.Fatalf("scenario detail = %#v, %v", detail, err)
	}
}

func TestTidinessConnectKeepsUnavailableAndInvalidRequestsDistinct(t *testing.T) {
	client := &connectTidinessClient{err: errors.New("owner offline")}
	s := tidinessConnectServer{server: tidinessTestServer(client, StatusAvailable)}
	_, err := s.GetScore(context.Background(), connect.NewRequest(&tidinessv1.GetScoreRequest{ScenarioName: "demo"}))
	if connect.CodeOf(err) != connect.CodeUnavailable {
		t.Fatalf("owner failure code = %v, err=%v", connect.CodeOf(err), err)
	}
	_, err = s.GetScore(context.Background(), connect.NewRequest(&tidinessv1.GetScoreRequest{}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("empty scenario code = %v, err=%v", connect.CodeOf(err), err)
	}
	_, err = s.ListIssues(context.Background(), connect.NewRequest(&tidinessv1.ListIssuesRequest{ScenarioName: "demo", Limit: -1}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("negative limit code = %v, err=%v", connect.CodeOf(err), err)
	}
	if connect.CodeOf(err) == connect.CodeUnavailable && len(client.calls) != 1 {
		t.Fatalf("invalid request reached owner: %v", client.calls)
	}
}

func TestTidinessConnectUnavailableCapabilityDoesNotCallOwner(t *testing.T) {
	client := &connectTidinessClient{}
	s := tidinessConnectServer{server: tidinessTestServer(client, StatusUnavailable)}
	_, err := s.GetStaleness(context.Background(), connect.NewRequest(&tidinessv1.GetStalenessRequest{ScenarioName: "demo"}))
	if connect.CodeOf(err) != connect.CodeUnavailable || len(client.calls) != 0 {
		t.Fatalf("capability outage = err %v calls %v", err, client.calls)
	}
}

func intPtr(v int) *int { return &v }
