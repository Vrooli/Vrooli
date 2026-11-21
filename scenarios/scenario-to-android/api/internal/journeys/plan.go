// Package conformance declares the generated-app Android behavior contract.
// It contains no device implementation: native steps name device-control
// verbs and web steps name Browser Automation Studio flows.
package journeys

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
)

const (
	PlanID   = "android-generated-app-conformance-v1"
	Fixture  = "hello-mobile"
	Platform = "android"

	TemporaryAppTestPlanID     = "android-scenario-app-temporary-test-v1"
	DurableAppPlanID           = "android-scenario-app-durable-v1"
	UpdateAppPlanID            = "android-scenario-app-update-v1"
	VerifyAppPlanID            = "android-scenario-app-verify-v1"
	OfflineTransitionPlanID    = "android-scenario-app-offline-transition-v1"
	UpdatePreservationFlowID   = "scenario-app-update-preservation"
	UpdatePreservationFlowPath = "bas/cases/mobile/update-preservation-flow.json"
)

type ReadinessPolicy struct {
	ID             string `json:"id"`
	Reason         string `json:"reason"`
	TimeoutMS      int64  `json:"timeout_ms"`
	PollIntervalMS int64  `json:"poll_interval_ms"`
	StabilityCount int    `json:"stability_count"`
	Cancellation   string `json:"cancellation"`
}

type SettlePolicy struct {
	ID           string `json:"id"`
	Reason       string `json:"reason"`
	MinimumMS    int64  `json:"minimum_ms"`
	MaximumMS    int64  `json:"maximum_ms"`
	PollInterval int64  `json:"poll_interval_ms"`
	Cancellation string `json:"cancellation"`
}

type Step struct {
	ID                   string         `json:"id"`
	Kind                 string         `json:"kind"`
	Reference            string         `json:"reference,omitempty"`
	Target               string         `json:"target,omitempty"`
	TimeoutMS            int64          `json:"timeout_ms"`
	RequiredCapabilities []string       `json:"required_capabilities,omitempty"`
	Arguments            map[string]any `json:"arguments,omitempty"`
}

type Chapter struct {
	ID                    string          `json:"id"`
	Purpose               string          `json:"purpose"`
	Expected              string          `json:"expected"`
	MinimumAndroidVersion string          `json:"minimum_android_version,omitempty"`
	RequiredCapabilities  []string        `json:"required_capabilities"`
	Readiness             ReadinessPolicy `json:"readiness"`
	Settle                SettlePolicy    `json:"settle"`
	Steps                 []Step          `json:"steps"`
}

type Plan struct {
	SchemaVersion string    `json:"schema_version"`
	ID            string    `json:"id"`
	Platform      string    `json:"platform"`
	Fixture       string    `json:"fixture"`
	Chapters      []Chapter `json:"chapters"`
}

// JourneyPlan converts the published chapter contract into the provider-neutral
// execution shape while retaining chapter and step identity in every action.
func (p Plan) JourneyPlan() deliveryramp.JourneyPlan {
	steps := make([]deliveryramp.JourneyStepSpec, 0)
	for _, chapter := range p.Chapters {
		for stepIndex, declared := range chapter.Steps {
			action := declared.Reference
			args := map[string]string{"chapter_id": chapter.ID, "target": declared.Target, "reference": declared.Reference}
			if declared.TimeoutMS > 0 {
				args["timeout_ms"] = strconv.FormatInt(declared.TimeoutMS, 10)
			}
			if len(declared.RequiredCapabilities) > 0 {
				args["required_capabilities"] = strings.Join(declared.RequiredCapabilities, ",")
			}
			if declared.Kind == "bas" {
				action = "bas-flow"
				args["flow_reference"] = declared.Target
			} else if declared.Kind == "skip" {
				action = "not-applicable"
				args["reason"] = declared.Target
			}
			for key, value := range declared.Arguments {
				if text, ok := value.(string); ok {
					args[key] = text
				}
			}
			var assertion *deliveryramp.AssertionSpec
			if stepIndex == len(chapter.Steps)-1 && declared.Kind != "skip" {
				assertion = &deliveryramp.AssertionSpec{ID: chapter.ID + "/expected", Expected: chapter.Expected, Description: chapter.Purpose}
			}
			steps = append(steps, deliveryramp.JourneyStepSpec{
				ID: chapter.ID + "/" + declared.ID, Purpose: chapter.Purpose, Action: action, Arguments: args, Capture: true,
				Readiness: deliveryramp.ReadinessPolicy{ID: chapter.Readiness.ID, Reason: chapter.Readiness.Reason, Timeout: time.Duration(chapter.Readiness.TimeoutMS) * time.Millisecond, PollInterval: time.Duration(chapter.Readiness.PollIntervalMS) * time.Millisecond, StabilityCount: chapter.Readiness.StabilityCount, Cancellation: chapter.Readiness.Cancellation},
				Settle:    deliveryramp.SettlePolicy{ID: chapter.Settle.ID, Reason: chapter.Settle.Reason, Minimum: time.Duration(chapter.Settle.MinimumMS) * time.Millisecond, Maximum: time.Duration(chapter.Settle.MaximumMS) * time.Millisecond, PollInterval: time.Duration(chapter.Settle.PollInterval) * time.Millisecond, Cancellation: chapter.Settle.Cancellation},
				Assertion: assertion,
			})
		}
	}
	return deliveryramp.JourneyPlan{SchemaVersion: deliveryramp.JourneyEvidenceVersion, ID: p.ID, Capability: "generated-app", Purpose: "Android generated-app conformance", Profile: "normal", Steps: steps}
}

func AndroidPlan() Plan {
	chapters := []Chapter{
		chapter("install_cold_start", "Install and launch a generated app from a clean state.", "the first usable frame and app title are visible", []string{"app-lifecycle", "screenshot"}, step("package-preflight", "device", "package-state", "absent", "app-lifecycle"), step("wake", "device", "screen", "wake", "input"), step("unlock", "device", "screen", "unlock", "input"), step("install", "device", "install", "", "app-lifecycle"), step("launch", "device", "launch", "", "app-lifecycle"), step("observe", "device", "observe", "", "screenshot"), basStep("web-flow", "hello-mobile-smoke")),
		{ID: "permission_deny_grace", Purpose: "Deny the notification runtime permission and relaunch.", Expected: "the app remains usable after denial", MinimumAndroidVersion: "13", RequiredCapabilities: []string{"permissions", "app-lifecycle", "screenshot"}, Steps: []Step{step("deny", "device", "revoke-permission", "android.permission.POST_NOTIFICATIONS", "permissions"), step("launch", "device", "launch", "", "app-lifecycle"), basStep("web-flow", "hello-mobile-smoke")}},
		chapter("background_resume", "Background and resume the generated app.", "the app returns with its state intact", []string{"input", "app-lifecycle", "screenshot"}, step("background", "device", "key", "KEYCODE_HOME", "input"), step("resume", "device", "launch", "", "app-lifecycle"), basStep("web-flow", "hello-mobile-smoke")),
		chapter("process_death_restore", "Force-stop and relaunch the generated app.", "persisted state is restored after process death", []string{"app-lifecycle", "screenshot"}, step("stop", "device", "stop", "", "app-lifecycle"), step("launch", "device", "launch", "", "app-lifecycle"), basStep("web-flow", "hello-mobile-smoke")),
		chapter("rotation_size_class", "Rotate the target and inspect the responsive layout.", "the input remains usable after rotation", []string{"orientation", "screenshot"}, step("rotate", "device", "rotate", "landscape", "orientation"), basStep("web-flow", "hello-mobile-smoke")),
		chapter("keyboard_avoidance", "Focus the fixture input and inspect keyboard avoidance.", "the focused input remains visible", []string{"semantic-tree", "screenshot"}, basStep("web-flow", "hello-mobile-smoke")),
		chapter("offline_transition", "Toggle connectivity and inspect both terminal states.", "offline and online state are explicit", []string{"network-control", "screenshot"}, step("offline", "device", "network", "offline", "network-control"), basStep("web-flow-offline", "hello-mobile-smoke"), step("online", "device", "network", "online", "network-control"), basStep("web-flow-online", "hello-mobile-smoke")),
		chapter("deep_link", "Open the generated app through its canonical deep link.", "the home route is rendered", []string{"app-lifecycle", "screenshot"}, step("open", "device", "deep-link", "vrooli-hello://home", "app-lifecycle"), basStep("web-flow", "hello-mobile-smoke")),
		chapter("notification_tap", "Resolve the fixture notification target.", "the notification action reaches the app", []string{"semantic-tree", "screenshot"}, basStep("web-flow", "hello-mobile-smoke")),
		chapter("back_navigation", "Use native back navigation from a fixture route.", "back returns to the home route", []string{"input", "screenshot"}, step("back", "device", "key", "KEYCODE_BACK", "input"), basStep("web-flow", "hello-mobile-smoke")),
		chapter("update_migration", "Install an update over existing app data.", "app-owned local storage survives the update", []string{"app-lifecycle", "screenshot"}, step("update", "device", "install", "", "app-lifecycle"), step("launch", "device", "launch", "", "app-lifecycle"), basStepWithFlowPath("web-flow", "scenario-app-update-preservation", "bas/cases/mobile/update-preservation-flow.json")),
		chapter("clean_uninstall", "Uninstall the generated app and verify package absence.", "the app package is absent", []string{"app-lifecycle", "screenshot"}, step("uninstall", "device", "uninstall", "", "app-lifecycle"), step("assert-absent", "device", "package-state", "absent", "app-lifecycle")),
	}
	for i := range chapters {
		chapters[i].Readiness = ReadinessPolicy{ID: chapters[i].ID + "-ready", Reason: "wait for the target-owned readiness signal", TimeoutMS: 30000, PollIntervalMS: 250, StabilityCount: 2, Cancellation: "context cancellation or timeout"}
		chapters[i].Settle = SettlePolicy{ID: chapters[i].ID + "-settle", Reason: "allow the target surface to settle after a state-changing verb", MinimumMS: 250, MaximumMS: 3000, PollInterval: 100, Cancellation: "context cancellation"}
	}
	return Plan{SchemaVersion: "android-conformance.v1", ID: PlanID, Platform: Platform, Fixture: Fixture, Chapters: chapters}
}

// InstalledAppOfflineTransitionPlan exercises only the published network chapter
// on an existing, identity-checked hello-mobile fixture. Driver reuses the
// installed-app verification lifecycle: no package installation or removal.
func InstalledAppOfflineTransitionPlan() deliveryramp.JourneyPlan {
	plan := AndroidPlan()
	for _, declared := range plan.Chapters {
		if declared.ID != "offline_transition" {
			continue
		}
		declared.Steps = append([]Step{
			step("wake", "device", "screen", "wake", "screen-control"),
			step("unlock", "device", "screen", "unlock", "screen-control"),
			step("launch", "device", "launch", "", "app-lifecycle"),
		}, declared.Steps...)
		plan.Chapters = []Chapter{declared}
		break
	}
	journey := plan.JourneyPlan()
	journey.ID = OfflineTransitionPlanID
	journey.Profile = "installed-offline-transition"
	journey.Purpose = "Exercise the existing hello-mobile fixture offline and online without replacing or removing its package"
	for index := range journey.Steps {
		spec := &journey.Steps[index]
		if spec.Action == "bas-flow" {
			spec.Arguments["flow_path"] = DefaultScenarioBASFlowPath
			if spec.Assertion == nil {
				spec.Assertion = &deliveryramp.AssertionSpec{ID: spec.ID + "/expected", Expected: "scenario BAS assertions passed while the target is offline"}
			}
		}
	}
	return journey
}

// AndroidPlanForOSVersion adapts platform-version-specific conformance chapters
// from Device Control's observed Android release. Unknown versions fail closed.
func AndroidPlanForOSVersion(osVersion string) (Plan, error) {
	major, err := androidMajorVersion(osVersion)
	if err != nil {
		return Plan{}, err
	}
	plan := AndroidPlan()
	for index := range plan.Chapters {
		chapter := &plan.Chapters[index]
		minimum, err := androidMajorVersion(chapter.MinimumAndroidVersion)
		if err != nil || minimum == 0 || major >= minimum {
			continue
		}
		chapter.Purpose = "Notification permission denial is not supported by this Android release."
		chapter.Expected = fmt.Sprintf("not applicable: Android %d predates Android %s", major, chapter.MinimumAndroidVersion)
		chapter.Steps = []Step{{
			ID: "not-applicable", Kind: "skip",
			Target: fmt.Sprintf("requires Android %s or newer; target reports Android %d", chapter.MinimumAndroidVersion, major),
		}}
	}
	if err := plan.Validate(); err != nil {
		return Plan{}, fmt.Errorf("validate Android %s conformance plan: %w", strings.TrimSpace(osVersion), err)
	}
	return plan, nil
}

func androidMajorVersion(version string) (int, error) {
	version = strings.TrimSpace(strings.TrimPrefix(strings.ToLower(strings.TrimSpace(version)), "android "))
	majorText, _, _ := strings.Cut(version, ".")
	major, err := strconv.Atoi(majorText)
	if err != nil || major <= 0 {
		return 0, fmt.Errorf("Android OS version %q is missing a numeric major version", version)
	}
	return major, nil
}

// ScenarioAppPlan returns a short, scenario-owned BAS journey for an exact
// generated Android artifact. Temporary tests require a previously absent
// package and rely on Driver's lease-bound deferred uninstall and readback.
// Durable deployment starts absent; in-place update preservation is exercised
// separately by AndroidPlan's signer-checked update_migration chapter.
func ScenarioAppPlan(mode string) (deliveryramp.JourneyPlan, error) {
	mode = strings.ToLower(strings.TrimSpace(mode))
	planID := ""
	switch mode {
	case "temporary":
		planID = TemporaryAppTestPlanID
	case "durable":
		planID = DurableAppPlanID
	case "update":
		planID = UpdateAppPlanID
	case "verify":
		planID = VerifyAppPlanID
	default:
		return deliveryramp.JourneyPlan{}, fmt.Errorf("unsupported Android scenario-app mode %q", mode)
	}
	steps := []deliveryramp.JourneyStepSpec{
		{ID: "wake", Purpose: "wake the target display for the installed app test", Action: "screen", Arguments: map[string]string{"target": "wake", "required_capabilities": "screen-control"}},
		{ID: "unlock", Purpose: "unlock the target through its Device Control auth profile when needed", Action: "screen", Arguments: map[string]string{"target": "unlock", "required_capabilities": "screen-control"}},
		{ID: "cold-launch", Purpose: "launch the exact installed scenario artifact", Action: "launch", Arguments: map[string]string{"required_capabilities": "app-lifecycle"}},
		// WebView attach is checked by Device Control after launch. A catalog
		// snapshot taken before install cannot report an app-owned WebView socket.
		{ID: "cold-bas", Purpose: "assert the scenario's cold-start WebView journey", Action: "bas-flow", Assertion: &deliveryramp.AssertionSpec{ID: "cold-bas/expected", Expected: "scenario BAS cold-start assertions passed"}},
		{ID: "background", Purpose: "background the running app while preserving its process", Action: "key", Arguments: map[string]string{"target": "KEYCODE_HOME", "required_capabilities": "input"}},
		{ID: "warm-launch", Purpose: "resume the installed app without reinstalling it", Action: "launch", Arguments: map[string]string{"required_capabilities": "app-lifecycle"}},
		{ID: "warm-bas", Purpose: "assert the scenario's warm-resume WebView journey", Action: "bas-flow", Assertion: &deliveryramp.AssertionSpec{ID: "warm-bas/expected", Expected: "scenario BAS warm-resume assertions passed"}},
	}
	if mode == "durable" {
		steps = append(steps, deliveryramp.JourneyStepSpec{ID: "verify-installed", Purpose: "verify the durable scenario package remains installed", Action: "package-state", Arguments: map[string]string{"target": "installed", "required_capabilities": "app-lifecycle"}, Assertion: &deliveryramp.AssertionSpec{ID: "verify-installed/expected", Expected: "the exact scenario package remains installed"}})
	}
	if mode == "update" {
		steps = []deliveryramp.JourneyStepSpec{
			{ID: "cold-launch", Purpose: "launch the updated scenario artifact", Action: "launch", Arguments: map[string]string{"required_capabilities": "app-lifecycle"}},
			{ID: "update-preservation", Purpose: "verify app-owned state before any workflow can rewrite it", Action: "bas-flow", Arguments: map[string]string{"flow_reference": UpdatePreservationFlowID, "flow_path": UpdatePreservationFlowPath}, Assertion: &deliveryramp.AssertionSpec{ID: "update-preservation/expected", Expected: "app-owned local storage survives the update"}},
			{ID: "background", Purpose: "background the updated app while preserving its process", Action: "key", Arguments: map[string]string{"target": "KEYCODE_HOME", "required_capabilities": "input"}},
			{ID: "warm-launch", Purpose: "resume the updated app without reinstalling it", Action: "launch", Arguments: map[string]string{"required_capabilities": "app-lifecycle"}},
			{ID: "warm-bas", Purpose: "assert the updated app's warm-resume WebView journey", Action: "bas-flow", Assertion: &deliveryramp.AssertionSpec{ID: "warm-bas/expected", Expected: "scenario BAS warm-resume assertions passed"}},
			{ID: "verify-updated", Purpose: "verify the updated scenario package remains installed", Action: "package-state", Arguments: map[string]string{"target": "installed", "required_capabilities": "app-lifecycle"}, Assertion: &deliveryramp.AssertionSpec{ID: "verify-updated/expected", Expected: "the higher-version scenario package remains installed"}},
		}
	}
	if mode == "verify" {
		steps = []deliveryramp.JourneyStepSpec{
			{ID: "wake", Purpose: "wake the target display for the installed app check", Action: "screen", Arguments: map[string]string{"target": "wake", "required_capabilities": "screen-control"}},
			{ID: "unlock", Purpose: "unlock the target through its Device Control auth profile when needed", Action: "screen", Arguments: map[string]string{"target": "unlock", "required_capabilities": "screen-control"}},
			{ID: "cold-stop", Purpose: "stop the installed app process without changing its package or saved data", Action: "stop", Arguments: map[string]string{"required_capabilities": "app-lifecycle"}},
			{ID: "cold-launch", Purpose: "cold-launch the already-installed scenario package", Action: "launch", Arguments: map[string]string{"required_capabilities": "app-lifecycle"}},
			{ID: "saved-state", Purpose: "assert the installed app's saved state before any BAS input", Action: "bas-flow", Arguments: map[string]string{"flow_reference": UpdatePreservationFlowID, "flow_path": UpdatePreservationFlowPath}, Assertion: &deliveryramp.AssertionSpec{ID: "saved-state/expected", Expected: "app-owned local storage matches the registered preservation assertion"}},
			{ID: "cold-bas", Purpose: "assert cold-start behavior with the scenario's registered read-only BAS flow", Action: "bas-flow", Arguments: map[string]string{"flow_reference": UpdatePreservationFlowID, "flow_path": UpdatePreservationFlowPath}, Assertion: &deliveryramp.AssertionSpec{ID: "cold-bas/expected", Expected: "scenario BAS cold-start assertions passed without changing app data"}},
			{ID: "background", Purpose: "background the running app while preserving its process", Action: "key", Arguments: map[string]string{"target": "KEYCODE_HOME", "required_capabilities": "input"}},
			{ID: "warm-launch", Purpose: "resume the installed app without reinstalling it", Action: "launch", Arguments: map[string]string{"required_capabilities": "app-lifecycle"}},
			{ID: "warm-bas", Purpose: "assert warm-resume behavior with the scenario's registered read-only BAS flow", Action: "bas-flow", Arguments: map[string]string{"flow_reference": UpdatePreservationFlowID, "flow_path": UpdatePreservationFlowPath}, Assertion: &deliveryramp.AssertionSpec{ID: "warm-bas/expected", Expected: "scenario BAS warm-resume assertions passed without changing app data"}},
			{ID: "verify-installed", Purpose: "verify the same scenario package remains installed after BAS checks", Action: "package-state", Arguments: map[string]string{"target": "installed", "required_capabilities": "app-lifecycle"}, Assertion: &deliveryramp.AssertionSpec{ID: "verify-installed/expected", Expected: "the scenario package remains installed"}},
		}
	}
	return deliveryramp.JourneyPlan{SchemaVersion: deliveryramp.JourneyEvidenceVersion, ID: planID, Capability: "generated-app", Purpose: "Scenario-built Android app cold and warm BAS test", Profile: mode, Steps: steps}, nil
}

func chapter(id, purpose, expected string, required []string, steps ...Step) Chapter {
	return Chapter{ID: id, Purpose: purpose, Expected: expected, RequiredCapabilities: required, Steps: steps}
}

func step(id, kind, reference, target, capability string) Step {
	return Step{ID: id, Kind: kind, Reference: reference, Target: target, TimeoutMS: actionTimeoutMS(reference), RequiredCapabilities: []string{capability}}
}

// BAS attaches only after the native app has launched and exposed a debuggable
// WebView. That runtime capability cannot be truthfully required by the target
// catalog before the app exists; the WebViewAttacher is the authoritative
// runtime check.
func basStep(id, flow string) Step {
	return Step{ID: id, Kind: "bas", Reference: "flow", Target: flow, TimeoutMS: actionTimeoutMS("flow")}
}

func basStepWithFlowPath(id, flow, flowPath string) Step {
	return Step{ID: id, Kind: "bas", Reference: "flow", Target: flow, TimeoutMS: actionTimeoutMS("flow"), Arguments: map[string]any{"flow_path": flowPath}}
}

// actionTimeoutMS keeps individual device verbs bounded while allowing
// wireless APK transfers enough time to complete on a real handset.
func actionTimeoutMS(reference string) int64 {
	switch reference {
	case "install":
		return 120000
	case "launch", "stop", "uninstall", "package-state", "revoke-permission", "network", "rotate", "screen", "wake", "unlock", "key", "deep-link":
		return 60000
	case "observe", "clock-sample", "logcat-start", "logcat-stop":
		return 45000
	default:
		return 60000
	}
}

func (p Plan) Validate() error {
	if p.SchemaVersion == "" || p.ID != PlanID || p.Platform != Platform || p.Fixture != Fixture {
		return fmt.Errorf("invalid Android generated-app conformance identity")
	}
	if len(p.Chapters) != 12 {
		return fmt.Errorf("Android generated-app plan has %d chapters, want 12", len(p.Chapters))
	}
	seen := make(map[string]struct{}, len(p.Chapters))
	for _, chapter := range p.Chapters {
		if strings.TrimSpace(chapter.ID) == "" {
			return fmt.Errorf("conformance chapter id is required")
		}
		if _, ok := seen[chapter.ID]; ok {
			return fmt.Errorf("duplicate conformance chapter %q", chapter.ID)
		}
		seen[chapter.ID] = struct{}{}
		if strings.TrimSpace(chapter.Purpose) == "" || strings.TrimSpace(chapter.Expected) == "" || len(chapter.Steps) == 0 {
			return fmt.Errorf("conformance chapter %q is incomplete", chapter.ID)
		}
		if chapter.Readiness.TimeoutMS <= 0 || chapter.Readiness.PollIntervalMS <= 0 || chapter.Readiness.StabilityCount <= 0 {
			return fmt.Errorf("conformance chapter %q has an unbounded readiness policy", chapter.ID)
		}
		if chapter.Settle.MinimumMS < 0 || chapter.Settle.MaximumMS < chapter.Settle.MinimumMS || chapter.Settle.PollInterval <= 0 {
			return fmt.Errorf("conformance chapter %q has an invalid settle policy", chapter.ID)
		}
		for _, current := range chapter.Steps {
			if strings.TrimSpace(current.ID) == "" || strings.TrimSpace(current.Kind) == "" || (current.Kind != "bas" && current.Kind != "skip" && len(current.RequiredCapabilities) == 0) {
				return fmt.Errorf("conformance chapter %q has an incomplete step", chapter.ID)
			}
			if current.Kind == "skip" && strings.TrimSpace(current.Target) == "" {
				return fmt.Errorf("conformance chapter %q skip has no applicability reason", chapter.ID)
			}
			if current.Kind == "device" && (current.TimeoutMS <= 0 || current.TimeoutMS > 10*60*1000) {
				return fmt.Errorf("conformance chapter %q step %q has an invalid device timeout", chapter.ID, current.ID)
			}
			if current.Kind == "bas" && strings.TrimSpace(current.Target) == "" {
				return fmt.Errorf("conformance chapter %q BAS step has no flow reference", chapter.ID)
			}
		}
	}
	return nil
}
