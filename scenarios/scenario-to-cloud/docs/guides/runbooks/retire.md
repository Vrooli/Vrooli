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
scenario-to-cloud deployment resolve --scenario <scenario-id> --environment <env>
scenario-to-cloud operation list <deployment-id> --json
```

No non-terminal operation may be open. Confirm you are retiring the right
environment: a scenario's staging and production are separate deployments
with separate ids and target keys.

## 2. Protect the data you might need later

```bash
scenario-to-cloud deployment recovery-points capture <deployment-id> --json
scenario-to-cloud deployment recovery-points verify <deployment-id> --recovery-point <rp-id> --open
```

Retirement with `retention_policy: retain` leaves every binding in place on
the target, and every recovery point is retained by the retirement plan; a
verified recovery point is still the only copy that survives the host
itself, so copy it out before the provider resources go (§6).

## 3. Plan the retirement

```bash
curl -sS -X POST "$STC_API/api/v1/deployments/<deployment-id>/retire/plan" -H "Authorization: Bearer $STC_TOKEN" -H "Content-Type: application/json" -d '{"retention_policy":"retain"}'
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
curl -sS -X POST "$STC_API/api/v1/deployments/<deployment-id>/retire/apply" -H "Authorization: Bearer $STC_TOKEN" -H "Content-Type: application/json" -d '{"retention_policy":"retain","plan_digest":"<plan-digest>","request_key":"retire-<deployment-id>-<date>"}'
scenario-to-cloud operation wait <operation-id> --timeout 600
```

`plan_digest_mismatch` means the listing changed since you read it: plan
again. The operation records `desired_state: retired` on success; observers
(health, drift, autoheal) treat `retired` as unchanged and never restart it.

## 5. Verify

```bash
scenario-to-cloud edge status <deployment-id> --json          # no route for the public host
scenario-to-cloud deployment health <deployment-id> --json    # workload not running is the expected state
curl -sS "$STC_API/api/v1/deployments/<deployment-id>/desired-state" -H "Authorization: Bearer $STC_TOKEN"
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
scenario-to-cloud deployment delete <deployment-id> --cleanup
```

## Record

Deployment id, retention policy, plan digest, operation id, protected
recovery-point id, the retained-artifact listing, DNS removal time.
