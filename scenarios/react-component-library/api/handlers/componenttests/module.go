// Package componenttests exposes durable catalog-test reports over Connect.
package componenttests

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
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	"react-component-library/internal/adoptions"
	"react-component-library/internal/catalogcoverage"
	"react-component-library/internal/components"
	domain "react-component-library/internal/componenttests"
	"react-component-library/internal/module"

	componenttestsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/componenttests"
	componenttestsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/componenttests/componenttests_v1connect"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
)

func Module(db *sql.DB, assets components.Service, sourceRoot string, logger *log.Logger) module.Module {
	return ModuleWithExecutor(db, assets, sourceRoot, domain.NewBASCaptureExecutor(), logger)
}

// NewBASCaptureExecutor exposes the provider-owned browser boundary to the
// catalog evidence endpoint. Both component tests and catalog capture must
// use the same BAS-backed executor so they cannot drift into separate browser
// implementations.
func NewBASCaptureExecutor() domain.StoryExecutor {
	return domain.NewBASCaptureExecutor()
}

// ModuleWithGeneratedFixture wires the provider-owned generated scenario
// contract into the production Test Genie validation phase.
func ModuleWithGeneratedFixture(db *sql.DB, assets components.Service, adoptionService adoptions.Service, sourceRoot string, logger *log.Logger) module.Module {
	return ModuleWithExecutorAndFixture(db, assets, sourceRoot, domain.NewBASCaptureExecutor(), NewGeneratedFixtureValidator(adoptionService, assets, sourceRoot, logger), logger)
}

// ModuleWithExecutor keeps the browser boundary explicit for module tests;
// production always supplies the Chrome-backed executor above.
func ModuleWithExecutor(db *sql.DB, assets components.Service, sourceRoot string, executor domain.StoryExecutor, logger *log.Logger) module.Module {
	return ModuleWithExecutorAndFixture(db, assets, sourceRoot, executor, nil, logger)
}

// ModuleWithExecutorAndFixture keeps both the browser and generated-fixture
// boundaries explicit for module tests.
func ModuleWithExecutorAndFixture(db *sql.DB, assets components.Service, sourceRoot string, executor domain.StoryExecutor, fixture GeneratedFixtureValidator, logger *log.Logger) module.Module {
	svc := domain.NewService(domain.Runner{Assets: assets, Stories: assets, Executor: executor}, domain.NewSQLiteRepository(db))
	path, handler := componenttestsconnect.NewComponentTestsServiceHandler(&connectHandler{service: svc, assets: assets, logger: logger, evidence: catalogcoverage.NewEvidenceStore(db), sourceRoot: sourceRoot, sweeps: domain.NewSQLiteSweepRepository(db)})
	sharedPath, shared := scenariovalidationconnect.NewScenarioValidationServiceHandler(&sharedHandler{service: svc, assets: assets, sourceRoot: sourceRoot, logger: logger, evidence: catalogcoverage.NewEvidenceStore(db), fixture: fixture})
	return module.Module{Name: "component-tests", Mount: func(r *mux.Router) {
		connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler})
		connectx.RegisterServices(r, connectx.ServiceMount{Path: sharedPath, Handler: shared})
	}, Endpoints: Endpoints}
}

type connectHandler struct {
	service    *domain.Service
	assets     components.Service
	logger     *log.Logger
	evidence   *catalogcoverage.EvidenceStore
	sourceRoot string
	sweeps     domain.SweepRepository
}

func (h *connectHandler) RunComponentTest(ctx context.Context, req *connect.Request[componenttestsv1.RunComponentTestRequest]) (*connect.Response[componenttestsv1.RunComponentTestResponse], error) {
	report, err := h.service.Run(ctx, domain.Request{ComponentID: req.Msg.GetComponentId(), Version: req.Msg.GetVersion(), IncludeClosure: req.Msg.GetIncludeClosure()})
	if err != nil {
		return nil, h.error(err)
	}
	if err := h.recordContractEvidence(ctx, report); err != nil {
		return nil, h.error(err)
	}
	return connect.NewResponse(&componenttestsv1.RunComponentTestResponse{Report: toProto(report)}), nil
}

func (h *connectHandler) RerunComponentTest(ctx context.Context, req *connect.Request[componenttestsv1.RerunComponentTestRequest]) (*connect.Response[componenttestsv1.RerunComponentTestResponse], error) {
	report, err := h.service.Rerun(ctx, req.Msg.GetReportId())
	if err != nil {
		return nil, h.error(err)
	}
	if err := h.recordContractEvidence(ctx, report); err != nil {
		return nil, h.error(err)
	}
	return connect.NewResponse(&componenttestsv1.RerunComponentTestResponse{Report: toProto(report)}), nil
}

// recordContractEvidence is the narrow bridge between the durable browser
// contract runner and catalog maturity. A declared story result can support
// unit/interaction evidence; it cannot silently imply visual, accessibility,
// responsive, or production evidence, which have their own capture paths.
func (h *connectHandler) recordContractEvidence(ctx context.Context, report domain.Report) error {
	return recordContractEvidence(ctx, h.evidence, h.sourceRoot, report)
}

func recordContractEvidence(ctx context.Context, evidenceStore *catalogcoverage.EvidenceStore, sourceRoot string, report domain.Report) error {
	if evidenceStore == nil || strings.TrimSpace(sourceRoot) == "" {
		return nil
	}
	implementations, err := catalogcoverage.LoadImplementations(sourceRoot)
	if err != nil {
		return fmt.Errorf("load implementations for component-test evidence: %w", err)
	}
	byLibraryID := make(map[string]string, len(implementations))
	for _, implementation := range implementations {
		if implementation.CatalogID == "" {
			continue
		}
		byLibraryID["react-component-library:"+implementation.Name] = implementation.CatalogID
	}
	assets, err := catalogcoverage.LoadCatalog(filepath.Join(filepath.Dir(sourceRoot), "catalog"))
	if err != nil {
		return fmt.Errorf("load catalog for component-test evidence: %w", err)
	}
	targetByID := make(map[string]string, len(assets))
	for _, asset := range assets {
		target := "react-vite"
		if len(asset.Targets) > 0 && strings.TrimSpace(asset.Targets[0]) != "" {
			target = asset.Targets[0]
		}
		targetByID[asset.ID] = target
	}
	type aggregate struct {
		passed  bool
		blocked bool
		seen    bool
		version string
	}
	aggregates := map[string]*aggregate{}
	for _, result := range report.Results {
		if result.Stage != domain.StageDeclared {
			continue
		}
		catalogID := byLibraryID[result.AssetLibraryID]
		if catalogID == "" || targetByID[catalogID] == "" {
			continue
		}
		item := aggregates[catalogID]
		if item == nil {
			item = &aggregate{passed: true, version: result.Version}
			aggregates[catalogID] = item
		}
		item.seen = true
		if result.Verdict != domain.VerdictPassed {
			item.passed = false
		}
		if result.Verdict == domain.VerdictBlocked {
			item.blocked = true
		}
	}
	evidence := make([]catalogcoverage.GateEvidence, 0, len(aggregates)*2)
	for catalogID, item := range aggregates {
		if !item.seen {
			continue
		}
		result := "fail"
		if item.blocked {
			result = "skipped"
		} else if item.passed {
			result = "pass"
		}
		revision, err := catalogcoverage.CurrentRevision(sourceRoot, catalogID)
		if err != nil {
			return fmt.Errorf("compute component-test evidence revision for %s: %w", catalogID, err)
		}
		for _, gate := range []string{"unit", "interaction"} {
			evidence = append(evidence, catalogcoverage.GateEvidence{AssetID: catalogID, Target: targetByID[catalogID], Version: item.version, Gate: gate, Result: result, SourceRevision: revision})
		}
	}
	return evidenceStore.Save(ctx, evidence)
}

func (h *connectHandler) GetComponentTestReport(ctx context.Context, req *connect.Request[componenttestsv1.GetComponentTestReportRequest]) (*connect.Response[componenttestsv1.GetComponentTestReportResponse], error) {
	report, err := h.service.Get(ctx, req.Msg.GetId())
	if err != nil {
		return nil, h.error(err)
	}
	return connect.NewResponse(&componenttestsv1.GetComponentTestReportResponse{Report: toProto(report)}), nil
}

func (h *connectHandler) ListComponentTestReports(ctx context.Context, req *connect.Request[componenttestsv1.ListComponentTestReportsRequest]) (*connect.Response[componenttestsv1.ListComponentTestReportsResponse], error) {
	reports, err := h.service.List(ctx, req.Msg.GetComponentId(), req.Msg.GetVersion(), int(req.Msg.GetLimit()))
	if err != nil {
		return nil, h.error(err)
	}
	out := make([]*componenttestsv1.ComponentTestReport, 0, len(reports))
	for _, report := range reports {
		out = append(out, toProto(report))
	}
	return connect.NewResponse(&componenttestsv1.ListComponentTestReportsResponse{Reports: out}), nil
}

func (h *connectHandler) SweepComponentTests(ctx context.Context, req *connect.Request[componenttestsv1.SweepComponentTestsRequest]) (*connect.Response[componenttestsv1.SweepComponentTestsResponse], error) {
	if h.assets == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("component test sweep is not wired to the component registry"))
	}
	var sweep domain.Sweep
	if h.sweeps != nil {
		if probeErr := domain.NewBASCaptureExecutor().Probe(ctx); probeErr != nil {
			return nil, h.error(fmt.Errorf("refusing to start component sweep: %w", probeErr))
		}
		filter := strings.TrimSpace(req.Msg.GetComponentId())
		if req.Msg.GetResume() {
			resumed, resumeErr := h.sweeps.LatestOpen(ctx, filter)
			switch {
			case resumeErr == nil:
				sweep = resumed
			case errors.Is(resumeErr, sql.ErrNoRows):
				sweep, resumeErr = h.sweeps.Start(ctx, filter, req.Msg.GetIncludeClosure(), "")
				if resumeErr != nil {
					return nil, h.error(resumeErr)
				}
			default:
				return nil, h.error(resumeErr)
			}
		} else {
			var startErr error
			sweep, startErr = h.sweeps.Start(ctx, filter, req.Msg.GetIncludeClosure(), "")
			if startErr != nil {
				return nil, h.error(startErr)
			}
		}
	}
	assets, err := h.assets.List(ctx, components.SearchQuery{Limit: 100000})
	if err != nil {
		return nil, h.error(err)
	}
	if requested := strings.TrimSpace(req.Msg.GetComponentId()); requested != "" {
		filtered := assets[:0]
		for _, asset := range assets {
			if asset.ID == requested || asset.LibraryID == requested {
				filtered = append(filtered, asset)
			}
		}
		assets = filtered
	}
	sort.Slice(assets, func(i, j int) bool { return assets[i].LibraryID < assets[j].LibraryID })
	response := &componenttestsv1.SweepComponentTestsResponse{}
	for _, asset := range assets {
		versions, versionErr := h.assets.ListVersions(ctx, asset.ID, 100000)
		if versionErr != nil {
			response.Errors = append(response.Errors, fmt.Sprintf("%s: list versions: %v", asset.LibraryID, versionErr))
			response.Complete = false
			continue
		}
		sort.Slice(versions, func(i, j int) bool { return versions[i].Version < versions[j].Version })
		for _, version := range versions {
			// Evidence is governed for authored story contracts in the live
			// library corpus. Drafts and historical database rows without a
			// source-tree story contract are authoring/index records, not
			// runnable evidence targets; attempting them makes a corpus sweep
			// wait on a preview that cannot exist and distorts freshness counts.
			if !h.sweepableVersion(version) {
				response.Skipped++
				continue
			}
			response.Planned++
			key := asset.LibraryID + "@" + version.Version
			if req.Msg.GetResume() {
				previous := sweep.Results[key]
				if previous == "" && h.sweeps != nil {
					existing, listErr := h.service.List(ctx, asset.ID, version.Version, 1)
					if listErr != nil {
						response.Errors = append(response.Errors, fmt.Sprintf("%s@%s: read prior report: %v", asset.LibraryID, version.Version, listErr))
					} else if len(existing) > 0 && existing[0].Verdict != domain.VerdictBlocked && h.storyEvidenceFresh(version) {
						previous = string(existing[0].Verdict)
						sweep.Results[key] = previous
						_ = h.sweeps.Save(ctx, sweep)
					}
				}
				if previous != "" && previous != string(domain.VerdictBlocked) {
					response.Skipped++
					continue
				}
			}
			response.Started++
			report, runErr := h.service.Run(ctx, domain.Request{ComponentID: asset.ID, Version: version.Version, IncludeClosure: req.Msg.GetIncludeClosure()})
			if runErr != nil {
				response.Blocked++
				response.Errors = append(response.Errors, fmt.Sprintf("%s@%s: %v", asset.LibraryID, version.Version, runErr))
				if h.sweeps != nil {
					sweep.Results[key] = string(domain.VerdictBlocked)
					_ = h.sweeps.Save(ctx, sweep)
				}
				continue
			}
			if evidenceErr := h.recordContractEvidence(ctx, report); evidenceErr != nil {
				response.Errors = append(response.Errors, fmt.Sprintf("%s@%s: record evidence: %v", asset.LibraryID, version.Version, evidenceErr))
			}
			response.Reports = append(response.Reports, toProto(report))
			if h.sweeps != nil {
				sweep.Results[key] = string(report.Verdict)
				_ = h.sweeps.Save(ctx, sweep)
			}
			switch report.Verdict {
			case domain.VerdictPassed:
				response.Passed++
			case domain.VerdictFailed:
				response.Failed++
			case domain.VerdictBlocked:
				response.Blocked++
			}
		}
	}
	response.Complete = len(response.Errors) == 0 && response.Blocked == 0
	if h.sweeps != nil {
		sweep.Status = domain.SweepComplete
		if response.Blocked > 0 {
			sweep.Status = domain.SweepBlocked
		} else if len(response.Errors) > 0 || response.Failed > 0 {
			sweep.Status = domain.SweepFailed
		}
		sweep.CompletedAt = time.Now().UTC()
		if saveErr := h.sweeps.Save(ctx, sweep); saveErr != nil {
			return nil, h.error(saveErr)
		}
	}
	return connect.NewResponse(response), nil
}

func (h *connectHandler) storyEvidenceFresh(version components.ComponentVersion) bool {
	storyPath, ok := h.storyContractPath(version)
	if !ok {
		return false
	}
	contract, err := os.Stat(storyPath)
	if err != nil {
		return false
	}
	reports, err := h.service.List(context.Background(), version.ComponentID, version.Version, 1)
	return err == nil && len(reports) > 0 && reports[0].CreatedAt.After(contract.ModTime())
}

func (h *connectHandler) sweepableVersion(version components.ComponentVersion) bool {
	if version.Status == components.VersionStatusDraft {
		return false
	}
	_, ok := h.storyContractPath(version)
	return ok
}

func (h *connectHandler) storyContractPath(version components.ComponentVersion) (string, bool) {
	if h.sourceRoot == "" || strings.TrimSpace(version.SourcePath) == "" {
		return "", false
	}
	path := filepath.Join(h.sourceRoot, filepath.Dir(version.SourcePath), "story.json")
	if _, err := os.Stat(path); err != nil {
		return "", false
	}
	return path, true
}

func (h *connectHandler) error(err error) error {
	var validation domain.ValidationError
	switch {
	case errors.As(err, &validation):
		return connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, sql.ErrNoRows):
		return connect.NewError(connect.CodeNotFound, err)
	default:
		if h.logger != nil {
			h.logger.Printf("component test: %v", err)
		}
		return connect.NewError(connect.CodeInternal, err)
	}
}

func toProto(report domain.Report) *componenttestsv1.ComponentTestReport {
	out := &componenttestsv1.ComponentTestReport{Id: report.ID, RootLibraryId: report.RootLibraryID, RootVersion: report.RootVersion, IncludeClosure: report.IncludeClosure, CreatedAt: timestamppb.New(report.CreatedAt), Verdict: string(report.Verdict), Results: make([]*componenttestsv1.ComponentTestResult, 0, len(report.Results)), Artifacts: make([]*componenttestsv1.ComponentTestArtifact, 0, len(report.Artifacts))}
	for _, result := range report.Results {
		protoResult := &componenttestsv1.ComponentTestResult{Stage: string(result.Stage), AssetLibraryId: result.AssetLibraryID, Version: result.Version, Subject: result.Subject, Verdict: string(result.Verdict), Message: result.Message, Remediation: result.Remediation}
		for _, evidence := range result.Evidence {
			encoded, err := json.Marshal(evidence)
			if err != nil {
				continue
			}
			protoResult.Evidence = append(protoResult.Evidence, &componenttestsv1.ComponentTestEvidence{Kind: evidence.Kind, Json: string(encoded)})
		}
		out.Results = append(out.Results, protoResult)
	}
	for _, artifact := range report.Artifacts {
		out.Artifacts = append(out.Artifacts, &componenttestsv1.ComponentTestArtifact{Kind: artifact.Kind, Label: artifact.Label, StoryId: artifact.StoryID, AssetLibraryId: artifact.AssetLibraryID, Version: artifact.Version, Reference: artifact.Reference})
	}
	return out
}

var Endpoints = []module.EndpointDescriptor{
	{ID: "validation_validate_target", Path: scenariovalidationconnect.ScenarioValidationServiceValidateTargetProcedure, Method: "POST", Summary: "Validate a first-class repository target", Category: "validation"},
	{ID: "component_tests_run", Path: componenttestsconnect.ComponentTestsServiceRunComponentTestProcedure, Method: "POST", Summary: "Run a version-pinned component or hook contract", Category: "component-tests"},
	{ID: "component_tests_rerun", Path: componenttestsconnect.ComponentTestsServiceRerunComponentTestProcedure, Method: "POST", Summary: "Rerun a durable component test report", Category: "component-tests"},
	{ID: "component_tests_get", Path: componenttestsconnect.ComponentTestsServiceGetComponentTestReportProcedure, Method: "POST", Summary: "Get a durable component test report", Category: "component-tests"},
	{ID: "component_tests_list", Path: componenttestsconnect.ComponentTestsServiceListComponentTestReportsProcedure, Method: "POST", Summary: "List durable component test reports", Category: "component-tests"},
	{ID: "component_tests_sweep", Path: componenttestsconnect.ComponentTestsServiceSweepComponentTestsProcedure, Method: "POST", Summary: "Run the resumable full-corpus component-test sweep", Category: "component-tests"},
	{ID: "component_tests_validate_scenario", Path: scenariovalidationconnect.ScenarioValidationServiceValidateScenarioProcedure, Method: "POST", Summary: "Run the catalog component-test suite for Test Genie", Category: "component-tests"},
	{ID: "component_tests_preview_fix", Path: scenariovalidationconnect.ScenarioValidationServicePreviewFixProcedure, Method: "POST", Summary: "Preview component-test fixes", Category: "component-tests"},
	{ID: "component_tests_apply_fix", Path: scenariovalidationconnect.ScenarioValidationServiceApplyFixProcedure, Method: "POST", Summary: "Apply component-test fixes", Category: "component-tests"},
}

var ScenarioValidationProtoFile = scenariovalidationv1.File_scenario_validation_v1_validation_proto
