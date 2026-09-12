package reactvitest

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"

	"unit-health/internal/adapters"
	"unit-health/internal/testquality"
)

type nativeReport struct {
	RunID       string       `json:"run_id"`
	Schema      string       `json:"schema_version"`
	Version     string       `json:"runner_version"`
	Profile     string       `json:"support_profile"`
	Limitations []string     `json:"limitations"`
	Errors      []string     `json:"unhandled_errors"`
	Tests       []nativeTest `json:"tests"`
}
type nativeTest struct {
	Seed           *string            `json:"seed"`
	RetryCount     *int               `json:"retry_count"`
	RequirementIDs []string           `json:"requirement_ids"`
	ID             string             `json:"test_id"`
	File           string             `json:"file"`
	Project        string             `json:"project"`
	State          string             `json:"state"`
	Enabled        bool               `json:"require_assertions"`
	Status         testquality.Status `json:"assertion_status"`
	Reason         testquality.Reason `json:"reason"`
}

func (Analyzer) QualityArtifact(root, runID string) adapters.Artifact {
	return adapters.Artifact{Kind: "vitest-native", Label: "Native test observations", Path: filepath.Join(root, "coverage", "unit-health", runID+".json")}
}

func (Analyzer) CollectQuality(input adapters.QualityInput) ([]testquality.Result, testquality.Reason) {
	return collectQuality(input, nil)
}

func (Analyzer) CollectQualityEvidence(input adapters.QualityInput) ([]testquality.Result, []testquality.TestLinks, testquality.Reason) {
	var links []testquality.TestLinks
	rows, reason := collectQuality(input, &links)
	if reason != testquality.ReasonNone {
		return nil, nil, reason
	}
	return rows, links, reason
}

func collectQuality(input adapters.QualityInput, links *[]testquality.TestLinks) ([]testquality.Result, testquality.Reason) {
	if !input.Executed {
		return nil, testquality.NotExecuted
	}
	f, err := os.Open(input.Artifact.Path)
	if err != nil {
		return nil, testquality.MissingInput
	}
	defer f.Close()
	const maxBytes = 32 * 1024 * 1024
	data, err := io.ReadAll(io.LimitReader(f, maxBytes+1))
	if err != nil {
		return nil, testquality.MissingInput
	}
	if len(data) > maxBytes {
		return nil, testquality.ResolutionLimit
	}
	var report nativeReport
	if json.Unmarshal(data, &report) != nil {
		return nil, testquality.ParseFailure
	}
	if report.Schema != "vitest-native/v1" {
		return nil, testquality.UnsupportedVersion
	}
	if input.RunID == "" || report.RunID != input.RunID {
		return nil, testquality.StaleEvidence
	}
	if len(report.Tests) == 0 {
		return nil, testquality.MissingInput
	}
	if len(report.Tests) > 10000 {
		return nil, testquality.ResolutionLimit
	}
	results := make([]testquality.Result, 0, len(report.Tests))
	seen := map[testquality.Target]bool{}
	for _, test := range report.Tests {
		file := test.File
		if filepath.IsAbs(file) {
			file, err = filepath.Rel(input.Root, file)
			if err != nil {
				return nil, testquality.ParseFailure
			}
		}
		file = filepath.Clean(file)
		if test.ID == "" || file == "." || file == ".." || strings.HasPrefix(file, ".."+string(filepath.Separator)) {
			return nil, testquality.ParseFailure
		}
		target := testquality.Target{Workspace: input.Workspace, File: filepath.ToSlash(file), TestID: test.Project + ":" + test.ID}
		if seen[target] {
			return nil, testquality.ParseFailure
		}
		seen[target] = true
		if links != nil && test.RequirementIDs != nil {
			state := testquality.ExecutionUnknown
			if report.Version == "2.1.9" && report.Profile == "react-vitest-v2" && len(report.Errors) == 0 {
				switch test.State {
				case "pass":
					state = testquality.ExecutionPassed
				case "fail":
					state = testquality.ExecutionFailed
				case "skip", "todo":
					state = testquality.ExecutionSkipped
				case "run", "only":
					state = testquality.ExecutionNotRun
				}
			}
			*links = append(*links, testquality.TestLinks{Target: target, IDs: append([]string(nil), test.RequirementIDs...), EvidenceKind: testquality.Runtime,
				RunID: report.RunID, ExpectedRunID: input.RunID, Execution: state})
		}
		status, reason := testquality.Unknown, testquality.MissingInput
		switch {
		case report.Version != "2.1.9" || report.Profile != "react-vitest-v2":
			reason = testquality.UnsupportedVersion
		case input.TestKind != "unit" && input.TestKind != "local-integration":
			reason = testquality.UnsupportedTestKind
		case len(report.Errors) > 0:
			reason = testquality.MissingInput
		case !test.Enabled:
			reason = testquality.MissingInput
		case test.State == "skip" || test.State == "todo":
			reason = testquality.Skipped
		case test.Status == testquality.Unknown:
			reason = testquality.NormalizeReason(test.Reason)
		case test.State == "pass" && test.Status == testquality.CheckedClean && test.Reason == testquality.ReasonNone:
			status, reason = testquality.CheckedClean, testquality.ReasonNone
		case test.State == "fail" && test.Status == testquality.Violation && test.Reason == testquality.ReasonNone:
			status, reason = testquality.Violation, testquality.ReasonNone
		}
		var observation *testquality.RuntimeObservation
		if report.Version == "2.1.9" && report.Profile == "react-vitest-v2" {
			if test.RetryCount != nil && *test.RetryCount < 0 {
				return nil, testquality.ParseFailure
			}
			observation = &testquality.RuntimeObservation{RunID: report.RunID, State: test.State, Seed: test.Seed, RetryCount: test.RetryCount}
		}
		results = append(results, testquality.Result{RuntimeObservation: observation, RuleID: "assertion-observation", RuleVersion: "1", Target: target,
			TestKind: input.TestKind, SupportProfile: "react-vitest-v2", Status: status, Reason: reason,
			EvidenceKind: testquality.Runtime, Severity: testquality.Warning, Enforcement: testquality.Advisory,
			EvidenceRefs: []string{input.Artifact.Path}, Limitations: append([]string(nil), report.Limitations...)})
	}
	return results, testquality.ReasonNone
}
