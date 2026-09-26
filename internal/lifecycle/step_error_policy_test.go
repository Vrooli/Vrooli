package lifecycle

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/process"
	testresource "github.com/vrooli/vrooli/internal/resources/resourcestest"
	"github.com/vrooli/vrooli/internal/scenario"
)

// readLifecycleLog returns the scenario's lifecycle log, which is where a
// tolerated failure has to stay visible.
func readLifecycleLog(t *testing.T, home, slug string) string {
	t.Helper()
	path, err := process.ScenarioLifecycleLogPath(home, slug)
	if err != nil {
		t.Fatalf("ScenarioLifecycleLogPath: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read lifecycle log: %v", err)
	}
	return string(data)
}

// A step that declares on_error=continue must not fail its phase. The manifest
// schema has always accepted the field; before this, 14 steps across 13
// scenarios declared it and every one still aborted its phase on failure —
// which is how a missing optional Claude Code resource blocked web-console
// from starting at all.
func TestRunPhaseToleratesStepDeclaringOnErrorContinue(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	testresource.WritePortRegistry(t, root, nil)
	writeLifecycleFixtureManifest(t, root, scenario.ServiceManifest{
		Service: scenario.ServiceMetadata{Name: "alpha"},
		Lifecycle: scenario.Lifecycle{
			Setup: scenario.Phase{
				Steps: []scenario.PhaseStep{
					{Name: "optional", Exec: []string{"bash", "-c", "echo 'optional dependency absent'; exit 1"}, OnError: "continue"},
					{Name: "required", Exec: []string{"bash", "-c", "echo 'ran anyway'"}},
				},
			},
		},
	})

	runner, err := NewRunner(root, home, io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	result, err := runner.RunPhaseDetailed("alpha", "setup", PhaseOptions{})
	if err != nil {
		t.Fatalf("phase failed despite on_error=continue: %v", err)
	}
	if result.Status != PhaseExecutionCompleted {
		t.Fatalf("phase status = %q, want completed", result.Status)
	}
	if result.ExecutedSteps != 2 {
		t.Fatalf("executed steps = %d, want 2 (the tolerated step still ran)", result.ExecutedSteps)
	}
	// The phase succeeded; something in it did not. A caller reporting "all
	// steps ran" without this count would be quietly wrong.
	if result.FailedTolerated != 1 {
		t.Fatalf("failed-tolerated count = %d, want 1", result.FailedTolerated)
	}

	log := readLifecycleLog(t, home, "alpha")
	if !strings.Contains(log, "ran anyway") {
		t.Fatalf("later step did not run:\n%s", log)
	}
	// The tolerated failure must still be visible: tolerated is not hidden.
	if !strings.Contains(log, "optional dependency absent") {
		t.Fatalf("tolerated failure was not recorded:\n%s", log)
	}
}

// The default is unchanged: an undeclared policy still stops the phase.
func TestRunPhaseStopsWhenStepDeclaresNoErrorPolicy(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	testresource.WritePortRegistry(t, root, nil)
	writeLifecycleFixtureManifest(t, root, scenario.ServiceManifest{
		Service: scenario.ServiceMetadata{Name: "alpha"},
		Lifecycle: scenario.Lifecycle{
			Setup: scenario.Phase{
				Steps: []scenario.PhaseStep{
					{Name: "explode", Exec: []string{"bash", "-c", "exit 7"}},
					{Name: "unreached", Exec: []string{"bash", "-c", "echo 'should not run'"}},
				},
			},
		},
	})

	runner, err := NewRunner(root, home, io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	if _, err := runner.RunPhaseDetailed("alpha", "setup", PhaseOptions{}); err == nil {
		t.Fatal("expected an undeclared step failure to stop the phase")
	}
	if log := readLifecycleLog(t, home, "alpha"); strings.Contains(log, "should not run") {
		t.Fatalf("phase continued past a fatal step:\n%s", log)
	}
}

// on_error=retry re-runs the step until it succeeds or exhausts its attempts.
func TestRunPhaseRetriesStepDeclaringOnErrorRetry(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	markers := t.TempDir()
	first := filepath.Join(markers, "attempt1")
	second := filepath.Join(markers, "attempt2")
	testresource.WritePortRegistry(t, root, nil)
	// Succeeds only on the third attempt. Written without shell substitution:
	// argv goes through placeholder expansion, which claims "$(".
	script := "if [ -f " + second + " ]; then exit 0; elif [ -f " + first +
		" ]; then touch " + second + "; exit 1; else touch " + first + "; exit 1; fi"
	writeLifecycleFixtureManifest(t, root, scenario.ServiceManifest{
		Service: scenario.ServiceMetadata{Name: "alpha"},
		Lifecycle: scenario.Lifecycle{
			Setup: scenario.Phase{
				Steps: []scenario.PhaseStep{{
					Name:    "flaky",
					Exec:    []string{"bash", "-c", script},
					OnError: "retry",
					Retry:   &scenario.RetryPolicy{MaxAttempts: 4, Delay: 1},
				}},
			},
		},
	})

	runner, err := NewRunner(root, home, io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	result, err := runner.RunPhaseDetailed("alpha", "setup", PhaseOptions{})
	if err != nil {
		t.Fatalf("retrying step never succeeded: %v", err)
	}
	for _, marker := range []string{first, second} {
		if _, statErr := os.Stat(marker); statErr != nil {
			t.Fatalf("expected attempt marker %s: %v", marker, statErr)
		}
	}
	// A step that eventually succeeded is not a tolerated failure.
	if result.FailedTolerated != 0 {
		t.Fatalf("failed-tolerated count = %d, want 0", result.FailedTolerated)
	}
}

// Exhausting the attempts is still a phase failure: retry is not tolerance.
func TestRunPhaseFailsWhenRetriesAreExhausted(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	testresource.WritePortRegistry(t, root, nil)
	writeLifecycleFixtureManifest(t, root, scenario.ServiceManifest{
		Service: scenario.ServiceMetadata{Name: "alpha"},
		Lifecycle: scenario.Lifecycle{
			Setup: scenario.Phase{
				Steps: []scenario.PhaseStep{{
					Name:    "always-fails",
					Exec:    []string{"bash", "-c", "exit 3"},
					OnError: "retry",
					Retry:   &scenario.RetryPolicy{MaxAttempts: 2, Delay: 1},
				}},
			},
		},
	})

	runner, err := NewRunner(root, home, io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	if _, err := runner.RunPhaseDetailed("alpha", "setup", PhaseOptions{}); err == nil {
		t.Fatal("expected exhausted retries to fail the phase")
	}
}

func TestStepErrorPolicyDefaults(t *testing.T) {
	// An unrecognised policy is "stop": the safe default, and the historical
	// behaviour for every manifest written before the field was implemented.
	for _, value := range []string{"", "stop", "STOP", "nonsense"} {
		if got := stepOnError(scenario.PhaseStep{OnError: value}); got != onErrorStop {
			t.Fatalf("stepOnError(%q) = %q, want stop", value, got)
		}
	}
	if got := stepOnError(scenario.PhaseStep{OnError: "continue"}); got != onErrorContinue {
		t.Fatalf("stepOnError(continue) = %q", got)
	}
	if got := stepOnError(scenario.PhaseStep{OnError: "retry"}); got != onErrorRetry {
		t.Fatalf("stepOnError(retry) = %q", got)
	}

	if got := retryAttempts(scenario.PhaseStep{OnError: "retry"}); got != defaultRetryAttempts {
		t.Fatalf("retryAttempts with no policy = %d, want %d", got, defaultRetryAttempts)
	}
	if got := retryAttempts(scenario.PhaseStep{Retry: &scenario.RetryPolicy{MaxAttempts: 9}}); got != 9 {
		t.Fatalf("retryAttempts = %d, want 9", got)
	}

	linear := scenario.PhaseStep{Retry: &scenario.RetryPolicy{Delay: 100, Backoff: "linear"}}
	if got := retryDelay(linear, 4); got != 100*time.Millisecond {
		t.Fatalf("linear backoff delay = %s, want 100ms", got)
	}
	exponential := scenario.PhaseStep{Retry: &scenario.RetryPolicy{Delay: 100, Backoff: "exponential"}}
	if got := retryDelay(exponential, 4); got != 400*time.Millisecond {
		t.Fatalf("exponential backoff delay at attempt 4 = %s, want 400ms", got)
	}
	// A mis-declared backoff must not stall a phase indefinitely.
	if got := retryDelay(scenario.PhaseStep{Retry: &scenario.RetryPolicy{Delay: 1000, Backoff: "exponential"}}, 40); got != maxRetryDelay {
		t.Fatalf("capped delay = %s, want %s", got, maxRetryDelay)
	}
}
