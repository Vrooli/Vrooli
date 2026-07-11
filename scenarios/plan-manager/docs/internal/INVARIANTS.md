# Validation Replay and Idempotency Invariants

## Replay/Idempotency Invariants

- `StartValidation` persists the complete child set before dispatch.
- At most one non-terminal operation exists for `(plan_id, phase_id)`. This
  applies equally to unkeyed starts; a caller key is a retry alias, not the
  duplicate-work safety boundary.
- Known GCT baseline diffs deduplicate by typed semantic key
  `scenario-diff:<scenario>:<baseline>`, independent of `--json` rendering.
  Snapshot status is diagnostic/capture metadata, never a blocking child.
- Operation IDs and terminal result IDs are stable. Re-finalization upserts the
  same result ID rather than accumulating duplicate terminal results.
- Each terminal child checkpoint is skipped during recovery. A queued/running
  child may resume after restart; its downstream GCT command is itself addressed
  by durable baseline/run identity.
- A caller timeout/cancel only detaches `GetValidationOperation(wait=true)`.
  It returns a typed non-success attachment-ended error unless a durable
  terminal reread won the race; server-owned execution continues.
- PASS is impossible until every oracle child is terminal and comparable. No
  oracle, unavailable tools, timeout, or exit-2 produces UNKNOWN.

## Safe Retry Surface

Safe: repeat `validate start` with or without an idempotency key (active starts
coalesce); repeat show; reattach once with wait/resume by operation ID. Unsafe:
interpret a transport error or non-terminal inspection as a verdict.

## Enforcement

`api/internal/validation/service.go`, `checks.go`, `sqlite.go`, and their
race-enabled durable-operation tests enforce these boundaries. SQLite owns the
cross-request active-scope unique index; the service claims a child before it
starts an execution clock and owns child checkpoint locks.
