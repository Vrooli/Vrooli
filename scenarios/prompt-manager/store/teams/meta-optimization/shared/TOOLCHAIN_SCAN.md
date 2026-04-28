# Toolchain Scan

Latest scan result from `toolchain-validator`. Supersedes on each heartbeat.

---

## Latest scan: 2026-04-27

**Status: MATERIAL CHANGE since 2026-04-26.** Reference dirty; standards count dropped 36 → 24 (–12); all 9 High type-safety + quality-gates cleared; 2 new Medium violations surfaced (testing-standards + ui-interop). Critical Makefile `required_layout` persists.

### Tools run
- `scenario-auditor standards scan reference-react-vite --wait` → completed (job `standards-da887d1a-bed0-4810-a819-dc2478001346`, 24 violations across 103 files)
- `scenario-auditor security scan reference-react-vite --wait` → completed (job `security-a394ac11-4566-4934-b144-c9e3bf5e1161`, 0 vulns, 228 files)
- `tidiness-manager --auto-start scan reference-react-vite` → light scan (75 files / 13174 lines); lint 58 / type 0; `make lint` and `make type` still `failed (exit 2)`; 30 structural issues open (13 High duplication + 8 Medium + 9 Low)
- `test-genie run-tests reference-react-vite` → `api error (500): scenario tests failed` — still opaque (unchanged)
- `development-toolchain-validator reference list` → was DOWN this heartbeat (connection refused); `validate` subcommand still not shipped (`Error: Unknown command: validate`)

### Reference scenario
- `reference-react-vite` (operator-nominated 2026-04-24, accepted via `dec-1776981723540926630`)

### Violation summary (scenario-auditor standards)
- Critical: 1 (unchanged — `required_layout` on Makefile)
- High: 0 (was 9)
- Medium: 2 (was 5; 3 cleared, 2 new)
- Low: 20 (unchanged: 17 ui-a11y focus-visible + 3 prd-template)
- Info: 1 (unchanged)
- **Total: 24** (was 36)

### Top 3 violations (operator action)
1. **Critical · `required_layout` · `Makefile`** — id `b819dfae-0cd3-4c2c-a877-8d5684b1e777`. Persistent since 2026-04-25. Recommendation: "Add the required resource at Makefile" (still terse — the rule itself doesn't say which resource is missing).
2. **Medium · `testing-standards-v1` · `cli/domains`** (NEW) — id `fd7432bf-8916-4ccc-b1eb-fb2b79dede31`. "Create at least one *_test.go file in cli/domains/ to cover the source files". Surfaced because the type-safety / quality-gates cleanup brought the test-coverage rule into focus.
3. **Medium · `ui-interop-v1` · `ui/src/main.tsx:2`** (NEW) — id `568bbc85-fff3-4513-8ee7-520712626d2e`. "Add `import { initSpatialNav } from '@vrooli/iframe-bridge/spatial';` and call `initSpatialNav();` in main.tsx". Spatial-nav provider missing.

### Tidiness — structural breakdown (open queue: 30 + 58 lint surface)
- 13 High file-level duplication (handlers/repository/mocks/register tests)
- 8 Medium (7 duplication, 1 complexity in `api/config/config.go` cyclomatic 21)
- 9 Low duplication / length / complexity
- 58 lint surface findings (ERROR/WARNING) unchanged
- The 1 High `type_safety` on `ui/src/lib/api.test.ts` reported 2026-04-26 is no longer in the High band — appears resolved by the same cleanup pass that cleared the standards type-safety highs.

### New since last scan
- 2 Medium standards violations introduced by uncovering: `testing-standards-v1` on `cli/domains/`, `ui-interop-v1` (missing `initSpatialNav`) on `ui/src/main.tsx`.
- DTV API was DOWN this heartbeat (connection refused on `:16445`) — first time this happened in the scan window. Did not auto-start; `validate` would be unavailable anyway.

### Resolved since last scan
- All 9 High `type-safety` + `quality-gates` violations from 2026-04-26 cleared (TS_CONFIG_STRICT, noUncheckedIndexedAccess, ESLint typed config, `vite build` gating, testing.json strict flags).
- 3 Medium standards violations (delta from 5 → 2 net of the 2 new mediums; ≥3 mediums removed).
- 1 High `type_safety` tidiness on `ui/src/lib/api.test.ts`.

### Tool-runtime issues observed
- `test-genie run-tests` still returns opaque HTTP 500 — no severity, no failed-test list, no log artifact path. (Same as 2026-04-26.)
- DTV `validate`/`report` subcommands still not shipped. (Same.)
- `tidiness-manager` was stopped at heartbeat start (auto-started cleanly via `--auto-start`). Worth flagging if it persists, but not a tool defect — ambient lifecycle drift.
- DTV scenario was DOWN at heartbeat start. Not investigated further this heartbeat.

### Capability gaps (covered by accepted `dec-1777068259096417622`)
- DTV `validate <reference>` and `report --conflicts | --drift | --maturity | --tool-baselines` not shipped.
- `test-genie run-tests` has no structured failure response.

### Decisions this heartbeat
- 1 new `toolchain-violation` raised (see HANDOFF). Pending count in own contexts after raise: 1.
- `dec-1777154587228516340` already `accepted` 2026-04-26; nothing to supersede there.
- `dec-1777068259096417622` already `accepted`; capability-gap unchanged so no re-raise.
- **Read-only check:** 2 pending team decisions pre-heartbeat (well under 12 ceiling). Own-context cap (4): 1 used.

### Knowledge entries
- `toolchain-scan-2026-04-27` — supersedes `toolchain-scan-2026-04-26`

---

## Same-day re-scan: 2026-04-27 (later)

No material change since earlier today's scan. Re-run via fresh jobs (standards `standards-2589e747-7c38-46f8-98ea-85a3fb795ad0`, security `security-5be031ed-924b-42fc-a752-504d57f5d1be`) confirms identical state:
- Standards: 24 violations (Critical 1 / Med 2 / Low 20 / Info 1) — same shape, same Makefile `required_layout` Critical (id `8f80a41b`), same 2 Mediums (testing-standards-v1, ui-interop-v1).
- Security: 0 vulns (228 files).
- Tidiness: 30 open issues (13 High dup + 8 Med + 9 Low) + 58 lint surface — unchanged. tidiness-manager API base resolution failed on first invocation (CLI couldn't locate API port although scenario was healthy); succeeded on retry. Logged as a low-severity ambient observation, not a decision.
- test-genie: still opaque HTTP 500 — and a CLI rebuild-loop warning was emitted ("test-genie CLI rebuild loop detected") before the 500. Worth a future capability-gap update if it persists.
- DTV: scenario UP this re-run (was DOWN earlier today), but `validate`/`report` still not shipped. No state change in capability.

No new decisions raised. dec-1777327401931026247 still pending and current.
