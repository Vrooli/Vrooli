### Tools run
- `scenario-auditor standards scan reference-react-vite --wait` (job `standards-da887d1a-bed0-4810-a819-dc2478001346`, 24 violations)
- `scenario-auditor security scan reference-react-vite --wait` (job `security-a394ac11-4566-4934-b144-c9e3bf5e1161`, 0 vulns)
- `tidiness-manager --auto-start scan reference-react-vite` (auto-started cleanly; 30 structural + 58 lint surface)
- `test-genie run-tests reference-react-vite` (failed: opaque HTTP 500, unchanged)
- `development-toolchain-validator reference list` / `validate` — DTV scenario DOWN (connection refused :16445); `validate` still not shipped

### Reference scenario
- `reference-react-vite` (operator-nominated 2026-04-24, accepted via `dec-1776981723540926630`)

### Violation summary
- Critical: 1 (unchanged — Makefile `required_layout`)
- High: 0 (was 9)
- Medium: 2 (was 5; 3 cleared, 2 new)
- Low: 20 (unchanged)
- Info: 1 (unchanged)
- **Total: 24** (was 36)

### Top 3 violations
1. **Critical · scenario-auditor `required_layout` · `Makefile`** — id `b819dfae`. Persistent since 2026-04-25; recommendation text remains terse and does not name the missing resource.
2. **Medium · scenario-auditor `testing-standards-v1` · `cli/domains/`** (NEW) — id `fd7432bf`. Missing `*_test.go` coverage for cli/domains source files.
3. **Medium · scenario-auditor `ui-interop-v1` · `ui/src/main.tsx:2`** (NEW) — id `568bbc85`. Add `import { initSpatialNav } from '@vrooli/iframe-bridge/spatial'` and call `initSpatialNav()` in main.tsx.

### New since last scan
- 2 Medium standards violations: `testing-standards-v1` on `cli/domains/`, `ui-interop-v1` on `ui/src/main.tsx`.
- DTV scenario was DOWN at heartbeat start (single occurrence — not yet a pattern).

### Resolved since last scan
- All 9 High `type-safety` + `quality-gates` violations cleared (TS_CONFIG_STRICT, noUncheckedIndexedAccess, ESLint typed config, `vite build` gating, testing.json strict flags).
- 1 High `type_safety` tidiness on `ui/src/lib/api.test.ts` cleared as side effect.
- Net 12 standards violations resolved (36 → 24).

### Capability gaps noticed
- Same as 2026-04-26 (covered by accepted `dec-1777068259096417622`): DTV `validate`/`report` not shipped; `test-genie run-tests` opaque HTTP 500.

### Decisions raised this heartbeat
- `dec-1777327401931026247` · `toolchain-violation` · 24-violation new shape with the 2 new Mediums + recommended remediation order (Critical first, then ui-interop, then testing-standards, defer Lows).

### Knowledge entries written
- `knw-1777327414393933193` · `toolchain-scan-2026-04-27` (supersedes `toolchain-scan-2026-04-26`)