# Research budget report authorization

Treasury's `/api/v1/research-budgets/{run_id}/{snapshot|settlements}` routes
write immutable CF7 budget and settlement evidence. Each write requires the
canonical `research-budget.ingest` operation grant on
`agent-manager:run/<canonical-uuid>`.

When verified Treasury lacks that exact grant, Agent Manager raises one
Notification Hub decision for the exact run and returns `202 approval_pending`.
The request does not issue authority. A separate explicit human approval in the
existing decision flow can issue a Treasury-only grant for that one operation
and one run, with a maximum one-hour lifetime. Rejection, expiry, missing human
identity, stale decisions, and duplicate resolutions issue no grant. The next
Treasury report attempt retries the operation through the canonical
`RequireOperation` evaluator.

The approval question names the exact run target and says that the grant covers
reporting only. It cannot spend funds, authorize another run, or broaden the
Treasury service's capabilities. This path is limited to a persisted Agent
Manager run carrying the `cf7-test-only` tag, and the approval question names
the exact target. The operator must still inspect the run before approving.
The build-path decision itself did not approve a run or issue a grant.
