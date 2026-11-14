package validation

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"unit-health/internal/adapterregistry"
	"unit-health/internal/adapters"
)

// defaultCoverageThreshold is the advisory per-file coverage target used when
// a workspace's policy class declares no `coverage.minimum_percent`.
// LOW_COVERAGE is advisory in the maturity ladder: it surfaces the gap to the
// target but never fails the phase. Only a scenario's `coverage.committed`
// floor gates (see COVERAGE_BELOW_COMMITTED).
const defaultCoverageThreshold = 50.0

// committedCoverageNoiseMargin is how far below its committed floor a metric
// may read before the gate fails. Asynchronous UI tests can cover a branch or
// two differently from run to run (measured: 1.6 points of branch coverage on
// a 145-branch UI between a loaded and an idle host), so a floor compared
// exactly would flake. A real regression moves coverage by more than this.
const committedCoverageNoiseMargin = 1.0

// maxPerFileCoverageFindings bounds how many per-file LOW_COVERAGE findings a
// single workspace emits so a large, low-coverage codebase does not flood the
// report; the per-workspace roll-up still reports the true total.
const maxPerFileCoverageFindings = 25

// workspaceCoveragePolicy is the coverage contract one workspace owes, resolved
// from the `unit.policy_profile` class that owns it. Target is the policy goal
// (`coverage.minimum_percent`); Committed is the scenario's per-metric floor
// (`coverage.committed`), the only coverage value that fails the unit phase.
// A class without a commitment is measured and reported, not gated.
type workspaceCoveragePolicy struct {
	Target    float64
	Committed map[string]float64
	// ExpectsCommitment marks classes the templates gate by default (see
	// adapterregistry.CoverageCommittedByDefault). Such a class without a
	// commitment is reported, because its coverage can regress silently.
	ExpectsCommitment bool
}

// commitmentExpected reports whether a policy class is gated by default.
func commitmentExpected(class unitPolicyClass) bool {
	return adapterregistry.CoverageCommittedByDefault(class.Adapter.ID)
}

func coveragePolicyForWorkspace(scenarioRoot string, ws Workspace) workspaceCoveragePolicy {
	class := policyClassForWorkspace(scenarioRoot, ws)
	policy := workspaceCoveragePolicy{Target: defaultCoverageThreshold, Committed: class.Coverage.Committed, ExpectsCommitment: commitmentExpected(class)}
	if class.Coverage.MinimumPercent > 0 {
		policy.Target = class.Coverage.MinimumPercent
	}
	return policy
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

// analyzeCoverage parses the coverage artifacts written by this run's coverage
// commands and turns them into per-file CoverageTargets plus coverage
// findings. coverageRuns maps workspace ID to whether its coverage command
// passed; a workspace whose coverage command did not run this time is skipped
// so a stale artifact is never judged. Only a passing run is compared with the
// committed floor: when tests fail, the test failure is the finding.
func analyzeCoverage(scenario, scenarioRoot string, workspaces []Workspace, coverageRuns map[string]bool, now string) ([]CoverageTarget, []Finding) {
	var targets []CoverageTarget
	var findings []Finding

	for _, ws := range workspaces {
		passed, ran := coverageRuns[ws.ID]
		if ws.CoverageCommand == "" || !ran {
			// Coverage was not part of this run for the workspace; a missing
			// coverage config is already reported by the planner.
			continue
		}
		policy := coveragePolicyForWorkspace(scenarioRoot, ws)
		threshold := policy.Target
		report, ok := readWorkspaceCoverage(ws)
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
			if passed && len(policy.Committed) > 0 {
				findings = append(findings, committedCoverageFindings(scenario, ws, policy, nil, now)...)
			}
			continue
		}
		fileCov := report.Files

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
			fc := fileCoverage{covered: fileCov[name].Covered, total: fileCov[name].Total}
			if fc.total == 0 {
				continue
			}
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
		if agg.total == 0 {
			findings = append(findings, Finding{
				ID:           codeCoverageAbsent + "-" + ws.ID,
				Scenario:     scenario,
				WorkspaceID:  ws.ID,
				Language:     ws.Language,
				Code:         codeCoverageAbsent,
				Category:     "coverage",
				Severity:     codeSeverity[codeCoverageAbsent],
				FilePath:     ws.RootPath,
				Message:      fmt.Sprintf("Workspace %q coverage artifact contains no executable units.", ws.ID),
				Evidence:     "coverage artifact contained zero measured units",
				Expected:     "Coverage data for executable code, when present.",
				Observed:     "no executable units measured",
				WhyItMatters: "Without measured units, coverage for this workspace cannot be assessed.",
				Remediation:  "Check whether the workspace contains executable code and whether coverage includes it.",
				CreatedAt:    now,
			})
			continue
		}

		if passed && len(policy.Committed) > 0 {
			findings = append(findings, committedCoverageFindings(scenario, ws, policy, report.Totals, now)...)
		}
		if passed && len(policy.Committed) == 0 && policy.ExpectsCommitment {
			findings = append(findings, missingCommitmentFinding(scenario, ws, report.Totals, policy.Target, now))
		}

		// Emit a per-file LOW_COVERAGE finding for each under-target file
		// (lowest coverage first), bounded so a large codebase does not flood
		// the report. The workspace roll-up below reports the true total.
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
					Message:      fmt.Sprintf("File %s coverage %.1f%% is below the %.1f%% policy target.", bf.name, bf.percent, threshold),
					Evidence:     fmt.Sprintf("%d/%d units covered", fileCov[bf.name].Covered, fileCov[bf.name].Total),
					Expected:     fmt.Sprintf("Per-file coverage at or above %.1f%%.", threshold),
					Observed:     fmt.Sprintf("%.1f%%", bf.percent),
					WhyItMatters: "An under-covered file hides untested branches where regressions slip through.",
					Remediation:  "Add tests exercising the uncovered branches of this file.",
					CreatedAt:    now,
				})
			}
		}

		if gaps := targetGaps(report.Totals, threshold); len(gaps) > 0 {
			findings = append(findings, Finding{
				ID:           codeLowCoverage + "-" + ws.ID,
				Scenario:     scenario,
				WorkspaceID:  ws.ID,
				Language:     ws.Language,
				Code:         codeLowCoverage,
				Category:     "coverage",
				Severity:     codeSeverity[codeLowCoverage],
				FilePath:     ws.RootPath,
				Message:      fmt.Sprintf("Workspace %q coverage is below the %s%% policy target: %s.", ws.ID, formatPercent(threshold), strings.Join(gaps, ", ")),
				Evidence:     fmt.Sprintf("%s; %d file(s) measured", describeTotals(report.Totals), len(names)),
				Expected:     fmt.Sprintf("Every measured coverage metric at or above the %s%% policy target (unit.policy_profile coverage.minimum_percent).", formatPercent(threshold)),
				Observed:     strings.Join(gaps, ", "),
				WhyItMatters: "This is recorded coverage debt: behavior below the target is unverified. It does not fail the phase; only a drop below the committed floor does.",
				Remediation:  "Add tests for the least-covered files (see the per-file coverage targets), then raise the committed floor with `unit-health fix apply <scenario> --rules " + codeCoverageCommitmentBehind + "`.",
				CreatedAt:    now,
			})
		}
	}
	return targets, findings
}

// committedCoverageFindings compares a passing run's workspace totals with the
// scenario's committed floor. A metric below its floor, or a committed metric
// the artifact does not report, is COVERAGE_BELOW_COMMITTED and fails the
// phase. A floor the measurement has outgrown (while still under the target) is
// COVERAGE_COMMITMENT_BEHIND, an advisory prompt to ratchet it upward.
func committedCoverageFindings(scenario string, ws Workspace, policy workspaceCoveragePolicy, totals map[string]adapters.CoverageMetric, now string) []Finding {
	var below, unmeasured, raisable []string
	for _, metric := range committedMetricOrder(policy.Committed) {
		floor := policy.Committed[metric]
		total, ok := totals[metric]
		if !ok {
			unmeasured = append(unmeasured, metric)
			continue
		}
		measured := total.Percent()
		if measured < floor-committedCoverageNoiseMargin {
			below = append(below, fmt.Sprintf("%s %s%% < committed %s%% (%d/%d)", metric, formatPercent(round2(measured)), formatPercent(floor), total.Covered, total.Total))
			continue
		}
		if next := RatchetedCommitment(floor, measured, policy.Target); next > floor {
			raisable = append(raisable, fmt.Sprintf("%s %s%% -> %s%%", metric, formatPercent(floor), formatPercent(next)))
		}
	}
	committed := describeCommitment(policy.Committed)
	var findings []Finding
	if len(below) > 0 || len(unmeasured) > 0 {
		observed := below
		if len(unmeasured) > 0 {
			observed = append(append([]string(nil), below...), "not reported by the coverage artifact: "+strings.Join(unmeasured, ", "))
		}
		findings = append(findings, Finding{
			ID:           codeCoverageBelowCommitted + "-" + ws.ID,
			Scenario:     scenario,
			WorkspaceID:  ws.ID,
			Language:     ws.Language,
			Code:         codeCoverageBelowCommitted,
			Category:     "coverage",
			Severity:     codeSeverity[codeCoverageBelowCommitted],
			FilePath:     ws.RootPath,
			Message:      fmt.Sprintf("Workspace %q tests passed but coverage fell below the scenario's committed floor: %s.", ws.ID, strings.Join(observed, "; ")),
			Evidence:     fmt.Sprintf("measured %s; committed %s; policy target %s%%", describeTotals(totals), committed, formatPercent(policy.Target)),
			Expected:     fmt.Sprintf("Coverage at or above the committed floor %s (unit.policy_profile coverage.committed).", committed),
			Observed:     strings.Join(observed, "; "),
			WhyItMatters: "The committed floor is the coverage this scenario has already earned; falling below it means behavior that was tested no longer is.",
			Remediation:  "Add or restore tests for the code whose coverage dropped (see the per-file coverage targets). Do not lower coverage.committed to make the phase pass.",
			CreatedAt:    now,
		})
	}
	if len(raisable) > 0 {
		findings = append(findings, Finding{
			ID:           codeCoverageCommitmentBehind + "-" + ws.ID,
			Scenario:     scenario,
			WorkspaceID:  ws.ID,
			Language:     ws.Language,
			Code:         codeCoverageCommitmentBehind,
			Category:     "coverage",
			Severity:     codeSeverity[codeCoverageCommitmentBehind],
			FilePath:     ws.RootPath,
			Message:      fmt.Sprintf("Workspace %q coverage has outgrown its committed floor: %s.", ws.ID, strings.Join(raisable, ", ")),
			Evidence:     fmt.Sprintf("measured %s; committed %s; policy target %s%%", describeTotals(totals), committed, formatPercent(policy.Target)),
			Expected:     "The committed floor tracks measured coverage (rounded down) until it reaches the policy target.",
			Observed:     strings.Join(raisable, ", "),
			WhyItMatters: "A floor that lags measured coverage lets earned coverage erode without failing the phase.",
			Remediation:  "Run `unit-health fix apply <scenario> --rules " + codeCoverageCommitmentBehind + "` after a passing coverage run to ratchet coverage.committed upward.",
			CreatedAt:    now,
		})
	}
	return findings
}

// missingCommitmentFinding reports a default-gated class with no committed
// floor: its coverage is measured but a regression fails nothing.
func missingCommitmentFinding(scenario string, ws Workspace, totals map[string]adapters.CoverageMetric, target float64, now string) Finding {
	return Finding{
		ID:           codeCoverageCommitmentBehind + "-" + ws.ID,
		Scenario:     scenario,
		WorkspaceID:  ws.ID,
		Language:     ws.Language,
		Code:         codeCoverageCommitmentBehind,
		Category:     "coverage",
		Severity:     codeSeverity[codeCoverageCommitmentBehind],
		FilePath:     ws.RootPath,
		Message:      fmt.Sprintf("Workspace %q has no committed coverage floor, so a coverage regression fails nothing.", ws.ID),
		Evidence:     fmt.Sprintf("measured %s; policy target %s%%", describeTotals(totals), formatPercent(target)),
		Expected:     "unit.policy_profile coverage.committed records the measured baseline for this class.",
		Observed:     "no coverage.committed",
		WhyItMatters: "Without a floor, earned coverage can erode silently.",
		Remediation:  "Run `unit-health fix apply <scenario> --rules " + codeCoverageCommitmentBehind + "` after a passing coverage run to commit the measured baseline.",
		CreatedAt:    now,
	}
}

// RatchetedCommitment is the floor a measurement earns: measured coverage
// rounded down to a whole percent, capped at the policy target, and never
// lower than the current floor. The cap keeps a scenario that exceeds its
// target free to move above it; a floor already above the target is kept.
func RatchetedCommitment(current, measured, target float64) float64 {
	earned := math.Floor(measured)
	if target > 0 && earned > target {
		earned = target
	}
	if earned > current {
		return earned
	}
	return current
}

// targetGaps lists every measured metric below the policy target.
func targetGaps(totals map[string]adapters.CoverageMetric, target float64) []string {
	if target <= 0 {
		return nil
	}
	var gaps []string
	for _, metric := range adapters.CoverageMetricNames {
		total, ok := totals[metric]
		if !ok || total.Total == 0 {
			continue
		}
		if measured := total.Percent(); measured < target {
			gaps = append(gaps, fmt.Sprintf("%s %s%%", metric, formatPercent(round2(measured))))
		}
	}
	return gaps
}

func committedMetricOrder(committed map[string]float64) []string {
	order := make([]string, 0, len(committed))
	for _, metric := range adapters.CoverageMetricNames {
		if _, ok := committed[metric]; ok {
			order = append(order, metric)
		}
	}
	return order
}

func describeTotals(totals map[string]adapters.CoverageMetric) string {
	parts := make([]string, 0, len(totals))
	for _, metric := range adapters.CoverageMetricNames {
		if total, ok := totals[metric]; ok {
			parts = append(parts, fmt.Sprintf("%s %s%% (%d/%d)", metric, formatPercent(round2(total.Percent())), total.Covered, total.Total))
		}
	}
	if len(parts) == 0 {
		return "no workspace totals"
	}
	return strings.Join(parts, ", ")
}

func describeCommitment(committed map[string]float64) string {
	parts := make([]string, 0, len(committed))
	for _, metric := range committedMetricOrder(committed) {
		parts = append(parts, fmt.Sprintf("%s %s%%", metric, formatPercent(committed[metric])))
	}
	return strings.Join(parts, ", ")
}

func formatPercent(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
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
func readWorkspaceCoverage(ws Workspace) (adapters.CoverageReport, bool) {
	artifacts := make([]adapters.Artifact, 0, len(ws.TestArtifacts))
	for _, artifact := range ws.TestArtifacts {
		artifacts = append(artifacts, adapters.Artifact{Label: artifact.Label, Kind: artifact.Kind, Path: artifact.Reference})
	}
	if len(artifacts) == 0 {
		// Compatibility for hand-built contract fixtures that predate typed
		// artifact declarations. Production plans always carry artifacts.
		artifacts = adapters.DefaultCoverageArtifacts(ws.Language)
	}
	return adapters.ReadCoverage(ws.RootPath, artifacts)
}
