# Toolchain Scan

Latest scan result from `toolchain-validator`. Supersedes on each heartbeat.

---

## Latest scan: 2026-04-24

**Status: REFERENCE DIRTY — gold-star reference fails first toolchain scan**

### Tools run
- `scenario-auditor standards scan reference-react-vite --wait` → completed (job `standards-18dd3de9-aa0b-4039-bdc5-61b09f65c7f6`)
- `scenario-auditor security scan reference-react-vite --wait` → completed (0 vulns)
- `scenario-auditor scenarios health reference-react-vite` → health 100, 0 vulns
- `tidiness-manager scan reference-react-vite` → light scan; `make lint` and `make type` both `failed (exit 2)`; 2 long files (`api/internal/mocks/repository.go` 572L, `ui/src/App.test.tsx` 509L)
- `test-genie run-tests reference-react-vite` → **`api error (500): scenario tests failed`** (no actionable detail)
- `development-toolchain-validator reference list` → 1 registered (`reference-react-vite`, id `37cbef99-12c7-4513-a78e-0648ec6136a8`)
- `development-toolchain-validator validate <reference>` / `report ...` → **still not shipped** (CLI surface remains `reference`/`connection`/`status`/`configure`)

### Reference scenario
- `reference-react-vite` (operator-nominated 2026-04-24, accepted via `dec-1776981723540926630`)
- Registered in DTV; recorded in `docs/meta-optimization/REFERENCE_SCENARIOS.md` history table

### Violation summary (scenario-auditor standards)
- High: **41**
- Medium: **10**
- Low: **20**
- Info: **1**
- **Total: 72**

### Top rules by count
| Rule | Severity | Count |
|---|---|---|
| `stack-governance` (Makefile structure / lifecycle) | high | 37 |
| `ui-a11y-v1` (missing focus-visible styles) | low | 17 |
| `type-safety` (TS strict mode + protective comments + ESLint typed config) | high | 6 |
| `quality-gates` (build skips type check; testing.json strict not enabled) | high | 4 |
| `prd-template` (unexpected sections) | low | 4 |
| `go-quality` (missing `.golangci.yml` in `api/` and `cli/`) | high | 2 |
| `testing-standards-v1` (missing `cli/domains/` test file) | medium | 1 |
| `ui-interop-v1` (missing spatial-nav init in main.tsx) | medium | 1 |

### Top 3 violations (operator action)
1. **High · `stack-governance` (×37)** — Makefile is missing canonical structure: header incomplete, color constants wrong, required `.PHONY` targets undeclared/undefined (`check`, `lint-go`, `lint-ui`, `fmt-ui`, etc.), `help` text missing required strings, lifecycle targets don't call `vrooli scenario ...`. Tidiness-manager corroborates: `make lint`/`make type` exit 2 because targets aren't defined.
2. **High · `type-safety` (×6, in `ui/tsconfig.json` + `ui/eslint.config.js`)** — `strict: true` not set, `noUncheckedIndexedAccess` missing, no protective comment block, ESLint missing `tseslint.configs.strictTypeChecked` and per-rule CRITICAL comments. 31 dangerous TS patterns flagged across 26 files (mostly `as Type` casts).
3. **High · `quality-gates`** — `ui/package.json` `build` script runs `vite build` without `tsc --noEmit`, and `.vrooli/testing.json` does not enable `lint.handlers.{node_package,go_module}.strict=true`. Linter findings can pass silently.

### Comparison to prior scan (2026-04-23)
- **Prior:** BLOCKED, no reference designated. No violations measured.
- **Now:** Reference designated; first real scan produces 72 violations. This is a baseline, not a regression — but the reference is dirty against the very tools it's meant to be the gold standard for.

### New since last scan
- All 72 violations are new relative to baseline (prior scan was blocked).

### Resolved since last scan
- The blocker on reference designation (`dec-1776981723540926630`) was accepted by the operator on 2026-04-24.

### Tool-runtime issues observed
- **scenario-auditor** rebuilds itself on each invocation (stable rebuild loop fingerprint logged) — non-blocking but noisy.
- **test-genie** `run-tests` returns opaque HTTP 500 with no triage detail; HEARTBEAT.md's documented `test-genie run <reference>` command does not exist (current verb is `run-tests`). The flat 500 cannot be categorized by severity.
- **DTV** still does not implement the `validate`/`report` surface promised in HEARTBEAT.md. Now that a reference exists, this gap is real (previously deferred).

### Capability gaps noticed
- DTV `validate <reference>` and `report --conflicts | --drift | --maturity | --tool-baselines` not shipped. Fallback trio remains the operational path. **Now consequential** because a reference exists.
- `test-genie run-tests` has no structured failure response — a triage tool can't categorize the 500. Either the API needs structured output, or HEARTBEAT.md needs a different verb.

### Decisions this heartbeat
- New: 2 (one `toolchain-violation`, one `capability-gap`; see handoff for IDs)
- Superseded: 0 (prior `dec-1776981723540926630` already accepted by operator)
- Read-only mode: no (4 pending → ≤6 after this run, still under 12 ceiling)

### Knowledge entries
- `toolchain-scan-2026-04-24` — supersedes `toolchain-scan-2026-04-23`
