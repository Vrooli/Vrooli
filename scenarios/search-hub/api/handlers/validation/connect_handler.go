package validation

import (
	"context"
	"fmt"
	"log"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/metrics"
	"github.com/vrooli/maturity-go/assessment"
	"github.com/vrooli/maturity-go/autofix"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	"google.golang.org/protobuf/types/known/structpb"

	internalvalidation "search-hub/internal/validation"
)

type Deps struct {
	Logger       *log.Logger
	Validator    *internalvalidation.Service
	MaturitySpec *assessment.Spec
}

type connectHandler struct {
	deps Deps
}

func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	if d.Validator == nil {
		d.Validator = internalvalidation.New("")
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) ValidateScenario(ctx context.Context, req *connect.Request[scenariovalidationv1.ValidateScenarioRequest]) (*connect.Response[scenariovalidationv1.ValidateScenarioResponse], error) {
	collector := metrics.Start()
	report, err := h.deps.Validator.ValidateScenarioWithOptions(ctx, req.Msg.GetScenario(), req.Msg.GetPath(), internalvalidation.Options{
		IncludeEvals: req.Msg.GetIncludeExecution(),
	})
	if err != nil {
		collector.Stop()
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if h.deps.MaturitySpec == nil {
		collector.Stop()
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("maturity spec is required"))
	}
	maturity, err := internalvalidation.BuildMaturityAssessment(report.Scenario, report.Findings, *h.deps.MaturitySpec)
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

func (h *connectHandler) PreviewFix(_ context.Context, req *connect.Request[scenariovalidationv1.FixRequest]) (*connect.Response[scenariovalidationv1.FixResponse], error) {
	scenario, candidates, err := h.deps.Validator.PreviewFixes(req.Msg.GetScenario(), req.Msg.GetPath(), req.Msg.GetRuleIds())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(autofix.BuildFixResponse(scenario, false, candidates)), nil
}

func (h *connectHandler) ApplyFix(_ context.Context, req *connect.Request[scenariovalidationv1.FixRequest]) (*connect.Response[scenariovalidationv1.FixResponse], error) {
	scenario, candidates, err := h.deps.Validator.ApplyFixes(req.Msg.GetScenario(), req.Msg.GetPath(), req.Msg.GetRuleIds())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(autofix.BuildFixResponse(scenario, true, candidates)), nil
}

func nativeDetail(report internalvalidation.Report) (*structpb.Struct, error) {
	findings := make([]any, 0, len(report.Findings))
	for _, f := range report.Findings {
		findings = append(findings, map[string]any{
			"code":        f.Code,
			"severity":    string(f.Severity),
			"title":       f.Title,
			"message":     f.Message,
			"location":    f.Location,
			"remediation": f.Remediation,
			"advisory":    f.Severity != internalvalidation.SeverityError,
			"gating":      f.Severity == internalvalidation.SeverityError,
		})
	}
	return structpb.NewStruct(map[string]any{
		"scenario":      report.Scenario,
		"status":        report.Summary.Status(),
		"eval_evidence": evalEvidenceDetail(report.EvalEvidence),
		"summary": map[string]any{
			"providers": report.Summary.Providers,
			"errors":    report.Summary.Errors,
			"warnings":  report.Summary.Warnings,
		},
		"findings": findings,
	})
}

func evalEvidenceDetail(evidence []internalvalidation.EvalEvidence) []any {
	out := make([]any, 0, len(evidence))
	for _, e := range evidence {
		item := map[string]any{
			"suite_id":      e.SuiteID,
			"freshness":     e.Freshness,
			"corpus_status": e.CorpusStatus,
		}
		if e.LastRunID != "" {
			item["last_run_id"] = e.LastRunID
		}
		if e.LastRunAt != "" {
			item["last_run_at"] = e.LastRunAt
		}
		if e.FailureReason != "" {
			item["failure_reason"] = e.FailureReason
		}
		if e.GradeablePositives > 0 {
			item["recall"] = e.Recall
			item["recall_target"] = e.RecallTarget
		}
		// These are aggregate counts, not optional annotations. Emit zero
		// explicitly so JSON consumers can distinguish "none met" from a
		// projection that forgot the field.
		item["gradeable_positives"] = e.GradeablePositives
		item["met_cases"] = e.MetCases
		item["below_cases"] = e.BelowCases
		if e.LatencyP95Ms > 0 {
			item["latency_p95_ms"] = e.LatencyP95Ms
		}
		out = append(out, item)
	}
	return out
}
