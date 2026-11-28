package dependencyhealth

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/gomodreconcile"

	healthv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/dependency_health"
)

// goModReplaceRuleID is the shared fix-class id for a Go surface that requires an
// in-repo module without the local replace that makes it build. It is both the
// dependency-health finding rule and the ScenarioValidationService Fix-class id.
const goModReplaceRuleID = "dependency.gomod.replace.missing"

// goModEvidenceRuleID flags Go packages under a scenario docs/ tree without
// their own module boundary: they compile into the enclosing module (often the
// repository root) and drag their imports into its requirements.
const goModEvidenceRuleID = "dependency.gomod.evidence.unfenced"

// goModTidyRuleID is the readiness finding for a Go surface whose go.mod or
// go.sum differs from what its import closure requires. The same convergence
// fix that adds missing replaces repairs it, through the dependency gate.
const goModTidyRuleID = "dependency.go.tidy"

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
	evidenceRoot := h.scenarioDir(ctx, scenario)
	evidence, err := gomodreconcile.UnfencedEvidence(evidenceRoot, false)
	if err != nil {
		degraded = append(degraded, &healthv1.DegradedDependency{
			Id:         "gomod-replace.evidence",
			Dependency: scenario,
			Domain:     "gomod-replace",
			Reason:     fmt.Sprintf("scan docs evidence: %v", err),
		})
	}
	for _, dir := range evidence {
		id := strings.Join([]string{"gomod-replace", "evidence", slug(dir)}, ".")
		ids = append(ids, id)
		findings = append(findings, &healthv1.DependencyHealthFinding{
			Id:           id,
			Severity:     "ERROR",
			SourceDomain: "gomod-replace",
			Title:        "Docs evidence joins a Go module",
			Description:  fmt.Sprintf("Go packages under %s sit in a docs/ tree with no module boundary, so Go compiles them as part of the enclosing module and adds their imports to its requirements.", dir),
			Remediation:  "Fence the evidence tree with the docs-evidence convention: a go.mod at the top of its docs/ directory containing `module github.com/vrooli/vrooli/scenarios/" + scenario + "/docs-evidence` and the repository go directive (see docs/package-governance.md, Go module convergence). Then run `" + reconcileCLIName + " deps reconcile --all-modules --apply`.",
			FilePath:     relScenarioPath(filepath.Join(evidenceRoot, filepath.FromSlash(dir))),
			RuleId:       goModEvidenceRuleID,
			Observed:     "Go packages under " + dir + " without a go.mod at or below docs/",
			Expected:     "docs/go.mod fencing retained source",
		})
	}
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
			observed := fmt.Sprintf("require %s without a local replace", m.Module)
			expected := fmt.Sprintf("replace %s => %s", m.Module, m.RelPath)
			remediation := fmt.Sprintf("Run `%s deps reconcile --scenario %s --apply` to add `replace %s => %s`.", reconcileCLIName, scenario, m.Module, m.RelPath)
			if m.AddRequire {
				observed = fmt.Sprintf("import %s without require or local replace", m.Module)
				expected = fmt.Sprintf("require %s v0.0.0 and replace %s => %s", m.Module, m.Module, m.RelPath)
				remediation = fmt.Sprintf("Run `%s deps reconcile --scenario %s --apply` to add `require %s v0.0.0` and `replace %s => %s`.", reconcileCLIName, scenario, m.Module, m.Module, m.RelPath)
			}
			ids = append(ids, id)
			findings = append(findings, &healthv1.DependencyHealthFinding{
				Id:           id,
				Severity:     "ERROR",
				SourceDomain: "gomod-replace",
				Title:        "Missing in-repo module replace",
				Description:  fmt.Sprintf("Surface %s requires in-repo module %s without a local replace; go build (and scenario restart) will fail with a missing go.sum entry.", surfaceID(surface), m.Module),
				Remediation:  remediation,
				FilePath:     relScenarioPath(goModPath),
				RuleId:       goModReplaceRuleID,
				Observed:     observed,
				Expected:     expected,
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
