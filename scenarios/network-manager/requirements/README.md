# Network Manager Requirements

This registry translates the operational targets in `../PRD.md` into falsifiable implementation requirements. Future implementation work should tag tests with `[REQ:<id>]` and keep each `validation[]` reference aligned with real evidence.

## Registry Shape

- `index.json` imports one numbered module per P0/P1 target, plus a grouped P2 expansion module.
- Requirement IDs use `NM-P0-*`, `NM-P1-*`, and `NM-P2-*`.
- `prd_ref` values point at exact operational target IDs from `PRD.md`.
- All requirements are currently `planned` because this session intentionally stops after scaffold, PRD, requirements, and docs.

## Validation Expectations

During implementation:

- Domain, API, CLI, and UI tests should include `[REQ:ID]` tags.
- Manual validation is acceptable only for operational procedures that cannot be automated safely.
- P0 requirements should gain automated validation before the scenario is considered viable.
- Status should be earned from passing evidence rather than hand-flipped.

## Auto-sync guidance

`auto_sync_enabled` is `true` in `index.json`. Future comprehensive scenario test runs should let requirements sync earn statuses from `[REQ:ID]` tagged evidence. Do not hand-edit sync snapshots or force status changes to make reports look complete.

Useful commands:

```bash
vrooli scenario requirements validate network-manager
vrooli scenario requirements report network-manager
vrooli scenario requirements lint-prd network-manager
```

## Current P0 Coverage

| Requirement | Operational Target |
|---|---|
| `NM-P0-001` | `OT-P0-001` Network health snapshot |
| `NM-P0-002` | `OT-P0-002` AdGuard Home resolver management |
| `NM-P0-003` | `OT-P0-003` Conservative DNS filtering controls |
| `NM-P0-004` | `OT-P0-004` Device inventory |
| `NM-P0-005` | `OT-P0-005` Safe optimization experiments |
| `NM-P0-006` | `OT-P0-006` Capability adapter model |
| `NM-P0-007` | `OT-P0-007` Home Automation integration contract |
| `NM-P0-008` | `OT-P0-008` Privacy-preserving defaults |
| `NM-P0-009` | `OT-P0-009` Operator UI and CLI workflows |
