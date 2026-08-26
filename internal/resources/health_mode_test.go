package resources

import (
	"strings"
	"testing"

	resourcecontrol "github.com/vrooli/vrooli/internal/resources/control"
	runtimehealth "github.com/vrooli/vrooli/internal/resources/runtime/health"
)

// Feature: healthy means up AND on the backend it declared
//
//	As an operator and as autoheal
//	I want a resource serving from the wrong device reported as degraded
//	So that "Degraded is a state, never a secret" is enforced rather than
//	merely written down, and nothing restarts a resource that is working.

// Scenario: a failing readiness check means the resource is not serving.
func TestApplyHealthToStatusReportsReadinessFailureAsDown(t *testing.T) {
	// Given a health verdict where the readiness check failed
	health := HealthResult{Healthy: false, Serving: false, Message: "http check failed for http://127.0.0.1:9000/health"}

	// When it becomes a status
	status := applyHealthToStatus(Status{Running: true}, health)

	// Then the resource is neither healthy nor serving
	if status.Healthy == nil || *status.Healthy {
		t.Fatalf("Healthy = %v, want false", status.Healthy)
	}
	if status.Serving == nil || *status.Serving {
		t.Fatalf("Serving = %v, want false", status.Serving)
	}
	// And it is not labelled mode drift, because it is down rather than degraded
	if status.StatusCode == resourcecontrol.StatusCodeModeDrift {
		t.Fatal("StatusCode = mode_drift for a readiness failure, want a down status")
	}
	if status.Health != "unhealthy" {
		t.Fatalf("Health = %q, want unhealthy", status.Health)
	}
}

// Scenario: a resource on the wrong backend is degraded, not down.
func TestApplyHealthToStatusReportsModeDriftAsDegradedAndServing(t *testing.T) {
	// Given a resource that is answering requests from the CPU while it
	// declared CUDA
	health := HealthResult{
		Healthy:      false,
		Serving:      true,
		ModeDrift:    true,
		DeclaredMode: "cuda",
		ObservedMode: "cpu",
		ModeReason:   "nvidia-smi lists no compute process for pid 4242, so it is running on the CPU",
		Message:      "degraded: declared cuda but running on cpu",
	}

	// When it becomes a status
	status := applyHealthToStatus(Status{Running: true}, health)

	// Then it is not healthy
	if status.Healthy == nil || *status.Healthy {
		t.Fatalf("Healthy = %v, want false", status.Healthy)
	}
	// And it is still serving, so a consumer can tell it apart from down
	if status.Serving == nil || !*status.Serving {
		t.Fatalf("Serving = %v, want true; a degraded resource still answers requests", status.Serving)
	}
	// And it carries the mode_drift status code with both modes
	if status.StatusCode != resourcecontrol.StatusCodeModeDrift {
		t.Fatalf("StatusCode = %q, want %q", status.StatusCode, resourcecontrol.StatusCodeModeDrift)
	}
	if status.DeclaredMode != "cuda" || status.ObservedMode != "cpu" || !status.ModeDrift {
		t.Fatalf("mode fields = %q/%q/%v, want cuda/cpu/true", status.DeclaredMode, status.ObservedMode, status.ModeDrift)
	}
	if status.Health != "degraded" {
		t.Fatalf("Health = %q, want degraded", status.Health)
	}
	// And the reason survives, so an operator does not have to re-derive it
	if !strings.Contains(status.ModeReason, "running on the CPU") {
		t.Fatalf("ModeReason = %q, want the host's evidence", status.ModeReason)
	}
}

// Scenario: a resource on its declared backend is healthy and serving.
func TestApplyHealthToStatusReportsAgreementAsHealthy(t *testing.T) {
	// Given a resource on the backend it declared
	health := HealthResult{Healthy: true, Serving: true, DeclaredMode: "cuda", ObservedMode: "cuda", Message: "healthy"}

	// When it becomes a status
	status := applyHealthToStatus(Status{Running: true}, health)

	// Then it is healthy, serving, and free of drift
	if status.Healthy == nil || !*status.Healthy {
		t.Fatalf("Healthy = %v, want true", status.Healthy)
	}
	if status.Serving == nil || !*status.Serving {
		t.Fatalf("Serving = %v, want true", status.Serving)
	}
	if status.ModeDrift || status.StatusCode == resourcecontrol.StatusCodeModeDrift {
		t.Fatalf("status = %+v, want no drift", status)
	}
	if status.Health != "healthy" {
		t.Fatalf("Health = %q, want healthy", status.Health)
	}
}

// Scenario: an idle accelerated resource is healthy even when placement is not
// yet observable.
func TestApplyHealthToStatusReportsPlacementUndeterminedAsHealthy(t *testing.T) {
	health := HealthResult{
		Healthy:               true,
		Serving:               true,
		PlacementUndetermined: true,
		DeclaredMode:          "cuda",
		ModeReason:            "no model is resident, so placement cannot be read yet",
	}

	status := applyHealthToStatus(Status{Running: true}, health)

	if status.Healthy == nil || !*status.Healthy || status.Serving == nil || !*status.Serving {
		t.Fatalf("health/serving = %v/%v, want true/true", status.Healthy, status.Serving)
	}
	if status.StatusCode != resourcecontrol.StatusCodePlacementUndetermined {
		t.Fatalf("StatusCode = %q, want %q", status.StatusCode, resourcecontrol.StatusCodePlacementUndetermined)
	}
	if status.Health != "healthy" || status.ModeDrift {
		t.Fatalf("status = %+v, want healthy without mode drift", status)
	}
	if !strings.Contains(status.Message, "no model is resident") {
		t.Fatalf("Message = %q, want the undetermined reason", status.Message)
	}
}

// Scenario: undetermined placement does not erase an independent readiness
// failure.
func TestApplyHealthToStatusKeepsReadinessFailureWithUndeterminedPlacement(t *testing.T) {
	status := applyHealthToStatus(Status{Running: true}, HealthResult{
		Healthy:               false,
		Serving:               false,
		PlacementUndetermined: true,
		ModeReason:            "no model is resident, so placement cannot be read yet",
	})
	if status.Healthy == nil || *status.Healthy || status.Health != "unhealthy" {
		t.Fatalf("status = %+v, want unhealthy readiness result", status)
	}
	if status.StatusCode == resourcecontrol.StatusCodePlacementUndetermined {
		t.Fatalf("StatusCode = %q, want no healthy placement code on a failed readiness", status.StatusCode)
	}
}

func TestIsPlacementProbeLivenessFailureMatchesOnlyDeclaredGPUProbe(t *testing.T) {
	manifest := ResourceManifest{HealthChecks: []ResourceHealthCheck{
		{Kind: "liveness", Type: "command", Command: []string{"resource-ollama", "health-gpu"}},
	}}
	if !isPlacementProbeLivenessFailure(manifest, "resource-ollama health-gpu") {
		t.Fatal("declared health-gpu liveness was not recognized")
	}
	if isPlacementProbeLivenessFailure(manifest, "resource-ollama health-ready") {
		t.Fatal("readiness command was recognized as a placement probe")
	}
	if isPlacementProbeLivenessFailure(ResourceManifest{}, "resource-ollama health-gpu") {
		t.Fatal("undeclared health-gpu command was recognized")
	}
}

// Feature: the control plane runs both declared check kinds
//
// Before this, a liveness check could be declared and was never executed, so
// the only check in the fleet that detected a CPU fallback was unreachable.

// Scenario: liveness and readiness combine into one verdict.
func TestRunChecksCombinesReadinessAndLiveness(t *testing.T) {
	cases := []struct {
		scenario    string
		checks      []ResourceHealthCheck
		wantHealthy bool
		wantServing bool
		wantFailed  string
	}{
		{
			scenario:    "Given no checks at all, Then nothing is claimed",
			checks:      nil,
			wantHealthy: false,
			wantServing: false,
		},
		{
			scenario: "Given a passing readiness check and a passing liveness check, Then it is healthy and serving",
			checks: []ResourceHealthCheck{
				{Type: "command", Kind: "readiness", Command: []string{"true"}},
				{Type: "command", Kind: "liveness", Command: []string{"true"}},
			},
			wantHealthy: true,
			wantServing: true,
		},
		{
			scenario: "Given a failing readiness check, Then it is neither healthy nor serving",
			checks: []ResourceHealthCheck{
				{Type: "command", Kind: "readiness", Command: []string{"false"}},
				{Type: "command", Kind: "liveness", Command: []string{"true"}},
			},
			wantHealthy: false,
			wantServing: false,
		},
		{
			scenario: "Given a failing liveness check, Then it is serving but not healthy",
			checks: []ResourceHealthCheck{
				{Type: "command", Kind: "readiness", Command: []string{"true"}},
				{Type: "command", Kind: "liveness", Command: []string{"false"}},
			},
			wantHealthy: false,
			wantServing: true,
			wantFailed:  "false",
		},
		{
			scenario: "Given a check with no kind, Then it is treated as readiness",
			checks: []ResourceHealthCheck{
				{Type: "command", Command: []string{"false"}},
			},
			wantHealthy: false,
			wantServing: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.scenario, func(t *testing.T) {
			// When the control plane runs the declared checks
			result, err := runtimehealth.RunChecks(t.Context(), tc.checks, runtimehealth.Config{Root: t.TempDir()})
			if err != nil {
				t.Fatalf("RunChecks() = %v, want nil", err)
			}

			// Then healthy and serving reflect which kind failed
			if result.Healthy != tc.wantHealthy {
				t.Fatalf("Healthy = %v, want %v (message = %q)", result.Healthy, tc.wantHealthy, result.Message)
			}
			if result.Serving != tc.wantServing {
				t.Fatalf("Serving = %v, want %v (message = %q)", result.Serving, tc.wantServing, result.Message)
			}
			// And a failing liveness check names itself, so the operator knows
			// which contract was not met
			if tc.wantFailed != "" && !strings.Contains(result.LivenessFailed, tc.wantFailed) {
				t.Fatalf("LivenessFailed = %q, want it to name %q", result.LivenessFailed, tc.wantFailed)
			}
			if tc.wantFailed == "" && result.LivenessFailed != "" {
				t.Fatalf("LivenessFailed = %q, want empty", result.LivenessFailed)
			}
		})
	}
}
