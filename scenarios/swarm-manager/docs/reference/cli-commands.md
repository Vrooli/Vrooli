# CLI Commands

This document captures the canonical Swarm Manager CLI flows for backlog import and initiative management.

## Operations

Use startup briefs as the fastest first-answer packet for agent sessions:

```bash
swarm-manager sessions startup-brief --id sess_123 --json
swarm-manager sessions startup-brief --id sess_123 --refresh --json
swarm-manager portfolio brief --json
swarm-manager initiatives candidates --purpose next-action --json
swarm-manager backlog pending-questions --brief --json
```

These commands return bounded context packets with summaries, source counts,
freshness metadata, recommended next actions, and drill-down commands. They are
for startup and routing; use the detailed commands only after the brief points
to a specific scope.

Use the operations briefing as the fastest current-status packet for agents and
operators:

```bash
swarm-manager operations brief
swarm-manager operations brief --json
swarm-manager operations brief --window PT1H --json
```

Human output summarizes counts, attention items, active work, next actions, and
warnings. JSON output preserves the `GET /api/v1/operations/brief` response
shape for prompt context and automation.

Use the live operations list for drill-down and filtering:

```bash
swarm-manager operations list --json
swarm-manager operations list --lane execute --status running
swarm-manager operations list --owner-type initiative --q desktop-release
```

Filters encode directly to the API query parameters: `window`, repeatable
`status`, repeatable `lane`, repeatable `mode`, repeatable `owner_type`, and
`q`.

## Backlog Create

```bash
swarm-manager backlog create --data '{
  "kind":"idea",
  "name":"my-feature",
  "title":"My Feature",
  "acceptance_allow":["scenarios/swarm-manager/**"]
}'
```

Create with evidence files:

```bash
swarm-manager backlog create \
  --data '{"kind":"fix","name":"preview-crash","title":"Preview crash","acceptance_allow":["scenarios/app-monitor/**"]}' \
  --attach evidence/report.json=/tmp/report.json \
  --attach evidence/screenshot.png=/tmp/screenshot.png
```

Rules:
- input JSON is decoded strictly
- unknown fields fail fast
- do not send `scope`
- `--attach` is repeatable and uses `destination=source`
- attachment destinations must be safe relative paths and are sent through the
  multipart `POST /api/v1/backlog` contract

## Backlog Auto-Filer Suggestions

List suggested items:

```bash
swarm-manager backlog list --status suggested
```

Inspect a suggestion before acting:

```bash
swarm-manager backlog get --kind fix --name auto-filed-example
swarm-manager backlog get --kind fix --name auto-filed-example --json
```

Accept a suggestion by moving it into the normal backlog flow:

```bash
swarm-manager backlog update \
  --kind fix \
  --name auto-filed-example \
  --data '{"status":"backlog"}'
```

Dismiss a suggestion through the auto-filer RPC. This archives the item and
remembers its stable `finding_ref`, so the same finding is not suggested again.

```bash
swarm-manager backlog dismiss --kind fix --name auto-filed-example --reason "not actionable"
swarm-manager backlog dismiss --kind fix --name auto-filed-example --json
```

The dismiss command is for auto-filer suggestions. Use ordinary backlog update,
queue, review, and archive flows for accepted items.

## Backlog Auto-Filer Status

Read the current policy, latest cycle accounting, open-item cap, velocity brake,
and dismissal count:

```bash
swarm-manager autofiler status
swarm-manager autofiler status --json
```

Run one governed cycle immediately, using the same settings, cap, velocity
brake, and reconciliation rules as the background sweeper:

```bash
swarm-manager autofiler run-now
swarm-manager autofiler run-now --json
```

The same policy knobs live under `auto_filer` in Settings and are documented in
[DOC: docs/reference/configuration.md#backlog-auto-filer-settings-api].

## Backlog Batch Create

Preview first:

```bash
cat > /tmp/batch-items.json <<'EOF'
{
  "items": [
    {
      "kind": "research",
      "name": "desktop-release-control-plane-audit",
      "title": "Audit desktop release control plane",
      "initiative": "desktop-release-governance",
      "acceptance_allow": ["scenarios/deployment-manager/**"]
    }
  ],
  "initiatives": [
    {
      "name": "desktop-release-governance",
      "title": "Desktop Release Governance",
      "status": "active",
      "priority": 1
    },
    {
      "name": "desktop-release-telemetry",
      "title": "Desktop Release Telemetry",
      "status": "active",
      "priority": 2,
      "depends_on": ["desktop-release-governance"]
    }
  ]
}
EOF

swarm-manager backlog batch-create --file /tmp/batch-items.json --preview
```

Create after review:

```bash
swarm-manager backlog batch-create --file /tmp/batch-items.json
```

Notes:
- `initiative` is carried per item inside the JSON file
- initiative metadata lives in the top-level `initiatives` array
- there is no `--initiative` flag for batch create
- initiative `priority` accepts `1-10` (or `0` for unprioritized); `depends_on` takes bare initiative names (not `kind/name`), and the batch applies initiatives in topological order so a dependent initiative may be declared before its dependency

## Initiatives

Create:

```bash
swarm-manager initiatives create --data '{
  "name":"desktop-release-governance",
  "title":"Desktop Release Governance",
  "description":"Shared release-control and desktop delivery work.",
  "status":"active",
  "priority": 1,
  "depends_on": []
}'
```

Update partially (supply only fields that should change):

```bash
swarm-manager initiatives update --name desktop-release-governance --data '{
  "priority": 2,
  "depends_on": ["desktop-release-telemetry"]
}'
```

Fields:
- `priority`: `1-10` (or `0` for unprioritized)
- `depends_on`: array of bare initiative names; must reference existing initiatives; cycles and self-references are rejected
- `status`: `active` or `completed` (archiving is handled via `initiatives delete`)

Load initiative context (initiative + members + upstream + downstream) in one call:

```bash
swarm-manager initiatives context --name desktop-release-governance
swarm-manager initiatives context --name desktop-release-governance --json
```

Response shape (`--json`):

```json
{
  "initiative": { ... },
  "rollup": { "total": 3, "completed": 1, "in_progress": 1, "failed": 0, "pending": 1 },
  "items": [ { "kind": "idea", "name": "...", "title": "...", "status": "backlog", "priority": 3, "depends_on": [...] }, ... ],
  "upstream_initiatives": [ { "name": "upstream-a", "title": "Upstream A", ... } ],
  "downstream_initiatives": [ { "name": "downstream-b", "title": "Downstream B", ... } ]
}
```

Only direct upstream and downstream are returned — the endpoint is a one-hop neighborhood view, not a transitive traversal. Use it in place of the global `overview` command when the question is scoped to one initiative.


## Cascade semantics

The API maintains referential integrity automatically when items or initiatives are mutated. Callers do not need to emit follow-up cleanup calls:

| Operation | Cascade |
|-----------|---------|
| `backlog delete --kind K --name N` | Removes `"K/N"` from every other item's `depends_on`; removes it from its enclosing initiative's `items[]`. Atomic. |
| `backlog update --data '{"initiative":"X"}'` | Detaches from the old initiative's `items[]`, attaches to `X.items[]`. Rejects if `X` does not exist. |
| `backlog create --initiative X` | Validates `X` exists; adds the new ref to `X.items[]`. |
| `initiatives delete --name I` | Orphans every member item (clears their `initiative` field; items persist); scrubs `I` from every other initiative's `depends_on`. Atomic. |
| `initiatives add-items --items kind/name,...` | Rejects items that already belong to a different initiative; attaches orphans. To move an item, use `backlog update` instead. |
| `initiatives remove-items --items kind/name,...` | Removes from `items[]` and clears each item's `initiative` field if it matches. |

## Receipt observations

There is no `swarm-manager evidence` command. Query run-correlated operation
observations from Agent Manager instead:

```bash
curl "http://localhost:${AGENT_MANAGER_API_PORT}/api/v1/runs/<run-id>/observed-receipts"
```

Vrooli Events absence or an empty observation list is represented as degraded
or unobserved evidence; it never certifies a failure or blocks a transition.

## Execution Create

If `--mode` is omitted, the CLI resolves it from `GET /api/v1/settings`.

```bash
swarm-manager execution create --kind idea --name my-feature
```
