//go:build live

// Live smoke validation against a running BAS engine + a running UI scenario.
// Guarded by the `live` build tag so it never runs in the normal suite. Run with:
//
//	SMOKE_LIVE_SCENARIO=app-monitor SMOKE_LIVE_UI_URL=http://localhost:20000 \
//	  go test ./internal/smoke/ -tags live -run TestLiveSmokeViaBAS -v -count=1
package smoke

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLiveSmokeViaBAS(t *testing.T) {
	scenario := os.Getenv("SMOKE_LIVE_SCENARIO")
	uiURL := os.Getenv("SMOKE_LIVE_UI_URL")
	if scenario == "" || uiURL == "" {
		t.Skip("set SMOKE_LIVE_SCENARIO and SMOKE_LIVE_UI_URL to run the live smoke validation")
	}
	scenarioDir := os.Getenv("SMOKE_LIVE_SCENARIO_DIR")
	if scenarioDir == "" {
		scenarioDir = filepath.Join(os.Getenv("HOME"), "Vrooli", "scenarios", scenario)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	runner, err := NewBASRunner(ctx, WithRunnerLogger(os.Stdout), WithUIURL(uiURL))
	if err != nil {
		t.Fatalf("build BAS runner: %v", err)
	}

	result, err := runner.Run(ctx, scenario, scenarioDir, "live-validation")
	if err != nil {
		t.Fatalf("smoke run error: %v", err)
	}

	t.Logf("status=%s message=%q handshake(signaled=%t timed_out=%t %dms) net=%d page=%d console_err=%d console_warn=%d",
		result.Status, result.Message,
		result.Handshake.Signaled, result.Handshake.TimedOut, result.Handshake.DurationMs,
		result.NetworkFailureCount, result.PageErrorCount, result.ConsoleErrorCount, result.ConsoleWarningCount)
	t.Logf("artifacts: screenshot=%s console=%s network=%s readme=%s",
		result.Artifacts.Screenshot, result.Artifacts.Console, result.Artifacts.Network, result.Artifacts.Readme)

	if result.Status != StatusPassed {
		t.Fatalf("expected a passing smoke for a known-good UI; got %s: %s", result.Status, result.Message)
	}
	if !result.Handshake.Signaled {
		t.Fatalf("expected the iframe-bridge handshake to signal for %s", scenario)
	}
	if result.Artifacts.Screenshot == "" {
		t.Fatalf("expected a screenshot artifact to be captured")
	}
}
