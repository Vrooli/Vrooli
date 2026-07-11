// Package evidence implements GCT's phase-agnostic view over Test Genie run
// snapshots and typed artifact catalogs.
package evidence

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"connectrpc.com/connect"

	evidencepb "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/evidence"
	"github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/evidence/evidence_v1connect"
	runspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
)

const (
	defaultPageLimit = 20
	maxPageLimit     = 100
	defaultRunLimit  = 50
	maxRunLimit      = 200
)

// RunsClient is the Test Genie surface required by EvidenceService. The
// responses deliberately remain Test Genie proto records so additive
// descriptor and artifact metadata crosses GCT without a lossy translation.
type RunsClient interface {
	StartRun(ctx context.Context, request *runspb.StartRunRequest) (*runspb.StartRunResponse, error)
	ListRuns(ctx context.Context, scenario string, limit int) ([]*runspb.RunInfo, error)
	GetRun(ctx context.Context, scenario, runID string) (*runspb.GetRunResponse, error)
	ListRunArtifacts(ctx context.Context, scenario, runID string, kinds []string) (*runspb.ListRunArtifactsResponse, error)
}

func (s *Server) StartRun(ctx context.Context, req *connect.Request[runspb.StartRunRequest]) (*connect.Response[runspb.StartRunResponse], error) {
	if strings.TrimSpace(req.Msg.GetScenario()) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("scenario is required"))
	}
	response, err := s.runs.StartRun(ctx, req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	return connect.NewResponse(response), nil
}

type (
	Server struct{ runs RunsClient }
	Deps   struct{ Runs RunsClient }
)

func NewServer(d Deps) *Server { return &Server{runs: d.Runs} }

func NewHandler(d Deps, opts ...connect.HandlerOption) (string, http.Handler) {
	return evidence_v1connect.NewEvidenceServiceHandler(NewServer(d), opts...)
}

func pageBounds(limit, offset, total int) (start, end, normalizedLimit int) {
	if limit <= 0 {
		limit = defaultPageLimit
	}
	if limit > maxPageLimit {
		limit = maxPageLimit
	}
	if offset < 0 {
		offset = 0
	}
	start = min(offset, total)
	end = min(start+limit, total)
	return start, end, limit
}

func containsFold(value, query string) bool {
	return strings.Contains(strings.ToLower(value), strings.ToLower(query))
}

func runMatches(run *runspb.RunInfo, req *evidencepb.ListRunsRequest) bool {
	if req.GetStatus() != "" && run.GetStatus() != req.GetStatus() {
		return false
	}
	query := strings.TrimSpace(req.GetSearch())
	if query != "" {
		matched := containsFold(run.GetRunId(), query) || containsFold(run.GetGitSha(), query) || containsFold(run.GetGitBranch(), query)
		for _, phase := range run.GetPhases() {
			matched = matched || containsFold(phase.GetName(), query) || containsFold(phase.GetStatus(), query)
		}
		for _, descriptor := range run.GetDescriptorSnapshot().GetPhases() {
			matched = matched || containsFold(descriptor.GetPhase(), query) || containsFold(descriptor.GetDisplayName(), query) || containsFold(descriptor.GetDescription(), query) || containsFold(descriptor.GetProvider(), query)
		}
		if !matched {
			return false
		}
	}
	provider := strings.TrimSpace(req.GetProvider())
	phaseClass := strings.TrimSpace(req.GetPhaseClass())
	dimension := strings.TrimSpace(req.GetDimension())
	if provider == "" && phaseClass == "" && dimension == "" {
		return true
	}
	for _, descriptor := range run.GetDescriptorSnapshot().GetPhases() {
		if provider != "" && !strings.EqualFold(descriptor.GetProvider(), provider) {
			continue
		}
		if phaseClass != "" && !strings.EqualFold(descriptor.GetPhaseClass(), phaseClass) {
			continue
		}
		if dimension != "" {
			found := false
			for _, candidate := range descriptor.GetDimensions() {
				if strings.EqualFold(candidate, dimension) {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		return true
	}
	return false
}

func artifactMatches(artifact *runspb.ArtifactRef, query string) bool {
	query = strings.TrimSpace(query)
	if query == "" {
		return true
	}
	if containsFold(artifact.GetId(), query) || containsFold(artifact.GetKind(), query) || containsFold(artifact.GetLabel(), query) || containsFold(artifact.GetMediaType(), query) || containsFold(artifact.GetProducingPhase(), query) {
		return true
	}
	for key, value := range artifact.GetMetadata() {
		if containsFold(key, query) || containsFold(value, query) {
			return true
		}
	}
	return false
}

func (s *Server) ListRuns(ctx context.Context, req *connect.Request[evidencepb.ListRunsRequest]) (*connect.Response[evidencepb.ListRunsResponse], error) {
	scenario := strings.TrimSpace(req.Msg.GetScenario())
	if scenario == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("scenario is required"))
	}
	runs, err := s.runs.ListRuns(ctx, scenario, 0)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	filtered := make([]*runspb.RunInfo, 0, len(runs))
	for _, run := range runs {
		if run != nil && runMatches(run, req.Msg) {
			filtered = append(filtered, run)
		}
	}
	start, end, limit := pageBounds(int(req.Msg.GetLimit()), int(req.Msg.GetOffset()), len(filtered))
	return connect.NewResponse(&evidencepb.ListRunsResponse{
		Runs: filtered[start:end], Total: int32(len(filtered)), Limit: int32(limit),
		Offset: int32(start), HasMore: end < len(filtered),
	}), nil
}

func (s *Server) GetRun(ctx context.Context, req *connect.Request[evidencepb.GetRunRequest]) (*connect.Response[evidencepb.GetRunResponse], error) {
	scenario := strings.TrimSpace(req.Msg.GetScenario())
	runID := strings.TrimSpace(req.Msg.GetRunId())
	if scenario == "" || runID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("scenario and run_id are required"))
	}
	runResp, err := s.runs.GetRun(ctx, scenario, runID)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	catalog, err := s.runs.ListRunArtifacts(ctx, scenario, runID, req.Msg.GetArtifactKinds())
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	artifacts := make([]*runspb.ArtifactRef, 0, len(catalog.GetArtifacts()))
	for _, artifact := range catalog.GetArtifacts() {
		if artifact != nil && artifactMatches(artifact, req.Msg.GetArtifactSearch()) {
			artifacts = append(artifacts, artifact)
		}
	}
	start, end, limit := pageBounds(int(req.Msg.GetArtifactLimit()), int(req.Msg.GetArtifactOffset()), len(artifacts))
	degraded := append([]string(nil), runResp.GetDegradedReasons()...)
	degraded = append(degraded, catalog.GetDegradedReasons()...)
	return connect.NewResponse(&evidencepb.GetRunResponse{
		Run: runResp.GetRun(), TerminalSnapshotSchemaVersion: runResp.GetTerminalSnapshotSchemaVersion(), DegradedReasons: degraded,
		ArtifactCatalogSchemaVersion: catalog.GetSchemaVersion(), ArtifactCatalogDigest: catalog.GetDigest(),
		Artifacts: artifacts[start:end], ArtifactTotal: int32(len(artifacts)), ArtifactLimit: int32(limit),
		ArtifactOffset: int32(start), ArtifactsHaveMore: end < len(artifacts), LegacyArtifactsDiscovered: catalog.GetLegacyDiscovered(),
	}), nil
}

func (s *Server) ListEvidence(ctx context.Context, req *connect.Request[evidencepb.ListEvidenceRequest]) (*connect.Response[evidencepb.ListEvidenceResponse], error) {
	scenario := strings.TrimSpace(req.Msg.GetScenario())
	if scenario == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("scenario is required"))
	}
	runLimit := int(req.Msg.GetRunLimit())
	if runLimit <= 0 {
		runLimit = defaultRunLimit
	}
	if runLimit > maxRunLimit {
		runLimit = maxRunLimit
	}
	runs, err := s.runs.ListRuns(ctx, scenario, runLimit)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	items := make([]*evidencepb.EvidenceItem, 0)
	degraded := make([]string, 0)
	for _, run := range runs {
		if run == nil || (req.Msg.GetRunStatus() != "" && run.GetStatus() != req.Msg.GetRunStatus()) {
			continue
		}
		catalog, err := s.runs.ListRunArtifacts(ctx, scenario, run.GetRunId(), req.Msg.GetKinds())
		if err != nil {
			degraded = append(degraded, fmt.Sprintf("run %s: %v", run.GetRunId(), err))
			continue
		}
		for _, reason := range catalog.GetDegradedReasons() {
			degraded = append(degraded, fmt.Sprintf("run %s: %s", run.GetRunId(), reason))
		}
		for _, artifact := range catalog.GetArtifacts() {
			if artifact != nil && artifactMatches(artifact, req.Msg.GetSearch()) {
				items = append(items, &evidencepb.EvidenceItem{Run: run, Artifact: artifact})
			}
		}
	}
	sort.SliceStable(items, func(i, j int) bool {
		left, right := items[i], items[j]
		if left.GetRun().GetStartedAt() != right.GetRun().GetStartedAt() {
			return left.GetRun().GetStartedAt() > right.GetRun().GetStartedAt()
		}
		return left.GetArtifact().GetCreatedAt() > right.GetArtifact().GetCreatedAt()
	})
	start, end, limit := pageBounds(int(req.Msg.GetLimit()), int(req.Msg.GetOffset()), len(items))
	return connect.NewResponse(&evidencepb.ListEvidenceResponse{
		Items: items[start:end], Total: int32(len(items)), Limit: int32(limit), Offset: int32(start),
		HasMore: end < len(items), DegradedReasons: degraded,
	}), nil
}
