One lifecycle authority: The Web Console continuity domain owns every session lifecycle decision and receipt; process management only executes process commands.
Preservation before cleanup: Message-bearing or agent-history-bearing evidence becomes a recoverable tombstone before any retention decision.
Explicit state over absence: Lifecycle state remains queryable after process and workspace removal; a missing row is always integrity drift.
Rebuildable projections: Search indexes and topic summaries are rebuildable; raw agent history and durable conversation evidence are not.
Local versus global search: Web Console owns local continuity search; Agent Manager owns global conversation retrieval; Search Hub owns federation.
Typed replay safety: Every mutating lifecycle command uses a stable operation identifier and returns a durable receipt.
Dry-run reconciliation: Repair applies only against a reviewed reconciliation generation and remains idempotent on replay.
No autonomous destruction: Skills and programs may audit and recommend; permanent delete remains an explicit operator action.
Greenfield cutover: Route all callers through the new lifecycle, prove parity, and delete displaced mutation paths instead of keeping dual authorities.
Topic as metadata: Preserve original titles and topic history; derived topics improve discovery but never control identity, retention, or access.

