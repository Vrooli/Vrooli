# Phase 8 slice C — live migration evidence (PROMOTED)

Executed 2026-07-15 against the live swarm-manager persisted state, per
[RUNBOOK.md](RUNBOOK.md) and [MIGRATOR-DESIGN.md](MIGRATOR-DESIGN.md).
Final state: **promoted** (epoch 1, 125/125 documents, 0 quarantined).

## Backup identity (taken BEFORE any mutation; verified restorable)

| What | Value |
|---|---|
| Destination | `baseline-safety` (`979cbbc2-2994-4a17-8169-0ecb8265b645`), kopia repo `/home/matthalloran8/.vrooli/baseline-safety/repositories/baseline-safety.kopia` |
| Pre-mutation backup run | `967990ce-3542-4d94-913c-909744bcce39` COMPLETED 2026-07-15T19:33:10Z |
| Pre-mutation snapshots | `429fa497bb5cf5d837ad6eadbb39ce1c` (target `daaeb35c` domain-data), `bc7dfe3b4582dd83b9ac4b560bbb9b93` (target `e48373c2` data) — both over `~/.vrooli/data/vrooli/swarm-manager` |
| Restore-verify | `RESTORE_STATUS_VERIFIED` checksum `b2654265…`; full restore into scratch = 3376 files, sha256-identical to live on ALL non-events.db files |
| Fenced backup run | `c7bc9d57-6d82-4925-b0a1-2f17aa2500d9` COMPLETED (API stopped, WAL checkpointed); snapshots `5a46a259fa10b54fe4e1cae7380e63cd`, `5512e42b28ead75f67bc0e5da7314288`; restore-verify VERIFIED checksum `5d220dbf…` |
| Raw copies (state/cache/config — dbm covers only the data root) | `/home/matthalloran8/.vrooli/p8c-migration-backup/{state,cache,settings.json}` (pre-mutation) and `…/fenced/{state,cache,settings.json}` (post-quarantine-resolution, fenced) |
| Restore command | `data-backup-manager restores restore --target daaeb35c-38a8-486b-9926-eeff73b03ea5 --destination 979cbbc2-2994-4a17-8169-0ecb8265b645 --snapshot <id> --location <dest>` |

Retain all snapshots until Phase 9 closeout (`--keep-latest 0` honored).

## Quarantine dispositions (all via normal application surfaces)

1. **3 PENDING legacy queue entries** — `swarm-manager execution cancel`:
   `54af4aefa1199d31` (chore/vrooli-emulator-documentation), `5ff82ba17abd1670`
   (execute/audio-tools-greenfield-scenario), `08f09a76f24b0dc3`
   (execute/brand-manager-scenario-picker) → `canceled`, items restored to
   `backlog` (re-queueable). Reason: pre-P6-cutover entries; this host cannot
   spawn execution-grade runs, draining impossible.
2. **`542467c6884d7d09` failed vs agentops op running divergence** — root
   cause: `ReconcileStrandedRecords` marked the legacy record failed without
   reaping its correlated operation. Fixed in code
   (`api/internal/execution/reconcile.go`: missed-reap second pass, tested);
   applied by the boot sweep itself on restart
   (`op_reaps_attempted=[542467c6884d7d09]`) → op `exec-3968789f…` now
   canceled/canceled.
3. **3 dangling initiative members** — verified in git that all three missing
   items were user-committed deletions (fix/dtv-validation-api @ `ea0e81c661`
   2026-03-26; chore/swarm-manager-remove-legacy-tabbed-surfaces and
   execute/swarm-manager-mobile-graph-interaction @ `f3dfe2dfd8` 2026-04-10);
   removed the stale memberships via `initiatives remove-items`.
4. **Active clarification** research/emulator-extraction-and-service-plan
   `round-001-item-d1` (74063326) — asked AND answered 2026-04-04, item back
   in `backlog`, run terminal: stale orphaned thread, closed via
   `backlog clarify-action … --action got_it` → `resolved`.
5. **Tool false-positive refined (no data touched)** — F4 flagged
   execute/qa-git-control-tower-code-quality-20260408 review rounds 002+003
   for sharing run `9e6b2a3a…`: same owning item/workflow, sequential
   artifacts of one review run. `statemigrate` planF4 now quarantines only
   claims spanning >1 owning workflow (cross-owner = the design's ambiguity
   concept); covered by `TestSameOwnerMultiRoundRunClaimIsNotAmbiguous`.

NOT touched (pre-existing, visible debt, present since the P1 baseline and not
quarantined by the migrator): 2 `dangling_dependency` findings
(fix/dtv-cli-validate-and-report, fix/dtv-report-generation → deleted
fix/dtv-validation-api) and the foreign `data/deployment/deployment-report.json`.

## Inventory hashes (UPDATED tool with `agentops_doc` class, config included)

| Stage | files | primaries | anomalies | findings | content hash |
|---|---:|---:|---:|---:|---|
| Post-resolution (live, pre-fence) | 3390 | 2313 | 0 | 3 | `sha256:7069391868ab630eeefc7e2cc06b7b0071e1dd7d474c763cd50f2a86f42b609f` |
| Fenced run 1 | 3388 | 2313 | 0 | 3 | `sha256:d1bd3f4352446306f44ae53b8e92b74178540fdcf6391864b193fe406b9e4c3f` |
| Fenced run 2 (31 s later) | 3388 | 2313 | 0 | 3 | **identical** — fence stable |
| Post-promote | 3510 | 2313 | 0 | 3 | `sha256:ad7145f51b29dac7b423098d852a231103a1ea2815807dc59f232c833928e726` |

Pre-fence deltas explained: −2 files = events.db WAL+SHM checkpointed on stop;
+1 record = concurrent fleet learning-loop churn (`rec-eb131fa706adb376`);
class reshuffles from the tool's new `agentops_doc` classification.

Post-promote delta EXACT: −3 retired legacy rounds +124 new documents
+1 migration-status doc = +122. Only three classes changed
(`agentops_doc` 10→132, `om_round` hash-only — 3 rounds relocated
plan-manager-plan→plan-execution, `om_global_run_owners` replaced — 2 stale
terminal owners dropped). Every other class byte-identical. Findings post ⊆
pre; anomalies 0; grep-proof 0 quoted `plan-manager-plan` tokens under live
`mode-targets/`.

## Staging / promote

- Staging dir: `/home/matthalloran8/.vrooli/p8c-migration-staging` (staged tree
  + `backup/` + `retired/` + `reports/{stage,verify,promote}-report.json`).
- Staged content hash (pinned by stage, verify, promote):
  `sha256:76073c4399f0b2b208809b3895bda0ca0a72d0529c121ae17c54d46e45cb9454`.
- verify-staged: ALL checks green (source-pin 154, plan-hash, quarantine 0,
  per-family counts, 125 byte-match, schemas, F2 rewrite-equality, F4
  artifact-bytes-untouched).
- Families: F1 67 (+1 no-op holistic), F2 4 (+2 retired trees), F3 3 imports
  (+1 no-op post-cutover entry), F4 78, F5 1 (policy-projection doc;
  settings.json untouched), F6 external.
- `agentops/migration-status.json`: `promoted`, epoch 1, staged 125,
  promoted 125, quarantined 0.

## Read-back smoke (scenario restarted, healthy)

- Boot: `backlog-ops runner constructed modes=16`, no errors, stats replay ok.
- (a) Migrated item-level initiative dtv-meta-optimization-readiness →
  `wf-initiative-…`, state `open`, `strategy: parallel-items` visible.
- (b) F3 imports render honestly: history shows `execution-run [legacy import]`
  with real outcome/timestamps, `(imported pre-cutover record; no provenance)`
  — no fabricated provenance (hollow-snapshot fix below).
- (c) F2 targets resolvable: rounds live under
  `mode-targets/plan-execution/{733e1ac3,5dbc405c}` with `scope_kind`
  rewritten; prose byte-preserved.
- (d) UI HTTP 200; migration banner renders only for staged/quarantined —
  promoted is invisible by design (code-verified).
- (e) Settings GET byte-identical pre/post; `policy_projection` present.
- (f) `agent-operations validate` on migrated initiative: declared /
  compatible / resolved all true; bindings resolve sanely.
- **Golden projections: 8/8 byte-identical pre vs post** (backlog list,
  initiative dtv + blank-mode + list, goals, wf+history f5de76eb,
  wf decision-image). Captures in session scratchpad `p8c/golden-{pre,post}`.

## No-op + rollback rehearsal proof

- **No-op:** post-promote `statemigrate plan` against live (reopened, API up →
  read-only) = writes 0, quarantines 0, no-op 152 (F1 68, F2 1, F3 4, F4 78,
  F5 1).
- **Rollback rehearsal (on a COPY):** fenced snapshot `5a46a259…` restored to
  scratch + fenced state copy; stage→verify→promote on the copy reproduced the
  **exact same staged content hash** `76073c43…` (determinism proof); rollback
  restored the copy byte-identical (data + state hash manifests equal) except
  `agentops/migration-status.json` parked `state=quarantined` — the documented
  rollback contract, not drift.

## F6 Qdrant orphan cleanup (prepared path, run LAST)

- Preconditions honored: `storage.Collection` adoption unchanged in
  `internal/aisearch/env.go`; safety backups above; per-collection snapshots
  taken (`swarm-manager-backlog-…-2026-07-15-20-18-07.snapshot` 20.8 MB,
  `swarm-manager-initiatives-…-2026-07-15-20-18-08.snapshot` 3.2 MB, resident
  in Qdrant's snapshot store).
- Dropped `swarm-manager-backlog` (483 pts) + `swarm-manager-initiatives`
  (60 pts). Collection list now shows only underscore collections
  (600/68/1473 points, untouched). Reconcile RunOnce clean
  (unchanged 600/68, legacy 0/0); live semantic search healthy (20 results).

## Code changes landed with this slice (built + tested)

1. `api/internal/execution/reconcile.go` — missed-reap recovery pass
   (+ `migrations_setup.go` log line, + tests).
2. Hollow-snapshot readability: `api/internal/opsrunner/execution_store.go`
   decodes `agentops-legacy-execution-import` docs (real fields only,
   `LegacyImport` marker, `Reproduce` refuses with
   `ErrLegacyImportNotReproducible`); `agentopsdiag/service_projection.go`
   maps labeled rows; additive proto fields
   `AgentOpsExecutionSummary.legacy_import` /
   `AgentOpsOperationProjection.legacy_import`; CLI + UI render
   "legacy import" labels (+ tests).
3. `tools/statemigrate/plan.go` — F4 ambiguity rule scoped to cross-owner
   claims (+ fixture tests).

## Gates (verbatim)

- api: `go build ./... && go test ./...` → exit 0 (includes the
  `statemigrate_golden_test.go` which rebuilds and revalidates the tool).
- cli: `go build ./...` → OK (full `go test ./...` also green).
- tool: `go test ./...` → `ok swarm-manager-statemigrate`,
  `ok swarm-manager-statemigrate-inventory`.
- `cli-health validate scenario swarm-manager` →
  `status=passed errors=0 warnings=0 infos=0`.
- ui: `pnpm type-check` → clean.

## Deferred (explicit decisions)

- `tools/statemigrate` + `tools/statemigrate-inventory` are NOT deleted here:
  the opsrunner golden test builds the tool; both go together in Phase 9's
  single deletion pass.
- Phase transition is NOT performed by this slice; the main session runs
  phase validation.
