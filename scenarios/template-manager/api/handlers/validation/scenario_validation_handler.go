package validation

import (
	"context"
	"errors"
	"fmt"
	"log"

	"connectrpc.com/connect"
	"github.com/vrooli/vrooli/scenarios/template-manager/api/internal/templatevalidation"

	"github.com/vrooli/api-core/metrics"
	"github.com/vrooli/maturity-go/assessment"
	"github.com/vrooli/maturity-go/autofix"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

type ScenarioValidator interface {
	ValidateScenario(ctx context.Context, scenario, explicitPath string) (templatevalidation.Report, error)
}

type ScenarioFixer interface {
	Preview(root string, ruleIDs []string) ([]autofix.Candidate, error)
	Apply(root string, ruleIDs []string) ([]autofix.Candidate, error)
}

type ScenarioValidationDeps struct {
	Logger       *log.Logger
	Validator    ScenarioValidator
	Fixers       ScenarioFixer
	MaturitySpec *assessment.Spec
	Environment  *commonv1.CaptureEnvironment
}

type scenarioValidationHandler struct {
	deps ScenarioValidationDeps
}

func NewScenarioValidationHandler(d ScenarioValidationDeps) *scenarioValidationHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &scenarioValidationHandler{deps: d}
}

func (h *scenarioValidationHandler) ValidateScenario(ctx context.Context, req *connect.Request[scenariovalidationv1.ValidateScenarioRequest]) (*connect.Response[scenariovalidationv1.ValidateScenarioResponse], error) {
	if h.deps.Validator == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("template scenario validation is not wired"))
	}
	collector := metrics.Start(metrics.WithEnvironment(h.deps.Environment))
	report, err := h.deps.Validator.ValidateScenario(ctx, req.Msg.GetScenario(), req.Msg.GetPath())
	if err != nil {
		collector.Stop()
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	maturityAssessment, err := buildMaturityAssessment(report, h.deps.MaturitySpec)
	if err != nil {
		collector.Stop()
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("build maturity assessment: %w", err))
	}
	nativeDetail, err := buildNativeDetail(report)
	if err != nil {
		collector.Stop()
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("build native detail: %w", err))
	}
	resp, err := assessment.BuildValidationResponse(report.Scenario, maturityAssessment, nativeDetail, collector.Stop())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(resp), nil
}

func (h *scenarioValidationHandler) PreviewFix(ctx context.Context, req *connect.Request[scenariovalidationv1.FixRequest]) (*connect.Response[scenariovalidationv1.FixResponse], error) {
	return h.fix(ctx, req.Msg, false)
}

func (h *scenarioValidationHandler) ApplyFix(ctx context.Context, req *connect.Request[scenariovalidationv1.FixRequest]) (*connect.Response[scenariovalidationv1.FixResponse], error) {
	return h.fix(ctx, req.Msg, true)
}

func (h *scenarioValidationHandler) fix(ctx context.Context, req *scenariovalidationv1.FixRequest, apply bool) (*connect.Response[scenariovalidationv1.FixResponse], error) {
	if h.deps.Validator == nil || h.deps.Fixers == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("template fix registry is not wired"))
	}
	report, err := h.deps.Validator.ValidateScenario(ctx, req.GetScenario(), req.GetPath())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	var candidates []autofix.Candidate
	if apply {
		candidates, err = h.deps.Fixers.Apply(report.RootPath, req.GetRuleIds())
	} else {
		candidates, err = h.deps.Fixers.Preview(report.RootPath, req.GetRuleIds())
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	resp := &scenariovalidationv1.FixResponse{Scenario: report.Scenario, Applied: apply}
	for _, candidate := range candidates {
		resp.Candidates = append(resp.Candidates, &scenariovalidationv1.FixCandidate{
			RuleId:      candidate.RuleID,
			FilePath:    candidate.FilePath,
			Description: candidate.Description,
			Before:      candidate.Before,
			After:       candidate.After,
			Applied:     apply,
		})
	}
	if len(resp.Candidates) == 0 {
		resp.Messages = append(resp.Messages, "no deterministic template fixes available")
	}
	return connect.NewResponse(resp), nil
}

func buildMaturityAssessment(report templatevalidation.Report, spec *assessment.Spec) (*commonv1.MaturityAssessment, error) {
	if spec == nil {
		return nil, errors.New("maturity spec is required")
	}
	findings := make([]assessment.Finding, 0, len(report.Findings))
	for _, finding := range report.Findings {
		findings = append(findings, assessment.Finding{
			Code:             finding.Code,
			Severity:         string(finding.Severity),
			Title:            finding.Title,
			Message:          finding.Message,
			Location:         finding.Location,
			Remediation:      finding.Remediation,
			Phase:            spec.Phase,
			AutofixAvailable: finding.Autofix,
		})
	}
	return assessment.BuildProtoAssessment(assessment.BuildInput{
		Scenario: report.Scenario,
		Spec:     *spec,
		Findings: findings,
	})
}

func buildNativeDetail(report templatevalidation.Report) (*structpb.Struct, error) {
	return structpb.NewStruct(map[string]any{
		"scenario":  report.Scenario,
		"root_path": report.RootPath,
		"provenance": map[string]any{
			"template_id":      report.Provenance.TemplateID,
			"template_version": report.Provenance.TemplateVersion,
			"generated_at":     report.Provenance.GeneratedAt,
			"manifest_sha":     report.Provenance.ManifestSHA,
			"content_sha":      report.Provenance.ContentSHA,
			"adopted":          report.Provenance.Adopted,
		},
		"finding_count": len(report.Findings),
	})
}
