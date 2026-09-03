# Action Audit - Rolling

Action health and adoption snapshot. Maintained by `skill-optimizer`.

## Baseline - 2026-08-09

| Action | Status | Validation | Discoverability | Disposition | Notes |
|--------|--------|------------|-----------------|-------------|-------|
| action:scenario.status.show | active | valid; dry-run passes with `--scenario=prompt-manager` | discoverability verified by `prompt-manager action list`; live registry entry | adopted | Active read-only seed Action; owner `project:vrooli`. |
| action:team.swarm.work.list | missing | 404 on show/validate/run | absent from `prompt-manager action list` | stale-register | Previously recorded seed is not in the live Action registry; do not count it as active or propose consumers until reintroduced and revalidated. |
| agent-system.framework-health | active | valid; dry-run passes (`prompt-manager graph audit`) | exact match found by `prompt-manager discover` | existing-action-reference | Read-only framework-health sensor Action owned by `scenario:prompt-manager`; API-read permission; runnable. Validation has an owner-governance warning because the owning scenario lacks declared `cli/manifest.json` governance. Proposed use: replace manual framework-health collection references in `agent-system-audit`; no new Action needed. |
| skill-validation-contract-audit | no exact Action | blocked | discovery returned skill guidance, not an Action | capability-work-item | No governed one-command Action currently owns skill command/reference validation; backlog `fix-skill-validation-current-cli-contracts` records the required owner work. |

## Measurement Signals

- Action count by status: 1 active, 1 stale register entry.
- Active runnable Action count: 1.
- Validation failures: 0 in targeted seed validation.
- Graph inbound warnings: 0 for the one registered seed; the missing `team.swarm.work.list` entry has no live contract to validate.
- Run history signals: no post-adoption usage baseline yet.
- Skill prose collapsed to Action references: 0.
- Skill prose collapsed to Action references: 1 proposed (`agent-system-audit` → `agent-system.framework-health`); acceptance pending owner work item.
- Repeated manual operation count from run-introspector: not yet measured.
- 2026-08-29 discovery for visited-tracker coverage found no exact Action; retain as skill-improvement work, not an Action candidate.

## Revisit Queue

1. After four meta-optimization heartbeats, compare Action discoveries/runs against repeated manual operations.
2. Continue adding seed Actions only when one stable Vrooli-controlled CLI command owns the operation.
