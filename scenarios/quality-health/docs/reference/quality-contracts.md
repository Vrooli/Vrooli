# Quality Contracts

Quality Health v1 must preserve or strengthen the current static-quality rules hidden behind Scenario Auditor and Tidiness Manager.

## Contract Model

Contracts are keyed by:

- language,
- framework,
- surface kind,
- tooling.

They are not keyed by scenario template. Templates may seed compliant files, but Quality Health owns policy.

## Migrated Rule IDs

| Rule ID | Scope | Required v1 Behavior |
|---|---|---|
| `TS_CONFIG_STRICT` | TypeScript UI | `strict: true`, `noUncheckedIndexedAccess: true`, parseable JSONC, and required protective phrases. |
| `ESLINT_SAFETY_RULES` | TypeScript ESLint | Required React hooks, no-non-null-assertion, no-explicit-any, unsafe operation, and import cycle rules at or above minimum levels, plus required comments. |
| `TS_DANGEROUS_PATTERNS` | TS/JS source | Count `as any`, broad type assertions, `@ts-ignore`, and non-null assertions with top-file evidence. |
| `ESLINT_TYPED_CONFIG` | TypeScript ESLint | Typed linting through `strictTypeChecked` plus `parserOptions.project` or accepted modern equivalent, and TypeScript import resolver when `import/no-cycle` is enabled. |
| `NODE_BUILD_TYPECHECK` | Node UI/package | Build script must run type checking before bundling or have an equivalent type-check gate. |
| `TESTING_CONFIG_LINT_STRICT` | `.vrooli/testing.json` | Scenario testing metadata must require strict lint/type handlers. |
| `GO_MOD_PRESENT_FOR_API_OR_CLI` | Go API/CLI | Go surfaces require `go.mod`. |
| `GO_LINT_CONFIG_PRESENT` | Go API/CLI | Go surfaces require `.golangci.yml` or equivalent config. |
| `GO_LINT_REQUIRED_LINTERS` | Go API/CLI | Required linters: `errcheck`, `gofumpt`, `govet`, `ineffassign`, `staticcheck`, `typecheck`, `unused`. |
| `MAKEFILE_QUALITY_GATES` | Scenario root | Makefile must expose quality gates for UI and Go formatting/linting. |

## Protective Comment Contract

The comment requirements are policy evidence, not decoration.

`tsconfig.json` must contain these phrases:

- `SAFETY-CRITICAL RULES`
- `DO NOT REMOVE OR WEAKEN`
- `DON'T: Use type assertions (as X)`
- `UI crashes are the #1 production issue`

ESLint config must contain:

- `SAFETY-CRITICAL RULES - DO NOT REMOVE, DISABLE, OR WEAKEN`
- `// CRITICAL:` comments tied to rules-of-hooks, no-non-null-assertion, unsafe operation rules, and import cycle rules.

These comments exist because agents under pressure may try to loosen validation to make tests pass. The correct repair path is to fix code with null checks, optional chaining, nullish coalescing, type guards, and precise types.

## Parity Rule

Quality Health must not remove a finding currently produced by Tidiness Manager / Scenario Auditor unless a documented stronger replacement exists and fixture tests prove the replacement.
