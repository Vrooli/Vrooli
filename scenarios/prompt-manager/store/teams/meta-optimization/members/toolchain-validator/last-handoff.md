### Tools run
- `development-toolchain-validator status` + `reference list` (via `--api-base http://localhost:16445` override)
- Fallback trio (`scenario-auditor`, `test-genie`, `tidiness-manager`) **not invoked** — no reference designated to scan against.

### Reference scenario
- `docs/meta-optimization/REFERENCE_SCENARIOS.md` → **unset** (operator to nominate)
- `development-toolchain-validator reference list` → **0 registered**

### Violation summary
- Critical: 0 (not measured — no target)
- Major: 0 (not measured — no target)
- Minor: 0 (not measured — no target)

### Top 3 violations
1. **blocker · operator-action · gold-star reference unset** — primary job cannot run until nominated.
2. **minor · DTV CLI · saved `api_base=http://example.com` is stale** — real URL is `http://localhost:16445`; overridable via `--api-base`, non-blocking.
3. **minor · DTV CLI · `validate`/`report` subcommands from HEARTBEAT.md not yet shipped** — current surface is `reference`/`connection`/`status`/`configure`; fallback trio remains the only operational path.

### New since last scan
- First scan — baseline.

### Resolved since last scan
- N/A.

### Capability gaps noticed
- Consolidated DTV `validate <reference>` and `report --conflicts|--drift|--maturity|--tool-baselines` commands promised in HEARTBEAT.md not implemented. Deferred from `capability-gap` decision until a reference exists to meaningfully scan.

### Decisions raised this heartbeat
- `dec-1776981723540926630` · `toolchain-violation` · designate a gold-star reference scenario so toolchain-validator can run its primary job.

### Knowledge entries written
- `knw-1776981717192131553` · `toolchain-scan-2026-04-23` (first entry; no prior to supersede).