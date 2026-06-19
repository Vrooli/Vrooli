# Requirements Registry

Organize requirement modules by PRD operational targets, keeping the filesystem structure aligned with the "what" articulated in the PRD. Create folders such as `01-<target-name>/` as needed (numbers preserve ordering but do **not** imply priority).

This registry is PRD-specific: it maps every Tunnel Manager operational target (`OT-P0-*`, `OT-P1-*`, `OT-P2-*` in `PRD.md`) to the owning domain module (see `docs/concepts/DOMAINS.md`).

## Modules
| Folder | Domain | Operational targets |
|---|---|---|
| `01-exposure-manifest/` | routes | OT-P0-001 |
| `02-cloudflare-ingress/` | config | OT-P0-002, OT-P1-002 |
| `03-exposure-tiers/` | exposure | OT-P0-003/004/005/006, OT-P2-001/003 |
| `04-port-compliance/` | audit | OT-P0-007 |
| `05-tunnel-health/` | tunnel | OT-P0-008, OT-P1-003/006 |
| `06-liveness-probes/` | probes | OT-P0-009/010, OT-P1-001 |
| `07-auto-recovery/` | recovery | OT-P0-011, OT-P1-005 |
| `08-cli-interface/` | cli | OT-P0-012 |
| `09-web-dashboard/` | ui | OT-P1-004 |
| `10-app-monitor-integration/` | exposure-query | OT-P1-007 |

## Lifecycle
1. Operational targets in PRD map to folders here.
2. `requirements/index.json` imports each module; tests auto-sync their status when they run.
3. Coverage summaries live in `coverage/phase-results/` after each test phase.

## Validation
- Validate the registry structure and target linkage:
  `prd-control-tower requirements validate tunnel-manager --json`
- Auto-sync requirement status from tests by running the suite:
  `vrooli scenario test tunnel-manager`

## Contributor Notes
- Add folders/modules that match your scenario’s PRD targets (P0/P1/P2) instead of reusing other scenarios’ names.
- Tag tests with `[REQ:ID]` so auto-sync can update status.
- Never add compatibility shims (duplicate folders or alias imports) during migrations—let things fail temporarily instead of adding debt.
- Keep this README under 100 lines. Use `scenarios/test-genie/docs/reference/requirement-schema.md` for schema details and `scenarios/test-genie/docs/phases/business/requirements-sync.md` for auto-sync behavior.
