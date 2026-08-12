// Package conformance owns the provider-neutral Android platform chapter set.
// It describes what a ramp must prove; device-control supplies the verbs and
// evidence references used to execute it.
package conformance

import (
	"fmt"
	"strings"

	"device-control/internal/execution"
)

const (
	PlanID    = "android-physical-conformance-v1"
	FixtureID = "hello-mobile"
)

type Step struct {
	ID                   string         `json:"id"`
	Kind                 string         `json:"kind"`
	Target               string         `json:"target,omitempty"`
	RequiredCapabilities []string       `json:"required_capabilities,omitempty"`
	TimeoutMS            int64          `json:"timeout_ms"`
	Arguments            map[string]any `json:"arguments,omitempty"`
}

type Chapter struct {
	ID                   string   `json:"id"`
	Purpose              string   `json:"purpose"`
	Expected             string   `json:"expected"`
	RequiredCapabilities []string `json:"required_capabilities"`
	Steps                []Step   `json:"steps"`
}

type Plan struct {
	SchemaVersion string    `json:"schema_version"`
	ID            string    `json:"id"`
	Platform      string    `json:"platform"`
	DeviceKind    string    `json:"device_kind"`
	Fixture       string    `json:"fixture"`
	Chapters      []Chapter `json:"chapters"`
}

type Fixture struct {
	ID          string `json:"id"`
	PackageName string `json:"package_name"`
	APKPath     string `json:"apk_path"`
	DeepLink    string `json:"deep_link,omitempty"`
	Version     string `json:"version,omitempty"`
}

func AndroidPlan() Plan {
	plan := Plan{
		SchemaVersion: "android-conformance.v1",
		ID:            PlanID,
		Platform:      "android",
		DeviceKind:    "physical",
		Fixture:       FixtureID,
		Chapters: []Chapter{
			{ID: "install_cold_start", Purpose: "Install the fixture and launch it from a clean state.", Expected: "first usable frame is captured", RequiredCapabilities: []string{"app-lifecycle", "screenshot"}, Steps: []Step{{ID: "install", Kind: "install", RequiredCapabilities: []string{"app-lifecycle"}}, {ID: "launch", Kind: "launch", RequiredCapabilities: []string{"app-lifecycle"}}, {ID: "observe", Kind: "observe", RequiredCapabilities: []string{"screenshot"}}}},
			{ID: "permission_deny_grace", Purpose: "Deny a runtime permission and verify graceful startup.", Expected: "fixture remains usable after denial", RequiredCapabilities: []string{"permissions", "app-lifecycle", "screenshot"}, Steps: []Step{{ID: "revoke", Kind: "revoke-permission", RequiredCapabilities: []string{"permissions"}}, {ID: "launch", Kind: "launch", RequiredCapabilities: []string{"app-lifecycle"}}, {ID: "observe", Kind: "observe", RequiredCapabilities: []string{"screenshot"}}}},
			{ID: "background_resume", Purpose: "Background and resume without losing the fixture state.", Expected: "state is preserved after resume", RequiredCapabilities: []string{"input", "app-lifecycle", "screenshot"}, Steps: []Step{{ID: "home", Kind: "key", Target: "KEYCODE_HOME", RequiredCapabilities: []string{"input"}}, {ID: "resume", Kind: "launch", RequiredCapabilities: []string{"app-lifecycle"}}, {ID: "observe", Kind: "observe", RequiredCapabilities: []string{"screenshot"}}}},
			{ID: "process_death_restore", Purpose: "Force-stop and relaunch the fixture.", Expected: "prior state is restored after process death", RequiredCapabilities: []string{"app-lifecycle", "screenshot"}, Steps: []Step{{ID: "stop", Kind: "stop", RequiredCapabilities: []string{"app-lifecycle"}}, {ID: "launch", Kind: "launch", RequiredCapabilities: []string{"app-lifecycle"}}, {ID: "observe", Kind: "observe", RequiredCapabilities: []string{"screenshot"}}}},
			{ID: "rotation_size_class", Purpose: "Rotate the device and check the fixture remains usable.", Expected: "layout remains usable after rotation", RequiredCapabilities: []string{"orientation", "screenshot"}, Steps: []Step{{ID: "rotate", Kind: "rotate", Target: "landscape", RequiredCapabilities: []string{"orientation"}}, {ID: "observe", Kind: "observe", RequiredCapabilities: []string{"screenshot"}}}},
			{ID: "keyboard_avoidance", Purpose: "Focus the fixture input and inspect the resulting layout.", Expected: "focused input is not obscured by the keyboard", RequiredCapabilities: []string{"semantic-tree", "screenshot"}, Steps: []Step{{ID: "input", Kind: "semantic-target", Target: "hello-mobile-input", RequiredCapabilities: []string{"semantic-tree"}}, {ID: "observe", Kind: "observe", RequiredCapabilities: []string{"screenshot"}}}},
			{ID: "offline_transition", Purpose: "Toggle connectivity and verify a bounded offline state.", Expected: "offline state is explicit and recovers", RequiredCapabilities: []string{"network-control", "screenshot"}, Steps: []Step{{ID: "offline", Kind: "network", Target: "offline", RequiredCapabilities: []string{"network-control"}}, {ID: "offline-observe", Kind: "observe", RequiredCapabilities: []string{"screenshot"}}, {ID: "online", Kind: "network", Target: "online", RequiredCapabilities: []string{"network-control"}}, {ID: "online-observe", Kind: "observe", RequiredCapabilities: []string{"screenshot"}}}},
			{ID: "deep_link", Purpose: "Open the fixture through its canonical deep link.", Expected: "the deep-linked route is rendered", RequiredCapabilities: []string{"app-lifecycle", "screenshot"}, Steps: []Step{{ID: "open-link", Kind: "deep-link", RequiredCapabilities: []string{"app-lifecycle"}}, {ID: "observe", Kind: "observe", RequiredCapabilities: []string{"screenshot"}}}},
			{ID: "notification_tap", Purpose: "Resolve the fixture notification target semantically.", Expected: "notification action opens the expected fixture route", RequiredCapabilities: []string{"semantic-tree", "screenshot"}, Steps: []Step{{ID: "notification", Kind: "semantic-target", Target: "hello-mobile-notification", RequiredCapabilities: []string{"semantic-tree"}}, {ID: "observe", Kind: "observe", RequiredCapabilities: []string{"screenshot"}}}},
			{ID: "back_navigation", Purpose: "Use native back navigation from a fixture route.", Expected: "back navigation returns to the prior route", RequiredCapabilities: []string{"input", "screenshot"}, Steps: []Step{{ID: "back", Kind: "key", Target: "KEYCODE_BACK", RequiredCapabilities: []string{"input"}}, {ID: "observe", Kind: "observe", RequiredCapabilities: []string{"screenshot"}}}},
			{ID: "update_migration", Purpose: "Install an update over existing fixture data.", Expected: "fixture data survives the update", RequiredCapabilities: []string{"app-lifecycle", "screenshot"}, Steps: []Step{{ID: "install-update", Kind: "install", RequiredCapabilities: []string{"app-lifecycle"}}, {ID: "launch", Kind: "launch", RequiredCapabilities: []string{"app-lifecycle"}}, {ID: "observe", Kind: "observe", RequiredCapabilities: []string{"screenshot"}}}},
			{ID: "clean_uninstall", Purpose: "Remove the fixture and verify the package is gone.", Expected: "fixture package is uninstalled", RequiredCapabilities: []string{"app-lifecycle", "screenshot"}, Steps: []Step{{ID: "uninstall", Kind: "uninstall", RequiredCapabilities: []string{"app-lifecycle"}}, {ID: "assert-absent", Kind: "package-state", Target: "absent", RequiredCapabilities: []string{"app-lifecycle"}}, {ID: "observe", Kind: "observe", RequiredCapabilities: []string{"screenshot"}}}},
		},
	}
	enrichAndroidAssertions(&plan)
	return plan
}

func enrichAndroidAssertions(plan *Plan) {
	for i := range plan.Chapters {
		chapter := &plan.Chapters[i]
		appendAssertion := func(id, target, expected string) {
			chapter.Steps = append(chapter.Steps, Step{ID: id, Kind: "semantic-assert", Target: target, RequiredCapabilities: []string{"semantic-tree"}, Arguments: map[string]any{"expected": expected}})
		}
		switch chapter.ID {
		case "install_cold_start":
			appendAssertion("assert-title", "hello-mobile-title", "Hello Mobile")
		case "permission_deny_grace":
			appendAssertion("assert-input", "hello-mobile-input", "hello-mobile-input")
		case "background_resume":
			chapter.Steps = append([]Step{{ID: "seed-state", Kind: "text", Target: "hello-mobile-state", RequiredCapabilities: []string{"input"}}}, chapter.Steps...)
			appendAssertion("assert-state", "hello-mobile-input", "hello-mobile-state")
		case "process_death_restore":
			appendAssertion("assert-restored-state", "hello-mobile-input", "hello-mobile-state")
		case "rotation_size_class":
			appendAssertion("assert-input", "hello-mobile-input", "hello-mobile-input")
		case "keyboard_avoidance":
			appendAssertion("assert-input", "hello-mobile-input", "hello-mobile-input")
		case "offline_transition":
			insertAfter(chapter, "offline-observe", Step{ID: "assert-offline", Kind: "semantic-assert", Target: "hello-mobile-connectivity", RequiredCapabilities: []string{"semantic-tree"}, Arguments: map[string]any{"expected": "Connectivity: offline"}})
			appendAssertion("assert-online", "hello-mobile-connectivity", "Connectivity: online")
		case "deep_link":
			appendAssertion("assert-route", "hello-mobile-route", "Route: home")
		case "notification_tap":
			appendAssertion("assert-notification", "hello-mobile-notification", "hello-mobile-notification")
		case "back_navigation":
			appendAssertion("assert-route", "hello-mobile-route", "Route: home")
		case "update_migration":
			appendAssertion("assert-migrated-state", "hello-mobile-input", "hello-mobile-state")
		}
	}
}

func insertAfter(chapter *Chapter, stepID string, addition Step) {
	for i, step := range chapter.Steps {
		if step.ID == stepID {
			chapter.Steps = append(chapter.Steps, Step{})
			copy(chapter.Steps[i+2:], chapter.Steps[i+1:])
			chapter.Steps[i+1] = addition
			return
		}
	}
	chapter.Steps = append(chapter.Steps, addition)
}

func (p Plan) Validate() error {
	if p.SchemaVersion == "" || p.ID == "" || p.Platform != "android" || p.DeviceKind != "physical" || p.Fixture != FixtureID {
		return fmt.Errorf("invalid Android conformance plan identity")
	}
	if len(p.Chapters) != 12 {
		return fmt.Errorf("Android conformance plan has %d chapters, want 12", len(p.Chapters))
	}
	seen := make(map[string]bool, len(p.Chapters))
	for _, chapter := range p.Chapters {
		if strings.TrimSpace(chapter.ID) == "" || seen[chapter.ID] {
			return fmt.Errorf("duplicate or empty conformance chapter %q", chapter.ID)
		}
		seen[chapter.ID] = true
		if len(chapter.Steps) == 0 || strings.TrimSpace(chapter.Expected) == "" {
			return fmt.Errorf("conformance chapter %q has no bounded steps or expected result", chapter.ID)
		}
		for _, step := range chapter.Steps {
			if step.TimeoutMS < 0 || step.TimeoutMS > 10*60*1000 {
				return fmt.Errorf("conformance step %q has invalid timeout", step.ID)
			}
		}
	}
	return nil
}

func (c Chapter) Flow(fixture Fixture) execution.Flow {
	steps := make([]execution.Step, 0, len(c.Steps))
	for _, step := range c.Steps {
		target := step.Target
		args := map[string]any{}
		if step.Kind == "install" {
			target = fixture.APKPath
			args["package"] = fixture.PackageName
		}
		if step.Kind == "launch" || step.Kind == "stop" || step.Kind == "uninstall" || step.Kind == "clear-data" {
			target = fixture.PackageName
			args["package"] = fixture.PackageName
		}
		if step.Kind == "package-state" {
			target = fixture.PackageName
			args["package"] = fixture.PackageName
			args["expected"] = step.Target
		}
		if step.Kind == "revoke-permission" {
			args["package"] = fixture.PackageName
			args["permission"] = "android.permission.POST_NOTIFICATIONS"
		}
		if step.Kind == "deep-link" {
			target = fixture.DeepLink
			args["package"] = fixture.PackageName
		}
		steps = append(steps, execution.Step{ID: step.ID, Kind: step.Kind, Target: target, RequiredCapabilities: append([]string{}, step.RequiredCapabilities...), TimeoutMS: step.TimeoutMS, Arguments: args})
	}
	return execution.Flow{ID: PlanID + ":" + c.ID, Name: c.Purpose, Steps: steps}
}

func (f Fixture) Validate() error {
	if f.ID != FixtureID || strings.TrimSpace(f.PackageName) == "" || strings.TrimSpace(f.APKPath) == "" || strings.TrimSpace(f.DeepLink) == "" {
		return fmt.Errorf("fixture must provide id %q, package_name, apk_path, and deep_link", FixtureID)
	}
	return nil
}
