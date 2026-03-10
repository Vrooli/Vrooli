package main

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

func AllRuleDefinitions() []RuleDefinition {
	return []RuleDefinition{
		{
			ID:             "GO_CLI_WORKSPACE_INDEPENDENCE",
			Title:          "Go CLI builds without workspace mode",
			Summary:        "Ensures Go-based scenario CLIs build with `GOWORK=off` so they don't depend on a repo-level `go.work`.",
			WhyImportant:   "A single bad `go.work` entry can break every `go` command in the repo. This rule enforces that CLIs are self-contained via their own `go.mod` plus required `replace` directives (e.g. for `packages/proto`) and explicit wiring when a CLI imports API internals.",
			Category:       "go",
			Severity:       "error",
			DefaultEnabled: true,
			Fixable:        true,
		},
		{
			ID:             "REACT_VITE_UI_INSTALLS_DEPENDENCIES",
			Title:          "React/Vite UI installs dependencies correctly",
			Summary:        "Ensures React/Vite scenario UIs install `ui/package.json` dependencies in a way that makes `pnpm run build` actually find tools like `vite`.",
			WhyImportant:   "In a monorepo, `pnpm install` can accidentally behave like a workspace install (and leave `ui/node_modules` missing), causing `vite: not found` during `build-ui`. This rule enforces an explicit lifecycle install command (`pnpm install --ignore-workspace`) so setup is deterministic.",
			Category:       "typescript",
			Severity:       "error",
			DefaultEnabled: true,
			Fixable:        true,
		},
		{
			ID:             "MAKEFILE_STRUCTURE",
			Title:          "Makefile follows canonical structure",
			Summary:        "Enforces canonical Makefile structure with STRICT consistency for interoperability. All scenarios must follow identical structure including fmt-go/lint-go/fmt-ui/lint-ui targets.",
			WhyImportant:   "STRICT consistency ensures agents and humans can rely on standard targets across all scenarios. Any deviation breaks tooling and creates confusion.",
			Category:       "makefile",
			Severity:       "error",
			DefaultEnabled: true,
			Fixable:        true,
		},
		{
			ID:             "MAKEFILE_LIFECYCLE",
			Title:          "Makefile lifecycle targets use Vrooli CLI",
			Summary:        "Ensures lifecycle targets (start, stop, test, logs, status) call the Vrooli CLI with canonical messaging.",
			WhyImportant:   "Keeps lifecycle orchestration consistent and prevents direct execution regressions. Every scenario must delegate to `vrooli scenario <verb>` so process naming, port allocation, and health checks work correctly.",
			Category:       "makefile",
			Severity:       "error",
			DefaultEnabled: true,
			Fixable:        true,
		},
		{
			ID:             "MAKEFILE_QUALITY",
			Title:          "Makefile quality targets have proper guards",
			Summary:        "Validates fmt/lint/check targets invoke canonical sub-commands and enforce strict Go formatting/linting logic with proper guards and fallbacks.",
			WhyImportant:   "Keeps code quality workflows discoverable and consistent across scenarios. Guards prevent failures when api/ directory is absent; fallbacks ensure formatting works even without gofumpt.",
			Category:       "makefile",
			Severity:       "warning",
			DefaultEnabled: true,
			Fixable:        true,
		},
	}
}
