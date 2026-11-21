# Runbook: retire a deployment

Outcome: the public route is gone, the workload is stopped, grants are
revoked, data is either retained in place or (explicitly) deleted, retained
artifacts are listed, and the deployment record carries
`desired_state: retired`. Order and dispositions:
`docs/reference/activation-and-reconciliation.md` §8.

Retirement has no CLI verb yet; it uses two REST routes classified as
destructive (`docs/reference/authorization-matrix.md`) with the same bearer
the CLI uses.

## 1. Identify and freeze

```bash
scenario-to-cloud deployment resolve --scenario "$SCENARIO_ID" --environment "$ENVIRONMENT"
scenario-to-cloud operation list "$DEPLOYMENT_ID" --json
```

No non-terminal operation may be open. Confirm you are retiring the right
environment: a scenario's staging and production are separate deployments
with separate ids and target keys.

## 2. Protect the data you might need later

```bash
scenario-to-cloud deployment recovery-points capture "$DEPLOYMENT_ID" --json
scenario-to-cloud deployment recovery-points verify "$DEPLOYMENT_ID" --recovery-point "$RECOVERY_POINT_ID" --open
```

Retirement with `retention_policy: retain` leaves every binding in place on
the target, and every recovery point is retained by the retirement plan; a
verified recovery point is still the only copy that survives the host
itself, so copy it out before the provider resources go (§6).

## 3. Plan the retirement

```bash
curl -sS -X POST "$STC_API/api/v1/deployments/$DEPLOYMENT_ID/retire/plan" -H "Authorization: Bearer $STC_TOKEN" -H "Content-Type: application/json" -d '{"retention_policy":"retain"}'
```

The response lists, in owner order, what will be **deleted** and what will be
**retained**, and the compiled `retire`-scope plan with its `plan_digest`:

| Order | Object | `retain` | `delete` |
|---|---|---|---|
| 1 | route (`edge.route.retire`) | deleted | deleted |
| 2 | runtime (`workload.stop`, scoped: shared resources other deployments demand keep running) | stopped | stopped |
| 3 | grants (`grants.revoke`) | revoked | revoked |
| 4 | data (`data.retire`) | kept in place | **refused** today (`unsupported_capability`) until a target data-retire owner exists; nothing is silently retained under a delete policy |
| 5 | artifacts (`artifacts.retire`) | active, previous and recovery-referenced releases and every recovery point retained; unreferenced staged releases reported for the prune owner | same |

Without `retention_policy` a deployment that holds data is `blocked` and
apply refuses. Read the listing; the digest is the identity of exactly this
listing.

## 4. Apply the reviewed digest

```bash
curl -sS -X POST "$STC_API/api/v1/deployments/$DEPLOYMENT_ID/retire/apply" -H "Authorization: Bearer $STC_TOKEN" -H "Content-Type: application/json" -d "{\"retention_policy\":\"retain\",\"plan_digest\":\"$PLAN_DIGEST\",\"request_key\":\"retire-$DEPLOYMENT_ID-$DATE\"}"
scenario-to-cloud operation wait "$OPERATION_ID" --timeout 600
```

`plan_digest_mismatch` means the listing changed since you read it: plan
again. The operation records `desired_state: retired` on success; observers
(health, drift, autoheal) treat `retired` as unchanged and never restart it.

## 5. Verify

```bash
scenario-to-cloud edge status "$DEPLOYMENT_ID" --json
scenario-to-cloud deployment health "$DEPLOYMENT_ID" --json
curl -sS "$STC_API/api/v1/deployments/$DEPLOYMENT_ID/desired-state" -H "Authorization: Bearer $STC_TOKEN"
```

Required: `desired_state: retired`, `observation_may_restart: false`; the
credential bindings show revoked versions (`credential list`).

## 6. External clean-up the cloud does not perform

- **DNS**: the cloud observes zones, it does not mutate them outside a
  delegated zone. Remove the records at the registrar/provider.
- **Provider resources** (the VPS itself, volumes, snapshots): the
  operator's, out of scope.
- **Target artifacts**: retained releases and recovery points stay until the
  prune owner runs; `bundle vps-gc <deployment-id>` keeps the protected set
  and removes unreferenced bundles.

## 7. Remove the record (optional, last)

Only after the retained data has been moved or is no longer wanted. Deleting
the record deletes the identity every receipt refers to; keep the retirement
receipt and the recovery-point id in the record first.

```bash
scenario-to-cloud deployment delete "$DEPLOYMENT_ID" --cleanup
```

## Record

Deployment id, retention policy, plan digest, operation id, protected
recovery-point id, the retained-artifact listing, DNS removal time.
