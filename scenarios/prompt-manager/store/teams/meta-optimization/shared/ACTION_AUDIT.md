# Action Audit - Rolling

Action health and adoption snapshot. Maintained by `skill-optimizer`.

## Baseline - 2026-05-01

| Action | Status | Validation | Discoverability | Disposition | Notes |
|--------|--------|------------|-----------------|-------------|-------|
| action:scenario.status.show | active | valid; dry-run passes | mixed discovery returns `show scenario status`; graph health 0.725 with inbound references | adopted | First active read-only seed Action. |
| action:team.swarm.work.list | active | valid; dry-run passes | mixed discovery returns `list team work`; graph health 0.725 with inbound references | adopted | Read-only Swarm Manager work lookup with `apiRead` permission. |

## Measurement Signals

- Action count by status: 2 active.
- Active runnable Action count: 2.
- Validation failures: 0 in targeted seed validation.
- Graph inbound warnings: 0 after Action references landed.
- Run history signals: no post-adoption usage baseline yet.
- Skill prose collapsed to Action references: 0.
- Repeated manual operation count from run-introspector: not yet measured.

## Revisit Queue

1. After four meta-optimization heartbeats, compare Action discoveries/runs against repeated manual operations.
2. Continue adding seed Actions only when one stable Vrooli-controlled CLI command owns the operation.
