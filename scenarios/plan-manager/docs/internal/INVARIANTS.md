# Validation Replay and Idempotency Invariants

## Replay/Idempotency Invariants

- `StartValidation` persists the complete child set before dispatch.
- An explicit idempotency key is replay-safe within `(plan_id, phase_id,
  execution_id, scope_generation)` and does not collapse a distinct explicit
  request onto earlier evidence. Only unkeyed starts coalesce while active in
  that same execution-local scope.
- Known GCT baseline diffs deduplicate by typed semantic key
  `scenario-diff:<scenario>:<baseline>`, independent of `--json` rendering.
  Snapshot status is diagnostic/capture metadata, never a blocking child.
- Operation IDs and terminal result IDs are stable. Re-finalization upserts the
  same result ID rather than accumulating duplicate terminal results.
- Plan Manager never resumes a producer child. A queued ticket remains a durable
  request for the agent to reattach through Git Control Tower/Test Genie and
  later synchronize by ticket id.
- PASS is impossible until every oracle child is terminal and comparable. No
  oracle, unavailable tools, timeout, or exit-2 produces UNKNOWN.
- Only one active execution may exist for a plan. Duplicate starts return the
  existing execution IDs plus exact resume/abandon commands; they never create
  a second durable run.
- Abandonment is terminal, reasoned, auditable, and idempotent. Abandoned
  executions cannot resume or contribute completion/velocity accounting, but
  remain queryable and no row is deleted.
- Import supersession creates the replacement, lineage edge, and archival of
  the replaced plan in one repository transaction.

## Authored mutation invariants

- A phase field mask is the only partial-update selector: omission preserves,
  explicit empty clears, and immutable/computed paths are rejected.
- Quality is assessed from an immutable pre-mutation report and the proposed
  post-mutation plan. Active/complete execution-grade regressions require an
  explicit acknowledgement and always return typed impact.
- Every authored reference and relevant-context field repairable during
  authoring remains list/add/update/remove repairable after finalization.

## Safe Retry Surface

Safe: repeat `validate start` with the same explicit key; repeat `show` and
`sync`. Unkeyed active starts coalesce only within the same execution and scope
generation. A different explicit key intentionally requests fresh evidence.
Unsafe: interpret producer start output or a non-terminal sync as a verdict.

## Enforcement

`api/internal/validation/service.go`, `checks.go`, `sqlite.go`, and their
durable-operation tests enforce these boundaries. SQLite owns the scoped replay
index; Git Control Tower and Test Genie own all producer execution clocks and
child checkpoint lifecycle.
