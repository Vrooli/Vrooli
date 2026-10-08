---
name: "swarm-manager-goal-context"
description: "Shared reference for loading a goal, its derived scope, and milestones before proposing goal changes or reviewing milestone delivery."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["tools","conversation"]
  tags: ["swarm-manager","goal","milestone","context"]
  status: "active"
  revision: 1
  createdAt: "2026-07-22T00:00:00Z"
  updatedAt: "2026-07-22T00:00:00Z"
  requires:
    scenarios: ["swarm-manager"]
    commands: ["swarm-manager backlog", "swarm-manager goals"]
  origin:
    kind: "authored"
---
# Swarm Manager Goal Context

Load a goal and its milestones before proposing a goal change or reviewing milestone delivery. Use the goal's derived scope as the authoritative membership view.

## When to load context

Load context before you propose a target, milestone, dependency, or item-assignment change. Load context before you judge a milestone. Do not load it for a read-only task that cannot change a goal or milestone decision.

## One command, one call

```bash
swarm-manager goals context --name "<goal-name>"
```

The command returns the goal's derived scope: target roots, closure, ready and blocked counts, milestones with their rollups, and unassigned scope items. Use the default human output for reasoning. Use `--json` only for programmatic parsing.

## Scope interpretation

- The derived scope is the authoritative closure from the goal targets. Do not replace it with client-side traversal.
- A milestone owns only its assigned scope items. An unassigned item remains visible in the goal but does not count as milestone work.
- An item can belong to one milestone in a goal. Use milestone assignment instead of duplicating membership.
- A milestone dependency describes ordering inside one goal. There is no goal-to-goal dependency: ordering between goals is expressed by item dependencies, which is why an item in one goal's closure can block another goal's work.

## Reuse-before-create

Search before you propose new work: `swarm-manager backlog search-ai "<intent>" --json`. Then inspect existing milestones and their items. Prefer assigning or updating an existing in-scope item when it can meet the stated acceptance criterion. Propose a new item only when neither the search nor the scope surfaces one that can absorb the work; name what you searched and why nothing fit.

## Authority boundary

The context command is read-only. Do not edit derived scope, milestone rollups, or graph projections. Mutations require an operator-approved Swarm proposal or a declared CLI command.

## No known operational edge cases for standard usage.
