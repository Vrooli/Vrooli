package validation

import (
	"strings"
	"testing"
	"time"

	"unit-health/internal/evidence"
	"unit-health/internal/executor"
	"unit-health/internal/runhistory"
	"unit-health/internal/testquality"
)

func TestRunRecordRetainsNativeIdentitySeedAndRetryCount(t *testing.T) {
	identity := reliabilityTestIdentity(t, "source", "go1")
	seed, retry := "7", 2
	plan := ExecutionPlan{Commands: []PlannedCommand{{WorkspaceID: "ui", Command: "vitest", WorkingDirectory: "/ui"}}}
	response := Response{RunID: "request", Scenario: "demo", CommandResults: []CommandResult{{Command: "vitest", WorkingDirectory: "/ui", Status: "passed", ComparisonIdentity: identity}},
		TestQuality: &testquality.Report{Results: []testquality.Result{{Target: testquality.Target{Workspace: "ui", File: "a.test.ts", TestID: "case-1"}, RuntimeObservation: &testquality.RuntimeObservation{RunID: "native", State: "pass", Seed: &seed, RetryCount: &retry}}}}}
	rec := buildRunRecord(response, plan, time.Now())
	if len(rec.NativeTests) != 1 {
		t.Fatalf("missing native observation: %+v", rec)
	}
	sample := rec.NativeTests[0]
	if sample.NativeRunID != "native" || sample.Identity == nil || sample.Identity.TestID != "case-1" || sample.Identity.Seed == nil || *sample.Identity.Seed != "7" || sample.RetryCount == nil || *sample.RetryCount != 2 || sample.RetryOrdinal != nil {
		t.Fatalf("incorrect native record: %+v", sample)
	}
	if sample.Identity.Comparable(identity) {
		t.Fatal("per-test evidence joined a command cohort")
	}
}

func reliabilityTestIdentity(t *testing.T, source, toolchain string) *runhistory.ComparisonIdentity {
	t.Helper()
	key, err := evidence.NewKey(evidence.KeyInput{
		SourceDigest: source, ConfigDigest: "config", DependencyLockDigest: "lock",
		ToolchainIdentity: toolchain, AdapterID: "go", AdapterVersion: "1",
		CoverageMode: "test", ArtifactSchema: "unit-health.response.v1",
	})
	if err != nil {
		t.Fatal(err)
	}
	return &runhistory.ComparisonIdentity{Evidence: key, Selection: "go test ./..."}
}

// [REQ:UH-ANALYZE-004]
func TestReliabilityExcludesIncompatibleAndInfrastructureHistory(t *testing.T) {
	identity := reliabilityTestIdentity(t, "source", "go1")
	plan := ExecutionPlan{Commands: []PlannedCommand{{WorkspaceID: "api", Command: "go test ./...", WorkingDirectory: "/api"}}}
	for _, tc := range []struct {
		name            string
		identity        *runhistory.ComparisonIdentity
		status, failure string
	}{
		{"legacy", nil, "failed", executor.ClassTestFailure},
		{"source changed", reliabilityTestIdentity(t, "source2", "go1"), "failed", executor.ClassTestFailure},
		{"toolchain changed", reliabilityTestIdentity(t, "source", "go2"), "failed", executor.ClassTestFailure},
		{"timeout", identity, "timeout", executor.ClassTimeoutHang},
		{"system", identity, "error", executor.ClassSystem},
		{"dependency", identity, "failed", executor.ClassMissingDependency},
		{"unclassified failure", identity, "failed", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			results := []CommandResult{{Command: "go test ./...", WorkingDirectory: "/api", Status: "passed", ComparisonIdentity: identity}}
			history := []runhistory.CommandSample{{WorkspaceID: "api", Command: "go test ./...", Status: tc.status, FailureClass: tc.failure, Identity: tc.identity}}
			_, findings := analyzeDiagnostics("demo", nil, plan, results, history, fixedNowStr)
			if _, ok := findingByCode(findings, codeTestFlakeSuspected); ok {
				t.Fatalf("false flake: %+v", findings)
			}
			// The same exclusion applies when the uncertain outcome is current.
			results[0].Status, results[0].FailureClass, results[0].ComparisonIdentity = tc.status, tc.failure, tc.identity
			history[0].Status, history[0].FailureClass, history[0].Identity = "passed", "", identity
			_, findings = analyzeDiagnostics("demo", nil, plan, results, history, fixedNowStr)
			if _, ok := findingByCode(findings, codeTestFlakeSuspected); ok {
				t.Fatalf("false current flake: %+v", findings)
			}
		})
	}
}

func TestReliabilitySeedChangesAndRuntimeCohort(t *testing.T) {
	identity := reliabilityTestIdentity(t, "source", "go1")
	other := reliabilityTestIdentity(t, "source", "go1")
	seed := "42"
	other.Seed = &seed
	if len(comparableHistory(identity, []runhistory.CommandSample{{Identity: other}})) != 0 {
		t.Fatal("changed seed joined cohort")
	}
	plan := ExecutionPlan{Commands: []PlannedCommand{{WorkspaceID: "api", Command: "go test ./...", WorkingDirectory: "/api"}}}
	results := []CommandResult{{Command: "go test ./...", WorkingDirectory: "/api", Status: "passed", DurationMS: 9000, ComparisonIdentity: identity}}
	var history []runhistory.CommandSample
	for i := 0; i < 3; i++ {
		history = append(history, runhistory.CommandSample{WorkspaceID: "api", Command: "go test ./...", Status: "passed", DurationMS: 1000, Identity: other})
	}
	_, findings := analyzeDiagnostics("demo", nil, plan, results, history, fixedNowStr)
	if _, ok := findingByCode(findings, codeTestRuntimeGrowth); ok {
		t.Fatal("runtime compared changed seed")
	}
	for i := range history {
		history[i].Identity = identity
	}
	diagnostics, findings := analyzeDiagnostics("demo", nil, plan, results, history, fixedNowStr)
	if _, ok := findingByCode(findings, codeTestRuntimeGrowth); !ok {
		t.Fatal("compatible runtime growth not detected")
	}
	if len(diagnostics) != 2 || !strings.Contains(diagnostics[1].Evidence, "3 comparable runs") || !strings.Contains(diagnostics[1].Evidence, identity.CohortDigest()) {
		t.Fatalf("missing sample scope: %+v", diagnostics)
	}
}

func TestCommandComparisonSelectionIncludesEnvironmentWithoutSecrets(t *testing.T) {
	key := reliabilityTestIdentity(t, "source", "go1").Evidence
	plan := ExecutionPlan{Commands: []PlannedCommand{{Executable: "go", Args: []string{"test", "./..."}, Env: map[string]string{"TOKEN": "sensitive-value"}}}}
	first := commandComparisonIdentities(key, plan)[0]
	if first == nil || strings.Contains(first.Selection, "sensitive-value") {
		t.Fatal("missing identity or exposed secret")
	}
	if !first.Comparable(commandComparisonIdentities(key, plan)[0]) {
		t.Fatal("identical selection not comparable")
	}
	plan.Commands[0].Env["TOKEN"] = "different-value"
	if first.Comparable(commandComparisonIdentities(key, plan)[0]) {
		t.Fatal("changed environment joined cohort")
	}
	plan.Commands[0].Env["TOKEN"] = "sensitive-value"
	plan.Commands[0].Args = []string{"test", "-run", "TestOne"}
	if first.Comparable(commandComparisonIdentities(key, plan)[0]) {
		t.Fatal("changed selection joined cohort")
	}
}
