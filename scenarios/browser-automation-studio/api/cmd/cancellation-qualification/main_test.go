package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vrooli/browser-automation-studio/internal/cancellationqualification"
)

func TestRunWritesOnlyACompleteValidatedReceiptUnderEvidenceRoot(t *testing.T) {
	scenarioRoot, err := findScenarioRoot(".")
	if err != nil {
		t.Fatal(err)
	}
	evidenceRoot := filepath.Join(scenarioRoot, ".vrooli/runtime/rehabilitation-evidence")
	observationsPath := filepath.Join(t.TempDir(), "observations.jsonl")
	observationsFile, err := os.Create(observationsPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"cancellation", "timeout", "driverDeath", "apiRestart", "retriedStart"} {
		terminal := "failed"
		if name == "cancellation" {
			terminal = "cancelled"
		}
		record := cancellationqualification.ObservationRecord{
			Case: name,
			Observation: cancellationqualification.CaseObservation{
				OwnerTest:                "owner-" + name,
				Observed:                 true,
				Passed:                   true,
				ExternalEffects:          1,
				TerminalStatus:           terminal,
				LiveResourcesBeforeClose: 1,
				LiveResourcesAfterClose:  0,
				InputStoppedMS:           100,
				CleanupMS:                300,
				RecoveryMS:               500,
				UncertainEffect:          true,
				RetryAdmitted:            false,
			},
		}
		if err := json.NewEncoder(observationsFile).Encode(record); err != nil {
			t.Fatal(err)
		}
	}
	if err := observationsFile.Close(); err != nil {
		t.Fatal(err)
	}

	outputPath := filepath.Join(evidenceRoot, fmt.Sprintf("cancellation-recovery-cli-test-%d.json", time.Now().UnixNano()))
	t.Cleanup(func() { _ = os.Remove(outputPath) })
	var output bytes.Buffer
	args := []string{"--build", "sha256:fixture-build", "--observations", observationsPath, "--output", outputPath}
	if err := run(args, &output); err != nil {
		t.Fatalf("write validated receipt: %v", err)
	}
	if output.String() != outputPath+"\n" {
		t.Fatalf("receipt command output = %q", output.String())
	}
	raw, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	var receipt cancellationqualification.Receipt
	if err := json.Unmarshal(raw, &receipt); err != nil {
		t.Fatal(err)
	}
	if err := cancellationqualification.Validate(scenarioRoot, receipt, "sha256:fixture-build"); err != nil {
		t.Fatalf("written receipt failed validation: %v", err)
	}
	if err := run(args, &output); err == nil {
		t.Fatal("receipt command overwrote an existing receipt")
	}

	outsidePath := filepath.Join(t.TempDir(), "cancellation-recovery-outside.json")
	outsideArgs := []string{"--build", "sha256:fixture-build", "--observations", observationsPath, "--output", outsidePath}
	if err := run(outsideArgs, &output); err == nil {
		t.Fatal("receipt command wrote outside the rehabilitation evidence owner")
	}
}
