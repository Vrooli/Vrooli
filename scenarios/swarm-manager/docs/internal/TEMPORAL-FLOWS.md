# Temporal Flows Documentation

## Declared workflow lifecycle

1. An operator or deterministic domain event selects a transition from the
   registry.
2. Swarm Manager authorizes the request, preflights integrations, and builds an
   immutable input snapshot with an idempotency key.
3. Agent Manager starts a pinned declared workflow revision and records its
   journal.
4. Swarm Manager collects a typed terminal result, verifies that its captured
   domain frontiers are current, and applies it exactly once.
5. A stale, rejected, cancelled, or abstained result is preserved as history
   without a domain mutation.

Workflow waits, branches, retries, child work, and budgets are durable Agent
Manager behavior. Swarm does not poll or reconstruct those mechanics.

## Human session lifecycle

An Agent Session is created before its first message. Operator messages may
continue that session's Run, while session context and proposals are persisted
by Swarm Manager. Historical sessions remain viewable; retired kinds cannot be
created or resumed.

## Recovery

Workflow correlations and application claims make process restart safe. A
terminal result may be collected again after a crash, but the existing claim is
returned rather than applying the result twice. Integration outages defer or
reject starts according to the transition's declared requirements.
