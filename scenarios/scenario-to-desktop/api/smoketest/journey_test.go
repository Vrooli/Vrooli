package smoketest

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"scenario-to-desktop-api/procmetrics"
)

type journeyTestDriver struct {
	geometry *procmetrics.WindowGeometry
	closed   bool
}

func (d *journeyTestDriver) IsAvailable(context.Context) bool { return true }
func (d *journeyTestDriver) LargestVisibleWindow(context.Context, string) (*procmetrics.WindowGeometry, error) {
	return d.geometry, nil
}

func (d *journeyTestDriver) WindowGeometry(context.Context, string) (*procmetrics.WindowGeometry, error) {
	return d.geometry, nil
}
func (d *journeyTestDriver) ActivateWindow(context.Context, string) error           { return nil }
func (d *journeyTestDriver) MaximizeWindow(context.Context, string, int, int) error { return nil }
func (d *journeyTestDriver) ResizeWindow(context.Context, string, int, int) error   { return nil }
func (d *journeyTestDriver) MoveWindow(context.Context, string, int, int) error     { return nil }
func (d *journeyTestDriver) Click(context.Context, string, int, int, int) error     { return nil }
func (d *journeyTestDriver) KeyPress(_ context.Context, _ string, key string) error {
	if key == "alt+F4" {
		return nil
	}
	return nil
}
func (d *journeyTestDriver) Type(context.Context, string, string) error { return nil }

type journeyTestCapture struct{ next int }

func (c *journeyTestCapture) Capture(context.Context, string, string, string, string) (EvidenceReference, error) {
	c.next++
	return EvidenceReference{ID: "capture-" + string(rune('a'+c.next)), Kind: "screenshot"}, nil
}

type journeyTestAPI struct{}

func (journeyTestAPI) Greet(_ context.Context, name string) (string, error) {
	return "Hello, " + name + "!", nil
}

func (journeyTestAPI) Probe(_ context.Context, operation string) (JourneyOperationResult, error) {
	if operation == "provider_observation" {
		return JourneyOperationResult{
			Observed: "provider=managed-private;route=private-bundle",
			Route:    "private-bundle",
			Provider: &JourneyProviderObservation{DeploymentMode: "bundled", ProviderTier: "managed-private", ServiceIdentity: "fixture", Readiness: "ready", SafeRouteClass: "private-bundle"},
		}, nil
	}
	return JourneyOperationResult{Observed: "operation=bundled-private;mode=bundled-private", Route: "private-bundle"}, nil
}

type communicationContractAPI struct {
	provider  string
	route     string
	mode      string
	operation string
}

func (communicationContractAPI) Greet(_ context.Context, name string) (string, error) {
	return "Hello, " + name + "!", nil
}

func (a communicationContractAPI) Probe(_ context.Context, operation string) (JourneyOperationResult, error) {
	if operation == "provider_observation" {
		return JourneyOperationResult{
			Observed: "provider=" + a.provider + ";route=" + a.route,
			Route:    a.route,
			Provider: &JourneyProviderObservation{
				DeploymentMode:  a.mode,
				ProviderTier:    a.provider,
				ServiceIdentity: "communication-fixture",
				Readiness:       "ready",
				SafeRouteClass:  a.route,
			},
		}, nil
	}
	return JourneyOperationResult{
		Observed: "operation=" + a.operation + ";mode=" + a.mode,
		Route:    a.route,
	}, nil
}

type journeyTestWaiter struct {
	err         error
	waitCount   int
	settleCount int
}

func (w *journeyTestWaiter) WaitUntil(context.Context, ReadinessPolicy, func(context.Context) (bool, string, error)) (WaitResult, error) {
	w.waitCount++
	if w.err != nil {
		return WaitResult{}, w.err
	}
	return WaitResult{Observed: "fake-ready", Attempts: 1}, nil
}

func (w *journeyTestWaiter) Settle(context.Context, SettlePolicy) error {
	w.settleCount++
	return nil
}

func TestDesktopJourney_RegisteredFixtureProducesReviewableTimeline(t *testing.T) {
	driver := &journeyTestDriver{geometry: &procmetrics.WindowGeometry{Width: 1280, Height: 720}}
	waiter := &journeyTestWaiter{}
	service := &DefaultService{journeyDriver: driver, journeyClock: RealClock{}, journeyWaiter: waiter, journeyCapture: &journeyTestCapture{}, journeyAPI: journeyTestAPI{}}
	result := service.runDesktopJourney(context.Background(), "smoke-1", "hello-desktop", "linux", recordingState{
		captureID: "recording-1", displayID: ":99", displayWidth: 1280, displayHeight: 720, windowManager: "openbox", titlebar: true,
	})

	if result.Disposition != journeyPass {
		t.Fatalf("disposition = %q, want pass: %+v", result.Disposition, result)
	}
	if result.SchemaVersion != JourneySchemaVersion || result.PlanID == "" || result.Capability != "hello-desktop" {
		t.Fatalf("missing versioned plan identity: %+v", result)
	}
	if len(result.Steps) != 8 || len(result.Events) == 0 {
		t.Fatalf("steps/events = %d/%d, want 8/non-empty", len(result.Steps), len(result.Events))
	}
	semantic := result.Steps[2]
	if semantic.Purpose == "" || semantic.AssertionStatus != JourneyStepPassed || semantic.ObservedState != semantic.ExpectedState {
		t.Fatalf("semantic step lacks assertion evidence: %+v", semantic)
	}
	if !journeyHasScreenshotPairs(result.Steps) {
		t.Fatal("successful fixture must have before/after evidence for every action")
	}
	if waiter.waitCount != len(result.Steps) || waiter.settleCount != len(result.Steps) {
		t.Fatalf("wait/settle calls = %d/%d, want %d/%d", waiter.waitCount, waiter.settleCount, len(result.Steps), len(result.Steps))
	}
}

func TestDesktopJourney_MissingCapabilityIsUnavailable(t *testing.T) {
	service := &DefaultService{}
	result := service.runDesktopJourneyCapability(context.Background(), "smoke-1", "demo", "linux", recordingState{}, "not-registered")
	if result.Disposition != JourneyDispositionUnavailable || result.DegradedReason != "capability_not_registered" {
		t.Fatalf("result = %+v, want unavailable capability_not_registered", result)
	}
}

func TestDesktopJourney_ReadinessTimeoutFailsClosedAndCleansUp(t *testing.T) {
	driver := &journeyTestDriver{geometry: &procmetrics.WindowGeometry{Width: 1280, Height: 720}}
	waiter := &journeyTestWaiter{err: errors.New("readiness policy timed out")}
	service := &DefaultService{journeyDriver: driver, journeyClock: RealClock{}, journeyWaiter: waiter, journeyCapture: &journeyTestCapture{}, journeyAPI: journeyTestAPI{}}
	result := service.runDesktopJourney(context.Background(), "smoke-1", "hello-desktop", "linux", recordingState{captureID: "rec", displayID: ":99", displayWidth: 1280, displayHeight: 720, windowManager: "openbox", titlebar: true})
	if result.Disposition == journeyPass || len(result.Steps) != 1 || !strings.Contains(result.Steps[0].Error, "timed out") {
		t.Fatalf("timeout result = %+v, want failed step and no pass", result)
	}
}

func TestDesktopJourney_CancellationDoesNotPass(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	service := &DefaultService{journeyDriver: &journeyTestDriver{geometry: &procmetrics.WindowGeometry{Width: 1280, Height: 720}}, journeyClock: RealClock{}, journeyWaiter: &journeyTestWaiter{}, journeyCapture: &journeyTestCapture{}, journeyAPI: journeyTestAPI{}}
	result := service.runDesktopJourney(ctx, "smoke-1", "hello-desktop", "linux", recordingState{captureID: "rec", displayID: ":99", displayWidth: 1280, displayHeight: 720, windowManager: "openbox", titlebar: true})
	if result.Disposition == journeyPass {
		t.Fatal("cancelled journey must never pass")
	}
}

func TestDesktopJourney_XdotoolAbsentIsDegradedAndNeverPasses(t *testing.T) {
	detector := procmetrics.NewXdotoolDetector(func(context.Context, []string, string, ...string) ([]byte, error) {
		return nil, errors.New("command not found")
	}, slog.Default())
	service := &DefaultService{windowDetector: detector}
	result := service.runDesktopJourney(context.Background(), "smoke-1", "demo", "linux", recordingState{
		captureID:     "recording-1",
		displayID:     ":99",
		windowManager: "openbox",
		titlebar:      true,
	})

	if result.Disposition != journeyDegraded {
		t.Fatalf("disposition = %q, want degraded", result.Disposition)
	}
	if result.DegradedReason != "xdotool_unavailable" {
		t.Fatalf("degraded reason = %q, want xdotool_unavailable", result.DegradedReason)
	}
	if result.Disposition == journeyPass {
		t.Fatal("degraded journey must never pass")
	}
	if !result.RecordingStartedBeforeLaunch {
		t.Fatal("journey should retain recording-before-launch provenance")
	}
}

func TestDesktopJourney_CommunicationFixtureProvesProviderAndRoute(t *testing.T) {
	service := &DefaultService{
		journeyDriver:  &journeyTestDriver{geometry: &procmetrics.WindowGeometry{Width: 1280, Height: 720}},
		journeyClock:   RealClock{},
		journeyWaiter:  &journeyTestWaiter{},
		journeyCapture: &journeyTestCapture{},
		journeyAPI:     journeyTestAPI{},
	}
	result := service.runDesktopJourneyCapability(context.Background(), "smoke-1", "demo", "linux", recordingState{captureID: "rec", displayID: ":99", displayWidth: 1280, displayHeight: 720, windowManager: "openbox", titlebar: true}, "bundled.private.v1")
	if result.Disposition != JourneyDispositionPass {
		t.Fatalf("communication disposition = %q, want pass: %+v", result.Disposition, result)
	}
	if result.ProviderObservation == nil || result.ProviderObservation.ProviderTier != "managed-private" || result.ProviderObservation.SafeRouteClass != "private-bundle" {
		t.Fatalf("provider observation = %#v", result.ProviderObservation)
	}
	if result.Steps[1].AssertionStatus != JourneyStepPassed || result.Steps[2].AssertionStatus != JourneyStepPassed {
		t.Fatalf("provider/operation assertions = %#v/%#v", result.Steps[1], result.Steps[2])
	}
}

func TestDesktopJourney_CommunicationFixturesHaveDedicatedMachineContracts(t *testing.T) {
	tests := []struct {
		capability string
		provider   string
		route      string
		mode       string
		operation  string
	}{
		{capability: "bundled.private.v1", provider: "managed-private", route: "private-bundle", mode: "bundled-private", operation: "bundled-private"},
		{capability: "tier2.tier1.thin-client.v1", provider: "tier1-local-vrooli", route: "scenario-api-proxy", mode: "thin-client", operation: "thin-client"},
		{capability: "tier2.tier1.shared.v1", provider: "tier1-local-vrooli", route: "shared-resource", mode: "shared-resource", operation: "shared-resource"},
		{capability: "bundled.private.fallback.v1", provider: "managed-private", route: "private-bundle", mode: "private-fallback", operation: "private-fallback"},
	}
	for _, tc := range tests {
		t.Run(tc.capability, func(t *testing.T) {
			service := &DefaultService{
				journeyDriver:  &journeyTestDriver{geometry: &procmetrics.WindowGeometry{Width: 1280, Height: 720}},
				journeyClock:   RealClock{},
				journeyWaiter:  &journeyTestWaiter{},
				journeyCapture: &journeyTestCapture{},
				journeyAPI: communicationContractAPI{
					provider: tc.provider, route: tc.route, mode: tc.mode, operation: tc.operation,
				},
			}
			result := service.runDesktopJourneyCapability(context.Background(), "smoke-communication", "fixture", "linux", recordingState{
				captureID: "recording-1", displayID: ":99", displayWidth: 1280, displayHeight: 720, windowManager: "openbox", titlebar: true,
			}, tc.capability)
			if result.Disposition != JourneyDispositionPass {
				t.Fatalf("disposition = %q, want pass: %+v", result.Disposition, result)
			}
			if result.ProviderObservation == nil || result.ProviderObservation.ProviderTier != tc.provider || result.ProviderObservation.SafeRouteClass != tc.route {
				t.Fatalf("provider observation = %#v, want provider %q route %q", result.ProviderObservation, tc.provider, tc.route)
			}
			if len(result.Steps) != 4 || result.Steps[1].AssertionStatus != JourneyStepPassed || result.Steps[2].AssertionStatus != JourneyStepPassed {
				t.Fatalf("steps = %+v, want provider and operation assertions passed", result.Steps)
			}
		})
	}
}

func TestDesktopJourney_PeerCapabilityIsExplicitlyUnsupported(t *testing.T) {
	service := &DefaultService{journeyClock: RealClock{}}
	result := service.runDesktopJourneyCapability(context.Background(), "smoke-1", "demo", "linux", recordingState{}, "tier2.tier2.peer.v1")
	if result.Disposition != JourneyDispositionUnsupported || result.DegradedReason != "peer_protocol_not_implemented" {
		t.Fatalf("peer result = %+v", result)
	}
}

func TestJourneyPoliciesSerializeMilliseconds(t *testing.T) {
	data, err := json.Marshal(ReadinessPolicy{ID: "ready", Timeout: 12 * time.Second, PollInterval: 100 * time.Millisecond, StabilityCount: 2})
	if err != nil || !strings.Contains(string(data), `"timeout_ms":12000`) || strings.Contains(string(data), `12000000000`) {
		t.Fatalf("readiness policy JSON = %s, err=%v", data, err)
	}
	var decoded ReadinessPolicy
	if err := json.Unmarshal(data, &decoded); err != nil || decoded.Timeout != 12*time.Second || decoded.StabilityCount != 2 {
		t.Fatalf("decoded readiness policy = %#v, err=%v", decoded, err)
	}
}

func TestJourneyProfilesAreBoundedAndNamed(t *testing.T) {
	plan := helloDesktopFixture{}.Plan(JourneyInput{SmokeTestID: "smoke-1"})
	fast, err := applyJourneyProfile(plan, "fast-ci")
	if err != nil || fast.Profile != "fast-ci" || fast.Steps[0].Settle.Minimum != 100*time.Millisecond || fast.Steps[0].Readiness.Timeout != 3*time.Second {
		t.Fatalf("fast profile = %#v, err=%v", fast, err)
	}
	slow, err := applyJourneyProfile(plan, "diagnostic-slow")
	if err != nil || slow.Profile != "diagnostic-slow" || slow.Steps[0].Settle.Maximum != 4*time.Second || slow.Steps[0].Readiness.Timeout != 30*time.Second {
		t.Fatalf("slow profile = %#v, err=%v", slow, err)
	}
	if _, err := applyJourneyProfile(plan, "unbounded"); err == nil {
		t.Fatal("unknown profile must be rejected")
	}
}

func TestDefaultJourneyWaiterRequiresStableReadiness(t *testing.T) {
	waiter := defaultJourneyWaiter{clock: RealClock{}}
	policy := ReadinessPolicy{ID: "stable", Timeout: time.Second, PollInterval: time.Millisecond, StabilityCount: 2}
	attempts := 0
	result, err := waiter.WaitUntil(context.Background(), policy, func(context.Context) (bool, string, error) {
		attempts++
		return attempts >= 3, "window-ready", nil
	})
	if err != nil || result.Attempts < 4 || attempts < 4 {
		t.Fatalf("wait result = %#v, attempts=%d, err=%v; readiness must stabilize after a flap", result, attempts, err)
	}
}

func TestDefaultJourneyWaiterRejectsInvalidPolicies(t *testing.T) {
	waiter := defaultJourneyWaiter{clock: RealClock{}}
	if _, err := waiter.WaitUntil(context.Background(), ReadinessPolicy{ID: "invalid", Timeout: time.Second}, func(context.Context) (bool, string, error) { return true, "", nil }); err == nil {
		t.Fatal("readiness policy without poll interval must be rejected")
	}
	if err := waiter.Settle(context.Background(), SettlePolicy{ID: "invalid", Minimum: time.Second, Maximum: 500 * time.Millisecond}); err == nil {
		t.Fatal("settle policy with inverted bounds must be rejected")
	}
}

func TestValidateJourneyTimelineRejectsAmbiguousOrUnsafeRecords(t *testing.T) {
	now := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)
	valid := JourneyResult{
		SchemaVersion:   JourneySchemaVersion,
		EvidenceVersion: "journey-evidence.v2",
		SmokeTestID:     "smoke-1",
		ScenarioName:    "demo",
		Capability:      "hello-desktop",
		PlanID:          "hello-desktop.baseline.v2",
		CreatedAt:       now,
		CompletedAt:     now.Add(time.Second),
		Steps:           []JourneyStep{{ID: "launch", Name: "launch", Action: "window_activate", Disposition: JourneyStepPassed, StartedAt: now, CompletedAt: now.Add(500 * time.Millisecond), MonotonicStartMs: 0, MonotonicEndMs: 500}},
		Events:          []JourneyEvent{{Type: "step_completed", StartedAt: now, CompletedAt: now, MonotonicStartMs: 0, MonotonicEndMs: 0}},
	}
	if err := ValidateJourneyTimeline(valid); err != nil {
		t.Fatalf("valid timeline rejected: %v", err)
	}

	tests := map[string]func(*JourneyResult){
		"duplicate step IDs":      func(value *JourneyResult) { value.Steps = append(value.Steps, value.Steps[0]) },
		"negative video offset":   func(value *JourneyResult) { negative := int64(-1); value.Steps[0].VideoStartOffsetMs = &negative },
		"redaction leak":          func(value *JourneyResult) { value.Steps[0].ObservedState = "bearer token" },
		"event timestamp missing": func(value *JourneyResult) { value.Events[0].StartedAt = time.Time{} },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			value := valid
			value.Steps = append([]JourneyStep(nil), valid.Steps...)
			value.Events = append([]JourneyEvent(nil), valid.Events...)
			mutate(&value)
			if err := ValidateJourneyTimeline(value); err == nil {
				t.Fatalf("unsafe timeline unexpectedly accepted: %#v", value)
			}
		})
	}
}

func TestJourneyRequiresScreenshotBeforeAndAfterEveryInteraction(t *testing.T) {
	complete := []JourneyStep{{Action: "pointer_click", BeforeCaptureID: "before", AfterCaptureID: "after"}}
	if !journeyHasScreenshotPairs(complete) {
		t.Fatal("complete interaction screenshot pair should be accepted")
	}
	incomplete := []JourneyStep{{Action: "pointer_click", BeforeCaptureID: "before"}}
	if journeyHasScreenshotPairs(incomplete) {
		t.Fatal("missing after screenshot should be rejected")
	}
}
