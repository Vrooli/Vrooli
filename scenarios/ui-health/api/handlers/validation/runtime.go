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
// It is a no-op (nil findings) when:
//   - staticOnly is set (the caller asked for static checks only — no BAS, no
//     auto-start);
//   - no runtime Checker is wired;
//   - the scenario has no UI surface (nothing to render).
//
// Code Facts is consulted only on the execution path, so a `--static-only`
// validation makes zero calls to Code Facts or BAS.
func (h *connectHandler) runtimeFindings(ctx context.Context, scenario, scenarioDir string, staticOnly bool) []manifestvalidation.Finding {
	if staticOnly || h.deps.Runtime == nil {
		return nil
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
