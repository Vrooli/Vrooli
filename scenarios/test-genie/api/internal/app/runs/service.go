// Package runs hosts the Connect-RPC RunsService handler that exposes
// test-genie's append-only run index (coverage/runs.index.json) to external
// callers (the test-genie CLI and git-control-tower baseline adapters). It is a
// thin wrapper over internal/shared/runs.
package runs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"connectrpc.com/connect"

	"test-genie/internal/execution"
	"test-genie/internal/orchestrator"
	"test-genie/internal/orchestrator/phases"
	"test-genie/internal/runmanager"
	"test-genie/internal/selfhealthsnapshots"
	sharedartifacts "test-genie/internal/shared/artifacts"
	sharedruns "test-genie/internal/shared/runs"

	"github.com/vrooli/api-core/discovery"
	runspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
	visualpb "github.com/vrooli/vrooli/packages/proto/gen/go/ui-health/v1/visualhealth"
	visualconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ui-health/v1/visualhealth/visualhealth_v1connect"
)

// executionPlanner previews a run plan to derive its ETA and surface validation
// errors synchronously (bad preset/phase) before a run is started.
type executionPlanner interface {
	Preview(ctx context.Context, req orchestrator.SuiteExecutionRequest) (*execution.ExecutionPlanPreview, error)
}

// Service implements runs_v1connect.RunsServiceHandler. Beyond the read-only run
// index surface, it owns the durable run lifecycle (StartRun/FollowRun/WaitRun/
// AbortRun/GetRunStatus) by delegating to the run manager.
type Service struct {
	scenariosRoot string
	runManager    *runmanager.Manager
	planner       executionPlanner
	// ledgerSource feeds the GetSelfHealth reliability ledger (compute-on-read
	// aggregation over persisted runs). Satisfied by *execution.SuiteExecutionRepository.
	ledgerSource ledgerSource
	// snapshotReader is the optional persisted self-health trend store. When
	// wired, GetSelfHealth fills the ledger's captured_at + trend delta against
	// the latest snapshot and (on include_trend) the windowed series. nil keeps
	// the compute-on-read path unchanged (no trend fields).
	snapshotReader snapshotReader
	visualHealth   visualHealthComparer
	// fleetSource feeds GetFleetHealth's fleet ledger (compute-on-read
	// aggregation over the stored runs of EVERY scenario). nil → GetFleetHealth
	// is Unimplemented. fleetRoster, when set, supplies the full fleet roster so
	// the ledger can surface never-tested-in-window coverage gaps.
	fleetSource fleetLedgerSource
	fleetRoster func(ctx context.Context) ([]string, error)
}

// ledgerSource is the read seam GetSelfHealth's reliability ledger composes over.
type ledgerSource interface {
	AggregatePhaseObservations(ctx context.Context, since time.Time, limit int) ([]execution.PhaseObservation, error)
	CountRunOutcomes(ctx context.Context, since time.Time, limit int) ([]execution.RunOutcomeCount, error)
}

// snapshotReader is the read seam over the persisted self-health snapshot store.
type snapshotReader interface {
	Latest(ctx context.Context) (selfhealthsnapshots.Snapshot, bool, error)
	Series(ctx context.Context, q selfhealthsnapshots.SeriesQuery) ([]selfhealthsnapshots.Snapshot, error)
}

// fleetLedgerSource is the read seam GetFleetHealth's fleet ledger composes over
// — satisfied by *execution.SuiteExecutionRepository. It is a superset of
// ledgerSource (adds the per-scenario run rollup); kept separate so widening it
// never disturbs the existing GetSelfHealth seam or its fakes.
type fleetLedgerSource interface {
	AggregatePhaseObservations(ctx context.Context, since time.Time, limit int) ([]execution.PhaseObservation, error)
	AggregateScenarioRuns(ctx context.Context, since time.Time, limit int) ([]execution.ScenarioRunRollup, error)
}

// SetSnapshotReader wires the optional persisted-trend read store. Returns the
// service for chaining at construction sites.
func (s *Service) SetSnapshotReader(r snapshotReader) *Service {
	s.snapshotReader = r
	return s
}

// SetFleetSource wires the GetFleetHealth fleet-ledger source and an optional
// roster provider (for never-tested-in-window). Returns the service for chaining.
func (s *Service) SetFleetSource(src fleetLedgerSource, roster func(ctx context.Context) ([]string, error)) *Service {
	s.fleetSource = src
	s.fleetRoster = roster
	return s
}

// NewService returns a Service. scenariosRoot resolves each request's scenario
// slug to its physical directory so the run index can be addressed. runManager
// and planner power the durable run-lifecycle RPCs. ledgerSource feeds the
// GetSelfHealth reliability ledger.
func NewService(scenariosRoot string, runManager *runmanager.Manager, planner executionPlanner, ledgerSource ledgerSource) *Service {
	return &Service{scenariosRoot: scenariosRoot, runManager: runManager, planner: planner, ledgerSource: ledgerSource, visualHealth: defaultVisualHealthComparer{}}
}

func (s *Service) SetVisualHealthComparer(comparer visualHealthComparer) *Service {
	s.visualHealth = comparer
	return s
}

type visualHealthComparer interface {
	CompareArtifacts(context.Context, *visualpb.CompareArtifactsRequest) (*visualpb.CompareArtifactsResponse, error)
}

type defaultVisualHealthComparer struct{}

func (defaultVisualHealthComparer) CompareArtifacts(ctx context.Context, req *visualpb.CompareArtifactsRequest) (*visualpb.CompareArtifactsResponse, error) {
	baseURL, err := discovery.ResolveScenarioURLDefault(ctx, "ui-health")
	if err != nil {
		return nil, err
	}
	client := visualconnect.NewVisualHealthServiceClient(&http.Client{Timeout: 60 * time.Second}, baseURL)
	resp, err := client.CompareArtifacts(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, err
	}
	return resp.Msg, nil
}

func (s *Service) scenarioDir(scenario string) (string, error) {
	scenario = strings.TrimSpace(scenario)
	if scenario == "" {
		return "", connect.NewError(connect.CodeInvalidArgument, errors.New("scenario is required"))
	}
	return filepath.Join(s.scenariosRoot, scenario), nil
}

// ListRuns enumerates runs for a scenario, newest-first.
func (s *Service) ListRuns(ctx context.Context, req *connect.Request[runspb.ListRunsRequest]) (*connect.Response[runspb.ListRunsResponse], error) {
	dir, err := s.scenarioDir(req.Msg.GetScenario())
	if err != nil {
		return nil, err
	}
	records, err := sharedruns.NewIndex(dir).List()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	statusFilter := strings.TrimSpace(req.Msg.GetStatus())
	limit := int(req.Msg.GetLimit())
	out := make([]*runspb.RunInfo, 0, len(records))
	for _, r := range records {
		if statusFilter != "" && r.Status != statusFilter {
			continue
		}
		out = append(out, toRunInfo(r))
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return connect.NewResponse(&runspb.ListRunsResponse{Runs: out}), nil
}

// GetRun returns a single run record.
func (s *Service) GetRun(ctx context.Context, req *connect.Request[runspb.GetRunRequest]) (*connect.Response[runspb.GetRunResponse], error) {
	dir, err := s.scenarioDir(req.Msg.GetScenario())
	if err != nil {
		return nil, err
	}
	rec, err := sharedruns.NewIndex(dir).Find(strings.TrimSpace(req.Msg.GetRunId()))
	if err != nil {
		return nil, mapRunError(err)
	}
	return connect.NewResponse(&runspb.GetRunResponse{Run: toRunInfo(rec)}), nil
}

// DeleteRun removes a run's artifacts and index entry.
func (s *Service) DeleteRun(ctx context.Context, req *connect.Request[runspb.DeleteRunRequest]) (*connect.Response[runspb.DeleteRunResponse], error) {
	dir, err := s.scenarioDir(req.Msg.GetScenario())
	if err != nil {
		return nil, err
	}
	if err := sharedruns.DeleteRun(dir, strings.TrimSpace(req.Msg.GetRunId()), req.Msg.GetForce()); err != nil {
		return nil, mapRunError(err)
	}
	return connect.NewResponse(&runspb.DeleteRunResponse{Deleted: true}), nil
}

// PinRun protects a run from retention GC.
func (s *Service) PinRun(ctx context.Context, req *connect.Request[runspb.PinRunRequest]) (*connect.Response[runspb.PinRunResponse], error) {
	dir, err := s.scenarioDir(req.Msg.GetScenario())
	if err != nil {
		return nil, err
	}
	pinnedBy := strings.TrimSpace(req.Msg.GetPinnedBy())
	if pinnedBy == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("pinned_by is required"))
	}
	idx := sharedruns.NewIndex(dir)
	runID := strings.TrimSpace(req.Msg.GetRunId())
	if err := idx.Pin(runID, sharedruns.PinRecord{
		PinnedBy: pinnedBy,
		PinnedAt: time.Now().UTC(),
		Reason:   strings.TrimSpace(req.Msg.GetReason()),
	}); err != nil {
		return nil, mapRunError(err)
	}
	rec, err := idx.Find(runID)
	if err != nil {
		return nil, mapRunError(err)
	}
	return connect.NewResponse(&runspb.PinRunResponse{Run: toRunInfo(rec)}), nil
}

// UnpinRun removes a consumer's pin.
func (s *Service) UnpinRun(ctx context.Context, req *connect.Request[runspb.UnpinRunRequest]) (*connect.Response[runspb.UnpinRunResponse], error) {
	dir, err := s.scenarioDir(req.Msg.GetScenario())
	if err != nil {
		return nil, err
	}
	idx := sharedruns.NewIndex(dir)
	runID := strings.TrimSpace(req.Msg.GetRunId())
	if err := idx.Unpin(runID, strings.TrimSpace(req.Msg.GetPinnedBy())); err != nil {
		return nil, mapRunError(err)
	}
	rec, err := idx.Find(runID)
	if err != nil {
		return nil, mapRunError(err)
	}
	return connect.NewResponse(&runspb.UnpinRunResponse{Run: toRunInfo(rec)}), nil
}

// CompareRuns classifies per-phase differences between two runs.
func (s *Service) CompareRuns(ctx context.Context, req *connect.Request[runspb.CompareRunsRequest]) (*connect.Response[runspb.CompareRunsResponse], error) {
	dir, err := s.scenarioDir(req.Msg.GetScenario())
	if err != nil {
		return nil, err
	}
	idx := sharedruns.NewIndex(dir)
	recA, err := idx.Find(strings.TrimSpace(req.Msg.GetRunIdA()))
	if err != nil {
		return nil, mapRunError(err)
	}
	recB, err := idx.Find(strings.TrimSpace(req.Msg.GetRunIdB()))
	if err != nil {
		return nil, mapRunError(err)
	}
	resp := comparePhases(recA, recB, strings.TrimSpace(req.Msg.GetPhase()))
	return connect.NewResponse(resp), nil
}

// FindRun returns the newest completed run matching the shape filter, or
// found=false when none matches. It is the reuse primitive git-control-tower
// queries before starting a comprehensive run: "is there already a clean-tree
// comprehensive+baseline run at this sha?" Matching is exact on every non-empty
// filter; status defaults to "passed"; require_clean excludes dirty-tree runs.
func (s *Service) FindRun(ctx context.Context, req *connect.Request[runspb.FindRunRequest]) (*connect.Response[runspb.FindRunResponse], error) {
	dir, err := s.scenarioDir(req.Msg.GetScenario())
	if err != nil {
		return nil, err
	}
	records, err := sharedruns.NewIndex(dir).List()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	wantStatus := strings.TrimSpace(req.Msg.GetStatus())
	if wantStatus == "" {
		wantStatus = sharedruns.StatusPassed
	}
	gitSha := strings.TrimSpace(req.Msg.GetGitSha())
	treeDigest := strings.TrimSpace(req.Msg.GetTreeDigest())
	preset := strings.TrimSpace(req.Msg.GetPreset())
	captureProfile := strings.TrimSpace(req.Msg.GetCaptureProfile())
	phaseSetDigest := strings.TrimSpace(req.Msg.GetPhaseSetDigest())
	if phaseSetDigest == "" && preset == phases.PresetComprehensive.String() {
		phaseSetDigest = phases.PhaseSetDigest(phases.DefaultPresets()[phases.PresetComprehensive.String()])
	}
	requireClean := req.Msg.GetRequireClean()

	// List() returns newest-first, so the first match is the newest.
	for _, r := range records {
		if r.Status != wantStatus {
			continue
		}
		if requireClean && r.GitDirty {
			continue
		}
		if gitSha != "" && r.GitSha != gitSha {
			continue
		}
		if treeDigest != "" && r.TreeDigest != treeDigest {
			continue
		}
		if preset != "" && r.Preset != preset {
			continue
		}
		if captureProfile != "" && r.CaptureProfile != captureProfile {
			continue
		}
		if phaseSetDigest != "" && r.PhaseSetDigest != phaseSetDigest {
			continue
		}
		return connect.NewResponse(&runspb.FindRunResponse{Found: true, Run: toRunInfo(r)}), nil
	}
	return connect.NewResponse(&runspb.FindRunResponse{Found: false}), nil
}

// GetPhaseArtifact returns the raw phase-results JSON for a run+phase.
func (s *Service) GetPhaseArtifact(ctx context.Context, req *connect.Request[runspb.GetPhaseArtifactRequest]) (*connect.Response[runspb.GetPhaseArtifactResponse], error) {
	dir, err := s.scenarioDir(req.Msg.GetScenario())
	if err != nil {
		return nil, err
	}
	phase := strings.TrimSpace(req.Msg.GetPhase())
	if phase == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("phase is required"))
	}
	path := sharedartifacts.PhaseResultsPath(dir, strings.TrimSpace(req.Msg.GetRunId()), phase+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("phase artifact not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&runspb.GetPhaseArtifactResponse{
		Content:     string(data),
		ContentType: "application/json",
	}), nil
}

// findingsArtifactDoc mirrors the on-disk findings.json shape (writer:
// orchestrator.writeFindingsArtifact) for the fields GetRunFindings surfaces.
// The proto standing/summary round-trip because they were written with
// encoding/json using their proto json tags.
type findingsArtifactDoc struct {
	Scenario    string `json:"scenario"`
	RunID       string `json:"runId"`
	Verdict     string `json:"verdict"`
	CompletedAt string `json:"completedAt"`
	Phases      []struct {
		Name             string                        `json:"name"`
		Status           string                        `json:"status"`
		FindingSource    string                        `json:"findingSource"`
		MaturityStanding *runspb.PhaseMaturityStanding `json:"maturityStanding"`
		FindingsSummary  *runspb.PhaseFindingsSummary  `json:"findingsSummary"`
	} `json:"phases"`
}

// GetRunFindings returns the per-phase maturity standing (Phase Capability
// Contract) persisted in the run's findings.json, so an agent can revisit where
// each capability stands without re-running the suite.
func (s *Service) GetRunFindings(ctx context.Context, req *connect.Request[runspb.GetRunFindingsRequest]) (*connect.Response[runspb.GetRunFindingsResponse], error) {
	dir, err := s.scenarioDir(req.Msg.GetScenario())
	if err != nil {
		return nil, err
	}
	runID := strings.TrimSpace(req.Msg.GetRunId())
	var path string
	if runID == "" || strings.EqualFold(runID, "latest") {
		path = sharedartifacts.LatestFindingsArtifactPath(dir)
	} else {
		path = sharedartifacts.RunFindingsArtifactPath(dir, runID)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("no findings artifact for run %q", runID))
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	var doc findingsArtifactDoc
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("parse findings artifact: %w", err))
	}
	phases := make([]*runspb.RunFindingsPhase, 0, len(doc.Phases))
	for _, p := range doc.Phases {
		phases = append(phases, &runspb.RunFindingsPhase{
			Name:             p.Name,
			Status:           p.Status,
			FindingSource:    p.FindingSource,
			MaturityStanding: p.MaturityStanding,
			FindingsSummary:  p.FindingsSummary,
		})
	}
	return connect.NewResponse(&runspb.GetRunFindingsResponse{
		Scenario:    doc.Scenario,
		RunId:       doc.RunID,
		Verdict:     doc.Verdict,
		CompletedAt: doc.CompletedAt,
		Phases:      phases,
	}), nil
}

// ListRunVideos enumerates the recorded workflow videos for a run. The binary
// content is served by the REST artifact route; this returns the relative-path
// handles that route consumes.
func (s *Service) ListRunVideos(ctx context.Context, req *connect.Request[runspb.ListRunVideosRequest]) (*connect.Response[runspb.ListRunVideosResponse], error) {
	dir, err := s.scenarioDir(req.Msg.GetScenario())
	if err != nil {
		return nil, err
	}
	runID := strings.TrimSpace(req.Msg.GetRunId())
	if runID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("run_id is required"))
	}
	videos, err := sharedartifacts.ListRunVideos(dir, runID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := make([]*runspb.RunVideo, 0, len(videos))
	for _, v := range videos {
		out = append(out, &runspb.RunVideo{
			Workflow:  v.Workflow,
			RelPath:   v.RelPath,
			SizeBytes: v.SizeBytes,
		})
	}
	return connect.NewResponse(&runspb.ListRunVideosResponse{Videos: out}), nil
}

// ListRunVisuals enumerates the per-page UI visual artifacts (screenshot +
// optional video) a run captured under the baseline capture profile. The binary
// content is served by the REST artifact route; this returns the structured
// page set + rel-path handles git-control-tower diffs at the metadata level.
func (s *Service) ListRunVisuals(ctx context.Context, req *connect.Request[runspb.ListRunVisualsRequest]) (*connect.Response[runspb.ListRunVisualsResponse], error) {
	dir, err := s.scenarioDir(req.Msg.GetScenario())
	if err != nil {
		return nil, err
	}
	runID := strings.TrimSpace(req.Msg.GetRunId())
	if runID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("run_id is required"))
	}
	visuals, err := sharedartifacts.ListRunVisuals(dir, runID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := make([]*runspb.RunVisual, 0, len(visuals))
	for _, v := range visuals {
		out = append(out, &runspb.RunVisual{
			Page:                v.Page,
			Label:               v.Label,
			ScreenshotRelPath:   v.ScreenshotRelPath,
			VideoRelPath:        v.VideoRelPath,
			ScreenshotSizeBytes: v.ScreenshotSizeBytes,
		})
	}
	return connect.NewResponse(&runspb.ListRunVisualsResponse{Visuals: out}), nil
}

// CompareRunVisuals delegates visual delta analysis to ui-health. Test Genie
// only enumerates run artifacts and supplies inline screenshot bytes; ui-health
// owns the pixel math and verdict taxonomy.
func (s *Service) CompareRunVisuals(ctx context.Context, req *connect.Request[runspb.CompareRunVisualsRequest]) (*connect.Response[runspb.CompareRunVisualsResponse], error) {
	dir, err := s.scenarioDir(req.Msg.GetScenario())
	if err != nil {
		return nil, err
	}
	baseRunID := strings.TrimSpace(req.Msg.GetBaseRunId())
	curRunID := strings.TrimSpace(req.Msg.GetCurrentRunId())
	if baseRunID == "" || curRunID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("base_run_id and current_run_id are required"))
	}

	baseVisuals, err := sharedartifacts.ListRunVisuals(dir, baseRunID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	curVisuals, err := sharedartifacts.ListRunVisuals(dir, curRunID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	comparer := s.visualHealth
	if comparer == nil {
		comparer = defaultVisualHealthComparer{}
	}
	visualResp, err := comparer.CompareArtifacts(ctx, &visualpb.CompareArtifactsRequest{
		Scenario:     req.Msg.GetScenario(),
		BaseRunId:    baseRunID,
		CurrentRunId: curRunID,
		Base:         s.visualCompareArtifacts(dir, baseRunID, baseVisuals),
		Current:      s.visualCompareArtifacts(dir, curRunID, curVisuals),
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("ui-health visual comparison unavailable: %w", err))
	}
	deltas := make([]*runspb.VisualDelta, 0, len(visualResp.GetDeltas()))
	for _, d := range visualResp.GetDeltas() {
		deltas = append(deltas, &runspb.VisualDelta{
			Page:            d.GetPage(),
			Label:           d.GetLabel(),
			Status:          d.GetStatus(),
			ChangedFraction: d.GetChangedFraction(),
		})
	}
	return connect.NewResponse(&runspb.CompareRunVisualsResponse{Deltas: deltas}), nil
}

func (s *Service) visualCompareArtifacts(dir, runID string, visuals []sharedartifacts.RunVisual) []*visualpb.CompareArtifact {
	out := make([]*visualpb.CompareArtifact, 0, len(visuals))
	for _, v := range visuals {
		var screenshot []byte
		if v.ScreenshotRelPath != "" {
			if b, err := readRunArtifact(dir, runID, v.ScreenshotRelPath); err == nil {
				screenshot = b
			}
		}
		out = append(out, &visualpb.CompareArtifact{
			Page:          v.Page,
			Label:         v.Label,
			ScreenshotPng: screenshot,
			ScreenshotRef: &visualpb.ArtifactRef{
				Scenario:  filepath.Base(dir),
				RunId:     runID,
				RelPath:   v.ScreenshotRelPath,
				MediaType: "image/png",
			},
		})
	}
	return out
}

// readRunArtifact resolves and reads a run-relative artifact's bytes.
func readRunArtifact(dir, runID, relPath string) ([]byte, error) {
	abs, err := sharedartifacts.ResolveRunArtifact(dir, runID, relPath)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(abs)
}

// ResolveArtifact maps a (scenario, runID, run-relative path) to an absolute
// filesystem path under the run's artifact root, rejecting traversal. Used by
// the REST binary artifact route to stream videos without exposing
// scenariosRoot. Returns os.ErrNotExist when the resolved file is missing.
func (s *Service) ResolveArtifact(scenario, runID, relPath string) (string, error) {
	dir, err := s.scenarioDir(scenario)
	if err != nil {
		return "", err
	}
	abs, err := sharedartifacts.ResolveRunArtifact(dir, strings.TrimSpace(runID), relPath)
	if err != nil {
		return "", err
	}
	if _, statErr := os.Stat(abs); statErr != nil {
		return "", statErr
	}
	return abs, nil
}

func mapRunError(err error) error {
	switch {
	case errors.Is(err, sharedruns.ErrRunNotFound):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, sharedruns.ErrRunPinned):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	default:
		return connect.NewError(connect.CodeInternal, err)
	}
}
