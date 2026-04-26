# Toolchain Scan

Latest scan result from `toolchain-validator`. Supersedes on each heartbeat.

---

## Latest scan: 2026-04-26

**Status: NO MATERIAL CHANGE since 2026-04-25.** Reference dirty; same shape as prior heartbeat.

### Tools run
- `scenario-auditor standards scan reference-react-vite --wait` → completed (job `standards-e890c6a2-d27a-4164-83cd-bb1dccc7fc80`, 36 violations across 103 files — identical to 2026-04-25)
- `scenario-auditor security scan reference-react-vite --wait` → completed (job `security-7d50f3d6-522c-46ee-baf6-139b2513271e`, 0 vulns, 228 files)
- `tidiness-manager scan reference-react-vite` → light scan (75 files / 13174 lines); lint issues 58, type 0; `make lint` and `make type` still `failed (exit 2)`; 92 open issues in total (14 High + 9 Medium + 11 Low structural + 58 ERROR/WARNING lint surface)
- `test-genie run-tests reference-react-vite` → `api error (500): scenario tests failed` — still opaque, no triage detail (unchanged)
- `development-toolchain-validator reference list` → 1 registered (`reference-react-vite`, id `37cbef99-12c7-4513-a78e-0648ec6136a8`); `validate`/`report` subcommands still not shipped

### Reference scenario
- `reference-react-vite` (operator-nominated 2026-04-24, accepted via `dec-1776981723540926630`)

### Violation summary (scenario-auditor standards)
- Critical: 1 (unchanged — `required_layout` on Makefile)
- High: 9 (unchanged)
- Medium: 5 (unchanged)
- Low: 20 (unchanged)
- Info: 1 (unchanged)
- **Total: 36** (unchanged)

### Top 3 violations (operator action — unchanged from 2026-04-25)
1. **Critical · `required_layout` · `Makefile`** — id `1c96109f-07e6-4f0e-a2db-8d7815ad17d6`. Persistent since the 2026-04-25 cleanup that cleared the 37 stack-governance highs introduced this Critical instead.
2. **High · `type-safety` (×6, in `ui/tsconfig.json` + `ui/eslint.config.js`)** — `strict: true` not set, `noUncheckedIndexedAccess` missing, no protective comment block, ESLint typed config incomplete. Auto-fix exists: `scenario-auditor fix reference-react-vite --rules TS_CONFIG_STRICT`.
3. **High · `quality-gates` (×4)** — `ui/package.json` `build` skips `tsc --noEmit`; `.vrooli/testing.json` has `lint.handlers.{node_package,go_module}.enabled=false`.

### Tidiness — accurate breakdown (corrects yesterday's "20 issues" framing)
- 14 High structural (13 file-level duplication on handlers/repository/mocks/register tests + 1 type_safety on `ui/src/lib/api.test.ts`)
- 9 Medium (7 duplication, 1 complexity in `api/config/config.go` cyclomatic 21, 1 type_safety on `ui/src/consts/selectors.ts`)
- 11 Low duplication
- 58 lint findings surfaced as ERROR (2) / WARNING (56) categories
- **Total open: 92** — yesterday's "20 open" was a `--limit 20` artifact; the underlying count was already this size

### New since last scan
- **None.** Standards counts identical, security identical, tidiness structural counts unchanged, lint surface unchanged, test-genie unchanged, DTV surface unchanged.

### Resolved since last scan
- **None.**

### Tool-runtime issues observed (unchanged)
- `test-genie run-tests` still returns opaque HTTP 500 — no severity, no failed-test list, no log artifact path.
- DTV `validate`/`report` surface still not shipped despite reference being registered.

### Capability gaps (covered by accepted `dec-1777068259096417622`)
- DTV `validate <reference>` and `report --conflicts | --drift | --maturity | --tool-baselines` not shipped.
- `test-genie run-tests` has no structured failure response.

### Decisions this heartbeat
- **None.** Stop-condition: no material change since 2026-04-25.
- Prior `dec-1777154587228516340` (toolchain-violation) is now `accepted` — operator picked it up. Pending count in own contexts: 0.
- Prior `dec-1777068259096417622` (capability-gap) is `accepted` — operator picked it up.
- **Read-only check:** 3 pending team decisions (well under 12 ceiling). Own-context cap (4 in toolchain-violation + capability-gap): currently 0 — under limit.

### Knowledge entries
- `toolchain-scan-2026-04-26` — supersedes `toolchain-scan-2026-04-25` (records "no change" so the timeline isn't ambiguous about whether a heartbeat ran)
