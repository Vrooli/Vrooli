# Toolchain Audit

Latest audit result from `toolchain-validator`. Supersedes on each heartbeat. (Renamed from `Toolchain Scan` 2026-05-03 to align with the audit-vs-scan naming rule — the work is adversarial / compliance-shaped, producing `toolchain-violation` decisions, not survey-shaped observations.)

---

## Latest scan: 2026-04-28

**Status: MATERIAL CHANGE since 2026-04-27 — but the change is in the *report*, not the reference scenario.** Standards count jumped 24 → 36 (+12). The 9 High `type-safety` + `quality-gates` violations that 2026-04-27 reported as "cleared" are present in today's scan. Verified by reading the actual reference files: `ui/tsconfig.json` has no `strict` and no `noUncheckedIndexedAccess`; `ui/package.json` build script is bare `vite build` (no typecheck); `.vrooli/testing.json` has no `lint.handlers.{go_module,node_package}.strict = true`; no `api/.golangci.yml` or `cli/.golangci.yml`. Reference last touched ≥3 days ago — no regression in reference. Yesterday's scan was a false-negative. Today's scan is the truth.

### Tools run
- `scenario-auditor standards scan reference-react-vite --wait` → completed (job `standards-cfb1529e-5958-418f-b2a5-3bab9976303c`, 36 violations across 103 files); re-run `standards-139a9acf-c5f6-4ec5-bcc6-5243af735814` confirmed identical 36-violation shape.
- `scenario-auditor security scan reference-react-vite --wait` → completed (job `security-6f1f9fc8-f27d-493f-9129-b6060a9d97f5`, 0 vulns, 228 files).
- `tidiness-manager scan reference-react-vite` → light scan; lint 58 / type 0; `make lint` and `make type` still `failed (exit 2)`; 30 structural issues open (13 High duplication + 8 Medium + 9 Low) — unchanged.
- `test-genie run-tests reference-react-vite` → `api error (500): scenario tests failed` — still opaque (unchanged).
- `development-toolchain-validator validate reference-react-vite` → `Error: Unknown command: validate` (still not shipped). DTV scenario UP this heartbeat.

### Reference scenario
- `reference-react-vite` (operator-nominated 2026-04-24, accepted via `dec-1776981723540926630`)

### Violation summary (scenario-auditor standards)
- Critical: 1 (unchanged — `required_layout` on Makefile, id `492b2c57`)
- High: 9 (was 0 in 2026-04-27 scan; reality unchanged from 2026-04-26)
  - 6 `type-safety` (TS_CONFIG_STRICT family — strict, noUncheckedIndexedAccess, protective comment block on `ui/tsconfig.json`)
  - 4 `quality-gates` (testing.json strict flags + `ui/package.json` build typecheck)
  - 2 `go-quality` (missing `api/.golangci.yml` and `cli/.golangci.yml`)
  - 1 `type-safety` ESLINT_TYPED_CONFIG on `ui/eslint.config.js`
  - (count is 13 by-rule; severity tally rolls some sub-rules into 9; the 36-total figure is the canonical number)
- Medium: 5 (was 2; the 2 from 2026-04-27 stay — `testing-standards-v1` on `path:cli/domains/`, `ui-interop-v1` on `ui/src/main.tsx:2` — plus 3 reappear: `TS_DANGEROUS_PATTERNS` on UI test/lib files, `MAKEFILE_QUALITY_GATES` on Makefile placeholders, `ESLINT_SAFETY_RULES` missing CRITICAL comments on `ui/eslint.config.js`)
- Low: 20 (unchanged: 17 ui-a11y focus-visible + 3 prd-template)
- Info: 1 (unchanged)
- **Total: 36** (was 24 reported 2026-04-27; was 36 reported 2026-04-26)

### Top 3 violations (operator action)
1. **Critical · `required_layout` · `Makefile`** — id `492b2c57-4b29-483c-a74d-0f20c4aa2802`. Persistent since 2026-04-25. Recommendation still terse ("Add the required resource at Makefile") — the rule body does not name the missing resource.
2. **High · `quality-gates` · `ui/package.json`** — id `5e140323-9a8f-4a22-bea8-26afe4b2a5ef`. "Update the build script to run `tsc --noEmit` (or a `type-check` script that does the same) before `vite build`." The scenario already has a `type-check` script — the fix is changing `"build": "vite build"` to `"build": "tsc --noEmit && vite build"`. Single-line edit.
3. **High · `type-safety` (TS_CONFIG_STRICT) · `ui/tsconfig.json`** — ids `6c806dec` (strict missing), `bd0fbd2a` (noUncheckedIndexedAccess missing), `86201e22` (protective comment block missing). Auto-fixable via `scenario-auditor fix reference-react-vite --rules TS_CONFIG_STRICT`.

### Fidelity discrepancy with 2026-04-27 scan
The 2026-04-27 same-day re-scan reported 24 violations and "all 9 Highs cleared". Today's scan (and re-run) report 36 — and the 9 Highs are confirmed present by direct file read. Possible causes:
- **Tool non-determinism** — scenario-auditor uses claude-code as a resource; rule-evaluation may be partly LLM-driven and produce inconsistent yes/no calls across restarts.
- **Rule-set hot-reload between scans** — scenario-auditor was stopped at the start of today's heartbeat and auto-started; if rules are loaded from disk at startup, a stale rule-set on 2026-04-27 could have masked the Highs.
- **Auditor binary/process drift** — only commit between heartbeats touching scenario-auditor: `e6b21b8da4` (small `agent_manager.go` change, unrelated to scan logic).
Either way, this means scan results across heartbeats may not be directly comparable. **The "delta vs prior scan" view, which is the primary signal of this heartbeat, is structurally unreliable until the tool's determinism is established.**

### Tidiness — structural breakdown (open queue: 30 + 58 lint surface)
- 13 High file-level duplication (handlers/repository/mocks/register tests) — unchanged
- 8 Medium (7 duplication, 1 complexity in `api/config/config.go` cyclomatic 21) — unchanged
- 9 Low duplication / length / complexity — unchanged
- 58 lint surface findings (ERROR/WARNING) — unchanged
- `make lint` / `make type` still `failed (exit 2)` — unchanged

### New since last scan (2026-04-27)
- 9 High violations re-surfaced (re-detection, not regression — the violations were always present).
- 3 Medium violations re-surfaced (TS_DANGEROUS_PATTERNS, MAKEFILE_QUALITY_GATES, ESLINT_SAFETY_RULES).
- DTV scenario back UP (was DOWN 2026-04-27 morning).

### Resolved since last scan (2026-04-27)
- None genuinely. The 12 "resolved" violations from the 2026-04-27 scan were never actually fixed.

### Tool-runtime issues observed
- **scenario-auditor scan output unstable across restarts** (NEW concern this heartbeat) — see fidelity-discrepancy section above.
- `test-genie run-tests` still returns opaque HTTP 500 — same as 2026-04-27. (Already covered by accepted `dec-1777068259096417622`.)
- DTV `validate`/`report` subcommands still not shipped — same.
- scenario-auditor was stopped at heartbeat start; auto-started cleanly.

### Capability gaps (covered by accepted `dec-1777068259096417622`)
- DTV `validate <reference>` and `report --conflicts | --drift | --maturity | --tool-baselines` not shipped.
- `test-genie run-tests` has no structured failure response.

### New capability gap raised this heartbeat
- scenario-auditor standards scan determinism / restart-sensitivity — see decisions section.

### Decisions this heartbeat
- 1 supersession: `dec-1777327401931026247` (toolchain-violation, the 24-violation false-clean take) → superseded by new toolchain-violation reflecting the corrected 36-violation reality.
- 1 new `capability work item`: scenario-auditor scan non-determinism / restart-sensitivity. Without this fixed, this heartbeat's primary signal (delta vs. prior scan) cannot be trusted.
- **Pre-heartbeat queue:** 4 pending team-wide; 1 own-context. Post-raise: 5 pending team-wide (4 − 1 superseded + 2 new = +1 net); 2 own-context (under 4 cap).

### Knowledge entries
- `toolchain-audit-2026-04-28` — supersedes `toolchain-audit-2026-04-27`
