package dependencyhealth

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	healthv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/dependency_health"
)

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

func (h *connectHandler) evaluateSecurityHealth(ctx context.Context, _ string) (*healthv1.DependencyHealthSection, []*healthv1.DependencyHealthFinding, []*healthv1.DegradedDependency) {
	lookup := h.commandLookup
	if lookup == nil {
		lookup = exec.LookPath
	}
	if _, err := lookup("security-health"); err != nil {
		return section("security-index", "Security Health dependency index", "degraded", "Security Health CLI is unavailable; SDA skipped dependency index freshness status without running vulnerability scanners."), nil, []*healthv1.DegradedDependency{
			{
				Id:         "security-health-deps-status",
				Dependency: "security-health",
				Domain:     "security-index",
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
		return section("security-index", "Security Health dependency index", "degraded", "Security Health dependency index status is unavailable; SDA did not run vulnerability scanners."), nil, []*healthv1.DegradedDependency{
			{
				Id:         "security-health-deps-status",
				Dependency: "security-health deps status",
				Domain:     "security-index",
				Reason:     observed,
			},
		}
	}
	var status securityHealthDependencyStatus
	if err := json.Unmarshal([]byte(out), &status); err != nil {
		return section("security-index", "Security Health dependency index", "degraded", "Security Health dependency index status returned unparseable JSON."), nil, []*healthv1.DegradedDependency{
			{
				Id:         "security-health-deps-status",
				Dependency: "security-health deps status",
				Domain:     "security-index",
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
			Domain:     "security-index",
			Reason:     summary,
		})
	}
	return sectionWithFindingIDs("security-index", "Security Health dependency index", sectionStatus, summary, nil), nil, degraded
}
