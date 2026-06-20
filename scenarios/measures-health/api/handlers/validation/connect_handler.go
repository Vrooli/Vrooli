// Package validation is the Connect-RPC surface for the measures coverage
// validator. Scenario-level validation is served through the shared
// ScenarioValidationService; measures-health's native coverage report is packed
// into native_detail for its own UI.
package validation

import (
	"context"
	"fmt"
	"log"

	"connectrpc.com/connect"

	"github.com/vrooli/api-core/metrics"
	"github.com/vrooli/maturity-go/assessment"
	"github.com/vrooli/measures-go/manifestscan"
	"measures-health/internal/runhistory"
	internal "measures-health/internal/validation"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/measures-health/v1/validation"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
)

// Validator is the subset of internal/validation the handler depends on (a seam
// so handler tests inject a fake without touching the filesystem).
type Validator interface {
	ValidateScenario(ctx context.Context, scenario string, probe bool) (internal.Report, error)
	ListFleetCoverage(ctx context.Context, scenarios []string) ([]internal.FleetEntry, error)
}

// RunRecorder persists a measures-validation run to the validation_run history.
// Optional (nil = no persistence); production wires *runhistory.Repository. The
// write happens here at the top-level ValidateScenario RPC ONLY — never inside
// the per-scenario fleet rollup (ListFleetCoverage), which would amplify writes.
type RunRecorder interface {
	Record(ctx context.Context, run runhistory.Run) error
}

// Deps wires the Connect validation handler.
type Deps struct {
	Validator    Validator
	Recorder     RunRecorder
	Logger       *log.Logger
	MaturitySpec *assessment.Spec
	// Environment is the host CaptureEnvironment captured once at module init
	// (os/arch/cpu/mem/present-GPUs). nil is safe — the metrics collector
	// backfills os/arch/num_cpu from the stdlib.
	Environment *commonv1.CaptureEnvironment
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler constructs the handler over its deps.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) ValidateScenario(ctx context.Context, req *connect.Request[scenariovalidationv1.ValidateScenarioRequest]) (*connect.Response[scenariovalidationv1.ValidateScenarioResponse], error) {
	collector := metrics.Start(metrics.WithEnvironment(h.deps.Environment))
	rep, err := h.deps.Validator.ValidateScenario(internal.WithMetrics(ctx, collector), req.Msg.GetScenario(), req.Msg.GetIncludeExecution())
	if err != nil {
		collector.Stop()
		h.deps.Logger.Printf("validation.ValidateScenario(%q): %v", req.Msg.GetScenario(), err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	// Persist the run to the validation_run history (best-effort: a storage
	// failure must not fail the validation RPC). Top-level only — the fleet
	// rollup never reaches here, so writes are not amplified per scenario.
	if h.deps.Recorder != nil {
		errs, warns, _ := rep.Summary()
		if rerr := h.deps.Recorder.Record(ctx, runhistory.Run{
			Scenario:     rep.Scenario,
			Passed:       rep.Passed,
			ErrorCount:   errs,
			WarningCount: warns,
		}); rerr != nil {
			h.deps.Logger.Printf("validation.ValidateScenario(%q): record run history: %v", rep.Scenario, rerr)
		}
	}
	native, err := reportToProto(rep, h.deps.MaturitySpec)
	if err != nil {
		collector.Stop()
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	execMetrics := collector.Stop()
	resp, err := assessment.BuildValidationResponse(native.GetScenario(), native.GetAssessment(), native, execMetrics)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("build shared validation response: %w", err))
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) ListFleetCoverage(ctx context.Context, req *connect.Request[validationv1.ListFleetCoverageRequest]) (*connect.Response[validationv1.ListFleetCoverageResponse], error) {
	entries, err := h.deps.Validator.ListFleetCoverage(ctx, req.Msg.GetScenarios())
	if err != nil {
		h.deps.Logger.Printf("validation.ListFleetCoverage: %v", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	resp := &validationv1.ListFleetCoverageResponse{Entries: make([]*validationv1.FleetEntry, 0, len(entries))}
	for _, e := range entries {
		resp.Entries = append(resp.Entries, &validationv1.FleetEntry{
			Scenario:     e.Scenario,
			Passed:       e.Passed,
			Expected:     int32(e.Expected),
			Covered:      int32(e.Covered),
			Waived:       int32(e.Waived),
			Uncovered:    int32(e.Uncovered),
			WorstTier:    tierToProto(e.WorstTier),
			MeasureCount: int32(e.MeasureCount),
		})
	}
	return connect.NewResponse(resp), nil
}

func reportToProto(rep internal.Report, spec *assessment.Spec) (*validationv1.ScenarioCoverageReport, error) {
	errs, warns, infos := rep.Summary()
	out := &validationv1.ScenarioCoverageReport{
		Scenario:        rep.Scenario,
		Passed:          rep.Passed,
		Summary:         &validationv1.Summary{Errors: int32(errs), Warnings: int32(warns), Infos: int32(infos)},
		SkippedScanners: rep.SkippedScanners,
		Domains:         make([]*validationv1.DomainCoverage, 0, len(rep.Domains)),
		Findings:        make([]*validationv1.Finding, 0, len(rep.Findings)),
	}
	for _, d := range rep.Domains {
		dc := &validationv1.DomainCoverage{
			Domain:       d.Domain,
			Status:       statusToProto(d.Status),
			MeasureCount: int32(d.MeasureCount),
			Tier:         tierToProto(d.Tier),
			WaiverReason: d.WaiverReason,
			Note:         d.Note,
			Measures:     make([]*validationv1.MeasureSummary, 0, len(d.Measures)),
		}
		for _, m := range d.Measures {
			dc.Measures = append(dc.Measures, &validationv1.MeasureSummary{
				Name:          m.Name,
				Intent:        m.Intent,
				Tier:          tierToProto(m.Tier),
				Effect:        m.Effect,
				QuestionCount: int32(m.QuestionCount),
				ProbePassed:   m.ProbePassed,
				ProbeDetail:   m.ProbeDetail,
				TierNote:      m.TierNote,
			})
		}
		out.Domains = append(out.Domains, dc)
	}
	for _, f := range rep.Findings {
		out.Findings = append(out.Findings, &validationv1.Finding{
			RuleId:      f.RuleID,
			Severity:    severityToProto(f.Severity),
			Title:       f.Title,
			Description: f.Description,
			Remediation: f.Remediation,
			FilePath:    f.FilePath,
			Scanner:     f.Scanner,
		})
	}
	if spec != nil {
		assessment, err := buildMaturityAssessment(rep, *spec)
		if err != nil {
			return nil, fmt.Errorf("build maturity assessment: %w", err)
		}
		out.Assessment = assessment
	}
	return out, nil
}

func buildMaturityAssessment(rep internal.Report, spec assessment.Spec) (*commonv1.MaturityAssessment, error) {
	findings := make([]assessment.Finding, 0, len(rep.Findings))
	for _, f := range rep.Findings {
		findings = append(findings, assessment.Finding{
			Code:        f.RuleID,
			Severity:    severityToProto(f.Severity).String(),
			Title:       f.Title,
			Message:     f.Description,
			Location:    f.FilePath,
			Remediation: f.Remediation,
			Phase:       spec.Phase,
		})
	}
	return assessment.BuildProtoAssessment(assessment.BuildInput{
		Scenario: rep.Scenario,
		Spec:     spec,
		Findings: findings,
	})
}

func severityToProto(s internal.Severity) validationv1.Severity {
	switch s {
	case internal.SeverityError:
		return validationv1.Severity_SEVERITY_ERROR
	case internal.SeverityWarning:
		return validationv1.Severity_SEVERITY_WARNING
	default:
		return validationv1.Severity_SEVERITY_INFO
	}
}

func statusToProto(s internal.DomainStatus) validationv1.DomainStatus {
	switch s {
	case internal.StatusCovered:
		return validationv1.DomainStatus_DOMAIN_STATUS_COVERED
	case internal.StatusUncovered:
		return validationv1.DomainStatus_DOMAIN_STATUS_UNCOVERED
	case internal.StatusWaived:
		return validationv1.DomainStatus_DOMAIN_STATUS_WAIVED
	case internal.StatusNotExpected:
		return validationv1.DomainStatus_DOMAIN_STATUS_NOT_EXPECTED
	default:
		return validationv1.DomainStatus_DOMAIN_STATUS_UNSPECIFIED
	}
}

func tierToProto(t manifestscan.Tier) validationv1.Tier {
	switch t {
	case manifestscan.TierFull:
		return validationv1.Tier_TIER_FULL
	case manifestscan.TierPartial:
		return validationv1.Tier_TIER_PARTIAL
	case manifestscan.TierFallback:
		return validationv1.Tier_TIER_FALLBACK
	default:
		return validationv1.Tier_TIER_UNSPECIFIED
	}
}
