# Swarm Manager Goal Context

Load a goal and its milestones before proposing a goal change or reviewing milestone delivery. Use the goal's derived scope as the authoritative membership view.

## When to load context

Load context before you propose a target, milestone, dependency, or item-assignment change. Load context before you judge a milestone. Do not load it for a read-only task that cannot change a goal or milestone decision.

## One command, one call

```bash
swarm-manager goals context --name "<goal-name>"
```

The command returns the goal identity, target roots, derived scope, milestones, milestone rollups, unassigned scope items, and direct goal dependencies. Use the default human output for reasoning. Use `--json` only for programmatic parsing.

## Scope interpretation

- The derived scope is the authoritative closure from the goal targets. Do not replace it with client-side traversal.
- A milestone owns only its assigned scope items. An unassigned item remains visible in the goal but does not count as milestone work.
- An item can belong to one milestone in a goal. Use milestone assignment instead of duplicating membership.
- Goal dependencies describe cross-goal ordering. A milestone dependency describes ordering inside one goal. Escalate a cross-goal milestone dependency to operator review.

## Reuse-before-create

Inspect existing milestones and their items before proposing new work. Prefer assigning or updating an existing in-scope item when it can meet the stated acceptance criterion. Propose a new item only when no existing scope item can absorb the work; state that evidence in the proposal.

## Authority boundary

The context command is read-only. Do not edit derived scope, milestone rollups, or graph projections. Mutations require an operator-approved Swarm proposal or a declared CLI command.

## No known operational edge cases for standard usage.
