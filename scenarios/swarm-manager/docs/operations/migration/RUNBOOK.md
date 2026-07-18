# swarm-manager state migration — safety runbook

> Historical migration runbook. It documents the retired runtime for audit
> purposes and must not be used to operate current Swarm Manager behavior. See
> [Target Operating Model](../../concepts/TARGET-OPERATING-MODEL.md).

Operational procedure for the Phase 8 persisted-state migration of the
"Declarative agent operations for backlog items and initiatives" plan. It exists
to make the migration **backed-up, fenced, staged, verified, and reversible**.

Authoritative state map and reconciliation anchors come from the Phase 1
inventory: `tools/statemigrate-inventory/` (tool) →
`docs/operations/migration/inventory/inventory-phase1.json` (baseline). The tool
is read-only; re-run it to produce pre/post snapshots.

> **Golden rule:** never edit an original document in place. Stage complete
> replacement documents, verify, then atomically swap. If any stop-condition
> (below) fires, restore from backup and abort.

---

## 0. Storage roots in scope

Per the inventory (`internal/runtimepaths` → `api-core/storage`, app `vrooli`,
scenario `swarm-manager`, namespace-aware):

| class | live root | holds |
|---|---|---|
| data | `~/.vrooli/data/vrooli/swarm-manager/` | backlog items (`ideas/ research/ fix/ execute/ chore/<name>/spec.json` + per-item `workshop/`, `review/`, `evidence/`, `clarify/`, `.swarm/`, `acceptance-validation.json`), `initiatives/<name>/{initiative.json,graph.json,review/,.feedback-lock,modes/}`, `goals/<name>/goal.json`, `records/<scenario>/<kind>/<id>.json`, `mode-targets/<targetType>/<id>/modes/<mode>/{executions/,run-owners.json,rounds/}`, `operating-mode-run-owners/run-owners.json`, `events.db` (+`-wal`/`-shm`), `plan-ref-sweep-manifest.jsonl`, `autofiler/`, `auto-drain.json` |
| state | `~/.vrooli/state/vrooli/swarm-manager/` | `agent-activities.json`, `execution-runs.json`, `engagement-owners.json`, `circuit-breaker.json`, `queue.json` |
| cache | `~/.vrooli/cache/vrooli/swarm-manager/` | `captures/<id>/{capture.json,classification.json,attachments/}` (disposable) |
| config | `scenarios/swarm-manager/config/settings.json` | scenario settings (repo, not runtime) |

Notes carried from Phase 1:
- A sibling `*_shadow` namespace, if present, is **separate state** — inventory
  it and migrate (or explicitly exclude) it too.
- `data/deployment/deployment-report.json` is a **foreign artifact** written by
  `scenario-dependency-analyzer`; it is **not** swarm-manager state — exclude
  from migration and reconciliation (inventory flags it `ambiguous_ownership`).
- `events.db` is live SQLite (WAL). Treat as opaque: back it up with the WAL
  checkpointed; never hand-edit. The inventory hashes it but does not open it.

---

## 1. Backup procedure (must succeed before any staging)

Take a safety snapshot of swarm-manager's registered targets (Postgres — none
here — and the data dir) via **data-backup-manager**:

```sh
# 1a. Ensure the reserved safety destination exists (idempotent).
data-backup-manager safety ensure-destination --json

# 1b. Ensure swarm-manager's targets are registered (derives from service.json).
data-backup-manager safety register-targets --scenario swarm-manager --json

# 1c. Take the safety backup NOW. Keep-latest 0 => keep all pre-migration snapshots.
data-backup-manager safety backup-now --scenario swarm-manager --keep-latest 0 --json
```

- Record the returned snapshot ids in the migration epoch marker (§4).
- **State + cache classes are not covered by data-dir backup if they live outside
  the data root.** Additionally take a raw filesystem copy of the `state` and
  `cache` roots (and any `*_shadow` roots) to a scratch location, since the
  migration touches `execution-runs.json`/`engagement-owners.json`/etc.:
  ```sh
  cp -a ~/.vrooli/state/vrooli/swarm-manager  <backup-scratch>/state
  cp -a ~/.vrooli/cache/vrooli/swarm-manager  <backup-scratch>/cache
  ```
- **Checkpoint the SQLite WAL** before copying `events.db` so the snapshot is
  consistent (the migration itself must not run while the API holds the DB — see
  the quiescence fence).

**Stop condition:** if backup 1c fails, or the state/cache copy fails, or the
epoch marker cannot be written — **do not proceed**.

---

## 2. Quiescence / maintenance fence (freeze writers)

The migration must run against a **quiescent** store: nothing may write runtime
state while documents are being staged and swapped. Enter the fence and confirm
each writer is stopped:

1. **Stop scheduling / autonomous loops.** Disable auto-drain
   (`data/auto-drain.json` → disabled) and the auto-filer, and stop the swarm
   scheduler so no new runs are queued.
2. **Stop workshop auto-advance.** No new workshop/clarification rounds may be
   spawned (freezes writes to `<item>/workshop/**`).
3. **Stop execution polling.** The execution poller mutates `execution-runs.json`,
   `circuit-breaker.json`, `engagement-owners.json`, and item statuses — it must
   be idle.
4. **Stop review + initiative-review spawning.** No new review rounds
   (`<item>/review/**`, `initiatives/<name>/review/**`) or operating-mode phase
   rounds (`mode-targets/**`, `initiatives/<name>/modes/**`).
5. **Fence mutation endpoints.** Put the API into maintenance so backlog/
   initiative/goal/settings mutation handlers reject writes (or stop the scenario
   entirely — preferred, and required to release the `events.db` lock for a
   consistent copy).
6. **Confirm no active unowned runs.** Re-run the inventory; `expected_absent`
   should show no in-flight `execution-runs.json` running rows, and there must be
   **no** `orphaned_execution` finding and **no** open `engagement-owners`
   entries. Active runs during migration is a stop condition.

Verify the fence held by taking the **pre-migration inventory** (§4) twice and
confirming the `totals.content_hash` is stable across the two runs *and* across a
30-second gap. A changing hash means a writer is still live — do not proceed.

---

## 3. Staging strategy (stage replacements, never edit in place)

1. For every document to be transformed, write the **complete replacement** to a
   staging tree that mirrors the live layout (e.g.
   `<migration-scratch>/staged/<class>/…`). Never open an original for writing.
2. Stage atomically per file: write `*.tmp` in the staging tree, fsync, then the
   final swap into the live root is a single `rename(2)` (same filesystem) during
   the cutover step — mirroring `storage.WriteJSONAtomic`.
3. Preserve identities exactly. Identity = **folder name** for backlog items
   (`kind/name`), initiatives (`name`), goals (`name`); `id` for records,
   captures, operating-mode executions; array position is not identity. Renaming
   a backlog-item or initiative folder breaks every `depends_on`, `items`, and
   `initiative` back-reference — do not rename during migration.
4. Regenerate, don't migrate, **derived** artifacts: `initiatives/<name>/graph.json`
   is a rebuildable projection of item `depends_on`; stage it fresh rather than
   transforming it.
5. Leave **disposable cache** (`captures/**`) and **foreign**
   (`deployment/**`) artifacts untouched.

---

## 4. Migration epoch marker

Write one marker file at the start, under the migration scratch (NOT a runtime
root), capturing the reversible frame:

```
docs/operations/migration/epoch-<n>.json
{
  "epoch": <n>,
  "backup_snapshot_ids": [...],           // from §1c
  "state_cache_backup_path": "<backup-scratch>",
  "pre_inventory_content_hash": "sha256:…",   // from §5 pre-run
  "pre_inventory_counts": { "backlog_item": N, "initiative": N, ... },
  "fence_confirmed": true
}
```

The epoch marker is the single source of truth for rollback (§7). Do not start
the swap without it.

---

## 5. Verification (pre/post reconciliation)

Reconciliation uses the Phase 1 inventory tool as the oracle.

**Pre-migration:** run the tool against the fenced live roots →
`inventory/pre-migration.json`. Record `totals.content_hash`, per-class `count`,
`plan_refs`, and the referential-findings/anomaly counts into the epoch marker.

**Post-migration** (after the swap, before lifting the fence): run the tool again
→ `inventory/post-migration.json`, then reconcile:

- **Object counts** per primary class match the intended transform (unchanged
  where the migration is a reshape, or match the planned delta exactly — no
  unexplained drift).
- **Identities**: the set of stable identities per class is preserved (diff the
  `objects[].identity` sets); any add/drop must be intended and listed in the
  plan.
- **Referential integrity does not regress**: post-migration
  `referential_findings` is a subset of pre-migration (migration may *fix*
  danglers; it must not introduce new ones). **Zero new anomalies.**
- **plan-ref usage**: `plan_refs.unmanaged` does not increase.
- **Ownership**: no new `ambiguous_run_owners`.
- **Content-hash accounting**: every file whose content changed is an intended
  target of the migration; hash-diff the two inventories' per-class
  `content_hash` and per-object `sha256` and explain every difference. Untouched
  classes must have identical `content_hash`.
- **UI/API projection parity**: after lifting the fence on a **copy/staging**
  instance first, confirm the backlog list, initiative detail, goals, and
  operating-mode surfaces render the same entities/counts the inventory reports.

**Stop condition:** any unexplained count/identity drift, any new anomaly or
referential finding, any unexplained content-hash change, or projection mismatch
→ roll back (§7).

---

## 6. Operator resolution for quarantined objects

The Phase 1 inventory already surfaces the classes needing a human decision
*before* migration. Resolve each and record the decision in the epoch marker;
do not let the migration silently absorb them:

- **`unclassified_artifact` findings** — a file matching no known pattern.
  Identify the owner; either add it to the migration plan or exclude it with a
  recorded reason. Never migrate an unknown file blind.
- **Unmanaged plan-refs** (`plan_refs.unmanaged_details`) — refs with a
  non-`plan-manager` provider, empty `plan_id`, or unknown role. Repair or
  quarantine each before migrating the owning item/initiative; a migration must
  not propagate an unmanaged ref.
- **Ambiguous ownership** (`ownership.ambiguous_run_owners`, and the
  `ambiguous_ownership` finding for the foreign deployment report) — a run id
  mapped to multiple owners, or a foreign artifact under the data root. Decide
  the true owner (or exclude the foreign artifact) before touching run-owner
  indexes.
- **Corrupt / `<unparseable>` documents** (`anomalies` type `corrupt_json`, and
  primary objects with status `<unparseable>`) — hand-repair from backup or
  quarantine to `<migration-scratch>/quarantine/` with a recorded reason. Never
  transform an unparseable document.
- **`invalid_status` anomalies** — normalize to a canonical value per
  `internal/backlogstatus` / initiative / goal enums, or quarantine. (Phase 1
  live run: 0 remaining after enum reconciliation.)
- **`initiative_membership_divergence` / dangling refs** — decide the
  authoritative side (initiative `items` list vs item `initiative` field) and
  stage both sides consistently; do not carry the divergence forward.

---

## 7. Rollback contract

Rollback is always available until the fence is lifted and post-verification
passes.

**Restore procedure:**
1. Stop the scenario (release `events.db`).
2. Restore the data dir from the §1 safety snapshot:
   `data-backup-manager restores restore` against the epoch marker's snapshot id
   (verify first with `data-backup-manager restores verify`).
3. Restore the `state` and `cache` roots from the raw copy in
   `state_cache_backup_path`.
4. Re-run the inventory and confirm `totals.content_hash` **equals**
   `pre_inventory_content_hash` in the epoch marker. Equal hash = clean rollback.
5. Lift the fence; resume writers.

**Conditions that force STOP + rollback (any one):**
- The §1 backup (or state/cache copy, or epoch marker write) failed.
- The pre-migration inventory `content_hash` changed between two fenced runs
  (a writer is still live — the fence leaked).
- Any active/unowned run detected (`orphaned_execution`, open engagement, running
  execution row) during the fence.
- Post-migration reconciliation shows unexplained count/identity drift, a **new**
  anomaly or referential finding, an unexplained content-hash change, or a UI/API
  projection mismatch.
- A checksum mismatch on any restored/verified snapshot.

Only after post-verification passes cleanly and the fence is lifted with the API
healthy is the migration considered committed. Retain all pre-migration snapshots
(`--keep-latest 0`) until Phase 9 closeout.
