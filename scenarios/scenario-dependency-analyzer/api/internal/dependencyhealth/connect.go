package dependencyhealth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/gin-gonic/gin"
	"github.com/vrooli/api-core/discovery"
	vroolicli "github.com/vrooli/vrooli-cli-go"

	"scenario-dependency-analyzer/internal/dependencygovernance"
	"scenario-dependency-analyzer/internal/interfacegraph"

	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
	factsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
	factsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts/facts_v1connect"
	governancev1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/dependency_governance"
	healthv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/dependency_health"
	healthconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/dependency_health/dependency_health_v1connect"
	"google.golang.org/protobuf/types/known/structpb"
)

const (
	approvedDependencyGuidance = dependencygovernance.Guidance
	codeFactsScenarioID        = "code-facts"
	releaseAgeMinimumMinutes   = 10080
)

// RegisterConnectRoutes mounts the dependency-health producer contract.
func RegisterConnectRoutes(router *gin.Engine, scenariosDir func() string) {
	connectPath, connectHandler := healthconnect.NewDependencyHealthServiceHandler(&connectHandler{
		scenariosDir: scenariosDir,
	})
	router.Any(connectPath+"*path", gin.WrapH(connectHandler))
}

type connectHandler struct {
	scenariosDir      func() string
	surfaceDiscoverer surfaceDiscoverer
	commandLookup     func(string) (string, error)
	commandRunner     commandRunner
	statusFetcher     runtimeStatusFetcher
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
	securitySection, securityDegraded := h.evaluateSecurityHealth(ctx)
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
	resp.CommandResults = append(resp.CommandResults, commandResults...)
	resp.DegradedDependencies = append(resp.DegradedDependencies, degraded...)
	resp.DegradedDependencies = append(resp.DegradedDependencies, runtimeDegraded...)
	resp.DegradedDependencies = append(resp.DegradedDependencies, securityDegraded...)
	driftSection, driftFindings, degraded := h.evaluateDrift(ctx, scenario)
	resp.Sections = append(resp.Sections, driftSection)
	resp.Findings = append(resp.Findings, driftFindings...)
	resp.DegradedDependencies = append(resp.DegradedDependencies, degraded...)
	finalize(resp)
	return connect.NewResponse(resp), nil
}

type releaseAgePolicy struct {
	Path                     string
	MinimumReleaseAge        int
	HasMinimumReleaseAge     bool
	MinimumReleaseAgeExclude []string
}

type releaseAgeException struct {
	PackageName   string `json:"package_name"`
	Spec          string `json:"spec"`
	Rationale     string `json:"rationale"`
	ApprovedBy    string `json:"approved_by"`
	ApprovedDate  string `json:"approved_date"`
	ReviewExpires string `json:"review_expires"`
	State         string `json:"state"`
}

type releaseAgeExceptionFile struct {
	ReleaseAgeExceptions []releaseAgeException `json:"release_age_exceptions"`
}

type securityHealthDependencyStatus struct {
	Available            bool   `json:"available"`
	IndexedCount         int    `json:"indexed_count"`
	VulnerableCount      int    `json:"vulnerable_count"`
	LastReconcileAt      string `json:"last_reconcile_at"`
	LastReconcileOutcome string `json:"last_reconcile_outcome"`
	IndexedVectors       int    `json:"indexed_vectors"`
	ExpectedVectors      int    `json:"expected_vectors"`
	IndexReady           bool   `json:"index_ready"`
}

func (h *connectHandler) evaluateReleaseAge(scenario string, surfaces []*healthv1.DependencyHealthSurface) (*healthv1.DependencyHealthSection, []*healthv1.DependencyHealthFinding, *healthv1.DependencyPolicySummary) {
	scenarioDir := filepath.Join(h.resolveScenariosDir(), scenario)
	exceptions := loadReleaseAgeExceptions(filepath.Join(filepath.Dir(h.resolveScenariosDir()), ".vrooli", "dependencies", "approved-dependencies.json"))
	var findings []*healthv1.DependencyHealthFinding
	policies := map[string]struct{}{}
	exceptionCount := 0
	pnpmSurfaces := 0

	for _, surface := range surfaces {
		if !isJavaScriptSurface(surface) || packageManagerForSurface(surface) != "pnpm" || !fileExists(filepath.Join(surface.GetRootPath(), "package.json")) {
			continue
		}
		pnpmSurfaces++
		policy, found, err := readReleaseAgePolicy(surface.GetRootPath(), scenarioDir)
		if err != nil {
			findings = append(findings, releaseAgeFinding("policy."+surfaceID(surface)+".unreadable", "ERROR", "pnpm release-age policy is unreadable", "SDA could not read the pnpm workspace policy for this surface.", "Fix the reported pnpm-workspace.yaml file, then rerun dependency health.", surface, "dependency.release_age.policy_readable", err.Error(), "readable pnpm release-age policy"))
			continue
		}
		if !found {
			findings = append(findings, releaseAgeFinding("policy."+surfaceID(surface)+".missing", "ERROR", "pnpm release-age policy is missing", "This pnpm-managed dependency surface does not define a pnpm-workspace.yaml release-age gate.", "Add `minimumReleaseAge: 10080` to the surface's pnpm-workspace.yaml, or document an operator-approved exception.", surface, "dependency.release_age.minimum_configured", "missing minimumReleaseAge", fmt.Sprintf("minimumReleaseAge >= %d", releaseAgeMinimumMinutes)))
			continue
		}
		policies[filepath.ToSlash(policy.Path)] = struct{}{}
		if !policy.HasMinimumReleaseAge {
			findings = append(findings, releaseAgeFinding("policy."+surfaceID(surface)+".minimum-missing", "ERROR", "pnpm release-age minimum is missing", "This pnpm workspace policy does not set minimumReleaseAge.", "Add `minimumReleaseAge: 10080` to the reported pnpm-workspace.yaml.", surface, "dependency.release_age.minimum_configured", "minimumReleaseAge unset", fmt.Sprintf("minimumReleaseAge >= %d", releaseAgeMinimumMinutes)))
		} else if policy.MinimumReleaseAge < releaseAgeMinimumMinutes {
			findings = append(findings, releaseAgeFinding("policy."+surfaceID(surface)+".minimum-too-low", "ERROR", "pnpm release-age minimum is too low", "This pnpm workspace allows dependency versions newer than the Vrooli default cooldown.", "Raise minimumReleaseAge to at least 10080 minutes, or file an explicit exception for the package that must bypass the cooldown.", surface, "dependency.release_age.minimum_value", fmt.Sprintf("minimumReleaseAge=%d", policy.MinimumReleaseAge), fmt.Sprintf("minimumReleaseAge >= %d", releaseAgeMinimumMinutes)))
		}
		for _, excluded := range policy.MinimumReleaseAgeExclude {
			exceptionCount++
			if !hasApprovedReleaseAgeException(exceptions, excluded) {
				findings = append(findings, releaseAgeFinding("policy."+surfaceID(surface)+".exclude."+slug(excluded), "ERROR", "pnpm release-age exclusion lacks governance approval", "A package bypasses the release-age gate but no approved dependency governance exception with rationale and expiry was found.", "Record the exception in .vrooli/dependencies/approved-dependencies.json with package/spec, rationale, approver, and review expiry, or remove the exclusion.", surface, "dependency.release_age.exclude_governed", excluded, "approved release-age exception with rationale and expiry"))
			}
		}
	}

	policyNames := sortedSet(policies)
	summary := &healthv1.DependencyPolicySummary{
		Status:                   statusFromFindings(findings, "release-age"),
		ReleaseAgeMinimumMinutes: releaseAgeMinimumMinutes,
		ReleaseAgeExceptions:     int32(exceptionCount),
		Policies:                 policyNames,
	}
	if pnpmSurfaces == 0 {
		summary.Status = "not_applicable"
		return section("release-age", "Package release-age policy", "not_applicable", "No pnpm-managed JavaScript/TypeScript dependency surfaces were discovered."), findings, summary
	}
	text := fmt.Sprintf("%d pnpm-managed dependency surface(s) checked for minimumReleaseAge >= %d minutes.", pnpmSurfaces, releaseAgeMinimumMinutes)
	return sectionWithFindingIDs("release-age", "Package release-age policy", summary.GetStatus(), text, findingIDs(findings, "release-age")), findings, summary
}

func (h *connectHandler) evaluateSecurityHealth(ctx context.Context) (*healthv1.DependencyHealthSection, []*healthv1.DegradedDependency) {
	lookup := h.commandLookup
	if lookup == nil {
		lookup = exec.LookPath
	}
	if _, err := lookup("security-health"); err != nil {
		return section("security", "Security Health dependency index", "degraded", "Security Health CLI is unavailable; SDA skipped dependency index freshness status without running vulnerability scanners."), []*healthv1.DegradedDependency{
			{
				Id:         "security-health-deps-status",
				Dependency: "security-health",
				Domain:     "security",
				Reason:     fmt.Sprintf("security-health CLI unavailable: %v", err),
			},
		}
	}
	runner := h.commandRunner
	if runner == nil {
		runner = execRunner{}
	}
	statusCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	out, err := runner.Run(statusCtx, filepath.Dir(h.resolveScenariosDir()), "security-health", "deps", "status", "--json")
	if err != nil {
		observed := strings.TrimSpace(out)
		if observed == "" {
			observed = err.Error()
		}
		return section("security", "Security Health dependency index", "degraded", "Security Health dependency index status is unavailable; SDA did not run vulnerability scanners."), []*healthv1.DegradedDependency{
			{
				Id:         "security-health-deps-status",
				Dependency: "security-health deps status",
				Domain:     "security",
				Reason:     observed,
			},
		}
	}
	var status securityHealthDependencyStatus
	if err := json.Unmarshal([]byte(out), &status); err != nil {
		return section("security", "Security Health dependency index", "degraded", "Security Health dependency index status returned unparseable JSON."), []*healthv1.DegradedDependency{
			{
				Id:         "security-health-deps-status",
				Dependency: "security-health deps status",
				Domain:     "security",
				Reason:     fmt.Sprintf("parse status JSON: %v", err),
			},
		}
	}
	sectionStatus := "pass"
	if !status.Available || !status.IndexReady {
		sectionStatus = "degraded"
	}
	summary := fmt.Sprintf("Security Health dependency index available=%t ready=%t indexed=%d vulnerable=%d.", status.Available, status.IndexReady, status.IndexedCount, status.VulnerableCount)
	if status.LastReconcileAt != "" {
		summary += " Last reconcile: " + status.LastReconcileAt + "."
	}
	var degraded []*healthv1.DegradedDependency
	if sectionStatus == "degraded" {
		degraded = append(degraded, &healthv1.DegradedDependency{
			Id:         "security-health-deps-status",
			Dependency: "security-health dependency index",
			Domain:     "security",
			Reason:     summary,
		})
	}
	return section("security", "Security Health dependency index", sectionStatus, summary), degraded
}

type surfaceDiscoverer interface {
	Discover(ctx context.Context, scenario, scenarioDir, repoRoot string, useCache bool) (surfaceInventory, error)
}

type commandRunner interface {
	Run(ctx context.Context, dir string, name string, args ...string) (string, error)
}

type runtimeStatusFetcher interface {
	ResourceStatuses(ctx context.Context) (*cliv1.ResourceStatusesResponse, error)
	ScenarioStatus(ctx context.Context, name string) (*cliv1.ScenarioStatusSingle, error)
}

type execRunner struct{}

func (execRunner) Run(ctx context.Context, dir string, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOWORK=off")
	out, err := cmd.CombinedOutput()
	return string(out), err
}

type dependencyManifest struct {
	Dependencies struct {
		Resources map[string]dependencyResource `json:"resources"`
		Scenarios map[string]dependencyScenario `json:"scenarios"`
	} `json:"dependencies"`
}

type dependencyResource struct {
	Enabled  *bool  `json:"enabled"`
	Required bool   `json:"required"`
	Type     string `json:"type"`
}

type dependencyScenario struct {
	Enabled       *bool  `json:"enabled"`
	Required      bool   `json:"required"`
	StartupPolicy string `json:"startup_policy"`
}

func (h *connectHandler) evaluateGovernance(scenario string, surfaces []*healthv1.DependencyHealthSurface) (*healthv1.DependencyHealthSection, []*healthv1.DependencyHealthFinding, *healthv1.DependencyGovernanceSummary) {
	repoRoot := filepath.Dir(h.resolveScenariosDir())
	registry := dependencygovernance.NewRegistry(repoRoot)
	observed := dependencygovernance.ScanSurfaceDependencies(governanceSurfaces(surfaces))
	result, err := registry.ValidateObserved(scenario, observed)
	if err != nil {
		finding := &healthv1.DependencyHealthFinding{
			Id:           "governance.registry.unreadable",
			Severity:     "ERROR",
			SourceDomain: "governance",
			Title:        "Approved dependency registry unavailable",
			Description:  "SDA could not read or parse the approved dependency registry.",
			Remediation:  "Fix .vrooli/dependencies/approved-dependencies.json, then rerun dependency health.",
			FilePath:     ".vrooli/dependencies/approved-dependencies.json",
			RuleId:       "dependency.governance.registry_readable",
			Observed:     err.Error(),
			Expected:     "readable approved dependency registry",
		}
		summary := &healthv1.DependencyGovernanceSummary{Status: "fail", Guidance: approvedDependencyGuidance}
		return sectionWithFindingIDs("governance", "Approved dependency governance", "fail", "Approved dependency registry could not be evaluated.", []string{finding.GetId()}), []*healthv1.DependencyHealthFinding{finding}, summary
	}

	findings := make([]*healthv1.DependencyHealthFinding, 0, len(result.GetFindings()))
	for _, finding := range result.GetFindings() {
		findings = append(findings, governanceHealthFinding(finding))
	}
	summary := governanceHealthSummary(result.GetSummary())
	status := summary.GetStatus()
	if status == "" {
		status = statusFromFindings(findings, "governance")
	}
	text := fmt.Sprintf("%d observed third-party dependency declaration(s) checked against %d recorded governance decision(s).", result.GetSummary().GetObserved(), governanceRecordCount(result.GetSummary()))
	if status == "not_configured" {
		text = "Approved dependency registry is present but has no records yet; observed dependencies are reported as needs-review guidance, not allowlist failures."
	}
	return sectionWithFindingIDs("governance", "Approved dependency governance", status, text, findingIDs(findings, "governance")), findings, summary
}

func governanceSurfaces(surfaces []*healthv1.DependencyHealthSurface) []dependencygovernance.Surface {
	out := make([]dependencygovernance.Surface, 0, len(surfaces))
	for _, surface := range surfaces {
		out = append(out, dependencygovernance.Surface{
			ID:       surface.GetId(),
			Language: surface.GetLanguage(),
			RootPath: surface.GetRootPath(),
		})
	}
	return out
}

func governanceHealthSummary(summary *governancev1.DependencyGovernanceSummary) *healthv1.DependencyGovernanceSummary {
	if summary == nil {
		return &healthv1.DependencyGovernanceSummary{Status: "not_configured", Guidance: approvedDependencyGuidance}
	}
	return &healthv1.DependencyGovernanceSummary{
		Status:                  summary.GetStatus(),
		Approved:                summary.GetApproved(),
		ApprovedWithConstraints: summary.GetApprovedWithConstraints(),
		NeedsReview:             summary.GetNeedsReview() + summary.GetUnrecorded(),
		Blocked:                 summary.GetBlocked(),
		Deprecated:              summary.GetDeprecated(),
		Guidance:                approvedDependencyGuidance,
	}
}

func governanceRecordCount(summary *governancev1.DependencyGovernanceSummary) int32 {
	if summary == nil {
		return 0
	}
	return summary.GetApproved() + summary.GetApprovedWithConstraints() + summary.GetNeedsReview() + summary.GetBlocked() + summary.GetDeprecated()
}

func governanceHealthFinding(finding *governancev1.ApprovedDependencyFinding) *healthv1.DependencyHealthFinding {
	return &healthv1.DependencyHealthFinding{
		Id:           finding.GetId(),
		Severity:     finding.GetSeverity(),
		SourceDomain: "governance",
		Title:        finding.GetTitle(),
		Description:  finding.GetDescription(),
		Remediation:  finding.GetRemediation(),
		FilePath:     finding.GetFilePath(),
		RuleId:       "dependency.governance.approved_dependency",
		Observed:     finding.GetObserved(),
		Expected:     finding.GetExpected(),
	}
}

func (h *connectHandler) evaluateRuntime(ctx context.Context, scenario string) (*healthv1.DependencyHealthSection, []*healthv1.DependencyHealthFinding, []*healthv1.DegradedDependency) {
	manifest, err := loadDependencyManifest(filepath.Join(h.resolveScenariosDir(), scenario, ".vrooli", "service.json"))
	if err != nil {
		finding := &healthv1.DependencyHealthFinding{
			Id:           "runtime.manifest.unreadable",
			Severity:     "ERROR",
			SourceDomain: "runtime",
			Title:        "Dependency manifest unavailable",
			Description:  "SDA could not read the scenario dependency declarations from .vrooli/service.json.",
			Remediation:  "Fix or restore the scenario .vrooli/service.json file, then rerun dependency health.",
			FilePath:     filepath.Join("scenarios", scenario, ".vrooli", "service.json"),
			RuleId:       "dependency.runtime.manifest_readable",
			Observed:     err.Error(),
			Expected:     "readable service manifest",
		}
		return sectionWithFindingIDs("runtime", "Runtime dependencies", "fail", "Dependency manifest could not be read.", []string{finding.GetId()}), []*healthv1.DependencyHealthFinding{finding}, nil
	}

	requiredResources := manifest.requiredResources()
	requiredScenarios := manifest.requiredScenarios()
	if len(requiredResources) == 0 && len(requiredScenarios) == 0 {
		return section("runtime", "Runtime dependencies", "pass", "No required runtime resources or scenarios declared."), nil, nil
	}

	fetcher := h.statusFetcher
	if fetcher == nil {
		fetcher = vroolicli.New()
	}

	var findings []*healthv1.DependencyHealthFinding
	var degraded []*healthv1.DegradedDependency
	findings = append(findings, h.checkRequiredResources(ctx, fetcher, scenario, requiredResources, &degraded)...)
	findings = append(findings, h.checkRequiredScenarios(ctx, fetcher, scenario, requiredScenarios, &degraded)...)

	status := statusFromFindings(findings, "runtime")
	if len(degraded) > 0 && status == "pass" {
		status = "degraded"
	}
	summary := fmt.Sprintf("%d required resource(s), %d required scenario dependency(ies) checked.", len(requiredResources), len(requiredScenarios))
	if len(findings) > 0 {
		summary = fmt.Sprintf("%s %d runtime finding(s).", summary, len(findings))
	}
	if len(degraded) > 0 {
		summary = fmt.Sprintf("%s %d degraded runtime integration(s).", summary, len(degraded))
	}
	return sectionWithFindingIDs("runtime", "Runtime dependencies", status, summary, findingIDs(findings, "runtime")), findings, degraded
}

func loadDependencyManifest(path string) (*dependencyManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var manifest dependencyManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}
	return &manifest, nil
}

func (m *dependencyManifest) requiredResources() []string {
	if m == nil {
		return nil
	}
	var names []string
	for name, dep := range m.Dependencies.Resources {
		if dep.Enabled != nil && !*dep.Enabled {
			continue
		}
		if dep.Required {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

func (m *dependencyManifest) requiredScenarios() []string {
	if m == nil {
		return nil
	}
	var names []string
	for name, dep := range m.Dependencies.Scenarios {
		if dep.Enabled != nil && !*dep.Enabled {
			continue
		}
		if dep.Required || strings.TrimSpace(dep.StartupPolicy) == "must_start" {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

func (h *connectHandler) checkRequiredResources(ctx context.Context, fetcher runtimeStatusFetcher, scenario string, required []string, degraded *[]*healthv1.DegradedDependency) []*healthv1.DependencyHealthFinding {
	if len(required) == 0 {
		return nil
	}
	resp, err := fetcher.ResourceStatuses(ctx)
	if err != nil {
		*degraded = append(*degraded, &healthv1.DegradedDependency{
			Id:         "runtime-resource-status",
			Dependency: "vrooli resource status",
			Domain:     "runtime",
			Reason:     fmt.Sprintf("resource status unavailable: %v", err),
		})
		return nil
	}
	byName := make(map[string]*cliv1.ResourceStatus, len(resp.GetResources()))
	for _, status := range resp.GetResources() {
		if name := status.GetResource().GetName(); name != "" {
			byName[name] = status
		}
	}
	var findings []*healthv1.DependencyHealthFinding
	for _, name := range required {
		status, ok := byName[name]
		if !ok {
			findings = append(findings, runtimeFinding(scenario, "resource."+name+".missing", "ERROR", "Required resource missing from runtime status", "A required Vrooli resource was declared but not found in `vrooli resource status`.", "Install or enable the resource, then rerun dependency health.", "dependency.runtime.resource_present", name+" not found", "resource present in Vrooli runtime status"))
			continue
		}
		healthy, known := boolValue(status.GetHealthy())
		switch {
		case !status.GetRunning():
			findings = append(findings, runtimeFinding(scenario, "resource."+name+".stopped", "ERROR", "Required resource is not running", "A required Vrooli resource is installed but not running.", "Start the resource with `vrooli resource start "+name+"` or restart the owning scenario stack, then rerun dependency health.", "dependency.runtime.resource_running", resourceObserved(status), "running=true"))
		case known && !healthy:
			findings = append(findings, runtimeFinding(scenario, "resource."+name+".unhealthy", "ERROR", "Required resource is unhealthy", "A required Vrooli resource is running but its health probe is failing.", "Inspect `vrooli resource status "+name+" --json`, resolve the probe failure, then rerun dependency health.", "dependency.runtime.resource_healthy", resourceObserved(status), "healthy=true"))
		}
	}
	return findings
}

func (h *connectHandler) checkRequiredScenarios(ctx context.Context, fetcher runtimeStatusFetcher, scenario string, required []string, degraded *[]*healthv1.DegradedDependency) []*healthv1.DependencyHealthFinding {
	var findings []*healthv1.DependencyHealthFinding
	for _, name := range required {
		resp, err := fetcher.ScenarioStatus(ctx, name)
		if err != nil {
			*degraded = append(*degraded, &healthv1.DegradedDependency{
				Id:         "runtime-scenario-status-" + slug(name),
				Dependency: name,
				Domain:     "runtime",
				Reason:     fmt.Sprintf("scenario status unavailable: %v", err),
			})
			continue
		}
		item := resp.GetScenario()
		healthy, known := boolValue(item.GetHealthStatus())
		switch {
		case item.GetStatus() != "running":
			findings = append(findings, runtimeFinding(scenario, "scenario."+name+".not-running", "ERROR", "Required scenario dependency is not running", "A required scenario dependency is not currently running.", "Start it with `vrooli scenario start "+name+"` or `vrooli scenario restart "+name+"`, then rerun dependency health.", "dependency.runtime.scenario_running", "status="+emptyAs(item.GetStatus(), "unknown"), "status=running"))
		case known && !healthy:
			findings = append(findings, runtimeFinding(scenario, "scenario."+name+".unhealthy", "ERROR", "Required scenario dependency is unhealthy", "A required scenario dependency is running but its health probe is failing.", "Inspect `vrooli scenario status "+name+" --json`, resolve the health failure, then rerun dependency health.", "dependency.runtime.scenario_healthy", "running=true health=false", "healthy=true"))
		}
	}
	return findings
}

type surfaceInventory struct {
	Surfaces       []*healthv1.DependencyHealthSurface
	DegradedReason string
}

type codeFactsSurfaceDiscoverer struct{}

func (codeFactsSurfaceDiscoverer) Discover(ctx context.Context, scenario, scenarioDir, repoRoot string, useCache bool) (surfaceInventory, error) {
	resolver := discovery.NewResolver(discovery.ResolverConfig{})
	baseURL, err := resolver.ResolveScenarioURLDefault(ctx, codeFactsScenarioID)
	if err != nil {
		return fallbackSurfaceInventory(scenarioDir, fmt.Sprintf("Code Facts unavailable: %v", err)), nil
	}
	client := factsconnect.NewCodeFactsServiceClient(http.DefaultClient, baseURL)
	resp, err := client.DescribeCodeFacts(ctx, connect.NewRequest(&factsv1.DescribeCodeFactsRequest{
		Target: &factsv1.CodeTarget{
			Kind:     factsv1.TargetKind_TARGET_KIND_SCENARIO,
			Scenario: scenario,
			Path:     scenarioDir,
			RepoRoot: repoRoot,
		},
		Include:  []factsv1.FactFamily{factsv1.FactFamily_FACT_FAMILY_SURFACES, factsv1.FactFamily_FACT_FAMILY_PARSE_UNITS},
		UseCache: useCache,
	}))
	if err != nil {
		return fallbackSurfaceInventory(scenarioDir, fmt.Sprintf("Code Facts unavailable: %v", err)), nil
	}
	return surfacesFromCodeFacts(resp.Msg, scenarioDir), nil
}

func (h *connectHandler) evaluateReadiness(ctx context.Context, scenario string, useCache bool) ([]*healthv1.DependencyHealthSurface, *healthv1.DependencyHealthSection, *healthv1.DependencyHealthSection, []*healthv1.DependencyHealthFinding, []*healthv1.DependencyHealthCommandResult, []*healthv1.DegradedDependency) {
	scenarioDir := filepath.Join(h.resolveScenariosDir(), scenario)
	repoRoot := filepath.Dir(h.resolveScenariosDir())
	discoverer := h.surfaceDiscoverer
	if discoverer == nil {
		discoverer = codeFactsSurfaceDiscoverer{}
	}
	inv, err := discoverer.Discover(ctx, scenario, scenarioDir, repoRoot, useCache)
	if err != nil {
		inv = fallbackSurfaceInventory(scenarioDir, fmt.Sprintf("Code Facts discovery failed: %v", err))
	}

	var degraded []*healthv1.DegradedDependency
	if inv.DegradedReason != "" {
		degraded = append(degraded, &healthv1.DegradedDependency{
			Id:         "code-facts-surfaces",
			Dependency: codeFactsScenarioID,
			Domain:     "surfaces",
			Reason:     inv.DegradedReason,
		})
	}

	findings, commandResults := h.checkReadiness(ctx, scenario, inv.Surfaces)
	surfacesStatus := "pass"
	surfacesSummary := fmt.Sprintf("%d Code Facts surface(s) discovered.", len(inv.Surfaces))
	if inv.DegradedReason != "" {
		surfacesStatus = "degraded"
		surfacesSummary = fmt.Sprintf("%d fallback surface(s) discovered after Code Facts degradation.", len(inv.Surfaces))
	}
	if len(inv.Surfaces) == 0 {
		surfacesStatus = "warn"
		surfacesSummary = "No dependency validation surfaces were discovered."
		findings = append(findings, &healthv1.DependencyHealthFinding{
			Id:           "surfaces.none",
			Severity:     "WARNING",
			SourceDomain: "surfaces",
			Title:        "No dependency surfaces discovered",
			Description:  "SDA could not discover Go, JavaScript, TypeScript, or Python surfaces to validate.",
			Remediation:  "Ensure Code Facts can describe this scenario or add scenario surface metadata.",
			RuleId:       "dependency.surfaces.none",
			Observed:     "no surfaces",
			Expected:     "at least one supported or explicitly unsupported surface",
		})
	}

	readinessStatus := statusFromFindings(findings, "readiness")
	readinessSummary := summarizeReadiness(findings, commandResults)
	return inv.Surfaces,
		section("surfaces", "Code Facts surfaces", surfacesStatus, surfacesSummary),
		sectionWithFindingIDs("readiness", "Dependency readiness", readinessStatus, readinessSummary, findingIDs(findings, "readiness")),
		findings, commandResults, degraded
}

func (h *connectHandler) checkReadiness(ctx context.Context, scenario string, surfaces []*healthv1.DependencyHealthSurface) ([]*healthv1.DependencyHealthFinding, []*healthv1.DependencyHealthCommandResult) {
	var findings []*healthv1.DependencyHealthFinding
	var commands []*healthv1.DependencyHealthCommandResult
	lookup := h.commandLookup
	if lookup == nil {
		lookup = exec.LookPath
	}
	runner := h.commandRunner
	if runner == nil {
		runner = execRunner{}
	}

	requiredCommands := map[string]string{
		"bash": "baseline tool required to run local phases",
		"curl": "baseline tool required to call local scenario APIs",
		"jq":   "baseline tool required to inspect JSON command output",
	}
	for _, surface := range surfaces {
		switch normalizedLanguage(surface.GetLanguage()) {
		case "go":
			requiredCommands["go"] = "Go runtime required by discovered Go dependency surface"
		case "javascript", "typescript":
			requiredCommands["node"] = "Node.js runtime required by discovered JavaScript/TypeScript dependency surface"
			manager := packageManagerForSurface(surface)
			requiredCommands[manager] = "package manager required by discovered JavaScript/TypeScript dependency surface"
		case "python":
			requiredCommands["python3"] = "Python runtime required by discovered Python dependency surface"
		case "", "unknown":
			findings = append(findings, readinessFinding("unsupported-surface."+surfaceID(surface), "WARNING", "Unsupported dependency surface", "SDA could not determine a supported language for this surface.", "Add Code Facts language evidence for the surface or mark it unsupported explicitly.", surface, "dependency.surface.unsupported", surface.GetLanguage(), "go, javascript, typescript, or python"))
		default:
			findings = append(findings, readinessFinding("unsupported-surface."+surfaceID(surface), "WARNING", "Unsupported dependency surface language", "SDA does not yet implement dependency readiness checks for this surface language.", "Add SDA dependency readiness support for this language before relying on the dependency health result.", surface, "dependency.surface.unsupported_language", surface.GetLanguage(), "supported dependency language"))
		}
	}
	for _, name := range sortedKeys(requiredCommands) {
		id := "command." + name
		path, err := lookup(name)
		if err != nil {
			commands = append(commands, &healthv1.DependencyHealthCommandResult{Id: id, Command: name, Status: "missing", Summary: requiredCommands[name]})
			findings = append(findings, &healthv1.DependencyHealthFinding{
				Id:           "readiness.command." + name + ".missing",
				Severity:     "ERROR",
				SourceDomain: "readiness",
				Title:        "Required command missing",
				Description:  fmt.Sprintf("Command %q is required: %s.", name, requiredCommands[name]),
				Remediation:  fmt.Sprintf("Install or expose %s on PATH.", name),
				RuleId:       "dependency.command.available",
				Observed:     "command not found",
				Expected:     name + " available on PATH",
			})
			continue
		}
		commands = append(commands, &healthv1.DependencyHealthCommandResult{Id: id, Command: name, Status: "pass", Summary: "available at " + path})
	}

	for _, cmd := range []struct {
		name       string
		constraint string
	}{
		{name: "go", constraint: ">=1.21"},
		{name: "node", constraint: ">=18.0.0"},
		{name: "python3", constraint: ">=3.10.0"},
	} {
		if _, ok := requiredCommands[cmd.name]; ok {
			findings = append(findings, h.checkVersion(ctx, scenarioDirFromSurfaces(surfaces), runner, cmd.name, cmd.constraint)...)
		}
	}

	for _, surface := range surfaces {
		switch normalizedLanguage(surface.GetLanguage()) {
		case "go":
			findings = append(findings, h.checkGoSurface(ctx, runner, surface)...)
		case "javascript", "typescript":
			findings = append(findings, checkNodeSurface(surface)...)
		}
	}
	return findings, commands
}

func (h *connectHandler) evaluateDrift(ctx context.Context, scenario string) (*healthv1.DependencyHealthSection, []*healthv1.DependencyHealthFinding, []*healthv1.DegradedDependency) {
	builder := interfacegraph.NewBuilder(
		interfacegraph.NewProtoHealthClient(nil, nil),
		interfacegraph.NewCodeFactsClient(nil, nil),
	)
	detector := interfacegraph.NewDriftDetector(builder, h.resolveScenariosDir())
	report, err := detector.Detect(ctx, interfacegraph.BuildRequest{
		Scenarios: []string{scenario},
		RepoRoot:  filepath.Dir(h.resolveScenariosDir()),
	})
	if err != nil {
		return section("graph", "Dependency graph drift", "degraded", "Graph drift could not be evaluated."), nil, []*healthv1.DegradedDependency{
			{
				Id:         "graph-drift",
				Dependency: "scenario-dependency-analyzer",
				Domain:     "graph",
				Reason:     fmt.Sprintf("dependency graph drift unavailable: %v", err),
			},
		}
	}

	findings := make([]*healthv1.DependencyHealthFinding, 0, len(report.Findings))
	ids := make([]string, 0, len(report.Findings))
	for _, finding := range report.Findings {
		id := driftFindingID(finding)
		ids = append(ids, id)
		findings = append(findings, &healthv1.DependencyHealthFinding{
			Id:           id,
			Severity:     finding.Severity,
			SourceDomain: "graph",
			Title:        "Declared dependency graph drift",
			Description:  finding.Message,
			Remediation:  driftRemediation(finding.Kind),
			FilePath:     filepath.Join("scenarios", finding.Scenario, ".vrooli", "service.json"),
			RuleId:       "dependency." + finding.Kind,
			Observed:     driftObserved(finding),
			Expected:     driftExpected(finding),
		})
	}
	status := "pass"
	summary := "Declared scenario dependencies match import evidence."
	if len(findings) > 0 {
		status = statusFromFindings(findings, "graph")
		summary = fmt.Sprintf("%d dependency graph drift finding(s).", len(findings))
	}
	out := section("graph", "Dependency graph drift", status, summary)
	out.FindingIds = ids
	return out, findings, nil
}

func surfacesFromCodeFacts(report *factsv1.CodeFactsReport, scenarioDir string) surfaceInventory {
	if report == nil {
		return fallbackSurfaceInventory(scenarioDir, "Code Facts returned an empty report")
	}
	root := firstNonEmpty(report.GetTarget().GetRootPath(), scenarioDir)
	parseByRoot := map[string]*factsv1.ParseUnit{}
	for _, unit := range report.GetParseUnits() {
		parseByRoot[filepath.Clean(absPath(root, unit.GetRootPath()))] = unit
	}
	var surfaces []*healthv1.DependencyHealthSurface
	for _, src := range report.GetSurfaces() {
		rootPath := absPath(root, src.GetPath())
		unit := nearestParseUnit(parseByRoot, rootPath)
		surfaces = append(surfaces, &healthv1.DependencyHealthSurface{
			Id:             firstNonEmpty(src.GetId(), filepath.Base(rootPath)),
			Kind:           enumSuffix(src.GetKind().String(), "SURFACE_KIND_"),
			Language:       surfaceLanguage(unit, rootPath),
			Framework:      frameworkFromRoot(rootPath),
			RootPath:       rootPath,
			ParseUnitRoot:  parseUnitRoot(unit, root),
			ConfigPath:     parseUnitConfig(unit, root),
			Status:         enumSuffix(src.GetStatus().String(), "SURFACE_STATUS_"),
			PackageManager: packageManagerFromRoot(rootPath),
			Confidence:     bestConfidence(src.GetEvidence()),
		})
	}
	if len(surfaces) == 0 {
		return fallbackSurfaceInventory(scenarioDir, "Code Facts returned no surfaces")
	}
	sort.Slice(surfaces, func(i, j int) bool { return surfaces[i].GetId() < surfaces[j].GetId() })
	return surfaceInventory{Surfaces: surfaces}
}

func fallbackSurfaceInventory(scenarioDir, reason string) surfaceInventory {
	var surfaces []*healthv1.DependencyHealthSurface
	for _, spec := range []struct {
		id   string
		kind string
		dir  string
	}{
		{id: "api", kind: "api", dir: "api"},
		{id: "cli", kind: "cli", dir: "cli"},
		{id: "ui", kind: "ui", dir: "ui"},
	} {
		root := filepath.Join(scenarioDir, spec.dir)
		if !dirExists(root) {
			continue
		}
		surfaces = append(surfaces, &healthv1.DependencyHealthSurface{
			Id:             spec.id,
			Kind:           spec.kind,
			Language:       languageFromRoot(root),
			Framework:      frameworkFromRoot(root),
			RootPath:       root,
			ParseUnitRoot:  root,
			ConfigPath:     configPathFromRoot(root),
			Status:         "known",
			PackageManager: packageManagerFromRoot(root),
			Confidence:     0.4,
		})
	}
	return surfaceInventory{Surfaces: surfaces, DegradedReason: reason}
}

func nearestParseUnit(units map[string]*factsv1.ParseUnit, root string) *factsv1.ParseUnit {
	root = filepath.Clean(root)
	var best *factsv1.ParseUnit
	bestLen := -1
	for unitRoot, unit := range units {
		if strings.HasPrefix(root, unitRoot) && len(unitRoot) > bestLen {
			best = unit
			bestLen = len(unitRoot)
		}
	}
	return best
}

func surfaceLanguage(unit *factsv1.ParseUnit, root string) string {
	if unit != nil && strings.TrimSpace(unit.GetLanguage()) != "" {
		return normalizedLanguage(unit.GetLanguage())
	}
	return languageFromRoot(root)
}

func normalizedLanguage(language string) string {
	value := strings.ToLower(strings.TrimSpace(language))
	switch value {
	case "js", "javascript":
		return "javascript"
	case "ts", "typescript":
		return "typescript"
	case "py", "python3":
		return "python"
	default:
		return value
	}
}

func parseUnitRoot(unit *factsv1.ParseUnit, root string) string {
	if unit == nil {
		return ""
	}
	return absPath(root, unit.GetRootPath())
}

func parseUnitConfig(unit *factsv1.ParseUnit, root string) string {
	if unit == nil || unit.GetConfigPath() == "" {
		return ""
	}
	return absPath(root, unit.GetConfigPath())
}

func absPath(root, path string) string {
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	return filepath.Clean(filepath.Join(root, filepath.FromSlash(path)))
}

func enumSuffix(value, prefix string) string {
	value = strings.TrimPrefix(strings.ToUpper(strings.TrimSpace(value)), prefix)
	value = strings.ToLower(value)
	if value == "unspecified" {
		return "unknown"
	}
	return value
}

func bestConfidence(evidence []*factsv1.Evidence) float64 {
	best := 0.0
	for _, ev := range evidence {
		if ev.GetConfidence() > best {
			best = ev.GetConfidence()
		}
	}
	if best == 0 {
		return 0.8
	}
	return best
}

func languageFromRoot(root string) string {
	switch {
	case fileExists(filepath.Join(root, "go.mod")):
		return "go"
	case fileExists(filepath.Join(root, "tsconfig.json")):
		return "typescript"
	case fileExists(filepath.Join(root, "package.json")):
		return "javascript"
	case fileExists(filepath.Join(root, "pyproject.toml")), fileExists(filepath.Join(root, "requirements.txt")):
		return "python"
	default:
		return "unknown"
	}
}

func frameworkFromRoot(root string) string {
	data, err := os.ReadFile(filepath.Join(root, "package.json"))
	if err != nil {
		return ""
	}
	text := string(data)
	switch {
	case strings.Contains(text, `"vite"`) && strings.Contains(text, `"react"`):
		return "react-vite"
	case strings.Contains(text, `"react"`):
		return "react"
	case strings.Contains(text, `"vite"`):
		return "vite"
	default:
		return "node"
	}
}

func packageManagerFromRoot(root string) string {
	switch {
	case fileExists(filepath.Join(root, "pnpm-lock.yaml")):
		return "pnpm"
	case fileExists(filepath.Join(root, "package-lock.json")):
		return "npm"
	case fileExists(filepath.Join(root, "yarn.lock")):
		return "yarn"
	default:
		return ""
	}
}

func packageManagerForSurface(surface *healthv1.DependencyHealthSurface) string {
	if manager := strings.TrimSpace(surface.GetPackageManager()); manager != "" {
		return manager
	}
	return "pnpm"
}

func configPathFromRoot(root string) string {
	for _, name := range []string{"go.mod", "tsconfig.json", "package.json", "pyproject.toml", "requirements.txt"} {
		path := filepath.Join(root, name)
		if fileExists(path) {
			return path
		}
	}
	return ""
}

func (h *connectHandler) checkGoSurface(ctx context.Context, runner commandRunner, surface *healthv1.DependencyHealthSurface) []*healthv1.DependencyHealthFinding {
	root := surface.GetRootPath()
	modPath := filepath.Join(root, "go.mod")
	if !fileExists(modPath) {
		return []*healthv1.DependencyHealthFinding{readinessFinding("go."+surfaceID(surface)+".missing-go-mod", "WARNING", "Go surface has no go.mod", "A discovered Go surface does not have a go.mod at its root.", "Expose the correct parse-unit root through Code Facts or add a go.mod for this Go module.", surface, "dependency.go.mod_present", "missing go.mod", "go.mod at the Go surface root")}
	}
	var findings []*healthv1.DependencyHealthFinding
	for _, missing := range missingLocalReplaces(root, modPath) {
		findings = append(findings, readinessFinding("go."+surfaceID(surface)+".replace."+slug(missing), "ERROR", "Local replace target missing", "A go.mod replace directive points to a local path that does not exist.", "Update the replace directive so the local target exists, or remove the stale replacement.", surface, "dependency.go.local_replace_resolves", missing, "existing local replace target"))
	}
	out, err := runner.Run(ctx, root, "go", "mod", "tidy", "-diff")
	if err != nil {
		observed := strings.TrimSpace(out)
		if observed == "" {
			observed = err.Error()
		}
		findings = append(findings, readinessFinding("go."+surfaceID(surface)+".tidy-diff", "ERROR", "Go module metadata is not tidy", "go mod tidy -diff reported module metadata drift or failed.", "Run `cd "+filepath.ToSlash(root)+" && GOWORK=off go mod tidy`, then rerun dependency health.", surface, "dependency.go.tidy", observed, "go mod tidy -diff exits cleanly"))
	}
	return findings
}

func checkNodeSurface(surface *healthv1.DependencyHealthSurface) []*healthv1.DependencyHealthFinding {
	root := surface.GetRootPath()
	if !fileExists(filepath.Join(root, "package.json")) {
		return []*healthv1.DependencyHealthFinding{readinessFinding("node."+surfaceID(surface)+".missing-package-json", "WARNING", "JavaScript surface has no package.json", "A discovered JavaScript/TypeScript surface does not have a package.json at its root.", "Expose the correct package root through Code Facts or add package.json for this surface.", surface, "dependency.node.package_json_present", "missing package.json", "package.json at the JS/TS surface root")}
	}
	lockfiles := detectLockfiles(root)
	var findings []*healthv1.DependencyHealthFinding
	if len(lockfiles) == 0 {
		findings = append(findings, readinessFinding("node."+surfaceID(surface)+".missing-lockfile", "ERROR", "JavaScript lockfile missing", "A JavaScript/TypeScript dependency surface has package.json but no supported lockfile.", "Commit the correct lockfile for this package manager, usually pnpm-lock.yaml for Vrooli scenario surfaces.", surface, "dependency.node.lockfile_present", "no supported lockfile", "one lockfile: pnpm-lock.yaml, package-lock.json, or yarn.lock"))
	}
	if len(lockfiles) > 1 {
		findings = append(findings, readinessFinding("node."+surfaceID(surface)+".multiple-lockfiles", "ERROR", "Conflicting JavaScript lockfiles", "A JavaScript/TypeScript dependency surface has multiple package-manager lockfiles.", "Keep exactly one lockfile for the intended package manager and remove stale lockfiles.", surface, "dependency.node.single_lockfile", strings.Join(lockfiles, ", "), "exactly one package-manager lockfile"))
	}
	if !dirExists(filepath.Join(root, "node_modules")) {
		findings = append(findings, readinessFinding("node."+surfaceID(surface)+".node-modules-missing", "WARNING", "JavaScript install state is missing locally", "node_modules is absent for this JavaScript/TypeScript surface. This is local readiness, not dependency declaration drift.", "Install dependencies in the reported workspace without changing dependency declarations, for example `pnpm install --ignore-workspace` when pnpm is the intended manager.", surface, "dependency.node.install_state", "node_modules missing", "local install state present when local execution needs it"))
	}
	return findings
}

func readReleaseAgePolicy(surfaceRoot, scenarioDir string) (releaseAgePolicy, bool, error) {
	for _, path := range candidatePNPMWorkspacePaths(surfaceRoot, scenarioDir) {
		if !fileExists(path) {
			continue
		}
		policy, err := parseReleaseAgePolicy(path)
		return policy, true, err
	}
	return releaseAgePolicy{}, false, nil
}

func candidatePNPMWorkspacePaths(surfaceRoot, scenarioDir string) []string {
	surfaceRoot = filepath.Clean(surfaceRoot)
	scenarioDir = filepath.Clean(scenarioDir)
	var paths []string
	for dir := surfaceRoot; ; dir = filepath.Dir(dir) {
		paths = append(paths, filepath.Join(dir, "pnpm-workspace.yaml"))
		if dir == scenarioDir || dir == filepath.Dir(dir) {
			break
		}
	}
	return paths
}

func parseReleaseAgePolicy(path string) (releaseAgePolicy, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return releaseAgePolicy{}, err
	}
	policy := releaseAgePolicy{Path: path}
	lines := strings.Split(string(data), "\n")
	for i := 0; i < len(lines); i++ {
		line := stripYAMLComment(lines[i])
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		key, value, ok := strings.Cut(trimmed, ":")
		if !ok {
			continue
		}
		switch strings.TrimSpace(key) {
		case "minimumReleaseAge":
			var minutes int
			if _, err := fmt.Sscanf(strings.TrimSpace(value), "%d", &minutes); err != nil {
				return releaseAgePolicy{}, fmt.Errorf("parse minimumReleaseAge in %s: %w", path, err)
			}
			policy.MinimumReleaseAge = minutes
			policy.HasMinimumReleaseAge = true
		case "minimumReleaseAgeExclude":
			inline := strings.TrimSpace(value)
			if inline == "[]" {
				continue
			}
			for j := i + 1; j < len(lines); j++ {
				next := stripYAMLComment(lines[j])
				if strings.TrimSpace(next) == "" {
					continue
				}
				if !strings.HasPrefix(next, " ") && !strings.HasPrefix(next, "\t") {
					break
				}
				item := strings.TrimSpace(next)
				if !strings.HasPrefix(item, "-") {
					continue
				}
				excluded := strings.TrimSpace(strings.TrimPrefix(item, "-"))
				excluded = strings.Trim(excluded, `"'`)
				if excluded != "" {
					policy.MinimumReleaseAgeExclude = append(policy.MinimumReleaseAgeExclude, excluded)
				}
			}
		}
	}
	return policy, nil
}

func stripYAMLComment(line string) string {
	inSingle := false
	inDouble := false
	for i, r := range line {
		switch r {
		case '\'':
			if !inDouble {
				inSingle = !inSingle
			}
		case '"':
			if !inSingle {
				inDouble = !inDouble
			}
		case '#':
			if !inSingle && !inDouble {
				return line[:i]
			}
		}
	}
	return line
}

func loadReleaseAgeExceptions(path string) []releaseAgeException {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var file releaseAgeExceptionFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil
	}
	return file.ReleaseAgeExceptions
}

func hasApprovedReleaseAgeException(exceptions []releaseAgeException, excluded string) bool {
	for _, exception := range exceptions {
		if !strings.EqualFold(strings.TrimSpace(exception.State), "approved") {
			continue
		}
		if strings.TrimSpace(exception.Rationale) == "" || strings.TrimSpace(exception.ReviewExpires) == "" {
			continue
		}
		if sameReleaseAgeException(exception, excluded) {
			return true
		}
	}
	return false
}

func sameReleaseAgeException(exception releaseAgeException, excluded string) bool {
	excluded = strings.TrimSpace(excluded)
	spec := strings.TrimSpace(exception.Spec)
	if spec != "" && strings.EqualFold(spec, excluded) {
		return true
	}
	name := strings.TrimSpace(exception.PackageName)
	if name == "" {
		return false
	}
	if strings.EqualFold(name, excluded) {
		return true
	}
	excludedName, _, hasVersion := strings.Cut(excluded, "@")
	if strings.HasPrefix(excluded, "@") {
		parts := strings.SplitN(strings.TrimPrefix(excluded, "@"), "@", 2)
		excludedName = "@" + parts[0]
		hasVersion = len(parts) == 2
	}
	return !hasVersion && strings.EqualFold(name, excludedName)
}

func releaseAgeFinding(id, severity, title, description, remediation string, surface *healthv1.DependencyHealthSurface, ruleID, observed, expected string) *healthv1.DependencyHealthFinding {
	return &healthv1.DependencyHealthFinding{
		Id:           "release-age." + slug(id),
		Severity:     severity,
		SourceDomain: "release-age",
		Title:        title,
		Description:  description,
		Remediation:  remediation,
		FilePath:     releaseAgeFilePath(surface),
		SurfaceId:    surface.GetId(),
		RuleId:       ruleID,
		Observed:     observed,
		Expected:     expected,
	}
}

func releaseAgeFilePath(surface *healthv1.DependencyHealthSurface) string {
	for _, path := range candidatePNPMWorkspacePaths(surface.GetRootPath(), filepath.Dir(surface.GetRootPath())) {
		if fileExists(path) {
			return relScenarioPath(path)
		}
	}
	return relScenarioPath(filepath.Join(surface.GetRootPath(), "pnpm-workspace.yaml"))
}

func isJavaScriptSurface(surface *healthv1.DependencyHealthSurface) bool {
	switch normalizedLanguage(surface.GetLanguage()) {
	case "javascript", "typescript":
		return true
	default:
		return false
	}
}

func (h *connectHandler) checkVersion(ctx context.Context, dir string, runner commandRunner, command string, constraintRaw string) []*healthv1.DependencyHealthFinding {
	out, err := runner.Run(ctx, dir, command, versionArgs(command)...)
	if err != nil {
		return []*healthv1.DependencyHealthFinding{{
			Id:           "readiness.version." + command + ".unavailable",
			Severity:     "ERROR",
			SourceDomain: "readiness",
			Title:        "Runtime version unavailable",
			Description:  fmt.Sprintf("SDA could not read %s version.", command),
			Remediation:  fmt.Sprintf("Install or activate %s matching %s.", command, constraintRaw),
			RuleId:       "dependency.runtime.version",
			Observed:     err.Error(),
			Expected:     command + " " + constraintRaw,
		}}
	}
	version, err := parseVersionOutput(out)
	if err != nil {
		return []*healthv1.DependencyHealthFinding{{
			Id:           "readiness.version." + command + ".unparseable",
			Severity:     "ERROR",
			SourceDomain: "readiness",
			Title:        "Runtime version unparseable",
			Description:  fmt.Sprintf("SDA could not parse %s version output.", command),
			Remediation:  fmt.Sprintf("Ensure %s prints a parseable version matching %s.", command, constraintRaw),
			RuleId:       "dependency.runtime.version",
			Observed:     strings.TrimSpace(out),
			Expected:     command + " " + constraintRaw,
		}}
	}
	min, err := parseVersion(strings.TrimPrefix(constraintRaw, ">="))
	if err != nil {
		return nil
	}
	if !version.atLeast(min) {
		return []*healthv1.DependencyHealthFinding{{
			Id:           "readiness.version." + command + ".too-old",
			Severity:     "ERROR",
			SourceDomain: "readiness",
			Title:        "Runtime version too old",
			Description:  fmt.Sprintf("%s version %s is below required %s.", command, version, constraintRaw),
			Remediation:  fmt.Sprintf("Install or activate %s %s.", command, constraintRaw),
			RuleId:       "dependency.runtime.version",
			Observed:     version.String(),
			Expected:     constraintRaw,
		}}
	}
	return nil
}

type version struct {
	major int
	minor int
	patch int
}

func (v version) String() string {
	return fmt.Sprintf("%d.%d.%d", v.major, v.minor, v.patch)
}

func (v version) atLeast(min version) bool {
	if v.major != min.major {
		return v.major > min.major
	}
	if v.minor != min.minor {
		return v.minor > min.minor
	}
	return v.patch >= min.patch
}

var versionNumberRE = regexp.MustCompile(`\d+(?:\.\d+){0,2}`)

func parseVersionOutput(output string) (version, error) {
	match := versionNumberRE.FindString(strings.TrimSpace(output))
	if match == "" {
		return version{}, fmt.Errorf("could not parse version from %q", strings.TrimSpace(output))
	}
	return parseVersion(match)
}

func parseVersion(raw string) (version, error) {
	parts := strings.Split(strings.TrimSpace(raw), ".")
	values := []int{0, 0, 0}
	if len(parts) == 0 || len(parts) > 3 {
		return version{}, fmt.Errorf("invalid version %q", raw)
	}
	for i, part := range parts {
		var value int
		if _, err := fmt.Sscanf(part, "%d", &value); err != nil {
			return version{}, fmt.Errorf("invalid version %q: %w", raw, err)
		}
		values[i] = value
	}
	return version{major: values[0], minor: values[1], patch: values[2]}, nil
}

func versionArgs(command string) []string {
	if command == "go" {
		return []string{"version"}
	}
	return []string{"--version"}
}

func missingLocalReplaces(moduleRoot, modPath string) []string {
	data, err := os.ReadFile(modPath)
	if err != nil {
		return nil
	}
	var missing []string
	inBlock := false
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}
		if inBlock {
			if line == ")" {
				inBlock = false
				continue
			}
			if miss := missingReplaceTarget(moduleRoot, line); miss != "" {
				missing = append(missing, miss)
			}
			continue
		}
		if strings.HasPrefix(line, "replace (") {
			inBlock = true
			continue
		}
		if strings.HasPrefix(line, "replace ") {
			if miss := missingReplaceTarget(moduleRoot, strings.TrimPrefix(line, "replace ")); miss != "" {
				missing = append(missing, miss)
			}
		}
	}
	sort.Strings(missing)
	return missing
}

var localReplaceRE = regexp.MustCompile(`^([^\s]+)(?:\s+v\S+)?\s+=>\s+(\S+)`)

func missingReplaceTarget(moduleRoot, line string) string {
	match := localReplaceRE.FindStringSubmatch(line)
	if match == nil || !isLocalPath(match[2]) {
		return ""
	}
	target := match[2]
	if !filepath.IsAbs(target) {
		target = filepath.Join(moduleRoot, filepath.FromSlash(target))
	}
	if dirExists(target) {
		return ""
	}
	return filepath.ToSlash(match[2])
}

func isLocalPath(path string) bool {
	return strings.HasPrefix(path, ".") || filepath.IsAbs(path)
}

func detectLockfiles(root string) []string {
	var out []string
	for _, name := range []string{"pnpm-lock.yaml", "package-lock.json", "yarn.lock"} {
		if fileExists(filepath.Join(root, name)) {
			out = append(out, name)
		}
	}
	return out
}

func readinessFinding(id, severity, title, description, remediation string, surface *healthv1.DependencyHealthSurface, ruleID, observed, expected string) *healthv1.DependencyHealthFinding {
	return &healthv1.DependencyHealthFinding{
		Id:           "readiness." + slug(id),
		Severity:     severity,
		SourceDomain: "readiness",
		Title:        title,
		Description:  description,
		Remediation:  remediation,
		FilePath:     relScenarioPath(surface.GetRootPath()),
		SurfaceId:    surface.GetId(),
		RuleId:       ruleID,
		Observed:     observed,
		Expected:     expected,
	}
}

func runtimeFinding(scenario, id, severity, title, description, remediation, ruleID, observed, expected string) *healthv1.DependencyHealthFinding {
	return &healthv1.DependencyHealthFinding{
		Id:           "runtime." + slug(id),
		Severity:     severity,
		SourceDomain: "runtime",
		Title:        title,
		Description:  description,
		Remediation:  remediation,
		FilePath:     filepath.Join("scenarios", scenario, ".vrooli", "service.json"),
		RuleId:       ruleID,
		Observed:     observed,
		Expected:     expected,
	}
}

func boolValue(value *structpb.Value) (bool, bool) {
	if value == nil {
		return false, false
	}
	if b, ok := value.GetKind().(*structpb.Value_BoolValue); ok {
		return b.BoolValue, true
	}
	return false, false
}

func resourceObserved(status *cliv1.ResourceStatus) string {
	if status == nil {
		return "missing status"
	}
	healthy, known := boolValue(status.GetHealthy())
	parts := []string{
		fmt.Sprintf("running=%t", status.GetRunning()),
	}
	if known {
		parts = append(parts, fmt.Sprintf("healthy=%t", healthy))
	} else {
		parts = append(parts, "healthy=unknown")
	}
	if msg := strings.TrimSpace(status.GetMessage()); msg != "" {
		parts = append(parts, "message="+msg)
	}
	return strings.Join(parts, " ")
}

func emptyAs(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func sectionWithFindingIDs(id, title, status, summary string, ids []string) *healthv1.DependencyHealthSection {
	out := section(id, title, status, summary)
	out.FindingIds = ids
	return out
}

func statusFromFindings(findings []*healthv1.DependencyHealthFinding, domain string) string {
	status := "pass"
	for _, finding := range findings {
		if domain != "" && finding.GetSourceDomain() != domain {
			continue
		}
		switch strings.ToUpper(strings.TrimSpace(finding.GetSeverity())) {
		case "ERROR", "CRITICAL", "HIGH":
			return "fail"
		case "WARNING", "WARN", "MEDIUM":
			status = "warn"
		}
	}
	return status
}

func summarizeReadiness(findings []*healthv1.DependencyHealthFinding, commandResults []*healthv1.DependencyHealthCommandResult) string {
	status := statusFromFindings(findings, "readiness")
	if status == "pass" {
		return fmt.Sprintf("Host commands, runtimes, modules, and packages passed readiness checks (%d command probe(s)).", len(commandResults))
	}
	return fmt.Sprintf("%d readiness finding(s) across host commands, runtimes, modules, and packages.", len(findingIDs(findings, "readiness")))
}

func findingIDs(findings []*healthv1.DependencyHealthFinding, domain string) []string {
	var ids []string
	for _, finding := range findings {
		if domain == "" || finding.GetSourceDomain() == domain {
			ids = append(ids, finding.GetId())
		}
	}
	sort.Strings(ids)
	return ids
}

func sortedKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sortedSet(values map[string]struct{}) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func scenarioDirFromSurfaces(surfaces []*healthv1.DependencyHealthSurface) string {
	for _, surface := range surfaces {
		root := strings.TrimSpace(surface.GetRootPath())
		if root != "" {
			return root
		}
	}
	return ""
}

func surfaceID(surface *healthv1.DependencyHealthSurface) string {
	return firstNonEmpty(surface.GetId(), filepath.Base(surface.GetRootPath()), "surface")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func slug(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

func relScenarioPath(path string) string {
	parts := strings.Split(filepath.ToSlash(path), "/scenarios/")
	if len(parts) < 2 {
		return filepath.ToSlash(path)
	}
	return "scenarios/" + parts[len(parts)-1]
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func (h *connectHandler) resolveScenariosDir() string {
	if h.scenariosDir == nil {
		return ""
	}
	return strings.TrimSpace(h.scenariosDir())
}

func section(id, title, status, summary string) *healthv1.DependencyHealthSection {
	return &healthv1.DependencyHealthSection{
		Id:      id,
		Title:   title,
		Status:  status,
		Summary: summary,
	}
}

func finalize(resp *healthv1.DependencyHealthResponse) {
	summary := &healthv1.DependencyHealthSummary{
		Sections:             int32(len(resp.Sections)),
		Surfaces:             int32(len(resp.Surfaces)),
		Findings:             int32(len(resp.Findings)),
		DegradedDependencies: int32(len(resp.DegradedDependencies)),
	}
	passed := true
	for _, section := range resp.Sections {
		if strings.EqualFold(strings.TrimSpace(section.GetStatus()), "pending") {
			passed = false
		}
	}
	for _, finding := range resp.Findings {
		switch strings.ToUpper(strings.TrimSpace(finding.GetSeverity())) {
		case "ERROR", "CRITICAL", "HIGH":
			summary.Errors++
			passed = false
		case "WARNING", "WARN", "MEDIUM":
			summary.Warnings++
		default:
			summary.Infos++
		}
	}
	resp.Summary = summary
	resp.Passed = passed && len(resp.DegradedDependencies) == 0
}

func driftFindingID(finding interfacegraph.DriftFinding) string {
	parts := []string{"graph", finding.Scenario, finding.Dependency, finding.Kind}
	for i, part := range parts {
		parts[i] = strings.ReplaceAll(strings.ToLower(strings.TrimSpace(part)), "_", "-")
	}
	return strings.Join(parts, ".")
}

func driftRemediation(kind string) string {
	switch strings.TrimSpace(kind) {
	case interfacegraph.DriftUndeclaredUsed:
		return "Declare the scenario dependency in .vrooli/service.json or remove the import-level usage."
	case interfacegraph.DriftDeclaredWithoutProof:
		return "Confirm the dependency is runtime-only, or remove the stale declaration if it is no longer used."
	default:
		return "Review declared scenario dependencies against the actual interface graph."
	}
}

func driftObserved(finding interfacegraph.DriftFinding) string {
	if finding.ActualEvidence && !finding.Declared {
		return fmt.Sprintf("%s imports %s without a matching declaration", finding.Scenario, finding.Dependency)
	}
	if finding.Declared && !finding.ActualEvidence {
		return fmt.Sprintf("%s declares %s without import-level evidence", finding.Scenario, finding.Dependency)
	}
	return finding.Message
}

func driftExpected(finding interfacegraph.DriftFinding) string {
	switch strings.TrimSpace(finding.Kind) {
	case interfacegraph.DriftUndeclaredUsed:
		return "Import-level scenario dependencies are declared in .vrooli/service.json."
	case interfacegraph.DriftDeclaredWithoutProof:
		return "Declared scenario dependencies have import-level evidence or documented runtime-only rationale."
	default:
		return "Declared dependency graph matches actual dependency evidence."
	}
}

var _ healthconnect.DependencyHealthServiceHandler = (*connectHandler)(nil)
