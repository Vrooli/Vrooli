package catalog

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"react-component-library/internal/capabilities"
	"react-component-library/internal/catalogcoverage"
	"react-component-library/internal/catalogexperience"
	"react-component-library/internal/components"
	componenttests "react-component-library/internal/componenttests"
	"react-component-library/internal/gates"
	"react-component-library/internal/module"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	catalogv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/catalog"
	catalogconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/catalog/catalog_v1connect"
)

type handler struct {
	repoRoot        string
	evidence        *catalogcoverage.EvidenceStore
	reports         reportCache
	quarantineMu    sync.RWMutex
	quarantined     map[string]bool
	quarantineKnown bool
	captureRunner   *componenttests.Runner
}

// Module exposes the same live coverage projection used by the component-test
// phase. No CLI-only computation or second catalog join is permitted.
func Module(repoRoot string, dbs ...*sql.DB) module.Module {
	var evidence *catalogcoverage.EvidenceStore
	if len(dbs) > 0 && dbs[0] != nil {
		evidence = catalogcoverage.NewEvidenceStore(dbs[0])
	}
	h := &handler{repoRoot: repoRoot, evidence: evidence, quarantined: map[string]bool{}}
	return h.module()
}

// ModuleWithCapture wires the catalog evidence endpoint to the same
// version-pinned BAS-backed runner used by component tests. Keeping the
// runner injectable makes the catalog projection testable without launching
// a browser, while production receives the real BAS executor.
func ModuleWithCapture(repoRoot string, db *sql.DB, assets components.Service, executor componenttests.StoryExecutor) module.Module {
	var evidence *catalogcoverage.EvidenceStore
	if db != nil {
		evidence = catalogcoverage.NewEvidenceStore(db)
	}
	h := &handler{
		repoRoot:      repoRoot,
		evidence:      evidence,
		quarantined:   map[string]bool{},
		captureRunner: &componenttests.Runner{Assets: assets, Stories: assets, Executor: executor},
	}
	return h.module()
}

func (h *handler) module() module.Module {
	// Warm the coverage report in the background at startup. The first
	// computation costs ~45s because it runs the full gate suite including the
	// toolchain-spawning `types` runner; paying that on a user's first page
	// view is what made the coverage page appear broken. Detached from startup
	// so it never delays the health check — until it lands, a cold request
	// still computes synchronously and is correct, just slow.
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		_, _ = h.report(ctx)
	}()
	path, service := catalogconnect.NewCatalogServiceHandler(h)
	return module.Module{
		Name:      "catalog",
		Mount:     func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: service}) },
		Endpoints: Endpoints,
	}
}

var Endpoints = []module.EndpointDescriptor{
	{ID: "catalog_coverage", Path: catalogconnect.CatalogServiceGetCoverageProcedure, Method: "POST", Summary: "Report achieved catalog maturity", Category: "catalog"},
	{ID: "catalog_next", Path: catalogconnect.CatalogServiceListNextWorkProcedure, Method: "POST", Summary: "Rank catalog next work", Category: "catalog"},
	{ID: "catalog_gate", Path: catalogconnect.CatalogServiceRunGateProcedure, Method: "POST", Summary: "Run a declarative catalog gate", Category: "catalog"},
	{ID: "catalog_graph", Path: catalogconnect.CatalogServiceGetAssetRelationshipsProcedure, Method: "POST", Summary: "Read catalog asset relationships", Category: "catalog"},
	{ID: "catalog_structure", Path: catalogconnect.CatalogServiceGetCatalogStructureProcedure, Method: "POST", Summary: "Read catalog structure", Category: "catalog"},
	{ID: "catalog_reconcile", Path: catalogconnect.CatalogServiceReconcileGraphProcedure, Method: "POST", Summary: "Reconcile catalog dependency graphs", Category: "catalog"},
	{ID: "catalog_ports", Path: catalogconnect.CatalogServiceGetAssetPortContractProcedure, Method: "POST", Summary: "Read asset host obligations", Category: "catalog"},
	{ID: "catalog_score_history", Path: catalogconnect.CatalogServiceGetScoreHistoryProcedure, Method: "POST", Summary: "Read carried-forward asset score history", Category: "catalog"},
	{ID: "catalog_health", Path: catalogconnect.CatalogServiceGetHealthOverviewProcedure, Method: "POST", Summary: "Read server-computed catalog health", Category: "catalog"},
	{ID: "catalog_capture_evidence", Path: catalogconnect.CatalogServiceCaptureEvidenceProcedure, Method: "POST", Summary: "Capture declared visual evidence", Category: "catalog"},
}

// report serves the coverage projection through the revision-keyed cache. The
// underlying computation executes every gate runner including the toolchain-
// spawning `types` gate, so it must not run once per request; see
// report_cache.go for the measurement that motivated this.
func (h *handler) report(ctx context.Context) (*catalogcoverage.Report, error) {
	return h.reports.get(ctx, h.repoRoot, h.computeReport)
}

func (h *handler) computeReport(ctx context.Context) (*catalogcoverage.Report, error) {
	assets, err := catalogcoverage.LoadCatalog(filepath.Join(h.repoRoot, "scenarios", "react-component-library", "catalog"))
	if err != nil {
		return nil, err
	}
	impls, err := catalogcoverage.LoadImplementations(filepath.Join(h.repoRoot, "scenarios", "react-component-library", "library"))
	if err != nil {
		return nil, err
	}
	definitions, err := catalogcoverage.LoadGateDefinitions(filepath.Join(h.repoRoot, "scenarios", "react-component-library", "catalog", "config.json"))
	if err != nil {
		return nil, err
	}
	if err := h.ensureQuarantines(definitions); err != nil {
		return nil, err
	}
	evidence, err := catalogcoverage.MergeExperienceEvidence(ctx, h.repoRoot, h.evidence, catalogexperience.Fetcher(h.repoRoot))
	if err != nil {
		return nil, err
	}
	return func() *catalogcoverage.Report {
		r := catalogcoverage.ComputeWithEvidence(assets, impls, evidence, definitions)
		declarations := make(map[string][]string, len(assets))
		for _, asset := range assets {
			declarations[asset.ID] = append([]string(nil), asset.Capabilities...)
		}
		r.CapabilityReport = capabilities.ReconcileDeclared(ctx, declarations)
		if mismatches, reconcileErr := catalogcoverage.ReconcileKinds(h.repoRoot, assets, impls); reconcileErr == nil {
			r.KindMismatches = mismatches
		}
		return &r
	}(), nil
}

func (h *handler) GetCoverage(ctx context.Context, _ *connect.Request[catalogv1.GetCoverageRequest]) (*connect.Response[catalogv1.GetCoverageResponse], error) {
	report, err := h.report(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("compute catalog coverage: %w", err))
	}
	return connect.NewResponse(&catalogv1.GetCoverageResponse{Report: toProto(report)}), nil
}

func (h *handler) ListNextWork(ctx context.Context, req *connect.Request[catalogv1.ListNextWorkRequest]) (*connect.Response[catalogv1.ListNextWorkResponse], error) {
	report, err := h.report(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("compute catalog next work: %w", err))
	}
	lane := strings.ToLower(strings.TrimSpace(req.Msg.GetLane()))
	promote, build := catalogcoverage.NextWorkLanes(*report, int(req.Msg.GetLimit()))
	rows := promote
	if lane == "build" {
		rows = build
	} else if lane == "all" {
		rows = append(append([]catalogcoverage.Row{}, promote...), build...)
	} else {
		lane = "promote"
	}
	out := make([]*catalogv1.CoverageRow, 0, len(rows))
	for i := range rows {
		out = append(out, rowProto(rows[i]))
	}
	return connect.NewResponse(&catalogv1.ListNextWorkResponse{Rows: out, Maturity: maturityProto(report), Lane: lane, Promote: rowsProto(promote), Build: rowsProto(build)}), nil
}

func (h *handler) RunGate(ctx context.Context, req *connect.Request[catalogv1.RunGateRequest]) (*connect.Response[catalogv1.RunGateResponse], error) {
	gate := strings.TrimSpace(req.Msg.GetGate())
	definitions, definitionErr := catalogcoverage.LoadGateDefinitions(filepath.Join(h.repoRoot, "scenarios", "react-component-library", "catalog", "config.json"))
	if definitionErr != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("load catalog gate definitions: %w", definitionErr))
	}
	if req.Msg.GetAll() {
		aggregate := &catalogv1.RunGateResponse{Gate: "all"}
		for _, definition := range definitions {
			result, runErr := h.RunGate(ctx, connect.NewRequest(&catalogv1.RunGateRequest{Gate: definition.ID, CalibrationOnly: req.Msg.GetCalibrationOnly()}))
			if runErr != nil {
				return nil, runErr
			}
			aggregate.InspectedFiles += result.Msg.InspectedFiles
			aggregate.Findings = append(aggregate.Findings, result.Msg.Findings...)
			aggregate.RunnerErrors = append(aggregate.RunnerErrors, result.Msg.RunnerErrors...)
			aggregate.EvidenceRowsWritten += result.Msg.EvidenceRowsWritten
			aggregate.Calibration = append(aggregate.Calibration, result.Msg.Calibration...)
			aggregate.NonDiscriminating = aggregate.NonDiscriminating || result.Msg.NonDiscriminating
			if len(result.Msg.SurfaceVerdictCounts) > 0 {
				if aggregate.SurfaceVerdictCounts == nil {
					aggregate.SurfaceVerdictCounts = map[string]int32{}
				}
				for verdict, count := range result.Msg.SurfaceVerdictCounts {
					aggregate.SurfaceVerdictCounts[verdict] += count
				}
			}
		}
		return connect.NewResponse(aggregate), nil
	}
	knownGate := false
	for _, definition := range definitions {
		if definition.ID == gate {
			knownGate = true
			break
		}
	}
	if !knownGate {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("unknown catalog gate %q", gate))
	}
	definition := catalogcoverage.GateDefinition{}
	for _, candidate := range definitions {
		if candidate.ID == gate {
			definition = candidate
			break
		}
	}
	runner := h.gateRunner(gate)
	calibration, calibrationErr := gates.Calibrate(h.repoRoot, gate, runner)
	if calibrationErr != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("calibrate catalog gate %q: %w", gate, calibrationErr))
	}
	calibrationResults := make([]*catalogv1.CalibrationResult, 0, len(calibration.Results))
	for _, item := range calibration.Results {
		calibrationResults = append(calibrationResults, &catalogv1.CalibrationResult{Gate: item.Gate, Fixture: item.Fixture, RequiredFailureCode: item.RequiredFailureCode, ObservedFailureCode: item.ObservedFailureCode, Status: item.Status, Message: item.Message})
	}
	nonDiscriminating := definition.Blocking && calibration.NonDiscriminating
	h.quarantineMu.Lock()
	if nonDiscriminating {
		h.quarantined[gate] = true
	} else {
		delete(h.quarantined, gate)
	}
	h.quarantineKnown = true
	h.quarantineMu.Unlock()
	response := &catalogv1.RunGateResponse{Gate: gate, Calibration: calibrationResults, NonDiscriminating: nonDiscriminating}
	if req.Msg.GetCalibrationOnly() {
		return connect.NewResponse(response), nil
	}
	var (
		result gates.Result
		err    error
	)
	if runner == nil {
		result, err = gates.UnmeasuredGate(h.repoRoot)
	} else {
		result, err = runner(h.repoRoot)
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("run catalog gate %q: %w", gate, err))
	}
	result = gates.NormalizeResult(h.repoRoot, result)
	if nonDiscriminating {
		// Calibration failure is a quarantine state. Preserve no corpus finding
		// as evidence: a gate that cannot prove it detects its own planted defect
		// is unmeasured, never failed or passed.
		result, err = gates.UnmeasuredGate(h.repoRoot)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("quarantine catalog gate %q: %w", gate, err))
		}
	}
	evidenceRowsWritten := 0
	if h.evidence != nil {
		for _, definition := range definitions {
			if definition.ID != gate {
				continue
			}
			rows, evidenceErr := catalogcoverage.EvidenceFromResult(ctx, h.repoRoot, definition, result, result.InspectedAssets)
			if evidenceErr != nil {
				return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("derive evidence for gate %q: %w", gate, evidenceErr))
			}
			if evidenceErr = h.evidence.Save(ctx, rows); evidenceErr != nil {
				return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("persist evidence for gate %q: %w", gate, evidenceErr))
			}
			h.reports.invalidate()
			evidenceRowsWritten = len(rows)
			break
		}
	}
	// Severity is the gate's declared blocking flag, not a constant. Reporting
	// every finding as "error" made the non-blocking gates (graph-reconciled,
	// forced-colors, documentation, migration) indistinguishable from the
	// blocking ones, so a reader had no way to tell reported drift from a
	// release-stopping defect.
	severity := "error"
	if definitionErr == nil {
		for _, definition := range definitions {
			if definition.ID == gate && !definition.Blocking {
				severity = "warning"
				break
			}
		}
	}
	response.InspectedFiles = int32(result.Inspected)
	if len(result.SurfaceCounts) > 0 {
		response.SurfaceVerdictCounts = make(map[string]int32, len(result.SurfaceCounts))
		for verdict, count := range result.SurfaceCounts {
			response.SurfaceVerdictCounts[verdict] = int32(count)
		}
	}
	if len(result.CompositionScores) > 0 {
		response.CompositionScores = make(map[string]float64, len(result.CompositionScores))
		for assetID, score := range result.CompositionScores {
			response.CompositionScores[assetID] = score
		}
		response.CompositionMedian = result.CompositionMedian
		response.BespokeEscapeCount = int32(len(result.BespokeEscapes))
		for _, escape := range result.BespokeEscapes {
			response.CompositionEscapes = append(response.CompositionEscapes, &catalogv1.CompositionEscape{AssetId: escape.AssetID, Reason: escape.Reason})
		}
	}
	response.Findings = make([]*catalogv1.GateFinding, 0, len(result.Findings)+1)
	response.RunnerErrors = make([]*catalogv1.GateFinding, 0, len(result.RunnerError))
	response.EvidenceRowsWritten = int32(evidenceRowsWritten)
	if nonDiscriminating {
		response.Findings = append(response.Findings, &catalogv1.GateFinding{Code: "catalog.gate_non_discriminating", Message: "gate calibration passed without detecting its planted-error fixture; corpus verdict was quarantined as unmeasured", Severity: "error", Remediation: "Repair the gate runner or its calibration fixture, then rerun catalog gates --calibration-only. A green corpus result is not evidence until the named fixture fails.", DocsRef: "docs/internal/TESTING.md"})
	}
	for _, finding := range result.Findings {
		response.Findings = append(response.Findings, &catalogv1.GateFinding{
			Code:        finding.Code,
			Message:     finding.Message,
			AssetId:     finding.AssetID,
			Severity:    severity,
			File:        finding.File,
			Line:        int32(finding.Line),
			Remediation: finding.Remediation,
			DocsRef:     finding.DocsRef,
		})
	}
	for _, finding := range result.RunnerError {
		response.RunnerErrors = append(response.RunnerErrors, &catalogv1.GateFinding{Code: finding.Code, Message: finding.Message, AssetId: finding.AssetID, Severity: "error", File: finding.File, Line: int32(finding.Line), Remediation: finding.Remediation, DocsRef: finding.DocsRef})
	}
	return connect.NewResponse(response), nil
}

func (h *handler) gateRunner(gate string) gates.GateRunner {
	return gates.GateRunnerFor(gate)
}

func (h *handler) ensureQuarantines(definitions []catalogcoverage.GateDefinition) error {
	h.quarantineMu.Lock()
	defer h.quarantineMu.Unlock()
	if h.quarantineKnown {
		return nil
	}
	for _, definition := range definitions {
		if !definition.Blocking {
			continue
		}
		calibration, err := gates.Calibrate(h.repoRoot, definition.ID, h.gateRunner(definition.ID))
		if err != nil {
			return fmt.Errorf("calibrate quarantine state for %q: %w", definition.ID, err)
		}
		if calibration.NonDiscriminating {
			h.quarantined[definition.ID] = true
		}
	}
	h.quarantineKnown = true
	return nil
}

func (h *handler) GetScoreHistory(ctx context.Context, req *connect.Request[catalogv1.GetScoreHistoryRequest]) (*connect.Response[catalogv1.GetScoreHistoryResponse], error) {
	if h.evidence == nil {
		return connect.NewResponse(&catalogv1.GetScoreHistoryResponse{}), nil
	}
	history, err := h.evidence.ScoreHistory(ctx, h.repoRoot, req.Msg.GetSince())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("read catalog score history: %w", err))
	}
	out := make([]*catalogv1.ScoreHistoryPoint, 0, len(history))
	for _, point := range history {
		out = append(out, scoreHistoryProto(point))
	}
	return connect.NewResponse(&catalogv1.GetScoreHistoryResponse{Points: out}), nil
}

func (h *handler) GetHealthOverview(ctx context.Context, _ *connect.Request[catalogv1.GetHealthOverviewRequest]) (*connect.Response[catalogv1.GetHealthOverviewResponse], error) {
	report, err := h.report(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("compute catalog health: %w", err))
	}
	assets, err := catalogcoverage.LoadCatalog(filepath.Join(h.repoRoot, "scenarios", "react-component-library", "catalog"))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("load catalog health assets: %w", err))
	}
	rows := map[string]catalogcoverage.Row{}
	for _, row := range report.Rows {
		if row.AssetID != "" {
			rows[row.AssetID] = row
		}
	}
	response := &catalogv1.GetHealthOverviewResponse{Coverage: toProto(report), KindMismatchCount: int32(len(report.KindMismatches))}
	for _, mismatch := range report.KindMismatches {
		response.KindMismatches = append(response.KindMismatches, &catalogv1.KindMismatch{AssetId: mismatch.AssetID, DeclaredKind: mismatch.DeclaredKind, DerivedKind: mismatch.DerivedKind, Message: mismatch.Message})
	}
	h.quarantineMu.RLock()
	for gate := range h.quarantined {
		response.QuarantinedGates = append(response.QuarantinedGates, gate)
	}
	h.quarantineMu.RUnlock()
	sort.Strings(response.QuarantinedGates)
	for _, asset := range assets {
		row := rows[asset.ID]
		health := "blocked"
		if row.Bucket == catalogcoverage.BucketPlannedBuilt && row.AssetScore >= 0.9 {
			health = "healthy"
		} else if row.Bucket == catalogcoverage.BucketPlannedBuilt && row.AssetScore >= 0.5 {
			health = "degraded"
		}
		staleness := 0.0
		if row.NewestEvidence != "" {
			if stamp, parseErr := time.Parse(time.RFC3339Nano, row.NewestEvidence); parseErr == nil {
				staleness = time.Since(stamp).Hours() / 24
			}
		}
		response.Nodes = append(response.Nodes, &catalogv1.HealthNode{Asset: &catalogv1.AssetNode{AssetId: asset.ID, Name: asset.Name, Kind: asset.Kind, Rung: int32(asset.Rung), RungName: asset.RungName, Domain: asset.Domain, DomainOrder: int32(asset.DomainOrder)}, Score: row.AssetScore * 100, Weight: row.Weight, Health: health, StalenessDays: staleness, VisualCurrent: row.VisualEvidence})
		for _, required := range asset.Requires {
			response.Edges = append(response.Edges, &catalogv1.HealthEdge{FromAssetId: asset.ID, ToAssetId: required, Relation: "requires"})
		}
		for _, suggested := range asset.Suggests {
			response.Edges = append(response.Edges, &catalogv1.HealthEdge{FromAssetId: asset.ID, ToAssetId: suggested, Relation: "suggests"})
		}
	}
	promote, _ := catalogcoverage.NextWorkLanes(*report, 0)
	for _, row := range promote {
		response.Promote = append(response.Promote, rowProto(row))
	}
	if h.evidence != nil {
		if history, historyErr := h.evidence.ScoreHistory(ctx, h.repoRoot, ""); historyErr == nil {
			for _, point := range history {
				response.History = append(response.History, scoreHistoryProto(point))
				response.InstrumentMovedCount += int32(point.InstrumentMoved)
			}
		}
	}
	return connect.NewResponse(response), nil
}

func (h *handler) CaptureEvidence(ctx context.Context, req *connect.Request[catalogv1.CaptureEvidenceRequest]) (*connect.Response[catalogv1.CaptureEvidenceResponse], error) {
	if h.captureRunner == nil || h.captureRunner.Assets == nil || h.captureRunner.Executor == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("catalog evidence capture is not wired to the BAS-backed component runner"))
	}
	assetID := strings.TrimSpace(req.Msg.GetAssetId())
	if assetID == "" && !req.Msg.GetAll() {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("asset_id is required unless all=true"))
	}
	root := filepath.Join(h.repoRoot, "scenarios", "react-component-library")
	directory := filepath.Join(root, "captures", "catalog")
	if assetID == "" {
		assetID = "all"
	}
	directory = filepath.Join(directory, assetID)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("create capture directory: %w", err))
	}
	catalogRoot := filepath.Join(root, "catalog")
	assets, assetErr := catalogcoverage.LoadCatalog(catalogRoot)
	impls, implErr := catalogcoverage.LoadImplementations(filepath.Join(root, "library"))
	if assetErr != nil || implErr != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("load capture catalog: %v %v", assetErr, implErr))
	}
	wanted := map[string]bool{}
	if req.Msg.GetAll() {
		for _, asset := range assets {
			wanted[asset.ID] = true
		}
	} else {
		wanted[assetID] = true
	}
	implByAsset := map[string]catalogcoverage.Implementation{}
	for _, impl := range impls {
		if impl.CatalogID != "" {
			implByAsset[impl.CatalogID] = impl
		}
	}
	// Catalog-wide capture is deliberately bounded. The client can resume from
	// next_offset without repeating already completed assets or exceeding the
	// Connect client deadline.
	if req.Msg.GetAll() {
		ids := make([]string, 0, len(wanted))
		for id := range wanted {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		offset := int(req.Msg.GetOffset())
		if offset < 0 || offset > len(ids) {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("offset %d is outside catalog range 0..%d", offset, len(ids)))
		}
		limit := int(req.Msg.GetLimit())
		if limit <= 0 || offset+limit > len(ids) {
			limit = len(ids) - offset
		}
		wanted = map[string]bool{}
		for _, id := range ids[offset : offset+limit] {
			wanted[id] = true
		}
	}
	existing := map[string]bool{}
	if req.Msg.GetChangedOnly() && h.evidence != nil {
		if previous, listErr := h.evidence.List(ctx); listErr == nil {
			for _, item := range previous {
				if item.Gate == "visual" && item.Result == "pass" {
					existing[item.AssetID] = true
				}
			}
		}
	}
	type captureRow struct {
		AssetID     string   `json:"assetId"`
		Version     string   `json:"version"`
		StoryID     string   `json:"storyId"`
		StoryIDs    []string `json:"storyIds,omitempty"`
		ArtifactKind string   `json:"artifactKind,omitempty"`
		Result      string   `json:"result"`
		Report      any      `json:"report"`
	}
	rows := make([]captureRow, 0)
	// A bounded --all run writes one durable manifest across requests. Starting
	// at offset zero intentionally begins a fresh migration; later batches
	// merge into the prior manifest so an interrupted run remains inspectable.
	if req.Msg.GetAll() && req.Msg.GetOffset() > 0 {
		if prior, readErr := os.ReadFile(filepath.Join(directory, "capture-manifest.json")); readErr == nil {
			var priorManifest struct {
				Captures []captureRow `json:"captures"`
			}
			if json.Unmarshal(prior, &priorManifest) == nil {
				rows = append(rows, priorManifest.Captures...)
			}
		}
	}
	missing := []string{}
	var evidenceRows []catalogcoverage.GateEvidence
	for id := range wanted {
		impl, ok := implByAsset[id]
		if !ok || impl.Latest == "" {
			missing = append(missing, id)
			continue
		}
		if req.Msg.GetChangedOnly() && existing[id] {
			continue
		}
		libraryID := "react-component-library:" + impl.Name
		component, getErr := h.captureRunner.Assets.GetByLibraryID(ctx, libraryID)
		if getErr != nil {
			missing = append(missing, id)
			continue
		}
		report, runErr := h.captureRunner.Run(ctx, componenttests.Request{ComponentID: component.ID, Version: impl.Latest, IncludeClosure: false})
		revision, revisionErr := catalogcoverage.CurrentRevision(h.repoRoot, id)
		if revisionErr != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("hash capture asset %q: %w", id, revisionErr))
		}
		assetTarget := "react-vite"
		for _, asset := range assets {
			if asset.ID == id && len(asset.Targets) > 0 && asset.Targets[0] != "" {
				assetTarget = asset.Targets[0]
			}
		}
		if runErr != nil {
			rows = append(rows, captureRow{AssetID: id, Version: impl.Latest, ArtifactKind: "individual", Result: "failed", Report: map[string]any{"error": runErr.Error(), "libraryId": libraryID}})
			measurement, _ := json.Marshal(map[string]any{"runner": "bas-capture-service", "error": runErr.Error()})
			evidenceRows = append(evidenceRows, catalogcoverage.GateEvidence{AssetID: id, Target: assetTarget, Gate: "visual", Version: impl.Latest, Result: "fail", MeasurementJSON: string(measurement), SourceRevision: revision})
			continue
		}
		assetPassed := report.Verdict == componenttests.VerdictPassed
		for _, result := range report.Results {
			if result.Stage != componenttests.StageEvidence {
				continue
			}
			storyPassed := result.Verdict == componenttests.VerdictPassed
			if !storyPassed {
				assetPassed = false
			}
			row := captureRow{AssetID: id, Version: result.Version, StoryID: result.Subject, ArtifactKind: "individual", Result: string(result.Verdict), Report: result}
			if strings.HasPrefix(result.Subject, "review-sheet:") {
				row.ArtifactKind = "review-sheet"
				row.StoryIDs = strings.Split(strings.TrimPrefix(result.Subject, "review-sheet:"), ",")
			}
			rows = append(rows, row)
		}
		measurement, _ := json.Marshal(map[string]any{"runner": "bas-capture-service", "reportId": report.ID, "artifacts": report.Artifacts})
		result := "fail"
		if assetPassed {
			result = "pass"
		}
		evidenceRows = append(evidenceRows, catalogcoverage.GateEvidence{AssetID: id, Target: assetTarget, Gate: "visual", Version: impl.Latest, Result: result, MeasurementJSON: string(measurement), SourceRevision: revision})
	}
	manifest := map[string]any{"asset_id": assetID, "changed_only": req.Msg.GetChangedOnly(), "runner": "browser-automation-studio.capture-service", "captured_at": time.Now().UTC().Format(time.RFC3339Nano), "captures": rows, "bounded": req.Msg.GetAll() && req.Msg.GetLimit() > 0, "offset": req.Msg.GetOffset(), "limit": req.Msg.GetLimit()}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("encode capture manifest: %w", err))
	}
	if err := os.WriteFile(filepath.Join(directory, "capture-manifest.json"), append(data, '\n'), 0o644); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("write capture manifest: %w", err))
	}
	if h.evidence != nil {
		// Evidence capture is a durable write. If a CLI disconnects after BAS
		// has completed, do not discard the already-collected measurement merely
		// because the request context was canceled while the database write ran.
		persistCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := h.evidence.Save(persistCtx, evidenceRows); err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("persist capture evidence: %w", err))
		}
		h.reports.invalidate()
	}
	nextOffset := int(req.Msg.GetOffset())
	if req.Msg.GetAll() {
		nextOffset += len(wanted)
	}
	complete := !req.Msg.GetAll() || nextOffset >= len(assets)
	return connect.NewResponse(&catalogv1.CaptureEvidenceResponse{AssetId: assetID, CaptureDirectory: directory, WorkbenchUrl: "/workbench?asset=" + assetID, RowsWritten: int32(len(evidenceRows)), MissingContractAssets: missing, NextOffset: int32(nextOffset), Complete: complete}), nil
}

func toProto(report *catalogcoverage.Report) *catalogv1.CoverageReport {
	out := &catalogv1.CoverageReport{Totals: map[string]int32{}, Maturity: maturityProto(report), CompositionScores: map[string]float64{}, CompositionMedian: report.CompositionMedian, BespokeEscapeCount: int32(report.BespokeEscapeCount), CompositionBlockedAssetCount: int32(report.CompositionBlockedAssetCount), DeclaredCapabilityAssetCount: int32(report.CapabilityReport.DeclaredAssetCount), DeclaredUncheckableAssetCount: int32(report.CapabilityReport.UncheckableAssetCount), UnmeasuredCapabilityAssetCount: int32(report.CapabilityReport.UnmeasuredAssetCount), CapabilityDeclarationCount: int32(report.CapabilityReport.DeclarationCount)}
	for assetID, score := range report.CompositionScores {
		out.CompositionScores[assetID] = score
	}
	for _, row := range report.Rows {
		out.Rows = append(out.Rows, rowProto(row))
	}
	for key, value := range report.Totals {
		out.Totals[string(key)] = int32(value)
	}
	for key, value := range report.ByDomain {
		out.ByDomain = append(out.ByDomain, &catalogv1.Rollup{Key: key, Planned: int32(value.Planned), Built: int32(value.Built)})
	}
	for key, value := range report.ByPriority {
		out.ByPriority = append(out.ByPriority, &catalogv1.Rollup{Key: key, Planned: int32(value.Planned), Built: int32(value.Built)})
	}
	for _, capability := range report.CapabilityReport.Capabilities {
		out.CapabilityCoverage = append(out.CapabilityCoverage, &catalogv1.DeclaredCapabilityCoverage{
			Capability:         capability.Capability,
			Title:              capability.Title,
			Status:             capability.Status,
			Checkable:          capability.Checkable,
			Unmeasured:         capability.Unmeasured,
			DeclaredAssetCount: int32(capability.DeclaredAssetCount),
			AssetIds:           capability.AssetIDs,
			Blockers:           capability.Blockers,
		})
	}
	return out
}

func rowProto(row catalogcoverage.Row) *catalogv1.CoverageRow {
	return &catalogv1.CoverageRow{AssetId: row.AssetID, Name: row.Name, Domain: row.Domain, Kind: row.Kind, Priority: row.Priority, Bucket: string(row.Bucket), Platform: row.Platform, Target: row.Target, Achieved: string(row.Achieved), Implementation: row.Implementation, BlocksDownstream: int32(row.BlocksDownstream), Rung: int32(row.Rung), RungName: row.RungName, DomainOrder: int32(row.DomainOrder), AssetScore: row.AssetScore * 100, Weight: row.Weight, PassedGates: row.PassedGates, FailedGates: row.FailedGates, NearestBlockingGate: row.NearestBlockingGate, NewestEvidence: row.NewestEvidence, VisualEvidence: row.VisualEvidence}
}

func rowsProto(rows []catalogcoverage.Row) []*catalogv1.CoverageRow {
	out := make([]*catalogv1.CoverageRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, rowProto(row))
	}
	return out
}

func maturityProto(report *catalogcoverage.Report) *catalogv1.MaturitySummary {
	m := report.Maturity
	out := &catalogv1.MaturitySummary{Total: int32(m.Total), AtOrAboveTarget: int32(m.AtOrAboveTarget), ByRung: map[string]int32{}}
	for key, value := range m.ByRung {
		out.ByRung[string(key)] = int32(value)
	}
	for key, value := range report.ByGate {
		out.ByGate = append(out.ByGate, &catalogv1.ScoreBreakdown{Key: key, Passed: int32(value.Passed), Applicable: int32(value.Applicable), Score: value.Score * 100})
	}
	for key, value := range report.ByRungScore {
		out.ByRungScore = append(out.ByRungScore, &catalogv1.ScoreBreakdown{Key: string(key), Passed: int32(value.Passed), Applicable: int32(value.Applicable), Score: value.Score * 100})
	}
	for _, value := range report.Corpus {
		out.Corpus = append(out.Corpus, &catalogv1.CorpusStatus{Gate: value.Gate, Result: value.Result, FindingCount: int32(value.FindingCount), RunnerErrorCount: int32(value.RunnerErrorCount)})
	}
	out.WeightedAssetScore = report.Score
	out.ScoreWeightNumerator = report.ScoreWeightNumerator
	out.ScoreWeightDenominator = report.ScoreWeightDenominator
	out.PassEvidence = int32(report.PassEvidence)
	out.FailEvidence = int32(report.FailEvidence)
	out.UnmeasuredEvidence = int32(report.UnmeasuredEvidence)
	out.KindMismatchCount = int32(len(report.KindMismatches))
	out.CatalogCompletion = metricProto(m.CatalogCompletion)
	out.MandatoryGateCoverage = metricProto(m.MandatoryGateCoverage)
	out.WeightedQuality = metricProto(m.WeightedQuality)
	out.ProductionReadyCoverage = metricProto(m.ProductionReadyCoverage)
	return out
}

func scoreHistoryProto(point catalogcoverage.ScoreHistory) *catalogv1.ScoreHistoryPoint {
	out := &catalogv1.ScoreHistoryPoint{RecordedAt: point.RecordedAt, Score: point.Score, AssetsAt_100: int32(point.AssetsAt100), AssetsBelow_50: int32(point.AssetsBelow50), WeightVectorRegenerated: point.WeightVectorRegenerated, ScoringModelVersion: int32(point.ScoringModelVersion), SourceRevision: point.SourceRevision, InstrumentMovedCount: int32(point.InstrumentMoved), KindMismatchCount: int32(point.KindMismatchCount)}
	for _, event := range point.Events {
		out.Events = append(out.Events, &catalogv1.ScoreHistoryEvent{Type: event.Type, AssetId: event.AssetID, SourceRevision: event.SourceRevision, DeclaredKind: event.DeclaredKind, DerivedKind: event.DerivedKind})
	}
	return out
}

func metricProto(metric catalogcoverage.CoverageMetric) *catalogv1.CoverageMetric {
	return &catalogv1.CoverageMetric{Numerator: int32(metric.Numerator), Denominator: int32(metric.Denominator), Ratio: metric.Ratio}
}
