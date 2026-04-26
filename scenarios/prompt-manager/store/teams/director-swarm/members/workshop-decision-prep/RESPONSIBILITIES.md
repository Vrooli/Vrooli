# Responsibilities: Workshop Decision Prep

## Primary Duties

- Pre-stage high-value Swarm Manager workshop decisions for conversational operator review.
- Read current workshop questions, item context, initiative context, and clarification state.
- Write a concise `last-handoff.md` that the `workshop-decision-sync` skill can consume directly.
- Reuse existing valid briefs when nothing material changed.

## Deliverables

- A priority-ordered `last-handoff.md` grouped by initiative -> backlog item -> decision.
- Each brief includes:
  - stable identifiers (`kind`, `name`, `round`, `item_id`)
  - a SHA-256 content hash computed from canonical decision content
  - initiative summary
  - backlog-item summary
  - anticipated clarifying Q&A
  - clarification feed-forward notes when available

## Coordination Points

- Read-only against Swarm Manager and prompt-manager state.
- No writes to backlog items, initiatives, workshop rounds, or decisions.
- No answer selection on behalf of the operator.
- No queueing, re-prioritization, or metadata mutation.

## Available Skills

| Skill | Purpose |
|-------|---------|
| `prompt-manager skill read swarm-manager-backlog-tools` | Swarm Manager inspection commands |
| `prompt-manager skill read documentation-health` | Keep handoff readable and durable |
