# Phase-8 migrator design (slice A preflight — design only; slice B builds it)

> Historical migration design. Its implementation references are retired and
> this document is preserved solely for migration provenance. Current behavior
> is defined by [Target Operating Model](../../concepts/TARGET-OPERATING-MODEL.md).

Concrete design for the one-shot persisted-state migrator of the
"Declarative agent operations for backlog items and initiatives" plan, Phase 8.
Companion documents (authoritative; this design does not restate them):

- [RUNBOOK.md](RUNBOOK.md) — backup / fence / staging / epoch / verification / rollback contract.
- [LEGACY-MAPPING.md](LEGACY-MAPPING.md) — the identity-level mapping (incl. §9 settings equivalence).
- [inventory/inventory-phase1-summary.md](inventory/inventory-phase1-summary.md) — anchor;
  [inventory/inventory-phase8-preflight-summary.md](inventory/inventory-phase8-preflight-summary.md) — this slice's preflight snapshot.
- [../qdrant-namespace-migration-prep.md](../qdrant-namespace-migration-prep.md) — the PREPARED Qdrant orphan cleanup.

Preflight facts this design is built on (verified live 2026-07-15):

- Live roots (resolved via api-core storage, confirmed by `vrooli scenario status swarm-manager` + on-disk):
  - data: `~/.vrooli/data/vrooli/swarm-manager/`
  - state: `~/.vrooli/state/vrooli/swarm-manager/` (present today: `agent-activities.json`, `execution-runs.json`)
  - cache: `~/.vrooli/cache/vrooli/swarm-manager/`
  - config: `scenarios/swarm-manager/config/settings.json` (repo)
- **Finding 80cb2437 verdict (a):** system-default bindings resolve fine at rest;
  the UNRESOLVED-everything symptom was a P7 diagnostics wiring bug (empty
  `OperationVersion` vs version-pinned bindings), fixed in slice A. Consequence
  for this design: **system-default bindings are CODE catalog**
  (`scenarios/swarm-manager/bindings/*.json`) — there is *no* binding seeding
  step in the data migration. Only settings-derived *policy parameters* migrate (F5).
- No live agent runs at preflight; 3 PENDING legacy queue entries + 1 legacy/agentops
  state divergence exist (see §7 quiesce preconditions).

---

## 1. Migrator shape

A **standalone, offline, one-shot CLI** at `scenarios/swarm-manager/tools/statemigrate/`
(sibling of `statemigrate-inventory/`, same constraints: own Go module,
stdlib-only reads/writes, never imports the api module, **deleted in Phase 9**
per storage-steer's one-shot migration policy). It must NOT be an API endpoint
or a boot-time path: the runbook fence (§2.5) prescribes **scenario stopped**
during staging+promotion, and an in-process migrator cannot run against a
stopped scenario. It shares the agentops document *schemas* by reading the JSON
schema files from `api/internal/agentops/schemas/` (path passed via flag), so
validation is against the exact shipped schemas without importing internal code.

Subcommands (each maps to a runbook step and a `migration-status.json` state):

| Subcommand | Runbook step | Effect |
|---|---|---|
| `plan` | §5 pre | Read-only: computes every transformation, prints per-family counts + quarantine list. No writes. |
| `stage` | §3 | Writes complete replacement docs into `<scratch>/staged/**`, quarantine into `<scratch>/quarantine/**`, writes epoch marker, sets status `staged`. |
| `verify-staged` | §5 | Validates every staged doc against its schema + reconciles counts/hashes against the pre-inventory. Read-only on live roots. |
| `promote` | §3.2 cutover | Atomic `rename(2)` swaps staged docs into live roots. Sets status `promoted`. Refuses unless `verify-staged` passed and the fence is confirmed. |
| `rollback` | §7 | Restores from the epoch marker's backup ids (delegates to data-backup-manager + raw state/cache copy) and re-verifies the pre-inventory content hash. |

All writes follow `storage.WriteJSONAtomic` semantics (tmp file + fsync + rename
on the same filesystem). Originals are **never opened for writing** (runbook
golden rule).

### migration-status.json lifecycle

The migrator is the writer of `<dataRoot>/agentops/migration-status.json`; the
Phase-7 operator surface (`opsrunner/migration_status.go`, RPC
`GetMigrationStatus`) is the reader. Exact contract:

- absent file ⇒ `not-started` (reader default; the migrator writes nothing until `stage`).
- `stage` success ⇒ `{kind:"agentops-migration-status", schema_version:"1", state:"staged", epoch:N, staged_count, quarantined_count, started_at, updated_at, report_path}`.
- `promote` success ⇒ `state:"promoted"`, `promoted_count` = docs swapped.
- any stop-condition / `rollback` ⇒ `state:"quarantined"` with `quarantined_count`
  and `report_path` → the epoch report naming what fired.
- The status file itself is promoted LAST (after all family swaps), so a crash
  mid-promote leaves `staged` + a partially-promoted tree that `verify-staged`
  detects by hash (every family swap is individually idempotent, §3–6 below).

---

## 2. Transformation families — overview

| # | Family | Sources (real) | Target shape | Scale (preflight) |
|---|---|---|---|---|
| F1 | Initiative `item-level`/blank mode → member-item strategy | `initiatives/<name>/initiative.json` `.mode` | initiative workflow instance with `strategy` | 35 `item-level` + 32 blank = 67 of 68 (1 `holistic-loop` unchanged) |
| F2 | `plan-manager-plan` → `plan-execution` target id | `mode-targets/plan-manager-plan/{733e1ac3-…,5dbc405c-…}/**` | same docs under `mode-targets/plan-execution/`, `target_kind` strings rewritten | 2 target dirs, 3 rounds, manifests, run-owner index keys |
| F3 | `execution-runs.json` → workflow operation history | `state/execution-runs.json` (today: 4 entries) | `operations[]` records + legacy execution-import snapshots | 4 entries today; grows/shrinks until fence |
| F4 | Workshop/clarify/review/evidence rounds → workflow+execution linkage | per-item `workshop/ review/ evidence/ clarify/` + `initiatives/<n>/review/`, `plan_ref` | per-target `agentops/workflow.json` correlation records | 283 workshop rounds, 3 clarifications, 185 review + 9 evidence artifacts, 1 initiative review round, 144 managed plan_refs |
| F5 | Settings → transition-policy revisions | `config/settings.json` | authored `policy/*.json` parameter revisions (LEGACY-MAPPING §9) | 1 doc |
| F6 | Qdrant orphan collections | Qdrant `swarm-manager-backlog` (483 pts), `swarm-manager-initiatives` (60 pts) | deleted (derived index; PREPARED path) | 2 collections |

Explicitly **not** migrated (mapping already fixed): goals, records, captures
(out of scope), `plan_ref_sweep_manifest` (archival), `events.db` (opaque),
`data/deployment/deployment-report.json` (foreign, excluded), item/initiative
domain truth files (stay authoritative; only correlated).

The following sections give, per family: source → target, schema, idempotency,
quarantine, verification, rollback.

---

## 3. F1 — initiative member-item strategy

- **Source:** `initiatives/<name>/initiative.json` field `.mode`
  (verified live values: `holistic-loop`×1, `item-level`×35, absent×32; e.g.
  `~/.vrooli/data/vrooli/swarm-manager/initiatives/agent-inbox-unified-retrieval/initiative.json` has no `mode`).
- **Target:** a staged `initiatives/<name>/agentops/workflow.json` —
  `agentops-workflow-instance` (schema `workflow-instance.schema.json`) with
  `domain:{kind:"initiative", id:<name>}`, `state` projected from initiative
  status (`active`→`running`? NO — see decision below), and
  `strategy:{name:"parallel-items"}` referencing the default member-item
  strategy (`member-item-strategy.schema.json`: `strategy=parallel-items`,
  `item_operation=execution-run`). `item-level` and blank canonicalize
  identically (LEGACY-MAPPING §3).
- **Decision (for slice B):** initial workflow `state` for migrated initiatives is
  `idle` (or the schema's at-rest state) — the migration must NOT synthesize
  `running` workflows for initiatives with no live operation; state transitions
  belong to the runner post-migration. Only the strategy + identity are seeded.
- **Initiative.json itself is untouched in Phase 8** (`mode` key deletion is
  Phase 9 legacy removal). The workflow instance is authoritative for strategy
  from cutover on.
- **Idempotency:** target file keyed by initiative name; no-op when an
  `agentops/workflow.json` already exists for the initiative with a `strategy`
  (re-stage produces byte-identical doc; digest compare). Existing live
  workflows (none exist today for initiatives — verified: only
  `execute/decision-image-handoff-manifest-schema` and 4 plan-execution
  workflows exist) are never overwritten: an existing instance ⇒ merge is
  FORBIDDEN, the initiative is **quarantined** for operator review.
- **Quarantine:** unknown `.mode` value (anything other than `holistic-loop` /
  `item-level` / blank) → quarantine, never guess. The 3 `dangling_initiative_item`
  members (LEGACY-MAPPING §7) are dropped from the correlated member set and
  recorded in the quarantine report (domain `items` array untouched in Phase 8).
- **Verification:** 67 staged instances, each validating against
  `workflow-instance.schema.json`; count reconciliation 67 = 68 − 1 holistic;
  UI golden projection: initiative detail + operating-mode surfaces render the
  same mode/strategy labels pre/post (the read path falls back item-level→strategy).
- **Rollback:** staged docs are NEW files; rollback = delete the promoted
  `initiatives/<name>/agentops/workflow.json` files (enumerated in the epoch
  report) or full restore per runbook §7.

## 4. F2 — plan-manager-plan → plan-execution

- **Sources (real, verified):**
  - `mode-targets/plan-manager-plan/733e1ac3-f311-4ff0-96b7-65b2a7d70b29/modes/phased-plan-drain/rounds/round-{001,002}.json`
  - `mode-targets/plan-manager-plan/5dbc405c-e04a-476a-9a0d-4454dcacdfac/modes/phased-plan-drain/rounds/round-001.json`
  - per-mode `run-owners.json`/manifests under those dirs, plus any
    `target_kind:"plan-manager-plan"` strings inside round/manifest docs, plus
    entries in `operating-mode-run-owners/run-owners.json` whose
    `target_kind`=`plan-manager-plan` (none today — current entries are
    `backlog-item`; the scan must still cover it).
  - NOTE: `data/plan-executions/<uuid>/agentops/**` (4 dirs) are ALREADY
    new-format (written by the P6 runner) — F2 must NOT touch them; they are the
    proof the new path works.
- **Target:** identical docs under `mode-targets/plan-execution/<same-id>/…`
  with every embedded `plan-manager-plan` target-kind string rewritten to
  `plan-execution`. Round/manifest docs keep their own schema (operating-mode
  engine docs); the only agentops-validated piece is any workflow correlation
  added later. Proto wire number unchanged (LEGACY-MAPPING §1).
- **Idempotency:** presence of `mode-targets/plan-execution/<id>` with matching
  content digest ⇒ no-op; a **residual** `mode-targets/plan-manager-plan/` dir
  after promote is the non-idempotent marker (promote = stage new tree, swap in,
  move old tree to `<scratch>/retired/`, never delete live).
- **Quarantine:** any `mode-targets/plan-ref/**` dir (none observed, P1+P8) →
  quarantine per LEGACY-MAPPING §1. A doc whose in-document `scope_id` disagrees
  with its sanitized folder name resolves by the **in-document id** (RUNBOOK/
  LEGACY-MAPPING §6 hazard); unresolvable disagreement → quarantine.
- **Verification:** file-count + per-file sha256 equality between retired tree
  and new tree modulo the exact string rewrite; grep-proof: zero occurrences of
  `plan-manager-plan` under live `mode-targets/` post-promote; run-owner index
  keys still resolve (every `scope_id` maps to an existing dir via the sanitizer).
- **Rollback:** move the retired tree back; delete `mode-targets/plan-execution/`.

## 5. F3 — execution-runs.json → workflow operation history

- **Source:** `state/execution-runs.json` — array of legacy run entries. Real
  example (preflight): entry `542467c6884d7d09` (chore/qa-scenario-to-desktop-…)
  with `status:"failed"`, `failure_reason`, `queued_at/started_at/finished_at`,
  `mode`, `operation:"generator"`, `started_by`, `pre_exec_baselines`, and —
  because it postdates the P6 cutover — `op_workflow_id` + `op_execution_id`
  already pointing at `plan-executions/f5de76eb-…/agentops/`.
- **Target (two-part, preserving FULL fidelity):**
  1. An `OperationExecutionRecord` appended to the owning target's
     `agentops/workflow.json` `operations[]`: `operation` mapped from the legacy
     `operation`/mode (`generator`→`execution-run`, retry→`execution-retry`,
     fixup→`execution-fixup`, followup→`execution-followup`; unknown → quarantine),
     `execution_id` = existing `op_execution_id` when present else
     `exec-legacy-<legacy execution_id>`, `state`/`outcome` mapped from legacy
     `status` (`failed`→`failed`, `completed`→`completed/accepted`, terminal audit
     preserved), `idempotency_key` = `legacy-import-<legacy execution_id>`.
  2. A full-fidelity import snapshot
     `agentops/executions/exec-legacy-<id>.json` with a NEW document kind
     `agentops-legacy-execution-import` (slice B adds
     `legacy-execution-import.schema.json` beside the other agentops schemas):
     a small typed header (kind, schema_version, operation, execution_id,
     workflow_instance_id, imported_at) + the **verbatim legacy entry** under
     `legacy:` — every timestamp, attempt count, review link, cost metadata,
     failure reason, `pre_exec_baselines` byte-preserved. Records already carry
     `OpWorkflowID`/`OpExecutionID`, so record-side linkage needs no rewrite.
     Rationale: `ExecutionProvenance` (execution-provenance.schema.json) requires
     compiled-mode/prompt-catalog digests that never existed for legacy runs —
     fabricating them would violate the no-fabricated-identity rule, hence a
     dedicated import kind instead of a fake provenance.
- **Entries that already have `op_workflow_id`/`op_execution_id`** (post-cutover
  entries): correlation exists; F3 only reconciles state (see divergence note) and
  does NOT create an import snapshot.
- **Idempotency:** workflow `idempotency_keys` carries `legacy-import-<id>`;
  re-run skips present keys; snapshot write is digest-compared.
- **Quarantine:** entry with unmappable `operation`/`mode`, or whose
  `(backlog_kind, backlog_name)` matches no item on disk (canonicalizing the
  three legacy key spellings per LEGACY-MAPPING §6), or a PENDING/RUNNING entry
  at fence time (see §7 — active runs are a stop condition; pending entries must
  be drained or explicitly held+migrated as `pending` operations by operator
  decision recorded in the epoch marker).
- **Known divergence to reconcile BEFORE staging:** legacy entry
  `542467c6884d7d09` is `failed` while
  `plan-executions/f5de76eb-…/agentops/workflow.json` still shows its operation
  `exec-3968789f881f54ab844ab708` as `state:"running"` — run the Phase-7
  `RunReconciliation` RPC (or the boot sweep with `orphanSnapshotGrace`=15m
  elapsed — it has) and confirm both sides terminal before the fence closes.
- **Verification:** per-entry: one operations[] record + (pre-cutover entries)
  one import snapshot; count reconciliation entries=records; every import
  snapshot validates; UI golden projection: `agent-operations inspect-workflow`
  / `list-execution-history` shows the legacy runs with original timestamps.
- **Rollback:** workflow.json files are staged replacements (originals in the
  epoch backup); import snapshots are new files enumerated in the epoch report.

## 6. F4 — workshop/clarification/review/evidence correlation

- **Sources (bytes stay in place; correlation only — LEGACY-MAPPING §4):**
  - `data/<kind>/<name>/workshop/round-*.json` (283), `workshop/clarifications/*` (3),
    `clarify/*` (1), `review/**` (185), `evidence/**` (9),
    `acceptance-validation.json` (56), `.swarm/**` (136), item docs (229),
    per-item `plan_ref` inside `spec.json` + initiative `plan_ref` (144 managed, 0 unmanaged),
    `initiatives/<name>/review/**` (1 round), pending-advance state (now durable
    scheduler intents — `timers[]` on the workflow), review decisions inside round docs.
- **Target:** the owning target's `agentops/workflow.json` gains
  `operations[]` correlation records ONLY for rounds that carry a run identity
  (run id / execution id in the round doc); rounds without any run identity are
  **history without correlation** — they are NOT invented into operations.
  Evidence requests/review decisions correlate onto the same records
  (`outcome`, review links live in the F3-style import snapshots when a legacy
  run backs them; otherwise the artifact path itself is the evidence, referenced
  from the quarantine-free correlation report, not rewritten).
  `plan_ref` fields: **unchanged** (domain fields, not targets — LEGACY-MAPPING §1).
- **Decision (for slice B):** correlation is *additive metadata on the workflow
  instance*, never a rewrite of round/review/evidence artifacts. If slice B
  finds a round doc whose embedded run id matches an `agent-activities.json`
  run with ambiguous ownership → quarantine (none today; `ambiguous_run_owners=0`).
- **Idempotency:** correlation records keyed `legacy-round-<relpath-digest>`
  in `idempotency_keys`.
- **Verification:** artifact class `content_hash` for
  workshop/review/evidence/clarify classes must be **identical** pre/post
  (bytes untouched — this is the strongest reconciliation signal); workflow
  instances validate; counts: correlated-record count = rounds-with-run-identity
  count computed by `plan`.
- **Rollback:** staged workflow.json replacement only.

## 7. F5 — settings → policy revisions

- **Source:** `scenarios/swarm-manager/config/settings.json` (repo file, 1.4 KB,
  hash unchanged since P1). Mapping table: LEGACY-MAPPING §9 (18 policy-control
  fields → `agentops.PolicyControls` destinations; the rest stay pure settings).
- **Target:** parameterized transition-policy documents in the shipped catalog
  `scenarios/swarm-manager/policy/*.json` (policy revisions are CODE, like
  bindings — the 80cb2437 verdict applies here too). The *file*
  `config/settings.json` is **not rewritten** — unknown/forward-compatible
  fields are preserved by never touching the file (§9 guarantee: no persisted
  field renamed/dropped/rewritten). The migrator's role is only to VERIFY
  equivalence: `ProjectPolicyControls(settings)` == the values expressed in the
  authored policy docs (the §9 equivalence tests already pin
  defaults==projection). If the operator's live settings differ from defaults
  (they do not today — hash equals P1), slice B stages updated policy revisions
  reproducing the projected values.
- **Idempotency:** pure function of settings bytes → policy parameter set;
  digest compare.
- **Quarantine:** none (settings parse is fail-closed; an unparseable settings
  file is a stop condition, not a quarantine).
- **Verification:** the §9 equivalence assertions run as migrator checks;
  golden projection: `SettingsResponse.policy_projection` identical pre/post.
- **Rollback:** git-tracked repo files; revert the staged policy docs.

## 8. F6 — Qdrant orphan cleanup (PREPARED path)

Execute exactly `docs/operations/qdrant-namespace-migration-prep.md`:
delete legacy hyphen collections `swarm-manager-backlog` (483 pts) and
`swarm-manager-initiatives` (60 pts) after (1) confirming `storage.Collection`
adoption unchanged in `internal/aisearch/env.go`, (2) the §1 safety backup,
(3) per-collection Qdrant snapshots. Qdrant is a derived index — the reconciler
rebuilds underscore collections from the stores of record; rollback = restore
the collection snapshot (or simply let the reconciler rebuild). Idempotency:
delete of an absent collection is a no-op. Verification: collection list shows
only underscore collections; live search endpoints healthy; reconciler
`RunOnce` clean. **Ordering:** F6 runs LAST, after post-verification of F1–F5,
because it is the only irreversible-ish step (snapshot-guarded) and is fully
independent.

---

## 9. Quiesce fence — concrete writer enumeration (from running code)

Every background writer in the CURRENT binaries (`api/main.go` + route files),
and how to stop it:

| Writer | Where started | What it mutates | How to stop |
|---|---|---|---|
| Execution poller | `main.go:903` `executionHandler.StartBackgroundWorker` (`internal/execution/handler.go:48`) | `execution-runs.json`, item `spec.json` statuses, `circuit-breaker.json`, `engagement-owners.json` | **no runtime flag** — scenario stop |
| Review worker + boot orphan sweep | `main.go:908` + `routes_execution.go:300–315` (`review.NewSweeper`) | `<item>/review/**`, item statuses | no flag — scenario stop |
| Initiative-review worker | `main.go:913` | `initiatives/<n>/review/**` | no flag — scenario stop |
| Ops refresh driver | `main.go:926` `RefreshDriver.Run` (5 s tick) | active mode-target rounds, `agentops/**` | no flag — scenario stop |
| Ops scheduler (durable intents, auto-advance) | `main.go:930` `Scheduler.Run` (5 s tick) | fires `workshop-round`/`workshop-finalize` Invokes → `workshop/**`, `agentops/**`, `timers[]` | no flag — scenario stop |
| Boot-time orphan-snapshot reconcile | `routes_backlog_ops_runner.go:178` (`ReconcileOrphanSnapshots`, 15 m grace) | `agentops/executions/**`, workflow records | boot-only — runs on next start, so run it deliberately BEFORE the fence (resolves the F3 divergence), then stop |
| Auto-filer sweeper | `main.go:936` | backlog items (suggest/create) | currently `enabled=false` in settings (boot log confirms) — still stop the scenario |
| Workshop auto-advance (deferred intents) | via ops scheduler (above) + `backlog` save paths | `workshop/**` | scenario stop |
| aisearch boot reconcile + SyncLoop | `main.go:957` `startAISearchBackground` | Qdrant only (no fs) | `AI_SEARCH_SYNC_DISABLED` kill-switch exists, but scenario stop covers it |
| Event log | every API mutation | `events.db{,-wal,-shm}` | scenario stop (also required to checkpoint WAL + release the lock, RUNBOOK §1/§2.5) |
| API mutation handlers | all routes | everything | scenario stop |

**Conclusion (matches RUNBOOK §2.5's "preferred"):** there is no per-writer
runtime fence — the quiesce mechanism is `vrooli scenario stop swarm-manager`,
then the double-inventory hash-stability check (§2 fence verification). The
migrator refuses to `stage`/`promote` while the API port answers.

**Fence preconditions surfaced by this preflight (must be resolved first):**

1. **3 PENDING legacy queue entries** in `execution-runs.json`
   (`54af4aefa1199d31` chore/vrooli-emulator-documentation,
   `5ff82ba17abd1670` execute/audio-tools-greenfield-scenario,
   `08f09a76f24b0dc3` execute/brand-manager-scenario-picker; items `queued`).
   Operator decision before the fence: let the drain scheduler run them to
   terminal, cancel them (item back to `backlog`), or migrate them as `pending`
   operations (F3 quarantine branch). Active/running rows at fence time are a
   hard stop (RUNBOOK §7).
2. **Legacy/agentops divergence** on `542467c6884d7d09` ↔
   `wf-plan-execution-f5de76eb…/exec-3968789f…` (legacy failed, agentops still
   running): run `RunReconciliation` and confirm terminal (§5/F3).
3. **Stale run-owner index entries**: `operating-mode-run-owners/run-owners.json`
   holds owners `10a85226-…` and `f09e2a4d-…` whose agent runs are `complete`
   in `agent-activities.json`. Slice B rule: at stage time, an owner entry whose
   run is terminal is dropped from the staged index (recorded in the epoch
   report), because carrying a dead owner forward would block post-migration
   exclusivity.
4. **8 `unclassified_artifact` findings** = the NEW-format
   `plan-executions/*/agentops/**` docs (tool predates the agentops classes).
   Resolution per RUNBOOK §6: classify, don't quarantine — slice B teaches
   `statemigrate-inventory` an `agentops_doc` class so pre/post reconciliation
   sees them as first-class (they must survive migration byte-identical except
   where F3 reconciliation touches f5de76eb).

---

## 10. Backup readiness (verified today, read-only)

- `data-backup-manager` scenario starts and answers (`targets list` verified):
  target `swarm-manager/domain-data` (id `daaeb35c-…`, kind=filesystem,
  locator `~/.vrooli/data/vrooli/swarm-manager`) is **already registered** —
  registered 2026-05-27, matches the live data root exactly.
- Destination `elements-local` (kopia, AES256) at
  `/media/matthalloran8/Elements/vrooli-backups` — drive mounted, 1.7 TB free.
- The reserved **safety destination does not exist yet** — `safety
  ensure-destination` (idempotent) is step 1a of the runbook and must run at
  migration time; `safety register-targets --scenario swarm-manager` will
  re-derive targets (idempotent over the existing registration).
- **Gap the runbook already covers:** only the DATA root is a registered
  target. `state/` and `cache/` roots (and the repo `config/settings.json`)
  are NOT covered by data-backup-manager — the raw `cp -a` copies in RUNBOOK §1
  are load-bearing, not belt-and-suspenders.

---

## 11. Verification & reconciliation (cross-family)

Beyond per-family checks (above), the migrator's `verify-staged`/post-promote
gate is RUNBOOK §5 verbatim, with these concrete anchors:

- Pre-fence inventory ×2 (30 s apart) must byte-match (fence proof).
- Post-promote inventory: classes NOT named by a family
  (records, goals, captures, item artifacts under F4, events.db) must have
  **identical `content_hash`** to the pre-fence run.
- Identity sets per primary class unchanged (F1–F5 add files, never
  rename/remove identities; F2's rename is a *path segment*, identities are the
  in-document ids).
- Referential findings post ⊆ pre; anomalies = 0; `plan_refs.unmanaged` = 0.
- Golden projections on a staging copy first (RUNBOOK §5): backlog list,
  initiative detail (mode/strategy label), goals, operating-mode surfaces,
  `agent-operations inspect-workflow/bindings/list-execution-history`.

## 12. Rollback (cross-family)

RUNBOOK §7 verbatim; family-specific notes in §3–8. The epoch report must
enumerate: every NEW file promoted (F1 instances, F3 import snapshots, F4
workflow replacements), every REPLACED file (workflow.json, run-owner indexes,
`execution-runs.json` if rewritten), the RETIRED tree (F2), and the two Qdrant
snapshot ids (F6) — so rollback is a mechanical inverse even without the full
data-dir restore.

## 13. Open decisions for slice B (explicit, so they don't dissolve)

1. F3 pending-entry disposition (drain vs cancel vs migrate-as-pending) —
   operator call at fence time, recorded in the epoch marker.
2. F1 at-rest workflow `state` value for migrated initiatives (proposed `idle`;
   confirm against `AllWorkflowStates` and the runner's expectations).
3. The new `legacy-execution-import` schema (F3) — name/kind pinned here;
   shape finalized in slice B next to the other agentops schemas.
4. Whether F4 correlation lands in Phase 8 at all or is deferred: it is
   additive and non-destructive; if schedule pressure hits, F4 can promote
   after F1–F3/F5 with its own epoch. (F2, F3, F5 are the load-bearing
   cutover families; F6 is independent cleanup.)
5. Teach `statemigrate-inventory` the `agentops_doc` class (§9.4) BEFORE the
   pre-fence inventory, so the reconciliation oracle is clean.
