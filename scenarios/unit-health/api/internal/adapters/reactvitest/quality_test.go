package reactvitest

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"unit-health/internal/adapters"
	"unit-health/internal/testquality"
)

func qualityFixture(t *testing.T) (adapters.QualityInput, nativeReport) {
	t.Helper()
	root := t.TempDir()
	in := adapters.QualityInput{Root: root, Workspace: "ui", RunID: "current-command", TestKind: "unit", Executed: true, Artifact: adapters.Artifact{Path: filepath.Join(root, "native.json")}}
	r := nativeReport{RunID: in.RunID, Schema: "vitest-native/v1", Version: "2.1.9", Profile: "react-vitest-v2", Tests: []nativeTest{{ID: "native-id", File: filepath.Join(root, "a.test.ts"), State: "pass", Enabled: true, Status: testquality.CheckedClean, Reason: testquality.ReasonNone}}}
	return in, r
}
func writeNative(t *testing.T, in adapters.QualityInput, r nativeReport) {
	t.Helper()
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(in.Artifact.Path, data, 0600); err != nil {
		t.Fatal(err)
	}
}

func TestNativeRetryAndSeedRemainFinalMetadataNotAttemptHistory(t *testing.T) {
	in, report := qualityFixture(t)
	seed, count := "0", 1
	report.Tests[0].Seed, report.Tests[0].RetryCount = &seed, &count
	writeNative(t, in, report)
	rows, reason := (Analyzer{}).CollectQuality(in)
	if reason != testquality.ReasonNone || len(rows) != 1 {
		t.Fatalf("collect: %+v %s", rows, reason)
	}
	native := rows[0].RuntimeObservation
	if native == nil || native.State != "pass" || native.RunID != in.RunID || native.Seed == nil || *native.Seed != "0" || native.RetryCount == nil || *native.RetryCount != 1 || native.RetryOrdinal != nil {
		t.Fatalf("lost final metadata: %+v", native)
	}
	report.Tests[0].RetryCount = nil
	writeNative(t, in, report)
	rows, _ = (Analyzer{}).CollectQuality(in)
	if rows[0].RuntimeObservation.RetryCount != nil {
		t.Fatal("absent retry count became zero")
	}
	count = -1
	report.Tests[0].RetryCount = &count
	writeNative(t, in, report)
	if _, reason := (Analyzer{}).CollectQuality(in); reason != testquality.ParseFailure {
		t.Fatalf("negative retry count accepted: %s", reason)
	}
}

func TestNativeRequirementEvidenceKeepsLinksSeparateFromAssertionActivity(t *testing.T) {
	in, report := qualityFixture(t)
	report.Tests[0].RequirementIDs = []string{"UH-CORE-001", "UH-CORE-010"}
	report.Tests[0].Enabled = false // Execution is still observed without requireAssertions.
	skip := report.Tests[0]
	skip.ID = "parameter-2"
	skip.State = "skip"
	report.Tests = append(report.Tests, skip)
	writeNative(t, in, report)
	rows, links, reason := (Analyzer{}).CollectQualityEvidence(in)
	if reason != testquality.ReasonNone || len(rows) != 2 || len(links) != 2 {
		t.Fatalf("evidence: %+v %+v %s", rows, links, reason)
	}
	if rows[0].Status != testquality.Unknown || links[0].Execution != testquality.ExecutionPassed || links[1].Execution != testquality.ExecutionSkipped || links[1].Target.TestID != ":parameter-2" {
		t.Fatalf("assertion/link conflation: %+v %+v", rows, links)
	}
	if len(links[0].IDs) != 2 || links[0].ExpectedRunID != in.RunID {
		t.Fatalf("link provenance: %+v", links)
	}
	report.RunID = "stale-command"
	writeNative(t, in, report)
	_, links, reason = (Analyzer{}).CollectQualityEvidence(in)
	if reason != testquality.StaleEvidence || len(links) != 0 {
		t.Fatalf("stale handoff retained links: %+v %s", links, reason)
	}
}

func TestNativeRequirementEvidenceRejectsPartialArtifact(t *testing.T) {
	in, report := qualityFixture(t)
	report.Tests[0].RequirementIDs = []string{"UH-CORE-001"}
	report.Tests = append(report.Tests, report.Tests[0])
	writeNative(t, in, report)
	rows, links, reason := (Analyzer{}).CollectQualityEvidence(in)
	if reason != testquality.ParseFailure || len(rows) != 0 || len(links) != 0 {
		t.Fatalf("duplicate artifact leaked partial evidence: %+v %+v %s", rows, links, reason)
	}
}
func TestNativeQualityTruthfulStates(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*nativeReport)
		status testquality.Status
		reason testquality.Reason
	}{
		{"supported pass", func(*nativeReport) {}, testquality.CheckedClean, testquality.ReasonNone},
		{"unsupported version", func(r *nativeReport) { r.Version = "3.2.4" }, testquality.Unknown, testquality.UnsupportedVersion},
		{"disabled", func(r *nativeReport) { r.Tests[0].Enabled = false }, testquality.Unknown, testquality.MissingInput},
		{"skip cannot be clean", func(r *nativeReport) { r.Tests[0].State = "skip" }, testquality.Unknown, testquality.Skipped},
		{"unknown future status", func(r *nativeReport) { r.Tests[0].Status = "future" }, testquality.Unknown, testquality.MissingInput},
		{"failed setup", func(r *nativeReport) {
			r.Tests[0].State = "fail"
			r.Tests[0].Status = testquality.Unknown
			r.Tests[0].Reason = testquality.MissingInput
		}, testquality.Unknown, testquality.MissingInput},
		{"assertion absence", func(r *nativeReport) { r.Tests[0].State = "fail"; r.Tests[0].Status = testquality.Violation }, testquality.Violation, testquality.ReasonNone},
		{"unhandled error", func(r *nativeReport) { r.Errors = []string{"unhandled rejection"} }, testquality.Unknown, testquality.MissingInput},
	} {
		t.Run(tc.name, func(t *testing.T) {
			in, report := qualityFixture(t)
			tc.change(&report)
			writeNative(t, in, report)
			rows, reason := (Analyzer{}).CollectQuality(in)
			if reason != testquality.ReasonNone || len(rows) != 1 {
				t.Fatalf("rows=%v reason=%s", rows, reason)
			}
			if rows[0].Status != tc.status || rows[0].Reason != tc.reason {
				t.Fatalf("got %+v", rows[0])
			}
			if rows[0].Enforcement != testquality.Advisory || rows[0].Target.File != "a.test.ts" {
				t.Fatalf("scope/enforcement: %+v", rows[0])
			}
		})
	}
}
func TestNativeQualityRejectsStaleMissingMalformedAndDuplicateEvidence(t *testing.T) {
	in, report := qualityFixture(t)
	if _, reason := (Analyzer{}).CollectQuality(in); reason != testquality.MissingInput {
		t.Fatal(reason)
	}
	report.RunID = "old-command"
	writeNative(t, in, report)
	if _, reason := (Analyzer{}).CollectQuality(in); reason != testquality.StaleEvidence {
		t.Fatal(reason)
	}
	report.RunID = in.RunID
	report.Tests = append(report.Tests, report.Tests[0])
	writeNative(t, in, report)
	if _, reason := (Analyzer{}).CollectQuality(in); reason != testquality.ParseFailure {
		t.Fatal(reason)
	}
	if err := os.WriteFile(in.Artifact.Path, []byte("{"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, reason := (Analyzer{}).CollectQuality(in); reason != testquality.ParseFailure {
		t.Fatal(reason)
	}
	in.Executed = false
	if _, reason := (Analyzer{}).CollectQuality(in); reason != testquality.NotExecuted {
		t.Fatal(reason)
	}
}

// This integration check consumes a freshly generated retained native fixture;
// it is opt-in because installed runner fixtures are governed outside Go tests.
func TestNativeQualityReporterConformance(t *testing.T) {
	path := os.Getenv("UNIT_HEALTH_NATIVE_FIXTURE")
	if path == "" {
		t.Skip("provide fresh run-native.mjs output via UNIT_HEALTH_NATIVE_FIXTURE")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Handoff json.RawMessage `json:"nativeHandoff"`
	}
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	var report nativeReport
	if err := json.Unmarshal(fixture.Handoff, &report); err != nil {
		t.Fatal(err)
	}
	if len(report.Tests) != 18 {
		t.Fatalf("native test count: %d", len(report.Tests))
	}
	in := adapters.QualityInput{Workspace: "ui", Root: filepath.Dir(report.Tests[0].File), RunID: report.RunID, TestKind: "unit", Executed: true, Artifact: adapters.Artifact{Path: filepath.Join(t.TempDir(), "handoff.json")}}
	if err := os.WriteFile(in.Artifact.Path, fixture.Handoff, 0600); err != nil {
		t.Fatal(err)
	}
	rows, reason := (Analyzer{}).CollectQuality(in)
	if reason != testquality.ReasonNone || len(rows) != 18 {
		t.Fatalf("rows=%d reason=%s", len(rows), reason)
	}
	var names struct {
		Tests []struct {
			ID      string `json:"test_id"`
			Name    string `json:"name"`
			Project string `json:"project"`
		} `json:"tests"`
	}
	if err := json.Unmarshal(fixture.Handoff, &names); err != nil {
		t.Fatal(err)
	}
	expected := map[string]testquality.Status{
		"C021 direct matcher": testquality.CheckedClean, "C022 delegated axe assertion": testquality.CheckedClean,
		"C023 empty": testquality.Violation, "C024 unreachable": testquality.Violation,
		"C025 bare expect": testquality.CheckedClean, "C026 tautology": testquality.CheckedClean,
		"empty body": testquality.CheckedClean, "C028 Node assertion": testquality.Violation,
		"C029 parameter 1": testquality.CheckedClean, "C029 parameter 2": testquality.Violation,
		"C030 local expect first": testquality.CheckedClean, "C030 local expect second": testquality.CheckedClean,
		"C031 failing custom matcher": testquality.Unknown, "body never executes": testquality.Unknown,
		"C033 retry": testquality.CheckedClean, "C034 skipped": testquality.Unknown,
		"C037 extended": testquality.CheckedClean, "C039 returned async matcher": testquality.CheckedClean,
	}
	byID := map[string]testquality.Result{}
	for _, row := range rows {
		byID[row.Target.TestID] = row
	}
	for _, test := range names.Tests {
		want, ok := expected[test.Name]
		if !ok {
			t.Fatalf("unexpected native case %q", test.Name)
		}
		row := byID[test.Project+":"+test.ID]
		if row.Status != want {
			t.Errorf("%s: got %s, want %s", test.Name, row.Status, want)
		}
		if len(row.Limitations) == 0 {
			t.Errorf("%s missing native limitations", test.Name)
		}
		delete(expected, test.Name)
	}
	if len(expected) != 0 {
		t.Fatalf("missing native cases: %v", expected)
	}
}
