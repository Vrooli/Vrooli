# Action Audit - Rolling

Action health and adoption snapshot. Maintained by `skill-optimizer`.

## Baseline - 2026-08-09

| Action | Status | Validation | Discoverability | Disposition | Notes |
|--------|--------|------------|-----------------|-------------|-------|
| action:scenario.status.show | active | valid; dry-run passes with `--scenario=prompt-manager` | discoverability verified by `prompt-manager action list`; live registry entry | adopted | Active read-only seed Action; owner `project:vrooli`. |
| action:team.swarm.work.list | missing | 404 on show/validate/run | absent from `prompt-manager action list` | stale-register | Previously recorded seed is not in the live Action registry; do not count it as active or propose consumers until reintroduced and revalidated. |

## Measurement Signals

- Action count by status: 1 active, 1 stale register entry.
- Active runnable Action count: 1.
- Validation failures: 0 in targeted seed validation.
- Graph inbound warnings: 0 for the one registered seed; the missing `team.swarm.work.list` entry has no live contract to validate.
- Run history signals: no post-adoption usage baseline yet.
- Skill prose collapsed to Action references: 0.
- Repeated manual operation count from run-introspector: not yet measured.

## Revisit Queue

1. After four meta-optimization heartbeats, compare Action discoveries/runs against repeated manual operations.
2. Continue adding seed Actions only when one stable Vrooli-controlled CLI command owns the operation.
