package dependencyhealth

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"connectrpc.com/connect"

	"github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/dependencygovernance"
	"github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/gomodreconcile"

	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
)

// PreviewFix and ApplyFix implement the shared ScenarioValidationService Fix RPC
// for Go module convergence: each Go surface's go.mod and go.sum are brought in
// line with its import closure (missing in-repo replaces plus the third-party
// requirements and checksums its shared modules pull in). PreviewFix reports the
// exact converged files; ApplyFix writes them. Requirements a consumer would
// newly record pass the dependency governance gate first; a refused surface is
// left untouched and reported with the `deps approved` command that unblocks
// it. This is the single SDA-owned write path for Go module metadata, behind
// `deps reconcile` and the test-genie deterministic-fix aggregate.
func (h *connectHandler) PreviewFix(ctx context.Context, req *connect.Request[scenariovalidationv1.FixRequest]) (*connect.Response[scenariovalidationv1.FixResponse], error) {
	return h.runFix(ctx, req, false)
}

func (h *connectHandler) ApplyFix(ctx context.Context, req *connect.Request[scenariovalidationv1.FixRequest]) (*connect.Response[scenariovalidationv1.FixResponse], error) {
	return h.runFix(ctx, req, true)
}

// Messages that callers key on. A refused or failed surface is never written,
// so it cannot be a FixCandidate (candidates must apply identically to their
// preview); these prefixes carry the outcome instead.
const (
	fixMessageRefused = "refused "
	fixMessageFailed  = "failed "
)

func (h *connectHandler) runFix(ctx context.Context, req *connect.Request[scenariovalidationv1.FixRequest], apply bool) (*connect.Response[scenariovalidationv1.FixResponse], error) {
	if h == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("dependency health handler is not configured"))
	}
	msg := req.Msg
	if msg == nil {
		msg = &scenariovalidationv1.FixRequest{}
	}
	scenario := strings.TrimSpace(msg.GetScenario())
	path := strings.TrimSpace(msg.GetPath())
	if scenario == "" && path == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("scenario or path is required"))
	}
	if !wantsGoModRule(msg.GetRuleIds()) {
		// A caller selected other rule ids; SDA only owns Go module convergence.
		return connect.NewResponse(&scenariovalidationv1.FixResponse{Scenario: scenario}), nil
	}

	repoRoot := filepath.Dir(h.resolveScenariosDir())
	if strings.TrimSpace(repoRoot) == "" || repoRoot == "." {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("repo root could not be resolved"))
	}
	topo, err := gomodreconcile.LoadTopology(repoRoot)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	registry := dependencygovernance.NewRegistry(repoRoot)

	var goMods []string
	switch {
	case path != "" && scenario == "" && fileExists(filepath.Join(path, "go.mod")):
		// A bare module path selects exactly that module. Fleet callers address
		// every module individually, and a module directory may enclose others
		// (the repository root encloses all of them).
		goMods = []string{filepath.Join(path, "go.mod")}
	case path != "":
		goMods = goSurfaceGoMods(path)
	default:
		goMods = goSurfaceGoMods(filepath.Join(h.resolveScenariosDir(), scenario))
	}

	resp := &scenariovalidationv1.FixResponse{Scenario: scenario}
	var messages []string
	for _, goModPath := range goMods {
		conv, err := gomodreconcile.Converge(ctx, goModPath, topo)
		if err != nil {
			// A malformed surface is isolated evidence, not a reason to discard
			// the valid candidates from its sibling surfaces.
			messages = append(messages, fmt.Sprintf("%s%s: %v", fixMessageFailed, relScenarioPath(goModPath), err))
			continue
		}
		candidates, message := h.fixSurface(conv, repoRoot, registry, apply)
		conv.Close()
		resp.Candidates = append(resp.Candidates, candidates...)
		messages = append(messages, message...)
	}
	resp.Applied = apply && len(resp.Candidates) > 0
	if len(resp.Candidates) == 0 && len(messages) == 0 {
		messages = append(messages, "All Go surfaces are converged on their import closure.")
	}
	resp.Messages = append(resp.Messages, messages...)
	return connect.NewResponse(resp), nil
}

// fixSurface gates and (when applying) writes one surface's convergence,
// returning its candidates or the refusal/failure messages.
func (h *connectHandler) fixSurface(conv *gomodreconcile.Convergence, repoRoot string, registry *dependencygovernance.Registry, apply bool) ([]*scenariovalidationv1.FixCandidate, []string) {
	goModPath := conv.GoModPath
	var messages []string
	if len(conv.Evidence) > 0 {
		return nil, append(messages, fmt.Sprintf("%s%s: Go packages under docs/ join this module (%s); fence each evidence tree with a docs/go.mod (docs-evidence convention, docs/package-governance.md) before reconciling", fixMessageRefused, relScenarioPath(goModPath), strings.Join(conv.Evidence, ", ")))
	}
	if !conv.Changed() {
		return nil, nil
	}
	owner, surface := moduleOwner(repoRoot, filepath.Dir(goModPath))
	refusals, err := registry.GateRequirementSync(owner, surface, syncRequirements(conv))
	if err != nil {
		return nil, append(messages, fmt.Sprintf("%s%s: dependency gate: %v", fixMessageFailed, relScenarioPath(goModPath), err))
	}
	if len(refusals) > 0 {
		for _, refusal := range refusals {
			messages = append(messages, fmt.Sprintf("%s%s: %s", fixMessageRefused, relScenarioPath(goModPath), refusal))
		}
		return nil, messages
	}
	if apply {
		if err := conv.Write(); err != nil {
			return nil, append(messages, fmt.Sprintf("%s%s: %v", fixMessageFailed, relScenarioPath(goModPath), err))
		}
	}
	return convergenceCandidates(conv, apply), nil
}

// convergenceCandidates renders one candidate per changed file. The go.mod
// candidate names the replaces and requirement changes; the go.sum candidate
// carries the checksum synchronization.
func convergenceCandidates(conv *gomodreconcile.Convergence, applied bool) []*scenariovalidationv1.FixCandidate {
	rule := goModTidyRuleID
	if len(conv.Missing) > 0 {
		rule = goModReplaceRuleID
	}
	var out []*scenariovalidationv1.FixCandidate
	if conv.ModChanged() {
		out = append(out, &scenariovalidationv1.FixCandidate{
			RuleId:      rule,
			FilePath:    relScenarioPath(conv.GoModPath),
			Description: describeConvergence(conv),
			Before:      conv.GoModBefore,
			After:       conv.GoModAfter,
			Applied:     applied,
		})
	}
	if conv.VendorChanged() {
		before, after := conv.VendorModules()
		description := "Re-vendor the committed vendor/ tree for the converged go.mod (go mod vendor)."
		if preserved := conv.VendorPreserved(); len(preserved) > 0 {
			description += " Preserved non-module vendor inputs: " + strings.Join(preserved, ", ") + "."
		}
		out = append(out, &scenariovalidationv1.FixCandidate{
			RuleId:      rule,
			FilePath:    relScenarioPath(filepath.Join(filepath.Dir(conv.GoModPath), "vendor", "modules.txt")),
			Description: description,
			Before:      before,
			After:       after,
			Applied:     applied,
		})
	}
	if conv.SumChanged() {
		out = append(out, &scenariovalidationv1.FixCandidate{
			RuleId:      rule,
			FilePath:    relScenarioPath(filepath.Join(filepath.Dir(conv.GoModPath), "go.sum")),
			Description: "Synchronize go.sum checksums with the converged requirements.",
			Before:      conv.GoSumBefore,
			After:       conv.GoSumAfter,
			Applied:     applied,
		})
	}
	return out
}

func syncRequirements(conv *gomodreconcile.Convergence) []dependencygovernance.SyncRequirement {
	recorded := conv.Recorded()
	out := make([]dependencygovernance.SyncRequirement, 0, len(recorded))
	for _, change := range recorded {
		out = append(out, dependencygovernance.SyncRequirement{Path: change.Path, Version: change.To, Direct: change.Direct})
	}
	return out
}

// moduleOwner names the governance scope of a module directory: the scenario
// and surface for scenario modules, and the target kind for the rest.
func moduleOwner(repoRoot, moduleDir string) (owner, surface string) {
	rel, err := filepath.Rel(repoRoot, moduleDir)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		// Outside the repository (a scenario being generated or validated in a
		// workspace): the conventional <scenario>/<surface> layout names it.
		return filepath.Base(filepath.Dir(moduleDir)), filepath.Base(moduleDir)
	}
	parts := strings.Split(filepath.ToSlash(rel), "/")
	switch {
	case rel == ".":
		return "vrooli", "control-plane"
	case len(parts) >= 2 && parts[0] == "scenarios":
		if len(parts) == 2 {
			return parts[1], ""
		}
		return parts[1], parts[2]
	case parts[0] == "resources":
		if len(parts) == 1 {
			return "resources", "resource"
		}
		return parts[1], "resource"
	default:
		return parts[len(parts)-1], parts[0]
	}
}

// goSurfaceGoMods returns the go.mod files for a scenario's surfaces in
// deterministic order. Most scenarios use nested surfaces such as api/cli/ui,
// but some Go scenarios keep the API surface at the scenario root.
// Documentation, evidence, fixture, and BAS directories hold retained or
// synthetic modules (including fencing modules such as a docs/go.mod) that are
// never build surfaces; converging them would rewrite evidence.
func goSurfaceGoMods(scenarioDir string) []string {
	var matches []string
	_ = filepath.WalkDir(scenarioDir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if entry.IsDir() {
			if path == scenarioDir {
				return nil
			}
			switch entry.Name() {
			case ".git", "node_modules", "vendor", "data", "dist", "build", ".cache", "phase-cache", "coverage",
				"docs", "testdata", "evidence", "fixtures", ".vrooli", "bas":
				return filepath.SkipDir
			}
			return nil
		}
		if entry.Name() == "go.mod" {
			matches = append(matches, path)
		}
		return nil
	})
	sort.Strings(matches)
	return matches
}

// wantsGoModRule reports whether the requested rule ids select Go module
// convergence. Every Go module rule resolves to the same atomic per-surface
// convergence, so selecting any of them converges the whole surface.
func wantsGoModRule(ruleIDs []string) bool {
	if len(ruleIDs) == 0 {
		return true
	}
	for _, id := range ruleIDs {
		switch strings.TrimSpace(strings.ToLower(id)) {
		case goModReplaceRuleID, goModTidyRuleID:
			return true
		}
	}
	return false
}

func describeConvergence(conv *gomodreconcile.Convergence) string {
	var parts []string
	for _, m := range conv.Missing {
		if m.AddRequire {
			parts = append(parts, "require "+m.Module+" v0.0.0 and replace "+m.Module+" => "+m.RelPath)
			continue
		}
		parts = append(parts, "replace "+m.Module+" => "+m.RelPath)
	}
	for _, module := range conv.Dropped {
		parts = append(parts, "drop unused replace "+module)
	}
	for _, change := range conv.Requirements {
		parts = append(parts, change.String())
	}
	if len(parts) == 0 {
		return "Tidy go.mod formatting."
	}
	return "Converge go.mod: " + strings.Join(parts, "; ")
}
