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
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"connectrpc.com/connect"

	"github.com/vrooli/freshness-go/treedigest"
	"test-genie/internal/execution"
	"test-genie/internal/executionevidence"
	"test-genie/internal/orchestrator"
	"test-genie/internal/orchestrator/phases"
	"test-genie/internal/runmanager"
	"test-genie/internal/selfhealthsnapshots"
	sharedartifacts "test-genie/internal/shared/artifacts"
	sharedruns "test-genie/internal/shared/runs"
	"test-genie/internal/targetmodel"

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
	ledgerSource   ledgerSource
	costSource     execution.CostSource
	storedMetrics  func(context.Context, string) bool
	retentionStore sharedruns.DetailStore
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

const maxPhaseArtifactResponseBytes = 256 * 1024

// SetRetentionStore attaches compact execution history to the same lifecycle
// authority used by the explicit DeleteRun RPC.
func (s *Service) SetRetentionStore(store sharedruns.DetailStore) {
	if s != nil {
		s.retentionStore = store
	}
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

// SetCostSource wires the measured phase-cost read model.
func (s *Service) SetCostSource(src execution.CostSource) *Service {
	if s != nil {
		s.costSource = src
	}
	return s
}

// SetStoredMetricsProbe wires the durable execution-history check used by
// provider conformance. It prevents live response fields from being counted
// as adoption when persistence silently drops them.
func (s *Service) SetStoredMetricsProbe(probe func(context.Context, string) bool) *Service {
	if s != nil {
		s.storedMetrics = probe
	}
	return s
}

func (s *Service) GetCostReport(ctx context.Context, req *connect.Request[runspb.GetCostReportRequest]) (*connect.Response[runspb.GetCostReportResponse], error) {
	if s.costSource == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("cost report is unavailable"))
	}
	window := time.Duration(req.Msg.GetWindowSeconds()) * time.Second
	if window <= 0 {
		window = 7 * 24 * time.Hour
	}
	compareWindow := time.Duration(req.Msg.GetCompareWindowSeconds()) * time.Second
	now := time.Now().UTC()
	current, err := s.costSource.CostReport(ctx, req.Msg.GetScenario(), now.Add(-window), now)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	var previous []execution.CostSummary
	if compareWindow > 0 {
		previous, err = s.costSource.CostReport(ctx, req.Msg.GetScenario(), now.Add(-window-compareWindow), now.Add(-window))
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}
	prior := make(map[string]execution.CostSummary, len(previous))
	for _, p := range previous {
		prior[p.Scenario+"\x00"+p.Phase] = p
	}
	out := make([]*runspb.CostPhaseSummary, 0, len(current))
	for _, c := range current {
		row := &runspb.CostPhaseSummary{Scenario: c.Scenario, Phase: c.Phase, SampleCount: int32(c.SampleCount), PassingSampleCount: int32(c.PassingSampleCount), FailingSampleCount: int32(c.FailingSampleCount), ReliableSampleCount: int32(c.ReliableSampleCount), ExcludedSampleCount: int32(c.ExcludedSampleCount), TotalWallClockMs: c.TotalWallClockMs, MedianWallClockMs: c.MedianWallClockMs, P90WallClockMs: c.P90WallClockMs, PassingMedianWallClockMs: c.PassingMedianWallClockMs, PassingP90WallClockMs: c.PassingP90WallClockMs, FailingMedianWallClockMs: c.FailingMedianWallClockMs, FailingP90WallClockMs: c.FailingP90WallClockMs, TotalCpuUserMs: c.TotalCPUUserMs, MaxPeakRssBytes: c.MaxPeakRSSBytes, PredictionSampleCount: int32(c.PredictionSampleCount), PredictionErrorTotalMs: c.PredictionErrorTotalMs, PredictionMeanAbsoluteErrorMs: c.PredictionMeanAbsoluteErrorMs, PredictionMeanAbsoluteErrorPercent: c.PredictionMeanAbsoluteErrorPercent, CacheHitCount: int32(c.CacheHitCount), ExecutedSampleCount: int32(c.ExecutedSampleCount), CacheHitRatePercent: c.CacheHitRatePercent, CacheAuditCount: int32(c.CacheAuditCount), CacheAuditMismatchCount: int32(c.CacheAuditMismatchCount), CacheNoSavingCount: int32(c.CacheNoSavingCount), CacheAuditWallClockMs: c.CacheAuditWallClockMs, EstimatedGrossSavedWallClockMs: c.EstimatedGrossSavedWallClockMs, EstimatedNetSavedWallClockMs: c.EstimatedNetSavedWallClockMs}
		if p, ok := prior[c.Scenario+"\x00"+c.Phase]; ok {
			row.ChangeWallClockMs = c.TotalWallClockMs - p.TotalWallClockMs
			if p.TotalWallClockMs != 0 {
				row.ChangePercent = float64(row.ChangeWallClockMs) / float64(p.TotalWallClockMs) * 100
			}
		}
		out = append(out, row)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].TotalWallClockMs > out[j].TotalWallClockMs })
	return connect.NewResponse(&runspb.GetCostReportResponse{Phases: out, WindowSeconds: int64(window.Seconds()), CompareWindowSeconds: int64(compareWindow.Seconds())}), nil
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

// scenarioManifest is the marker that makes a directory under scenarios/ a
// scenario. It matches the `scenario` target marker in
// .vrooli/repo-contract.json; the two must stay in step.
const scenarioManifest = ".vrooli/service.json"

// scenarioDir resolves a run-query target to the directory owning its run
// artifacts. A kind:id expression goes through the target model; a bare slug is
// a scenario, and must carry a scenario manifest to be one.
//
// The manifest check is the point. Without it any well-formed word resolved to
// scenarios/<word>, so querying run history for a resource, a package or a
// top-level directory name — minio, maturity-go, docs, internal — produced a
// path that the read itself then created on disk.
func (s *Service) scenarioDir(scenario string) (string, error) {
	scenario = strings.TrimSpace(scenario)
	if scenario == "" {
		return "", connect.NewError(connect.CodeInvalidArgument, errors.New("scenario is required"))
	}
	if strings.Contains(scenario, ":") {
		target, err := targetmodel.Resolve(filepath.Dir(s.scenariosRoot), scenario)
		if err != nil {
			return "", connect.NewError(connect.CodeInvalidArgument, err)
		}
		artifactRoot, err := targetmodel.ArtifactRoot(filepath.Dir(s.scenariosRoot), target)
		if err != nil {
			return "", connect.NewError(connect.CodeFailedPrecondition, err)
		}
		return artifactRoot, nil
	}
	if scenario == "." || scenario == ".." || strings.ContainsAny(scenario, `/\`) || filepath.Clean(scenario) != scenario {
		return "", connect.NewError(connect.CodeInvalidArgument, errors.New("scenario must be a scenario slug or kind:id target"))
	}
	dir := filepath.Join(s.scenariosRoot, scenario)
	if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(scenarioManifest))); err != nil {
		// A well-formed name with no manifest is a missing scenario, not a
		// malformed request. NotFound keeps callers from reading it as a
		// scenario that merely has no runs yet.
		return "", connect.NewError(connect.CodeNotFound,
			fmt.Errorf("no scenario %q under %s (expected %s)", scenario, s.scenariosRoot, scenarioManifest))
	}
	return dir, nil
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
	leases := sharedruns.NewPinLeaseStore(dir)
	for _, r := range records {
		if statusFilter != "" && r.Status != statusFilter {
			continue
		}
		active, err := leases.ActiveForRun(r.RunID, time.Now())
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("read run leases: %w", err))
		}
		out = append(out, withLeasePins(toRunInfo(r), active))
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
	projection, err := loadRunProjection(sharedruns.NewIndex(dir), strings.TrimSpace(req.Msg.GetRunId()))
	if err != nil {
		return nil, mapRunError(err)
	}
	return connect.NewResponse(&runspb.GetRunResponse{
		Run:                           toTerminalRunInfo(projection.record, projection.result, projection.descriptors),
		TerminalSnapshotSchemaVersion: int32(projection.schemaVersion),
		DegradedReasons:               projection.degraded,
	}), nil
}

// DeleteRun removes a run's artifacts and index entry.
func (s *Service) DeleteRun(ctx context.Context, req *connect.Request[runspb.DeleteRunRequest]) (*connect.Response[runspb.DeleteRunResponse], error) {
	dir, err := s.scenarioDir(req.Msg.GetScenario())
	if err != nil {
		return nil, err
	}
	retention := sharedruns.NewRetentionService(dir, sharedruns.DefaultRetentionPolicy()).WithDetailStore(s.retentionStore)
	if err := retention.Delete(ctx, strings.TrimSpace(req.Msg.GetRunId()), req.Msg.GetForce()); err != nil {
		return nil, mapRunError(err)
	}
	return connect.NewResponse(&runspb.DeleteRunResponse{Deleted: true}), nil
}

// PinRun grants an expiring protection lease. The public RPC currently has no
// TTL field, so this compatibility surface grants the documented default
// rather than recreating the old indefinite index pin.
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
	if _, err := idx.Find(runID); err != nil {
		return nil, mapRunError(err)
	}
	lease, err := sharedruns.NewPinLeaseStore(dir).Grant(runID, pinnedBy, strings.TrimSpace(req.Msg.GetReason()), sharedruns.DefaultPinLeaseTTL, time.Now())
	if err != nil {
		return nil, mapRunError(err)
	}
	rec, err := idx.Find(runID)
	if err != nil {
		return nil, mapRunError(err)
	}
	return connect.NewResponse(&runspb.PinRunResponse{Run: withLeasePin(toRunInfo(rec), lease)}), nil
}

// UnpinRun revokes a consumer's protection lease.
func (s *Service) UnpinRun(ctx context.Context, req *connect.Request[runspb.UnpinRunRequest]) (*connect.Response[runspb.UnpinRunResponse], error) {
	dir, err := s.scenarioDir(req.Msg.GetScenario())
	if err != nil {
		return nil, err
	}
	idx := sharedruns.NewIndex(dir)
	runID := strings.TrimSpace(req.Msg.GetRunId())
	if err := sharedruns.NewPinLeaseStore(dir).Revoke(runID, strings.TrimSpace(req.Msg.GetPinnedBy())); err != nil {
		return nil, mapRunError(err)
	}
	rec, err := idx.Find(runID)
	if err != nil {
		return nil, mapRunError(err)
	}
	active, err := sharedruns.NewPinLeaseStore(dir).ActiveForRun(runID, time.Now())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("read run leases: %w", err))
	}
	return connect.NewResponse(&runspb.UnpinRunResponse{Run: withLeasePins(toRunInfo(rec), active)}), nil
}

// CompareRuns classifies per-phase differences between two runs.
func (s *Service) CompareRuns(ctx context.Context, req *connect.Request[runspb.CompareRunsRequest]) (*connect.Response[runspb.CompareRunsResponse], error) {
	dir, err := s.scenarioDir(req.Msg.GetScenario())
	if err != nil {
		return nil, err
	}
	idx := sharedruns.NewIndex(dir)
	projectionA, err := loadRunProjection(idx, strings.TrimSpace(req.Msg.GetRunIdA()))
	if err != nil {
		return nil, mapRunError(err)
	}
	projectionB, err := loadRunProjection(idx, strings.TrimSpace(req.Msg.GetRunIdB()))
	if err != nil {
		return nil, mapRunError(err)
	}
	phaseFilter := strings.TrimSpace(req.Msg.GetPhase())
	if phaseFilter == "" {
		if resp, comparable := compareStablePreflightFailures(projectionA, projectionB); comparable {
			return connect.NewResponse(resp), nil
		}
	}
	resp := comparePhases(projectionA, projectionB, phaseFilter)
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
	requireGateQuality := req.Msg.GetRequireGateQuality()
	matchCurrentSource := req.Msg.GetMatchCurrentSource()
	currentConfiguration := ""
	if matchCurrentSource {
		sourceDir := dir
		if targetExpression := strings.TrimSpace(req.Msg.GetScenario()); strings.Contains(targetExpression, ":") {
			target, resolveErr := targetmodel.Resolve(filepath.Dir(s.scenariosRoot), targetExpression)
			if resolveErr != nil {
				return nil, connect.NewError(connect.CodeInvalidArgument, resolveErr)
			}
			sourceDir = target.Path
		}
		if treeDigest, err = treedigest.Compute(sourceDir); err != nil {
			return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("compute current source fingerprint: %w", err))
		}
		if s.planner == nil {
			return connect.NewResponse(&runspb.FindRunResponse{Found: false}), nil
		}
		previewRequest := orchestrator.SuiteExecutionRequest{
			ScenarioName: req.Msg.GetScenario(), Preset: preset, CaptureProfile: captureProfile,
		}
		if strings.Contains(strings.TrimSpace(req.Msg.GetScenario()), ":") {
			previewRequest.Target = req.Msg.GetScenario()
		}
		preview, err := s.planner.Preview(ctx, previewRequest)
		if err != nil || preview == nil {
			return connect.NewResponse(&runspb.FindRunResponse{Found: false}), nil
		}
		currentConfiguration = preview.ConfigurationFingerprint
		phaseSetDigest = preview.PhaseSetDigest
	}

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
		projection, projectionErr := loadRunProjection(sharedruns.NewIndex(dir), r.RunID)
		if requireGateQuality && (projectionErr != nil || !toTerminalRunInfo(projection.record, projection.result, projection.descriptors).GetGateQuality()) {
			continue
		}
		if matchCurrentSource && (projectionErr != nil || projection.result == nil || !projection.result.SourceStable || projection.result.ConfigurationFingerprint != currentConfiguration) {
			continue
		}
		active, err := sharedruns.NewPinLeaseStore(dir).ActiveForRun(r.RunID, time.Now())
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("read run leases: %w", err))
		}
		return connect.NewResponse(&runspb.FindRunResponse{Found: true, Run: withLeasePins(toTerminalRunInfo(projection.record, projection.result, projection.descriptors), active)}), nil
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
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("phase artifact not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if info.Size() > maxPhaseArtifactResponseBytes {
		return nil, connect.NewError(connect.CodeResourceExhausted, fmt.Errorf("phase artifact exceeds %d-byte response limit; use the run artifact endpoint", maxPhaseArtifactResponseBytes))
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&runspb.GetPhaseArtifactResponse{
		Content:     string(data),
		ContentType: "application/json",
	}), nil
}

// ListRunArtifacts projects the verified, run-owned artifact catalog without
// exposing its private storage locators. Runs predating catalogs use a
// read-only discovery projection with explicit legacy provenance.
func (s *Service) ListRunArtifacts(ctx context.Context, req *connect.Request[runspb.ListRunArtifactsRequest]) (*connect.Response[runspb.ListRunArtifactsResponse], error) {
	dir, err := s.scenarioDir(req.Msg.GetScenario())
	if err != nil {
		return nil, err
	}
	runID := strings.TrimSpace(req.Msg.GetRunId())
	if runID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("run_id is required"))
	}
	if _, err := sharedruns.NewIndex(dir).Find(runID); err != nil {
		return nil, mapRunError(err)
	}
	declarations := artifactPhaseDeclarations(dir, runID)
	catalog, err := sharedartifacts.ReadArtifactCatalog(dir, runID)
	if errors.Is(err, sharedartifacts.ErrArtifactCatalogNotFound) {
		catalog, err = sharedartifacts.DiscoverArtifactCatalog(dir, runID, declarations, time.Now().UTC(), true)
	}
	if err != nil {
		return nil, mapArtifactError(err)
	}
	kinds := make(map[string]struct{}, len(req.Msg.GetKinds()))
	for _, kind := range req.Msg.GetKinds() {
		if normalized := strings.ToLower(strings.TrimSpace(kind)); normalized != "" {
			kinds[normalized] = struct{}{}
		}
	}
	producingPhase := strings.TrimSpace(req.Msg.GetProducingPhase())
	out := make([]*runspb.ArtifactRef, 0, len(catalog.Artifacts))
	for _, artifact := range catalog.Artifacts {
		if len(kinds) > 0 {
			if _, ok := kinds[artifact.Kind]; !ok {
				continue
			}
		}
		if producingPhase != "" && artifact.ProducingPhase != producingPhase {
			continue
		}
		out = append(out, toArtifactRef(req.Msg.GetScenario(), runID, artifact))
	}
	response := &runspb.ListRunArtifactsResponse{
		SchemaVersion:    int32(catalog.SchemaVersion),
		Digest:           catalog.Digest,
		Artifacts:        out,
		LegacyDiscovered: catalog.LegacyDiscovered,
	}
	if catalog.LegacyDiscovered {
		response.DegradedReasons = []string{"run predates persisted artifact catalogs; evidence was discovered read-only"}
	}
	return connect.NewResponse(response), nil
}

// GetRunArtifact returns safe metadata for one artifact and verifies that its
// bytes still resolve to a regular file inside this run's allowed roots.
func (s *Service) GetRunArtifact(ctx context.Context, req *connect.Request[runspb.GetRunArtifactRequest]) (*connect.Response[runspb.GetRunArtifactResponse], error) {
	dir, err := s.scenarioDir(req.Msg.GetScenario())
	if err != nil {
		return nil, err
	}
	runID := strings.TrimSpace(req.Msg.GetRunId())
	artifactID := strings.TrimSpace(req.Msg.GetArtifactId())
	if runID == "" || artifactID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("run_id and artifact_id are required"))
	}
	if _, err := sharedruns.NewIndex(dir).Find(runID); err != nil {
		return nil, mapRunError(err)
	}
	artifact, _, err := sharedartifacts.ResolveCatalogArtifact(dir, runID, artifactID, artifactPhaseDeclarations(dir, runID))
	if err != nil {
		return nil, mapArtifactError(err)
	}
	return connect.NewResponse(&runspb.GetRunArtifactResponse{
		Artifact:         toArtifactRef(req.Msg.GetScenario(), runID, artifact),
		LegacyDiscovered: artifact.Provenance == sharedartifacts.ArtifactProvenanceLegacy,
	}), nil
}

func artifactPhaseDeclarations(scenarioDir, runID string) []sharedartifacts.ArtifactPhaseDeclaration {
	snapshot, err := sharedruns.ReadDescriptorSnapshot(scenarioDir, runID)
	if err != nil {
		return nil
	}
	out := make([]sharedartifacts.ArtifactPhaseDeclaration, 0, len(snapshot.Phases))
	for _, descriptor := range snapshot.Phases {
		out = append(out, sharedartifacts.ArtifactPhaseDeclaration{
			Phase: descriptor.Phase, EvidenceKinds: append([]string(nil), descriptor.EvidenceKinds...),
		})
	}
	return out
}

func toArtifactRef(scenario, runID string, artifact sharedartifacts.ArtifactRef) *runspb.ArtifactRef {
	relationships := make([]*runspb.ArtifactRelationship, 0, len(artifact.Relationships))
	for _, relationship := range artifact.Relationships {
		relationships = append(relationships, &runspb.ArtifactRelationship{
			Type: relationship.Type, TargetArtifactId: relationship.TargetArtifactID,
		})
	}
	var comparison *runspb.ArtifactComparison
	if artifact.Comparison != nil {
		comparison = &runspb.ArtifactComparison{
			Semantics: artifact.Comparison.Semantics,
			Analyzer:  artifact.Comparison.Analyzer,
			Metadata:  cloneStringMap(artifact.Comparison.Metadata),
		}
	}
	provenance := runspb.ArtifactProvenance_ARTIFACT_PROVENANCE_CATALOG
	if artifact.Provenance == sharedartifacts.ArtifactProvenanceLegacy {
		provenance = runspb.ArtifactProvenance_ARTIFACT_PROVENANCE_LEGACY_DISCOVERY
	}
	return &runspb.ArtifactRef{
		Id:               artifact.ID,
		Kind:             artifact.Kind,
		MediaType:        artifact.MediaType,
		Label:            artifact.Label,
		ProducingPhase:   artifact.ProducingPhase,
		SizeBytes:        artifact.SizeBytes,
		CreatedAt:        artifact.CreatedAt,
		AccessCapability: runspb.ArtifactAccessCapability_ARTIFACT_ACCESS_CAPABILITY_STREAM,
		AccessPath: "/api/v1/scenarios/" + url.PathEscape(strings.TrimSpace(scenario)) + "/runs/" +
			url.PathEscape(runID) + "/artifacts/" + url.PathEscape(artifact.ID),
		Metadata:      cloneStringMap(artifact.Metadata),
		Relationships: relationships,
		Comparison:    comparison,
		Provenance:    provenance,
	}
}

func cloneStringMap(input map[string]string) map[string]string {
	if len(input) == 0 {
		return nil
	}
	out := make(map[string]string, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

// GetRunFindings returns the bounded phase-summary projection from the evidence
// manifest. It intentionally does not open findings.json: that artifact owns
// detailed findings and is accessed only through an explicit artifact route.
func (s *Service) GetRunFindings(ctx context.Context, req *connect.Request[runspb.GetRunFindingsRequest]) (*connect.Response[runspb.GetRunFindingsResponse], error) {
	dir, err := s.scenarioDir(req.Msg.GetScenario())
	if err != nil {
		return nil, err
	}
	runID := strings.TrimSpace(req.Msg.GetRunId())
	if runID == "" || strings.EqualFold(runID, "latest") {
		latest, err := latestRunID(dir)
		if err != nil {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		runID = latest
	}
	manifest, err := executionevidence.ReadManifest(sharedartifacts.RunDir(dir, runID))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			projection, projectionErr := loadRunProjection(sharedruns.NewIndex(dir), runID)
			if projectionErr == nil && projection.record.Status == sharedruns.StatusFailed && len(projection.record.Phases) == 0 {
				return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("run %q failed before phase execution, so it has no findings manifest; inspect the terminal error with test-genie runs status --scenario %s %s", runID, req.Msg.GetScenario(), runID))
			}
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("no evidence manifest for run %q", runID))
		}
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("read evidence manifest: %w", err))
	}
	phases := make([]*runspb.RunFindingsPhase, 0, len(manifest.Phases))
	for _, p := range manifest.Phases {
		phases = append(phases, &runspb.RunFindingsPhase{
			Name:              p.Name,
			Status:            p.Status,
			FindingSource:     p.FindingSource,
			PhasePresentation: p.PhasePresentation,
			FindingsSummary:   p.FindingsSummary,
		})
	}
	return connect.NewResponse(&runspb.GetRunFindingsResponse{
		Scenario:    manifest.Scenario,
		RunId:       manifest.RunID,
		Verdict:     manifest.Verdict,
		CompletedAt: manifest.CreatedAt.UTC().Format(time.RFC3339),
		Phases:      phases,
	}), nil
}

// latestRunID reads the lightweight latest manifest. It intentionally does not
// fall back to a copied findings document: detailed findings are single-sourced
// under the immutable run directory.
func latestRunID(scenarioDir string) (string, error) {
	data, err := os.ReadFile(sharedartifacts.LatestManifestPath(scenarioDir))
	if err != nil {
		return "", fmt.Errorf("no latest run manifest: %w", err)
	}
	var manifest struct {
		RunID string `json:"run_id"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return "", fmt.Errorf("parse latest run manifest: %w", err)
	}
	if strings.TrimSpace(manifest.RunID) == "" {
		return "", errors.New("latest run manifest has no run id")
	}
	return strings.TrimSpace(manifest.RunID), nil
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
	baseArtifactIDs, err := screenshotArtifactIDsByPage(dir, baseRunID)
	if err != nil {
		return nil, mapArtifactError(err)
	}
	curArtifactIDs, err := screenshotArtifactIDsByPage(dir, curRunID)
	if err != nil {
		return nil, mapArtifactError(err)
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
			Page:              d.GetPage(),
			Label:             d.GetLabel(),
			Status:            d.GetStatus(),
			ChangedFraction:   d.GetChangedFraction(),
			BaseArtifactId:    baseArtifactIDs[d.GetPage()],
			CurrentArtifactId: curArtifactIDs[d.GetPage()],
		})
	}
	return connect.NewResponse(&runspb.CompareRunVisualsResponse{Deltas: deltas}), nil
}

func screenshotArtifactIDsByPage(scenarioDir, runID string) (map[string]string, error) {
	catalog, err := sharedartifacts.ReadArtifactCatalog(scenarioDir, runID)
	if errors.Is(err, sharedartifacts.ErrArtifactCatalogNotFound) {
		catalog, err = sharedartifacts.DiscoverArtifactCatalog(
			scenarioDir, runID, artifactPhaseDeclarations(scenarioDir, runID), time.Now().UTC(), true,
		)
	}
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	for _, artifact := range catalog.Artifacts {
		if artifact.Kind == sharedartifacts.ArtifactKindScreenshot && artifact.Metadata["page"] != "" {
			out[artifact.Metadata["page"]] = artifact.ID
		}
	}
	return out, nil
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

// ResolveArtifactByID resolves the verified catalog entry used by the opaque
// REST byte route. The returned metadata is safe to use for content headers.
func (s *Service) ResolveArtifactByID(scenario, runID, artifactID string) (sharedartifacts.ArtifactRef, string, error) {
	dir, err := s.scenarioDir(scenario)
	if err != nil {
		return sharedartifacts.ArtifactRef{}, "", err
	}
	runID = strings.TrimSpace(runID)
	if _, err := sharedruns.NewIndex(dir).Find(runID); err != nil {
		return sharedartifacts.ArtifactRef{}, "", err
	}
	return sharedartifacts.ResolveCatalogArtifact(
		dir, runID, strings.TrimSpace(artifactID), artifactPhaseDeclarations(dir, runID),
	)
}

func mapArtifactError(err error) error {
	switch {
	case errors.Is(err, sharedartifacts.ErrArtifactNotFound), errors.Is(err, os.ErrNotExist):
		return connect.NewError(connect.CodeNotFound, errors.New("artifact not found"))
	case errors.Is(err, sharedartifacts.ErrUnsafeArtifact):
		return connect.NewError(connect.CodePermissionDenied, errors.New("artifact reference is unsafe"))
	case errors.Is(err, sharedartifacts.ErrInvalidArtifactCatalog), errors.Is(err, sharedartifacts.ErrUnsupportedArtifactCatalogVersion):
		return connect.NewError(connect.CodeFailedPrecondition, errors.New("artifact catalog is unavailable or invalid"))
	default:
		return connect.NewError(connect.CodeInternal, err)
	}
}

func mapRunError(err error) error {
	switch {
	case errors.Is(err, sharedruns.ErrRunNotFound):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, sharedruns.ErrRunPinned):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	case errors.Is(err, runmanager.ErrStaleCursor):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	default:
		return connect.NewError(connect.CodeInternal, err)
	}
}
