package dependencyhealth

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"scenario-dependency-analyzer/internal/gomodreconcile"

	healthv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/dependency_health"
)

// goModReplaceRuleID is the shared fix-class id for a Go surface that requires an
// in-repo module without the local replace that makes it build. It is both the
// dependency-health finding rule and the ScenarioValidationService Fix-class id.
const goModReplaceRuleID = "dependency.gomod.replace.missing"

// evaluateGoModReplace flags every Go surface that requires an in-repo module
// without a matching local replace — the fan-out gap that breaks `go build` at
// restart time. Findings are ERROR (gating) so the test-genie dependencies phase
// fails before a human hits the broken restart, and each names the reconcile
// command that repairs it.
func (h *connectHandler) evaluateGoModReplace(ctx context.Context, scenario string, surfaces []*healthv1.DependencyHealthSurface) (*healthv1.DependencyHealthSection, []*healthv1.DependencyHealthFinding, []*healthv1.DegradedDependency) {
	repoRoot := filepath.Dir(h.resolveScenariosDir())
	if strings.TrimSpace(repoRoot) == "" || repoRoot == "." {
		return section("gomod-replace", "In-repo go.mod replaces", "degraded", "Repo root could not be resolved."), nil, []*healthv1.DegradedDependency{{
			Id:         "gomod-replace",
			Dependency: "scenario-dependency-analyzer",
			Domain:     "gomod-replace",
			Reason:     "repo root unavailable for go.mod replace reconciliation",
		}}
	}
	topo, err := gomodreconcile.LoadTopology(repoRoot)
	if err != nil {
		return section("gomod-replace", "In-repo go.mod replaces", "degraded", "In-repo module topology could not be read."), nil, []*healthv1.DegradedDependency{{
			Id:         "gomod-replace",
			Dependency: "scenario-dependency-analyzer",
			Domain:     "gomod-replace",
			Reason:     fmt.Sprintf("load module topology: %v", err),
		}}
	}

	var findings []*healthv1.DependencyHealthFinding
	var degraded []*healthv1.DegradedDependency
	ids := make([]string, 0)
	for _, surface := range surfaces {
		if !strings.EqualFold(strings.TrimSpace(surface.GetLanguage()), "go") {
			continue
		}
		goModPath := filepath.Join(surface.GetRootPath(), "go.mod")
		if !fileExists(goModPath) {
			continue
		}
		missing, err := gomodreconcile.Plan(ctx, goModPath, topo)
		if err != nil {
			degraded = append(degraded, &healthv1.DegradedDependency{
				Id:         "gomod-replace." + slug(surfaceID(surface)),
				Dependency: surfaceID(surface),
				Domain:     "gomod-replace",
				Reason:     fmt.Sprintf("plan go.mod replaces: %v", err),
			})
			continue
		}
		for _, m := range missing {
			id := strings.Join([]string{"gomod-replace", slug(surfaceID(surface)), slug(m.Module)}, ".")
			ids = append(ids, id)
			findings = append(findings, &healthv1.DependencyHealthFinding{
				Id:           id,
				Severity:     "ERROR",
				SourceDomain: "gomod-replace",
				Title:        "Missing in-repo module replace",
				Description:  fmt.Sprintf("Surface %s requires in-repo module %s without a local replace; go build (and scenario restart) will fail with a missing go.sum entry.", surfaceID(surface), m.Module),
				Remediation:  fmt.Sprintf("Run `%s deps reconcile --scenario %s --apply` to add `replace %s => %s`.", reconcileCLIName, scenario, m.Module, m.RelPath),
				FilePath:     relScenarioPath(goModPath),
				RuleId:       goModReplaceRuleID,
				Observed:     fmt.Sprintf("require %s without a local replace", m.Module),
				Expected:     fmt.Sprintf("replace %s => %s", m.Module, m.RelPath),
			})
		}
	}

	status := "pass"
	summary := "All Go surfaces declare local replaces for their in-repo module requires."
	if len(findings) > 0 {
		status = statusFromFindings(findings, "gomod-replace")
		summary = fmt.Sprintf("%d Go surface(s) require an in-repo module without a local replace.", len(findings))
	}
	return sectionWithFindingIDs("gomod-replace", "In-repo go.mod replaces", status, summary, ids), findings, degraded
}

// reconcileCLIName is the operator command that repairs the finding.
const reconcileCLIName = "scenario-dependency-analyzer"
