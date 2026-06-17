package dependencyhealth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/gin-gonic/gin"
	"github.com/vrooli/maturity-go/assessment"

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
	}
	nativePath, nativeHandler := healthconnect.NewDependencyHealthServiceHandler(handler)
	sharedPath, sharedHandler := scenariovalidationconnect.NewScenarioValidationServiceHandler(handler)
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

	surfaces, surfacesSection, readinessSection, readinessFindings, commandResults, degraded := h.evaluateReadiness(ctx, scenario, msg.GetUseCache())
	resp.Surfaces = append(resp.Surfaces, surfaces...)
	runtimeSection, runtimeFindings, runtimeDegraded := h.evaluateRuntime(ctx, scenario)
	governanceSection, governanceFindings, governanceSummary := h.evaluateGovernance(scenario, surfaces)
	releaseAgeSection, releaseAgeFindings, policySummary := h.evaluateReleaseAge(scenario, surfaces)
	securitySection, securityFindings, securityDegraded := h.evaluateSecurityHealth(ctx, scenario)
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
	driftSection, driftFindings, degraded := h.evaluateDrift(ctx, scenario)
	resp.Sections = append(resp.Sections, driftSection)
	resp.Findings = append(resp.Findings, driftFindings...)
	resp.DegradedDependencies = append(resp.DegradedDependencies, degraded...)
	finalize(resp)
	assessment, err := buildMaturityAssessment(resp, h.spec)
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
	native, err := h.ValidateDependencyHealth(ctx, connect.NewRequest(&healthv1.ValidateDependencyHealthRequest{
		Scenario: scenario,
		UseCache: true,
	}))
	if err != nil {
		return nil, err
	}
	resp, err := assessment.BuildValidationResponse(native.Msg.GetScenario(), native.Msg.GetAssessment(), native.Msg)
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

var _ healthconnect.DependencyHealthServiceHandler = (*connectHandler)(nil)
var _ scenariovalidationconnect.ScenarioValidationServiceHandler = (*connectHandler)(nil)
