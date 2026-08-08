package smoketest

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"scenario-to-desktop-api/procmetrics"
)

// JourneySchemaVersion is the version of the persisted journey sidecar. New
// fields are additive; readers must reject versions they cannot interpret.
const JourneySchemaVersion = 2

const (
	JourneyDispositionPass        = "pass"
	JourneyDispositionFailed      = "failed"
	JourneyDispositionDegraded    = "degraded"
	JourneyDispositionUnavailable = "unavailable"
	JourneyDispositionUnsupported = "unsupported"
	JourneyDispositionNotRun      = "not_run"

	JourneyStepPassed      = "passed"
	JourneyStepFailed      = "failed"
	JourneyStepDegraded    = "degraded"
	JourneyStepUnavailable = "unavailable"
	JourneyStepNotRun      = "not_run"
)

// ReadinessPolicy describes a bounded condition wait. A policy is part of the
// evidence contract so a reviewer can distinguish waiting for a signal from an
// unexplained idle period.
type ReadinessPolicy struct {
	ID             string        `json:"id"`
	Reason         string        `json:"reason"`
	Timeout        time.Duration `json:"timeout_ms"`
	PollInterval   time.Duration `json:"poll_interval_ms"`
	StabilityCount int           `json:"stability_count"`
	Cancellation   string        `json:"cancellation"`
}

func (p ReadinessPolicy) MarshalJSON() ([]byte, error) {
	type wire struct {
		ID             string `json:"id"`
		Reason         string `json:"reason"`
		TimeoutMs      int64  `json:"timeout_ms"`
		PollIntervalMs int64  `json:"poll_interval_ms"`
		StabilityCount int    `json:"stability_count"`
		Cancellation   string `json:"cancellation"`
	}
	return json.Marshal(wire{ID: p.ID, Reason: p.Reason, TimeoutMs: p.Timeout.Milliseconds(), PollIntervalMs: p.PollInterval.Milliseconds(), StabilityCount: p.StabilityCount, Cancellation: p.Cancellation})
}

func (p *ReadinessPolicy) UnmarshalJSON(data []byte) error {
	type wire struct {
		ID             string `json:"id"`
		Reason         string `json:"reason"`
		TimeoutMs      int64  `json:"timeout_ms"`
		PollIntervalMs int64  `json:"poll_interval_ms"`
		StabilityCount int    `json:"stability_count"`
		Cancellation   string `json:"cancellation"`
	}
	var value wire
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	p.ID, p.Reason, p.Timeout, p.PollInterval, p.StabilityCount, p.Cancellation = value.ID, value.Reason, time.Duration(value.TimeoutMs)*time.Millisecond, time.Duration(value.PollIntervalMs)*time.Millisecond, value.StabilityCount, value.Cancellation
	return nil
}

// SettlePolicy records an intentional visual settle period after an action.
type SettlePolicy struct {
	ID           string        `json:"id"`
	Reason       string        `json:"reason"`
	Minimum      time.Duration `json:"minimum_ms"`
	Maximum      time.Duration `json:"maximum_ms"`
	PollInterval time.Duration `json:"poll_interval_ms"`
	Cancellation string        `json:"cancellation"`
}

func (p SettlePolicy) MarshalJSON() ([]byte, error) {
	type wire struct {
		ID           string `json:"id"`
		Reason       string `json:"reason"`
		MinimumMs    int64  `json:"minimum_ms"`
		MaximumMs    int64  `json:"maximum_ms"`
		PollInterval int64  `json:"poll_interval_ms"`
		Cancellation string `json:"cancellation"`
	}
	return json.Marshal(wire{ID: p.ID, Reason: p.Reason, MinimumMs: p.Minimum.Milliseconds(), MaximumMs: p.Maximum.Milliseconds(), PollInterval: p.PollInterval.Milliseconds(), Cancellation: p.Cancellation})
}

func (p *SettlePolicy) UnmarshalJSON(data []byte) error {
	type wire struct {
		ID           string `json:"id"`
		Reason       string `json:"reason"`
		MinimumMs    int64  `json:"minimum_ms"`
		MaximumMs    int64  `json:"maximum_ms"`
		PollInterval int64  `json:"poll_interval_ms"`
		Cancellation string `json:"cancellation"`
	}
	var value wire
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	p.ID, p.Reason, p.Minimum, p.Maximum, p.PollInterval, p.Cancellation = value.ID, value.Reason, time.Duration(value.MinimumMs)*time.Millisecond, time.Duration(value.MaximumMs)*time.Millisecond, time.Duration(value.PollInterval)*time.Millisecond, value.Cancellation
	return nil
}

type AssertionSpec struct {
	ID          string `json:"id"`
	Expected    string `json:"expected"`
	Description string `json:"description,omitempty"`
}

// JourneyStepSpec is serializable plan data. Action execution is deliberately
// kept in the fixture registry, not embedded in the persisted plan.
type JourneyStepSpec struct {
	ID        string          `json:"id"`
	Purpose   string          `json:"purpose"`
	Action    string          `json:"action"`
	Capture   bool            `json:"capture"`
	Readiness ReadinessPolicy `json:"readiness"`
	Settle    SettlePolicy    `json:"settle"`
	Assertion *AssertionSpec  `json:"assertion,omitempty"`
}

type JourneyPlan struct {
	SchemaVersion string            `json:"schema_version"`
	ID            string            `json:"id"`
	Capability    string            `json:"capability"`
	Purpose       string            `json:"purpose"`
	Profile       string            `json:"profile"`
	Steps         []JourneyStepSpec `json:"steps"`
}

type JourneyEvent struct {
	Type             string    `json:"type"`
	StepID           string    `json:"step_id,omitempty"`
	PolicyID         string    `json:"policy_id,omitempty"`
	Observed         string    `json:"observed,omitempty"`
	StartedAt        time.Time `json:"started_at"`
	CompletedAt      time.Time `json:"completed_at"`
	MonotonicStartMs int64     `json:"monotonic_start_ms"`
	MonotonicEndMs   int64     `json:"monotonic_end_ms"`
	Reason           string    `json:"reason,omitempty"`
}

type EvidenceReference struct {
	ID        string `json:"id"`
	Kind      string `json:"kind"`
	URI       string `json:"uri,omitempty"`
	MediaType string `json:"media_type,omitempty"`
	Checksum  string `json:"checksum,omitempty"`
	Redacted  bool   `json:"redacted,omitempty"`
}

// WorkflowExecutionReference is the provider-neutral handoff for a semantic
// scenario validation. The provider owns the execution and its artifacts; the
// desktop producer only records the identities needed to bind both layers to
// one validation cell.
type WorkflowExecutionReference struct {
	Provider       string              `json:"provider"`
	AssetID        string              `json:"asset_id"`
	ExecutionID    string              `json:"execution_id"`
	RunID          string              `json:"run_id"`
	ArtifactDigest string              `json:"artifact_digest"`
	TargetID       string              `json:"target_id"`
	CellID         string              `json:"cell_id"`
	Disposition    string              `json:"disposition"`
	Artifacts      []EvidenceReference `json:"artifacts,omitempty"`
}

// ValidateLink verifies that provider-owned evidence is bound to the same
// durable validation identity as the desktop evidence. A passing workflow
// must expose at least one checksummed artifact reference.
func (r WorkflowExecutionReference) ValidateLink(runID, artifactDigest, targetID, cellID string) error {
	for name, value := range map[string]string{
		"provider": r.Provider, "asset_id": r.AssetID, "execution_id": r.ExecutionID,
		"run_id": r.RunID, "artifact_digest": r.ArtifactDigest, "target_id": r.TargetID,
		"cell_id": r.CellID, "disposition": r.Disposition,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("workflow reference %s is required", name)
		}
	}
	if r.RunID != runID || r.ArtifactDigest != artifactDigest || r.TargetID != targetID || r.CellID != cellID {
		return fmt.Errorf("workflow reference is not bound to the desktop validation identity")
	}
	if strings.EqualFold(r.Disposition, JourneyDispositionPass) {
		if len(r.Artifacts) == 0 {
			return fmt.Errorf("passing workflow reference requires artifacts")
		}
		for index, artifact := range r.Artifacts {
			if strings.TrimSpace(artifact.ID) == "" || strings.TrimSpace(artifact.Kind) == "" || strings.TrimSpace(artifact.URI) == "" || strings.TrimSpace(artifact.Checksum) == "" || !artifact.Redacted {
				return fmt.Errorf("workflow artifact %d is missing identity or checksum", index)
			}
		}
	}
	return nil
}

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
	Capture(context.Context, string, string, string, string) (EvidenceReference, error)
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
	Observed string                      `json:"observed"`
	Provider *JourneyProviderObservation `json:"provider,omitempty"`
	Route    string                      `json:"route,omitempty"`
}

type JourneyProviderObservation struct {
	DeploymentMode   string `json:"deployment_mode"`
	ProviderTier     string `json:"provider_tier"`
	ServiceIdentity  string `json:"service_identity"`
	ArtifactDigest   string `json:"artifact_digest,omitempty"`
	Readiness        string `json:"readiness"`
	FallbackDecision string `json:"fallback_decision,omitempty"`
	SafeRouteClass   string `json:"safe_route_class"`
	LeaseExpiresAt   string `json:"lease_expires_at,omitempty"`
}

type JourneyProcessObserver interface {
	Observe(context.Context) (string, error)
}

type JourneySupportStatus interface {
	Supported(JourneyInput) (bool, string)
}

type JourneyWaiter interface {
	WaitUntil(context.Context, ReadinessPolicy, func(context.Context) (bool, string, error)) (WaitResult, error)
	Settle(context.Context, SettlePolicy) error
}

type WaitResult struct {
	Observed string
	Attempts int
}

type JourneyObservation struct {
	Observed string
	Geometry *procmetrics.WindowGeometry
	Provider *JourneyProviderObservation
	Route    string
}

type JourneyAction func(context.Context, DesktopDriver, JourneyAPIProbe, JourneyInput) (JourneyObservation, error)

type JourneyFixture interface {
	Capability() string
	Plan(JourneyInput) JourneyPlan
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
	journeyRegistry.RLock()
	defer journeyRegistry.RUnlock()
	if _, ok := journeyRegistry.fixtures[strings.TrimSpace(scenario)]; ok {
		return strings.TrimSpace(scenario)
	}
	return "desktop.launch.baseline"
}

func defaultReadiness(id, reason string) ReadinessPolicy {
	return ReadinessPolicy{ID: id, Reason: reason, Timeout: 12 * time.Second, PollInterval: 100 * time.Millisecond, StabilityCount: 1, Cancellation: "context_or_timeout"}
}

func defaultSettle(id, reason string) SettlePolicy {
	return SettlePolicy{ID: id, Reason: reason, Minimum: 500 * time.Millisecond, Maximum: 2 * time.Second, PollInterval: 100 * time.Millisecond, Cancellation: "context_or_timeout"}
}

func applyJourneyProfile(plan JourneyPlan, profile string) (JourneyPlan, error) {
	profile = strings.TrimSpace(profile)
	if profile == "" || profile == plan.Profile {
		return plan, nil
	}
	if profile != "fast-ci" && profile != "diagnostic-slow" {
		return JourneyPlan{}, fmt.Errorf("unsupported journey profile %q", profile)
	}
	steps := append([]JourneyStepSpec(nil), plan.Steps...)
	for index := range steps {
		steps[index].Readiness = plan.ReadinessForProfile(steps[index].Readiness, profile)
		steps[index].Settle = settleForProfile(steps[index].Settle, profile)
	}
	plan.Profile = profile
	plan.Steps = steps
	return plan, nil
}

func (p JourneyPlan) ReadinessForProfile(policy ReadinessPolicy, profile string) ReadinessPolicy {
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

func settleForProfile(policy SettlePolicy, profile string) SettlePolicy {
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

func (helloDesktopFixture) Plan(input JourneyInput) JourneyPlan {
	semanticName := "Vrooli-" + strings.ReplaceAll(input.SmokeTestID, "-", "")
	if len(semanticName) > 24 {
		semanticName = semanticName[:24]
	}
	windowReady := defaultReadiness("target_window_visible", "wait for the usable application window")
	settle := defaultSettle("visual_settle", "allow the application surface to settle for human review")
	return JourneyPlan{
		SchemaVersion: "2",
		ID:            "hello-desktop.baseline.v2",
		Capability:    "hello-desktop",
		Purpose:       "Prove the generated Hello Desktop application launches, responds semantically, supports desktop interaction, and shuts down cleanly.",
		Profile:       "normal-review",
		Steps: []JourneyStepSpec{
			{ID: "activate", Purpose: "Bring the generated application window to the foreground.", Action: "window_activate", Capture: true, Readiness: windowReady, Settle: settle},
			{ID: "maximize", Purpose: "Prove the application can occupy the review display.", Action: "window_maximize", Capture: true, Readiness: windowReady, Settle: settle, Assertion: &AssertionSpec{ID: "window.maximized", Expected: "window covers at least 90% of the display"}},
			{ID: "semantic_greet", Purpose: "Prove input reaches the application and the application API returns the expected greeting.", Action: "semantic_greet", Capture: true, Readiness: windowReady, Settle: settle, Assertion: &AssertionSpec{ID: "hello-desktop.greeting", Expected: "Hello, " + semanticName + "!"}},
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
func (baselineFixture) Plan(JourneyInput) JourneyPlan {
	windowReady := defaultReadiness("target_window_visible", "wait for the usable application window")
	settle := defaultSettle("visual_settle", "allow the application surface to settle for human review")
	return JourneyPlan{
		SchemaVersion: "2",
		ID:            "desktop.launch.baseline.v2",
		Capability:    "desktop.launch.baseline",
		Purpose:       "Prove an arbitrary generated desktop application launches, accepts basic window/input actions, and shuts down cleanly.",
		Profile:       "normal-review",
		Steps: []JourneyStepSpec{
			{ID: "activate", Purpose: "Bring the generated application window to the foreground.", Action: "window_activate", Capture: true, Readiness: windowReady, Settle: settle},
			{ID: "maximize", Purpose: "Prove the application can occupy the review display.", Action: "window_maximize", Capture: true, Readiness: windowReady, Settle: settle, Assertion: &AssertionSpec{ID: "window.maximized", Expected: "window covers at least 90% of the display"}},
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

func (f communicationFixture) Plan(JourneyInput) JourneyPlan {
	return JourneyPlan{
		SchemaVersion: "2",
		ID:            f.capability + ".v1",
		Capability:    f.capability,
		Purpose:       "Prove the selected desktop communication and resource-provider path with a credential-free operation result.",
		Profile:       "normal-review",
		Steps: []JourneyStepSpec{
			{ID: "activate", Purpose: "Bring the generated application window to the foreground before communication review.", Action: "window_activate", Capture: true, Readiness: defaultReadiness("target_window_visible", "wait for the usable application window"), Settle: defaultSettle("visual_settle", "allow the application surface to settle for human review")},
			{ID: "provider_observation", Purpose: "Show which provider tier and safe route were selected.", Action: "provider_observation", Capture: true, Readiness: defaultReadiness("provider_ready", "wait for the selected provider to become ready"), Settle: defaultSettle("provider_settle", "allow provider metadata to become visible"), Assertion: &AssertionSpec{ID: "provider.route", Expected: "provider=" + f.tier + ";route=" + f.route}},
			{ID: "communication_operation", Purpose: "Exercise the capability operation through the selected route.", Action: "communication_operation", Capture: true, Readiness: defaultReadiness("operation_ready", "wait for the communication operation to be ready"), Settle: defaultSettle("operation_settle", "allow the operation result to render"), Assertion: &AssertionSpec{ID: "communication.operation", Expected: "operation=" + f.operation + ";mode=" + f.mode}},
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

func (unsupportedPeerFixture) Plan(JourneyInput) JourneyPlan {
	return JourneyPlan{SchemaVersion: "2", ID: "tier2.tier2.peer.v1", Capability: "tier2.tier2.peer.v1", Purpose: "Tier 2 to Tier 2 communication is gated on an implemented authenticated peer protocol.", Profile: "diagnostic"}
}
func (unsupportedPeerFixture) Actions() map[string]JourneyAction { return nil }

func registerCommunicationFixtures() {
	fixtures := []JourneyFixture{
		communicationFixture{capability: "bundled.private.v1", mode: "bundled-private", tier: "managed-private", route: "private-bundle", operation: "bundled-private"},
		communicationFixture{capability: "tier2.tier1.thin-client.v1", mode: "thin-client", tier: "tier1-local-vrooli", route: "scenario-api-proxy", operation: "thin-client"},
		communicationFixture{capability: "tier2.tier1.shared.v1", mode: "shared-resource", tier: "tier1-local-vrooli", route: "shared-resource", operation: "shared-resource"},
		communicationFixture{capability: "bundled.private.fallback.v1", mode: "private-fallback", tier: "managed-private", route: "private-bundle", operation: "private-fallback"},
		unsupportedPeerFixture{},
	}
	for _, fixture := range fixtures {
		if err := RegisterJourneyFixture(fixture); err != nil {
			panic(err)
		}
	}
}
