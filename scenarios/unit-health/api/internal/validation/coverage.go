package validation

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// defaultCoverageThreshold is used when the target scenario declares no coverage
// gate. Zero means "measure but do not gate": coverage is reported and a missing
// artifact yields an advisory COVERAGE_ABSENT, but no LOW_COVERAGE is emitted.
const defaultCoverageThreshold = 0.0

// testingConfig is the subset of `.vrooli/testing.json` Unit Health reads for a
// coverage threshold. The schema does not yet standardize a coverage block, so
// both shapes are accepted and the highest declared threshold wins.
type testingConfig struct {
	Coverage struct {
		ThresholdPercent float64 `json:"threshold_percent"`
		Threshold        float64 `json:"threshold"`
	} `json:"coverage"`
	Unit struct {
		CoverageThreshold float64 `json:"coverage_threshold"`
	} `json:"unit"`
}

// coverageThresholdPercent resolves the per-scenario coverage gate from
// `.vrooli/testing.json`, defaulting to defaultCoverageThreshold.
func coverageThresholdPercent(scenarioRoot string) float64 {
	raw := readFileString(filepath.Join(scenarioRoot, ".vrooli", "testing.json"))
	if raw == "" {
		return defaultCoverageThreshold
	}
	var cfg testingConfig
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return defaultCoverageThreshold
	}
	best := defaultCoverageThreshold
	for _, v := range []float64{cfg.Coverage.ThresholdPercent, cfg.Coverage.Threshold, cfg.Unit.CoverageThreshold} {
		if v > best {
			best = v
		}
	}
	return best
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
	threshold := coverageThresholdPercent(scenarioRoot)
	var targets []CoverageTarget
	var findings []Finding

	for _, ws := range workspaces {
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
				Evidence:     "no coverage.out, coverage-summary.json, or lcov.info under the workspace",
				Expected:     "A coverage artifact written by the canonical coverage command.",
				Observed:     "no coverage artifact",
				WhyItMatters: "Without a coverage artifact, hardening depth for this workspace is invisible.",
				Remediation:  "Ensure the coverage command writes its artifact to the workspace (Go coverage.out, Vitest coverage/).",
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
		for _, name := range names {
			fc := fileCov[name]
			agg.covered += fc.covered
			agg.total += fc.total
			targets = append(targets, CoverageTarget{
				ID:              ws.ID + ":" + name,
				Language:        ws.Language,
				SurfaceID:       ws.ID,
				FilePath:        name,
				CoveredLines:    fc.covered,
				TotalLines:      fc.total,
				CoveragePercent: round2(fc.percent()),
				Threshold:       threshold,
				Status:          coverageStatus(fc.percent(), threshold),
			})
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
	switch ws.Language {
	case "go":
		if cov, ok := parseGoCoverProfile(filepath.Join(ws.RootPath, "coverage.out")); ok {
			return cov, true
		}
		return nil, false
	default:
		// TypeScript/Vite (and any JS) write to coverage/ via Istanbul/V8.
		dir := filepath.Join(ws.RootPath, "coverage")
		if cov, ok := parseCoverageSummary(filepath.Join(dir, "coverage-summary.json")); ok {
			return cov, true
		}
		if cov, ok := parseLCOV(filepath.Join(dir, "lcov.info")); ok {
			return cov, true
		}
		return nil, false
	}
}

// parseGoCoverProfile parses a Go cover profile into per-file statement coverage.
func parseGoCoverProfile(path string) (map[string]fileCoverage, bool) {
	f, err := os.Open(path)
	if err != nil {
		return nil, false
	}
	defer f.Close()

	cov := map[string]fileCoverage{}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1024*1024), 4*1024*1024)
	lines := 0
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "mode:") {
			continue
		}
		// Format: <name>:<startLine>.<col>,<endLine>.<col> <numStmts> <count>
		fields := strings.Fields(line)
		if len(fields) != 3 {
			continue
		}
		ci := strings.LastIndex(fields[0], ":")
		if ci <= 0 {
			continue
		}
		name := fields[0][:ci]
		stmts, err1 := strconv.ParseInt(fields[1], 10, 64)
		count, err2 := strconv.ParseInt(fields[2], 10, 64)
		if err1 != nil || err2 != nil {
			continue
		}
		fc := cov[name]
		fc.total += stmts
		if count > 0 {
			fc.covered += stmts
		}
		cov[name] = fc
		lines++
	}
	if lines == 0 {
		return nil, false
	}
	return cov, true
}

// coverageSummaryEntry mirrors Istanbul/Vitest json-summary line metrics.
type coverageSummaryEntry struct {
	Lines struct {
		Total   int64 `json:"total"`
		Covered int64 `json:"covered"`
	} `json:"lines"`
}

// parseCoverageSummary parses a Vitest/Istanbul coverage-summary.json file,
// dropping the synthetic "total" aggregate (Unit Health re-aggregates per file).
func parseCoverageSummary(path string) (map[string]fileCoverage, bool) {
	raw := readFileString(path)
	if raw == "" {
		return nil, false
	}
	var summary map[string]coverageSummaryEntry
	if err := json.Unmarshal([]byte(raw), &summary); err != nil {
		return nil, false
	}
	cov := map[string]fileCoverage{}
	for name, entry := range summary {
		if strings.EqualFold(name, "total") {
			continue
		}
		cov[name] = fileCoverage{covered: entry.Lines.Covered, total: entry.Lines.Total}
	}
	if len(cov) == 0 {
		return nil, false
	}
	return cov, true
}

// parseLCOV parses an lcov.info file into per-file line coverage using LF/LH.
func parseLCOV(path string) (map[string]fileCoverage, bool) {
	f, err := os.Open(path)
	if err != nil {
		return nil, false
	}
	defer f.Close()

	cov := map[string]fileCoverage{}
	var current string
	var fc fileCoverage
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1024*1024), 4*1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		switch {
		case strings.HasPrefix(line, "SF:"):
			current = strings.TrimPrefix(line, "SF:")
			fc = fileCoverage{}
		case strings.HasPrefix(line, "LF:"):
			fc.total = parseInt64(strings.TrimPrefix(line, "LF:"))
		case strings.HasPrefix(line, "LH:"):
			fc.covered = parseInt64(strings.TrimPrefix(line, "LH:"))
		case line == "end_of_record":
			if current != "" {
				cov[current] = fc
			}
			current = ""
		}
	}
	if len(cov) == 0 {
		return nil, false
	}
	return cov, true
}

func parseInt64(s string) int64 {
	v, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	if err != nil {
		return 0
	}
	return v
}
