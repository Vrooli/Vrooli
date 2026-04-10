package main

import (
	"context"
	"fmt"
	"reflect"
)

// RuleRunner executes a rule check against a scenario.
type RuleRunner func(ctx context.Context, repoRoot, scenarioName string) RuleResult

// RuleFixer applies an auto-fix for a rule. Returns one or more results
// (e.g. FixMakefileAll returns one per MAKEFILE_* rule ID).
type RuleFixer func(ctx context.Context, repoRoot, scenarioName string, dryRun bool) []FixResult

// RuleDefinition describes a governance rule.
type RuleDefinition struct {
	ID             string `json:"id"`
	Title          string `json:"title"`
	Summary        string `json:"summary"`
	WhyImportant   string `json:"why_important"`
	Category       string `json:"category"`
	Severity       string `json:"severity"`
	DefaultEnabled bool   `json:"default_enabled"`
	Fixable        bool   `json:"fixable"`
}

// RuleEntry is the single source of truth for a rule: its metadata, runner, and fixer.
type RuleEntry struct {
	Definition RuleDefinition
	Runner     RuleRunner
	Fixer      RuleFixer // nil means not fixable
	// FixerGroup groups rules that share the same underlying fix function.
	// Rules in the same group have their fixer called once per scenario,
	// and results are distributed to all group members. Empty means no grouping.
	FixerGroup string
}

// AllRules returns the canonical registry of all governance rules.
// This is the SINGLE SOURCE OF TRUTH — handlers, config, and UI all derive from this.
func AllRules() []RuleEntry {
	return []RuleEntry{
		{
			Definition: RuleDefinition{
				ID:             "GO_CLI_WORKSPACE_INDEPENDENCE",
				Title:          "Go CLI builds without workspace mode",
				Summary:        "Ensures Go-based scenario CLIs build with `GOWORK=off` so they don't depend on a repo-level `go.work`.",
				WhyImportant:   "A single bad `go.work` entry can break every `go` command in the repo. This rule enforces that CLIs are self-contained via their own `go.mod` plus required `replace` directives (e.g. for `packages/proto`) and explicit wiring when a CLI imports API internals.",
				Category:       "go",
				Severity:       "error",
				DefaultEnabled: true,
				Fixable:        true,
			},
			Runner: RunGoCliWorkspaceIndependence,
			Fixer:  FixGoCliWorkspaceIndependence,
		},
		{
			Definition: RuleDefinition{
				ID:             "REACT_VITE_UI_INSTALLS_DEPENDENCIES",
				Title:          "React/Vite UI installs dependencies correctly",
				Summary:        "Ensures React/Vite scenario UIs install `ui/package.json` dependencies using `pnpm install --ignore-workspace` so `pnpm run build` can find tools like `vite`.",
				WhyImportant:   "In a monorepo, `pnpm install` can accidentally behave like a workspace install (and leave `ui/node_modules` missing), causing `vite: not found` during `build-ui`. This rule enforces pnpm specifically (not npm or yarn) because the monorepo uses pnpm workspaces — mixing package managers causes lockfile conflicts and phantom dependencies. The `--ignore-workspace` flag ensures each UI installs its own deps deterministically.",
				Category:       "typescript",
				Severity:       "error",
				DefaultEnabled: true,
				Fixable:        true,
			},
			Runner: RunReactViteUIInstallsDependencies,
			Fixer:  FixReactViteUIInstallsDependencies,
		},
		{
			Definition: RuleDefinition{
				ID:             "MAKEFILE_STRUCTURE",
				Title:          "Makefile follows canonical structure",
				Summary:        "Enforces canonical Makefile structure with STRICT consistency for interoperability. All scenarios must follow identical structure including fmt-go/lint-go/fmt-ui/lint-ui targets.",
				WhyImportant:   "STRICT consistency ensures agents and humans can rely on standard targets across all scenarios. Any deviation breaks tooling and creates confusion.",
				Category:       "makefile",
				Severity:       "error",
				DefaultEnabled: true,
				Fixable:        true,
			},
			Runner:     RunMakefileStructure,
			Fixer:      FixMakefileAll,
			FixerGroup: "makefile",
		},
		{
			Definition: RuleDefinition{
				ID:             "MAKEFILE_LIFECYCLE",
				Title:          "Makefile lifecycle targets use Vrooli CLI",
				Summary:        "Ensures lifecycle targets (start, stop, test, logs, status) call the Vrooli CLI with canonical messaging.",
				WhyImportant:   "Keeps lifecycle orchestration consistent and prevents direct execution regressions. Every scenario must delegate to `vrooli scenario <verb>` so process naming, port allocation, and health checks work correctly.",
				Category:       "makefile",
				Severity:       "error",
				DefaultEnabled: true,
				Fixable:        true,
			},
			Runner:     RunMakefileLifecycle,
			Fixer:      FixMakefileAll,
			FixerGroup: "makefile",
		},
		{
			Definition: RuleDefinition{
				ID:             "MAKEFILE_QUALITY",
				Title:          "Makefile quality targets have proper guards",
				Summary:        "Validates fmt/lint/check targets invoke canonical sub-commands and enforce strict Go formatting/linting logic with proper guards and fallbacks.",
				WhyImportant:   "Keeps code quality workflows discoverable and consistent across scenarios. Guards prevent failures when api/ directory is absent; fallbacks ensure formatting works even without gofumpt.",
				Category:       "makefile",
				Severity:       "warning",
				DefaultEnabled: true,
				Fixable:        true,
			},
			Runner:     RunMakefileQuality,
			Fixer:      FixMakefileAll,
			FixerGroup: "makefile",
		},
	}
}

// AllRuleDefinitions returns just the metadata for all rules.
// Used by handlers that only need definition info (e.g. listing rules).
func AllRuleDefinitions() []RuleDefinition {
	entries := AllRules()
	defs := make([]RuleDefinition, len(entries))
	for i, e := range entries {
		defs[i] = e.Definition
	}
	return defs
}

// ValidateRuleRegistry checks the rule registry for invariant violations.
// Returns an error describing the first problem found, or nil if all rules are valid.
// Called at startup to catch configuration bugs before the server accepts requests.
func ValidateRuleRegistry() error {
	entries := AllRules()
	seenIDs := make(map[string]struct{}, len(entries))
	fixerGroups := make(map[string]uintptr) // group name → fixer function pointer

	for i, entry := range entries {
		id := entry.Definition.ID
		if id == "" {
			return fmt.Errorf("rule at index %d has an empty ID", i)
		}
		if _, dup := seenIDs[id]; dup {
			return fmt.Errorf("duplicate rule ID: %s", id)
		}
		seenIDs[id] = struct{}{}

		if entry.Runner == nil {
			return fmt.Errorf("rule %s has nil Runner", id)
		}
		if entry.Definition.Fixable && entry.Fixer == nil {
			return fmt.Errorf("rule %s is marked Fixable but has nil Fixer", id)
		}
		if !entry.Definition.Fixable && entry.Fixer != nil {
			return fmt.Errorf("rule %s has a Fixer but is not marked Fixable", id)
		}

		// Validate fixer group consistency: all entries in the same group
		// must share the same fixer function.
		if entry.FixerGroup != "" {
			ptr := reflect.ValueOf(entry.Fixer).Pointer()
			if existing, ok := fixerGroups[entry.FixerGroup]; ok {
				if ptr != existing {
					return fmt.Errorf("rule %s is in FixerGroup %q but uses a different Fixer than other members", id, entry.FixerGroup)
				}
			} else {
				fixerGroups[entry.FixerGroup] = ptr
			}
		}
	}

	return nil
}
