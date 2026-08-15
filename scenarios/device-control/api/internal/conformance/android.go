// Package conformance owns device-control's provider-neutral Android
// capability self-test. Generated-app behavior belongs to the Android ramp;
// this package only proves that a device can provide the verbs it advertises.
package conformance

import (
	"fmt"
	"strings"

	"device-control/internal/execution"
)

const PlanID = "android-device-capability-self-test-v1"

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
	Chapters      []Chapter `json:"chapters"`
}

func AndroidCapabilityPlan() Plan {
	return Plan{
		SchemaVersion: "android-device-capability.v1",
		ID:            PlanID,
		Platform:      "android",
		DeviceKind:    "physical-or-emulator",
		Chapters: []Chapter{
			{ID: "device_observation", Purpose: "Capture one redacted device frame.", Expected: "the device exposes a visible surface", RequiredCapabilities: []string{"screenshot"}, Steps: []Step{{ID: "observe", Kind: "observe", RequiredCapabilities: []string{"screenshot"}}}},
			{ID: "input_dispatch", Purpose: "Dispatch a provider-neutral input action.", Expected: "input dispatch completes without bypassing the strategy", RequiredCapabilities: []string{"input"}, Steps: []Step{{ID: "key", Kind: "key", Target: "KEYCODE_BACK", RequiredCapabilities: []string{"input"}}}},
			{ID: "semantic_resolution", Purpose: "Resolve a semantic target through the device strategy.", Expected: "semantic resolution returns a governed target", RequiredCapabilities: []string{"semantic-tree"}, Steps: []Step{{ID: "semantic", Kind: "semantic-target", Target: "device-control-self-test", RequiredCapabilities: []string{"semantic-tree"}}}},
			{ID: "network_control", Purpose: "Exercise the network-control verb and restore connectivity.", Expected: "network state can be changed and restored", RequiredCapabilities: []string{"network-control"}, Steps: []Step{{ID: "offline", Kind: "network", Target: "offline", RequiredCapabilities: []string{"network-control"}}, {ID: "online", Kind: "network", Target: "online", RequiredCapabilities: []string{"network-control"}}}},
			{ID: "screen_recording", Purpose: "Capture a short native evidence recording.", Expected: "recording is measurable and redacted", RequiredCapabilities: []string{"screen-recording"}, Steps: []Step{{ID: "record", Kind: "screenrecord", RequiredCapabilities: []string{"screen-recording"}}}},
		},
	}
}

// AndroidPlan remains a narrow compatibility name for callers that request
// device-control's Android plan. It is deliberately the capability self-test;
// it never names a generated app or fixture.
func AndroidPlan() Plan { return AndroidCapabilityPlan() }

func (p Plan) Validate() error {
	if p.SchemaVersion == "" || p.ID != PlanID || p.Platform != "android" || p.DeviceKind == "" {
		return fmt.Errorf("invalid Android device capability plan identity")
	}
	if len(p.Chapters) != 5 {
		return fmt.Errorf("Android device capability plan has %d chapters, want 5", len(p.Chapters))
	}
	seen := make(map[string]bool, len(p.Chapters))
	for _, chapter := range p.Chapters {
		if strings.TrimSpace(chapter.ID) == "" || seen[chapter.ID] {
			return fmt.Errorf("duplicate or empty capability chapter %q", chapter.ID)
		}
		seen[chapter.ID] = true
		if len(chapter.Steps) == 0 || strings.TrimSpace(chapter.Expected) == "" {
			return fmt.Errorf("capability chapter %q has no bounded steps or expected result", chapter.ID)
		}
		for _, step := range chapter.Steps {
			if strings.TrimSpace(step.ID) == "" || step.TimeoutMS < 0 || step.TimeoutMS > 10*60*1000 {
				return fmt.Errorf("capability chapter %q contains an invalid step", chapter.ID)
			}
		}
	}
	return nil
}

func (c Chapter) Flow() execution.Flow {
	steps := make([]execution.Step, 0, len(c.Steps))
	for _, step := range c.Steps {
		steps = append(steps, execution.Step{ID: step.ID, Kind: step.Kind, Target: step.Target, RequiredCapabilities: append([]string{}, step.RequiredCapabilities...), TimeoutMS: step.TimeoutMS, Arguments: step.Arguments})
	}
	return execution.Flow{ID: PlanID + ":" + c.ID, Name: c.Purpose, Steps: steps}
}
