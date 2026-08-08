# Agent event attribution

Swarm Manager persists two different identifiers for a verified agent write:

- `actor_id` is the verified agent profile key (`ProfileKey`). It identifies
  the team member or profile that filed the item.
- `run_id` is the verified Agent Manager execution UUID. It identifies the
  particular run and is never used as the actor identity.

The request middleware resolves `VROOLI_AGENT_IDENTITY_TOKEN` through
Agent Manager. Durable request handlers pass that context to the event emitter,
which records `verification_status=verified` only after that server-side check.

## Writers that can carry verified agent identity

These request-bound writers receive the middleware context and use the
request-aware emitter path:

- backlog item creation, including Connect and batch creation;
- backlog status, archive, retry, recovery, and review transitions;
- record supersession;
- review-round failure and completion paths when invoked through an API request.

The backlog projection also stores the same verified provenance in `created_by`,
so callers can select items with `backlog list --actor-id <profile-key>`.

## Writers that cannot carry verified agent identity

These paths intentionally remain unverified because there is no inbound agent
request context:

- startup/schema initialization and migrations;
- scheduler, sweeper, and background maintenance loops;
- imported or replayed harness evidence without an identity token;
- tests and local fixture writers that use a background context.

Those rows retain an explicit non-verified status (`absent`, `invalid`, or
`unavailable`) and must not be counted as agent-attributed writes. Harness
session headers are stored separately as observations and are not actor proof.
