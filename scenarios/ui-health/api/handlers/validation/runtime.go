package validation

import (
	"context"

	"ui-health/internal/codefacts"
	"ui-health/internal/services/manifestvalidation"
	"ui-health/internal/uiruntime"
)

// runtimeFindings runs the runtime/render group when execution is requested and
// applicable, returning any findings to fold into the single report.
//
// A skipped collector is never a runtime success. Static-only and unconfigured
// execution paths therefore return an explicit not-evaluated observation. This
// preserves the zero-BAS property of static validation while preventing maturity
// from treating an absent runtime group as a clean render.
//
// Code Facts is consulted only on the execution path, so a `--static-only`
// validation makes zero calls to Code Facts or BAS.
func (h *connectHandler) runtimeFindings(ctx context.Context, scenario, scenarioDir string, staticOnly bool) []manifestvalidation.Finding {
	if staticOnly {
		return []manifestvalidation.Finding{{
			Severity:   manifestvalidation.SeverityInfo,
			Code:       "runtime_not_evaluated_static_only",
			Location:   scenarioDir,
			Message:    "runtime render was not evaluated because this validation was requested in static-only mode",
			Suggestion: "Run validation with execution enabled to collect desktop and mobile screenshot, DOM, layout, viewport, and interaction evidence.",
		}}
	}
	if h.deps.Runtime == nil {
		return []manifestvalidation.Finding{{
			Severity:   manifestvalidation.SeverityInfo,
			Code:       "runtime_not_evaluated_unconfigured",
			Location:   scenarioDir,
			Message:    "runtime render was not evaluated because no runtime evidence collector is configured",
			Suggestion: "Configure Browser Automation Studio runtime collection before treating runtime render maturity as complete.",
		}}
	}
	facts := h.describeFacts(ctx, scenario, scenarioDir)
	if !facts.HasUI {
		return nil
	}
	return h.deps.Runtime.Check(ctx, uiruntime.Input{
		Scenario:    scenario,
		ScenarioDir: scenarioDir,
		Facts:       facts,
	})
}

// describeFacts resolves the scenario's UI facts via Code Facts, degrading to a
// filesystem probe when no Describer is wired or Code Facts is unreachable.
func (h *connectHandler) describeFacts(ctx context.Context, scenario, scenarioDir string) codefacts.Facts {
	describer := h.deps.CodeFacts
	if describer == nil {
		describer = codefacts.New()
	}
	return describer.Describe(ctx, scenario, scenarioDir)
}
