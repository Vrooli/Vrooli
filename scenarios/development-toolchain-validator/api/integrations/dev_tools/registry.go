package dev_tools

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// toolSpec is the single extension point for the tool baseline. Adding a
// future tool = one registry entry (argv builder + expectation
// evaluator) + one data/tool-expectations/<tool>.json file. No schema or
// core change is required.
type toolSpec struct {
	name string

	// commands builds the ordered argv sequence (each entry is the args
	// after the binary name) to run for this tool against the golden.
	// Only the LAST command's output is evaluated; earlier commands are
	// preparatory (e.g. a recalculation step) and must each exit zero or
	// the run is classified as a run failure.
	commands func(slug, absPath string, exp Expectation) [][]string

	// evaluate inspects the final command's output and decides whether the
	// tool's success expectation held, returning a human-readable detail.
	evaluate func(final CommandResult, exp Expectation) (met bool, detail string)
}

// defaultRegistry is the closed, typed set of tools the baseline can run.
// Scenario-auditor is intentionally excluded: it is covered inside
// test-genie's standards/architecture phases. flow-verifier and
// architecture-cartographer are likewise covered by test-genie and are
// not baseline tools.
func defaultRegistry() map[string]toolSpec {
	return map[string]toolSpec{
		"test-genie": {
			name: "test-genie",
			commands: func(slug, absPath string, exp Expectation) [][]string {
				preset := strings.TrimSpace(exp.Preset)
				if preset == "" {
					preset = "comprehensive"
				}
				return [][]string{
					{"execute", slug, "--scenario-path", absPath, "--preset", preset, "--json"},
				}
			},
			evaluate: evaluateTestGenie,
		},
		"scenario-completeness-scoring": {
			name: "scenario-completeness-scoring",
			commands: func(slug, absPath string, exp Expectation) [][]string {
				// Force a fresh recalculation against the golden's current
				// state, then read back the numeric score. --auto-start
				// brings the completeness-scoring service up if it is down.
				return [][]string{
					{"--auto-start", "score", "calculate", slug},
					{"--auto-start", "score", "get", slug, "--json"},
				}
			},
			evaluate: evaluateCompleteness,
		},
	}
}

// evaluateTestGenie parses test-genie's execute --json report. The
// expectation holds only when the suite succeeded with every phase
// passing.
func evaluateTestGenie(final CommandResult, _ Expectation) (bool, string) {
	var report struct {
		Success      bool `json:"success"`
		PhaseSummary struct {
			Total  int `json:"total"`
			Passed int `json:"passed"`
			Failed int `json:"failed"`
		} `json:"phaseSummary"`
		Phases []struct {
			Name   string `json:"name"`
			Status string `json:"status"`
		} `json:"phases"`
		Error string `json:"error"`
	}
	raw := extractJSONObject(final.Stdout)
	if raw == nil {
		return false, fmt.Sprintf("could not locate test-genie --json output (exit %d)", final.ExitCode)
	}
	if err := json.Unmarshal(raw, &report); err != nil {
		return false, fmt.Sprintf("invalid test-genie --json output: %v", err)
	}

	var failed []string
	for _, p := range report.Phases {
		if !strings.EqualFold(strings.TrimSpace(p.Status), "passed") {
			failed = append(failed, fmt.Sprintf("%s(%s)", p.Name, p.Status))
		}
	}
	if report.Success && len(failed) == 0 {
		return true, fmt.Sprintf("all %d phase(s) passed", report.PhaseSummary.Total)
	}
	detail := fmt.Sprintf("%d/%d phase(s) failed", len(failed), report.PhaseSummary.Total)
	if len(failed) > 0 {
		detail += ": " + strings.Join(failed, ", ")
	}
	if strings.TrimSpace(report.Error) != "" {
		detail += " — " + report.Error
	}
	return false, detail
}

// evaluateCompleteness parses scenario-completeness-scoring's score get
// --json output and checks the numeric score against the floor.
func evaluateCompleteness(final CommandResult, exp Expectation) (bool, string) {
	floor := exp.ScoreFloor
	if floor <= 0 {
		floor = defaultScoreFloor
	}
	var resp struct {
		Scenario string  `json:"scenario"`
		Score    float64 `json:"score"`
	}
	raw := extractJSONObject(final.Stdout)
	if raw == nil {
		return false, fmt.Sprintf("could not locate completeness score output (exit %d)", final.ExitCode)
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return false, fmt.Sprintf("invalid completeness score output: %v", err)
	}
	if resp.Score >= floor {
		return true, fmt.Sprintf("score %.1f >= floor %.1f", resp.Score, floor)
	}
	return false, fmt.Sprintf("score %.1f < floor %.1f", resp.Score, floor)
}

// extractJSONObject returns the outermost JSON object found in b, or nil
// when none is present. Tools print their JSON report to stdout, but a
// stray log line before/after must not break parsing — so we bracket
// from the first '{' to the last '}'.
func extractJSONObject(b []byte) []byte {
	start := bytes.IndexByte(b, '{')
	end := bytes.LastIndexByte(b, '}')
	if start < 0 || end < 0 || end < start {
		return nil
	}
	return b[start : end+1]
}
