package catalog

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"react-component-library/internal/capabilities"
	"react-component-library/internal/catalogbuild"
	"react-component-library/internal/catalogcoverage"
	"react-component-library/internal/catalogexperience"
	"react-component-library/internal/components"
	componenttests "react-component-library/internal/componenttests"
	"react-component-library/internal/gates"
	"react-component-library/internal/jobs"
	"react-component-library/internal/librarywalk"
	"react-component-library/internal/module"
	"react-component-library/internal/versionledger"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	catalogv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/catalog"
	catalogconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/catalog/catalog_v1connect"
	"google.golang.org/protobuf/proto"
)

type handler struct {
	repoRoot        string
	evidence        *catalogcoverage.EvidenceStore
	reports         reportCache
	quarantineMu    sync.RWMutex
	quarantined     map[string]bool
	quarantineKnown bool
	captureRunner   *componenttests.Runner
	testService     *componenttests.Service
	assets          components.Service
	jobRunner       *jobs.Runner
	checkMu         sync.RWMutex
	checkCache      map[string]*catalogv1.CheckAssetResponse
	evidenceMu      sync.Mutex
}

type (
	skipCalibrationContextKey  struct{}
	skipJobAdmissionContextKey struct{}
	preparedSetsContextKey     struct{}
)

// Module exposes the same live coverage projection used by the component-test
// phase. No CLI-only computation or second catalog join is permitted.
func Module(repoRoot string, dbs ...*sql.DB) module.Module {
	var evidence *catalogcoverage.EvidenceStore
	if len(dbs) > 0 && dbs[0] != nil {
		evidence = catalogcoverage.NewEvidenceStore(dbs[0])
	}
	h := &handler{repoRoot: repoRoot, evidence: evidence, quarantined: map[string]bool{}, jobRunner: jobs.New(nil), checkCache: map[string]*catalogv1.CheckAssetResponse{}}
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
		testService: componenttests.NewService(componenttests.Runner{Assets: assets, Stories: assets, Executor: executor, Revision: func(ctx context.Context, assetID, version string) (string, error) {
			return catalogcoverage.CurrentRevisionForVersion(repoRoot, resolveRevisionLibraryID(ctx, assets, repoRoot, assetID), version)
		}}, componenttests.NewSQLiteRepository(db)),
		assets:     assets,
		jobRunner:  jobs.New(db),
		checkCache: map[string]*catalogv1.CheckAssetResponse{},
	}
	return h.module()
}

func (h *handler) module() module.Module {
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
	{ID: "catalog_asset_check", Path: catalogconnect.CatalogServiceCheckAssetProcedure, Method: "POST", Summary: "Check one asset and return one verdict", Category: "catalog"},
	{ID: "catalog_graph", Path: catalogconnect.CatalogServiceGetAssetRelationshipsProcedure, Method: "POST", Summary: "Read catalog asset relationships", Category: "catalog"},
	{ID: "catalog_structure", Path: catalogconnect.CatalogServiceGetCatalogStructureProcedure, Method: "POST", Summary: "Read catalog structure", Category: "catalog"},
	{ID: "catalog_reconcile", Path: catalogconnect.CatalogServiceReconcileGraphProcedure, Method: "POST", Summary: "Reconcile catalog dependency graphs", Category: "catalog"},
	{ID: "catalog_ports", Path: catalogconnect.CatalogServiceGetAssetPortContractProcedure, Method: "POST", Summary: "Read asset host obligations", Category: "catalog"},
	{ID: "catalog_score_history", Path: catalogconnect.CatalogServiceGetScoreHistoryProcedure, Method: "POST", Summary: "Read carried-forward asset score history", Category: "catalog"},
	{ID: "catalog_health", Path: catalogconnect.CatalogServiceGetHealthOverviewProcedure, Method: "POST", Summary: "Read server-computed catalog health", Category: "catalog"},
	{ID: "catalog_readiness", Path: catalogconnect.CatalogServiceGetReadinessProcedure, Method: "POST", Summary: "Report catalog readiness and triage", Category: "catalog"},
	{ID: "catalog_capture_evidence", Path: catalogconnect.CatalogServiceCaptureEvidenceProcedure, Method: "POST", Summary: "Capture declared visual evidence", Category: "catalog"},
}

func (h *handler) CheckAsset(ctx context.Context, req *connect.Request[catalogv1.CheckAssetRequest]) (*connect.Response[catalogv1.CheckAssetResponse], error) {
	assetID := strings.TrimSpace(req.Msg.GetAssetId())
	if assetID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("asset_id is required"))
	}
	cacheKey := ""
	if h.assets != nil {
		if component, resolveErr := resolveAsset(ctx, h.assets, assetID); resolveErr == nil {
			version := firstNonEmpty(req.Msg.GetVersion(), component.LatestVersion)
			if revision, revisionErr := catalogcoverage.CurrentRevisionForVersion(h.repoRoot, component.LibraryID, version); revisionErr == nil {
				cacheKey = component.LibraryID + "@" + version + "#" + revision
				h.checkMu.RLock()
				cached := h.checkCache[cacheKey]
				h.checkMu.RUnlock()
				if cached != nil && cached.GetVerdict() != "STALE_TESTS" {
					return connect.NewResponse(proto.Clone(cached).(*catalogv1.CheckAssetResponse)), nil
				}
			}
		}
	}
	started := time.Now()
	stages := make([]*catalogv1.AssetCheckStage, 0, 4)
	var response *catalogv1.CheckAssetResponse
	addStage := func(name, status, detail string, since time.Time) {
		stages = append(stages, &catalogv1.AssetCheckStage{Name: name, Status: status, Seconds: time.Since(since).Seconds(), Detail: detail})
		if response != nil {
			response.Stages = stages
		}
	}
	buildStart := time.Now()
	if _, err := catalogbuild.Build(h.repoRoot, catalogbuild.Options{}); err != nil {
		addStage("generator", "fault", err.Error(), buildStart)
		return connect.NewResponse(&catalogv1.CheckAssetResponse{AssetId: assetID, Verdict: "FAULT", Stages: stages}), nil
	}
	if _, err := catalogbuild.Build(h.repoRoot, catalogbuild.Options{Check: true}); err != nil {
		addStage("generator", "fault", err.Error(), buildStart)
		return connect.NewResponse(&catalogv1.CheckAssetResponse{AssetId: assetID, Verdict: "FAULT", Stages: stages}), nil
	}
	addStage("generator", "passed", "catalog build and check passed", buildStart)

	gateStart := time.Now()
	gateResp, err := h.RunGate(ctx, connect.NewRequest(&catalogv1.RunGateRequest{All: true, AssetId: assetID}))
	if err != nil {
		addStage("gates", "fault", err.Error(), gateStart)
		return connect.NewResponse(&catalogv1.CheckAssetResponse{AssetId: assetID, Verdict: "FAULT", Stages: stages}), nil
	}
	findings := append([]*catalogv1.GateFinding(nil), gateResp.Msg.Findings...)
	findings = append(findings, gateResp.Msg.RunnerErrors...)
	blocking := 0
	for _, finding := range findings {
		if finding.GetBlocking() || strings.EqualFold(finding.GetSeverity(), "error") || finding.GetSeverity() == "" {
			blocking++
		}
	}
	gateStatus := "passed"
	if blocking > 0 {
		gateStatus = "blocked"
	}
	addStage("gates", gateStatus, fmt.Sprintf("%d finding(s), %d blocking", len(findings), blocking), gateStart)

	response = &catalogv1.CheckAssetResponse{AssetId: assetID, Findings: findings, Stages: stages}
	if h.testService == nil || h.assets == nil {
		if blocking > 0 {
			addStage("tests", "skipped", "gate findings block the asset; component tests were not run", started)
			packageStart := time.Now()
			packageResp, packageErr := h.RunGate(ctx, connect.NewRequest(&catalogv1.RunGateRequest{Gate: "dist-resolution", AssetId: assetID}))
			if packageErr != nil {
				addStage("package", "fault", packageErr.Error(), packageStart)
				response.Verdict = "FAULT"
				return connect.NewResponse(response), nil
			}
			response.Findings = append(response.Findings, packageResp.Msg.Findings...)
			addStage("package", packageStageStatus(packageResp.Msg.Findings), fmt.Sprintf("%d package finding(s)", len(packageResp.Msg.Findings)), packageStart)
			response.Verdict = "BLOCKED"
			return connect.NewResponse(response), nil
		}
		response.Verdict = "FAULT"
		addStage("tests", "fault", "component-test service is not configured", started)
		return connect.NewResponse(response), nil
	}
	component, resolveErr := resolveAsset(ctx, h.assets, assetID)
	if resolveErr != nil {
		if blocking > 0 {
			addStage("tests", "skipped", "gate findings block the asset; indexed component lookup was not needed", started)
			packageStart := time.Now()
			packageResp, packageErr := h.RunGate(ctx, connect.NewRequest(&catalogv1.RunGateRequest{Gate: "dist-resolution", AssetId: assetID}))
			if packageErr != nil {
				addStage("package", "fault", packageErr.Error(), packageStart)
				response.Verdict = "FAULT"
				return connect.NewResponse(response), nil
			}
			response.Findings = append(response.Findings, packageResp.Msg.Findings...)
			addStage("package", packageStageStatus(packageResp.Msg.Findings), fmt.Sprintf("%d package finding(s)", len(packageResp.Msg.Findings)), packageStart)
			response.Verdict = "BLOCKED"
			return connect.NewResponse(response), nil
		}
		response.Verdict = "FAULT"
		addStage("tests", "fault", resolveErr.Error(), started)
		return connect.NewResponse(response), nil
	}
	requestedVersion := firstNonEmpty(req.Msg.GetVersion(), component.LatestVersion)
	if component.AssetKind == components.AssetKindHook {
		addStage("tests", "n/a", "hooks have no browser-backed component-test stage", started)
		packageStart := time.Now()
		packageResp, packageErr := h.RunGate(ctx, connect.NewRequest(&catalogv1.RunGateRequest{Gate: "dist-resolution", AssetId: assetID}))
		if packageErr != nil {
			addStage("package", "fault", packageErr.Error(), packageStart)
			response.Verdict = "FAULT"
			return connect.NewResponse(response), nil
		}
		response.Findings = append(response.Findings, packageResp.Msg.Findings...)
		addStage("package", packageStageStatus(packageResp.Msg.Findings), fmt.Sprintf("%d package finding(s)", len(packageResp.Msg.Findings)), packageStart)
		for _, finding := range packageResp.Msg.Findings {
			if strings.EqualFold(finding.GetSeverity(), "error") || finding.GetSeverity() == "" {
				blocking++
			}
		}
		if blocking > 0 {
			response.Verdict = "BLOCKED"
		} else {
			response.Verdict = "PUBLISHABLE"
		}
		return connect.NewResponse(response), nil
	}
	testStart := time.Now()
	// The scoped gates already validate the dependency closure. The browser
	// verdict belongs to the requested asset; running every transitive
	// foundation story here lets an unrelated helper contract poison the
	// asset's one-command result.
	testRequest := componenttests.Request{ComponentID: component.LibraryID, Version: requestedVersion, IncludeClosure: false}
	var report componenttests.Report
	var reused bool
	var testErr error
	// Asset check is the single publishability command: it must reuse a fresh
	// report or create one, rather than returning an intermediate stale verdict
	// that requires a second command. RunWithReuse preserves the fast path.
	if req.Msg.GetRunTests() {
		report, reused, testErr = h.testService.RunWithReuse(ctx, testRequest)
	} else {
		report, reused, testErr = h.testService.Reusable(ctx, testRequest)
		if testErr == nil && !reused {
			addStage("tests", "stale", "no fresh report exists; rerun with --run-tests", testStart)
			packageStart := time.Now()
			packageResp, packageErr := h.RunGate(ctx, connect.NewRequest(&catalogv1.RunGateRequest{Gate: "dist-resolution", AssetId: assetID}))
			if packageErr != nil {
				addStage("package", "fault", packageErr.Error(), packageStart)
				response.Verdict = "FAULT"
				return connect.NewResponse(response), nil
			}
			response.Findings = append(response.Findings, packageResp.Msg.Findings...)
			addStage("package", packageStageStatus(packageResp.Msg.Findings), fmt.Sprintf("%d package finding(s)", len(packageResp.Msg.Findings)), packageStart)
			response.Verdict = "STALE_TESTS"
			return connect.NewResponse(response), nil
		}
	}
	if testErr != nil {
		response.Verdict = "FAULT"
		addStage("tests", "fault", testErr.Error(), testStart)
		return connect.NewResponse(response), nil
	}
	response.ReusedReportId = report.ID
	response.SourceRevision = report.SourceRevision
	if report.Verdict != componenttests.VerdictPassed {
		blocking++
	}
	// The evidence gate reads every materialized version of an asset so its
	// corpus result remains complete. An asset check, however, is explicitly
	// version-pinned (or resolves to the latest version); historical evidence
	// must not poison that requested version's verdict.
	if report.Verdict == componenttests.VerdictPassed {
		filtered := findings[:0]
		removedBlocking := 0
		for _, finding := range findings {
			if finding.GetCode() == "catalog.evidence_freshness" {
				if finding.GetBlocking() || strings.EqualFold(finding.GetSeverity(), "error") || finding.GetSeverity() == "" {
					removedBlocking++
				}
				continue
			}
			filtered = append(filtered, finding)
		}
		findings = filtered
		blocking -= removedBlocking
		response.Findings = findings
		if len(stages) > 1 {
			gateStage := stages[1]
			gateStage.Status = map[bool]string{true: "blocked", false: "passed"}[blocking > 0]
			gateStage.Detail = fmt.Sprintf("%d finding(s), %d blocking", len(findings), blocking)
		}
	}
	addStage("tests", componentTestStageStatus(report.Verdict), fmt.Sprintf("report %s (%s; reused=%t)", report.ID, report.Verdict, reused), testStart)
	packageStart := time.Now()
	packageResp, packageErr := h.RunGate(ctx, connect.NewRequest(&catalogv1.RunGateRequest{Gate: "dist-resolution", AssetId: assetID}))
	if packageErr != nil {
		addStage("package", "fault", packageErr.Error(), packageStart)
		response.Verdict = "FAULT"
		return connect.NewResponse(response), nil
	}
	findings = append(findings, packageResp.Msg.Findings...)
	for _, finding := range packageResp.Msg.Findings {
		if strings.EqualFold(finding.GetSeverity(), "error") || finding.GetSeverity() == "" {
			blocking++
		}
	}
	response.Findings = findings
	addStage("package", packageStageStatus(packageResp.Msg.Findings), fmt.Sprintf("%d package finding(s)", len(packageResp.Msg.Findings)), packageStart)
	if blocking > 0 {
		response.Verdict = "BLOCKED"
	} else {
		response.Verdict = "PUBLISHABLE"
	}
	if cacheKey != "" && response.GetVerdict() != "STALE_TESTS" {
		h.checkMu.Lock()
		h.checkCache[cacheKey] = proto.Clone(response).(*catalogv1.CheckAssetResponse)
		h.checkMu.Unlock()
	}
	return connect.NewResponse(response), nil
}

func resolveAsset(ctx context.Context, assets components.Service, requested string) (components.Component, error) {
	if component, err := assets.Get(ctx, requested); err == nil {
		if component.LibraryID != "" && component.LibraryID != requested {
			return component, nil
		}
	}
	if component, err := assets.GetByLibraryID(ctx, requested); err == nil {
		if component.LibraryID != "" {
			return component, nil
		}
	}
	// Catalog ids are the public asset-check vocabulary, while the registry
	// service historically only exposed primary and library-id lookups. Resolve
	// the public id through the same indexed projection instead of making the
	// one-command workflow require callers to translate identifiers themselves.
	indexed, err := assets.List(ctx, components.SearchQuery{Limit: 1000})
	if err == nil {
		for _, component := range indexed {
			if component.CatalogID == requested && component.LibraryID != "" {
				return component, nil
			}
		}
	}
	return components.Component{}, fmt.Errorf("asset %q is not indexed", requested)
}

func resolveRevisionLibraryID(ctx context.Context, assets components.Service, repoRoot, requested string) string {
	if component, err := assets.Get(ctx, requested); err == nil && component.LibraryID != "" {
		return component.LibraryID
	}
	if component, err := assets.GetByLibraryID(ctx, requested); err == nil && component.LibraryID != "" {
		return component.LibraryID
	}
	if indexed, err := assets.List(ctx, components.SearchQuery{Limit: 2000}); err == nil {
		for _, component := range indexed {
			if component.ID == requested || component.CatalogID == requested || component.LibraryID == requested {
				return component.LibraryID
			}
		}
	}
	implementations, err := catalogcoverage.LoadImplementations(filepath.Join(repoRoot, "scenarios", "react-component-library", "library"))
	if err == nil {
		for _, implementation := range implementations {
			if implementation.CatalogID == requested && implementation.LibraryID != "" {
				return implementation.LibraryID
			}
		}
	}
	return requested
}

func packageStageStatus(findings []*catalogv1.GateFinding) string {
	for _, finding := range findings {
		if finding == nil || finding.GetBlocking() || strings.EqualFold(finding.GetSeverity(), "error") || finding.GetSeverity() == "" {
			return "blocked"
		}
	}
	return "passed"
}

func componentTestStageStatus(verdict componenttests.Verdict) string {
	if verdict == componenttests.VerdictPassed {
		return "passed"
	}
	return "blocked"
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func findingCatalogID(finding gates.Finding) string {
	if finding.Scope == gates.FindingScopeCorpus || strings.HasPrefix(finding.AssetID, "__corpus__") {
		return ""
	}
	return firstNonEmpty(finding.CatalogID, finding.AssetID)
}

func findingScope(finding gates.Finding) catalogv1.FindingScope {
	if finding.Scope == gates.FindingScopeCorpus || strings.HasPrefix(finding.AssetID, "__corpus__") {
		return catalogv1.FindingScope_FINDING_SCOPE_CORPUS
	}
	return catalogv1.FindingScope_FINDING_SCOPE_ASSET
}

func findingSeverity(severity string) catalogv1.FindingSeverity {
	if severity == "info" {
		return catalogv1.FindingSeverity_FINDING_SEVERITY_INFO
	}
	if severity == "warning" {
		return catalogv1.FindingSeverity_FINDING_SEVERITY_WARNING
	}
	return catalogv1.FindingSeverity_FINDING_SEVERITY_BLOCKING
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

func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}

func (h *handler) RunGate(ctx context.Context, req *connect.Request[catalogv1.RunGateRequest]) (*connect.Response[catalogv1.RunGateResponse], error) {
	gate := strings.TrimSpace(req.Msg.GetGate())
	definitions, definitionErr := catalogcoverage.LoadGateDefinitions(filepath.Join(h.repoRoot, "scenarios", "react-component-library", "catalog", "config.json"))
	if definitionErr != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("load catalog gate definitions: %w", definitionErr))
	}
	if req.Msg.GetAll() {
		prepared := map[gates.Reads]librarywalk.Set{}
		readLevel := gates.ReadsCorpus
		if strings.TrimSpace(req.Msg.GetAssetId()) != "" {
			readLevel = gates.ReadsClosure
		}
		set, setErr := librarywalk.Files(ctx, h.repoRoot, librarywalk.Scope{Assets: assetSetSlice(req.Msg.GetAssetId())}, librarywalk.Reads(readLevel))
		if setErr != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("prepare catalog gate inputs: %w", setErr))
		}
		prepared[readLevel] = set
		ctx = context.WithValue(ctx, preparedSetsContextKey{}, prepared)
		aggregate := &catalogv1.RunGateResponse{Gate: "all"}
		type gateResult struct {
			index int
			resp  *connect.Response[catalogv1.RunGateResponse]
			err   error
		}
		ordered := make([]gateResult, len(definitions))
		assetID := req.Msg.GetAssetId()
		runErr := h.jobRunner.RunMatrix(ctx, len(definitions), minInt(4, len(definitions)), func(matrixCtx context.Context, index int) error {
			definition := definitions[index]
			if definition.ID == "documentation" && !req.Msg.GetIncludeAdvisory() {
				return nil
			}
			if assetID != "" {
				if registered, ok := gates.Lookup(definition.ID); ok && registered.Reads == gates.ReadsCorpus {
					return nil
				}
			}
			nestedCtx := context.WithValue(matrixCtx, skipCalibrationContextKey{}, true)
			nestedCtx = context.WithValue(nestedCtx, skipJobAdmissionContextKey{}, true)
			result, runErr := h.RunGate(nestedCtx, connect.NewRequest(&catalogv1.RunGateRequest{Gate: definition.ID, AssetId: assetID, CalibrationOnly: req.Msg.GetCalibrationOnly()}))
			ordered[index] = gateResult{index: index, resp: result, err: runErr}
			return runErr
		})
		if runErr != nil {
			log.Printf("catalog matrix canceled: %v", runErr)
			if errors.Is(runErr, context.Canceled) || errors.Is(runErr, context.DeadlineExceeded) {
				return nil, connect.NewError(connect.CodeCanceled, runErr)
			}
			return nil, connect.NewError(connect.CodeInternal, runErr)
		}
		for _, result := range ordered {
			if result.err != nil {
				if errors.Is(result.err, context.Canceled) || errors.Is(result.err, context.DeadlineExceeded) {
					return nil, connect.NewError(connect.CodeCanceled, result.err)
				}
				return nil, result.err
			}
			if result.resp == nil {
				continue
			}
			aggregate.InspectedFiles += result.resp.Msg.InspectedFiles
			aggregate.Findings = append(aggregate.Findings, result.resp.Msg.Findings...)
			aggregate.RunnerErrors = append(aggregate.RunnerErrors, result.resp.Msg.RunnerErrors...)
			aggregate.EvidenceRowsWritten += result.resp.Msg.EvidenceRowsWritten
			aggregate.Calibration = append(aggregate.Calibration, result.resp.Msg.Calibration...)
			aggregate.NonDiscriminating = aggregate.NonDiscriminating || result.resp.Msg.NonDiscriminating
			if len(result.resp.Msg.SurfaceVerdictCounts) > 0 {
				if aggregate.SurfaceVerdictCounts == nil {
					aggregate.SurfaceVerdictCounts = map[string]int32{}
				}
				for verdict, count := range result.resp.Msg.SurfaceVerdictCounts {
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
	// Calibration runs against an isolated overlay. In particular, the
	// immutable-version fixture creates an overlay database; using the live
	// routed database here would make the planted drift invisible and falsely
	// quarantine an otherwise measurable gate.
	gateCalibrationRunner := gates.GateRunnerFor(gate)
	calibration := gates.CalibrationReport{}
	var calibrationErr error
	if _, skipCalibration := ctx.Value(skipCalibrationContextKey{}).(bool); gateCalibrationRunner != nil && !skipCalibration {
		calibration, calibrationErr = gates.Calibrate(h.repoRoot, gate, gateCalibrationRunner)
	}
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
	var runtimeDB *sql.DB
	if h.evidence != nil {
		runtimeDB = h.evidence.Database()
	}
	if runner == nil {
		result, err = gates.UnmeasuredGate(h.repoRoot)
	} else {
		scope := gates.Scope{Context: ctx, Root: h.repoRoot, Assets: h.scopedAssetIDs(req.Msg.GetAssetId()), DB: runtimeDB, Revision: func(libraryID, version string) (string, error) {
			return catalogcoverage.CurrentRevisionForVersion(h.repoRoot, libraryID, version)
		}}
		if prepared, ok := ctx.Value(preparedSetsContextKey{}).(map[gates.Reads]librarywalk.Set); ok {
			if registered, found := gates.Lookup(gate); found {
				scope.Set = prepared[registered.Reads]
			}
		}
		result, _, err = gates.Run(gate, scope)
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("run catalog gate %q: %w", gate, err))
	}
	if err := catalogcoverage.AnnotateFindings(h.repoRoot, gate, &result); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("annotate catalog gate %q: %w", gate, err))
	}
	if gate == "version-liveness" && h.evidence != nil && h.evidence.Database() != nil {
		items, planHash, planErr := versionledger.NewRepository(h.evidence.Database(), filepath.Join(h.repoRoot, "scenarios", "react-component-library", "library")).PlanCleanup(ctx, versionledger.CleanupScope{})
		if planErr != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("compute version retirement plan: %w", planErr))
		}
		eligible := 0
		for _, item := range items {
			if item.Eligible {
				eligible++
			}
		}
		result.InformationalFindings = append(result.InformationalFindings, gates.Finding{
			Code:        "catalog.version_retirement_plan",
			AssetID:     "__corpus__",
			Message:     fmt.Sprintf("retirement plan computed: %d safe candidate(s) out of %d; no versions applied without an explicit plan hash and confirmation", eligible, len(items)),
			Remediation: fmt.Sprintf("Review `react-component-library versions plan-cleanup --json` (plan hash %s), then apply only with the reviewed hash and explicit confirmation.", planHash),
			DocsRef:     "docs/concepts/ARCHITECTURE.md#version-lifecycle",
		})
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
			// Evidence persistence is part of the cancellable job. A client
			// disconnect must not leave a matrix running in the background just
			// to finish a durable write.
			// The gate matrix runs concurrently, while SQLite permits only a
			// bounded number of writers. Serialize the short evidence commits so
			// one gate cannot cancel its siblings through a transient lock error.
			h.evidenceMu.Lock()
			evidenceErr = h.evidence.Save(ctx, rows)
			h.evidenceMu.Unlock()
			if evidenceErr != nil {
				return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("persist evidence for gate %q: %w", gate, evidenceErr))
			}
			h.reports.invalidate()
			evidenceRowsWritten = len(rows)
			break
		}
	}
	// Severity is the gate's declared blocking flag, not a constant. Reporting
	// every finding as "error" made the non-blocking gates (graph-reconciled,
	// forced-colors, documentation, and release-process gates) indistinguishable from the
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
	response.Findings = make([]*catalogv1.GateFinding, 0, len(result.Findings)+len(result.InformationalFindings)+1)
	response.RunnerErrors = make([]*catalogv1.GateFinding, 0, len(result.RunnerError))
	response.EvidenceRowsWritten = int32(evidenceRowsWritten)
	if nonDiscriminating {
		response.Findings = append(response.Findings, &catalogv1.GateFinding{Code: "catalog.gate_non_discriminating", Message: "gate calibration passed without detecting its planted-error fixture; corpus verdict was quarantined as unmeasured", Severity: "error", Remediation: "Repair the gate runner or its calibration fixture, then rerun catalog gates --calibration-only. A green corpus result is not evidence until the named fixture fails.", DocsRef: "docs/internal/TESTING.md", Scope: catalogv1.FindingScope_FINDING_SCOPE_CORPUS, Blocking: true, Owner: "scenarios/react-component-library/catalog", SeverityClass: catalogv1.FindingSeverity_FINDING_SEVERITY_BLOCKING})
	}
	for _, finding := range result.Findings {
		response.Findings = append(response.Findings, &catalogv1.GateFinding{
			Code:           finding.Code,
			Message:        finding.Message,
			AssetId:        finding.AssetID,
			Severity:       severity,
			File:           finding.File,
			Line:           int32(finding.Line),
			Remediation:    finding.Remediation,
			DocsRef:        finding.DocsRef,
			RuleSource:     string(finding.RuleSource),
			RuleDeclaredIn: finding.RuleDeclaredIn,
			CatalogId:      findingCatalogID(finding),
			LibraryId:      finding.LibraryID,
			Scope:          findingScope(finding),
			Blocking:       severity == "error",
			Owner:          finding.Owner,
			SeverityClass:  findingSeverity(severity),
		})
	}
	for _, finding := range result.InformationalFindings {
		response.Findings = append(response.Findings, &catalogv1.GateFinding{
			Code:           finding.Code,
			Message:        finding.Message,
			AssetId:        finding.AssetID,
			Severity:       "info",
			File:           finding.File,
			Line:           int32(finding.Line),
			Remediation:    finding.Remediation,
			DocsRef:        finding.DocsRef,
			RuleSource:     string(finding.RuleSource),
			RuleDeclaredIn: finding.RuleDeclaredIn,
			CatalogId:      findingCatalogID(finding),
			LibraryId:      finding.LibraryID,
			Scope:          findingScope(finding),
			Blocking:       false,
			Owner:          finding.Owner,
			SeverityClass:  catalogv1.FindingSeverity_FINDING_SEVERITY_INFO,
		})
	}
	for _, finding := range result.RunnerError {
		response.RunnerErrors = append(response.RunnerErrors, &catalogv1.GateFinding{Code: finding.Code, Message: finding.Message, AssetId: finding.AssetID, Severity: "error", File: finding.File, Line: int32(finding.Line), Remediation: finding.Remediation, DocsRef: finding.DocsRef, RuleSource: string(finding.RuleSource), RuleDeclaredIn: finding.RuleDeclaredIn, CatalogId: findingCatalogID(finding), LibraryId: finding.LibraryID, Scope: findingScope(finding), Blocking: true, Owner: finding.Owner, SeverityClass: catalogv1.FindingSeverity_FINDING_SEVERITY_BLOCKING})
	}
	if assetID := strings.TrimSpace(req.Msg.GetAssetId()); assetID != "" {
		filtered := response.Findings[:0]
		for _, finding := range response.Findings {
			if finding.GetAssetId() == assetID || finding.GetAssetId() == strings.TrimPrefix(assetID, "react-component-library:") {
				filtered = append(filtered, finding)
			}
		}
		response.Findings = filtered
		filteredErrors := response.RunnerErrors[:0]
		for _, finding := range response.RunnerErrors {
			if finding.GetAssetId() == assetID || finding.GetAssetId() == strings.TrimPrefix(assetID, "react-component-library:") {
				filteredErrors = append(filteredErrors, finding)
			}
		}
		response.RunnerErrors = filteredErrors
	}
	return connect.NewResponse(response), nil
}

func assetSetSlice(assetID string) map[string]struct{} {
	if strings.TrimSpace(assetID) == "" {
		return nil
	}
	return map[string]struct{}{assetID: {}}
}

func (h *handler) scopedAssetIDs(assetID string) []string {
	assetID = strings.TrimSpace(assetID)
	if assetID == "" {
		return nil
	}
	index, err := h.graph()
	if err != nil {
		return []string{assetID}
	}
	closure, err := index.Closure(assetID)
	if err != nil {
		return []string{assetID}
	}
	ids := make([]string, 0, len(closure)+1)
	ids = append(ids, assetID)
	for _, node := range closure {
		ids = append(ids, node.ID)
	}
	return ids
}

func (h *handler) gateRunner(gate string) gates.GateRunner {
	if gate == "released-version-immutable" && h.evidence != nil && h.evidence.Database() != nil {
		return func(scope gates.Scope) (gates.Result, error) {
			scope.DB = h.evidence.Database()
			return gates.ValidateReleasedVersionImmutable(scope)
		}
	}
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
		calibration, err := gates.Calibrate(h.repoRoot, definition.ID, gates.GateRunnerFor(definition.ID))
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
	response.Run = readinessRun(h.repoRoot)
	if configData, configErr := os.ReadFile(filepath.Join(h.repoRoot, "scenarios", "react-component-library", "catalog", "config.json")); configErr == nil {
		var config readinessConfigFile
		if json.Unmarshal(configData, &config) == nil {
			response.Config = readinessConfigProjection(config, report)
			h.quarantineMu.RLock()
			response.Config.QuarantinedGates = int32(len(h.quarantined))
			h.quarantineMu.RUnlock()
		}
	}
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
		AssetID      string   `json:"assetId"`
		Version      string   `json:"version"`
		StoryID      string   `json:"storyId"`
		StoryIDs     []string `json:"storyIds,omitempty"`
		ArtifactKind string   `json:"artifactKind,omitempty"`
		Result       string   `json:"result"`
		Report       any      `json:"report"`
	}
	rows := make([]captureRow, 0)
	// A bounded --all run writes one durable manifest across requests. Starting
	// at offset zero intentionally begins a fresh bounded capture; later batches
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
