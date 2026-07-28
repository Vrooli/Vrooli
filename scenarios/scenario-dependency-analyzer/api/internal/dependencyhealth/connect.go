package dependencyhealth

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/gin-gonic/gin"
	"github.com/vrooli/api-core/metrics"
	"github.com/vrooli/maturity-go/assessment"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	healthv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/dependency_health"
	healthconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/dependency_health/dependency_health_v1connect"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
)

const (
	codeFactsScenarioID      = "code-facts"
	releaseAgeMinimumMinutes = 10080
)

type Options struct {
	MaturitySpec *assessment.Spec
	// Environment is the host CaptureEnvironment captured once at server init
	// (os/arch/cpu/mem/present-GPUs). nil is safe — the metrics collector
	// backfills os/arch/num_cpu from the stdlib.
	Environment *commonv1.CaptureEnvironment
}

// RegisterConnectRoutes mounts the dependency-health producer contract.
func RegisterConnectRoutes(router *gin.Engine, scenariosDir func() string, opts ...Options) {
	var cfg Options
	if len(opts) > 0 {
		cfg = opts[0]
	}
	handler := &connectHandler{
		scenariosDir: scenariosDir,
		spec:         cfg.MaturitySpec,
		environment:  cfg.Environment,
	}
	nativePath, nativeHandler := healthconnect.NewDependencyHealthServiceHandler(handler)
	// DescribeProvider answers readiness from this provider's own descriptor, so a
	// readiness probe no longer costs a full target analysis. A load failure yields
	// the zero Describer, which reports Unimplemented and makes consumers fall back.
	describer, _ := assessment.LoadDescriber(filepath.Join(scenariosDir(), "scenario-dependency-analyzer"))
	sharedPath, sharedHandler := scenariovalidationconnect.NewScenarioValidationServiceHandler(assessment.Serve(handler, describer))
	router.Any(nativePath+"*path", gin.WrapH(nativeHandler))
	router.Any(sharedPath+"*path", gin.WrapH(sharedHandler))
}

type connectHandler struct {
	scenariosDir      func() string
	surfaceDiscoverer surfaceDiscoverer
	commandLookup     func(string) (string, error)
	commandRunner     commandRunner
	statusFetcher     runtimeStatusFetcher
	spec              *assessment.Spec
	// environment is the host CaptureEnvironment captured once at server init.
	// nil is safe — the metrics collector backfills os/arch/num_cpu from stdlib.
	environment *commonv1.CaptureEnvironment
}

func (h *connectHandler) ValidateDependencyHealth(ctx context.Context, req *connect.Request[healthv1.ValidateDependencyHealthRequest]) (*connect.Response[healthv1.DependencyHealthResponse], error) {
	if h == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("dependency health handler is not configured"))
	}
	msg := req.Msg
	if msg == nil {
		msg = &healthv1.ValidateDependencyHealthRequest{}
	}
	scenario := strings.TrimSpace(msg.GetScenario())
	if scenario == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("scenario is required"))
	}

	collector := metricsFrom(ctx)

	resp := &healthv1.DependencyHealthResponse{
		Scenario:    scenario,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339Nano),
		GovernanceSummary: &healthv1.DependencyGovernanceSummary{
			Status:   "not_configured",
			Guidance: approvedDependencyGuidance,
		},
		PolicySummary: &healthv1.DependencyPolicySummary{
			Status: "not_configured",
		},
	}

	readiness := collector.Stage("readiness")
	surfaces, surfacesSection, readinessSection, readinessFindings, commandResults, degraded := h.evaluateReadiness(ctx, scenario, msg.GetUseCache())
	resp.Surfaces = append(resp.Surfaces, surfaces...)
	readiness.End()

	runtime := collector.Stage("runtime")
	runtimeSection, runtimeFindings, runtimeDegraded := h.evaluateRuntime(ctx, scenario)
	runtime.End()

	governance := collector.Stage("governance")
	governanceSection, governanceFindings, governanceSummary := h.evaluateGovernance(scenario, surfaces)
	governance.End()

	releaseAge := collector.Stage("release-age")
	releaseAgeSection, releaseAgeFindings, policySummary := h.evaluateReleaseAge(ctx, scenario, surfaces)
	releaseAge.End()

	security := collector.Stage("security")
	securitySection, securityFindings, securityDegraded := h.evaluateSecurityHealth(ctx, scenario)
	security.End()

	resp.GovernanceSummary = governanceSummary
	resp.PolicySummary = policySummary
	resp.Sections = append(resp.Sections,
		surfacesSection,
		readinessSection,
		runtimeSection,
		governanceSection,
		releaseAgeSection,
		securitySection,
	)
	resp.Findings = append(resp.Findings, readinessFindings...)
	resp.Findings = append(resp.Findings, runtimeFindings...)
	resp.Findings = append(resp.Findings, governanceFindings...)
	resp.Findings = append(resp.Findings, releaseAgeFindings...)
	resp.Findings = append(resp.Findings, securityFindings...)
	resp.CommandResults = append(resp.CommandResults, commandResults...)
	resp.DegradedDependencies = append(resp.DegradedDependencies, degraded...)
	resp.DegradedDependencies = append(resp.DegradedDependencies, runtimeDegraded...)
	resp.DegradedDependencies = append(resp.DegradedDependencies, securityDegraded...)

	drift := collector.Stage("drift")
	driftSection, driftFindings, degraded := h.evaluateDrift(ctx, scenario)
	drift.End()

	resp.Sections = append(resp.Sections, driftSection)
	resp.Findings = append(resp.Findings, driftFindings...)
	resp.DegradedDependencies = append(resp.DegradedDependencies, degraded...)

	gomod := collector.Stage("gomod-replace")
	gomodSection, gomodFindings, gomodDegraded := h.evaluateGoModReplace(ctx, scenario, surfaces)
	gomod.End()

	resp.Sections = append(resp.Sections, gomodSection)
	resp.Findings = append(resp.Findings, gomodFindings...)
	resp.DegradedDependencies = append(resp.DegradedDependencies, gomodDegraded...)
	finalize(resp)
	collector.Gauge("findings", float64(len(resp.GetFindings())))

	maturityStage := collector.Stage("maturity-assessment")
	assessment, err := buildMaturityAssessment(resp, h.spec)
	maturityStage.End()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("build dependency maturity assessment: %w", err))
	}
	resp.Assessment = assessment
	return connect.NewResponse(resp), nil
}

// ValidateScenario adapts SDA's rich dependency-health report to the shared
// ScenarioValidationService contract consumed by Test Genie. The full native
// DependencyHealthResponse remains available in native_detail for SDA's own CLI.
func (h *connectHandler) ValidateScenario(ctx context.Context, req *connect.Request[scenariovalidationv1.ValidateScenarioRequest]) (*connect.Response[scenariovalidationv1.ValidateScenarioResponse], error) {
	if h == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("dependency health handler is not configured"))
	}
	msg := req.Msg
	if msg == nil {
		msg = &scenariovalidationv1.ValidateScenarioRequest{}
	}
	scenario := strings.TrimSpace(msg.GetScenario())
	if scenario == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("scenario is required"))
	}
	// When the caller resolved an explicit scenario path (e.g. Test Genie running
	// deep template validation against a temp scenario outside the repo
	// scenarios/ tree), thread it so the evaluation stages read service.json,
	// surfaces, and release-age policy from that directory.
	evalCtx := withScenarioPath(ctx, msg.GetPath())
	collector := metrics.Start(metrics.WithEnvironment(h.environment))
	native, err := h.ValidateDependencyHealth(withMetrics(evalCtx, collector), connect.NewRequest(&healthv1.ValidateDependencyHealthRequest{
		Scenario: scenario,
		UseCache: true,
	}))
	if err != nil {
		collector.Stop()
		return nil, err
	}
	execMetrics := collector.Stop()
	resp, err := assessment.BuildValidationResponse(native.Msg.GetScenario(), native.Msg.GetAssessment(), native.Msg, execMetrics)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("build shared validation response: %w", err))
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) resolveScenariosDir() string {
	if h.scenariosDir == nil {
		return ""
	}
	return strings.TrimSpace(h.scenariosDir())
}

var (
	_ healthconnect.DependencyHealthServiceHandler = (*connectHandler)(nil)
	// connectHandler implements every validation RPC except DescribeProvider,
	// which the shared assessment.Describer composes in at the mount site.
	_ assessment.ValidationServer = (*connectHandler)(nil)
)
