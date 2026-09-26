package cancellationqualification

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestRecordObservationWritesOnlyValidatedMeasurements(t *testing.T) {
	t.Setenv(ObservationsEnv, "")
	if err := RecordObservation("TestOwner", "timeout", passingObservation()); err != nil {
		t.Fatalf("disabled observation writer: %v", err)
	}

	path := filepath.Join(t.TempDir(), "observations.jsonl")
	t.Setenv(ObservationsEnv, path)
	if err := RecordObservation("TestOwner", "timeout", passingObservation()); err != nil {
		t.Fatalf("record valid owner observation: %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var record ObservationRecord
	if err := json.Unmarshal(raw, &record); err != nil {
		t.Fatalf("decode owner observation: %v", err)
	}
	if record.Case != "timeout" || record.Observation.OwnerTest != "TestOwner" || !record.Observation.Passed {
		t.Fatalf("recorded observation = %+v", record)
	}
	if err := RecordObservation("TestOwner", "timeout", CaseObservation{}); err == nil {
		t.Fatal("invalid observation was recorded")
	}
}

func TestRequiredSourceFilesCoverEveryJ07OwnerTest(t *testing.T) {
	got := make(map[string]bool, len(RequiredSourceFiles))
	for _, source := range RequiredSourceFiles {
		got[source] = true
	}
	for _, source := range []string{
		"api/automation/executor/session_lifecycle_test.go",
		"api/services/workflow/execution_results_test.go",
		"api/services/recovery/service_test.go",
		"api/handlers/executions/service_test.go",
		"api/cmd/cancellation-qualification/main.go",
		"api/cmd/cancellation-qualification/main_test.go",
		"api/cmd/cancellation-qualification/qualification.mjs",
		"api/cmd/cancellation-restart-cohort/qualification.mjs",
		"playwright-driver/tests/integration/typed-action-semantics.test.ts",
		"playwright-driver/tests/unit/idempotency/session-idempotency.test.ts",
		"playwright-driver/tests/unit/routes/session-run.test.ts",
	} {
		if !got[source] {
			t.Errorf("required source digests omit J07 owner test %s", source)
		}
	}
}

func TestAssembleReceiptRequiresExactlyOneValidObservationForEveryCase(t *testing.T) {
	root := filepath.Clean("../../..")
	valid := make([]ObservationRecord, 0, len(requiredCases))
	for _, name := range requiredCases {
		observation := passingObservation()
		observation.OwnerTest = "Owner-" + name
		if name == "cancellation" {
			observation.TerminalStatus = "cancelled"
		}
		valid = append(valid, ObservationRecord{Case: name, Observation: observation})
	}
	if _, err := AssembleReceipt(root, "sha256:fixture-build", valid); err != nil {
		t.Fatalf("assemble complete owner receipt: %v", err)
	}

	if _, err := AssembleReceipt(root, "sha256:fixture-build", valid[:len(valid)-1]); err == nil {
		t.Fatal("incomplete observation set was accepted")
	}
	duplicates := append(append([]ObservationRecord(nil), valid...), valid[0])
	if _, err := AssembleReceipt(root, "sha256:fixture-build", duplicates); err == nil {
		t.Fatal("duplicate case observation was accepted")
	}
	unknown := append(append([]ObservationRecord(nil), valid...), ObservationRecord{Case: "invented", Observation: passingObservation()})
	if _, err := AssembleReceipt(root, "sha256:fixture-build", unknown); err == nil {
		t.Fatal("unknown case observation was accepted")
	}
}

func TestValidateRequiresEveryCurrentCaseAndEnforcesBands(t *testing.T) {
	root := filepath.Clean("../../..")
	receipt, err := NewReceipt(root, "sha256:fixture-build")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range requiredCases {
		receipt.Cases[name] = passingObservation()
	}
	cancelObservation := receipt.Cases["cancellation"]
	cancelObservation.TerminalStatus = "cancelled"
	receipt.Cases["cancellation"] = cancelObservation
	if err := Validate(root, receipt, "sha256:fixture-build"); err != nil {
		t.Fatalf("valid receipt rejected: %v", err)
	}

	for _, tc := range []struct {
		name   string
		mutate func(*Receipt)
	}{
		{name: "missing driver-death case", mutate: func(r *Receipt) { delete(r.Cases, "driverDeath") }},
		{name: "late input stop", mutate: func(r *Receipt) { o := r.Cases["cancellation"]; o.InputStoppedMS = 1001; r.Cases["cancellation"] = o }},
		{name: "late resource cleanup", mutate: func(r *Receipt) { o := r.Cases["timeout"]; o.CleanupMS = 5001; r.Cases["timeout"] = o }},
		{name: "late session recovery", mutate: func(r *Receipt) { o := r.Cases["apiRestart"]; o.RecoveryMS = 10001; r.Cases["apiRestart"] = o }},
		{name: "replayed uncertain effect", mutate: func(r *Receipt) { o := r.Cases["retriedStart"]; o.RetryAdmitted = true; r.Cases["retriedStart"] = o }},
		{name: "lost effect uncertainty", mutate: func(r *Receipt) { o := r.Cases["driverDeath"]; o.UncertainEffect = false; r.Cases["driverDeath"] = o }},
		{name: "stale build", mutate: func(r *Receipt) { r.Runtime.BuildIdentity = "sha256:stale-build" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			candidate := receipt
			candidate.Cases = cloneCases(receipt.Cases)
			candidate.SourceFiles = cloneSources(receipt.SourceFiles)
			tc.mutate(&candidate)
			if err := Validate(root, candidate, "sha256:fixture-build"); err == nil {
				t.Fatal("invalid receipt was accepted")
			}
		})
	}
	for _, source := range RequiredSourceFiles {
		t.Run("stale source "+source, func(t *testing.T) {
			candidate := receipt
			candidate.Cases = cloneCases(receipt.Cases)
			candidate.SourceFiles = cloneSources(receipt.SourceFiles)
			candidate.SourceFiles[source] = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
			if err := Validate(root, candidate, "sha256:fixture-build"); err == nil {
				t.Fatalf("changed owner source %s was accepted", source)
			}
		})
	}
}

func passingObservation() CaseObservation {
	return CaseObservation{
		OwnerTest:                "TestOwner",
		Observed:                 true,
		Passed:                   true,
		ExternalEffects:          1,
		TerminalStatus:           "failed",
		LiveResourcesBeforeClose: 1,
		LiveResourcesAfterClose:  0,
		InputStoppedMS:           100,
		CleanupMS:                300,
		RecoveryMS:               500,
		UncertainEffect:          true,
	}
}

func cloneCases(source map[string]CaseObservation) map[string]CaseObservation {
	copy := make(map[string]CaseObservation, len(source))
	for key, value := range source {
		copy[key] = value
	}
	return copy
}

func cloneSources(source map[string]string) map[string]string {
	copy := make(map[string]string, len(source))
	for key, value := range source {
		copy[key] = value
	}
	return copy
}
