package validation

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"connectrpc.com/connect"

	"ui-health/internal/autofix"
	"ui-health/internal/services/manifestvalidation"

	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
)

// FixProvider is the slice of the ui-health auto-fix registry the handler drives.
// The internal/autofix.Fixer satisfies it.
type FixProvider interface {
	PreviewFixResponse(scenario, root string, ruleIDs []string) (*scenariovalidationv1.FixResponse, error)
	ApplyFixResponse(scenario, root string, ruleIDs []string) (*scenariovalidationv1.FixResponse, error)
	CanFix(root, ruleID, findingPath string) bool
}

var errFixerNotWired = errors.New("ui-health fixer not wired")

// PreviewFix previews ui-health's deterministic remediations for a scenario.
func (h *connectHandler) PreviewFix(_ context.Context, req *connect.Request[scenariovalidationv1.FixRequest]) (*connect.Response[scenariovalidationv1.FixResponse], error) {
	return h.fix(req, false)
}

// ApplyFix writes ui-health's deterministic remediations for a scenario.
func (h *connectHandler) ApplyFix(_ context.Context, req *connect.Request[scenariovalidationv1.FixRequest]) (*connect.Response[scenariovalidationv1.FixResponse], error) {
	return h.fix(req, true)
}

func (h *connectHandler) fix(req *connect.Request[scenariovalidationv1.FixRequest], apply bool) (*connect.Response[scenariovalidationv1.FixResponse], error) {
	if h.deps.Fixer == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errFixerNotWired)
	}
	scenario := strings.TrimSpace(req.Msg.GetScenario())
	if scenario == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("scenario is required"))
	}
	root := h.resolveScenarioRoot(scenario, req.Msg.GetPath())
	var (
		resp *scenariovalidationv1.FixResponse
		err  error
	)
	if apply {
		resp, err = h.deps.Fixer.ApplyFixResponse(scenario, root, req.Msg.GetRuleIds())
	} else {
		resp, err = h.deps.Fixer.PreviewFixResponse(scenario, root, req.Msg.GetRuleIds())
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("ui-health fix %q: %w", scenario, err))
	}
	return connect.NewResponse(resp), nil
}

// resolveScenarioRoot mirrors manifestvalidation's root resolution: an explicit
// request path wins, otherwise the scenario resolves under <repoRoot>/scenarios.
func (h *connectHandler) resolveScenarioRoot(scenario, explicitPath string) string {
	if p := strings.TrimSpace(explicitPath); p != "" {
		return p
	}
	return filepath.Join(h.deps.RepoRoot, "scenarios", scenario)
}

// enrichAutofix stamps each finding with its fix classification and the live
// AutofixAvailable signal. AutofixAvailable is true only when the code maps to a
// registered fixer AND that fixer can currently remediate the specific finding
// (root + location) — so the flag never claims a no-op fix, and it stays
// consistent with the maturity.json declaration the conformance check enforces.
func enrichAutofix(report *manifestvalidation.Report, fixer FixProvider, root string) {
	if report == nil {
		return
	}
	for i := range report.Findings {
		code := report.Findings[i].Code
		class := autofix.FixClassFor(code)
		report.Findings[i].FixClass = string(class)
		report.Findings[i].AutofixAvailable = class.Autofixable() &&
			fixer != nil &&
			fixer.CanFix(root, code, report.Findings[i].Location)
	}
}
