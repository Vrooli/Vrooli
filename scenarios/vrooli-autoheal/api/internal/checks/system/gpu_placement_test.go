package system

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
)

// Feature: a healthy card is not the same thing as a used card
//
//	As an operator watching the GPU check
//	I want it to say when the resources that asked for this card are not on it
//	So that an idle card at 35 degrees stops passing as "everything is fine"
//	while the fleet runs on the CPU.

type stubPlacement struct {
	drifted []DriftedResource
	err     error
}

func (s stubPlacement) DriftedResources(context.Context) ([]DriftedResource, error) {
	return s.drifted, s.err
}

func okResult() checks.Result {
	score := 90
	return checks.Result{
		CheckID: "system-gpu",
		Status:  checks.StatusOK,
		Message: "1 GPU(s) healthy",
		Details: map[string]interface{}{},
		Metrics: &checks.HealthMetrics{Score: &score},
	}
}

// Scenario: a drifted resource turns a healthy card into a warning.
func TestPlacementRaisesAHealthyCardToWarning(t *testing.T) {
	// Given a healthy card and a resource serving from the CPU
	reporter := stubPlacement{drifted: []DriftedResource{{Name: "whisper", Declared: "cuda", Observed: "cpu"}}}

	// When placement is folded into the result
	result := applyPlacement(context.Background(), reporter, okResult())

	// Then the verdict becomes a warning naming the resource
	if result.Status != checks.StatusWarning {
		t.Fatalf("Status = %v, want warning", result.Status)
	}
	for _, want := range []string{"whisper", "declared cuda", "on cpu"} {
		if !strings.Contains(result.Message, want) {
			t.Fatalf("Message = %q, want it to contain %q", result.Message, want)
		}
	}
	// And the machine-readable list is there for a consumer
	names, _ := result.Details["driftedResources"].([]string)
	if len(names) != 1 || names[0] != "whisper" {
		t.Fatalf("driftedResources = %v, want [whisper]", result.Details["driftedResources"])
	}
	// And a sub-check records it beside the device thresholds
	found := false
	for _, sub := range result.Metrics.SubChecks {
		if sub.Name == "accelerator-placement" && !sub.Passed {
			found = true
		}
	}
	if !found {
		t.Fatalf("SubChecks = %+v, want a failing accelerator-placement entry", result.Metrics.SubChecks)
	}
}

// Scenario: placement never masks a more urgent device problem.
func TestPlacementNeverDowngradesACriticalCard(t *testing.T) {
	// Given a card that is critically hot and a drifted resource
	critical := okResult()
	critical.Status = checks.StatusCritical
	critical.Message = "GPU 0 temperature critical: 91°C"
	reporter := stubPlacement{drifted: []DriftedResource{{Name: "whisper", Declared: "cuda", Observed: "cpu"}}}

	// When placement is folded in
	result := applyPlacement(context.Background(), reporter, critical)

	// Then the critical verdict stands
	if result.Status != checks.StatusCritical {
		t.Fatalf("Status = %v, want the critical verdict to stand", result.Status)
	}
	// And the placement finding is appended rather than lost
	if !strings.Contains(result.Message, "temperature critical") || !strings.Contains(result.Message, "whisper") {
		t.Fatalf("Message = %q, want both the temperature and the placement finding", result.Message)
	}
}

// Scenario: nothing drifted records that fact explicitly.
func TestPlacementRecordsACleanFleet(t *testing.T) {
	// Given every accelerator-declaring resource on its declared backend
	result := applyPlacement(context.Background(), stubPlacement{}, okResult())

	// Then the healthy verdict stands
	if result.Status != checks.StatusOK {
		t.Fatalf("Status = %v, want ok", result.Status)
	}
	// And the clean state is recorded rather than left silent
	if count, _ := result.Details["driftedResourceCount"].(int); count != 0 {
		t.Fatalf("driftedResourceCount = %v, want 0", result.Details["driftedResourceCount"])
	}
	found := false
	for _, sub := range result.Metrics.SubChecks {
		if sub.Name == "accelerator-placement" && sub.Passed {
			found = true
		}
	}
	if !found {
		t.Fatalf("SubChecks = %+v, want a passing accelerator-placement entry", result.Metrics.SubChecks)
	}
}

// Scenario: a placement read that fails does not fabricate a verdict.
func TestPlacementFailureIsRecordedNotGuessed(t *testing.T) {
	// Given a placement source that cannot answer
	reporter := stubPlacement{err: errors.New("control plane unreachable")}

	// When placement is folded in
	result := applyPlacement(context.Background(), reporter, okResult())

	// Then the device verdict is untouched and the failure is recorded
	if result.Status != checks.StatusOK {
		t.Fatalf("Status = %v; an unreadable placement must not change the device verdict", result.Status)
	}
	if detail, _ := result.Details["placementError"].(string); !strings.Contains(detail, "unreachable") {
		t.Fatalf("placementError = %v, want the underlying failure", result.Details["placementError"])
	}
	// And no drift is claimed
	if _, claimed := result.Details["driftedResources"]; claimed {
		t.Fatal("a failed placement read must not claim a drift verdict either way")
	}
}

// Scenario: with no reporter the check is exactly what it was before.
func TestPlacementIsInertWithoutAReporter(t *testing.T) {
	before := okResult()
	result := applyPlacement(context.Background(), nil, before)
	if result.Status != before.Status || result.Message != before.Message {
		t.Fatalf("result = %+v, want it unchanged from %+v", result, before)
	}
}

// Scenario: a drifted resource with an unreadable backend still reads clearly.
func TestDriftedResourceRendersAnUnknownBackend(t *testing.T) {
	got := DriftedResource{Name: "kokoro", Declared: "cuda", Observed: ""}.String()
	if !strings.Contains(got, "an unknown backend") {
		t.Fatalf("String() = %q, want it to name the observed backend as unknown", got)
	}
}

func TestAcceleratorDeclaringResourcesIncludesExplicitCPUFallback(t *testing.T) {
	root := t.TempDir()
	resourceDir := filepath.Join(root, "resources", "whisper")
	if err := os.MkdirAll(resourceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := []byte(`{"acceleration":{"backends":["cpu"]}}`)
	if err := os.WriteFile(filepath.Join(resourceDir, "resource.json"), manifest, 0o644); err != nil {
		t.Fatal(err)
	}
	resources := acceleratorDeclaringResources(root)
	if len(resources) != 1 || resources[0] != "whisper" {
		t.Fatalf("resources = %v, want explicit CPU resource included", resources)
	}
}
