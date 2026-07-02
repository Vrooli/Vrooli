package validation

import (
	"context"
	"fmt"
	"log"
	"os"

	"connectrpc.com/connect"

	"business-health/internal/assessment"
	localautofix "business-health/internal/autofix"
	"business-health/internal/checks"
	"business-health/internal/extraction"

	"github.com/vrooli/api-core/metrics"
	maturity "github.com/vrooli/maturity-go/assessment"
	"github.com/vrooli/maturity-go/autofix"
	contractv1 "github.com/vrooli/vrooli/packages/proto/gen/go/business-health/v1/contract"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
)

// Engine is the check engine seam (implemented by internal/checks.Engine).
type Engine interface {
	ValidateScenario(ctx context.Context, scenario, path string) (checks.Report, error)
}

type Deps struct {
	Logger  *log.Logger
	Engine  Engine
	Builder *assessment.Builder
	Fixers  *autofix.Registry
	// Extractor backs the matrix/drift RPCs (the same contract source the
	// engine composes).
	Extractor extraction.Extractor
	// RepoRoot anchors target resolution and fixer application paths.
	RepoRoot string
	// Environment is the host CaptureEnvironment captured once at module
	// init. nil is safe — the metrics collector backfills os/arch/num_cpu.
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

// validate runs the engine + assessment pipeline once; both service mounts
// (shared scenario-validation and native contract) compose it.
func (h *connectHandler) validate(ctx context.Context, scenario, path string) (checks.Report, *commonv1.MaturityAssessment, *commonv1.ExecutionMetrics, error) {
	if h.deps.Engine == nil || h.deps.Builder == nil {
		return checks.Report{}, nil, nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("validation engine is not wired"))
	}
	collector := metrics.Start(metrics.WithEnvironment(h.deps.Environment))
	report, err := h.deps.Engine.ValidateScenario(checks.WithMetrics(ctx, collector), scenario, path)
	if err != nil {
		collector.Stop()
		return checks.Report{}, nil, nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	maturityAssessment, err := h.deps.Builder.Build(report.Scenario, report.Findings)
	if err != nil {
		collector.Stop()
		return checks.Report{}, nil, nil, connect.NewError(connect.CodeInternal, fmt.Errorf("build maturity assessment: %w", err))
	}
	return report, maturityAssessment, collector.Stop(), nil
}

// ValidateScenario implements the shared ScenarioValidationService mount.
func (h *connectHandler) ValidateScenario(ctx context.Context, req *connect.Request[scenariovalidationv1.ValidateScenarioRequest]) (*connect.Response[scenariovalidationv1.ValidateScenarioResponse], error) {
	report, maturityAssessment, execMetrics, err := h.validate(ctx, req.Msg.GetScenario(), req.Msg.GetPath())
	if err != nil {
		return nil, err
	}
	native := nativeReport(report)
	resp, err := maturity.BuildValidationResponse(report.Scenario, maturityAssessment, native, execMetrics)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("build shared validation response: %w", err))
	}
	return connect.NewResponse(resp), nil
}

// PreviewFix implements the shared Fix RPC (dry-run).
func (h *connectHandler) PreviewFix(ctx context.Context, req *connect.Request[scenariovalidationv1.FixRequest]) (*connect.Response[scenariovalidationv1.FixResponse], error) {
	if h.deps.Fixers == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("fixer registry is not wired"))
	}
	root, err := h.targetRoot(req.Msg.GetScenario(), req.Msg.GetPath())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	resp, err := h.deps.Fixers.PreviewFixResponse(req.Msg.GetScenario(), root, req.Msg.GetRuleIds())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(resp), nil
}

// ApplyFix implements the shared Fix RPC (writes). Rules apply one at a
// time with a re-preview between writes because several fixers can target
// the same file (see localautofix.ApplySequential).
func (h *connectHandler) ApplyFix(ctx context.Context, req *connect.Request[scenariovalidationv1.FixRequest]) (*connect.Response[scenariovalidationv1.FixResponse], error) {
	if h.deps.Fixers == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("fixer registry is not wired"))
	}
	root, err := h.targetRoot(req.Msg.GetScenario(), req.Msg.GetPath())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	candidates, err := localautofix.ApplySequential(h.deps.Fixers, root, req.Msg.GetRuleIds())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(autofix.BuildFixResponse(req.Msg.GetScenario(), true, candidates)), nil
}

// ValidateScenario (native ContractService): the rich report beside the
// shared assessment.
func (h *connectHandler) validateNative(ctx context.Context, scenario, path string) (*contractv1.ValidateScenarioResponse, error) {
	report, maturityAssessment, _, err := h.validate(ctx, scenario, path)
	if err != nil {
		return nil, err
	}
	return &contractv1.ValidateScenarioResponse{
		Scenario:       report.Scenario,
		Status:         statusLabel(maturity.DeriveValidationStatus(maturityAssessment)),
		TargetPath:     report.TargetPath,
		DegradedReason: report.DegradedReason,
		Report:         nativeReport(report),
		Assessment:     maturityAssessment,
	}, nil
}

// nativeReport builds the Phase 2 skeleton of the native detail: findings
// only. Capability rollups, matrix rows, and drift entries land with their
// domains (Phases 3, 4).
func nativeReport(report checks.Report) *contractv1.BusinessContractReport {
	out := &contractv1.BusinessContractReport{}
	for _, f := range report.Findings {
		out.Findings = append(out.Findings, &contractv1.ContractFinding{
			Code:        f.Code,
			Severity:    f.Severity,
			Message:     f.Message,
			Location:    firstLocation(f.Locations),
			Remediation: f.Suggestion,
		})
	}
	return out
}

// statusLabel renders the shared ValidationStatus enum as the provider
// verdict string ("PASSED"/"FAILED"/"DEGRADED"/"ERROR"/"SKIPPED").
func statusLabel(status scenariovalidationv1.ValidationStatus) string {
	name := status.String()
	const prefix = "VALIDATION_STATUS_"
	if len(name) > len(prefix) && name[:len(prefix)] == prefix {
		return name[len(prefix):]
	}
	return name
}

func firstLocation(locations []string) string {
	if len(locations) == 0 {
		return ""
	}
	return locations[0]
}

func (h *connectHandler) targetRoot(scenario, path string) (string, error) {
	if path != "" {
		return path, nil
	}
	if scenario == "" {
		return "", fmt.Errorf("scenario is required")
	}
	return fmt.Sprintf("%s/scenarios/%s", h.deps.RepoRoot, scenario), nil
}

// resolveTarget is targetRoot plus an existence check (matrix/drift RPCs
// need a real tree, not just a candidate path).
func (h *connectHandler) resolveTarget(scenario, path string) (string, error) {
	root, err := h.targetRoot(scenario, path)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(root); err != nil {
		return "", fmt.Errorf("target %q: %w", root, err)
	}
	return root, nil
}
