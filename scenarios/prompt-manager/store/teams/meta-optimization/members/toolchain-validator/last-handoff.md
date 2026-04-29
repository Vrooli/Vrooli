### Tools run
- `scenario-auditor standards scan reference-react-vite --wait` (job `standards-cfb1529e-5958-418f-b2a5-3bab9976303c`, 36 violations); confirm-rerun job `standards-139a9acf-c5f6-4ec5-bcc6-5243af735814` returned identical 36
- `scenario-auditor security scan reference-react-vite --wait` (job `security-6f1f9fc8-f27d-493f-9129-b6060a9d97f5`, 0 vulns)
- `tidiness-manager scan reference-react-vite` (light scan; 30 structural + 58 lint surface; `make lint` / `make type` exit 2)
- `test-genie run-tests reference-react-vite` (failed: opaque HTTP 500, unchanged)
- `development-toolchain-validator validate reference-react-vite` — DTV scenario UP, `validate` still not shipped (`Unknown command: validate`)

### Reference scenario
- `reference-react-vite` (operator-nominated 2026-04-24, accepted via `dec-1776981723540926630`)

### Violation summary
- Critical: 1 (unchanged — Makefile `required_layout`)
- High: 9 (was 0 in 2026-04-27 scan; reality matches 2026-04-26 — yesterday was a false-negative, not a regression)
- Medium: 5 (was 2; 3 reappeared)
- Low: 20 (unchanged)
- Info: 1 (unchanged)
- **Total: 36** (was 24; was 36 on 2026-04-26)

### Top 3 violations
1. **Critical · scenario-auditor `required_layout` · `Makefile`** — id `492b2c57`. Persistent since 2026-04-25. Recommendation still terse — does not name the missing resource.
2. **High · scenario-auditor `quality-gates` (NODE_BUILD_TYPECHECK) · `ui/package.json`** — id `5e140323`. One-line edit: `"build": "tsc --noEmit && vite build"`. The `type-check` script already exists.
3. **High · scenario-auditor `type-safety` (TS_CONFIG_STRICT family) · `ui/tsconfig.json`** — ids `6c806dec`, `bd0fbd2a`, `86201e22`. Auto-fixable via `scenario-auditor fix reference-react-vite --rules TS_CONFIG_STRICT`.

### New since last scan
- 9 High violations re-detected (re-detection, not regression — present all along; verified by direct file read of `ui/tsconfig.json`, `ui/package.json`, `.vrooli/testing.json`, and absence of `api/.golangci.yml` / `cli/.golangci.yml`; git log confirms reference unchanged ≥3 days).
- 3 Medium violations re-surfaced (TS_DANGEROUS_PATTERNS, MAKEFILE_QUALITY_GATES, ESLINT_SAFETY_RULES).
- DTV scenario back UP (was DOWN 2026-04-27).

### Resolved since last scan
- None genuinely. The 12 "resolved" violations from the 2026-04-27 scan were never actually fixed.

### Capability gaps noticed
- Same as 2026-04-26 (covered by accepted `dec-1777068259096417622`): DTV `validate`/`report` not shipped; `test-genie run-tests` opaque HTTP 500.
- **NEW this heartbeat:** scenario-auditor standards scan output unstable across heartbeats / restarts (36 → 24 → 36 across three scans on identical reference state). Raised as `dec-1777414066604011410`.

### Decisions raised this heartbeat
- `dec-1777327401931026247` (toolchain-violation, the 24-violation false-clean take) → marked `superseded`.
- `dec-1777414037655179064` · `toolchain-violation` · supersedes `dec-1777327401931026247`. Corrected 36-violation take with remediation order.
- `dec-1777414066604011410` · `capability-gap` · scenario-auditor scan non-determinism / restart-sensitivity blocks the team's primary signal.

### Knowledge entries written
- `knw-1777414085585863925` · `toolchain-scan-2026-04-28` (supersedes `toolchain-scan-2026-04-27`)