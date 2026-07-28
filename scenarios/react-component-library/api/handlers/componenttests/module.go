// Package componenttests exposes durable catalog-test reports over Connect.
package componenttests

import (
	"context"
	"database/sql"
	"errors"
	"log"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	componenttestsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/componenttests"
	componenttestsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/componenttests/componenttests_v1connect"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	"react-component-library/internal/components"
	domain "react-component-library/internal/componenttests"
	"react-component-library/internal/module"
)

func Module(db *sql.DB, assets components.Service, sourceRoot string, logger *log.Logger) module.Module {
	return ModuleWithExecutor(db, assets, sourceRoot, domain.NewChromeHarnessExecutor(), logger)
}

// ModuleWithExecutor keeps the browser boundary explicit for module tests;
// production always supplies the Chrome-backed executor above.
func ModuleWithExecutor(db *sql.DB, assets components.Service, sourceRoot string, executor domain.StoryExecutor, logger *log.Logger) module.Module {
	svc := domain.NewService(domain.Runner{Assets: assets, Stories: assets, Executor: executor}, domain.NewSQLiteRepository(db))
	path, handler := componenttestsconnect.NewComponentTestsServiceHandler(&connectHandler{service: svc, logger: logger})
	sharedPath, shared := scenariovalidationconnect.NewScenarioValidationServiceHandler(&sharedHandler{service: svc, assets: assets, logger: logger})
	return module.Module{Name: "component-tests", Mount: func(r *mux.Router) {
		connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler})
		connectx.RegisterServices(r, connectx.ServiceMount{Path: sharedPath, Handler: shared})
	}, Endpoints: Endpoints}
}

type connectHandler struct {
	service *domain.Service
	logger  *log.Logger
}

func (h *connectHandler) RunComponentTest(ctx context.Context, req *connect.Request[componenttestsv1.RunComponentTestRequest]) (*connect.Response[componenttestsv1.RunComponentTestResponse], error) {
	report, err := h.service.Run(ctx, domain.Request{ComponentID: req.Msg.GetComponentId(), Version: req.Msg.GetVersion(), IncludeClosure: req.Msg.GetIncludeClosure()})
	if err != nil {
		return nil, h.error(err)
	}
	return connect.NewResponse(&componenttestsv1.RunComponentTestResponse{Report: toProto(report)}), nil
}

func (h *connectHandler) RerunComponentTest(ctx context.Context, req *connect.Request[componenttestsv1.RerunComponentTestRequest]) (*connect.Response[componenttestsv1.RerunComponentTestResponse], error) {
	report, err := h.service.Rerun(ctx, req.Msg.GetReportId())
	if err != nil {
		return nil, h.error(err)
	}
	return connect.NewResponse(&componenttestsv1.RerunComponentTestResponse{Report: toProto(report)}), nil
}

func (h *connectHandler) GetComponentTestReport(ctx context.Context, req *connect.Request[componenttestsv1.GetComponentTestReportRequest]) (*connect.Response[componenttestsv1.GetComponentTestReportResponse], error) {
	report, err := h.service.Get(ctx, req.Msg.GetId())
	if err != nil {
		return nil, h.error(err)
	}
	return connect.NewResponse(&componenttestsv1.GetComponentTestReportResponse{Report: toProto(report)}), nil
}

func (h *connectHandler) ListComponentTestReports(ctx context.Context, req *connect.Request[componenttestsv1.ListComponentTestReportsRequest]) (*connect.Response[componenttestsv1.ListComponentTestReportsResponse], error) {
	reports, err := h.service.List(ctx, req.Msg.GetComponentId(), int(req.Msg.GetLimit()))
	if err != nil {
		return nil, h.error(err)
	}
	out := make([]*componenttestsv1.ComponentTestReport, 0, len(reports))
	for _, report := range reports {
		out = append(out, toProto(report))
	}
	return connect.NewResponse(&componenttestsv1.ListComponentTestReportsResponse{Reports: out}), nil
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
		out.Results = append(out.Results, &componenttestsv1.ComponentTestResult{Stage: string(result.Stage), AssetLibraryId: result.AssetLibraryID, Version: result.Version, Subject: result.Subject, Verdict: string(result.Verdict), Message: result.Message, Remediation: result.Remediation})
	}
	for _, artifact := range report.Artifacts {
		out.Artifacts = append(out.Artifacts, &componenttestsv1.ComponentTestArtifact{Kind: artifact.Kind, Label: artifact.Label, AssetLibraryId: artifact.AssetLibraryID, Version: artifact.Version, Reference: artifact.Reference})
	}
	return out
}

var Endpoints = []module.EndpointDescriptor{
	{ID: "component_tests_run", Path: componenttestsconnect.ComponentTestsServiceRunComponentTestProcedure, Method: "POST", Summary: "Run a version-pinned component or hook contract", Category: "component-tests"},
	{ID: "component_tests_rerun", Path: componenttestsconnect.ComponentTestsServiceRerunComponentTestProcedure, Method: "POST", Summary: "Rerun a durable component test report", Category: "component-tests"},
	{ID: "component_tests_get", Path: componenttestsconnect.ComponentTestsServiceGetComponentTestReportProcedure, Method: "POST", Summary: "Get a durable component test report", Category: "component-tests"},
	{ID: "component_tests_list", Path: componenttestsconnect.ComponentTestsServiceListComponentTestReportsProcedure, Method: "POST", Summary: "List durable component test reports", Category: "component-tests"},
	{ID: "component_tests_validate_scenario", Path: scenariovalidationconnect.ScenarioValidationServiceValidateScenarioProcedure, Method: "POST", Summary: "Run the catalog component-test suite for Test Genie", Category: "component-tests"},
	{ID: "component_tests_preview_fix", Path: scenariovalidationconnect.ScenarioValidationServicePreviewFixProcedure, Method: "POST", Summary: "Preview component-test fixes", Category: "component-tests"},
	{ID: "component_tests_apply_fix", Path: scenariovalidationconnect.ScenarioValidationServiceApplyFixProcedure, Method: "POST", Summary: "Apply component-test fixes", Category: "component-tests"},
}

var ScenarioValidationProtoFile = scenariovalidationv1.File_scenario_validation_v1_validation_proto
