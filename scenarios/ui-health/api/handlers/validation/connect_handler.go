// Package validation hosts the shared ScenarioValidationService handler for
// ui-health. The handler delegates to the manifestvalidation service and
// returns the common scenario-validation response shape.
package validation

import (
	"context"
	"fmt"
	"log"

	"connectrpc.com/connect"

	"ui-health/internal/codefacts"
	"ui-health/internal/services/manifestvalidation"
	"ui-health/internal/uiruntime"

	"github.com/vrooli/api-core/metrics"
	"github.com/vrooli/maturity-go/assessment"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
)

// Deps wires the seams the Connect validation handler needs.
type Deps struct {
	Logger       *log.Logger
	Validator    Validator
	MaturitySpec *assessment.Spec
	// Fixer drives the deterministic auto-fix RPCs (PreviewFix/ApplyFix) and
	// supplies the per-finding AutofixAvailable signal. nil disables fixing
	// (PreviewFix/ApplyFix return Unimplemented; findings report no autofix).
	Fixer FixProvider
	// RepoRoot is the repository root used to resolve a scenario's directory for
	// fix operations and the runtime AutofixAvailable check.
	RepoRoot string
	// CodeFacts answers "does this scenario have a UI surface, and what
	// framework" via the Code Facts authority (degrading to a filesystem probe).
	// nil is safe — the handler falls back to a default client.
	CodeFacts codefacts.Describer
	// Freshness backs the static UI-bundle freshness group (the canonical
	// content-hash engine via the typed vrooli CLI). nil is safe — the handler
	// falls back to a default client.
	Freshness freshnessClient
	// Runtime runs the BAS-driven runtime/render group when execution is
	// requested (include_execution=true) and the scenario has a UI. nil disables
	// the runtime group (static checks only).
	Runtime uiruntime.Checker
	// Environment is the host CaptureEnvironment captured once at module init
	// (os/arch/cpu/mem/present-GPUs). nil is safe — the metrics collector
	// backfills os/arch/num_cpu from the stdlib.
	Environment *commonv1.CaptureEnvironment
}

// Validator is the slice of manifestvalidation.Service the handler exercises.
type Validator interface {
	ValidateScenario(ctx context.Context, scenario string) (manifestvalidation.Report, error)
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

func (h *connectHandler) ValidateScenario(ctx context.Context, req *connect.Request[scenariovalidationv1.ValidateScenarioRequest]) (*connect.Response[scenariovalidationv1.ValidateScenarioResponse], error) {
	if h.deps.Validator == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("validation.ValidateScenario: validator not wired"))
	}
	// When the caller resolved an explicit scenario path (e.g. Test Genie running
	// deep template validation against a temp scenario outside the repo
	// scenarios/ tree), thread it so the validator reads from that directory.
	validateCtx := manifestvalidation.WithScenarioPath(ctx, req.Msg.GetPath())
	collector := metrics.Start(metrics.WithEnvironment(h.deps.Environment))
	report, err := h.deps.Validator.ValidateScenario(manifestvalidation.WithMetrics(validateCtx, collector), req.Msg.GetScenario())
	if err != nil {
		collector.Stop()
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	// Compose the static UI-interop check group into the same report (ui-health
	// is the single authority for these rules), then enrich every finding with
	// its fix classification and the live AutofixAvailable signal before the
	// assessment is built, so the maturity finding and the conformance rollup
	// carry an honest autofix flag.
	root := h.resolveScenarioRoot(req.Msg.GetScenario(), req.Msg.GetPath())
	report.Findings = append(report.Findings, runInteropFindings(root, req.Msg.GetScenario())...)
	// Static freshness group (group 4): the canonical content-hash engine flags a
	// stale UI bundle as a gating ERROR (restart remediation), parity with the
	// retired smoke phase. Runs on every validation, including --static-only —
	// it needs no BAS — and degrades gracefully when freshness can't be resolved.
	report.Findings = append(report.Findings, h.freshnessFindings(ctx, req.Msg.GetScenario(), root)...)
	// Runtime/render group: runs only when the caller requested execution
	// (include_execution=true, i.e. not --static-only) and the scenario has a UI.
	// Static-only validations make no Code Facts or BAS calls. The group never
	// hard-fails on infra absence — unreachable BAS yields skipped findings.
	staticOnly := !req.Msg.GetIncludeExecution()
	report.Findings = append(report.Findings, h.runtimeFindings(ctx, req.Msg.GetScenario(), root, staticOnly)...)
	enrichAutofix(&report, h.deps.Fixer, root)
	maturityAssessment, err := buildMaturityAssessment(report, h.deps.MaturitySpec)
	if err != nil {
		collector.Stop()
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("build maturity assessment: %w", err))
	}
	execMetrics := collector.Stop()
	resp, err := assessment.BuildValidationResponse(report.Scenario, maturityAssessment, nil, execMetrics)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("build shared validation response: %w", err))
	}
	return connect.NewResponse(resp), nil
}

func buildMaturityAssessment(rep manifestvalidation.Report, spec *assessment.Spec) (*commonv1.MaturityAssessment, error) {
	if spec == nil {
		return nil, nil
	}
	findings := make([]assessment.Finding, 0, len(rep.Findings))
	for _, f := range rep.Findings {
		findings = append(findings, assessment.Finding{
			Code:             f.Code,
			Severity:         severityToken(f.Severity),
			Message:          f.Message,
			Location:         f.Location,
			Remediation:      f.Suggestion,
			Phase:            spec.Phase,
			FixClass:         f.FixClass,
			AutofixAvailable: f.AutofixAvailable,
		})
	}
	return assessment.BuildProtoAssessment(assessment.BuildInput{
		Scenario: rep.Scenario,
		Spec:     *spec,
		Findings: findings,
	})
}

func severityToken(s manifestvalidation.Severity) string {
	switch s {
	case manifestvalidation.SeverityError:
		return "SEVERITY_ERROR"
	case manifestvalidation.SeverityWarning:
		return "SEVERITY_WARNING"
	case manifestvalidation.SeverityInfo:
		return "SEVERITY_INFO"
	default:
		return "SEVERITY_UNSPECIFIED"
	}
}
