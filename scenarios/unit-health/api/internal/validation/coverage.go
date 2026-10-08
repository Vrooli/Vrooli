package validation

import (
	"fmt"
	"sort"
	"strings"

	"unit-health/internal/adapters"
)

// defaultCoverageThreshold is the advisory per-file coverage gate used when a
// target has no canonical `unit.policy_profile` coverage floor for a workspace.
// It is a non-zero advisory (operator-chosen): LOW_COVERAGE is L4/advisory in
// the maturity ladder, so it warns and surfaces under-covered files but never
// gates maturity. Template-derived scenarios raise it through policy classes,
// not through legacy ad hoc coverage keys.
const defaultCoverageThreshold = 50.0

// maxPerFileCoverageFindings bounds how many per-file LOW_COVERAGE findings a
// single workspace emits so a large, low-coverage codebase does not flood the
// report; the per-workspace roll-up still reports the true total.
const maxPerFileCoverageFindings = 25

// coverageThresholdPercent resolves the coverage gate from the canonical
// `unit.policy_profile` class that owns the workspace. Old ad hoc coverage
// keys are intentionally ignored: after the policy-profile cutover, legacy
// shapes should be migrated instead of silently honored.
func coverageThresholdPercent(scenarioRoot string, ws Workspace) float64 {
	profile, _, ok, _ := loadUnitPolicyProfile("", scenarioRoot, "")
	if !ok {
		return defaultCoverageThreshold
	}
	for _, role := range profile.RequiredRoles {
		class, exists := profile.PolicyClasses[role.PolicyClass]
		if !exists {
			continue
		}
		if role.Match.Path != "" && pathMatches(role.Match.Path, ws.RootPath, scenarioRoot) {
			if class.Coverage.MinimumPercent > 0 {
				return class.Coverage.MinimumPercent
			}
			return defaultCoverageThreshold
		}
	}
	return defaultCoverageThreshold
}

// fileCoverage accumulates covered/total units for one file.
type fileCoverage struct {
	covered int64
	total   int64
}

func (f fileCoverage) percent() float64 {
	if f.total == 0 {
		return 0
	}
	return float64(f.covered) / float64(f.total) * 100
}

// analyzeCoverage parses coverage artifacts produced by the executed plan and
// turns them into per-file CoverageTargets plus LOW_COVERAGE/COVERAGE_ABSENT
// findings. It runs only after execution, so artifacts are fresh.
func analyzeCoverage(scenario, scenarioRoot string, workspaces []Workspace, now string) ([]CoverageTarget, []Finding) {
	var targets []CoverageTarget
	var findings []Finding

	for _, ws := range workspaces {
		threshold := coverageThresholdPercent(scenarioRoot, ws)
		if ws.CoverageCommand == "" {
			// Coverage was not part of the plan for this workspace; the missing
			// coverage config is already reported by the planner.
			continue
		}
		fileCov, ok := readWorkspaceCoverage(ws)
		if !ok {
			findings = append(findings, Finding{
				ID:           codeCoverageAbsent + "-" + ws.ID,
				Scenario:     scenario,
				WorkspaceID:  ws.ID,
				Language:     ws.Language,
				Code:         codeCoverageAbsent,
				Category:     "coverage",
				Severity:     codeSeverity[codeCoverageAbsent],
				FilePath:     ws.RootPath,
				Message:      fmt.Sprintf("Workspace %q ran a coverage command but no coverage artifact was found.", ws.ID),
				Evidence:     "no adapter-declared coverage artifact under the workspace",
				Expected:     "A coverage artifact written by the canonical coverage command.",
				Observed:     "no coverage artifact",
				WhyItMatters: "Without a coverage artifact, hardening depth for this workspace is invisible.",
				Remediation:  "Ensure the adapter-declared coverage artifact is written to the workspace by the governed coverage command.",
				CreatedAt:    now,
			})
			continue
		}

		names := make([]string, 0, len(fileCov))
		for name := range fileCov {
			names = append(names, name)
		}
		sort.Strings(names)
		var agg fileCoverage
		type belowFile struct {
			name    string
			percent float64
		}
		var below []belowFile
		for _, name := range names {
			fc := fileCov[name]
			agg.covered += fc.covered
			agg.total += fc.total
			status := coverageStatus(fc.percent(), threshold)
			targets = append(targets, CoverageTarget{
				ID:              ws.ID + ":" + name,
				Language:        ws.Language,
				SurfaceID:       ws.ID,
				FilePath:        name,
				CoveredLines:    fc.covered,
				TotalLines:      fc.total,
				CoveragePercent: round2(fc.percent()),
				Threshold:       threshold,
				Status:          status,
			})
			if status == "below" {
				below = append(below, belowFile{name: name, percent: fc.percent()})
			}
		}

		// B1: emit a per-file LOW_COVERAGE finding for each under-threshold file
		// (lowest coverage first), bounded so a large codebase does not flood the
		// report. The per-workspace roll-up below still reports the true total.
		if threshold > 0 {
			sort.SliceStable(below, func(i, j int) bool { return below[i].percent < below[j].percent })
			for i, bf := range below {
				if i >= maxPerFileCoverageFindings {
					break
				}
				findings = append(findings, Finding{
					ID:           codeLowCoverage + "-" + ws.ID + "-" + fileSlug(bf.name),
					Scenario:     scenario,
					WorkspaceID:  ws.ID,
					Language:     ws.Language,
					Code:         codeLowCoverage,
					Category:     "coverage",
					Severity:     codeSeverity[codeLowCoverage],
					FilePath:     bf.name,
					Message:      fmt.Sprintf("File %s coverage %.1f%% is below the %.1f%% threshold.", bf.name, bf.percent, threshold),
					Evidence:     fmt.Sprintf("%d/%d units covered", fileCov[bf.name].covered, fileCov[bf.name].total),
					Expected:     fmt.Sprintf("Per-file coverage at or above %.1f%%.", threshold),
					Observed:     fmt.Sprintf("%.1f%%", bf.percent),
					WhyItMatters: "An under-covered file hides untested branches where regressions slip through.",
					Remediation:  "Add tests exercising the uncovered branches of this file.",
					CreatedAt:    now,
				})
			}
		}

		if threshold > 0 && agg.total > 0 && agg.percent() < threshold {
			findings = append(findings, Finding{
				ID:           codeLowCoverage + "-" + ws.ID,
				Scenario:     scenario,
				WorkspaceID:  ws.ID,
				Language:     ws.Language,
				Code:         codeLowCoverage,
				Category:     "coverage",
				Severity:     codeSeverity[codeLowCoverage],
				FilePath:     ws.RootPath,
				Message:      fmt.Sprintf("Workspace %q coverage %.1f%% is below the %.1f%% threshold.", ws.ID, agg.percent(), threshold),
				Evidence:     fmt.Sprintf("%d/%d units covered across %d file(s)", agg.covered, agg.total, len(names)),
				Expected:     fmt.Sprintf("Coverage at or above %.1f%%.", threshold),
				Observed:     fmt.Sprintf("%.1f%%", agg.percent()),
				WhyItMatters: "Low coverage leaves behavior unverified and regressions undetected.",
				Remediation:  "Add tests for the least-covered files (see the per-file coverage targets) until the threshold is met.",
				CreatedAt:    now,
			})
		}
	}
	return targets, findings
}

// fileSlug makes a stable, id-safe fragment from a file path for finding IDs.
func fileSlug(name string) string {
	return strings.NewReplacer("/", "_", "\\", "_", ":", "_", " ", "_").Replace(name)
}

func coverageStatus(percent, threshold float64) string {
	switch {
	case threshold > 0 && percent < threshold:
		return "below"
	case percent == 0:
		return "uncovered"
	default:
		return "covered"
	}
}

func round2(v float64) float64 {
	return float64(int64(v*100+0.5)) / 100
}

// readWorkspaceCoverage finds and parses the workspace's coverage artifact,
// preferring language-canonical locations. It returns false when no artifact
// exists.
func readWorkspaceCoverage(ws Workspace) (map[string]fileCoverage, bool) {
	artifacts := make([]adapters.Artifact, 0, len(ws.TestArtifacts))
	for _, artifact := range ws.TestArtifacts {
		artifacts = append(artifacts, adapters.Artifact{Label: artifact.Label, Kind: artifact.Kind, Path: artifact.Reference})
	}
	if len(artifacts) == 0 {
		// Compatibility for hand-built contract fixtures that predate typed
		// artifact declarations. Production plans always carry artifacts.
		artifacts = adapters.DefaultCoverageArtifacts(ws.Language)
	}
	metrics, ok := adapters.ReadCoverage(ws.RootPath, artifacts)
	if !ok {
		return nil, false
	}
	coverage := make(map[string]fileCoverage, len(metrics))
	for name, metric := range metrics {
		coverage[name] = fileCoverage{covered: metric.Covered, total: metric.Total}
	}
	return coverage, true
}
