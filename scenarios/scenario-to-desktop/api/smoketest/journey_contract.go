package smoketest

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
	"scenario-to-desktop-api/procmetrics"
)

// DesktopDriver is the consumer-owned seam for platform actions. The runner
// never imports or type-asserts the concrete xdotool implementation.
type DesktopDriver interface {
	IsAvailable(context.Context) bool
	LargestVisibleWindow(context.Context, string) (*procmetrics.WindowGeometry, error)
	WindowGeometry(context.Context, string) (*procmetrics.WindowGeometry, error)
	ActivateWindow(context.Context, string) error
	MaximizeWindow(context.Context, string, int, int) error
	ResizeWindow(context.Context, string, int, int) error
	MoveWindow(context.Context, string, int, int) error
	Click(context.Context, string, int, int, int) error
	KeyPress(context.Context, string, string) error
	Type(context.Context, string, string) error
}

type JourneyCapture interface {
	Capture(context.Context, string, string, string, string) (deliveryramp.EvidenceReference, error)
}

type JourneyAPIProbe interface {
	Greet(context.Context, string) (string, error)
}

// JourneyOperationProbe is the credential-free seam for communication and
// provider fixtures. Implementations return only safe route/provider metadata
// and a user-visible operation result.
type JourneyOperationProbe interface {
	Probe(context.Context, string) (JourneyOperationResult, error)
}

type JourneyOperationResult struct {
	Observed string                            `json:"observed"`
	Provider *deliveryramp.ProviderObservation `json:"provider,omitempty"`
	Route    string                            `json:"route,omitempty"`
}

type JourneyProcessObserver interface {
	Observe(context.Context) (string, error)
}

type JourneySupportStatus interface {
	Supported(JourneyInput) (bool, string)
}

type JourneyWaiter interface {
	WaitUntil(context.Context, deliveryramp.ReadinessPolicy, func(context.Context) (bool, string, error)) (WaitResult, error)
	Settle(context.Context, deliveryramp.SettlePolicy) error
}

type WaitResult struct {
	Observed string
	Attempts int
}

type JourneyObservation struct {
	Observed string
	Geometry *deliveryramp.Geometry
	Provider *deliveryramp.ProviderObservation
	Route    string
}

type JourneyAction func(context.Context, DesktopDriver, JourneyAPIProbe, JourneyInput) (JourneyObservation, error)

type JourneyFixture interface {
	Capability() string
	Plan(JourneyInput) deliveryramp.JourneyPlan
	Actions() map[string]JourneyAction
}

type JourneyInput struct {
	SmokeTestID   string
	ScenarioName  string
	Platform      string
	Display       string
	DisplayWidth  int
	DisplayHeight int
}

var journeyRegistry = struct {
	sync.RWMutex
	fixtures map[string]JourneyFixture
}{fixtures: make(map[string]JourneyFixture)}

func RegisterJourneyFixture(fixture JourneyFixture) error {
	if fixture == nil || strings.TrimSpace(fixture.Capability()) == "" {
		return fmt.Errorf("journey fixture and capability are required")
	}
	journeyRegistry.Lock()
	defer journeyRegistry.Unlock()
	key := strings.TrimSpace(fixture.Capability())
	if _, exists := journeyRegistry.fixtures[key]; exists {
		return fmt.Errorf("journey capability %q is already registered", key)
	}
	journeyRegistry.fixtures[key] = fixture
	return nil
}

func journeyFixture(capability string) (JourneyFixture, bool) {
	journeyRegistry.RLock()
	defer journeyRegistry.RUnlock()
	fixture, ok := journeyRegistry.fixtures[strings.TrimSpace(capability)]
	return fixture, ok
}

func capabilityForScenario(scenario string) string {
	scenario = strings.TrimSpace(scenario)
	if scenario == "browser-automation-studio" {
		return "monetization.trust-boundary.v1"
	}
	journeyRegistry.RLock()
	defer journeyRegistry.RUnlock()
	if _, ok := journeyRegistry.fixtures[scenario]; ok {
		return scenario
	}
	return "desktop.launch.baseline"
}

func defaultReadiness(id, reason string) deliveryramp.ReadinessPolicy {
	return deliveryramp.ReadinessPolicy{ID: id, Reason: reason, Timeout: 12 * time.Second, PollInterval: 100 * time.Millisecond, StabilityCount: 1, Cancellation: "context_or_timeout"}
}

func defaultSettle(id, reason string) deliveryramp.SettlePolicy {
	return deliveryramp.SettlePolicy{ID: id, Reason: reason, Minimum: 500 * time.Millisecond, Maximum: 2 * time.Second, PollInterval: 100 * time.Millisecond, Cancellation: "context_or_timeout"}
}

func applyJourneyProfile(plan deliveryramp.JourneyPlan, profile string) (deliveryramp.JourneyPlan, error) {
	profile = strings.TrimSpace(profile)
	if profile == "" || profile == plan.Profile {
		return plan, nil
	}
	if profile != "fast-ci" && profile != "diagnostic-slow" {
		return deliveryramp.JourneyPlan{}, fmt.Errorf("unsupported journey profile %q", profile)
	}
	steps := append([]deliveryramp.JourneyStepSpec(nil), plan.Steps...)
	for index := range steps {
		steps[index].Readiness = readinessForProfile(steps[index].Readiness, profile)
		steps[index].Settle = settleForProfile(steps[index].Settle, profile)
	}
	plan.Profile = profile
	plan.Steps = steps
	return plan, nil
}

func readinessForProfile(policy deliveryramp.ReadinessPolicy, profile string) deliveryramp.ReadinessPolicy {
	switch profile {
	case "fast-ci":
		policy.Timeout = minDuration(policy.Timeout, 3*time.Second)
		policy.PollInterval = minDuration(policy.PollInterval, 50*time.Millisecond)
	case "diagnostic-slow":
		policy.Timeout = maxDuration(policy.Timeout, 30*time.Second)
		policy.PollInterval = maxDuration(policy.PollInterval, 250*time.Millisecond)
	}
	return policy
}

func settleForProfile(policy deliveryramp.SettlePolicy, profile string) deliveryramp.SettlePolicy {
	switch profile {
	case "fast-ci":
		policy.Minimum = minDuration(policy.Minimum, 100*time.Millisecond)
		policy.Maximum = minDuration(policy.Maximum, time.Second)
		policy.PollInterval = minDuration(policy.PollInterval, 50*time.Millisecond)
	case "diagnostic-slow":
		policy.Minimum = maxDuration(policy.Minimum, time.Second)
		policy.Maximum = maxDuration(policy.Maximum, 4*time.Second)
		policy.PollInterval = maxDuration(policy.PollInterval, 250*time.Millisecond)
	}
	return policy
}

func minDuration(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}

func maxDuration(a, b time.Duration) time.Duration {
	if a > b {
		return a
	}
	return b
}

type helloDesktopFixture struct{}

func (helloDesktopFixture) Capability() string { return "hello-desktop" }

func (helloDesktopFixture) Plan(input JourneyInput) deliveryramp.JourneyPlan {
	semanticName := "Vrooli-" + strings.ReplaceAll(input.SmokeTestID, "-", "")
	if len(semanticName) > 24 {
		semanticName = semanticName[:24]
	}
	windowReady := defaultReadiness("target_window_visible", "wait for the usable application window")
	settle := defaultSettle("visual_settle", "allow the application surface to settle for human review")
	return deliveryramp.JourneyPlan{
		SchemaVersion: "2",
		ID:            "hello-desktop.baseline.v2",
		Capability:    "hello-desktop",
		Purpose:       "Prove the generated Hello Desktop application launches, responds semantically, supports desktop interaction, and shuts down cleanly.",
		Profile:       "normal-review",
		Steps: []deliveryramp.JourneyStepSpec{
			{ID: "activate", Purpose: "Bring the generated application window to the foreground.", Action: "window_activate", Capture: true, Readiness: windowReady, Settle: settle},
			{ID: "maximize", Purpose: "Prove the application can occupy the review display.", Action: "window_maximize", Capture: true, Readiness: windowReady, Settle: settle, Assertion: &deliveryramp.AssertionSpec{ID: "window.maximized", Expected: "window covers at least 90% of the display"}},
			{ID: "semantic_greet", Purpose: "Prove input reaches the application and the application API returns the expected greeting.", Action: "semantic_greet", Capture: true, Readiness: windowReady, Settle: settle, Assertion: &deliveryramp.AssertionSpec{ID: "hello-desktop.greeting", Expected: "Hello, " + semanticName + "!"}},
			{ID: "pointer_click", Purpose: "Prove pointer input reaches the generated application.", Action: "pointer_click", Capture: true, Readiness: windowReady, Settle: settle},
			{ID: "key_press", Purpose: "Prove keyboard input reaches the generated application.", Action: "key_press", Capture: true, Readiness: windowReady, Settle: settle},
			{ID: "resize", Purpose: "Prove the window can be resized without losing the application surface.", Action: "window_resize", Capture: true, Readiness: windowReady, Settle: settle},
			{ID: "move", Purpose: "Prove the window can be moved on the review display.", Action: "window_move", Capture: true, Readiness: windowReady, Settle: settle},
			{ID: "quit", Purpose: "Prove the application accepts a clean shutdown request.", Action: "quit_app", Capture: true, Readiness: windowReady, Settle: defaultSettle("shutdown_settle", "allow the process and window manager to settle after shutdown")},
		},
	}
}

func (helloDesktopFixture) Actions() map[string]JourneyAction {
	return map[string]JourneyAction{
		"semantic_greet": func(ctx context.Context, driver DesktopDriver, api JourneyAPIProbe, input JourneyInput) (JourneyObservation, error) {
			name := "Vrooli-" + strings.ReplaceAll(input.SmokeTestID, "-", "")
			if len(name) > 24 {
				name = name[:24]
			}
			for _, key := range []string{"Tab", "ctrl+a"} {
				if err := driver.KeyPress(ctx, input.Display, key); err != nil {
					return JourneyObservation{}, err
				}
			}
			if err := driver.Type(ctx, input.Display, name); err != nil {
				return JourneyObservation{}, err
			}
			for _, key := range []string{"Tab", "Return"} {
				if err := driver.KeyPress(ctx, input.Display, key); err != nil {
					return JourneyObservation{}, err
				}
			}
			if api == nil {
				return JourneyObservation{}, fmt.Errorf("semantic API probe is unavailable")
			}
			message, err := api.Greet(ctx, name)
			return JourneyObservation{Observed: message}, err
		},
	}
}

func init() {
	// Registration errors are programmer errors and must not be silently hidden.
	if err := RegisterJourneyFixture(helloDesktopFixture{}); err != nil {
		panic(err)
	}
	if err := RegisterJourneyFixture(baselineFixture{}); err != nil {
		panic(err)
	}
	registerCommunicationFixtures()
}

type baselineFixture struct{}

func (baselineFixture) Capability() string { return "desktop.launch.baseline" }
func (baselineFixture) Plan(JourneyInput) deliveryramp.JourneyPlan {
	windowReady := defaultReadiness("target_window_visible", "wait for the usable application window")
	settle := defaultSettle("visual_settle", "allow the application surface to settle for human review")
	return deliveryramp.JourneyPlan{
		SchemaVersion: "2",
		ID:            "desktop.launch.baseline.v2",
		Capability:    "desktop.launch.baseline",
		Purpose:       "Prove an arbitrary generated desktop application launches, accepts basic window/input actions, and shuts down cleanly.",
		Profile:       "normal-review",
		Steps: []deliveryramp.JourneyStepSpec{
			{ID: "activate", Purpose: "Bring the generated application window to the foreground.", Action: "window_activate", Capture: true, Readiness: windowReady, Settle: settle},
			{ID: "maximize", Purpose: "Prove the application can occupy the review display.", Action: "window_maximize", Capture: true, Readiness: windowReady, Settle: settle, Assertion: &deliveryramp.AssertionSpec{ID: "window.maximized", Expected: "window covers at least 90% of the display"}},
			{ID: "pointer_click", Purpose: "Prove pointer input reaches the generated application.", Action: "pointer_click", Capture: true, Readiness: windowReady, Settle: settle},
			{ID: "key_press", Purpose: "Prove keyboard input reaches the generated application.", Action: "key_press", Capture: true, Readiness: windowReady, Settle: settle},
			{ID: "resize", Purpose: "Prove the window can be resized without losing the application surface.", Action: "window_resize", Capture: true, Readiness: windowReady, Settle: settle},
			{ID: "move", Purpose: "Prove the window can be moved on the review display.", Action: "window_move", Capture: true, Readiness: windowReady, Settle: settle},
			{ID: "quit", Purpose: "Prove the application accepts a clean shutdown request.", Action: "quit_app", Capture: true, Readiness: windowReady, Settle: defaultSettle("shutdown_settle", "allow the process and window manager to settle after shutdown")},
		},
	}
}
func (baselineFixture) Actions() map[string]JourneyAction { return nil }

type communicationFixture struct {
	capability string
	mode       string
	tier       string
	route      string
	operation  string
}

func (f communicationFixture) Capability() string { return f.capability }

func (f communicationFixture) Plan(JourneyInput) deliveryramp.JourneyPlan {
	return deliveryramp.JourneyPlan{
		SchemaVersion: "2",
		ID:            f.capability + ".v1",
		Capability:    f.capability,
		Purpose:       "Prove the selected desktop communication and resource-provider path with a credential-free operation result.",
		Profile:       "normal-review",
		Steps: []deliveryramp.JourneyStepSpec{
			{ID: "activate", Purpose: "Bring the generated application window to the foreground before communication review.", Action: "window_activate", Capture: true, Readiness: defaultReadiness("target_window_visible", "wait for the usable application window"), Settle: defaultSettle("visual_settle", "allow the application surface to settle for human review")},
			{ID: "provider_observation", Purpose: "Show which provider tier and safe route were selected.", Action: "provider_observation", Capture: true, Readiness: defaultReadiness("provider_ready", "wait for the selected provider to become ready"), Settle: defaultSettle("provider_settle", "allow provider metadata to become visible"), Assertion: &deliveryramp.AssertionSpec{ID: "provider.route", Expected: "provider=" + f.tier + ";route=" + f.route}},
			{ID: "communication_operation", Purpose: "Exercise the capability operation through the selected route.", Action: "communication_operation", Capture: true, Readiness: defaultReadiness("operation_ready", "wait for the communication operation to be ready"), Settle: defaultSettle("operation_settle", "allow the operation result to render"), Assertion: &deliveryramp.AssertionSpec{ID: "communication.operation", Expected: "operation=" + f.operation + ";mode=" + f.mode}},
			{ID: "quit", Purpose: "Close the application after the provider and route assertions complete.", Action: "quit_app", Capture: true, Readiness: defaultReadiness("target_window_visible", "wait for the application window before shutdown"), Settle: defaultSettle("shutdown_settle", "allow the process and window manager to settle after shutdown")},
		},
	}
}

func (f communicationFixture) Actions() map[string]JourneyAction {
	return map[string]JourneyAction{
		"provider_observation":    operationJourneyAction("provider_observation"),
		"communication_operation": operationJourneyAction("communication_operation"),
	}
}

func operationJourneyAction(operation string) JourneyAction {
	return func(ctx context.Context, _ DesktopDriver, api JourneyAPIProbe, _ JourneyInput) (JourneyObservation, error) {
		probe, ok := api.(JourneyOperationProbe)
		if !ok {
			return JourneyObservation{}, fmt.Errorf("journey operation probe is unavailable for %s", operation)
		}
		result, err := probe.Probe(ctx, operation)
		if err != nil {
			return JourneyObservation{}, err
		}
		return JourneyObservation{Observed: result.Observed, Provider: result.Provider, Route: result.Route}, nil
	}
}

type unsupportedPeerFixture struct{}

func (unsupportedPeerFixture) Capability() string { return "tier2.tier2.peer.v1" }
func (unsupportedPeerFixture) Supported(JourneyInput) (bool, string) {
	return false, "peer_protocol_not_implemented"
}

func (unsupportedPeerFixture) Plan(JourneyInput) deliveryramp.JourneyPlan {
	return deliveryramp.JourneyPlan{SchemaVersion: "2", ID: "tier2.tier2.peer.v1", Capability: "tier2.tier2.peer.v1", Purpose: "Tier 2 to Tier 2 communication is gated on an implemented authenticated peer protocol.", Profile: "diagnostic"}
}
func (unsupportedPeerFixture) Actions() map[string]JourneyAction { return nil }

func registerCommunicationFixtures() {
	fixtures := []JourneyFixture{
		communicationFixture{capability: "bundled.private.v1", mode: "bundled-private", tier: "managed-private", route: "private-bundle", operation: "bundled-private"},
		communicationFixture{capability: "tier2.tier1.thin-client.v1", mode: "thin-client", tier: "tier1-local-vrooli", route: "scenario-api-proxy", operation: "thin-client"},
		communicationFixture{capability: "tier2.tier1.shared.v1", mode: "shared-resource", tier: "tier1-local-vrooli", route: "shared-resource", operation: "shared-resource"},
		communicationFixture{capability: "bundled.private.fallback.v1", mode: "private-fallback", tier: "managed-private", route: "private-bundle", operation: "private-fallback"},
		unsupportedPeerFixture{},
		monetizationBoundaryFixture{},
	}
	for _, fixture := range fixtures {
		if err := RegisterJourneyFixture(fixture); err != nil {
			panic(err)
		}
	}
}

// monetizationBoundaryFixture is deliberately provider-backed rather than a
// local mock: its probe talks to the bundled scenario API, which patches the
// local lease view and then exercises the real LPBS Class A boundary.
type monetizationBoundaryFixture struct{}

func (monetizationBoundaryFixture) Capability() string { return "monetization.trust-boundary.v1" }

func (monetizationBoundaryFixture) Plan(JourneyInput) deliveryramp.JourneyPlan {
	windowReady := defaultReadiness("target_window_visible", "wait for the usable application window")
	settle := defaultSettle("visual_settle", "allow the application surface to settle")
	return deliveryramp.JourneyPlan{
		SchemaVersion: "2",
		ID:            "monetization.trust-boundary.v1",
		Capability:    "monetization.trust-boundary.v1",
		Purpose:       "Prove a patched local entitlement cannot authorize a Class A operation while a Class B operation remains usable.",
		Profile:       "normal-review",
		Steps: []deliveryramp.JourneyStepSpec{
			{ID: "activate", Purpose: "Bring the bundled application into view before trust-boundary checks.", Action: "window_activate", Capture: true, Readiness: windowReady, Settle: settle},
			{ID: "tampered_class_a", Purpose: "Patch the local entitlement view and call the real Class A authority.", Action: "tampered_class_a", Capture: true, Readiness: windowReady, Settle: settle, Assertion: &deliveryramp.AssertionSpec{ID: "monetization.class_a.refused", Expected: "class_a=refused"}},
			{ID: "class_b_local", Purpose: "Run a local-capacity Class B operation under the signed plan lease.", Action: "class_b_local", Capture: true, Readiness: windowReady, Settle: settle, Assertion: &deliveryramp.AssertionSpec{ID: "monetization.class_b.allowed", Expected: "class_b=allowed"}},
			{ID: "quit", Purpose: "Close the bundled application after the boundary assertions.", Action: "quit_app", Capture: true, Readiness: windowReady, Settle: defaultSettle("shutdown_settle", "allow shutdown to settle")},
		},
	}
}

func (monetizationBoundaryFixture) Actions() map[string]JourneyAction {
	return map[string]JourneyAction{
		"tampered_class_a": operationJourneyAction("tampered_class_a"),
		"class_b_local":    operationJourneyAction("class_b_local"),
	}
}
