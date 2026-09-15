package dependencyhealth

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/interfacegraph"

	healthv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/dependency_health"
)

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
