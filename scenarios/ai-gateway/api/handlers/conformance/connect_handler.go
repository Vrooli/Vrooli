package conformance

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/metrics"
	"github.com/vrooli/maturity-go/assessment"
	conformancev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/conformance"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	"google.golang.org/protobuf/types/known/structpb"

	internalconformance "ai-gateway/internal/conformance"
)

type Deps struct {
	Scanner      *internalconformance.Scanner
	MaturitySpec *assessment.Spec
}

type connectHandler struct {
	scanner *internalconformance.Scanner
	deps    Deps
}

func NewConnectHandler(d Deps) *connectHandler {
	if d.Scanner == nil {
		d.Scanner = internalconformance.NewScanner()
	}
	return &connectHandler{scanner: d.Scanner, deps: d}
}

func (h *connectHandler) ScanScenario(ctx context.Context, req *connect.Request[conformancev1.ScanScenarioRequest]) (*connect.Response[conformancev1.ScanScenarioResponse], error) {
	report, err := h.scanner.Scan(ctx, internalconformance.ScanRequest{
		Scenario: req.Msg.GetScenario(),
		Path:     req.Msg.GetPath(),
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&conformancev1.ScanScenarioResponse{
		Scenario:        report.Scenario,
		MaturityLevel:   report.MaturityLevel,
		Findings:        report.Findings,
		Recommendations: report.Recommendations,
	}), nil
}

func (h *connectHandler) ValidateScenario(ctx context.Context, req *connect.Request[scenariovalidationv1.ValidateScenarioRequest]) (*connect.Response[scenariovalidationv1.ValidateScenarioResponse], error) {
	collector := metrics.Start()
	report, err := h.scanner.Scan(ctx, internalconformance.ScanRequest{
		Scenario: req.Msg.GetScenario(),
		Path:     req.Msg.GetPath(),
	})
	if err != nil {
		collector.Stop()
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if h.maturitySpec() == nil {
		collector.Stop()
		return nil, connect.NewError(connect.CodeInternal, errors.New("maturity spec is required"))
	}
	maturity, err := internalconformance.BuildMaturityAssessment(report, *h.maturitySpec())
	if err != nil {
		collector.Stop()
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("build maturity assessment: %w", err))
	}
	native, err := nativeDetail(report)
	if err != nil {
		collector.Stop()
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("build native detail: %w", err))
	}
	resp, err := assessment.BuildValidationResponse(report.Scenario, maturity, native, collector.Stop())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("build shared validation response: %w", err))
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) ValidateTarget(ctx context.Context, req *connect.Request[scenariovalidationv1.ValidateTargetRequest]) (*connect.Response[scenariovalidationv1.ValidateTargetResponse], error) {
	target := req.Msg.GetTarget()
	if target == nil || target.GetId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("target is required"))
	}
	path := req.Msg.GetPath()
	if path == "" {
		path = target.GetRoot()
	}
	if target.GetKind().String() != "VALIDATION_TARGET_KIND_PROJECT" {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("ai-gateway target validation is only defined for project targets"))
	}
	collector := metrics.Start()
	report, err := h.scanner.ScanProject(ctx, path)
	if err != nil {
		collector.Stop()
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if h.maturitySpec() == nil {
		collector.Stop()
		return nil, connect.NewError(connect.CodeInternal, errors.New("maturity spec is required"))
	}
	maturity, err := internalconformance.BuildMaturityAssessment(report, *h.maturitySpec())
	if err != nil {
		collector.Stop()
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("build target maturity assessment: %w", err))
	}
	native, err := nativeDetail(report)
	if err != nil {
		collector.Stop()
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	shared, err := assessment.BuildValidationResponse(report.Scenario, maturity, native, collector.Stop())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("build target validation response: %w", err))
	}
	return connect.NewResponse(&scenariovalidationv1.ValidateTargetResponse{Target: target, Status: shared.GetStatus(), Assessment: shared.GetAssessment(), NativeDetail: shared.GetNativeDetail(), Metrics: shared.GetMetrics()}), nil
}

func (h *connectHandler) PreviewFix(_ context.Context, req *connect.Request[scenariovalidationv1.FixRequest]) (*connect.Response[scenariovalidationv1.FixResponse], error) {
	return connect.NewResponse(&scenariovalidationv1.FixResponse{
		Scenario: req.Msg.GetScenario(),
		Applied:  false,
		Messages: []string{"AI conformance has no deterministic source rewrite fixers; use the migration guidance in ValidateScenario/ScanScenario."},
	}), nil
}

func (h *connectHandler) ApplyFix(_ context.Context, req *connect.Request[scenariovalidationv1.FixRequest]) (*connect.Response[scenariovalidationv1.FixResponse], error) {
	return connect.NewResponse(&scenariovalidationv1.FixResponse{
		Scenario: req.Msg.GetScenario(),
		Applied:  false,
		Messages: []string{"AI conformance apply is intentionally a no-op until safe deterministic migrations exist."},
	}), nil
}

func (h *connectHandler) maturitySpec() *assessment.Spec {
	return h.deps.MaturitySpec
}

func nativeDetail(report internalconformance.ScanReport) (*structpb.Struct, error) {
	findings := make([]any, 0, len(report.Findings))
	for _, finding := range report.Findings {
		findings = append(findings, map[string]any{
			"rule_id":     finding.GetRuleId(),
			"severity":    finding.GetSeverity(),
			"path":        finding.GetPath(),
			"message":     finding.GetMessage(),
			"remediation": finding.GetRemediation(),
		})
	}
	recommendations := make([]any, 0, len(report.Recommendations))
	for _, rec := range report.Recommendations {
		recommendations = append(recommendations, rec)
	}
	return structpb.NewStruct(map[string]any{
		"scenario":        report.Scenario,
		"maturity_level":  report.MaturityLevel,
		"findings":        findings,
		"recommendations": recommendations,
	})
}
