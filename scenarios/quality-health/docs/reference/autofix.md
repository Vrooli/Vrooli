# Autofix

Autofix is intentionally narrow in v1. It exists to make safe config repair repeatable without allowing broad automated source rewrites.

## Safety Rules

- Dry-run is the default.
- `--apply` is required for mutation.
- Only deterministic config file edits are allowed in v1.
- Source suppression edits are out of scope.
- Every applied edit must be represented as a preview first.
- Protective comments must be inserted or preserved, not stripped.

## Fix Class Contract

The rule registry is the source of truth for fix class:

- `autofix`: deterministic config repair is allowed, but a finding is marked `autofix_available` only when the fixer can preview a safe edit for the exact file/config shape.
- `detection_only`: Quality Health reports evidence and remediation, but does not mutate source suppressions or semantic code.

Missing files, parse errors, and unsupported config shapes are not autofixable findings even when the rule class is `autofix`.

## Implemented Autofix Candidates

| Rule ID | File | Allowed Fix |
|---|---|---|
| `TS_CONFIG_STRICT` | TypeScript config | Add `strict: true`, `noUncheckedIndexedAccess: true`, and protective comment block. |
| `ESLINT_SAFETY_RULES` | ESLint config | Add the safety-critical header, per-rule comments, and required rule entries when a safe config rewrite is available. |
| `ESLINT_TYPED_CONFIG` | ESLint config | Add typed linting configuration, parser options, and TypeScript import resolver baseline when a safe config rewrite is available. |
| `NODE_BUILD_TYPECHECK` | `package.json` | Ensure `build` runs `tsc --noEmit` before bundling. |
| `TESTING_CONFIG_LINT_STRICT` | `.vrooli/testing.json` | Enable strict Test Genie lint handlers for discovered Node and Go surfaces. |
| `GO_MOD_PRESENT_FOR_API_OR_CLI` | `go.mod` | Create a minimal Go module file for Go surfaces. |
| `GO_LINT_CONFIG_PRESENT` | `.golangci.yml` | Create baseline config for Go surfaces. |
| `GO_LINT_REQUIRED_LINTERS` | `.golangci.yml` | Add missing required linters and remove required linters from `disable`. |
| `MAKEFILE_QUALITY_GATES` | Makefile | Add missing Node and Go format/lint targets. |

## Detection-Only Rules

| Rule ID | Reason |
|---|---|
| `TS_DANGEROUS_PATTERNS` | Source suppressions and unsafe assertions require source-level judgment. |
| `GO_DANGEROUS_PATTERNS` | `//nolint` suppressions require a written reason or manual code repair. |

## Response Shape

Fix responses include:

- target scenario,
- candidate rule IDs,
- files,
- before/after content,
- applied status,
- messages.

## Guardrail

An autofix that makes config values strict but drops the safety-critical comments is a regression and must fail tests.
