package smoke

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"
	"time"

	"test-genie/internal/browsercapture"
	"test-genie/internal/evidence"
	"test-genie/internal/orchestrator/targetruntime"
	"test-genie/internal/pagediscovery"
	"test-genie/internal/smoke/artifacts"
	"test-genie/internal/smoke/preflight"
	"test-genie/internal/smoke/smokeconfig"
)

// Runner drives a UI smoke test on the BAS workflow engine. It runs the
// engine-agnostic preflight (UI directory, bundle freshness, UI-port discovery,
// auto-start, iframe-bridge dependency), captures evidence via the BAS workflow
// client, judges it with the shared evidence analyzer, and persists artifacts.
type Runner struct {
	logger    io.Writer
	timeout   time.Duration
	uiURL     string // optional override (skips auto-detection)
	autoStart bool

	// capturer drives the BAS workflow capture (single-page handshake smoke).
	// Required.
	capturer *browsercapture.Capturer
	// checker validates preconditions. Defaulted when nil.
	checker *preflight.Checker
	// freshness reports UI bundle staleness via the canonical content-hash
	// freshness engine. Defaulted to the CLI-backed checker when nil.
	freshness freshnessChecker
	// starter brings up the scenario when its UI port is absent. Defaulted to
	// the lifecycle-backed targetruntime starter when auto-start is enabled.
	starter scenarioStarter

	// allPages enables the all-pages visual capture mode (gated by the baseline
	// capture profile). Default false keeps smoke single-page (unchanged cost).
	allPages bool
	// captureVideo requests a VIDEO artifact per page in all-pages mode.
	captureVideo bool
	// multiCapturer issues one CaptureService.Capture per discovered page.
	// Required only when allPages is true.
	multiCapturer *browsercapture.MultiCapturer
	// pageDiscoverer enumerates the scenario's pages. Defaulted to the real
	// filesystem discoverer when nil.
	pageDiscoverer *pagediscovery.Discoverer

	// runID is captured at Run() time so the all-pages writer can key artifacts.
	runID string
}

// discoverer returns the configured page discoverer, defaulting to the real
// filesystem.
func (r *Runner) discoverer() *pagediscovery.Discoverer {
	if r.pageDiscoverer == nil {
		r.pageDiscoverer = pagediscovery.New(nil)
	}
	return r.pageDiscoverer
}

// scenarioStarter starts a scenario and reports its UI port. Implemented by the
// lifecycle-backed targetruntime; overridable in tests.
type scenarioStarter interface {
	Start(ctx context.Context, scenarioName string) (started bool, uiPort int, err error)
	Stop(ctx context.Context, scenarioName string) error
}

// NewRunner creates a Runner driving the given BAS workflow capturer.
func NewRunner(capturer *browsercapture.Capturer, opts ...RunnerOption) *Runner {
	r := &Runner{
		logger:   io.Discard,
		capturer: capturer,
	}
	for _, opt := range opts {
		opt(r)
	}
	if r.checker == nil {
		r.checker = preflight.NewChecker()
	}
	if r.freshness == nil {
		r.freshness = newCLIFreshnessChecker()
	}
	if r.starter == nil {
		r.starter = lifecycleStarter{}
	}
	return r
}

// RunnerOption configures a Runner.
type RunnerOption func(*Runner)

// WithRunnerLogger sets the logger for the runner.
func WithRunnerLogger(w io.Writer) RunnerOption {
	return func(r *Runner) { r.logger = w }
}

// WithRunnerTimeout sets a custom overall timeout for the runner.
func WithRunnerTimeout(timeout time.Duration) RunnerOption {
	return func(r *Runner) { r.timeout = timeout }
}

// WithUIURL sets a custom UI URL, bypassing auto-detection.
func WithUIURL(uiURL string) RunnerOption {
	return func(r *Runner) { r.uiURL = uiURL }
}

// WithAutoStart enables automatic scenario startup if the UI port is not detected.
func WithAutoStart(enabled bool) RunnerOption {
	return func(r *Runner) { r.autoStart = enabled }
}

// WithChecker overrides the preflight checker (for testing).
func WithChecker(c *preflight.Checker) RunnerOption {
	return func(r *Runner) { r.checker = c }
}

// WithFreshnessChecker overrides the UI bundle freshness checker (for testing).
func WithFreshnessChecker(f freshnessChecker) RunnerOption {
	return func(r *Runner) { r.freshness = f }
}

// WithScenarioStarter overrides the scenario starter (for testing).
func WithScenarioStarter(s scenarioStarter) RunnerOption {
	return func(r *Runner) { r.starter = s }
}

// WithAllPagesCapture enables the all-pages visual capture mode driven by the
// given MultiCapturer. video requests a per-page VIDEO artifact. This is gated
// by the baseline capture profile; default smoke leaves it off.
func WithAllPagesCapture(mc *browsercapture.MultiCapturer, video bool) RunnerOption {
	return func(r *Runner) {
		r.allPages = mc != nil
		r.multiCapturer = mc
		r.captureVideo = video
	}
}

// WithPageDiscoverer overrides the page discoverer (for testing).
func WithPageDiscoverer(d *pagediscovery.Discoverer) RunnerOption {
	return func(r *Runner) { r.pageDiscoverer = d }
}

// Run executes a UI smoke test for the given scenario. runID keys all artifacts
// under coverage/runs/<runID>/.
func (r *Runner) Run(ctx context.Context, scenarioName, scenarioDir, runID string) (*Result, error) {
	r.runID = runID
	smokeCfg := smokeconfig.LoadUISmokeConfig(scenarioDir)
	if !smokeCfg.Enabled {
		return Skipped(scenarioName, "UI smoke harness disabled via .vrooli/testing.json"), nil
	}

	handshakeTimeout := DefaultHandshakeTimeout
	if smokeCfg.HandshakeTimeoutMs > 0 {
		handshakeTimeout = time.Duration(smokeCfg.HandshakeTimeoutMs) * time.Millisecond
	}

	writer := artifacts.NewWriter(artifacts.WithRunID(runID))
	startTime := time.Now()

	// Step 1: UI directory present?
	if !r.checker.CheckUIDirectory(scenarioDir) {
		r.log("No UI directory detected, skipping smoke test")
		return r.persist(ctx, writer, scenarioDir, scenarioName, Skipped(scenarioName, "UI directory not detected")), nil
	}

	// Step 2: bundle freshness, via the canonical content-hash engine. A
	// resolution error is non-fatal (graceful degradation): log and proceed
	// rather than blocking a render on an infra hiccup.
	var bundleStatus *BundleStatus
	if stale, reason, err := r.freshness.UIBundleStale(ctx, scenarioName, scenarioDir); err != nil {
		r.log("Bundle freshness check failed: %v", err)
	} else {
		bundleStatus = &BundleStatus{Fresh: !stale, Reason: reason}
		if stale {
			r.log("UI bundle is stale: %s", reason)
			result := Blocked(scenarioName,
				fmt.Sprintf("%s\n  ↳ Fix: vrooli scenario restart %s\n  ↳ Then verify: vrooli scenario ui-smoke %s",
					reason, scenarioName, scenarioName),
				BlockedReasonBundleStale)
			result.Bundle = bundleStatus
			return r.persist(ctx, writer, scenarioDir, scenarioName, result), nil
		}
		r.log("UI bundle is fresh")
	}

	// Step 3: resolve the UI URL (explicit override, discovery, or auto-start).
	uiURL, blocked := r.resolveUIURL(ctx, scenarioName, scenarioDir)
	if blocked != nil {
		return r.persist(ctx, writer, scenarioDir, scenarioName, blocked), nil
	}
	if uiURL == "" {
		r.log("No UI port defined in service.json, skipping smoke test")
		return r.persist(ctx, writer, scenarioDir, scenarioName, Skipped(scenarioName, "Scenario does not define a UI port")), nil
	}

	// Step 4: iframe-bridge dependency.
	var bridgeStatus *BridgeStatus
	if bs, err := r.checker.CheckIframeBridge(ctx, scenarioDir); err != nil {
		r.log("iframe-bridge check failed: %v", err)
	} else if bs != nil {
		bridgeStatus = &BridgeStatus{DependencyPresent: bs.DependencyPresent, Version: bs.Version, Details: bs.Details}
		if !bs.DependencyPresent {
			r.log("iframe-bridge dependency missing")
			result := Failed(scenarioName, "@vrooli/iframe-bridge dependency missing in ui/package.json")
			result.IframeBridge = bridgeStatus
			return r.persist(ctx, writer, scenarioDir, scenarioName, result), nil
		}
		r.log("iframe-bridge dependency present")
	}

	// Step 5: capture evidence via the BAS workflow engine.
	r.log("Capturing UI smoke evidence for %s via BAS workflow engine", uiURL)
	captureCtx := ctx
	if r.timeout > 0 {
		var cancel context.CancelFunc
		captureCtx, cancel = context.WithTimeout(ctx, r.timeout)
		defer cancel()
	}
	capture, captureErr := r.capturer.Capture(captureCtx, browsercapture.Request{
		ScenarioURL:      uiURL,
		HandshakeSignals: smokeCfg.HandshakeSignals,
		HandshakeTimeout: handshakeTimeout,
		ViewportWidth:    DefaultViewportWidth,
		ViewportHeight:   DefaultViewportHeight,
	})
	if captureErr != nil {
		r.log("BAS smoke capture failed: %v", captureErr)
	}
	ev := capture.Evidence
	applyRenderHealth(&ev, capture.Screenshot)

	// Step 6: verdict via the shared evidence analyzer (single authority).
	verdict := evidence.Analyze(ev)
	result := r.buildResult(scenarioName, uiURL, ev, verdict, bundleStatus, bridgeStatus, time.Since(startTime))

	// Step 6b: all-pages visual capture (baseline capture profile only). Runs in
	// addition to the single-page handshake smoke; a failing page demotes the
	// overall result. Default depth skips this entirely (unchanged cost).
	if r.allPages && r.multiCapturer != nil {
		pagesResult := r.runAllPages(ctx, scenarioDir, scenarioName)
		result.PageCaptures = pagesResult.Captures
		if pagesResult.Failed && result.Status == StatusPassed {
			result.Status = StatusFailed
			result.Message = pagesResult.FailureMessage
		}
	}

	// Step 7: persist artifacts (screenshot + console + network + raw).
	r.writeArtifacts(ctx, writer, scenarioDir, scenarioName, ev, capture.Screenshot, result)

	r.log("Handshake: signaled=%t timed_out=%t duration=%dms error=%s",
		ev.Handshake.Signaled, ev.Handshake.TimedOut, ev.Handshake.DurationMs, ev.Handshake.Error)
	r.log("UI telemetry: network_failures=%d page_errors=%d console_errors=%d console_warnings=%d",
		result.NetworkFailureCount, result.PageErrorCount, result.ConsoleErrorCount, result.ConsoleWarningCount)

	return r.persist(ctx, writer, scenarioDir, scenarioName, result), nil
}

// resolveUIURL resolves the scenario's UI URL. It honors an explicit override,
// then port discovery, then (when enabled) auto-start. Returns a blocked result
// when a UI port is defined but unreachable; an empty URL with nil block means
// the scenario genuinely has no UI.
func (r *Runner) resolveUIURL(ctx context.Context, scenarioName, scenarioDir string) (string, *Result) {
	if strings.TrimSpace(r.uiURL) != "" {
		return r.uiURL, nil
	}

	if port, err := r.checker.CheckUIPort(ctx, scenarioName); err != nil {
		r.log("UI port discovery failed: %v", err)
	} else if port > 0 {
		r.log("Discovered UI port: %d", port)
		return fmt.Sprintf("http://localhost:%d", port), nil
	}

	portDef, err := r.checker.CheckUIPortDefined(scenarioDir)
	if err != nil {
		r.log("UI port definition check failed: %v", err)
	}
	if portDef == nil || !portDef.Defined {
		return "", nil
	}

	r.log("UI port defined in service.json (%s) but not detected - scenario may not be running", portDef.EnvVar)
	if r.autoStart && r.starter != nil {
		r.log("Attempting auto-start of scenario %s...", scenarioName)
		if _, uiPort, startErr := r.starter.Start(ctx, scenarioName); startErr != nil {
			r.log("Auto-start failed: %v", startErr)
		} else if uiPort > 0 {
			uiURL := fmt.Sprintf("http://localhost:%d", uiPort)
			r.log("Auto-start succeeded, UI available at %s", uiURL)
			return uiURL, nil
		}
	}

	var message string
	if r.autoStart {
		message = fmt.Sprintf("UI port is defined in service.json (%s) but not detected.\n"+
			"  ↳ Auto-start was attempted but failed.\n"+
			"  ↳ Fix: vrooli scenario restart %s\n"+
			"  ↳ Then check: vrooli scenario logs %s --step start-ui",
			portDef.EnvVar, scenarioName, scenarioName)
	} else {
		message = fmt.Sprintf("UI port is defined in service.json (%s) but not detected.\n"+
			"  ↳ The scenario may not be running or the UI server failed to start.\n"+
			"  ↳ Fix: vrooli scenario restart %s\n"+
			"  ↳ Or use: --auto-start to automatically start the scenario\n"+
			"  ↳ Then check: vrooli scenario logs %s --step start-ui",
			portDef.EnvVar, scenarioName, scenarioName)
	}
	return "", Blocked(scenarioName, message, BlockedReasonUIPortMissing)
}

// buildResult maps the analyzed evidence into the smoke Result.
func (r *Runner) buildResult(scenarioName, uiURL string, ev evidence.Evidence, verdict evidence.Verdict, bundle *BundleStatus, bridge *BridgeStatus, duration time.Duration) *Result {
	status := StatusPassed
	if !verdict.Passed() {
		status = StatusFailed
	}
	result := &Result{
		Scenario:   scenarioName,
		Status:     status,
		Message:    verdict.Message,
		Timestamp:  time.Now().UTC(),
		DurationMs: duration.Milliseconds(),
		UIURL:      uiURL,
		Handshake: HandshakeResult{
			Signaled:   ev.Handshake.Signaled,
			TimedOut:   ev.Handshake.TimedOut,
			DurationMs: ev.Handshake.DurationMs,
			Error:      ev.Handshake.Error,
		},
		NetworkFailureCount: verdict.NetworkFailureCount,
		PageErrorCount:      verdict.PageErrorCount,
		ConsoleErrorCount:   verdict.ConsoleErrorCount,
		ConsoleWarningCount: verdict.ConsoleWarningCount,
		Bundle:              bundle,
		IframeBridge:        bridge,
		StorageShim:         storageShimFromEvidence(ev),
	}
	if raw, err := json.Marshal(ev); err == nil {
		result.Raw = raw
	}
	return result
}

// writeArtifacts persists the screenshot, console, network, and raw evidence and
// records the resulting paths on the result.
func (r *Runner) writeArtifacts(ctx context.Context, writer *artifacts.Writer, scenarioDir, scenarioName string, ev evidence.Evidence, screenshot []byte, result *Result) {
	in := artifacts.Input{
		Screenshot: screenshot,
		Console:    consoleArtifacts(ev),
		Network:    networkArtifacts(ev),
	}
	if raw, err := json.Marshal(ev); err == nil {
		in.Raw = raw
	}
	paths, err := writer.WriteAll(ctx, scenarioDir, scenarioName, in)
	if err != nil {
		r.log("Failed to write artifacts: %v", err)
		return
	}
	result.Artifacts = ArtifactPaths{
		Screenshot: paths.Screenshot,
		Console:    paths.Console,
		Network:    paths.Network,
		Raw:        paths.Raw,
	}
}

// persist writes the result summary JSON + README and stamps the README path.
func (r *Runner) persist(ctx context.Context, writer *artifacts.Writer, scenarioDir, scenarioName string, result *Result) *Result {
	summary := result.summary()
	if err := writer.WriteResultJSON(ctx, scenarioDir, scenarioName, result, summary); err != nil {
		r.log("Failed to persist result: %v", err)
	}
	if readmePath, err := writer.WriteReadme(ctx, scenarioDir, scenarioName, summary); err != nil {
		r.log("Failed to write README: %v", err)
	} else {
		result.Artifacts.Readme = readmePath
	}
	return result
}

// summary builds the engine-agnostic artifact summary from a Result.
func (r *Result) summary() artifacts.Summary {
	s := artifacts.Summary{
		Scenario:   r.Scenario,
		Status:     string(r.Status),
		Message:    r.Message,
		Timestamp:  r.Timestamp,
		DurationMs: r.DurationMs,
		UIURL:      r.UIURL,
		Handshake: artifacts.HandshakeSummary{
			Signaled:   r.Handshake.Signaled,
			TimedOut:   r.Handshake.TimedOut,
			DurationMs: r.Handshake.DurationMs,
			Error:      r.Handshake.Error,
		},
		Paths: artifacts.Paths{
			Screenshot: r.Artifacts.Screenshot,
			Console:    r.Artifacts.Console,
			Network:    r.Artifacts.Network,
			Raw:        r.Artifacts.Raw,
			Readme:     r.Artifacts.Readme,
		},
	}
	if r.Bundle != nil {
		s.BundleKnown = true
		s.BundleFresh = r.Bundle.Fresh
		s.BundleReason = r.Bundle.Reason
	}
	return s
}

func consoleArtifacts(ev evidence.Evidence) []artifacts.ConsoleEntry {
	if len(ev.Console) == 0 {
		return nil
	}
	out := make([]artifacts.ConsoleEntry, len(ev.Console))
	for i, c := range ev.Console {
		out[i] = artifacts.ConsoleEntry{Level: c.Level, Message: c.Message}
	}
	return out
}

func networkArtifacts(ev evidence.Evidence) []artifacts.NetworkEntry {
	if len(ev.Network) == 0 {
		return nil
	}
	out := make([]artifacts.NetworkEntry, len(ev.Network))
	for i, n := range ev.Network {
		out[i] = artifacts.NetworkEntry{
			URL:          n.URL,
			Method:       n.Method,
			ResourceType: n.ResourceType,
			Status:       n.Status,
			ErrorText:    n.ErrorText,
		}
	}
	return out
}

func storageShimFromEvidence(ev evidence.Evidence) []StorageShimEntry {
	if len(ev.StorageShim) == 0 {
		return nil
	}
	out := make([]StorageShimEntry, len(ev.StorageShim))
	for i, s := range ev.StorageShim {
		out[i] = StorageShimEntry{Prop: s.Prop, Patched: s.Patched, Reason: s.Reason}
	}
	return out
}

func (r *Runner) log(format string, args ...any) {
	if r.logger == nil {
		return
	}
	msg := fmt.Sprintf(format, args...)
	if !strings.HasSuffix(msg, "\n") {
		msg += "\n"
	}
	fmt.Fprint(r.logger, msg)
}

// lifecycleStarter brings a scenario up through the lifecycle-backed
// targetruntime manager (never by executing binaries directly).
type lifecycleStarter struct{}

func (lifecycleStarter) Start(ctx context.Context, scenarioName string) (bool, int, error) {
	manager := targetruntime.New(scenarioName, "")
	lease, err := manager.EnsureRunning(ctx, targetruntime.Needs{UI: true}, nil)
	if err != nil {
		return false, 0, err
	}
	port, err := portFromURL(lease.URLs.UI)
	if err != nil {
		return lease.Started, 0, err
	}
	return lease.Started, port, nil
}

func (lifecycleStarter) Stop(ctx context.Context, scenarioName string) error {
	return targetruntime.New(scenarioName, "").Cleanup(ctx, targetruntime.Lease{Started: true}, nil)
}

func portFromURL(raw string) (int, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return 0, err
	}
	port := parsed.Port()
	if port == "" {
		return 0, fmt.Errorf("runtime URL %q does not include a port", raw)
	}
	return strconv.Atoi(port)
}
