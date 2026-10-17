# Requirements Registry — Security Health

This registry links the PRD operational targets (`../PRD.md` → `## 🎯 Operational Targets`) to implementation-facing technical requirements. Each `NN-<target>/module.json` mirrors one operational target via its `prd_ref` (e.g. `OT-P0-001`); numbers preserve ordering, not priority.

## Operational targets covered
- **P0** — substrate-aware validation, normalized severity contract, delegation-ready CLI, test-genie security producer, actionable remediation, graceful scanner degradation, incremental resource-bounded validation.
- **P1** — fleet dependency intelligence, semantic + structured dependency search, security posture UI, embeddable posture badge.
- **P2** — Python & JS/TS SAST, continuous CVE alerting, secret rotation workflows.

## Lifecycle & auto-sync
1. Operational targets in `PRD.md` map 1:1 to module folders here.
2. `index.json` imports each module; `auto_sync_enabled` lets test-genie update requirement `status` when tagged tests run.
3. Coverage summaries land in `coverage/phase-results/` after each test phase.

## Validation commands
- Validate this registry: `vrooli scenario requirements validate security-health --json`
- Validate the PRD ↔ requirements linkage: `vrooli scenario requirements validate security-health --json`
- Run the scenario's own audit (exercises tagged tests): `test-genie execute security-health --preset comprehensive`

## Contributor notes
- Tag tests with `[REQ:ID]` (e.g. `[REQ:REQ-P0-001]`) so auto-sync can flip `status`.
- Keep each requirement's `prd_ref` equal to its operational-target ID so PRD linkage stays green.
- Never add compatibility shims (duplicate folders or alias imports) during migrations — let things fail temporarily instead of adding debt.
- Keep this README under 100 lines. See `scenarios/test-genie/docs/reference/requirement-schema.md` for schema details and `scenarios/test-genie/docs/phases/business/requirements-sync.md` for auto-sync behavior.
