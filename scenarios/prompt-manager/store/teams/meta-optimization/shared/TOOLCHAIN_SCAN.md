# Toolchain Scan

Latest scan result from `toolchain-validator`. Supersedes on each heartbeat.

---

## Latest scan: 2026-04-23

**Status: BLOCKED — no gold-star reference scenario designated**

### Tools attempted
- `development-toolchain-validator` — running (localhost:16445, healthy, uptime 2.9d); CLI saved `api_base=http://example.com` is stale — used `--api-base http://localhost:16445` override for this scan.
- Fallback trio (`scenario-auditor`, `test-genie`, `tidiness-manager`) — not invoked; no target designated.

### Reference scenario
- `docs/meta-optimization/REFERENCE_SCENARIOS.md` → **unset** (operator to nominate).
- `development-toolchain-validator reference list` → **0 references registered**.

Because no gold-star reference is designated, no validation scan ran this heartbeat.

### Violation summary
- Critical: 0 (not measured)
- Major: 0 (not measured)
- Minor: 0 (not measured)

### Observations (not tool-reported, operator-actionable)
1. **Reference unset (blocker).** Per `REFERENCE_SCENARIOS.md` nomination rules, the operator must nominate. Until then, `toolchain-validator` cannot do its primary job. Filed as `toolchain-violation`.
2. **DTV CLI config drift.** `development-toolchain-validator configure` saved `api_base=http://example.com` while the scenario actually binds `http://localhost:16445`. Non-blocking because `--api-base` override works; worth fixing once. Noted here; no decision raised (under cap).
3. **DTV `reference` surface exists but empty.** The CLI supports `reference list|get|create|update|delete`. Once the operator nominates a scenario in `REFERENCE_SCENARIOS.md`, it should also be registered via `development-toolchain-validator reference create` so `validate` (when it lands) has a target.

### New since last scan
- First scan — baseline.

### Resolved since last scan
- N/A.

### Capability gaps noticed
- `development-toolchain-validator` does not yet expose the consolidated `validate <reference>` or `report --conflicts | --drift | --maturity | --tool-baselines` commands described in `HEARTBEAT.md`. The CLI today exposes only `reference` / `connection` / `status` / `configure`. This is a forthcoming feature per team charter; not filed as a decision this heartbeat — deferred until a reference exists to validate against.

### Decisions raised this heartbeat
- 1 × `toolchain-violation` — designate a gold-star reference scenario.

### Knowledge entries written
- `toolchain-scan-2026-04-23` (first entry; no prior to supersede).
