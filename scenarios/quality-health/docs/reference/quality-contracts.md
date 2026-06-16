# Quality Contracts

Quality Health v1 must preserve or strengthen the current static-quality rules hidden behind Scenario Auditor and Tidiness Manager.

## Contract Model

Contracts are **language-first**. Applicability is decided purely by:

- language (primary key),
- framework (optional narrowing),
- surface kind (optional narrowing),
- tooling.

They are **never** keyed by surface name (`ui/`, `api/`, `cli/`) and never by scenario template. Templates may seed compliant files, but Quality Health owns policy. Code Facts lets scenarios lay themselves out however they want; rule routing follows the detected language/tooling, so a TypeScript surface named `worker` and a Go surface named `edge` are evaluated exactly like ones named `ui`/`api`.

Routing lives in **one** registry-backed helper (`applicableContracts`). Surface-level packs declare a non-empty `Language`; the scenario-level pack is the only language-wildcard pack and only attaches to the synthetic `scenario` surface. Adding coverage for a new language requires only a registry entry plus an evaluator — no edits to dispatch branches.

### Contract Packs

| Contract ID | Applies To | Rules |
|---|---|---|
| `typescript-static-quality` | any `typescript` or `javascript` surface (framework/kind wildcard) | `TS_CONFIG_STRICT`, `ESLINT_SAFETY_RULES`, `TS_DANGEROUS_PATTERNS`, `ESLINT_TYPED_CONFIG`, `NODE_BUILD_TYPECHECK` |
| `go-static-quality` | any `go` surface (kind wildcard) | `GO_MOD_PRESENT_FOR_API_OR_CLI`, `GO_LINT_CONFIG_PRESENT`, `GO_LINT_REQUIRED_LINTERS` |
| `scenario-quality-gates` | the scenario root | `TESTING_CONFIG_LINT_STRICT`, `MAKEFILE_QUALITY_GATES` |

A `javascript` surface is folded onto the `typescript` pack because those rules evaluate shared tooling (eslint, `package.json` scripts, `.ts/.js` suppressions). `TS_CONFIG_STRICT` self-skips on a JavaScript-only surface (no `tsconfig.json`); a `typescript` surface missing its `tsconfig.json` still errors.

## Migrated Rule IDs

| Rule ID | Scope | Required v1 Behavior |
|---|---|---|
| `TS_CONFIG_STRICT` | any TypeScript surface | `strict: true`, `noUncheckedIndexedAccess: true`, parseable JSONC, and required protective phrases. Self-skips for JavaScript-only surfaces. |
| `ESLINT_SAFETY_RULES` | any TS/JS surface | Required React hooks, no-non-null-assertion, no-explicit-any, unsafe operation, and import cycle rules at or above minimum levels, plus required comments. |
| `TS_DANGEROUS_PATTERNS` | TS/JS source | Count `as any`, broad type assertions, `@ts-ignore`, and non-null assertions with top-file evidence. |
| `ESLINT_TYPED_CONFIG` | any TS/JS surface | Typed linting through `strictTypeChecked` plus `parserOptions.project` or accepted modern equivalent, and TypeScript import resolver when `import/no-cycle` is enabled. |
| `NODE_BUILD_TYPECHECK` | any Node package surface | Build script must run type checking before bundling or have an equivalent type-check gate. |
| `TESTING_CONFIG_LINT_STRICT` | `.vrooli/testing.json` | Scenario testing metadata must require strict lint/type handlers. |
| `GO_MOD_PRESENT_FOR_API_OR_CLI` | any Go surface | Go surfaces require `go.mod`. |
| `GO_LINT_CONFIG_PRESENT` | any Go surface | Go surfaces require `.golangci.yml` or equivalent config. |
| `GO_LINT_REQUIRED_LINTERS` | any Go surface | Required linters: `errcheck`, `gofumpt`, `govet`, `ineffassign`, `staticcheck`, `typecheck`, `unused`. |
| `MAKEFILE_QUALITY_GATES` | Scenario root | Makefile must expose quality gates for Node and Go formatting/linting. |
| `QUALITY_COVERAGE_GAP` | any uncovered surface | Info-only honesty signal: a discovered surface matched no contract pack. See [Coverage Gaps](#coverage-gaps). |

## Coverage Gaps

A surface that Code Facts discovers but no contract pack covers (e.g. a `python` or `rust` surface today) is reported honestly rather than silently green:

- its `ContractEvaluation.Status` is `uncovered` (with empty `contract_id`), distinct from `passed`,
- it carries one `QUALITY_COVERAGE_GAP` `info` finding (`category = coverage`) naming the surface id and detected language,
- the run summary appends `N surface(s) uncovered`,
- maturity is capped at **L2** while any surface is uncovered (you cannot claim "strict contracts satisfied" for a surface that received zero evaluation).

Coverage gaps are **informational**: they never flip run status to `failed` and never fail the Test Genie quality phase. The remediation is to file a capability-gap so a contract pack is added for that language.

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
