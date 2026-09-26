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
	ID                   string          `json:"id"`
	Purpose              string          `json:"purpose"`
	Expected             string          `json:"expected"`
	RequiredCapabilities []string        `json:"required_capabilities"`
	Readiness            ReadinessPolicy `json:"readiness"`
	Settle               SettlePolicy    `json:"settle"`
	Steps                []Step          `json:"steps"`
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
		for _, declared := range chapter.Steps {
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
			}
			for key, value := range declared.Arguments {
				if text, ok := value.(string); ok {
					args[key] = text
				}
			}
			steps = append(steps, deliveryramp.JourneyStepSpec{
				ID: chapter.ID + "/" + declared.ID, Purpose: chapter.Purpose, Action: action, Arguments: args, Capture: true,
				Readiness: deliveryramp.ReadinessPolicy{ID: chapter.Readiness.ID, Reason: chapter.Readiness.Reason, Timeout: time.Duration(chapter.Readiness.TimeoutMS) * time.Millisecond, PollInterval: time.Duration(chapter.Readiness.PollIntervalMS) * time.Millisecond, StabilityCount: chapter.Readiness.StabilityCount, Cancellation: chapter.Readiness.Cancellation},
				Settle:    deliveryramp.SettlePolicy{ID: chapter.Settle.ID, Reason: chapter.Settle.Reason, Minimum: time.Duration(chapter.Settle.MinimumMS) * time.Millisecond, Maximum: time.Duration(chapter.Settle.MaximumMS) * time.Millisecond, PollInterval: time.Duration(chapter.Settle.PollInterval) * time.Millisecond, Cancellation: chapter.Settle.Cancellation},
				Assertion: &deliveryramp.AssertionSpec{ID: chapter.ID + "/expected", Expected: chapter.Expected, Description: chapter.Purpose},
			})
		}
	}
	return deliveryramp.JourneyPlan{SchemaVersion: deliveryramp.JourneyEvidenceVersion, ID: p.ID, Capability: "generated-app", Purpose: "Android generated-app conformance", Profile: "normal", Steps: steps}
}

func AndroidPlan() Plan {
	chapters := []Chapter{
		chapter("install_cold_start", "Install and launch a generated app from a clean state.", "the first usable frame and app title are visible", []string{"app-lifecycle", "screenshot"}, step("wake", "device", "screen", "wake", "input"), step("unlock", "device", "screen", "unlock", "input"), step("install", "device", "install", "", "app-lifecycle"), step("launch", "device", "launch", "", "app-lifecycle"), step("observe", "device", "observe", "", "screenshot"), step("web-flow", "bas", "flow", "hello-mobile-smoke", "webview-attach")),
		chapter("permission_deny_grace", "Deny a runtime permission and relaunch.", "the app remains usable after denial", []string{"permissions", "app-lifecycle", "screenshot"}, step("deny", "device", "revoke-permission", "android.permission.POST_NOTIFICATIONS", "permissions"), step("launch", "device", "launch", "", "app-lifecycle"), step("web-flow", "bas", "flow", "hello-mobile-smoke", "webview-attach")),
		chapter("background_resume", "Background and resume the generated app.", "the app returns with its state intact", []string{"input", "app-lifecycle", "screenshot"}, step("background", "device", "key", "KEYCODE_HOME", "input"), step("resume", "device", "launch", "", "app-lifecycle"), step("web-flow", "bas", "flow", "hello-mobile-smoke", "webview-attach")),
		chapter("process_death_restore", "Force-stop and relaunch the generated app.", "persisted state is restored after process death", []string{"app-lifecycle", "screenshot"}, step("stop", "device", "stop", "", "app-lifecycle"), step("launch", "device", "launch", "", "app-lifecycle"), step("web-flow", "bas", "flow", "hello-mobile-smoke", "webview-attach")),
		chapter("rotation_size_class", "Rotate the target and inspect the responsive layout.", "the input remains usable after rotation", []string{"orientation", "screenshot"}, step("rotate", "device", "rotate", "landscape", "orientation"), step("web-flow", "bas", "flow", "hello-mobile-smoke", "webview-attach")),
		chapter("keyboard_avoidance", "Focus the fixture input and inspect keyboard avoidance.", "the focused input remains visible", []string{"semantic-tree", "screenshot"}, step("web-flow", "bas", "flow", "hello-mobile-smoke", "webview-attach")),
		chapter("offline_transition", "Toggle connectivity and inspect both terminal states.", "offline and online state are explicit", []string{"network-control", "screenshot"}, step("offline", "device", "network", "offline", "network-control"), step("web-flow-offline", "bas", "flow", "hello-mobile-smoke", "webview-attach"), step("online", "device", "network", "online", "network-control"), step("web-flow-online", "bas", "flow", "hello-mobile-smoke", "webview-attach")),
		chapter("deep_link", "Open the generated app through its canonical deep link.", "the home route is rendered", []string{"app-lifecycle", "screenshot"}, step("open", "device", "deep-link", "vrooli-hello://home", "app-lifecycle"), step("web-flow", "bas", "flow", "hello-mobile-smoke", "webview-attach")),
		chapter("notification_tap", "Resolve the fixture notification target.", "the notification action reaches the app", []string{"semantic-tree", "screenshot"}, step("web-flow", "bas", "flow", "hello-mobile-smoke", "webview-attach")),
		chapter("back_navigation", "Use native back navigation from a fixture route.", "back returns to the home route", []string{"input", "screenshot"}, step("back", "device", "key", "KEYCODE_BACK", "input"), step("web-flow", "bas", "flow", "hello-mobile-smoke", "webview-attach")),
		chapter("update_migration", "Install an update over existing app data.", "app state survives the update", []string{"app-lifecycle", "screenshot"}, step("update", "device", "install", "", "app-lifecycle"), step("launch", "device", "launch", "", "app-lifecycle"), step("web-flow", "bas", "flow", "hello-mobile-smoke", "webview-attach")),
		chapter("clean_uninstall", "Uninstall the generated app and verify package absence.", "the app package is absent", []string{"app-lifecycle", "screenshot"}, step("uninstall", "device", "uninstall", "", "app-lifecycle"), step("assert-absent", "device", "package-state", "absent", "app-lifecycle")),
	}
	for i := range chapters {
		chapters[i].Readiness = ReadinessPolicy{ID: chapters[i].ID + "-ready", Reason: "wait for the target-owned readiness signal", TimeoutMS: 30000, PollIntervalMS: 250, StabilityCount: 2, Cancellation: "context cancellation or timeout"}
		chapters[i].Settle = SettlePolicy{ID: chapters[i].ID + "-settle", Reason: "allow the target surface to settle after a state-changing verb", MinimumMS: 250, MaximumMS: 3000, PollInterval: 100, Cancellation: "context cancellation"}
	}
	return Plan{SchemaVersion: "android-conformance.v1", ID: PlanID, Platform: Platform, Fixture: Fixture, Chapters: chapters}
}

func chapter(id, purpose, expected string, required []string, steps ...Step) Chapter {
	return Chapter{ID: id, Purpose: purpose, Expected: expected, RequiredCapabilities: required, Steps: steps}
}

func step(id, kind, reference, target, capability string) Step {
	return Step{ID: id, Kind: kind, Reference: reference, Target: target, TimeoutMS: actionTimeoutMS(reference), RequiredCapabilities: []string{capability}}
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
			if strings.TrimSpace(current.ID) == "" || strings.TrimSpace(current.Kind) == "" || len(current.RequiredCapabilities) == 0 {
				return fmt.Errorf("conformance chapter %q has an incomplete step", chapter.ID)
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
