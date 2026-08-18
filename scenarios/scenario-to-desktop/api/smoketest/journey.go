package smoketest

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
	"scenario-to-desktop-api/captures"
	"scenario-to-desktop-api/procmetrics"
)

const (
	journeyPass                = string(deliveryramp.DispositionPass)
	journeyFailed              = string(deliveryramp.DispositionFailed)
	journeyDegraded            = string(deliveryramp.DispositionDegraded)
	journeyUnavailable         = string(deliveryramp.DispositionUnavailable)
	journeyUnsupported         = string(deliveryramp.DispositionUnsupported)
	journeyNotRun              = string(deliveryramp.DispositionNotRun)
	journeyStepPass            = string(deliveryramp.StepPassed)
	journeyStepFail            = string(deliveryramp.StepFailed)
	journeyStepDegraded        = string(deliveryramp.StepDegraded)
	journeyStepUnavailable     = string(deliveryramp.StepUnavailable)
	journeyStepNotRun          = string(deliveryramp.StepNotRun)
	journeyMainWindowMinWidth  = 600
	journeyMainWindowMinHeight = 400
	journeyStepTimeout         = 10 * time.Second
	journeyCaptureTimeout      = 5 * time.Second
)

type procmetricsDesktopDriver struct{ detector *procmetrics.XdotoolDetector }

func (d procmetricsDesktopDriver) IsAvailable(ctx context.Context) bool {
	return d.detector != nil && d.detector.IsAvailable(ctx)
}

func (d procmetricsDesktopDriver) LargestVisibleWindow(ctx context.Context, display string) (*procmetrics.WindowGeometry, error) {
	return d.detector.LargestVisibleWindow(ctx, 0, display)
}

func (d procmetricsDesktopDriver) WindowGeometry(ctx context.Context, display string) (*procmetrics.WindowGeometry, error) {
	return d.detector.WindowGeometry(ctx, 0, display)
}

func (d procmetricsDesktopDriver) ActivateWindow(ctx context.Context, display string) error {
	return d.detector.ActivateWindow(ctx, 0, display)
}

func (d procmetricsDesktopDriver) MaximizeWindow(ctx context.Context, display string, width, height int) error {
	return d.detector.MaximizeWindow(ctx, 0, display, width, height)
}

func (d procmetricsDesktopDriver) ResizeWindow(ctx context.Context, display string, width, height int) error {
	return d.detector.ResizeWindow(ctx, 0, display, width, height)
}

func (d procmetricsDesktopDriver) MoveWindow(ctx context.Context, display string, x, y int) error {
	return d.detector.MoveWindow(ctx, 0, display, x, y)
}

func (d procmetricsDesktopDriver) Click(ctx context.Context, display string, x, y, button int) error {
	return d.detector.Click(ctx, display, x, y, button)
}

func (d procmetricsDesktopDriver) KeyPress(ctx context.Context, display, key string) error {
	return d.detector.KeyPress(ctx, display, key)
}

func (d procmetricsDesktopDriver) Type(ctx context.Context, display, value string) error {
	return d.detector.Type(ctx, display, value)
}

type captureJourneySink struct{ service *DefaultService }

func (c captureJourneySink) Capture(ctx context.Context, smokeTestID, scenarioName, display, label string) (deliveryramp.EvidenceReference, error) {
	id, err := c.service.captureJourneyScreenshot(ctx, smokeTestID, scenarioName, display, label)
	if err != nil {
		return deliveryramp.EvidenceReference{}, err
	}
	return deliveryramp.EvidenceReference{ID: id, Kind: string(captures.CaptureScreenshot)}, nil
}

type defaultJourneyWaiter struct{ clock Clock }

func (w defaultJourneyWaiter) WaitUntil(ctx context.Context, policy deliveryramp.ReadinessPolicy, condition func(context.Context) (bool, string, error)) (WaitResult, error) {
	if policy.Timeout <= 0 || policy.PollInterval <= 0 {
		return WaitResult{}, fmt.Errorf("readiness policy %q must have positive timeout and poll interval", policy.ID)
	}
	deadline := w.clock.After(policy.Timeout)
	interval := w.clock.After(0)
	stabilityNeeded := policy.StabilityCount
	if stabilityNeeded < 1 {
		stabilityNeeded = 1
	}
	attempts := 0
	stable := 0
	for {
		select {
		case <-ctx.Done():
			return WaitResult{}, ctx.Err()
		case <-deadline:
			return WaitResult{Attempts: attempts}, fmt.Errorf("readiness policy %q timed out", policy.ID)
		case <-interval:
			attempts++
			ready, observed, err := condition(ctx)
			if err != nil {
				return WaitResult{Attempts: attempts}, err
			}
			if ready {
				stable++
				if stable >= stabilityNeeded {
					return WaitResult{Observed: observed, Attempts: attempts}, nil
				}
			} else {
				stable = 0
			}
			interval = w.clock.After(policy.PollInterval)
		}
	}
}

func (w defaultJourneyWaiter) Settle(ctx context.Context, policy deliveryramp.SettlePolicy) error {
	if policy.Minimum < 0 || policy.Maximum < policy.Minimum || policy.Maximum <= 0 {
		return fmt.Errorf("settle policy %q has invalid bounds", policy.ID)
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-w.clock.After(policy.Minimum):
		return nil
	}
}

type loopbackJourneyAPI struct{}

type monetizationJourneyAPI struct {
	resolver     func(context.Context, string) (string, error)
	scenarioName string
}

func (monetizationJourneyAPI) Greet(context.Context, string) (string, error) {
	return "", fmt.Errorf("greeting is not part of the monetization journey")
}

func (api monetizationJourneyAPI) Probe(ctx context.Context, operation string) (JourneyOperationResult, error) {
	baseURL := strings.TrimRight(os.Getenv("S2D_MONETIZATION_JOURNEY_URL"), "/")
	if baseURL == "" {
		baseURL = strings.TrimRight(os.Getenv("VROOLI_VALIDATION_RENDERER_URL"), "/")
	}
	if baseURL == "" && api.resolver != nil {
		resolved, err := api.resolver(ctx, api.scenarioName)
		if err != nil {
			return JourneyOperationResult{}, fmt.Errorf("resolve monetization journey renderer URL: %w", err)
		}
		baseURL = strings.TrimRight(resolved, "/")
	}
	if baseURL == "" {
		return JourneyOperationResult{}, fmt.Errorf("bundled scenario renderer URL is unavailable")
	}
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Scheme != "http" || parsed.Hostname() != "127.0.0.1" {
		return JourneyOperationResult{}, fmt.Errorf("monetization journey URL must target loopback HTTP")
	}
	parsed.Path = "/api/v1/internal/monetization/journey"
	query := parsed.Query()
	query.Set("operation", operation)
	parsed.RawQuery = query.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil) // #nosec G704 -- URL is restricted to loopback above.
	if err != nil {
		return JourneyOperationResult{}, err
	}
	if authorization := strings.TrimSpace(os.Getenv("S2D_MONETIZATION_AUTHORIZATION")); authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
	if identity := strings.TrimSpace(os.Getenv("S2D_MONETIZATION_USER_IDENTITY")); identity != "" {
		req.Header.Set("X-User-Email", identity)
	}
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req) // #nosec G704 -- request target is validated loopback-only.
	if err != nil {
		return JourneyOperationResult{}, fmt.Errorf("monetization journey request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return JourneyOperationResult{}, fmt.Errorf("monetization journey returned HTTP %d", resp.StatusCode)
	}
	var result JourneyOperationResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return JourneyOperationResult{}, fmt.Errorf("decode monetization journey result: %w", err)
	}
	return result, nil
}

func (loopbackJourneyAPI) Greet(ctx context.Context, expectedName string) (string, error) {
	baseURL := strings.TrimRight(os.Getenv("HELLO_DESKTOP_API_URL"), "/")
	if baseURL == "" {
		baseURL = "http://127.0.0.1:23100"
	}
	parsedBase, err := url.Parse(baseURL)
	if err != nil || parsedBase.Scheme != "http" || parsedBase.Hostname() != "127.0.0.1" {
		return "", fmt.Errorf("semantic bridge URL must target loopback HTTP")
	}
	parsedBase.Path = "/api/test/last-greeting"
	query := parsedBase.Query()
	query.Set("name", expectedName)
	parsedBase.RawQuery = query.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, parsedBase.String(), nil) // #nosec G704 -- URL is restricted to the local fixture bridge above.
	if err != nil {
		return "", err
	}
	response, err := (&http.Client{Timeout: 3 * time.Second}).Do(request) // #nosec G704 -- request target is validated loopback-only.
	if err != nil {
		return "", fmt.Errorf("semantic bridge request: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("semantic bridge returned HTTP %d", response.StatusCode)
	}
	var payload struct {
		Name    string `json:"name"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return "", fmt.Errorf("decode semantic bridge response: %w", err)
	}
	expectedMessage := "Hello, " + expectedName + "!"
	if payload.Name != expectedName || payload.Message != expectedMessage {
		return payload.Message, fmt.Errorf("semantic assertion mismatch: got %q, want %q", payload.Message, expectedMessage)
	}
	return payload.Message, nil
}

func (s *DefaultService) runDesktopJourney(ctx context.Context, smokeTestID, scenarioName, platform string, rec recordingState) deliveryramp.JourneyResult {
	capability := strings.TrimSpace(s.journeyCapability)
	if capability == "" {
		capability = strings.TrimSpace(os.Getenv("S2D_JOURNEY_CAPABILITY"))
	}
	if capability == "" {
		capability = capabilityForScenario(scenarioName)
	}
	return s.runDesktopJourneyCapability(ctx, smokeTestID, scenarioName, platform, rec, capability)
}

type journeySetup struct {
	base    deliveryramp.JourneyResult
	clock   Clock
	started time.Time
	input   JourneyInput
	plan    deliveryramp.JourneyPlan
	driver  DesktopDriver
	waiter  JourneyWaiter
	capture JourneyCapture
	api     JourneyAPIProbe
	actions map[string]JourneyAction
}

func (s *DefaultService) prepareJourney(ctx context.Context, smokeTestID, scenarioName, platform string, rec recordingState, capability string) (journeySetup, *deliveryramp.JourneyResult) {
	clock := s.journeyClock
	if clock == nil {
		clock = RealClock{}
	}
	started := clock.Now().UTC()
	input := JourneyInput{SmokeTestID: smokeTestID, ScenarioName: scenarioName, Platform: platform, Display: rec.displayID, DisplayWidth: rec.displayWidth, DisplayHeight: rec.displayHeight}
	fixture, ok := journeyFixture(capability)
	base := deliveryramp.JourneyResult{SchemaVersion: deliveryramp.JourneySchemaVersion, EvidenceVersion: deliveryramp.JourneyEvidenceVersion, SmokeTestID: smokeTestID, ScenarioName: scenarioName, Capability: capability, Platform: platform, Display: rec.displayID, WindowManager: rec.windowManager, Titlebar: rec.titlebar, RecordingStartedBeforeLaunch: rec.captureID != "", Disposition: deliveryramp.Disposition(journeyPass), CreatedAt: started}
	if !ok {
		return journeySetup{}, journeyResultWithFailure(base, clock, journeyUnavailable, "capability_not_registered")
	}
	plan := fixture.Plan(input)
	requestedProfile := strings.TrimSpace(os.Getenv("S2D_JOURNEY_PROFILE"))
	if requestedProfile != "" {
		var profileErr error
		plan, profileErr = applyJourneyProfile(plan, requestedProfile)
		if profileErr != nil {
			return journeySetup{}, journeyResultWithFailure(base, clock, journeyUnavailable, profileErr.Error())
		}
	}
	base.PlanID, base.Profile = plan.ID, plan.Profile
	if support, ok := fixture.(JourneySupportStatus); ok {
		if supported, reason := support.Supported(input); !supported {
			base.Disposition = deliveryramp.DispositionUnsupported
			base.DegradedReason = reason
			step := deliveryramp.JourneyStep{ID: "unsupported", Name: "unsupported", Purpose: plan.Purpose, Action: "unsupported", Disposition: deliveryramp.StepUnavailable, Error: reason, StartedAt: started}
			step = finishJourneyStep(step, clock, started)
			base.Steps = append(base.Steps, step)
			return journeySetup{}, journeyResultWithCompletion(base, clock)
		}
	}
	if runtime.GOOS != "linux" {
		return journeySetup{}, journeyResultWithFailure(base, clock, journeyUnavailable, "native_visual_runtime_not_supported_on_host")
	}
	if rec.windowManager == "" || !rec.titlebar {
		return journeySetup{}, journeyResultWithFailure(base, clock, journeyUnavailable, "window_manager_capability_unavailable")
	}
	driver := s.journeyDriver
	if driver == nil && s.windowDetector != nil {
		driver = procmetricsDesktopDriver{detector: s.windowDetector}
	}
	if driver == nil || !driver.IsAvailable(ctx) {
		return journeySetup{}, journeyResultWithFailure(base, clock, journeyDegraded, "xdotool_unavailable")
	}
	if err := ctx.Err(); err != nil {
		return journeySetup{}, journeyResultWithFailure(base, clock, journeyFailed, "context_"+err.Error())
	}
	waiter := s.journeyWaiter
	if waiter == nil {
		waiter = defaultJourneyWaiter{clock: clock}
	}
	capture := s.journeyCapture
	if capture == nil {
		capture = captureJourneySink{service: s}
	}
	api := s.journeyAPI
	if api == nil {
		if capability == "monetization.trust-boundary.v1" {
			api = monetizationJourneyAPI{resolver: s.apiURLResolver, scenarioName: scenarioName}
		} else {
			api = loopbackJourneyAPI{}
		}
	}
	return journeySetup{base: base, clock: clock, started: started, input: input, plan: plan, driver: driver, waiter: waiter, capture: capture, api: api, actions: fixture.Actions()}, nil
}

func journeyResultWithCompletion(result deliveryramp.JourneyResult, clock Clock) *deliveryramp.JourneyResult {
	result.CompletedAt = clock.Now().UTC()
	return &result
}

func journeyResultWithFailure(result deliveryramp.JourneyResult, clock Clock, disposition, reason string) *deliveryramp.JourneyResult {
	result.Disposition = deliveryramp.Disposition(disposition)
	result.DegradedReason = reason
	return journeyResultWithCompletion(result, clock)
}

func (s *DefaultService) runDesktopJourneyCapabilityLegacy(ctx context.Context, smokeTestID, scenarioName, platform string, rec recordingState, capability string) (result deliveryramp.JourneyResult) {
	setup, earlyResult := s.prepareJourney(ctx, smokeTestID, scenarioName, platform, rec, capability)
	if earlyResult != nil {
		return *earlyResult
	}
	base := setup.base
	cleanupNeeded := true
	defer func() {
		if cleanupNeeded {
			cleanupCtx, cancel := context.WithTimeout(context.Background(), journeyStepTimeout)
			if cleanupErr := setup.driver.KeyPress(cleanupCtx, rec.displayID, "alt+F4"); cleanupErr != nil {
				base.Disposition = deliveryramp.DispositionFailed
				base.DegradedReason = "cleanup_failed"
				s.recordJourneyEvent(&base, setup.clock, setup.started, "cleanup_failed", "", "clean_shutdown", "", cleanupErr.Error())
			} else {
				s.recordJourneyEvent(&base, setup.clock, setup.started, "cleanup_completed", "", "clean_shutdown", "", "")
			}
			cancel()
		}
		result = base
	}()

	for _, spec := range setup.plan.Steps {
		if err := ctx.Err(); err != nil {
			base.Disposition = deliveryramp.DispositionFailed
			base.DegradedReason = "context_" + err.Error()
			break
		}
		step, stop := s.runJourneyStep(ctx, &base, setup, spec, smokeTestID, scenarioName, rec)
		base.Steps = append(base.Steps, step)
		if stop {
			break
		}
		if spec.Action == "quit_app" {
			cleanupNeeded = false
		}
	}
	if base.Disposition == deliveryramp.Disposition(journeyPass) && !journeyHasScreenshotPairs(base.Steps) {
		base.Disposition, base.DegradedReason = deliveryramp.DispositionFailed, "interaction_screenshot_pair_missing"
	}
	base.CompletedAt = setup.clock.Now().UTC()
	return base
}

func (s *DefaultService) runJourneyStep(ctx context.Context, base *deliveryramp.JourneyResult, setup journeySetup, spec deliveryramp.JourneyStepSpec, smokeTestID, scenarioName string, rec recordingState) (deliveryramp.JourneyStep, bool) {
	step := deliveryramp.JourneyStep{ID: spec.ID, Name: spec.ID, Purpose: spec.Purpose, Action: spec.Action, Disposition: deliveryramp.StepDisposition(journeyStepPass), Readiness: spec.Readiness, Settle: spec.Settle, StartedAt: setup.clock.Now().UTC()}
	step.MonotonicStartMs = startedSubMs(setup.started, step.StartedAt)
	s.observeJourneyProcess(ctx, base, &step, setup, "before_action")
	s.recordJourneyEvent(base, setup.clock, setup.started, "step_started", step.ID, "", "", "")
	ready, err := s.waitForJourneyReadiness(ctx, setup, spec)
	if err != nil {
		eventType := "readiness_failed"
		if strings.Contains(strings.ToLower(err.Error()), "timed out") {
			eventType = "readiness_timeout"
		}
		s.recordJourneyEvent(base, setup.clock, setup.started, eventType, step.ID, spec.Readiness.ID, ready.Observed, err.Error())
		step.Disposition = deliveryramp.StepDisposition(dispositionForJourneyError(err))
		step.Error, step.DegradedReason = err.Error(), "readiness_failed"
		return s.finishJourneyStep(step, rec.recordingStartedAt, setup), true
	}
	if ready.Attempts > 1 {
		s.recordJourneyEvent(base, setup.clock, setup.started, "readiness_retry", step.ID, spec.Readiness.ID, ready.Observed, fmt.Sprintf("attempts=%d", ready.Attempts))
	}
	s.recordJourneyEvent(base, setup.clock, setup.started, "readiness_ready", step.ID, spec.Readiness.ID, ready.Observed, "")
	if spec.Capture {
		if captureErr := s.captureJourneyStep(ctx, &step, setup.capture, smokeTestID, scenarioName, rec.displayID, spec.ID, "before"); captureErr != nil {
			step.Disposition, step.Error, base.Disposition = deliveryramp.StepFailed, captureErr.Error(), deliveryramp.DispositionFailed
			return s.finishJourneyStep(step, rec.recordingStartedAt, setup), true
		}
	}
	action := setup.actions[spec.Action]
	if action == nil {
		action = genericJourneyAction(spec.Action)
	}
	if action == nil {
		step.Disposition, step.Error, base.Disposition = deliveryramp.StepUnavailable, "action is not supported by the selected capability", deliveryramp.DispositionUnsupported
		return s.finishJourneyStep(step, rec.recordingStartedAt, setup), true
	}
	actionCtx, cancel := context.WithTimeout(ctx, journeyStepTimeout)
	observation, actionErr := action(actionCtx, setup.driver, setup.api, setup.input)
	cancel()
	applyJourneyObservation(base, &step, spec, observation, actionErr, setup.input)
	s.observeJourneyProcess(ctx, base, &step, setup, "after_action")
	if settleErr := setup.waiter.Settle(ctx, spec.Settle); settleErr != nil && step.Error == "" {
		step.Disposition, step.Error, base.Disposition = deliveryramp.StepDisposition(dispositionForJourneyError(settleErr)), settleErr.Error(), deliveryramp.Disposition(dispositionForJourneyError(settleErr))
	}
	s.recordJourneyEvent(base, setup.clock, setup.started, "settle_completed", step.ID, spec.Settle.ID, spec.Settle.Reason, "")
	if spec.Capture {
		if captureErr := s.captureJourneyStep(ctx, &step, setup.capture, smokeTestID, scenarioName, rec.displayID, spec.ID, "after"); captureErr != nil {
			step.Disposition, step.Error, base.Disposition = deliveryramp.StepFailed, captureErr.Error(), deliveryramp.DispositionFailed
		}
	}
	step = s.finishJourneyStep(step, rec.recordingStartedAt, setup)
	s.recordJourneyEvent(base, setup.clock, setup.started, "step_completed", step.ID, "", step.ObservedState, step.Error)
	return step, step.Disposition != deliveryramp.StepDisposition(journeyStepPass)
}

func (s *DefaultService) waitForJourneyReadiness(ctx context.Context, setup journeySetup, spec deliveryramp.JourneyStepSpec) (WaitResult, error) {
	return setup.waiter.WaitUntil(ctx, spec.Readiness, func(checkCtx context.Context) (bool, string, error) {
		if spec.Readiness.ID != "target_window_visible" {
			return true, "condition_not_required", nil
		}
		window, err := setup.driver.LargestVisibleWindow(checkCtx, setup.input.Display)
		if err != nil {
			return false, "window_probe_error", err
		}
		if window != nil && window.Width >= journeyMainWindowMinWidth && window.Height >= journeyMainWindowMinHeight {
			return true, fmt.Sprintf("usable_window:%dx%d", window.Width, window.Height), nil
		}
		return false, "waiting_for_usable_window", nil
	})
}

func (s *DefaultService) captureJourneyStep(ctx context.Context, step *deliveryramp.JourneyStep, capture JourneyCapture, smokeTestID, scenarioName, display, stepID, position string) error {
	if step.Action == "" || capture == nil {
		return nil
	}
	captureCtx, cancel := context.WithTimeout(ctx, journeyCaptureTimeout)
	ref, err := capture.Capture(captureCtx, smokeTestID, scenarioName, display, position+"-"+stepID)
	cancel()
	if err != nil {
		return fmt.Errorf("%s screenshot: %w", position, err)
	}
	if position == "before" {
		step.BeforeCaptureID = ref.ID
	} else {
		step.AfterCaptureID = ref.ID
	}
	step.Evidence = append(step.Evidence, ref)
	return nil
}

func (s *DefaultService) observeJourneyProcess(ctx context.Context, base *deliveryramp.JourneyResult, step *deliveryramp.JourneyStep, setup journeySetup, phase string) {
	if s.journeyProcess == nil {
		return
	}
	if observed, err := s.journeyProcess.Observe(ctx); err == nil {
		if phase == "before_action" {
			step.ProcessBefore = observed
		} else {
			step.ProcessAfter = observed
		}
		s.recordJourneyEvent(base, setup.clock, setup.started, "process_observed", step.ID, "", observed, phase)
	}
}

func applyJourneyObservation(base *deliveryramp.JourneyResult, step *deliveryramp.JourneyStep, spec deliveryramp.JourneyStepSpec, observation JourneyObservation, actionErr error, input JourneyInput) {
	if actionErr != nil {
		step.Disposition = deliveryramp.StepDisposition(dispositionForJourneyError(actionErr))
		step.Error = actionErr.Error()
		base.Disposition = deliveryramp.Disposition(dispositionForJourneyError(actionErr))
		return
	}
	step.ObservedState, step.Geometry, step.Route = observation.Observed, observation.Geometry, observation.Route
	if observation.Provider != nil {
		base.ProviderObservation = observation.Provider
	}
	if spec.Assertion == nil {
		return
	}
	step.AssertionID, step.ExpectedState = spec.Assertion.ID, spec.Assertion.Expected
	step.AssertionStatus = assertionStatus(spec.Assertion.Expected, observation.Observed, observation.Geometry, input)
	if step.AssertionStatus != journeyStepPass {
		step.Disposition, step.Error, base.Disposition = deliveryramp.StepFailed, "assertion did not match observed result", deliveryramp.DispositionFailed
	}
}

func (s *DefaultService) finishJourneyStep(step deliveryramp.JourneyStep, recordingStartedAt time.Time, setup journeySetup) deliveryramp.JourneyStep {
	step = finishJourneyStep(step, setup.clock, setup.started)
	setVideoOffsets(&step, recordingStartedAt)
	return step
}

func genericJourneyAction(action string) JourneyAction {
	switch action {
	case "window_activate", "window_maximize", "pointer_click", "key_press", "window_resize", "window_move", "quit_app":
	default:
		return nil
	}
	return func(ctx context.Context, driver DesktopDriver, _ JourneyAPIProbe, input JourneyInput) (JourneyObservation, error) {
		var err error
		var geometry *procmetrics.WindowGeometry
		switch action {
		case "window_activate":
			err = driver.ActivateWindow(ctx, input.Display)
		case "window_maximize":
			err = driver.MaximizeWindow(ctx, input.Display, input.DisplayWidth, input.DisplayHeight)
			if err == nil {
				geometry, err = driver.WindowGeometry(ctx, input.Display)
			}
		case "pointer_click":
			err = driver.Click(ctx, input.Display, input.DisplayWidth/2, input.DisplayHeight/2, 1)
		case "key_press":
			err = driver.KeyPress(ctx, input.Display, "Return")
		case "window_resize":
			err = driver.ResizeWindow(ctx, input.Display, input.DisplayWidth*80/100, input.DisplayHeight*80/100)
			if err == nil {
				geometry, err = driver.WindowGeometry(ctx, input.Display)
			}
		case "window_move":
			err = driver.MoveWindow(ctx, input.Display, 32, 32)
			if err == nil {
				geometry, err = driver.WindowGeometry(ctx, input.Display)
			}
		case "quit_app":
			err = driver.KeyPress(ctx, input.Display, "alt+F4")
		}
		var sharedGeometry *deliveryramp.Geometry
		if geometry != nil {
			sharedGeometry = &deliveryramp.Geometry{X: geometry.X, Y: geometry.Y, Width: geometry.Width, Height: geometry.Height}
		}
		return JourneyObservation{Observed: action, Geometry: sharedGeometry}, err
	}
}

func assertionStatus(expected, observed string, geometry *deliveryramp.Geometry, input JourneyInput) string {
	if strings.HasPrefix(expected, "window covers") {
		if geometry != nil && geometry.Width >= input.DisplayWidth*90/100 && geometry.Height >= input.DisplayHeight*90/100 {
			return journeyStepPass
		}
		return journeyStepFail
	}
	if expected == observed {
		return journeyStepPass
	}
	return journeyStepFail
}

func dispositionForJourneyError(err error) string {
	if err == nil {
		return journeyStepPass
	}
	if err == context.Canceled || err == context.DeadlineExceeded {
		return journeyFailed
	}
	return journeyStepFail
}

func finishJourneyStep(step deliveryramp.JourneyStep, clock Clock, started time.Time) deliveryramp.JourneyStep {
	step.CompletedAt = clock.Now().UTC()
	step.MonotonicEndMs = startedSubMs(started, step.CompletedAt)
	return step
}

func setVideoOffsets(step *deliveryramp.JourneyStep, recordingStartedAt time.Time) {
	if step == nil || recordingStartedAt.IsZero() {
		return
	}
	start := step.StartedAt.Sub(recordingStartedAt).Milliseconds()
	end := step.CompletedAt.Sub(recordingStartedAt).Milliseconds()
	if start < 0 || end < start {
		return
	}
	step.VideoStartOffsetMs = &start
	step.VideoEndOffsetMs = &end
}

func startedSubMs(start, end time.Time) int64 {
	if end.Before(start) {
		return 0
	}
	return end.Sub(start).Milliseconds()
}

func (s *DefaultService) recordJourneyEvent(result *deliveryramp.JourneyResult, clock Clock, started time.Time, eventType, stepID, policyID, observed, reason string) {
	now := clock.Now().UTC()
	result.Events = append(result.Events, deliveryramp.JourneyEvent{Type: eventType, StepID: stepID, PolicyID: policyID, Observed: observed, StartedAt: now, CompletedAt: now, MonotonicStartMs: startedSubMs(started, now), MonotonicEndMs: startedSubMs(started, now), Reason: reason})
}

func journeyHasScreenshotPairs(steps []deliveryramp.JourneyStep) bool {
	if len(steps) == 0 {
		return false
	}
	for _, step := range steps {
		if step.Action == "screenshot" || step.Action == "window_geometry" {
			continue
		}
		if step.BeforeCaptureID == "" || step.AfterCaptureID == "" {
			return false
		}
	}
	return true
}

// ValidateJourneyTimeline is the fail-closed contract check used by the
// producer manifest and focused contract tests. It intentionally validates the
// persisted sidecar, not transient runner state.
func ValidateJourneyTimeline(journey deliveryramp.JourneyResult) error {
	if err := validateJourneyHeader(journey); err != nil {
		return err
	}
	if err := validateJourneySteps(journey); err != nil {
		return err
	}
	if err := validateJourneyEvents(journey); err != nil {
		return err
	}
	if journey.SchemaVersion >= deliveryramp.JourneySchemaVersion && len(journey.Steps) == 0 {
		return fmt.Errorf("versioned journey must contain chapters")
	}
	return validateJourneyRedaction(journey)
}

func validateJourneyHeader(journey deliveryramp.JourneyResult) error {
	if journey.SchemaVersion != 1 && journey.SchemaVersion != deliveryramp.JourneySchemaVersion {
		return fmt.Errorf("unsupported journey schema version %d", journey.SchemaVersion)
	}
	if strings.TrimSpace(journey.SmokeTestID) == "" || strings.TrimSpace(journey.ScenarioName) == "" {
		return fmt.Errorf("journey identity is required")
	}
	if journey.CreatedAt.IsZero() {
		return fmt.Errorf("journey start time is required")
	}
	if journey.SchemaVersion >= deliveryramp.JourneySchemaVersion {
		if strings.TrimSpace(journey.EvidenceVersion) == "" || strings.TrimSpace(journey.Capability) == "" || strings.TrimSpace(journey.PlanID) == "" {
			return fmt.Errorf("versioned journey requires evidence, capability, and plan identity")
		}
		if journey.CompletedAt.IsZero() {
			return fmt.Errorf("journey completion time is required")
		}
	}
	return nil
}

func validateJourneySteps(journey deliveryramp.JourneyResult) error {
	previousEnd := int64(0)
	stepIDs := make(map[string]struct{}, len(journey.Steps))
	for index, step := range journey.Steps {
		if step.StartedAt.IsZero() || step.CompletedAt.IsZero() || step.CompletedAt.Before(step.StartedAt) {
			return fmt.Errorf("journey step %d has invalid timestamps", index)
		}
		if step.MonotonicEndMs < step.MonotonicStartMs || step.MonotonicStartMs < previousEnd {
			return fmt.Errorf("journey step %d is out of order", index)
		}
		previousEnd = step.MonotonicEndMs
		if journey.SchemaVersion >= deliveryramp.JourneySchemaVersion && strings.TrimSpace(step.ID) == "" {
			return fmt.Errorf("journey step %d has no stable ID", index)
		}
		if step.ID != "" {
			if _, exists := stepIDs[step.ID]; exists {
				return fmt.Errorf("journey step %d repeats ID %q", index, step.ID)
			}
			stepIDs[step.ID] = struct{}{}
		}
		if step.VideoStartOffsetMs != nil && *step.VideoStartOffsetMs < 0 {
			return fmt.Errorf("journey step %d has invalid video start offset", index)
		}
		if step.VideoEndOffsetMs != nil && *step.VideoEndOffsetMs < 0 {
			return fmt.Errorf("journey step %d has invalid video end offset", index)
		}
		if step.VideoStartOffsetMs != nil && step.VideoEndOffsetMs != nil && *step.VideoEndOffsetMs < *step.VideoStartOffsetMs {
			return fmt.Errorf("journey step %d has invalid video offsets", index)
		}
	}
	return nil
}

func validateJourneyEvents(journey deliveryramp.JourneyResult) error {
	previousEvent := int64(0)
	for index, event := range journey.Events {
		if event.StartedAt.IsZero() || event.CompletedAt.IsZero() || event.CompletedAt.Before(event.StartedAt) || event.MonotonicEndMs < event.MonotonicStartMs || event.MonotonicStartMs < previousEvent {
			return fmt.Errorf("journey event %d is out of order", index)
		}
		previousEvent = event.MonotonicEndMs
	}
	return nil
}

func validateJourneyRedaction(journey deliveryramp.JourneyResult) error {
	data, err := json.Marshal(journey)
	if err != nil {
		return fmt.Errorf("serialize journey for redaction check: %w", err)
	}
	for _, forbidden := range []string{"authorization", "bearer ", "client_secret", "private_key"} {
		if strings.Contains(strings.ToLower(string(data)), forbidden) {
			return fmt.Errorf("journey contains forbidden credential material")
		}
	}
	return nil
}

func reviewFromJourney(journey deliveryramp.JourneyResult) *JourneyReview {
	review := &JourneyReview{SchemaVersion: journey.EvidenceVersion, Capability: journey.Capability, PlanID: journey.PlanID, Profile: journey.Profile, Disposition: string(journey.Disposition), Reason: journey.DegradedReason, EventCount: len(journey.Events), WorkflowRequired: journey.WorkflowRequired, WorkflowReference: journey.WorkflowReference, Chapters: make([]JourneyChapter, 0, len(journey.Steps))}
	if provider := journey.ProviderObservation; provider != nil {
		review.DeploymentMode = provider.DeploymentMode
		review.ProviderTier = provider.ProviderTier
		review.ServiceIdentity = provider.ServiceIdentity
		review.Readiness = provider.Readiness
		review.FallbackDecision = provider.FallbackDecision
		review.SafeRouteClass = provider.SafeRouteClass
	}
	for _, step := range journey.Steps {
		chapter := JourneyChapter{ID: step.ID, Purpose: step.Purpose, Action: step.Action, Disposition: string(step.Disposition), AssertionID: step.AssertionID, Expected: step.ExpectedState, Observed: step.ObservedState, Error: step.Error, VideoStartOffsetMs: step.VideoStartOffsetMs, VideoEndOffsetMs: step.VideoEndOffsetMs}
		for _, ref := range step.Evidence {
			chapter.EvidenceIDs = append(chapter.EvidenceIDs, ref.ID)
		}
		review.Chapters = append(review.Chapters, chapter)
	}
	return review
}

func (s *DefaultService) captureJourneyScreenshot(ctx context.Context, smokeTestID, scenarioName, display, label string) (string, error) {
	if s.captures == nil {
		return "", fmt.Errorf("capture storage unavailable")
	}
	tmpDir, err := os.MkdirTemp("", "vrooli-journey-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmpDir)
	path := filepath.Join(tmpDir, label+".png")
	command := `xwd -display "$DISPLAY" -root -silent | ffmpeg -y -f xwd_pipe -i - -frames:v 1 -update 1 "$OUTPUT"`
	result, err := runJourneyCommand(ctx, "sh", []string{"-c", command}, []string{"DISPLAY=" + display, "OUTPUT=" + path})
	if err != nil || result != nil && result.ExitCode != 0 {
		if err == nil {
			err = fmt.Errorf("screenshot command exited with code %d", result.ExitCode)
		}
		return "", err
	}
	cap, err := s.captures.SaveCapture(scenarioName, captures.CaptureScreenshot, "smoke-test:"+smokeTestID, path, 0, 0, 0)
	if err != nil {
		return "", err
	}
	return cap.ID, nil
}

func runJourneyCommand(ctx context.Context, command string, args []string, env []string) (*ExecutionResult, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("journey command args are required")
	}
	process := exec.CommandContext(ctx, command, args...)
	process.Env = append(os.Environ(), env...)
	output, err := process.CombinedOutput()
	result := &ExecutionResult{Combined: string(output), ExitCode: 0}
	if err != nil {
		result.ExitCode = 1
	}
	return result, err
}

func (s *DefaultService) persistJourney(journey deliveryramp.JourneyResult) string {
	if s.captures == nil {
		return ""
	}
	data, err := json.MarshalIndent(journey, "", "  ")
	if err != nil {
		return ""
	}
	tmp, err := os.CreateTemp("", "vrooli-journey-*.json")
	if err != nil {
		return ""
	}
	path := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(path)
		return ""
	}
	_ = tmp.Close()
	defer os.Remove(path)
	cap, err := s.captures.SaveCapture(journey.ScenarioName, captures.CaptureJourney, "smoke-test:"+journey.SmokeTestID, path, 0, 0, 0)
	if err != nil {
		return ""
	}
	return cap.ID
}
