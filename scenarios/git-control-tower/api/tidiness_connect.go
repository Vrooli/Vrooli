package main

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	tidinessv1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/tidiness"
	tidinessconnect "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/tidiness/tidiness_v1connect"
	"google.golang.org/protobuf/proto"
)

var _ tidinessconnect.TidinessServiceHandler = (*tidinessConnectServer)(nil)

type tidinessConnectServer struct{ server *Server }

func (s tidinessConnectServer) GetScore(ctx context.Context, req *connect.Request[tidinessv1.GetScoreRequest]) (*connect.Response[tidinessv1.Score], error) {
	if req == nil || req.Msg == nil {
		return nil, invalidConnectRequest("score request is required")
	}
	name, err := requiredScenario(req.Msg.GetScenarioName())
	if err != nil {
		return nil, err
	}
	ctx, cancel, err := s.server.tidinessContext(ctx, req.Msg.GetRepositoryId(), 10*time.Second)
	if err != nil {
		return nil, err
	}
	defer cancel()
	result, err := s.server.tidinessClient.GetTidinessScore(ctx, name)
	if err != nil {
		return nil, tidinessUnavailable(err)
	}
	return connect.NewResponse(scoreToProto(result)), nil
}

func (s tidinessConnectServer) ListIssues(ctx context.Context, req *connect.Request[tidinessv1.ListIssuesRequest]) (*connect.Response[tidinessv1.ListIssuesResponse], error) {
	if req == nil || req.Msg == nil {
		return nil, invalidConnectRequest("issue list request is required")
	}
	name, err := requiredScenario(req.Msg.GetScenarioName())
	if err != nil {
		return nil, err
	}
	limit := int(req.Msg.GetLimit())
	if limit < 0 {
		return nil, invalidConnectRequest("limit must be a positive integer")
	}
	if limit == 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}
	ctx, cancel, err := s.server.tidinessContext(ctx, req.Msg.GetRepositoryId(), 10*time.Second)
	if err != nil {
		return nil, err
	}
	defer cancel()
	issues, err := s.server.tidinessClient.GetIssues(ctx, name, strings.TrimSpace(req.Msg.GetFile()), strings.TrimSpace(req.Msg.GetCategory()), strings.TrimSpace(req.Msg.GetSeverity()), limit)
	if err != nil {
		return nil, tidinessUnavailable(err)
	}
	resp := &tidinessv1.ListIssuesResponse{Issues: make([]*tidinessv1.Issue, 0, len(issues))}
	for _, item := range issues {
		resp.Issues = append(resp.Issues, issueToProto(item))
	}
	return connect.NewResponse(resp), nil
}

func (s tidinessConnectServer) GetStaleness(ctx context.Context, req *connect.Request[tidinessv1.GetStalenessRequest]) (*connect.Response[tidinessv1.Staleness], error) {
	if req == nil || req.Msg == nil {
		return nil, invalidConnectRequest("staleness request is required")
	}
	name, err := requiredScenario(req.Msg.GetScenarioName())
	if err != nil {
		return nil, err
	}
	ctx, cancel, err := s.server.tidinessContext(ctx, req.Msg.GetRepositoryId(), 10*time.Second)
	if err != nil {
		return nil, err
	}
	defer cancel()
	result, err := s.server.tidinessClient.GetStaleness(ctx, name)
	if err != nil {
		return nil, tidinessUnavailable(err)
	}
	return connect.NewResponse(stalenessToProto(result)), nil
}

func (s tidinessConnectServer) TriggerLightScan(ctx context.Context, req *connect.Request[tidinessv1.TriggerLightScanRequest]) (*connect.Response[tidinessv1.LightScanResult], error) {
	if req == nil || req.Msg == nil {
		return nil, invalidConnectRequest("light scan request is required")
	}
	name, err := requiredScenario(req.Msg.GetScenarioName())
	if err != nil {
		return nil, err
	}
	ctx, cancel, err := s.server.tidinessContext(ctx, req.Msg.GetRepositoryId(), 120*time.Second)
	if err != nil {
		return nil, err
	}
	defer cancel()
	repo, err := s.server.resolveRepositorySelection(ctx, req.Msg.GetRepositoryId())
	if err != nil {
		return nil, err
	}
	absPath, err := resolveScenarioPath(repo.Path, name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("could not resolve scenario path"))
	}
	result, err := s.server.tidinessClient.TriggerLightScan(ctx, TidinessLightScanRequest{ScenarioPath: absPath, TimeoutSec: int(req.Msg.GetTimeoutSec()), Incremental: req.Msg.GetIncremental()})
	if err != nil {
		return nil, tidinessUnavailable(err)
	}
	return connect.NewResponse(lightScanToProto(result)), nil
}

func (s tidinessConnectServer) GetScenarioDetail(ctx context.Context, req *connect.Request[tidinessv1.GetScenarioDetailRequest]) (*connect.Response[tidinessv1.ScenarioDetail], error) {
	if req == nil || req.Msg == nil {
		return nil, invalidConnectRequest("scenario detail request is required")
	}
	name, err := requiredScenario(req.Msg.GetScenarioName())
	if err != nil {
		return nil, err
	}
	ctx, cancel, err := s.server.tidinessContext(ctx, req.Msg.GetRepositoryId(), 10*time.Second)
	if err != nil {
		return nil, err
	}
	defer cancel()
	result, err := s.server.tidinessClient.GetScenarioDetail(ctx, name)
	if err != nil {
		return nil, tidinessUnavailable(err)
	}
	return connect.NewResponse(scenarioDetailToProto(result)), nil
}

func (s *Server) tidinessContext(parent context.Context, repoID string, timeout time.Duration) (context.Context, context.CancelFunc, error) {
	ctx, cancel := context.WithTimeout(parent, timeout)
	if s.capabilities == nil || !s.capabilities.IsAvailable(ctx, "tidiness-manager") {
		cancel()
		return nil, nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("tidiness-manager is not available"))
	}
	if s.tidinessClient == nil {
		cancel()
		return nil, nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("tidiness-manager client is not configured"))
	}
	if _, err := s.resolveRepositorySelection(ctx, repoID); err != nil {
		cancel()
		return nil, nil, err
	}
	return ctx, cancel, nil
}

func (s *Server) resolveRepositorySelection(ctx context.Context, rawID string) (ResolvedRepo, error) {
	id := int64(0)
	if value := strings.TrimSpace(rawID); value != "" {
		parsed, err := strconv.ParseInt(value, 10, 64)
		if err != nil || parsed < 0 {
			return ResolvedRepo{}, invalidConnectRequest("repository_id must be a non-negative integer")
		}
		id = parsed
	}
	if s.repos != nil {
		repo, err := s.repos.ResolveID(ctx, id)
		if err != nil {
			return ResolvedRepo{}, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("repository context is unavailable: %s", err))
		}
		return repo, nil
	}
	if id != 0 || s.git == nil {
		return ResolvedRepo{}, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("repository context is unavailable"))
	}
	root := strings.TrimSpace(s.git.ResolveRepoRoot(ctx))
	if root == "" {
		return ResolvedRepo{}, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("repository root could not be resolved"))
	}
	return ResolvedRepo{Path: filepath.Clean(root), Source: "fallback"}, nil
}

func requiredScenario(raw string) (string, error) {
	name := strings.TrimSpace(raw)
	if name == "" {
		return "", invalidConnectRequest("scenario_name is required")
	}
	return name, nil
}

func invalidConnectRequest(message string) error {
	return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("%s", message))
}
func tidinessUnavailable(err error) error {
	return connect.NewError(connect.CodeUnavailable, fmt.Errorf("tidiness-manager request failed: %s", strings.TrimSpace(err.Error())))
}
func tidinessTime(value *time.Time) string {
	if value == nil || value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
}

func scoreToProto(x *TidinessScoreResponse) *tidinessv1.Score {
	if x == nil {
		return &tidinessv1.Score{}
	}
	out := &tidinessv1.Score{Scenario: x.Scenario, Score: x.Score, Violations: int32(x.Violations), LastScan: tidinessTime(x.LastScan)}
	if b := x.Breakdown; b != nil {
		out.Breakdown = &tidinessv1.Breakdown{LintScore: b.LintScore, TypeSafetyScore: b.TypeSafetyScore, ComplexityScore: b.ComplexityScore, FileLengthScore: b.FileLengthScore, TestCoverageScore: b.TestCoverageScore, TechDebtScore: b.TechDebtScore, CommentsScore: b.CommentsScore, DuplicationScore: b.DuplicationScore, LintIssues: int32(b.LintIssues), TypeIssues: int32(b.TypeIssues), LongFiles: int32(b.LongFiles), ComplexFunctions: int32(b.ComplexFunctions), TechDebtMarkers: int32(b.TechDebtMarkers), DuplicationIssues: int32(b.DuplicationIssues), TypeSafetyMarkers: int32(b.TypeSafetyMarkers), TestedFiles: int32(b.TestedFiles), TestableFiles: int32(b.TestableFiles)}
	}
	if m := x.Metrics; m != nil {
		out.Metrics = &tidinessv1.MetricsSummary{TotalFiles: int32(m.TotalFiles), TotalLines: int32(m.TotalLines), TotalCodeLines: int32(m.TotalCodeLines), Kloc: m.KLOC, AvgFileLength: m.AvgFileLength, MaxComplexity: int32(m.MaxComplexity), AvgComplexity: m.AvgComplexity, DuplicationPct: m.DuplicationPct, TestCoveragePct: m.TestCoveragePct}
	}
	return out
}
func issueToProto(x TidinessIssue) *tidinessv1.Issue {
	out := &tidinessv1.Issue{Id: int64(x.ID), Scenario: x.Scenario, FilePath: x.FilePath, Category: x.Category, Severity: x.Severity, Title: x.Title, Description: x.Description, AgentNotes: x.AgentNotes, RemediationSteps: x.RemediationSteps, Status: x.Status, CreatedAt: x.CreatedAt}
	if x.LineNumber != nil {
		out.LineNumber = proto.Int32(int32(*x.LineNumber))
	}
	if x.ColumnNumber != nil {
		out.ColumnNumber = proto.Int32(int32(*x.ColumnNumber))
	}
	return out
}
func stalenessToProto(x *TidinessStalenessInfo) *tidinessv1.Staleness {
	if x == nil {
		return &tidinessv1.Staleness{}
	}
	out := &tidinessv1.Staleness{IsStale: x.IsStale, StaleReason: x.StaleReason, RescanCommand: x.RescanCommand}
	if x.LastScanAt != nil {
		out.LastScanAt = *x.LastScanAt
	}
	if x.ModifiedFiles != 0 {
		out.ModifiedFiles = int32(x.ModifiedFiles)
	}
	return out
}
func lightScanToProto(x *TidinessLightScanResult) *tidinessv1.LightScanResult {
	if x == nil {
		return &tidinessv1.LightScanResult{}
	}
	out := &tidinessv1.LightScanResult{Scenario: x.Scenario, StartedAt: tidinessTime(&x.StartedAt), CompletedAt: tidinessTime(&x.CompletedAt), DurationMs: x.Duration, TotalFiles: int32(x.TotalFiles), TotalLines: int32(x.TotalLines), LintIssues: int32(x.LintIssuesCount), TypeIssues: int32(x.TypeIssuesCount), LongFilesCount: int32(x.LongFilesCount)}
	for _, item := range x.FileMetrics {
		out.FileMetrics = append(out.FileMetrics, &tidinessv1.FileMetric{Path: item.Path, Lines: int32(item.Lines), Extension: item.Extension})
	}
	for _, item := range x.LongFiles {
		out.LongFiles = append(out.LongFiles, &tidinessv1.LongFile{Path: item.Path, Lines: int32(item.Lines), Threshold: int32(item.Threshold)})
	}
	return out
}
func scenarioDetailToProto(x *TidinessScenarioDetail) *tidinessv1.ScenarioDetail {
	if x == nil {
		return &tidinessv1.ScenarioDetail{}
	}
	out := &tidinessv1.ScenarioDetail{Scenario: x.Scenario, LightIssues: int32(x.LightIssues), AiIssues: int32(x.AIIssues), LongFiles: int32(x.LongFiles)}
	for _, item := range x.Files {
		out.Files = append(out.Files, &tidinessv1.ScenarioFileInfo{Path: item.Path, Lines: int32(item.Lines), TotalIssues: int32(item.TotalIssues), VisitCount: int32(item.VisitCount)})
	}
	return out
}
