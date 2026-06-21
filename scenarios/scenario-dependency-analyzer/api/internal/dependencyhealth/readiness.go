package dependencyhealth

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"

	healthv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/dependency_health"
)

func (h *connectHandler) evaluateReadiness(ctx context.Context, scenario string, useCache bool) ([]*healthv1.DependencyHealthSurface, *healthv1.DependencyHealthSection, *healthv1.DependencyHealthSection, []*healthv1.DependencyHealthFinding, []*healthv1.DependencyHealthCommandResult, []*healthv1.DegradedDependency) {
	scenarioDir := h.scenarioDir(ctx, scenario)
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
