# Runbook: restore data from a recovery point

Outcome: a verified recovery point restored onto the same target or a
replacement, with a restore receipt that records the measured recovery
time (RTO) and the recovery-point age (RPO) against the declared budgets,
followed by a healthy current observation. Verb reference (rendered from
the owner): `docs/guides/recovery-runbook.md`.

Code rollback and data restore are different operations with different
prerequisites. A backup receipt proves a point was captured; only a restore
receipt with passing invariants proves the application can come back from it.

## 1. Choose the recovery point

```bash
scenario-to-cloud deployment recovery-points list <deployment-id> --json
```

Pick by `captured_at`, `release_digest` and `schema_version`: the point must
be readable by the release you intend to run afterwards. The newest point is
not automatically the right one after a schema change.

## 2. Verify before touching anything

```bash
scenario-to-cloud deployment recovery-points verify <deployment-id> --recovery-point <rp-id> --open
```

`--open` also proves the recovery key resolves through the credential
authority on the operator side. Refusals stop the procedure:
`recovery_point_corrupt` (choose an older point), `recovery_key_unavailable`
(recover the key custody first; `rotate.md` §5 if the store itself is gone).

## 3. Dry run

```bash
scenario-to-cloud deployment recovery-points restore <deployment-id> --recovery-point <rp-id> --dry-run --json
```

Prints what would be restored (bindings, paths, the invariants that will be
checked) and calls no restore. On a replacement host add
`--target-ref <machine-or-host>`; the dry run then also proves the target is
clean (`restore_target_not_clean` otherwise: an occupied binding path is
never overwritten).

## 4. Stop the workload (same-host restore)

A running workload writing into the binding during restore corrupts the
result. Stop it through the owner and confirm:

```bash
scenario-to-cloud deployment stop <deployment-id>
scenario-to-cloud deployment health <deployment-id> --json      # application_readiness failed is expected now
```

The stop records `desired_state: stopped`; observers never restart a stopped
deployment on their own (`docs/reference/activation-and-reconciliation.md` §7).

## 5. Restore

```bash
scenario-to-cloud deployment recovery-points restore <deployment-id> --recovery-point <rp-id> --request-key restore-<deployment-id>-<rp-id>
scenario-to-cloud deployment recovery-points restore <deployment-id> --recovery-point <rp-id> --target-ref <machine-or-host> --request-key restore-<deployment-id>-<rp-id>   # replacement host
```

The restore is a durable operation (wait/resume rules as everywhere).
The receipt carries: measured RTO from the moment the target was available,
recovery-point age at restore start, and each declared invariant with its
result. A failed invariant is a failed restore even if the bytes landed.

## 6. Bring the application back

Same host:

```bash
scenario-to-cloud deployment start <deployment-id> --yes --timeout 600
```

Replacement host: the deployment's target binding now points at the new
machine; run the install-scope plan so the release, configuration, edge and
credentials (`rotate.md` §5 first when the store was lost) are put in place
around the restored data:

```bash
scenario-to-cloud deployment plan <deployment-id>
scenario-to-cloud deployment apply <deployment-id> --plan-digest <plan-digest> --request-key rebuild-<deployment-id>-<date> --timeout 900
```

The plan binds the restored persistent data by recorded mapping; it never
copies or deletes it.

## 7. Verify

```bash
scenario-to-cloud deployment health <deployment-id> --json
scenario-to-cloud edge status <deployment-id> --json
```

Required: `HEALTHY`/`CURRENT` at the intended release; the application's own
acknowledged-write checks (the fixture oracle, or your own) pass.

## 8. Record

Recovery-point id and age, restore operation id, measured RTO, whether the
budgets (`docs/reference/support-policy.md`: RTO ≤ 15 min after target
available, RPO ≤ 5 min for the protected write stream) were met, and the
observation timestamp. A missed budget is recorded as missed.

## When restore is the wrong tool

- **Bad code, good data**: `update.md` §6a (governed rollback).
- **Schema moved forward and the predecessor cannot read it**: restore the
  pre-migration point, then roll back; `rollback_incompatible` carries the
  typed forward-repair or restore plan.
- **Host reachable, workload down**: `incident.md` §3 start scope.
