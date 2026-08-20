package dependencyhealth

import (
	"context"
	"errors"
	"path/filepath"
	"sort"
	"strings"

	"connectrpc.com/connect"

	"github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/gomodreconcile"

	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
)

// PreviewFix and ApplyFix implement the shared ScenarioValidationService Fix RPC
// for the dependency.gomod.replace.missing fix-class: they add the local replace
// directives a Go surface needs for its in-repo module requires. PreviewFix is a
// dry-run (before/after only); ApplyFix writes and tidies to a fixpoint. This is
// the single SDA-owned reconcile path that the `deps reconcile` CLI verb and the
// test-genie deterministic-fix aggregate both flow through.
func (h *connectHandler) PreviewFix(ctx context.Context, req *connect.Request[scenariovalidationv1.FixRequest]) (*connect.Response[scenariovalidationv1.FixResponse], error) {
	return h.runFix(ctx, req, false)
}

func (h *connectHandler) ApplyFix(ctx context.Context, req *connect.Request[scenariovalidationv1.FixRequest]) (*connect.Response[scenariovalidationv1.FixResponse], error) {
	return h.runFix(ctx, req, true)
}

func (h *connectHandler) runFix(ctx context.Context, req *connect.Request[scenariovalidationv1.FixRequest], apply bool) (*connect.Response[scenariovalidationv1.FixResponse], error) {
	if h == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("dependency health handler is not configured"))
	}
	msg := req.Msg
	if msg == nil {
		msg = &scenariovalidationv1.FixRequest{}
	}
	scenario := strings.TrimSpace(msg.GetScenario())
	if scenario == "" && strings.TrimSpace(msg.GetPath()) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("scenario or path is required"))
	}
	if !wantsGoModReplaceRule(msg.GetRuleIds()) {
		// A caller selected other rule ids; SDA only owns the go.mod replace class.
		return connect.NewResponse(&scenariovalidationv1.FixResponse{Scenario: scenario, Applied: apply}), nil
	}

	repoRoot := filepath.Dir(h.resolveScenariosDir())
	if strings.TrimSpace(repoRoot) == "" || repoRoot == "." {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("repo root could not be resolved"))
	}
	topo, err := gomodreconcile.LoadTopology(repoRoot)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	scenarioDir := strings.TrimSpace(msg.GetPath())
	if scenarioDir == "" {
		scenarioDir = filepath.Join(h.resolveScenariosDir(), scenario)
	}

	resp := &scenariovalidationv1.FixResponse{Scenario: scenario, Applied: apply}
	var messages []string
	for _, goModPath := range goSurfaceGoMods(scenarioDir) {
		var cand *gomodreconcile.Candidate
		var cerr error
		if apply {
			cand, cerr = gomodreconcile.ApplySurface(ctx, goModPath, topo)
		} else {
			cand, cerr = gomodreconcile.PreviewSurface(ctx, goModPath, topo)
		}
		if cerr != nil {
			return nil, connect.NewError(connect.CodeInternal, cerr)
		}
		if cand == nil {
			continue
		}
		resp.Candidates = append(resp.Candidates, &scenariovalidationv1.FixCandidate{
			RuleId:      goModReplaceRuleID,
			FilePath:    relScenarioPath(cand.GoModPath),
			Description: describeMissing(cand.Missing),
			Before:      cand.Before,
			After:       cand.After,
			Applied:     cand.Applied,
		})
	}
	if len(resp.Candidates) == 0 {
		messages = append(messages, "All Go surfaces already declare local replaces for their in-repo module requires.")
	}
	resp.Messages = messages
	return connect.NewResponse(resp), nil
}

// goSurfaceGoMods returns the go.mod files for a scenario's surfaces in
// deterministic order. Most scenarios use nested surfaces such as api/cli/ui,
// but some Go scenarios keep the API surface at the scenario root.
func goSurfaceGoMods(scenarioDir string) []string {
	matches, err := filepath.Glob(filepath.Join(scenarioDir, "*", "go.mod"))
	if err != nil {
		return nil
	}
	if fileExists(filepath.Join(scenarioDir, "go.mod")) {
		matches = append(matches, filepath.Join(scenarioDir, "go.mod"))
	}
	sort.Strings(matches)
	return matches
}

func wantsGoModReplaceRule(ruleIDs []string) bool {
	if len(ruleIDs) == 0 {
		return true
	}
	for _, id := range ruleIDs {
		if strings.EqualFold(strings.TrimSpace(id), goModReplaceRuleID) {
			return true
		}
	}
	return false
}

func describeMissing(missing []gomodreconcile.MissingReplace) string {
	if len(missing) == 0 {
		return "Add local replaces for in-repo module requires."
	}
	parts := make([]string, 0, len(missing))
	for _, m := range missing {
		if m.AddRequire {
			parts = append(parts, "require "+m.Module+" v0.0.0 and replace "+m.Module+" => "+m.RelPath)
			continue
		}
		parts = append(parts, "replace "+m.Module+" => "+m.RelPath)
	}
	return "Add " + strings.Join(parts, "; ")
}
