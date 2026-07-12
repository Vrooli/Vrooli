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
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
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
	if scenario == "." || scenario == ".." || strings.ContainsAny(scenario, `/\`) || filepath.Clean(scenario) != scenario {
		return "", connect.NewError(connect.CodeInvalidArgument, errors.New("scenario must be a scenario slug"))
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
	projectionA, err := loadRunProjection(idx, strings.TrimSpace(req.Msg.GetRunIdA()))
	if err != nil {
		return nil, mapRunError(err)
	}
	projectionB, err := loadRunProjection(idx, strings.TrimSpace(req.Msg.GetRunIdB()))
	if err != nil {
		return nil, mapRunError(err)
	}
	resp := comparePhases(projectionA, projectionB, strings.TrimSpace(req.Msg.GetPhase()))
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
		Name          string `json:"name"`
		Status        string `json:"status"`
		FindingSource string `json:"findingSource"`
		// MaturityStanding is decode-only historical evidence. It must remain
		// distinct from PhasePresentation: a retired standing cannot truthfully
		// be rendered as the canonical v1 provider presentation.
		MaturityStanding  *runspb.PhaseMaturityStanding `json:"maturityStanding"`
		PhasePresentation *commonv1.PhasePresentation   `json:"phasePresentation"`
		FindingsSummary   *runspb.PhaseFindingsSummary  `json:"findingsSummary"`
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
			Name:              p.Name,
			Status:            p.Status,
			FindingSource:     p.FindingSource,
			MaturityStanding:  p.MaturityStanding,
			PhasePresentation: p.PhasePresentation,
			FindingsSummary:   p.FindingsSummary,
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
	default:
		return connect.NewError(connect.CodeInternal, err)
	}
}
