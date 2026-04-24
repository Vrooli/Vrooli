# Swarm Manager Initiative Context

Load the immediate neighborhood of an initiative before proposing backlog changes. This is a shared reference — research and non-research workshop skills read it in so every flow bases its reuse-vs-create and reorg decisions on the same information.

## When to load context

Load context any time you are about to:

- Propose a **new** backlog item that might belong to an initiative.
- Conclude research with actions that **create**, **update**, or **delete** items.
- Decide whether to **reorder** or **reprioritize** work within an initiative.
- Judge whether a finding has implications for **sibling items** or **related initiatives**.

If you are doing a pure read-only investigation that will not touch the backlog, you do not need to load context.

## One command, one call

```bash
swarm-manager initiatives context --name <initiative-name>
```

The command prints, in one call:

- **Initiative** — name, title, description, status, priority, depends_on.
- **Rollup** — total / completed / in_progress / failed / pending counts for members.
- **Members** — every item currently in `items[]` with kind, name, title, status, priority, and its own `depends_on`. Archived members are marked.
- **Upstream initiatives** — the initiatives this one depends on (direct only, not transitive).
- **Downstream initiatives** — the initiatives that depend on this one (direct only).

Use the default human output for reading and reasoning. Reach for `--json` only when you are parsing the response programmatically.

## The graph artifact

Each initiative has a `graph.json` projection on disk, auto-materialized from its members' `depends_on` edges. It is the canonical shape for reasoning about the item graph (nodes with kind/title/status/priority/effort/archived, edges keyed `depends_on`). Reach for it when you need the topology, not just the list.

```bash
swarm-manager initiatives graph-show --name <initiative-name>
```

`graph.json` is **read-only to agents** — it is a projection, never a source of truth. Mutations go through backlog/initiatives endpoints (or an accepted feedback proposal); the projection updates itself.

You do not need to fetch `graph.json` when you are already running inside a skill that injects it as a variable (the feedback and review skills render it into `{{CURRENT_GRAPH}}`). Use `graph-show` from ad-hoc investigation skills that do not have that injection.

## Semantic interpretation

- **Members** are the plan within this initiative. They are the items that are candidates for updating or deleting if a research finding has invalidated them, and the items you should consider updating *instead* of creating a near-duplicate.
- **Upstream** initiatives are what this one blocks on. If your work reveals this initiative no longer depends on `X`, you may propose updating `depends_on` via an `Update initiative` action (research-conclusion-authoring) or a direct CLI call.
- **Downstream** initiatives are what this one unblocks. If your work reveals a finding with cross-initiative impact (e.g., "this initiative does not in fact gate initiative `Y` as originally planned"), flag it in the conclusion for the orchestrator to address.

## Reuse-before-create heuristic

Before proposing a new `Create backlog item`, do both of the following:

1. **Enumerate members.** Look at every item in the current initiative's `items[]`. If any existing item overlaps the intent of what you were about to create, prefer `Update backlog item` on that item (title, priority, depends_on, description) over creating a near-duplicate.
2. **Enumerate siblings across upstream and downstream.** If a sibling initiative already contains an item covering the same intent, consider whether moving it here via `Update backlog item` (changing the `initiative` field) is cleaner than introducing a new item.

Only when neither path fits should you propose `Create`. When you do, state in the action's `Reason` block why no existing item was sufficient.

## Referential integrity is a server invariant

You do **not** need to think about cascade bookkeeping. When you delete an item, the server automatically removes it from every other item's `depends_on` and from the enclosing initiative's `items[]`. When you move an item between initiatives via PATCH, both initiatives are kept in sync. When an initiative is deleted, its members are orphaned (the items persist; their `initiative` field is cleared) and dependent initiatives have the deleted name scrubbed from their `depends_on`.

Consequence: describe the action you want, not the bookkeeping. If the conclusion says *delete `idea/obsolete-cache-audit`*, the executor runs one delete call and the membership and dependency cleanups happen for free.

## Anti-patterns

- **Loading the global overview instead of this endpoint.** `swarm-manager overview` is the fire hose; `initiatives context` is the focused view. Do not client-filter the overview when this endpoint exists.
- **Transitive traversal.** The endpoint returns *direct* upstream and downstream only. If you need two hops, call the endpoint once more for that neighbor — do not try to build a fanout traversal in one agent step.
- **Cascade instructions.** Do not tell the executor to "also remove from initiative's items[]" or "update dependent items' depends_on" after a delete. The server handles it. If you find yourself writing such instructions, remove them.
- **Creating when you could update.** If a member item can absorb the intent with a title/priority/depends_on change, propose `Update backlog item`, not `Create backlog item`.
