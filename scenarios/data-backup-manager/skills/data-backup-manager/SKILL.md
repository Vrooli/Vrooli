---
name: "data-backup-manager"
description: "Use Data Backup Manager to register owned state, run encrypted backups, prove restores, and keep every destination writable or deliberately migrated."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["tools"]
  tags: ["backup", "restore", "recovery", "data-backup-manager", "retention", "coverage", "destinations"]
  icon: "database"
  status: "active"
  revision: 3
  createdAt: "2026-09-11T00:00:00Z"
  updatedAt: "2026-11-19T00:00:00Z"
  requires:
    scenarios: ["data-backup-manager", "vrooli-memory", "prompt-manager"]
    commands:
      - "data-backup-manager"
      - "vrooli-memory journal note"
      - "prompt-manager skill read"
  origin:
    kind: "authored"
---

## Tools focus: Data Backup Manager

Protect and recover scenario-owned state through one encrypted repository manager. Prefer a typed manager command over a hand-copied artifact, and accept only a durable run receipt or a verified restore as proof.

Required reading:
- `prompt-manager skill read safe-operator-script` — use before running or editing `path:scenarios/data-backup-manager/.private/scripts/repair-drive.sh`, and before authoring any new host-maintenance script for this scenario.
- `path:scenarios/data-backup-manager/docs/internal/INVARIANTS.md` — the write-verdict, protection, and repair rules this skill routes to.

### 1. Scope

In scope: registering owned targets, creating and inspecting destinations, binding plans, starting runs, reading coverage, running drills, restoring into clean targets, retaining snapshots, repairing a destination, and migrating a destination's filesystem.

Out of scope: inventing a second backup engine, copying a target's database by hand, treating a snapshot list as a restore, counting destination rows as storage devices, or formatting a disk without a person.

### 2. Decision path

For a combined read of registered targets and plans, run `program-runtime library run data-backup-manager.catalog-read --json`; add `--input owner=<scenario>` only when the target inventory should be filtered. The program follows up to four 50-row pages for each list. Treat `partial` plus a non-empty token as continuation, and continue with the corresponding owner list command using `--page-token` if needed. This is read-only; the bounded scan reports whether either list continues.

For protection or recovery, use the owner CLI after this quick check:

- Unregistered target: review the coverage report; register only after resolving its owner and locator.
- No enabled plan: create or update a plan after reviewing its target and destination set.
- Destination concern: inspect readiness; a write probe determines whether an unknown filesystem state blocks writes.
- Backup now: trigger one run and read its durable receipt; a timeout does not cancel it.
- Recovery proof: preview a drill, then accept only a verified drill or verified restore.
- Drive removal or repair: use the safe-eject or repair workflow below; neither is implied by a catalog read.

A caller timeout is not a cancellation. Reattach to the durable run and read its receipt. An uncertain effect is reconciled before it is replayed.

### 3. What "protected" means

| Successful copies | State | Operator action |
|---|---|---|
| Two or more distinct storage devices | Protected | None. |
| One storage device | Single-copy warning | Add a second device when the data matters. This is not a failure. |
| No successful copy | Unprotected | Fix the plan, the destination, or the source. This is the issue. |

Destination rows are not devices. Two destinations on one drive are one copy. Separate drives on one host protect against drive loss; off-host placement is a stronger claim that this host does not infer from a URL or backend name.

### 4. Write admission

One verdict decides every write. A dirty or needs-check filesystem, a read-only mount, write protection, or a failed bounded write probe blocks the write. An unobservable filesystem state with a passing probe is a warning, not a block. Copy count never gates a write: refusing the only copy leaves the data with fewer copies.

On Linux, NTFS has no metadata repair. `ntfsfix` resets the dirty flag and schedules a Windows check that never runs. Do not call that a repair, and do not offer it for unreadable metadata. Use ext4 for a destination this host must be able to repair.

### 5. In-use settings

| Symptom | Setting move |
|---|---|
| A destination is full under an alert+block cap | Raise the cap or reduce retention; the manager never evicts a backup silently. |
| A run is still executing | Reattach to its durable run and journal the run ID and standing. Do not start a second run for the same plan. |
| A drill is not eligible yet | Read the preview reason; wait for the plan's cadence or run a normal backup first. Do not lower the drill requirement. |
| A restore target is not clean | Prepare an owned, empty namespace with a compatible schema; journal the refusal code. |
| A destination is on a filesystem this host cannot repair | Run `destinations repair-options`, then migrate with `.private/scripts/repair-drive.sh` (it copies and restores the whole volume and re-binds every destination on that drive) or accept the risk with `--allow-unrepairable`. |
| The drive must be unplugged | `destinations safe-eject --id <id> --apply`; do not yank a mounted drive. |

### 6. After acting

Record the operation ID, run ID, snapshot ID, drill ID, destination change, and the coverage delta in the owning scenario's normal work record. Do not copy secret material or repository credentials into the journal.

### 7. Troubleshooting & Edge Cases

| Symptom | Likely cause | First check | Fix |
|---|---|---|---|
| `destination filesystem readiness is unknown` on a mounted drive | The filesystem exposes no dirty state, or no probe ran | `destinations readiness --location <bundle>` | If the probe passes, continue; the state is a warning, not a block. Do not reformat for this message alone. |
| `repair=limited` for a destination | The filesystem is not repairable on this host | `destinations repair-options --id <id>` | Migrate with `.private/scripts/repair-drive.sh --auto`; it probes in-place feasibility first and uses the safe staging copy when available. Do not run `ntfsfix` and call it a repair. |
| `destinations create` refuses a filesystem | The filesystem has no metadata repair here | The refusal names the filesystem and the risk | Use ext4, or pass `--allow-unrepairable` only after accepting that one unsafe shutdown can strand the drive. |
| A run is blocked by "readiness is unknown" although the drive is writable | The write probe did not run or did not pass | `destinations readiness` and the run receipt | Restore writable access; if the probe passes and the run still blocks, report the regression. |
| `safe-eject` refuses with no stable identity | The destination has no UUID or serial | `destinations get <id>` | Re-bind the volume identity, then eject. |
| Host volume lease timeout | An orphan lease process holds the flock | `lslocks` and the process command | Stop the owning operation; do not kill a writer mid-run. |
| Health still shows old failures | Health counts the last failed run per target | `runs list` and the target's last success | A successful run clears the entry; do not chase historical counts. |
| A drill passes but coverage still says unverified | Coverage requires every enabled destination copy to be verified | `coverage report --json` | Verify the remaining copies, or reduce the plan to the verified destination. |

Promotion note: a workflow that repeats unchanged (for example, destination migration) belongs in a governed program under this scenario, not in a longer skill. Record the candidate in `path:scenarios/data-backup-manager/docs/internal/PROBLEMS.md`.
