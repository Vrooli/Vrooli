package dependencyhealth

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	vroolicli "github.com/vrooli/vrooli-cli-go"

	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
	healthv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/dependency_health"
	"google.golang.org/protobuf/types/known/structpb"
)

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

func (h *connectHandler) evaluateRuntime(ctx context.Context, scenario string) (*healthv1.DependencyHealthSection, []*healthv1.DependencyHealthFinding, []*healthv1.DegradedDependency) {
	manifest, err := loadDependencyManifest(filepath.Join(h.scenarioDir(ctx, scenario), ".vrooli", "service.json"))
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
