# Offer Desk architecture

Offer Desk is the intent and decision side of the monetization pair. It owns typed offer-graph nodes, lifecycle transitions, triggers, facts, evaluations, proposals, and the read-time board. Money Ledger owns actual money; this scenario reads it through a bounded typed client and never writes or caches financial facts.

## Runtime shape

`api/handlers/offers/` exposes Catalog, Gates, and Board Connect services. `api/internal/catalog/` owns the graph schema and state machine. The graph accepts only the typed edge matrix. A candidate needs an attached machine-evaluable trigger. The evaluator records unknown for missing or stale facts and moves satisfied candidates to `trigger-met`; the scheduler invokes that evaluator without a caller.

Promotion is role-checked. Agents can create proposals; only an operator can transition a trigger-met node to active. Refusals name the rule and legal next states. The board ranks catalog state and joins optional Money Ledger actuals and posture, preserving source availability and partial qualification.

The CLI, UI, and `space --projection offers --json` consumer surface use the generated proto vocabulary. The scenario remains useful when Money Ledger is unreachable: catalog rows continue to rank and the board names the unavailable source instead of showing zero earnings or zero runway.
