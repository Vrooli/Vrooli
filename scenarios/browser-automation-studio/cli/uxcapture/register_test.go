package uxcapture

import (
	"reflect"
	"strings"
	"testing"
)

func TestValidateWaitSelectorIsScenarioOnlyAndBounded(t *testing.T) {
	if got, err := validateWaitSelector("web-console", ` [data-testid="pane-grid"] `); err != nil || got != `[data-testid="pane-grid"]` {
		t.Fatalf("validate scenario selector = %q, %v", got, err)
	}
	if _, err := validateWaitSelector("", `[data-testid="pane-grid"]`); err == nil {
		t.Fatal("expected a public capture to reject a scenario wait selector")
	}
	if _, err := validateWaitSelector("web-console", strings.Repeat("x", 513)); err == nil {
		t.Fatal("expected an overlong selector to be rejected")
	}
	if got, err := validateWaitSelector("web-console", "   "); err != nil || got != "" {
		t.Fatalf("blank selector = %q, %v", got, err)
	}
}

func TestBuildScenarioRequestCarriesWaitSelector(t *testing.T) {
	got := buildScenarioRequest("web-console", "/", nil, 1280, 720, "light", `[data-testid="pane-grid"]`, "", "")
	want := map[string]any{
		"scenario":      "web-console",
		"route":         "/",
		"steps":         []string(nil),
		"width":         1280,
		"height":        720,
		"theme":         "light",
		"wait_selector": `[data-testid="pane-grid"]`,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("scenario request = %#v, want %#v", got, want)
	}
}

func TestValidateRegionSelectorRequiresComponentTestScenarioAndIsBounded(t *testing.T) {
	if got, err := validateRegionSelector("react-component-library", "componenttests", `[data-preview-capture-boundary]`); err != nil || got != `[data-preview-capture-boundary]` {
		t.Fatalf("validate region selector = %q, %v", got, err)
	}
	if _, err := validateRegionSelector("react-component-library", "", `[data-preview-capture-boundary]`); err == nil {
		t.Fatal("expected an evidence profile error")
	}
	if _, err := validateRegionSelector("", "componenttests", `[data-preview-capture-boundary]`); err == nil {
		t.Fatal("expected a scenario-only error")
	}
	if _, err := validateRegionSelector("react-component-library", "componenttests", strings.Repeat("x", 257)); err == nil {
		t.Fatal("expected an overlong selector to be rejected")
	}
}

func TestBuildScenarioRequestCarriesComponentEvidenceProfile(t *testing.T) {
	got := buildScenarioRequest("react-component-library", "/preview/button/harness.html?version=1.2.3", nil, 1280, 900, "light", "", "componenttests", `[data-preview-capture-boundary]`)
	if got["evidence_profile"] != "componenttests" || got["region_selector"] != `[data-preview-capture-boundary]` {
		t.Fatalf("component evidence request = %#v", got)
	}
}
