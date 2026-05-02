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

## Operating Mode Catalog

Top-level `operating-mode` commands operate on the mode catalog itself (not a
specific initiative). Use these to browse modes, inspect linked initiatives,
or edit a mode's display label/description (persisted to the overlay file at
`<scenario>/.vrooli/operating-modes/overrides.json`).

```bash
swarm-manager operating-mode list                      # list modes with usage counts
swarm-manager operating-mode get --mode holistic-loop  # full detail incl. linked initiatives
swarm-manager operating-mode set --mode holistic-loop --label "Loop (renamed)" --description "Custom wording."
swarm-manager operating-mode set --mode holistic-loop --clear-description
```

## Initiative Operating Modes

List registered modes from the backend registry catalog:

```bash
swarm-manager initiatives mode-list
swarm-manager initiatives mode-list --json
```

Inspect the mode workspace:

```bash
swarm-manager initiatives mode-workspace --name desktop-release-governance
swarm-manager initiatives mode-workspace --name desktop-release-governance --json
```

Switch modes through the lifecycle boundary. When switching from `item-level` into
an initiative-scoped mode, add `--cancel-active-item-executions` only after you
intend to stop active member item runs. Switching out of an initiative-scoped
mode is rejected while any mode round is reserved or agent-running:

```bash
swarm-manager initiatives mode-switch \
  --name desktop-release-governance \
  --mode holistic-loop \
  --cancel-active-item-executions
```

Start and manage phase rounds. `mode-start` follows backend phase action state;
invalid phases are rejected even if the CLI command is formed correctly.
Round-control commands require `--mode` to name a registered non-default mode;
do not omit it or pass `item-level`:

```bash
swarm-manager initiatives mode-start --name desktop-release-governance --phase investigate
swarm-manager initiatives mode-refresh --name desktop-release-governance --mode holistic-loop --round 1
swarm-manager initiatives mode-cancel --name desktop-release-governance --mode holistic-loop --round 1
```

Apply the audited mark-complete reconciliation from a completed mode round:

```bash
swarm-manager initiatives mode-complete-items \
  --name desktop-release-governance \
  --mode holistic-loop \
  --round 3 \
  --run-id run_abc123 \
  --items execute/item-a,fix/item-b
```

Apply selected proposal-backed create/update/follow-up reconciliation from a
completed mode round:

```bash
swarm-manager initiatives mode-apply-backlog-sync \
  --name desktop-release-governance \
  --mode phased-plan-drain \
  --round 4 \
  --run-id run_def456 \
  --mutations m1,m3
```

Rules:
- `mode-switch` is the only CLI path for changing initiative `mode`; generic initiative updates do not own mode lifecycle.
- `mode-refresh`, `mode-cancel`, `mode-complete-items`, and `mode-apply-backlog-sync` are non-default-mode-only operations and should always include `--mode`.
- `mode-refresh` fails completed runs whose final output is missing or violates the registered phase output contract; those failures release the initiative lock and do not advance the phase graph.
- `mode-complete-items` requires the round's AgentManager `run_id` and only accepts member item refs from the round.
- `mode-apply-backlog-sync` requires the round's AgentManager `run_id` and applies the round's `backlog_sync.proposal` through the existing proposal boundary.

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
