package catalog

import (
	"context"
	"testing"

	"react-component-library/internal/catalogcoverage"
	"react-component-library/internal/components"
	"react-component-library/internal/componenttests"
	"react-component-library/internal/gates"
)

func TestAppearanceGatesUseVersionPinnedBrowserStories(t *testing.T) {
	for _, gate := range []string{"unit", "interaction", "accessibility", "responsive"} {
		if !appearanceGate(gate) {
			t.Errorf("%s must run against the version-pinned browser story report", gate)
		}
	}
	for _, gate := range []string{"visual", "composition", "tokens"} {
		if appearanceGate(gate) {
			t.Errorf("%s must keep its owning non-appearance evaluator", gate)
		}
	}
}

type appearanceAssetsStub struct {
	components.Service
	component components.Component
}

func (s appearanceAssetsStub) Get(context.Context, string) (components.Component, error) {
	return s.component, nil
}

type appearanceExecutorSpy struct{ calls int }

func (s *appearanceExecutorSpy) ExecuteStory(context.Context, string, string, string) (componenttests.StoryExecution, error) {
	s.calls++
	return componenttests.StoryExecution{Passed: true}, nil
}

func TestAppearanceGateReadOnlyInspectionNeverStartsCapture(t *testing.T) {
	assets := appearanceAssetsStub{component: components.Component{
		ID: "component-id", CatalogID: "controls.button", LibraryID: "react-component-library:Button", LatestVersion: "1.0.0",
	}}
	executor := &appearanceExecutorSpy{}
	testService := componenttests.NewService(componenttests.Runner{Assets: assets, Executor: executor}, nil)
	h := &handler{assets: assets, testService: testService}
	result, err := h.runAppearanceGate(context.Background(), "responsive", "controls.button", catalogcoverage.GateDefinition{}, false)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "unmeasured" || len(result.RunnerError) != 1 || result.RunnerError[0].Code != "catalog.responsive.evidence_missing" {
		t.Fatalf("missing current evidence = %+v, want explicit unmeasured result", result)
	}
	if executor.calls != 0 {
		t.Fatalf("read-only gate inspection invoked story executor %d times", executor.calls)
	}
}

func TestParseAxeEvidenceRequiresNamedEngineVersionAndResults(t *testing.T) {
	valid := `{"schema":"bas-accessibility-snapshot/v1","snapshot":"main\n button \"Save\"","axe":{"schema":"rcl-axe-results/v1","status":"complete","engine":"axe-core","version":"4.11.4","violations":[]}}`
	violations, measured := parseAxeEvidence(valid)
	if !measured || violations == nil || len(violations) != 0 {
		t.Fatalf("empty axe result should be measured and clean: measured=%t violations=%+v", measured, violations)
	}
	defect := `{"schema":"bas-accessibility-snapshot/v1","snapshot":"main","axe":{"schema":"rcl-axe-results/v1","status":"complete","engine":"axe-core","version":"4.11.4","violations":[{"id":"button-name","impact":"serious","description":"Buttons must have discernible text."}]}}`
	violations, measured = parseAxeEvidence(defect)
	if !measured || len(violations) != 1 || violations[0].ID != "button-name" {
		t.Fatalf("axe violation was not preserved: measured=%t violations=%+v", measured, violations)
	}
	for _, missing := range []string{
		`{"schema":"bas-accessibility-snapshot/v1","snapshot":"main"}`,
		`{"schema":"bas-accessibility-snapshot/v1","snapshot":"main","axe":{"engine":"other","version":"4.10.2","violations":[]}}`,
		`{"schema":"bas-accessibility-snapshot/v1","snapshot":"main","axe":{"status":"complete","engine":"axe-core","version":"4.11.4","violations":[]}}`,
		`{"schema":"bas-accessibility-snapshot/v1","snapshot":"main","axe":{"schema":"rcl-axe-results/v1","status":"unavailable","engine":"axe-core","version":"4.11.4","violations":[]}}`,
		`{"schema":"bas-accessibility-snapshot/v1","snapshot":"main","axe":{"engine":"axe-core","violations":[]}}`,
		`{"schema":"bas-accessibility-snapshot/v1","snapshot":"main","axe":{"engine":"axe-core","version":"4.10.2","violations":null}}`,
	} {
		if _, measured := parseAxeEvidence(missing); measured {
			t.Errorf("incomplete or foreign accessibility evidence must remain unmeasured: %s", missing)
		}
	}
}

func TestAxeGateParserConsumesVersionedRCLStoryResult(t *testing.T) {
	serialized := `{"schema":"rcl-axe-results/v1","status":"complete","engine":"axe-core","version":"4.11.4","violations":[{"id":"button-name","impact":"serious","description":"Buttons must have discernible text.","targets":["button:nth-child(1)"]}]}`
	violations, measured := parseAxeEvidence(serialized)
	if !measured || len(violations) != 1 || violations[0].ID != "button-name" || violations[0].Targets[0] != "button:nth-child(1)" {
		t.Fatalf("accessibility gate did not consume the story harness result: measured=%t violations=%+v", measured, violations)
	}
	for _, unavailable := range []string{
		`{"schema":"rcl-axe-results/v1","status":"unavailable","reason":"axe-core-not-loaded"}`,
		`{"schema":"rcl-axe-results/v1","status":"complete","engine":"other","version":"4.11.4","violations":[]}`,
	} {
		if _, measured := parseAxeEvidence(unavailable); measured {
			t.Errorf("unavailable or foreign story result was accepted: %s", unavailable)
		}
	}
}

func TestResponsiveGateUsesExactRequiredViewportWidths(t *testing.T) {
	for input, expected := range map[string]int{"320": 320, "390px": 390, " 1280 px ": 1280} {
		if got, ok := widthPixels(input); !ok || got != expected {
			t.Errorf("widthPixels(%q) = %d, %t; want %d, true", input, got, ok, expected)
		}
	}
	for _, input := range []string{"", "auto", "320rem", "0px", "-1px"} {
		if _, ok := widthPixels(input); ok {
			t.Errorf("widthPixels(%q) accepted a non-positive or non-pixel width", input)
		}
	}
}

func TestResponsiveGateNeedsBothRenderedPhoneWidths(t *testing.T) {
	definition := catalogcoverage.GateDefinition{RunnerConfig: struct {
		MinimumViewports         int   `json:"minimumViewports"`
		RequiredViewportWidthsPx []int `json:"requiredViewportWidthsPx"`
		RequireAxeResults        bool  `json:"requireAxeResults"`
		RequiresCapture          bool  `json:"requiresCapture"`
	}{RequiredViewportWidthsPx: []int{320, 390}}}
	report := componenttests.Report{Results: []componenttests.Result{
		{Stage: componenttests.StageDeclared, Verdict: componenttests.VerdictPassed, Geometry: &components.StoryGeometry{Width: "320px"}},
		{Stage: componenttests.StageDeclared, Verdict: componenttests.VerdictPassed, Geometry: &components.StoryGeometry{Width: "390px"}},
	}}
	result := gates.Result{Status: "measured"}
	applyAppearanceStoryEvidence(&result, "responsive", "controls.button", "rcl:Button", report, definition)
	if result.Status != "measured" || len(result.RunnerError) != 0 || len(result.Findings) != 0 {
		t.Fatalf("complete passing 320/390 evidence should be measured clean: %+v", result)
	}

	result = gates.Result{Status: "measured"}
	report.Results = report.Results[:1]
	applyAppearanceStoryEvidence(&result, "responsive", "controls.button", "rcl:Button", report, definition)
	if result.Status != "unmeasured" || len(result.RunnerError) != 1 || result.RunnerError[0].Code != "catalog.responsive.evidence_missing" {
		t.Fatalf("missing 390 px evidence must be visibly unmeasured: %+v", result)
	}
}

func TestAccessibilityGateRequiresAxeAndReportsViolations(t *testing.T) {
	definition := catalogcoverage.GateDefinition{RunnerConfig: struct {
		MinimumViewports         int   `json:"minimumViewports"`
		RequiredViewportWidthsPx []int `json:"requiredViewportWidthsPx"`
		RequireAxeResults        bool  `json:"requireAxeResults"`
		RequiresCapture          bool  `json:"requiresCapture"`
	}{RequireAxeResults: true}}
	clean := `{"schema":"bas-accessibility-snapshot/v1","snapshot":"main","axe":{"schema":"rcl-axe-results/v1","status":"complete","engine":"axe-core","version":"4.11.4","violations":[]}}`
	report := componenttests.Report{Results: []componenttests.Result{{
		Stage: componenttests.StageDeclared, Version: "1.0.0", Subject: "default", Verdict: componenttests.VerdictPassed,
		Evidence: []componenttests.Evidence{{Kind: "accessibility", AccessibilityJSON: clean}},
	}}}
	result := gates.Result{Status: "measured"}
	applyAppearanceStoryEvidence(&result, "accessibility", "controls.button", "rcl:Button", report, definition)
	if result.Status != "measured" || len(result.RunnerError) != 0 || len(result.Findings) != 0 {
		t.Fatalf("versioned clean axe evidence should pass: %+v", result)
	}

	report.Results[0].Evidence[0].AccessibilityJSON = `{"schema":"bas-accessibility-snapshot/v1","snapshot":"main","axe":{"schema":"rcl-axe-results/v1","status":"complete","engine":"axe-core","version":"4.11.4","violations":[{"id":"button-name","impact":"serious","description":"Button has no name."}]}}`
	result = gates.Result{Status: "measured"}
	applyAppearanceStoryEvidence(&result, "accessibility", "controls.button", "rcl:Button", report, definition)
	if len(result.Findings) != 1 || result.Findings[0].Code != "catalog.accessibility.axe.button-name" {
		t.Fatalf("axe violation should be an attributable finding: %+v", result)
	}

	report.Results[0].Evidence[0].AccessibilityJSON = `{"schema":"bas-accessibility-snapshot/v1","snapshot":"main"}`
	result = gates.Result{Status: "measured"}
	applyAppearanceStoryEvidence(&result, "accessibility", "controls.button", "rcl:Button", report, definition)
	if result.Status != "unmeasured" || len(result.RunnerError) != 1 {
		t.Fatalf("snapshot without axe results must not pass: %+v", result)
	}
}

func TestAccessibilityGateConsumesPersistedStoryAxeAndFailsClosedWhenUnavailable(t *testing.T) {
	definition := catalogcoverage.GateDefinition{RunnerConfig: struct {
		MinimumViewports         int   `json:"minimumViewports"`
		RequiredViewportWidthsPx []int `json:"requiredViewportWidthsPx"`
		RequireAxeResults        bool  `json:"requireAxeResults"`
		RequiresCapture          bool  `json:"requiresCapture"`
	}{RequireAxeResults: true}}
	// This is the exact envelope assembled by the RCL executor from its story
	// result and the separately captured accessibility snapshot.
	clean := `{"schema":"bas-accessibility-snapshot/v1","snapshot":"main","axe":{"schema":"rcl-axe-results/v1","status":"complete","engine":"axe-core","version":"4.11.4","violations":[]}}`
	report := componenttests.Report{Results: []componenttests.Result{{
		Stage: componenttests.StageDeclared, Version: "1.0.0", Subject: "default", Verdict: componenttests.VerdictPassed,
		Evidence: []componenttests.Evidence{{Kind: "accessibility", AccessibilityJSON: clean}},
	}}}
	result := gates.Result{Status: "measured"}
	applyAppearanceStoryEvidence(&result, "accessibility", "controls.button", "rcl:Button", report, definition)
	if result.Status != "measured" || len(result.RunnerError) != 0 || len(result.Findings) != 0 {
		t.Fatalf("persisted clean axe evidence should pass: %+v", result)
	}

	report.Results[0].Evidence[0].AccessibilityJSON = `{"schema":"bas-accessibility-snapshot/v1","snapshot":"main","axe":{"schema":"rcl-axe-results/v1","status":"complete","engine":"axe-core","version":"4.11.4","violations":[{"id":"button-name","impact":"serious","description":"Button needs a name.","targets":["button"]}]}}`
	result = gates.Result{Status: "measured"}
	applyAppearanceStoryEvidence(&result, "accessibility", "controls.button", "rcl:Button", report, definition)
	if len(result.Findings) != 1 || result.Findings[0].Code != "catalog.accessibility.axe.button-name" {
		t.Fatalf("persisted axe violation should become a gate finding: %+v", result)
	}

	report.Results[0].Evidence[0].AccessibilityJSON = `{"schema":"bas-accessibility-snapshot/v1","snapshot":"main","axe":{"schema":"rcl-axe-results/v1","status":"unavailable","reason":"axe-core-not-loaded"}}`
	result = gates.Result{Status: "measured"}
	applyAppearanceStoryEvidence(&result, "accessibility", "controls.button", "rcl:Button", report, definition)
	if result.Status != "unmeasured" || len(result.RunnerError) != 1 {
		t.Fatalf("unavailable persisted axe evidence must stay unmeasured: %+v", result)
	}
}
