package phases

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"test-genie/internal/orchestrator/workspace"
	"test-genie/internal/shared"

	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
)

// Coverage thresholds (global defaults — §8 of the maturity-ladder plan).
// A target below the error threshold is an ERROR finding; between the two
// thresholds it is a WARNING; at or above the warn threshold it is OK (no
// finding). These are the same numbers the EM `coverage` dimension scores
// against. They are intentionally hard-coded constants (not env-tunable) so
// the producer's signal is stable across runs; per-profile tuning lives in
// the EM profile, not here.
const (
	coverageErrorThresholdPct = 50.0
	coverageWarnThresholdPct  = 70.0
)

// coverageTarget is one measured coverage unit (a Go workspace profile or a
// Node LCOV report), reduced to a single percentage.
type coverageTarget struct {
	Name    string // human label, e.g. "go:api" or "node:ui"
	Lang    string // "go" | "node"
	Path    string // scenario-relative artifact path
	Percent float64
}

// runCoveragePhase parses the test-coverage artifacts produced by the unit
// phase (Go `*.coverage.out` profiles and Node LCOV reports) and emits a
// `coverage`-source finding for every target below the warn threshold. When no
// coverage artifact exists at all it emits a single informational finding
// ("coverage absent") rather than failing — a scenario with no tests yet is a
// real signal the controller should see, not a crash.
func runCoveragePhase(ctx context.Context, env workspace.Environment, logWriter io.Writer) RunReport {
	if report := CheckContext(ctx); report != nil {
		return *report
	}

	cleanLog := wrapLogSansANSI(logWriter)
	scenarioDir := strings.TrimSpace(env.ScenarioDir)
	shared.LogStep(cleanLog, "parsing coverage artifacts under %s/coverage", scenarioDir)

	targets := collectCoverageTargets(scenarioDir)
	obs := []Observation{NewSectionObservation("📊", "Coverage")}

	if len(targets) == 0 {
		shared.LogWarn(cleanLog, "no coverage artifacts found — coverage is unmeasured for this scenario")
		obs = append(obs, NewWarningObservation("No coverage artifacts found (run unit tests with coverage to measure)"))
		return RunReport{
			Observations: obs,
			Findings: []*architecturev1.ArchitectureFinding{
				newFinding(
					env.ScenarioName,
					architecturev1.FindingSource_FINDING_SOURCE_COVERAGE,
					"coverage_absent", "info",
					"No test-coverage artifacts were produced for this scenario.",
					"Add unit/integration tests so coverage can be measured (Go: `go test -coverprofile`; Node: an LCOV reporter).",
					nil, nil,
				),
			},
		}
	}

	findings := coverageFindings(env.ScenarioName, targets)
	for _, tgt := range targets {
		msg := fmt.Sprintf("%s: %.1f%% (%s)", tgt.Name, tgt.Percent, tgt.Path)
		switch {
		case tgt.Percent < coverageErrorThresholdPct:
			obs = append(obs, NewErrorObservation(msg))
		case tgt.Percent < coverageWarnThresholdPct:
			obs = append(obs, NewWarningObservation(msg))
		default:
			obs = append(obs, NewSuccessObservation(msg))
		}
	}
	if len(findings) == 0 {
		obs = append(obs, NewSuccessObservation(fmt.Sprintf("All %d coverage target(s) at or above %.0f%%", len(targets), coverageWarnThresholdPct)))
	}

	return RunReport{Observations: obs, Findings: findings}
}

// coverageFindings maps under-threshold targets into the shared
// ArchitectureFinding contract (source=COVERAGE). One finding per target so the
// campaign worklist stays per-target rather than per-file.
func coverageFindings(scenario string, targets []coverageTarget) []*architecturev1.ArchitectureFinding {
	out := make([]*architecturev1.ArchitectureFinding, 0, len(targets))
	for _, tgt := range targets {
		if tgt.Percent >= coverageWarnThresholdPct {
			continue
		}
		severity := "warning"
		if tgt.Percent < coverageErrorThresholdPct {
			severity = "error"
		}
		msg := fmt.Sprintf("Coverage for %s is %.1f%% (below the %.0f%% target).", tgt.Name, tgt.Percent, coverageWarnThresholdPct)
		suggestion := fmt.Sprintf("Add tests for %s to raise coverage to at least %.0f%%.", tgt.Name, coverageWarnThresholdPct)
		out = append(out, newFinding(
			scenario,
			architecturev1.FindingSource_FINDING_SOURCE_COVERAGE,
			"low_coverage:"+tgt.Name, severity, msg, suggestion,
			nonEmptyLocations(tgt.Path), nil,
		))
	}
	return out
}

// collectCoverageTargets discovers and parses every coverage artifact under the
// scenario's coverage/ directory. Missing or unparsable artifacts are skipped
// (a target with no measurable statements contributes nothing); the phase never
// fails on a malformed file.
func collectCoverageTargets(scenarioDir string) []coverageTarget {
	if scenarioDir == "" {
		return nil
	}
	coverageRoot := filepath.Join(scenarioDir, "coverage")
	var targets []coverageTarget

	// Go profiles: coverage/go/go-<rel>.coverage.out (one per Go workspace).
	goProfiles, _ := filepath.Glob(filepath.Join(coverageRoot, "go", "*.coverage.out"))
	sort.Strings(goProfiles)
	for _, p := range goProfiles {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		covered, total := parseGoCoverProfile(string(data))
		if total == 0 {
			continue
		}
		targets = append(targets, coverageTarget{
			Name:    "go:" + goProfileLabel(p),
			Lang:    "go",
			Path:    relTo(scenarioDir, p),
			Percent: pct(covered, total),
		})
	}

	// Node LCOV reports: coverage/**/lcov.info (Istanbul/nyc/vitest default).
	for _, p := range findLCOVReports(coverageRoot) {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		hit, found := parseLCOV(string(data))
		if found == 0 {
			continue
		}
		targets = append(targets, coverageTarget{
			Name:    "node:" + nodeReportLabel(coverageRoot, p),
			Lang:    "node",
			Path:    relTo(scenarioDir, p),
			Percent: pct(hit, found),
		})
	}

	sort.Slice(targets, func(i, j int) bool { return targets[i].Name < targets[j].Name })
	return targets
}

// findLCOVReports walks the coverage tree for lcov.info files. The Go coverage
// subtree is skipped (it never contains LCOV) to keep the walk cheap.
func findLCOVReports(coverageRoot string) []string {
	var out []string
	_ = filepath.WalkDir(coverageRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if d.Name() == "go" {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Name() == "lcov.info" {
			out = append(out, path)
		}
		return nil
	})
	sort.Strings(out)
	return out
}

// parseGoCoverProfile sums covered vs. total statements from a Go coverage
// profile (the `go test -coverprofile` format). Each data line is
// `file:start.col,end.col numStatements hitCount`; a statement block counts as
// covered when its hit count is > 0. The leading `mode:` line is ignored.
func parseGoCoverProfile(content string) (covered, total int) {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "mode:") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 3 {
			continue
		}
		numStmt, err1 := strconv.Atoi(fields[1])
		count, err2 := strconv.Atoi(fields[2])
		if err1 != nil || err2 != nil || numStmt <= 0 {
			continue
		}
		total += numStmt
		if count > 0 {
			covered += numStmt
		}
	}
	return covered, total
}

// parseLCOV sums lines-hit (LH) and lines-found (LF) across every record in an
// LCOV report. Line coverage is the canonical headline percentage for Node
// tooling, so we use LH/LF rather than function or branch coverage.
func parseLCOV(content string) (hit, found int) {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "LF:"):
			if n, err := strconv.Atoi(strings.TrimPrefix(line, "LF:")); err == nil {
				found += n
			}
		case strings.HasPrefix(line, "LH:"):
			if n, err := strconv.Atoi(strings.TrimPrefix(line, "LH:")); err == nil {
				hit += n
			}
		}
	}
	return hit, found
}

func pct(part, whole int) float64 {
	if whole <= 0 {
		return 0
	}
	return float64(part) / float64(whole) * 100
}

// goProfileLabel derives a short workspace label from a Go coverage profile
// filename (go-<rel>.coverage.out → <rel>).
func goProfileLabel(path string) string {
	base := filepath.Base(path)
	base = strings.TrimSuffix(base, ".coverage.out")
	base = strings.TrimPrefix(base, "go-")
	if base == "" {
		return "root"
	}
	return base
}

// nodeReportLabel derives a label from an LCOV report's parent directory
// relative to the coverage root (coverage/ui/lcov.info → "ui").
func nodeReportLabel(coverageRoot, path string) string {
	dir := filepath.Dir(path)
	rel, err := filepath.Rel(coverageRoot, dir)
	if err != nil || rel == "." || rel == "" {
		return "root"
	}
	return strings.ReplaceAll(rel, string(os.PathSeparator), "/")
}

func relTo(base, path string) string {
	rel, err := filepath.Rel(base, path)
	if err != nil {
		return path
	}
	return rel
}
