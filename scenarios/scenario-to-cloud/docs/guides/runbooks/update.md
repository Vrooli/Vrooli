# Runbook: update a deployment to a new release

Outcome: the target runs the new release, the predecessor is retained for
rollback, persistent data survived, and a current observation proves it.
Time: 3–10 minutes; declared maintenance downtime is bounded by the plan.

## 1. Identify the deployment and its current state

```bash
scenario-to-cloud deployment resolve --scenario <scenario-id> --environment production
scenario-to-cloud deployment health <deployment-id> --json
scenario-to-cloud operation list <deployment-id> --json
```

Do not start an update while `operation list` shows a non-terminal
operation (`running`, `reconciling`, `recovering`, `waiting_input`): finish or
cancel it first (`incident.md` §2). An unhealthy or stale current state is not
a reason to skip the update, but record it: the post-update verification
must not be compared against a green baseline that did not exist.

## 2. Protect the data (stateful workloads)

The plan captures a recovery point itself (`data.backup`) before
activation; on a schema-changing update it also requires one at the current
schema (`recovery_point_required` precondition). Capture one explicitly when
you want a named restore point independent of the operation:

```bash
scenario-to-cloud deployment recovery-points capture <deployment-id> --json
scenario-to-cloud deployment recovery-points verify <deployment-id> --recovery-point <rp-id> --open
```

The point is bound to the recorded release; retention protection is
reference-based (`docs/reference/activation-and-reconciliation.md` §9): the
point stays protected while that release is active or retained as the
rollback predecessor, or while an operation holds it, and pruning never
deletes a protected point. `verify --open` proves the recovery key resolves,
not just that the bytes are intact.

## 3. Review the update plan

```bash
scenario-to-cloud deployment plan <deployment-id> --force-bundle
```

`--force-bundle` rebuilds and re-signs the release from the current source;
without it the recorded release is planned again (which yields `no_op` when
the target already runs it). Read, in this order:

| Field | Decide |
|---|---|
| `outcome` | `no_op`: stop, nothing to change. `needs_input`: supply the handoff, re-plan. `apply`: continue |
| `release_digest` vs `observed_release_digest` from step 1 | this is the change you are making |
| activation strategy | `maintenance` (persistent data, legacy carries or pinned ports; stops the workload for the declared bound) or `side_by_side` (no declared downtime) |
| `release.retain_predecessor.rollback_eligible` and `rollback_reason` | `predecessor_retained` / `no_schema`: rollback is one command. `explicit_restore_required` / `schema_strategy_undeclared` / `code_rollback_not_declared`: rollback will be refused; a bad release means a restore (step 6b). Decide now whether that is acceptable |
| data effects | bindings bound by rename, `legacy_carry` paths carried by heuristic, `legacy_unmapped` directories that will be left in place |
| downtime | the bound the operator communicated |

## 4. Apply the reviewed digest

```bash
scenario-to-cloud deployment apply <deployment-id> --plan-digest <plan-digest> --request-key update-<scenario-id>-<release-short> --timeout 900
```

Exit handling is the same as in `deploy.md` §5. Activation order on the
target is stage → backup → stop (maintenance only) → activate (intent file,
data bind, runtime switch, pointer commit) → route apply → readiness →
predecessor retained (`docs/reference/activation-and-reconciliation.md` §4).
A crash between the runtime switch and the pointer commit leaves an
`interrupted_activation`; reconciliation (step 6c) completes it under a new
operation, it is never completed by hand.

## 5. Verify

```bash
scenario-to-cloud deployment health <deployment-id> --json
scenario-to-cloud edge status <deployment-id> --json
scenario-to-cloud inspect drift <deployment-id> --json
```

Required: `status: HEALTHY`, `freshness: CURRENT`, `observed_release_digest`
equal to the plan's `release_digest`, the public route unchanged (host,
upstream, listener id), private listeners still unrouted, `drift` reporting
`unchanged`.

## 6. If the release is bad

**a. Roll back to the retained predecessor** (eligible per step 3):

```bash
scenario-to-cloud deployment rollback <deployment-id> --dry-run --json
scenario-to-cloud deployment rollback <deployment-id> --confirm --preview-ref <preview-ref> --request-key rollback-<scenario-id>-<release-short>
```

Rollback is governed: the dry run issues a preview reference bound to the
predecessor's published evidence; `--confirm` executes exactly that
preview. `rollback_not_eligible` and `rollback_incompatible` are refusals
with the reason, not attempts.

**b. Restore data** when the predecessor cannot read the new schema
(`data_compatibility` in the refusal): follow `restore.md` with the recovery
point from step 2 or the one the operation captured, then roll back.

**c. Interrupted activation** (`activation_interrupted` in `inspect drift`,
reconciliation `blocked` with `resolve_interrupted_activation`):

```bash
curl -sS -X POST "$STC_API/api/v1/deployments/<deployment-id>/reconcile" -H "Authorization: Bearer $STC_TOKEN" -H "Content-Type: application/json" -d '{}'
```

The response carries the report and the compiled correction plan; resubmit
its `plan_digest` with a `request_key` in the same call to admit it, then
`operation wait <operation-id>`.

## 7. Retention afterwards

The pre-update point remains protected as long as the predecessor release
is retained for rollback; once a later update retires that predecessor, the
retention policy (`docs/reference/service-objectives.md`) may prune it. Keep
a copy outside the target if you need it longer.

## Record

Deployment id, previous and new release digests, plan digest, operation id,
recovery-point id, rollback eligibility as previewed, verification timestamp.
