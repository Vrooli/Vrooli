# Toolchain Scan

Latest scan result from `toolchain-validator`. Supersedes on each heartbeat.

---

## Latest scan: 2026-04-25

**Status: REFERENCE DIRTY — improvement on Makefile/stack-governance, but a NEW Critical surfaced and tidiness now exposes 20 duplication issues**

### Tools run
- `scenario-auditor standards scan reference-react-vite --wait` → completed (job `standards-b9cfe21b-7aed-4e7a-a507-aeab60a7c91e`, 36 violations across 103 files)
- `scenario-auditor security scan reference-react-vite --wait` → completed (0 vulns, 228 files)
- `tidiness-manager scan reference-react-vite` → light scan (75 files / 13174 lines); **lint issues: 58** (was 0 yesterday); type issues: 0; `make lint` and `make type` still `failed (exit 2)`; 20 open issues now visible (mostly file-level duplication)
- `test-genie run-tests reference-react-vite` → **`api error (500): scenario tests failed`** — still opaque, no triage detail (unchanged)
- `development-toolchain-validator reference list` → 1 registered (`reference-react-vite`, id `37cbef99-12c7-4513-a78e-0648ec6136a8`); `validate`/`report` subcommands still not shipped

### Reference scenario
- `reference-react-vite` (operator-nominated 2026-04-24, accepted via `dec-1776981723540926630`)

### Violation summary (scenario-auditor standards)
- **Critical: 1** ← **NEW** (was 0 yesterday)
- High: **9** (was 41 — large improvement)
- Medium: **5** (was 10)
- Low: **20** (unchanged)
- Info: **1** (unchanged)
- **Total: 36** (was 72)

### Top rules by count
| Rule | Severity | Count | Δ |
|---|---|---|---|
| `ui-a11y-v1` (missing focus-visible styles) | low | 17 | 0 |
| `type-safety` (TS strict + protective comments + ESLint typed config) | high | 6 | 0 |
| `quality-gates` (build skips type check; testing.json strict not enabled) | high (4) + medium (1) | 5 | +1 medium |
| `prd-template` (unexpected PRD sections) | low | 4 | 0 |
| `go-quality` (missing `.golangci.yml` in `api/` and `cli/`) | high | 2 | 0 |
| `testing-standards-v1` (missing `cli/domains/` test) | medium | 1 | 0 |
| `ui-interop-v1` (missing spatial-nav init in main.tsx) | medium | 1 | 0 |
| **`stack-governance` (Makefile)** | high | **0** | **−37** |
| **`required_layout` (Makefile)** | **critical** | **1** | **+1** |

### Top 3 violations (operator action)
1. **Critical · `required_layout` · `Makefile`** — "Add the required resource at Makefile / Scenario Required Structure". This is **NEW** since the 2026-04-24 scan. The remediation of the 37 stack-governance findings appears to have rewritten the Makefile in a way that now trips required_layout instead. Net: same root file, severity escalated from `high` to `critical`.
2. **High · `type-safety` (×6, in `ui/tsconfig.json` + `ui/eslint.config.js`)** — `strict: true` not set, `noUncheckedIndexedAccess` missing, no protective comment block, ESLint typed config incomplete. Auto-fix exists: `scenario-auditor fix reference-react-vite --rules TS_CONFIG_STRICT`. **Unchanged.**
3. **High · `quality-gates` (×4)** — `ui/package.json` `build` script runs `vite build` without `tsc --noEmit`; `.vrooli/testing.json` has `lint.handlers.{node_package,go_module}.enabled=false` and `strict=false`. **Unchanged.**

### New since last scan
- **+1 Critical** `required_layout` violation on `Makefile`.
- **+58 lint issues** surfaced by `tidiness-manager` (was 0 yesterday). Likely a tooling-surface change (yesterday lint plumbing returned 0 because make lint exit 2 short-circuited; today it's hitting underlying linters), not a code regression.
- **+20 open tidiness issues** visible via `tidiness-manager issues`: 14 High duplication (handlers, repository, mocks, register tests, `ui/src/lib/api.test.ts` 18 dangerous type-safety patterns), 6 Medium (duplication + complexity in `api/config/config.go` cyclomatic 21).

### Resolved since last scan
- **stack-governance Makefile (37 high)** — cleared (was the dominant rule yesterday). Likely the Makefile was overhauled, but the overhaul tripped `required_layout` instead.
- High count dropped 41 → 9.
- Total dropped 72 → 36.

### Tool-runtime issues observed (unchanged)
- **scenario-auditor** rebuild loop fingerprint still logged on every invocation (non-blocking, noisy).
- **test-genie** `run-tests` still returns opaque HTTP 500 — no severity, no failed-test list, no log artifact path.
- **DTV** `validate`/`report` surface still not shipped despite reference being registered.

### Capability gaps (unchanged)
- DTV `validate <reference>` and `report --conflicts | --drift | --maturity | --tool-baselines` not shipped (covered by pending `dec-1777068259096417622`).
- `test-genie run-tests` has no structured failure response (covered by same pending decision).

### Decisions this heartbeat
- **Superseded** `dec-1777068246086430656` (toolchain-violation) → replaced with fresh take that reflects the actual current shape (1 critical + 9 high, not 41 high).
- **Capability-gap** `dec-1777068259096417622` left as-is — content is still current.
- **Read-only check:** 4 pending team decisions (well under 12 ceiling). Own-context cap (4 in toolchain-violation + capability-gap): currently 2 — under limit.

### Knowledge entries
- `toolchain-scan-2026-04-25` — supersedes `toolchain-scan-2026-04-24`
