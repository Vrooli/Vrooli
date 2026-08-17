package dependencyhealth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/gin-gonic/gin"
	"github.com/vrooli/api-core/metrics"
	"github.com/vrooli/maturity-go/assessment"
	"github.com/vrooli/vrooli/packages/deployability"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	healthv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/dependency_health"
	healthconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/dependency_health/dependency_health_v1connect"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
	"scenario-dependency-analyzer/internal/coreset"
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

func loadPortabilitySpec(scenariosRoot string) (*assessment.Spec, error) {
	path := filepath.Join(scenariosRoot, "scenario-dependency-analyzer", ".vrooli", "test-genie-portability.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read portability descriptor: %w", err)
	}
	var descriptor struct {
		Maturity json.RawMessage `json:"maturity"`
	}
	if err := json.Unmarshal(data, &descriptor); err != nil {
		return nil, fmt.Errorf("decode portability descriptor: %w", err)
	}
	spec, err := assessment.ParseEmbeddedSpec(descriptor.Maturity, "scenario-dependency-analyzer", "portability")
	if err != nil {
		return nil, fmt.Errorf("parse portability maturity: %w", err)
	}
	return spec, nil
}

// RegisterConnectRoutes mounts the dependency-health producer contract.
func RegisterConnectRoutes(router *gin.Engine, scenariosDir func() string, opts ...Options) {
	var cfg Options
	if len(opts) > 0 {
		cfg = opts[0]
	}
	portabilitySpec, portabilitySpecErr := loadPortabilitySpec(scenariosDir())
	handler := &connectHandler{
		scenariosDir:       scenariosDir,
		spec:               cfg.MaturitySpec,
		environment:        cfg.Environment,
		portabilitySpec:    portabilitySpec,
		portabilitySpecErr: portabilitySpecErr,
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
	scenariosDir       func() string
	surfaceDiscoverer  surfaceDiscoverer
	commandLookup      func(string) (string, error)
	commandRunner      commandRunner
	statusFetcher      runtimeStatusFetcher
	spec               *assessment.Spec
	portabilitySpec    *assessment.Spec
	portabilitySpecErr error
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

	integration := collector.Stage("integration-conformance")
	integrationSection, integrationFindings, integrationDegraded := h.evaluateIntegrationConformance(ctx, scenario)
	integration.End()

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
		integrationSection,
		governanceSection,
		releaseAgeSection,
		securitySection,
	)
	resp.Findings = append(resp.Findings, readinessFindings...)
	resp.Findings = append(resp.Findings, runtimeFindings...)
	resp.Findings = append(resp.Findings, integrationFindings...)
	resp.Findings = append(resp.Findings, governanceFindings...)
	resp.Findings = append(resp.Findings, releaseAgeFindings...)
	resp.Findings = append(resp.Findings, securityFindings...)
	resp.CommandResults = append(resp.CommandResults, commandResults...)
	resp.DegradedDependencies = append(resp.DegradedDependencies, degraded...)
	resp.DegradedDependencies = append(resp.DegradedDependencies, runtimeDegraded...)
	resp.DegradedDependencies = append(resp.DegradedDependencies, integrationDegraded...)
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

func hasCapability(values []string, wanted string) bool {
	for _, value := range values {
		if strings.EqualFold(strings.TrimSpace(value), wanted) {
			return true
		}
	}
	return false
}

type portabilityServiceManifest struct {
	Dependencies struct {
		Resources map[string]json.RawMessage `json:"resources"`
	} `json:"dependencies"`
}

type portabilityResourceManifest struct {
	Name         string            `json:"name"`
	Bundling     string            `json:"bundling"`
	Platforms    map[string]string `json:"platforms"`
	Requirements struct {
		Class      string                        `json:"class"`
		Weight     float64                       `json:"weight"`
		RAMMB      float64                       `json:"ram_mb"`
		DiskMB     float64                       `json:"disk_mb"`
		CPUCores   float64                       `json:"cpu_cores"`
		GPU        *deployability.GPURequirement `json:"gpu"`
		Network    string                        `json:"network"`
		Source     string                        `json:"source"`
		Confidence string                        `json:"confidence"`
	} `json:"requirements"`
}

func (h *connectHandler) validatePortabilityHost(explicitPath, scenario string) error {
	root := explicitPath
	if root == "" {
		root = filepath.Join(h.resolveScenariosDir(), scenario)
	}
	serviceData, err := os.ReadFile(filepath.Join(root, ".vrooli", "service.json"))
	if err != nil {
		return fmt.Errorf("read target service manifest: %w", err)
	}
	var service portabilityServiceManifest
	if err := json.Unmarshal(serviceData, &service); err != nil {
		return fmt.Errorf("decode target service manifest: %w", err)
	}
	hostOS, ok := portabilityHostOS()
	if !ok {
		return fmt.Errorf("unsupported runtime OS %q", runtime.GOOS)
	}
	repoRoot := filepath.Dir(filepath.Dir(root))
	if err := coreset.ValidateConfiguredTrustedBaseClosure(repoRoot); err != nil {
		return fmt.Errorf("validate trusted-base closure: %w", err)
	}
	if err := validateToolAcquisitionCoverage(repoRoot); err != nil {
		return fmt.Errorf("validate tool acquisition paths: %w", err)
	}
	resourceRoot := filepath.Join(filepath.Dir(filepath.Dir(root)), "resources")
	names := make([]string, 0, len(service.Dependencies.Resources))
	for name := range service.Dependencies.Resources {
		names = append(names, name)
	}
	sort.Strings(names)
	declarations := make([]deployability.DependencyDeclaration, 0, len(names))
	for _, name := range names {
		var input struct {
			Required bool  `json:"required"`
			Enabled  *bool `json:"enabled"`
		}
		if raw := service.Dependencies.Resources[name]; len(raw) > 0 && string(raw) != "null" {
			if err := json.Unmarshal(raw, &input); err != nil {
				return fmt.Errorf("decode resource dependency %q: %w", name, err)
			}
		}
		if input.Enabled != nil && !*input.Enabled {
			continue
		}
		data, err := os.ReadFile(filepath.Join(resourceRoot, name, "resource.json"))
		if err != nil {
			return fmt.Errorf("read declared resource %q: %w", name, err)
		}
		var resource portabilityResourceManifest
		if err := json.Unmarshal(data, &resource); err != nil {
			return fmt.Errorf("decode resource %q: %w", name, err)
		}
		platforms := make(map[deployability.HostOS]deployability.PlatformDeclaration, len(resource.Platforms))
		for rawOS, status := range resource.Platforms {
			if resourceOS, ok := portabilityNormalizeOS(rawOS); ok {
				platforms[resourceOS] = deployability.PlatformDeclaration{Status: status}
			}
		}
		if resource.Name == "" {
			resource.Name = name
		}
		declarations = append(declarations, deployability.DependencyDeclaration{
			Kind: "resource", Name: resource.Name, Required: input.Required, Present: true,
			Bundling: deployability.Bundling(resource.Bundling), PlatformSupport: platforms,
			Requirements: &deployability.ResourceRequirements{
				Class: resource.Requirements.Class, Weight: resource.Requirements.Weight,
				RAMMB: resource.Requirements.RAMMB, DiskMB: resource.Requirements.DiskMB,
				CPUCores:       resource.Requirements.CPUCores,
				GPURequirement: resource.Requirements.GPU,
				Network:        resource.Requirements.Network, Source: resource.Requirements.Source,
				Confidence: resource.Requirements.Confidence,
			},
		})
	}
	resolution := deployability.Resolve(deployability.ResolutionInput{Target: deployability.TargetDeclaration{Name: scenario, Dependencies: declarations}, Tier: deployability.TierLocal, OS: hostOS})
	if resolution.Verdict == deployability.VerdictIneligible || resolution.Verdict == deployability.VerdictUnknown {
		messages := make([]string, 0, len(resolution.Reasons))
		for _, reason := range resolution.Reasons {
			messages = append(messages, reason.Message)
		}
		return fmt.Errorf("declared deployment input contradicts observed host %s: %s", hostOS, strings.Join(messages, "; "))
	}
	return nil
}

func portabilityHostOS() (deployability.HostOS, bool) {
	switch runtime.GOOS {
	case "linux":
		return deployability.HostOSLinux, true
	case "darwin":
		return deployability.HostOSMacOS, true
	case "windows":
		return deployability.HostOSWindows, true
	default:
		return "", false
	}
}

func portabilityNormalizeOS(value string) (deployability.HostOS, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "linux":
		return deployability.HostOSLinux, true
	case "macos", "darwin":
		return deployability.HostOSMacOS, true
	case "windows", "win32":
		return deployability.HostOSWindows, true
	default:
		return "", false
	}
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
	if hasCapability(msg.GetCapabilitySubset(), "portability") {
		if err := h.validatePortabilityHost(scenarioPathFrom(evalCtx), scenario); err != nil {
			return nil, connect.NewError(connect.CodeFailedPrecondition, err)
		}
		if h.portabilitySpecErr != nil || h.portabilitySpec == nil {
			if h.portabilitySpecErr != nil {
				return nil, connect.NewError(connect.CodeFailedPrecondition, h.portabilitySpecErr)
			}
			return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("portability maturity spec is not configured"))
		}
		collector := metrics.Start(metrics.WithEnvironment(h.environment))
		execMetrics := collector.Stop()
		portabilityAssessment, assessmentErr := buildMaturityAssessment(&healthv1.DependencyHealthResponse{Scenario: scenario}, h.portabilitySpec)
		if assessmentErr != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("build portability maturity assessment: %w", assessmentErr))
		}
		resp, responseErr := assessment.BuildValidationResponse(scenario, portabilityAssessment, &healthv1.DependencyHealthResponse{Scenario: scenario}, execMetrics)
		if responseErr != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("build portability validation response: %w", responseErr))
		}
		return connect.NewResponse(resp), nil
	}
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
