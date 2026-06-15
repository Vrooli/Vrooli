# Autofix

Autofix is intentionally narrow in v1. It exists to make safe config repair repeatable without allowing broad automated source rewrites.

## Safety Rules

- Dry-run is the default.
- `--apply` is required for mutation.
- Only deterministic config file edits are allowed in v1.
- Source suppression edits are out of scope.
- Every applied edit must be represented as a preview first.
- Protective comments must be inserted or preserved, not stripped.

## Implemented Autofix Candidates

| Rule ID | File | Allowed Fix |
|---|---|---|
| `TS_CONFIG_STRICT` | TypeScript config | Add `strict: true`, `noUncheckedIndexedAccess: true`, and protective comment block. |

## Planned Autofix Candidates

These are intentionally deferred until their config writers can preserve local structure and comments safely.

| Rule ID | File | Allowed Fix |
|---|---|---|
| `ESLINT_SAFETY_RULES` | ESLint config | Add missing safety header/per-rule comments and required rule entries when the config shape is supported. |
| `ESLINT_TYPED_CONFIG` | ESLint config | Add typed linting configuration only when the config format is recognized. |
| `GO_LINT_CONFIG_PRESENT` | `.golangci.yml` | Create baseline config for Go surfaces. |
| `GO_LINT_REQUIRED_LINTERS` | `.golangci.yml` | Add missing required linters. |
| `MAKEFILE_QUALITY_GATES` | Makefile | Add missing gate targets only if the scenario Makefile matches the generated structure. |

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
