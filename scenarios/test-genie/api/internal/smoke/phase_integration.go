package smoke

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"test-genie/internal/browsercapture"
	"test-genie/internal/captureprofile"
	"test-genie/internal/playbooks/execution"

	"github.com/vrooli/api-core/discovery"
)

// BASScenarioName is the scenario slug for Browser Automation Studio, the engine
// that drives the smoke capture.
const BASScenarioName = "browser-automation-studio"

// PhaseResult represents the result of running UI smoke as part of a phase.
type PhaseResult struct {
	// Success indicates whether the smoke test passed.
	Success bool
	// Message is a human-readable summary of the result.
	Message string
	// Result is the detailed smoke test result.
	Result *Result
	// Skipped indicates the test was skipped (not a failure).
	Skipped bool
	// Blocked indicates the test was blocked by preconditions.
	Blocked bool
}

// RunForPhase executes the UI smoke test and returns a result suitable for phase
// integration. It resolves the BAS workflow endpoint via discovery and drives
// the capture through the shared BAS workflow client. captureProfile is the
// capture-depth dial: "" keeps smoke single-page (unchanged cost); "baseline"
// adds the all-pages visual capture + video.
func RunForPhase(ctx context.Context, scenarioName, scenarioDir, uiURL, runID, captureProfile string, logWriter io.Writer) (*PhaseResult, error) {
	baseURL, err := ResolveBASBaseURL(ctx)
	if err != nil {
		return nil, fmt.Errorf("resolve browser-automation-studio endpoint: %w", err)
	}

	workflowClient := execution.NewClientWithConfig(baseURL, execution.DefaultClientConfig())
	capturer := browsercapture.New(browsercapture.NewLiveClientFrom(workflowClient))

	opts := []RunnerOption{WithRunnerLogger(logWriter)}
	if strings.TrimSpace(uiURL) != "" {
		opts = append(opts, WithUIURL(uiURL))
	}

	profile, ok := captureprofile.Resolve(captureProfile)
	if !ok {
		fmt.Fprintf(logWriter, "unknown capture profile %q; using default depth\n", captureProfile)
	}
	if profile.AllPages {
		mc := browsercapture.NewMultiCapturer(browsercapture.NewLiveCaptureClient(workflowClient))
		opts = append(opts, WithAllPagesCapture(mc, profile.Video))
	}

	runner := NewRunner(capturer, opts...)
	result, err := runner.Run(ctx, scenarioName, scenarioDir, runID)
	if err != nil {
		return nil, fmt.Errorf("ui smoke execution failed: %w", err)
	}

	return resultToPhaseResult(result), nil
}

// NewBASRunner resolves the BAS workflow endpoint via discovery and returns a
// smoke Runner driving it. Callers add WithUIURL/WithRunnerTimeout/WithAutoStart
// as needed. This is the single live construction path shared by the phase and
// the scenario-service UI-smoke API.
func NewBASRunner(ctx context.Context, opts ...RunnerOption) (*Runner, error) {
	baseURL, err := ResolveBASBaseURL(ctx)
	if err != nil {
		return nil, fmt.Errorf("resolve browser-automation-studio endpoint: %w", err)
	}
	capturer := browsercapture.New(browsercapture.NewLiveClient(baseURL, execution.DefaultClientConfig()))
	return NewRunner(capturer, opts...), nil
}

// ResolveBASBaseURL resolves the BAS API base URL (".../api/v1") via scenario
// discovery. The smoke capture and the runnability resource probe share this so
// there is exactly one BAS-endpoint resolution path.
func ResolveBASBaseURL(ctx context.Context) (string, error) {
	url, err := discovery.ResolveScenarioURL(ctx, BASScenarioName, "API_PORT")
	if err != nil {
		return "", err
	}
	url = strings.TrimRight(strings.TrimSpace(url), "/")
	if !strings.HasSuffix(url, "/api/v1") {
		url += "/api/v1"
	}
	return url, nil
}

// ProbeBAS reports whether the BAS workflow engine is reachable and healthy. It
// resolves the endpoint and issues a bounded health check, reusing the shared
// BAS client (no parallel probe). Used by the runnability gate to skip/degrade
// smoke when BAS is down instead of failing it hard.
func ProbeBAS(ctx context.Context) bool {
	baseURL, err := ResolveBASBaseURL(ctx)
	if err != nil {
		return false
	}
	probeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	client := execution.NewClient(baseURL)
	return client.Health(probeCtx) == nil
}

// resultToPhaseResult converts a Result to a PhaseResult.
func resultToPhaseResult(r *Result) *PhaseResult {
	pr := &PhaseResult{
		Result:  r,
		Message: r.Message,
	}

	switch r.Status {
	case StatusPassed:
		pr.Success = true
		if r.DurationMs > 0 {
			pr.Message = fmt.Sprintf("ui smoke passed (%dms)", r.DurationMs)
		} else {
			pr.Message = "ui smoke passed"
		}
	case StatusSkipped:
		pr.Success = true // skipped is not a failure
		pr.Skipped = true
	case StatusBlocked:
		pr.Success = false
		pr.Blocked = true
	case StatusFailed:
		pr.Success = false
	}

	return pr
}

// FormatObservation formats a PhaseResult as an observation message.
func (pr *PhaseResult) FormatObservation() string {
	if pr.Skipped {
		return fmt.Sprintf("UI smoke skipped: %s", pr.Message)
	}
	if pr.Blocked {
		return fmt.Sprintf("UI smoke blocked: %s", pr.Message)
	}
	if pr.Success {
		return pr.Message
	}
	return fmt.Sprintf("UI smoke failed: %s", pr.Message)
}

// ToError returns an error if the result indicates failure, otherwise nil.
func (pr *PhaseResult) ToError() error {
	if pr.Success || pr.Skipped {
		return nil
	}
	return fmt.Errorf("ui smoke %s: %s", pr.Result.Status, pr.Message)
}

// GetBundleStatus returns bundle status message if the bundle is stale.
func (pr *PhaseResult) GetBundleStatus() (bool, string) {
	if pr.Result == nil || pr.Result.Bundle == nil {
		return true, ""
	}
	if !pr.Result.Bundle.Fresh {
		return false, strings.TrimSpace(pr.Result.Bundle.Reason)
	}
	return true, ""
}
