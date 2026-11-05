---
name: "data-backup-manager"
description: "Use Data Backup Manager to register owned state, run encrypted backups, verify recovery points, and restore only into clean targets with durable evidence."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["tools"]
  tags: ["backup", "restore", "recovery", "data-backup-manager", "retention"]
  icon: "database"
  status: "active"
  revision: 1
  createdAt: "2026-09-11T00:00:00Z"
  updatedAt: "2026-09-11T00:00:00Z"
  requires:
    scenarios: ["data-backup-manager", "vrooli-memory", "prompt-manager"]
    commands: ["data-backup-manager", "vrooli-memory journal note", "prompt-manager skill read"]
  origin:
    kind: "authored"
---

## Tools focus: Data Backup Manager

Use Data Backup Manager when state owned by a scenario must be registered, backed
up, verified, retained, or restored. The manager owns encrypted repository
operations and restore evidence; callers provide ownership, target identity, and
the expected logical data. Command flags and typed request fields are defined by
`data-backup-manager --help` and its generated endpoint contract.

### 1. Scope

In scope: registering owned targets, configuring destinations and plans, starting
backup runs, inspecting recovery points, verifying artifacts, running restores,
and proving retention behavior.

Out of scope: inventing a second backup engine, copying a target's database by
hand, interpreting a remote path as a local filesystem path, or declaring a
restore successful from transport status alone.

### 2. Decision tree

```
I need to protect or recover state
├─ Is the state owner and target locator declared?
│  ├─ no → register the owned target, then stop until its identity is durable [S1]
│  └─ yes
├─ Is an encrypted destination available and healthy?
│  ├─ no → create or repair the destination through the manager, then verify readiness [S1]
│  └─ yes
├─ Is this a normal protection run?
│  ├─ yes → start one durable backup run and inspect its receipt [S1]
│  └─ no
├─ Is this a recoverability check?
│  ├─ yes → verify the recovery point, including keys and artifact digests, before restore [S1]
│  └─ no
├─ Is the replacement target clean, owned, and schema-compatible?
│  ├─ no → refuse the restore and return the typed prerequisite to its owner [S1]
│  └─ yes → restore through the manager and accept only a passing logical-data receipt [S1]
```

A caller timeout is not a backup cancellation. Reattach to the durable run and
read its receipt. If an effect is uncertain, reconcile it before replaying it.
Missing keys, corrupt artifacts, mismatched target generations, and non-clean
restore targets are refusals, not partial successes.

### 3. Verification

For a backup, confirm the owner, target binding, captured inventory, schema
version, encryption generation, artifact digests, and durable run standing. For a
restore, confirm every binding's rows or files, expected checksums, application
health, measured RTO/RPO, and the target receipt. Keep the recovery point outside
the original host failure domain and protect points referenced by active releases
or operations before pruning.

### 4. After acting

Record the operation ID, recovery-point ID, target generation, receipt standing,
and any refused prerequisite in the owning scenario's normal work record. Do not
copy secret material or repository credentials into the journal.

### 5. In-use settings

| Symptom | Setting move |
|---|---|
| A run is still executing | Reattach to its durable run; journal the run ID and observed standing. |
| Verification cannot resolve a key | Restore the credential-authority reference, then rerun verification; journal the key-generation mismatch. |
| Restore refuses a target | Prepare an owned clean namespace and compatible schema; journal the refusal code before retrying. |
| Retention wants to delete a referenced point | Keep the point and reconcile the active reference before changing retention. |

Promotion note: repeated operator joins should become a typed manager command or
governed program. Until that contract exists, keep the decision tree here and use
the manager's machine-readable receipts as the source of truth.
