package validation

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"connectrpc.com/connect"

	localassessment "experience-manager/internal/assessment"
	"experience-manager/internal/checks"
	"experience-manager/internal/spec"

	maturity "github.com/vrooli/maturity-go/assessment"
	"github.com/vrooli/maturity-go/autofix"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	contractv1 "github.com/vrooli/vrooli/packages/proto/gen/go/experience-manager/v1/contract"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
)

// Engine is the validation engine seam implemented by internal/checks.Engine.
type Engine interface {
	ValidateScenario(ctx context.Context, scenario, path string) (spec.Report, error)
}

type Deps struct {
	Logger      *log.Logger
	Engine      Engine
	Builder     *localassessment.Builder
	Fixers      *autofix.Registry
	RepoRoot    string
	Environment *commonv1.CaptureEnvironment
}

type connectHandler struct {
	deps Deps
}

func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

// validate is the single shared pipeline wrapper for the native and delegated
// service mounts.
func (h *connectHandler) validate(ctx context.Context, scenario, path string) (spec.Report, *commonv1.MaturityAssessment, *commonv1.ExecutionMetrics, error) {
	if h.deps.Engine == nil {
		return spec.Report{}, nil, nil, connect.NewError(connect.CodeInternal, fmt.Errorf("experience validation engine is not configured"))
	}
	report, err := h.deps.Engine.ValidateScenario(ctx, scenario, path)
	if err != nil {
		return report, nil, nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	builder := h.deps.Builder
	if builder == nil {
		var buildErr error
		builder, buildErr = localassessment.NewBuilder(localassessment.DefaultSpec())
		if buildErr != nil {
			return report, nil, nil, connect.NewError(connect.CodeInternal, fmt.Errorf("build maturity mapper: %w", buildErr))
		}
	}
	assessment, err := builder.Build(report.Scenario, report.Findings)
	if err != nil {
		return report, nil, nil, connect.NewError(connect.CodeInternal, fmt.Errorf("build maturity assessment: %w", err))
	}
	return report, assessment, nil, nil
}

// ValidateScenario implements the shared ScenarioValidationService mount.
func (h *connectHandler) ValidateScenario(ctx context.Context, req *connect.Request[scenariovalidationv1.ValidateScenarioRequest]) (*connect.Response[scenariovalidationv1.ValidateScenarioResponse], error) {
	report, maturityAssessment, execMetrics, err := h.validate(ctx, req.Msg.GetScenario(), req.Msg.GetPath())
	if err != nil {
		return nil, err
	}
	native := nativeReport(report)
	resp, err := maturity.BuildValidationResponse(report.Scenario, maturityAssessment, native, execMetrics, maturity.WithValidationStatus(sharedStatus(report.Findings)))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("build shared validation response: %w", err))
	}
	return connect.NewResponse(resp), nil
}

// PreviewFix implements the shared Fix RPC (dry-run).
func (h *connectHandler) PreviewFix(context.Context, *connect.Request[scenariovalidationv1.FixRequest]) (*connect.Response[scenariovalidationv1.FixResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("experience autofix registry is not implemented yet"))
}

// ApplyFix implements the shared Fix RPC (writes).
func (h *connectHandler) ApplyFix(context.Context, *connect.Request[scenariovalidationv1.FixRequest]) (*connect.Response[scenariovalidationv1.FixResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("experience autofix registry is not implemented yet"))
}

func (h *connectHandler) validateNative(ctx context.Context, scenario, path string) (*contractv1.ValidateScenarioResponse, error) {
	report, maturityAssessment, _, err := h.validate(ctx, scenario, path)
	if err != nil {
		return nil, err
	}
	return &contractv1.ValidateScenarioResponse{
		Scenario:       report.Scenario,
		Status:         nativeStatus(report.Findings),
		TargetPath:     report.TargetPath,
		DegradedReason: report.DegradedReason,
		Report:         nativeReport(report),
		Assessment:     maturityAssessment,
	}, nil
}

func nativeStatus(findings []spec.Finding) string {
	for _, finding := range findings {
		if finding.Severity == spec.SeverityError {
			return "FAILED"
		}
	}
	return "PASSED"
}

func nativeReport(report spec.Report) *contractv1.ExperienceContractReport {
	out := &contractv1.ExperienceContractReport{}
	for _, f := range checks.CapSeverity(report.Findings) {
		out.Findings = append(out.Findings, &contractv1.ExperienceFinding{
			Code:        f.Code,
			Severity:    f.Severity,
			Message:     f.Message,
			Location:    firstLocation(f.Locations),
			Remediation: f.Suggestion,
		})
	}
	return out
}

func firstLocation(locations []string) string {
	if len(locations) == 0 {
		return ""
	}
	return locations[0]
}

func sharedStatus(findings []spec.Finding) scenariovalidationv1.ValidationStatus {
	if strings.EqualFold(strings.TrimSpace(os.Getenv("EXPERIENCE_ALIGNMENT_GATE")), "strict") {
		return scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_UNSPECIFIED
	}
	for _, finding := range findings {
		if finding.Severity == spec.SeverityError {
			return scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED
		}
	}
	return scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_UNSPECIFIED
}
