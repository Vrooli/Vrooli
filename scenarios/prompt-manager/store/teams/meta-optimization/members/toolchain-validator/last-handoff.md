### Tools run
- `scenario-auditor standards scan reference-react-vite --wait` (job `standards-b9cfe21b-7aed-4e7a-a507-aeab60a7c91e`)
- `scenario-auditor security scan reference-react-vite --wait` (job `security-76a33291-bcd3-4562-ace9-de16b2fbe2b3`, 0 vulns)
- `tidiness-manager scan reference-react-vite` + `tidiness-manager issues reference-react-vite --limit 20`
- `test-genie run-tests reference-react-vite` (failed: opaque HTTP 500, unchanged)
- `development-toolchain-validator reference list` (validate/report still not shipped)

### Reference scenario
- `reference-react-vite` (operator-nominated 2026-04-24, accepted via `dec-1776981723540926630`)

### Violation summary
- **Critical: 1** (NEW)
- High: 9 (was 41)
- Medium: 5 (was 10)
- Low: 20
- Info: 1
- **Total: 36** (was 72)

### Top 3 violations
1. **Critical · scenario-auditor `required_layout` · `Makefile`** — "Scenario Required Structure"; new since 2026-04-24. The rewrite that cleared the 37 stack-governance highs introduced this Critical instead.
2. **High · scenario-auditor `type-safety` (×6) · `ui/tsconfig.json` + `ui/eslint.config.js`** — strict / noUncheckedIndexedAccess / protective comment / typed ESLint config still missing. Auto-fixable.
3. **High · scenario-auditor `quality-gates` (×4) · `ui/package.json` + `.vrooli/testing.json`** — `vite build` skips `tsc --noEmit`; testing.json strict flags off for both node_package + go_module.

### New since last scan
- +1 Critical `required_layout` on Makefile.
- +58 lint issues surfaced by tidiness (tooling-surface change, not code regression — yesterday's `make lint` exit 2 short-circuited).
- +20 open tidiness issues now visible: 14 High duplication (handlers, repository, mocks, register tests) + 1 High type_safety (`ui/src/lib/api.test.ts` 18 as-Type) + 5 Medium.

### Resolved since last scan
- All 37 `stack-governance` Makefile highs cleared.
- High count 41 → 9.
- Total 72 → 36.

### Capability gaps noticed
- Same as 2026-04-24 (covered by pending `dec-1777068259096417622`): `test-genie run-tests` returns opaque HTTP 500 with no structured failure data; DTV `validate <reference>` and `report --conflicts | --drift | --maturity | --tool-baselines` still not shipped.

### Decisions raised this heartbeat
- `dec-1777154587228516340` · `toolchain-violation` · supersedes `dec-1777068246086430656` with the current 36-violation / 1-Critical shape and a recommended remediation order (required_layout first, then auto-fix TS_CONFIG_STRICT, then golangci configs, then duplication consolidation, then a11y).
- `dec-1777068246086430656` marked `superseded`.
- `dec-1777068259096417622` (capability-gap) left as-is — content still current.

### Knowledge entries written
- `knw-1777154600872030412` · `toolchain-scan-2026-04-25` (supersedes `toolchain-scan-2026-04-24`)