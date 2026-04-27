# CLI Commands

This document captures the canonical Swarm Manager CLI flows for backlog import and initiative management.

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

## Execution Create

If `--mode` is omitted, the CLI resolves it from `GET /api/v1/settings`.

```bash
swarm-manager execution create --kind idea --name my-feature
```
