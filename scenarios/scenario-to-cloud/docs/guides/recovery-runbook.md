# Recovery runbook

Generated from `api/backup.Runbook` by `TestRunbookDocumentIsRendered`; edit the table, not this file.

Code rollback and data restoration are separate operations with different prerequisites. A backup receipt is not proof that the application can restore from it; only a restore receipt with passing invariants is. Every verb below is fenced and receipted per (operation, step) on the target, prints typed JSON, and exits 0 ok / 1 failed / 2 refused.

| Situation | Owner verb | Refuses |
|---|---|---|
| Capture a recovery point before a destructive schema change or activation | `cloud-target:data.backup` | `fence_stale`, `receipt_input_mismatch`, `backup_provider_unavailable`, `recovery_key_unavailable`, `invalid_argument` |
| Verify a recovery point (checksums, key availability, expected inventory) without restoring | `cloud-target:data.verify` | `recovery_point_corrupt`, `recovery_key_unavailable`, `verify_failed` |
| Restore a recovery point onto a replacement host | `cloud-target:data.restore` | `recovery_point_corrupt`, `recovery_key_unavailable`, `restore_target_not_clean`, `fence_stale` |
| Roll back code when the predecessor cannot read the current schema | `scenario-to-cloud:rollback-admission (api/backup.EvaluateRollback)` | `rollback_incompatible (carries a typed forward_repair or restore plan)` |
| Prune recovery points under a retention policy | `scenario-to-cloud:recovery-points.prune (api/backup.Service.Prune)` | `recovery_point_protected` |

## 1. Capture a recovery point before a destructive schema change or activation

Owner verb: `cloud-target:data.backup`

```
vrooli cloud-target data backup --deployment "$DEPLOYMENT_ID" --operation "$OPERATION_ID" --step data.backup --fence "$FENCE" --binding "$BINDING_JSON" --key-ref "$KEY_REF" --schema-version "$SCHEMA_VERSION" --configuration-digest "$CONFIGURATION_DIGEST" --credential-version-ref "$CREDENTIAL_VERSION_REF" --provider data-backup-manager --migration-posture "$MIGRATION_POSTURE"
```

Preconditions:

- Every binding is declared in the closure persistent_data with an owner and a provider; a database is captured only by its owner's database-native tooling (resources/postgres deployment.backup).
- The recovery key reference resolves through the credential authority on the operator side, never on the target being protected.
- The application quiesce hook is declared when the scenario owns migrations; otherwise the point records write_quiescence not_declared (snapshot_safe for database-native providers).

Refusals: `fence_stale`, `receipt_input_mismatch`, `backup_provider_unavailable`, `recovery_key_unavailable`, `invalid_argument`

Evidence: Receipt operations/<op>/data.backup.json and recovery-points/<id>/recovery-point.json (bindings, refs, checksums, consistency token, digest).

## 2. Verify a recovery point (checksums, key availability, expected inventory) without restoring

Owner verb: `cloud-target:data.verify`

```
vrooli cloud-target data verify --deployment "$DEPLOYMENT_ID" --recovery-point "$RECOVERY_POINT_ID" [--open] [--expect "$EXPECTED_JSON"]
```

Preconditions:

- --open resolves the recovery key and decrypts every artifact: run it from the operator side to prove the key lives outside the target failure domain.

Refusals: `recovery_point_corrupt`, `recovery_key_unavailable`, `verify_failed`

Evidence: Verify report: artifacts_intact, key_resolved, artifacts_opened, invariants[].

## 3. Restore a recovery point onto a replacement host

Owner verb: `cloud-target:data.restore`

```
vrooli cloud-target data restore --deployment "$DEPLOYMENT_ID" --operation "$OPERATION_ID" --step data.restore --fence "$FENCE" --recovery-point "$RECOVERY_POINT_ID" --into "$BINDING_LOCATOR"
```

Preconditions:

- The replacement host holds the release (release verify + stage) and the credentials the workload needs (credentials.provision) before data is restored.
- Every target binding is clean: an empty directory or a database without user tables. Restore never overwrites operator-owned data.
- The workload is stopped or not yet started; restoring under writes is not a supported path.

Refusals: `recovery_point_corrupt`, `recovery_key_unavailable`, `restore_target_not_clean`, `fence_stale`

Evidence: Receipt operations/<op>/data.restore.json with measured_rto_ms, recovery_point_age_ms and per-binding captured/restored inventories; the cloud restore receipt records the same against certification/budgets.json.

## 4. Roll back code when the predecessor cannot read the current schema

Owner verb: `scenario-to-cloud:rollback-admission (api/backup.EvaluateRollback)`

```
POST /api/v1/deployments/{id}/recovery-points/rollback-admission {target_schema, current_schema, readable_by[]}
```

Preconditions:

- The scenario declares deployment.recovery.{code_rollback, schema_strategy}; admission fails closed when a schema version is unobserved.

Refusals: `rollback_incompatible (carries a typed forward_repair or restore plan)`

Evidence: RollbackVerdict {compatible, reason_code, plan{kind, recovery_point_id, preconditions[], steps[]}}.

## 5. Prune recovery points under a retention policy

Owner verb: `scenario-to-cloud:recovery-points.prune (api/backup.Service.Prune)`

```
POST /api/v1/deployments/{id}/recovery-points/prune {keep_last, max_age_seconds}
```

Preconditions:

- Points referenced by the active or retained release, by a non-terminal operation or by a pin are protected and never deleted; the plan names their holders.

Refusals: `recovery_point_protected`

Evidence: PrunePlan {keep[], delete[], protected{id: holders[]}}.

## Budgets

Measured against `certification/budgets.json` qualification: `fresh_host_restore_seconds_max` (RTO) and `backup_recovery_point_seconds_max` (recovery-point age at restore start). A weaker budget may not be introduced after a failed drill.

## Recovery keys

Recovery points are sealed with AES-256-GCM (`internal/credentialpolicy.Seal`) under a key derived (PBKDF2-SHA256) from material resolved through the credential authority by reference (`logical_id:field`). The reference is recorded on the manifest and every receipt; the material never is. A restore on a replacement host needs the reference to resolve there, which is why the key lives with the operator's credential store, not on the protected target.
