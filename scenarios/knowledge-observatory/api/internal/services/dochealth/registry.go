package dochealth

import (
	"fmt"
	"sort"
	"strings"
)

// DOC: docs/internal/SEAMS.md#dochealth
// Check registry. Each documentation-health check declares an applicability
// scope so a single engine can run over a scenario (all checks) or a
// project-level docs path (generic checks only). The default selection is
// "every check applicable to the target"; an explicit --checks filter narrows
// it (e.g. when the full suite is too heavy on a large generic tree).

type checkScope int

const (
	scopeGeneric  checkScope = iota // applies to any documentation tree
	scopeScenario                   // requires a scenario contract (manifest/template)
)

func (s checkScope) String() string {
	if s == scopeScenario {
		return "scenario"
	}
	return "generic"
}

// Stable check identifiers, usable in DocHealthOptions.Checks / --checks.
const (
	checkStructure = "structure"
	checkContent   = "content"
	checkLinks     = "links"
	checkRefs      = "refs"
	checkCommands  = "commands"
	checkManifest  = "manifest"
	checkNumbers   = "numbers"
)

type checkSpec struct {
	Name        string
	Scope       checkScope
	Description string
}

// checkRegistry is the single source of truth for which checks exist and
// whether they are generic or scenario-scoped.
var checkRegistry = []checkSpec{
	{checkStructure, scopeScenario, "Contract placement: missing/misplaced/extra docs, manifest status, health score."},
	{checkContent, scopeGeneric, "Markdown, mermaid, and absolute-path content checks."},
	{checkLinks, scopeGeneric, "Local and external link validation."},
	{checkRefs, scopeGeneric, "Bidirectional reference audit ([CODE:], // DOC:, marked refs)."},
	{checkCommands, scopeGeneric, "Conservative Vrooli-owned command validation in fenced shell snippets."},
	{checkManifest, scopeScenario, "docs/manifest.json coverage (orphaned docs, missing registrations)."},
	{checkNumbers, scopeGeneric, "Drift-prone hardcoded-number / derived-count lint."},
}

func checkScopeOf(name string) (checkScope, bool) {
	for _, c := range checkRegistry {
		if c.Name == name {
			return c.Scope, true
		}
	}
	return scopeGeneric, false
}

func knownCheckNames() []string {
	out := make([]string, 0, len(checkRegistry))
	for _, c := range checkRegistry {
		out = append(out, c.Name)
	}
	sort.Strings(out)
	return out
}

// selection records which checks a caller asked for. An empty selection means
// "all checks applicable to the target".
type selection struct {
	enabled map[string]bool
}

// newSelection validates the requested check names and builds a selection.
func newSelection(checks []string) (selection, error) {
	enabled := make(map[string]bool)
	for _, raw := range checks {
		name := strings.TrimSpace(raw)
		if name == "" {
			continue
		}
		if _, ok := checkScopeOf(name); !ok {
			return selection{}, fmt.Errorf("unknown check %q (known: %s)", name, strings.Join(knownCheckNames(), ", "))
		}
		enabled[name] = true
	}
	if len(enabled) == 0 {
		return selection{}, nil
	}
	return selection{enabled: enabled}, nil
}

// runs reports whether check name should execute for the given target. A
// scenario-scoped check never runs against a non-scenario (generic) target.
func (sel selection) runs(name string, target docTarget) bool {
	scope, ok := checkScopeOf(name)
	if !ok {
		return false
	}
	if scope == scopeScenario && !target.isScenario {
		return false
	}
	if len(sel.enabled) == 0 {
		return true // all applicable
	}
	return sel.enabled[name]
}

// needsMarkdownFiles reports whether any selected, applicable check operates on
// the markdown file set (so the walker only runs when something consumes it).
func (sel selection) needsMarkdownFiles(target docTarget) bool {
	for _, name := range []string{checkContent, checkLinks, checkRefs, checkCommands, checkNumbers, checkManifest} {
		if sel.runs(name, target) {
			return true
		}
	}
	return false
}
