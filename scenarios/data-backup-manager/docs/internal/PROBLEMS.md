# Problems — Data Backup Manager

Persistent register of known issues, tech debt, and deferred work
specific to **this** scenario. Future agents read this file to avoid
re-discovering the same constraint.

This file ships empty in newly generated scenarios. Append entries as
they appear.

## What belongs here

- **Known bugs** that are real but not yet worth fixing
- **Tech debt** — workarounds that need a real fix later
- **Deferred work** — features descoped from a phase, with the reason
- **Architecture drift** — code/docs/tests that no longer line up with
  the intended capability map or boundary model
- **Constraints discovered the hard way** that aren't visible from
  the code (e.g., "this resource needs warm-up before the first call;
  see commit X")

## What does NOT belong here

- **Generic template issues** — those go in
  [`../guides/troubleshooting.md`](../guides/troubleshooting.md)

- **Open feature requests** — track those in PRD operational targets
- **Code comments** — if the constraint is local to one file, a
  comment there is more discoverable
- **Test failures** — fix them, don't document them

## Entry template

Use this shape so entries are scannable. Append newest at the bottom.

```markdown
### YYYY-MM-DD — short title

**Symptom:** What goes wrong, observable from outside the system.

**Root cause:** What actually causes it (or "unknown" if not yet diagnosed).

**Workaround:** What to do today to keep moving.

**Real fix:** What needs to happen for this entry to be deleted.

**Owner:** Who should drive the fix (or "unassigned").

**Refs:** Code paths, related issues, prior commits.
```

## Entries

### 2026-11-10 — Elements: 13 unreadable leftovers, backups stopped since 2026-08-19 (now asked about)

**Update:** the unreadable set is 13 `s/*/*.f.tmp.*` session-marker upload leftovers
(only the first is named by kopia's error). Backups last succeeded on 2026-08-19.
DBM now diagnoses this, raises one decision (skip as recorded exceptions, offline
repair, or fresh repository by an agent) and sends daily stalled-backup notices.
The leftovers themselves are untouched. Revoking an exception has no CLI yet:
`FileExceptionStore.Revoke` is the only path (a record under
`state/backup-exceptions`).

### 2026-11-04 — Elements Kopia repository contains an unreadable temp object

**Symptom:** Destination usage and Kopia blob statistics fail with `EIO` while
reading the exact unpaired content temp
`s/f43/64b7506cef44e4fbb3fc66ba7f2d9-sf40366ce03934d79144.f.tmp.8c9d63ca28444838`.
Snapshot enumeration succeeds after the separately authorized q-manifest
repairs, but the Elements backup plan remains unready and its 14 target
postures remain failed/overdue.

**Root cause:** The host currently returns an I/O error for this object on the
USB-attached NTFS destination. Evidence does not identify whether the fault is
in the device, USB transport, filesystem, or this object. Same-volume rename
also returned `EIO`; moving or deleting it is not a safe diagnostic.

**Current evidence:** The full 14-target source enumeration now succeeds after
repairing traversal on one owner-owned mode-0600 receipt directory. DBM's
current destination readiness check passes the complete estimated source bytes
and retention-three capacity check. The exact repository usage call still
fails with `EIO`. Mounted filesystem dirty/needs-check state is unavailable
through the current `ntfs-3g` mount, so overall readiness remains `UNKNOWN`.
The authorized identity-bound SMART query reports overall status passed,
35°C, and zero reallocated sectors, reallocation events and current-pending
sectors in the complete output prefix. The active broker truncates its raw JSON
at 8 KiB, so the remaining attributes and error/self-test logs are not
qualified; a passing summary does not rule out USB transport or filesystem
damage. The owner-side serializer now emits a bounded valid SMART summary, but
the setup-managed broker still needs the authenticated update to load it.

**Workaround:** Preserve the temp object, keep `elements-daily-runtime`
inactive, and do not capture, restore, prune, or mutate the repository until
the error is diagnosed. Obtain the already-approved read-only SMART evidence
through the identity-bound control-plane command after the setup-managed root
broker is updated, or use a read-only check from another authorized computer.

**Real fix:** Diagnose and resolve the drive/transport/filesystem fault at its
owning host layer, then re-run repository usage, filesystem readiness, the
overdue target captures, and an independent restore. Keep the plan inactive
until those checks pass.

**Owner:** Data Backup Manager recovery supervisor with host storage owner.

**Refs:** `api/handlers/destinations/recovery_monitor.go`,
`api/handlers/destinations/recovery_decision.go`,
`api/handlers/destinations/connect_handler.go`; recovery checkpoint
`~/.vrooli/research/cohorts/dbm-drive-recovery-20261103/CHECKPOINT.md`;
operator actions `~/.vrooli/research/ATTENTION.md`.

### 2026-11-03 — Recovery state was missing from the storage declaration (source fixed)

**Symptom:** The comprehensive Test Genie storage phase reported recovery
journal bytes under the canonical DBM state root as uncovered by any declared
storage entry.

**Root cause:** DBM's destination module writes crash-resumable journals and
Notification Hub decision records below `filePaths.StateDir`, but
`.vrooli/service.json` declared only `state/retention` there.

**Resolution:** Declared `state/volume-recovery` and
`state/volume-recovery-decisions` as durable owned directories with measured
size budgets. After its five-minute inventory TTL, Storage Manager reports both
entries and all 471,319,544 observed DBM bytes accounted for. The previous Test
Genie phase result remains historical until a new suite produces fresh evidence.

**Refs:** `api/main.go`, `api/handlers/destinations/module.go`,
`.vrooli/service.json`; Test Genie storage artifact from run
`20261103-213641-4ae1fc23`.

### 2026-06-03 — Discovered-vs-registered coverage gap closed (resolved)

**Symptom:** Discovery suggested durable state worth protecting (Vrooli runtime
+ coding-agent state), but only *registered* targets are eligible for plans and
runs. On a fresh install an operator could build a plan that looked protective
while it backed up a single self-registered target — accepting a suggestion was
a per-item manual `targets register`, with no bulk path and no plan-time guard.

**Root cause:** Two separate truths (registered catalog vs derived suggestions)
with no surface that reconciled them or enforced default coverage at plan
creation.

**Resolution:** Added the coverage domain (`CoverageService` —
`GetCoverageReport` + `AcceptDefaultTargets`) composing discovery + targets +
plans + runs + restores. `coverage accept-defaults` bulk-registers non-sensitive
discovered durable targets; sensitive credential/token targets require the
verified Notification Hub review; `plans create/update` are guarded
(`failed_precondition`) while non-sensitive recommendations remain unregistered
unless `allow_incomplete_coverage` is set. Surfaced in CLI (`coverage` group),
API (CoverageService + plan guard field) and UI (coverage banner on Overview /
Targets / Plans, plus an explicit "proceed with incomplete coverage" path).
Coverage reads no file contents and persists nothing. See FLOWS.md "Default
coverage acceptance".

### 2026-06-02 — Stale local kopia metadata is not auto-reaped

**Symptom:** Old e2e/canary runs can leave local resource-kopia config
directories under `~/.config/vrooli/resources/kopia/repos` even when
`resource-kopia repo list --json` returns `[]`. There is no DBM command that
inspects or reaps this stale local metadata.

**Root cause:** `destinations delete --delete-repository` removes local
metadata + vault refs for the destination it is given, but a test that creates a
repository and is killed before its cleanup runs (or a registry that drifts from
the on-disk config) can orphan a local config dir with no catalog row pointing at
it.

**Workaround:** The disposable proof (`scripts/prove-backup-restore.sh`) cleans
up everything it creates via `--delete-repository true` in its EXIT trap. For
pre-existing stale dirs, inspect manually before removing — never auto-delete
vault secrets for a repo that might still be registered.

**Real fix:** An inspection-first maintenance path (report orphaned local kopia
config dirs with no catalog row, then a guarded, opt-in reap). Deliberately
deferred: cleanup that deletes recoverability secrets must be guarded, not
automatic. See the production-ready destination-UX plan §7.

### 2026-06-02 — Filesystem destinations are now self-describing bundles (resolved)

**Symptom (was):** A filesystem destination folder was a bare encrypted kopia
repository — opaque in a file browser, with no README, recovery steps, or
non-secret manifest, and delete wording implied backend bytes were removed when
only local metadata was.

**Resolution:** Filesystem destinations are now self-describing bundles: the
operator-facing bundle root (`location`) holds `README.txt`, `RECOVERY.txt`, and
`vrooli-backup-destination.json`, with the vanilla kopia repository nested under
`repositories/<slug>.kopia` (`repository_location`). Snapshots carry
self-identifying kopia metadata (description/override-source/tags). Delete
wording across API/CLI/UI now states precisely what is and is not removed
(encrypted backend bytes always remain). The repository stays a vanilla kopia
repo restorable with plain kopia + passphrase.

### 2026-05-26 — Redis source backups are best-effort, not point-in-time

**Symptom:** A Redis source backup may not represent a single
transactionally-consistent instant. Keys written or deleted while the
snapshot is in progress can be partially captured.

**Root cause:** The Redis source kind captures by namespace prefix using
`SCAN` + `DUMP` over the live keyspace rather than a frozen snapshot.
`SCAN` is iterative and non-atomic by design, so the resulting artifact
is a near-consistent view, not a true point-in-time one.

**Workaround:** Accept the best-effort semantics for cache/ephemeral
state (the common Redis use). Where stronger consistency is needed,
prefer a quiesce hook (PRD OT-P1-001) around the source, or back up the
durable store that Redis fronts instead.

**Real fix:** Adopt a transactional snapshot path if/when one is
available for the target Redis deployment (e.g., an RDB/replica snapshot
the source CLI can hand off atomically). Until then this is an accepted
design limitation, not a bug.

**Owner:** unassigned.

**Refs:** `DECISIONS.md` (six source kinds; Redis best-effort);
`PRD.md` source-kind notes and OT-P1-001 (quiesce hooks).

### 2026-05-26 — Run/restore flows modeled at Level 2, not the Level-5 flow-verifier model

**Symptom:** `FLOWS.md` targets a Level-5 checked formal model (flow.json +
generated Quint + replay) for the backup-run and verified-restore flows. The
implementation ships Level-2 only: pure `Transition`/`CheckInvariants` +
matrix/trace tests in `api/internal/runs/lifecycle.go` and
`api/internal/restores/lifecycle.go`.

**Root cause:** Deliberate scope decision in the API+CLI implementation plan —
Level 2 captures the load-bearing invariants (partial-failure isolation, the
verify gate, no-eviction) executably; the Level-5 machinery is additive.

**Workaround:** The Level-2 transition functions and their tests are the
source of truth today. `make temporal-models` passes an absolute scenario root
to `flow-verifier`; using `--root .` from this scenario directory can be
misinterpreted by the external flow-verifier service as its own scenario root.

**Real fix:** Scaffold `flow-verifier flows new api/internal/runs --flow-id
backup-run --lang go` (and the restore flow), port the transition tables, and
wire replay — promoting both flows to Level 5.

**Owner:** unassigned. **Refs:** `FLOWS.md` (Deferred/Unmodeled Flows);
`internal/runs/lifecycle.go`; `internal/restores/lifecycle.go`.

### 2026-05-26 — Source resource-CLI surfaces are assumed, not yet reconciled

**Symptom:** The postgres/redis/qdrant/object source capturers shell out to
`resource-postgres|redis|qdrant|minio` with an assumed subcommand/flag surface
(e.g. `resource-postgres dump --database <n> --output <f>`). The real resource
CLIs may differ.

**Root cause:** The capturers were built to the design ideal before each source
resource CLI's exact surface was verified. Their unit tests assert the argv we
build; the round-trip integration tests are gated behind `DBM_SOURCE_INTEGRATION=1`.

**Workaround:** Filesystem + SQLite kinds need no resource and round-trip in
default tests. The other four are integration-gated and inert until enabled.

**Real fix:** Run each gated integration test against the real resource, then
reconcile the argv in `api/internal/sources/{postgres,redis,qdrant,object}.go`
with the actual CLI (fix-substrate the resource CLI if a needed verb is
missing). Update the assumed-surface table in the source files.

**Owner:** unassigned. **Refs:** `internal/sources/*.go`; `INTEGRATIONS.md`.

### 2026-05-26 — vault secret-read CLI is a stub

**Symptom:** `INTEGRATIONS.md` assumes a `resource-vault secret get` CLI,
but that surface is a stub.

**Root cause:** This substrate gap surfaced during implementation. Source
credentials are sidestepped today by having each source resource CLI
self-source its own credentials, so no direct vault read is needed.

**Workaround:** Source capture works without a raw vault read.

**Real fix:** Only add `resource-vault secret get` if a concrete source
genuinely needs a raw secret the manager must pass.

**Owner:** unassigned. **Refs:** `internal/engine/kopia.go`;
`INTEGRATIONS.md`; the kopia resource.

### 2026-05-26 — ListTargetStatus owner filter not wired

**Symptom:** `RunsService.ListTargetStatus` accepts an `owner` field but ignores
it — runs keys purely on target id and has no owner field in run history.

**Root cause:** The runs domain does not own targets; resolving owners would
couple it to the targets domain.

**Workaround:** The default status rollup now resolves current target ids at
the composition edge and excludes orphaned/deleted historical run outcomes.
Explicit target-id filtering still returns historical records.

**Real fix:** Resolve owner→target-ids via a targets adapter at the handler
edge and pass the id set to the repository (which already filters by id).

**Owner:** unassigned. **Refs:** `handlers/runs/connect_handler.go::ListTargetStatus`;
`internal/runs/repository.go::TargetStatuses`.

### 2026-06-03 — Filesystem/SQLite capturer single-file + symlink handling (resolved)

**Symptom:** The first real multi-target backup (16 default-coverage targets to
the Elements drive) returned `PARTIAL_FAILED`: every single-file target
(`*.toml`, `*.jsonl`, `secrets.json`) failed `capture: … copyFile create
".../fs": is a directory`; `~/.vrooli/state` failed `copy_file_range … is a
directory`; and the two SQLite targets passed capture but failed `restores
verify` with `snapshot restore … is a directory`.

**Root cause:** `sources/fs.go` always treated the staged artifact root `fs/` as
a directory: a single-file locator's only walk entry has `rel == "."`, so the
copy clobbered the directory. The walk also treated symlinks as regular files
(`d.IsDir()` is false for a symlink), so `os.Open` followed `~/.vrooli/state`'s
links into directories (`rules → ~/.codex/rules`) — and, worse, into the
deliberately-excluded `~/.codex/auth.json` credential. `sources/sqlite.go`
produced a single *file* artifact, but the engine snapshots the artifact and
later restores it into a fresh *directory*, which kopia cannot do for a
single-file snapshot root.

**Resolution:** fs.go stages single files under `fs/<basename>` and preserves
symlinks as symlinks (never dereferenced); `restores/service.go::checksumDir`
hashes link targets instead of opening them; sqlite.go stages into a directory
(`sqlite/snapshot.db`). Covered by new `sources` tests (single-file, symlink,
sqlite directory shape). After the fix a fresh run is `COMPLETED` with all 16
targets `RESTORE_STATUS_VERIFIED`.

**Owner:** resolved. **Refs:** `api/internal/sources/fs.go`,
`api/internal/sources/sqlite.go`, `api/internal/restores/service.go::checksumDir`.

### 2026-06-03 — Synchronous run execution is interruption-fragile (runs: resolved)

**Symptom:** A backup that ran longer than the transport timeouts (default
Connect client timeout; api-core 30 s `WriteTimeout`) had its connection severed
mid-operation. Because the server execed kopia with the *request* context, the
disconnect killed the kopia subprocess; the run was then stranded in
`RUN_STATUS_PENDING` forever (no terminal status, no reconciliation). The
orphaned PENDING rows for plan `elements-daily-runtime` were this.

**Root cause:** `runs.TriggerRun` executed the whole operation synchronously
inside the request handler and threaded `req.Context()` down to
`exec.CommandContext`, with no async execution, no incremental status
persistence, and no startup sweep.

**Resolution (runs — shipped 2026-06-03, async backbone):** `runs.TriggerRun`
now validates the plan, creates the run row, enqueues it onto a background
`runs.Executor` bound to a server-lifetime (non-request) context, and returns the
non-terminal run immediately. The worker persists `capturing`→`snapshotting`→
terminal transitions and writes each `TargetOutcome` as it lands (heartbeat via
`runs.updated_at`). On boot, `Service.Reconcile` closes any run left non-terminal
as `failed` with a reconciliation reason (fail-not-resume in v1). The `runs
trigger` CLI interim workaround was removed (it uses the standard client now).
The historical server `WriteTimeout: 6h` workaround was retained while restores
were synchronous. Refs:
`api/internal/runs/{service,executor,repository,sqlite,migrate}.go`,
`api/main.go` (execCtx + Reconcile + Cleanup drain),
`cli/domains/runs/handlers.go`.

**Current resolution (2026-08-05):** `restores.{VerifyTarget,RestoreTarget}`
now validate and persist a non-terminal record, then execute on a server-lifetime
background worker. Their status is reconciled fail-closed after restart, and the
transport no longer depends on an unlimited synchronous restore timeout. Resume
of interrupted work (versus fail-and-retry) remains deliberately deferred; v1
fails orphans cleanly.

**Owner:** run-resume — unassigned. **Refs:**
`api/internal/restores/service.go`, `cli/domains/restores/handlers.go`.

### 2026-06-03 — `next_scheduled_at` not yet on the freshness view (resolved)

**Symptom:** `runs status` / `TargetStatus` / `/health` report per-target
`overdue` and `last_success_age_seconds`, but not the *next* scheduled backup
time per target.

**Root cause:** "Next scheduled" for a target = the minimum, over the plans that
include it, of (plan's last fire + its schedule interval). That requires three
inputs the runs status rollup does not have: the target→plans membership, each
plan's schedule, and the **scheduler's in-memory `lastFire`** map
(`internal/scheduler/scheduler.go`), which is not currently exposed through any
seam.

**Resolution (2026-08-05):** The scheduler now exposes a typed
`NextScheduleSource` seam, the runs freshness rollup joins it by target, and the
API/UI surfaces `next_scheduled_at`. Manual-only targets remain unset. The
overdue and last-success rules remain the primary posture signals.

**Evidence:** `internal/scheduler/scheduler_test.go` and
`internal/runs/service_test.go::TestListTargetStatus_NextScheduledAt`.

**Owner:** resolved. **Refs:** `internal/scheduler/scheduler.go` (`lastFire`),
`internal/runs/service.go::ListTargetStatus`, Phase-6 section of
`data-backup-manager-backup-performance-observability-async-execution`.

### 2026-06-03 — Per-target kopia process/connect overhead (Phase 4 descoped)

**Symptom:** Every target costs a fixed ~1.4–1.5s regardless of size — the kopia
process spawn + repository connect per `resource-kopia snapshot create`. A run's
floor is therefore (targets / concurrency) × ~1.5s.

**Root cause:** One `resource-kopia` subprocess per target×destination, each
opening the repository.

**Investigated & rejected — multi-path batching.** `kopia snapshot create A B …`
runs one process for many paths, but `--override-source`, `--description`, and
`--tags` are applied **globally to every path**, not per-path (verified: two
sources batched with `--override-source dbm://shared` both got the colliding
source identity `dbm:/shared`). DBM stamps a **unique per-target**
`--override-source dbm://owner/name` plus per-target tags/description so a
standalone `kopia snapshot list` attributes each snapshot to its target (a
load-bearing, tested feature for Vrooli-independent recovery). Batching would
collapse all targets in a destination into one identity — a regression of that
guarantee. So multi-path cannot preserve per-target granularity and was not
adopted.

**Workaround (shipped):** Phase 1 (async) + Phase 2 (bounded concurrency,
`DBM_RUN_CONCURRENCY`=4) already cut the real 16-target run from ~35s to ~8s; the
per-target floor is masked by parallelism. Phase 3 (in-place fs snapshot) removed
the separate staging-copy I/O on top of that.

**Real fix (deferred):** A warm/persistent kopia connection reused across a run's
snapshots — which requires `resource-kopia` to expose kopia **server/daemon**
mode (open the repo once, submit many snapshots) — would amortize the connect
cost while keeping per-target identity. That is a substantial resource-level
change, out of scope for this plan.

**Spike + decision (2026-06-03) — RE-DEFERRED with rationale.** Benchmarked the
real overhead on the live `elements-local` repo:
- One `kopia repository status` (connect + op): **~0.2s**; the `resource-kopia`
  wrapper around it: **~0.5s** (vault read + registry resolve + spawn).
- A fresh-process `snapshot create` of a tiny path: **3.89s cold, then ~0.32s**
  on the next runs — and those warm runs were SEPARATE processes, proving kopia's
  on-disk content cache already persists across spawns. A warm connection would
  therefore save only the ~0.2–0.5s connect handshake per target, NOT the
  cache-warm-up that dominates a cold run.
- Targets already fan out at `DBM_RUN_CONCURRENCY`=4, so for a 16-target run the
  connect overhead is ~4 waves × ~0.5s ≈ **~2s total per run**, already parallel.

Decision: **re-defer.** The amortizable saving (~2s on a background, scheduled
backup) is marginal, while kopia server mode is a large daemon-lifecycle/auth/
connection-pooling change AND the per-target `--override-source dbm://owner/name`
identity constraint — the same blocker that killed multi-path batching above —
very likely does not survive kopia's server snapshot API (server snapshots are of
registered sources, not per-request override paths). Until that identity question
is answered, the work is high-cost/uncertain-payoff. No daemon scaffolding was
left behind (the spike was benchmark-only). Revisit only if a future profile
shows per-run latency is a real constraint or kopia's server API gains per-request
source override.

**Owner:** unassigned (resource-kopia server-mode is the enabling work). **Refs:**
`api/internal/engine/kopia.go::SnapshotCreate`, `api/internal/runs/service.go`
(fan-out), Phase-4 section of
`data-backup-manager-backup-performance-observability-async-execution`.

### 2026-06-03 — `destinations usage` fails: resource-kopia `repo stats --json` — RESOLVED 2026-06-03

**Symptom:** `data-backup-manager destinations usage <id>` errored with
`kopia … content stats --json: unknown long flag '--json'`.

**Root cause:** External — `resource-kopia repo stats --json` shelled out to
`kopia content stats --json`, but the installed kopia build does not accept
`--json` on `content stats`. Not a DBM defect; DBM only consumes the wrapper.

**Fix (shipped):** `resources/kopia/cli/internal/repo/repo.go::Stats` now derives
the repository's true physical (on-disk, post dedup+compression) footprint from
`kopia blob stats --raw` (exact byte integers) and emits a stable JSON summary
`{"physicalBytes":N,"blobCount":N}`; text mode passes kopia's native output
through. `api/internal/engine/kopia.go::RepoStats` parses `physicalBytes`.
Validated end-to-end: `destinations usage` now returns real bytes
(`usage=11097041777` against `elements-local`). This also supplies the physical
denominator for the dedup metric (Phase B).

**Owner:** resource-kopia. **Refs:** `resources/kopia/cli/internal/repo/repo.go`,
`api/internal/engine/kopia.go` (RepoStats); `cap-4ca780dbb3ba91af` resolved.

### 2026-08-05 — incident baseline and preflight hardening

The Elements volume `/dev/sda1` (UUID `E26A883E6A881189`) is present but not
mounted. The host reports NTFS and the observed condition is compatible with a
dirty volume after an unclean disconnect or shutdown; a power outage is not
proven. The [NTFS-3G FAQ](https://github.com/tuxera/ntfs-3g/wiki/NTFS-3G-FAQ)
documents filesystem checks after unsafe unmounts, which supports this as the
immediate trigger but does not identify what interrupted this drive. The
Ubuntu installer volume is FAT32 removable media and remains a warning-only
candidate, not a primary repository.

The latest scheduled plan run failed its planned targets because the
`elements-local` repository passphrase was unavailable. This is independent of
the volume condition. The legacy target pointing at
`~/.vrooli/secrets.json` is stale; the encrypted credential
authority file is the only eligible recovery target. Do not copy or inspect
plaintext credentials.

The run service now performs a read-only repository/source preflight once per
shared dependency, persists grouped incidents and stable root-cause codes, and
blocks affected fan-out before capture. Permanent operator-action failures are
not retry-storm eligible. Retention and staging-cleanup failures remain visible
as run evidence. Native filesystem repair and credential restoration remain
operator-owned gates.

The initial comprehensive validation run reached 16/19 phases. Standalone unit,
security, and business phases passed; the remaining performance gate measured
Lighthouse performance at `0.71` against the configured `0.75` error floor.
The threshold was not lowered. Subsequent overview/rendering and bundle-hardening
work closed that finding: the final comprehensive run on 2026-08-05 reached
19/19 phases, with Lighthouse performance above the configured floor. Preserve
the earlier result as incident history, but do not treat it as the current
automated validation status. The final production recovery drill remains gated
on recovering the original Kopia repository passphrases and performing native inspection of the
unmounted Elements volume; DBM must continue to report the plan as blocked until
those operator-owned steps are complete. Vault recovery is no longer a DBM gate.

**Evidence:** `data-backup-manager status --json`, `destinations list --json`,
`coverage report --json`, `runs list --json`, `lsblk`, and
`go test ./...` in `scenarios/data-backup-manager/api`.

### 2026-08-05 — P1 requirement ledger corrected

The P1 ledger had overstated delivery. Destination preparation dry-run is
implemented and covered, so `DBM-DEST-DRYRUN` remains complete. Quiesce hooks,
GFS retention, path/point-in-time restore selection, and notification delivery
are not implemented; their PRD and requirement records are now explicitly
planned with actionable validation notes. Logging and platform event seams are
diagnostic evidence, not a substitute for a real notification sink.

**Owner:** unassigned. **Refs:** `PRD.md`,
`requirements/09-post-launch/module.json`.

The recovery-drill scheduler also now rejects malformed or non-positive legacy
cadence values with a plan-scoped error instead of silently skipping them.
Failures while persisting the drill's running transition are recorded as
actionable failed evidence when catalog storage permits it.

**Evidence:** `api/internal/drills/service.go` and
`api/internal/drills/service_test.go`.

Backup run lifecycle transitions and per-outcome evidence writes now fail closed
when catalog persistence fails: a run cannot proceed past the capturing
transition, and persistence errors are included in the terminal run evidence.
Generic SQLite audit metadata probes likewise mark the inventory failed when a
page or schema query errors instead of converting the error to zero/empty facts.

**Evidence:** `api/internal/runs/service.go`,
`api/internal/runs/service_test.go`, and `api/internal/audits/dbcheck.go`.

### 2026-08-05 — Vault removed from the greenfield DBM/Kopia stack

The credential-authority recovery bundle was verified with the operator-held
recovery passphrase. It contains ten legacy credentials, but no Vault unseal
key and no `baseline-safety` or `elements-local` Kopia passphrase. Restoring it
would not recover those values, so it remains preserved and was not imported.

The greenfield contract is now credential-authority-only for DBM and
resource-kopia: native OS credential storage or encrypted authority storage is
the live source, recovery bundles are the portability mechanism, and Vault is
not a lifecycle dependency. Repository passphrases and future S3 fields use
per-repository authority identities. Existing encrypted repositories remain
preserved but cannot be certified without their original passphrases.

**Evidence:** `vrooli credentials recovery verify` output supplied by the
operator; `resources/kopia/cli/internal/credentials/`; `resource.json`;
`scenarios/data-backup-manager/.vrooli/service.json`; `PRD.md`;
`docs/internal/DECISIONS.md`.

## Architecture Drift

Use this section for deferred findings from `screaming-architecture-audit`.
Do not create a standalone architecture-audit report unless the work is
a migration handoff with a planned retirement path back into
`ARCHITECTURE.md`, `SEAMS.md`, or this file.

| Area | Drift | Maturity Impact | Real Fix |
|---|---|---|---|
| _None yet._ |  |  |  |

## Work ladder

- Rung: W0
- Evidence: the active `data-backup-manager-v2` goal describes an engine-backed capability in which scenarios self-register targets, matching PRD `OT-P0-001` (`Scenarios idempotently register/deregister backup targets`) and the API/CLI/UI registration surfaces. The separate active `persistence-inventory-and-contracts` goal says `NO per-scenario registration needed` and directs removal of `swarm-manager/api/internal/backup/register.go`, ending with `discovered by convention, never self-register`.
- Blocker: contract direction is contradictory. Do not rewrite the PRD or remove the registration surface until the operator decides whether convention-based discovery is only the default for api-core/storage state (with explicit registration retained for exceptions), or replaces OT-P0-001 entirely. Lower W1/W2/W3 gates are not authoritative while this W0 conflict remains unresolved.
- Measured: 2026-08-05

## Cross-references

- [`PROGRESS.md`](PROGRESS.md) — lifecycle log (forward-looking)
- [`SEAMS.md`](SEAMS.md) — boundary registry (load-bearing for tests)
- [`TESTING.md`](TESTING.md) — test patterns
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — generic-template issues

## Recovery identity re-resolution — 2026-11-03

An identity-bound recovery journal stores both the device's last observed path
and its stable UUID/serial. `ReadOnlyInspector.InspectDevice` previously chose
the saved device path whenever present, even when stable identity was available.
After reboot or replug, device enumeration can assign the Elements volume a new
path; the stale path can fail or refer to another volume. Recovery now resolves
by stored UUID/serial first, then verifies the returned device against the
stored filesystem identity before inspection. The volatile device path is used
only when no stable identity exists. If stable resolution is unavailable or
returns a mismatch, inspection fails closed.

This fixes the identity lookup defect in the resume owner. The 2026-11-03
commission also added the lifecycle-managed safe-remount monitor, mount-failure
notifications, a durable Notification Hub repair decision and signed
human-only approval callback, and retryable terminal result notifications.
The monitor remounts only after a positive read-only check, then verifies the
mounted/read-write result. It does not run filesystem repair without the exact
human approval.

The cross-platform owner path now resolves macOS unmounted state from
`diskutil info`, rather than depending only on the mount table. Windows accepts
the stable `\\?\Volume{GUID}\` selector when a drive letter is absent,
resolves it through `Get-Volume -UniqueId`, and invokes `Repair-Volume` against
that exact volume object. A Windows check is considered consistent only when
the cmdlet returns an explicit zero result; absent or nonzero output is not
treated as clean. Safe remount restores the recorded non-system drive letter
only if it is still unassigned, then re-observes the same volume GUID. Windows
system-root protection recognizes `C:\` paths independently of the control
plane's build host.

Focused `volumeremediation` and DBM inspector/readiness/recovery tests pass;
Windows and macOS test targets cross-compile. Test Genie did not start because
its provider catalog rejects the unrelated scratch scenario
`scenarios/.browser-automation-studio-e7-repeat-durable` whose declared scenario
name differs from its directory. These checks do not establish native Windows
or macOS hardware recovery. NTFS remains read-only on macOS. Linux Elements
repair/remount evidence is separately recorded in the active recovery cohort.

## Workspace checkpoints and isolated validation — 2026-09-08

## Portable destination identity and recovery locator — 2026-09-08

The destination catalog now persists a validated path relative to the observed
volume mount, alongside the last observed absolute path and device identity.
The same relative locator is included in the self-describing bundle manifest
and the recovery journal. This prevents a replugged drive or changed Windows
drive letter from being treated as the same destination solely because an old
absolute path still exists. Relative paths that escape the mount root are
rejected. Existing rows and manifests remain readable; the field is additive
and empty for legacy or S3 destinations until they are observed again.

Stable identity inventory is now explicit and testable on all supported build
targets: Linux enriches partitions from the parent disk's model/serial,
macOS uses `diskutil info` with a UUID, and Windows uses a read-only PowerShell
`Get-Volume` unique ID query. Every adapter has an injectable command seam;
fixture tests never run a host command. Cross-builds pass, but native runtime
evidence is still required before claiming hardware recovery support on macOS
or Windows.

Focused Test Genie validation `20260908-061400-3f13eaef` passed both unit and
proto phases. The run still reports advisory debt: two broad missing-seam
findings, low-coverage warnings in unrelated generated/adapter surfaces, and
the existing deprecated proto annotations/cross-domain layout findings. These
are maturity follow-ups, not evidence that any production backup or recovery
operation was executed.

User-authorized scope extends DBM to explicit workspace checkpoints. Registration
semantics remain unchanged; the older discovery-versus-registration dispute is
not resolved or used to remove an API. New OT-P0-012/013 describe preservation
and test isolation. Requirements remain in progress, not certified complete.

Implemented: additive workspace source kind across proto/API/CLI/UI; staged
versioned manifest; bounded inventory, source-drift checks, final-tree comparison;
Linux basic metadata profile with explicit unsupported-condition refusal;
confined exclusive copying, modes and modification times; verification through
the final workspace adapter; production command refusal in Go tests; production
test-mode RPC refusal; fixture-only proof script; ephemeral destination refusal.
The source remains live: observed stable scans are NOT atomic capture evidence.
No application-consistency or full-machine backup claim is made.

Operational evidence: baseline-safety/system-monitor preflight passed at
2026-09-08T03:13:40Z. Elements remains visible as an unmounted NTFS partition;
its repository cannot be certified. Active plans reference missing temporary
repositories. Historical credential errors did not recur in fresh preflight.
No passphrase rotation, credential import, mount, plan rebind, backup, restore,
or production restart was performed. A permanent destination decision remains
pending; merely changing a catalog path cannot recover a missing repository.
The installed native inspection command was stale and rejected `--device`
despite its source handler requiring it (QA knw-1788838120038500527). The CLI
manifest and user-local binary are now aligned; do not bypass native safety
gates or repair the volume as a private DBM operation.

Validation: all API and CLI package tests passed. Focused recovery/source/
preflight/engine/effect-guard tests passed with race detection. TypeScript and
scoped lint passed; 11 focused UI tests passed. Tests used fake engines/commands,
temporary databases and synthetic filesystem fixtures, never real Kopia backup
or restoration. Test Genie 20260908-032512-1450d01d passed structure and proto;
unit failed with UNIT_REQUIRED_ROLE_MISSING. QA knw-1788838015617743856. No
full-scenario readiness claim. Scoped proto refresh generated the changed
contracts but failed its downstream template-module parsing step (previously
recorded tooling problem); the proto phase confirms generation sync is clean.

## Recovery safeguards implemented — 2026-09-08

The volume recovery design now has an explicit read-only control-plane
capability contract (`host-volume-v1`), additive catalog columns for the last
observed device identity, and a private atomic JSON journal for resumable,
identity-bound remediation steps. A journal advances only after the control
plane reports a satisfied end state and the progress write succeeds; refusals
pause at the current step, retries accept `already_satisfied`, and dry-runs do
not advance the cursor. The journal stores no credentials.

The installed user-local CLI was rebuilt from source and atomically replaced.
`vrooli host volume capabilities --json` returned the expected contract. A
read-only inspection of `/dev/sda1` then confirmed the NTFS UUID, serial, size,
and unmounted state; no check, repair, mount, unmount, format, or backup was
performed.

Validation passed for the DBM API, control-plane volume remediation, and
`vrooli-cli-go` packages, including race detection; focused UI test,
type-check, and lint also passed. `make install` itself remains blocked by an
unrelated dirty-worktree package-adoption enum value, so the equivalent source
build and atomic user-local install were used. Test Genie structure/proto/unit
was queued behind another run and was not treated as a pass.

The recovery boundary was tightened after review: the mount adapter now uses
the plan's requested mount root when the volume is intentionally unmounted,
and the CLI accepts UUID/serial guards alongside the device path. The
control-plane capability probe now rejects a matching version that omits any
recovery action, and POSIX journal saves sync the parent directory after the
atomic rename. These changes were validated with fakes and race-enabled unit
tests; no host volume action was run.

The durable journal is now exposed through additive API and CLI operations for
starting, inspecting, and resuming a sequence. The generated TypeScript client
and React Query hooks are wired with dry-run as the default; a dedicated visual
recovery panel remains future UI work so the operator-facing flow can present
the confirmation and host-action boundaries clearly. Stable UUID/serial
resolution was implemented through the Linux `lsblk` inventory seam; macOS and
Windows native inventories had not yet landed at the time of this September
record. As of the 2026-11-03 follow-up, both DBM identity adapters and the
Windows control-plane observer exist with fixture coverage. Native hardware
recovery remains unverified on those platforms; Windows filesystem health is
deliberately `unknown` in its observer.

Recovery journal validation now also binds every step to the journal's absolute
mount location, requires a strong identity on every step, and rejects a file
whose embedded journal ID does not match its filename. These checks protect a
resume from redirected, stale, or manually corrupted state. Governed package
discovery now skips unresolved source-template module placeholders while
continuing to reject malformed real modules; the proto refresh completed and
updated DBM's installed generated client without manually editing a dependency
snapshot.

Follow-up validation on 2026-09-08 completed Test Genie structure, proto, and
unit phases successfully (`20260908-054459-14bb48bd`). The run still reports
advisory maturity findings: deprecated `@domain` annotations and cross-domain
proto reuse, plus two missing injectable seams in the broad unit-health scan.
Those findings do not permit a live recovery operation by themselves; keep the
recovery path behind its existing control-plane identity and confirmation gates.

### 2026-11-03 — NTFS-3G mount lacks a trustworthy clean-state signal

**Symptom:** The selected Elements destination is mounted read/write through
`ntfs-3g`, but DBM readiness remains UNKNOWN and blocks backup writes. The
control plane now resolves the on-disk format as NTFS and records `fuseblk` as
the separate mount driver. A consistency check is refused while the volume is
mounted, because the supported checker requires an unmounted volume.

**Root cause:** The Linux FUSE mount publishes `fuseblk` rather than its
on-disk NTFS type and does not expose the NTFS clean/dirty state while mounted.
The control-plane observer previously conflated the mount driver with the
filesystem format; that observer defect is fixed. A successful repair or
read/write mount is not clean-state evidence.

**Workaround:** Keep backup writes blocked while readiness is UNKNOWN. Do not
change the readiness gate based only on mount success. A read-only host
inspection may confirm device identity, mountpoint, driver and write mode, but
not filesystem consistency.

**Real fix:** The control-plane observer now retains on-disk format and mount
driver separately, and safely refuses a check while mounted. The remaining
work is to provide a trustworthy NTFS consistency observation after a safe
unmount, then remount only after a positive check or an exact verified-human
repair decision. Keep readiness UNKNOWN while the volume is mounted through a
driver that exposes no clean/dirty state. Re-evaluate capacity separately
before enabling backup writes. DBM recovery episodes are persisted per
destination, so repeated outages after a successful mount can raise a fresh
recovery and decision instead of being suppressed by an old completed journal.

**Owner:** Vrooli control plane (`internal/volumeremediation`) with DBM
readiness integration.

**Refs:** `internal/volumeremediation/observer.go`,
`scenarios/data-backup-manager/api/internal/sysmounts/volumestate.go`,
`docs/operations/RUNBOOK.md`, and the 2026-11-03 DBM recovery checkpoint.

### 2026-11-03 — Elements Kopia repository cannot enumerate manifests (open)

**Symptom:** The registered encrypted `elements-local` destination fails both
`data-backup-manager destinations usage` and
`resource-kopia snapshot list --repo elements-local --json`. Kopia reports a
missing manifest/cache blob (`q6390189f3d3250c965c737fd82cc103e-s42b636fa5d06735f144`),
and `stat` on a repository `.tmp` entry returns `EIO`. Treasury has no plan on
this destination; its separate same-host validation backup does not make this
Elements repository usable.

**Root cause:** Not yet distinguished. The filesystem's identity-bound
read-only check returned `consistent=yes`, but that check does not verify the
Kopia repository. The path-level I/O error and missing manifest blob are
consistent with repository/filesystem corruption or an underlying storage
fault. The exact unsafe-shutdown causal chain is not proven. NTFS-3G's
userspace filesystem behavior is also a repository risk: Kopia requires
reliable POSIX-style atomic and read-after-write behavior.

**Workaround:** Do not write new snapshots to this repository, clear its local
Kopia cache, or delete repository files. Preserve it for diagnosis. The
2026-11-03 Treasury validation plan on `baseline-safety` proves backup and
scratch-restore flow only; that destination shares the host disk and does not
qualify as off-host recovery protection.

**Real fix:** The first owner probe, `kopia repository repair filesystem`,
only supports recovery of a repository format blob; its dry-run returned no
reviewable action and does not target the missing snapshot manifest. Do not
present it as a general repository repair. Diagnose the missing manifest and
underlying `EIO` at the resource-kopia/control-plane boundary, then obtain an
exact verified-human decision only if a repository-specific repair with a
reviewable effect is found. Qualify a suitable destination, add Treasury's
plan only after capacity/retention are sized, and verify a scratch restore.
Keep DBM's readiness and backup gates fail-closed when repository health cannot
be established. Treasury's 424,778,594-byte target now has a verified scratch
restore from snapshot `9c685de6269fade090581b3b798009f1` on the healthy local
repository, but the funding gate remains closed pending an independent
off-host copy and restore.

**Owner:** resource-kopia repository lifecycle, DBM destination/readiness and
run health, with control-plane storage diagnostics.

**Refs:** `resources/kopia/cli/internal/repo/repo.go`,
`resources/kopia/cli/internal/snapshot/snapshot.go`, DBM repository-integrity
checkpoint `~/.vrooli/research/cohorts/dbm-repository-integrity-20261103/`,
[Kopia consistency and corruption guidance](https://kopia.io/docs/advanced/consistency/),
[Kopia filesystem repository repair reference](https://kopia.io/docs/reference/command-line/advanced/repository-repair-filesystem/).

**Follow-up evidence, 2026-11-03 23:06Z:** The 13 Elements plan targets still
have failed `repository_invalid` outcomes from the 21:18Z run; none of those
targets was captured, and their last-success evidence remains overdue. A
read-only recursive walk found no zero-length files and 13 separate `.tmp`
content paths returning `EIO`. `resource-kopia repo validate --name
elements-local` returned exit code 0, but snapshot enumeration still fails on
the missing manifest; the wrapper's repository validation does not establish
that every manifest/content reference is readable. An offline NTFS consistency
check previously passed, so neither that result nor Kopia's validation
distinguishes filesystem-driver behavior from device/transport faults. Do not
run repair, prune, or backup writes against this repository until the I/O
cause and a repo-specific recovery plan are qualified.

The disabled `elements-daily-runtime` plan has 14 targets and `keep_latest=3` (operator decision applied 2026-11-04). The retention change does not enable or run the plan.
For a conservative ~325 GB aggregate source estimate, current Elements free
space supports three copies (~976 GB) but not seven (~2.28 TB). The estimate is
uncertain because some state files were unreadable and nested target roots
overlap. `baseline-safety` has about 379 GB free on the same system NVMe and
cannot hold three copies of that aggregate. Notification Hub ask
`73b5d631-7ba3-40cf-90c6-5380cf79c8fb` was answered `retain3`; DBM initially
blocked the update because 63 recommended defaults are not registered. The
operator-authorized update used `--allow-incomplete-coverage`, changed only
retention metadata, and was read back as `keep_latest=3`. The plan remains
disabled and no run was made.

**Follow-up, 2026-11-04 01:04Z:** The operator also answered `commit` to the
exact additive index-recovery decision
`ad0a46dc-fac9-4704-92d5-4a600190b625`. The registered-filesystem owner command
committed the recovered indexes and did not delete indexes. Fresh
`resource-kopia snapshot list` still fails on the missing manifest blob
`q6390189f3d3250c965c737fd82cc103e-s42b636fa5d06735f144`; fresh DBM destination
usage reports the same missing blob and `EIO` on
`s/f43/64b7506cef44e4fbb3fc66ba7f2d9-sf40366ce03934d79144.f.tmp.8c9d63ca28444838`.
Therefore index recovery repaired neither the missing manifest reference nor
the failing filesystem path. The exact volume remains readable enough for an
identity-bound offline check (`consistent=yes`, `changed=false`) and RW remount,
but readiness reports mounted NTFS state UNKNOWN. `sudo -n smartctl -a /dev/sda`
cannot run without a password, and `dmesg` is restricted, so device/transport
health remains unclassified. The 14-target plan remains disabled at
`keep_latest=3`; 13 target outcomes are still failed and overdue. No new backup,
prune, or repository-specific repair was run. The follow-up result notification
was delivered (`c3bc384a-70cc-47c3-a3bc-97b435c13a23`, web-push receipt
`cf77d26a-388a-414c-8575-84623c060c0c`).

Kopia's current corruption guidance treats recovery as case-specific and
recommends trying a read-only repository open/list from another computer when
local cache corruption is plausible; it warns not to clear the cache on the
affected computer because it may contain recovery data. This is now the next
safe cross-platform discriminator if another authorized host is available. Do
not delete the reported `.tmp` path or run a general repository repair: the
registered repair command only recovers the repository format blob, and the
index commit did not fix this manifest/EIO case.

**Follow-up, 2026-11-04 01:31Z:** `resource-kopia repo cache-probe --name
elements-local --json` repeats snapshot enumeration using an isolated temporary
Kopia configuration and cache. It failed on the same missing manifest blob,
then removed its temporary state; corruption limited to the registered local
cache is now unlikely. This does not distinguish repository-object loss from
device, cable, bridge, or filesystem-driver I/O. The exact device reports USB
mass-storage transport, SCSI state `running`, and writable block nodes. DBM's
mounted NTFS clean/dirty state remains UNKNOWN. `sudo -n smartctl` requires a
password and kernel logs remain restricted. A discovered Mac Bridge node has no
reported attached drive; its read-scope relay call was refused before dispatch
because `vrooli:read` is not granted. The next diagnostic needs either
host-authorized read-only SMART/kernel evidence or a permitted second-computer
read-only open of this same repository. Preserve the registered cache and
repository; no backup, prune, repository repair, or filesystem mutation is
authorized by this diagnostic result. The `elements-daily-runtime` plan remains
disabled at `keep_latest=3`, with 13 failed/overdue targets.

**Follow-up, 2026-11-04 01:56Z — exact orphaned manifest candidate isolated:** Kopia cache documentation says its metadata cache stores `q` blobs. The registered cache contained only shard descriptors and no metadata/content blobs. Resource-kopia's disposable overlay probe found exactly one adjacent temp for the missing manifest object: 75,403 bytes, SHA-256 `e67b09268188205b39ae0389e16fee268f1f4a032d94f855907083eceffedc80`. The final blob path is absent. Copying this candidate into a temporary repository overlay under its expected sharded path made snapshot enumeration pass (`snapshot_list_passed`, output SHA-256 `94aeb2d1f95b0496a43379e486d9aaa8fb6e26b3780f69fda4cfe4d0c3c34747`); the source candidate and repository were untouched. This strongly isolates one missing manifest blob and supplies an exact repair candidate, but does not explain the other repository `EIO` paths or prove media health.

Resource-kopia now exposes `repo orphan-temp-blob-commit`: an exact repository/blob/hash-bound, registered-filesystem-only action that requires one adjacent candidate, stages and rehashes it, atomically links only to an absent final path, and preserves the source temp. Go build and resource install succeeded. The exact repository write is pending Notification Hub decision `8939a7e8-8ffa-4604-92d9-24701256638f` (recommended `restore_blob`; built-in Other and dismiss are available). No Elements repository write has run. Plan `elements-daily-runtime` remains disabled with 13 failed/overdue outcomes; post-repair snapshot enumeration, DBM destination readiness, target backups, and independent restore proof remain open.

**Decision and follow-up, 2026-11-04 01:58Z:** Notification Hub decision `8939a7e8-8ffa-4604-92d9-24701256638f` was answered `restore_blob`; exact blob `q6390189f3d3250c965c737fd82cc103e-s42b636fa5d06735f144` was published with the reviewed SHA-256, source candidate preserved, and no target replacement. Follow-up notification `ab478525-9676-4f49-965c-ddaf018f02e1` reports the result. Snapshot listing advanced and now fails on a *different* missing blob, `q057622e37ec0310f43e73da759cd37c3-saba5e3ba867f380d144`. A fresh isolated overlay probe found its one adjacent 4,339-byte temp candidate (SHA-256 `ba7fcb1f9d85c6b528abd5f424e232c870615d0f3bad4db28bc647ec025da29c`) and snapshot listing passed in the overlay. Its separate write decision `9d6c6756-75ed-4ab4-aaef-c0944f30c0f5` is pending. DBM readiness still reports mounted NTFS state UNKNOWN; destination usage still fails on missing manifest data and separate EIO. Plan remains disabled and no backup/restore/prune has run.

**Second repair outcome, 2026-11-04 02:02Z:** Notification Hub decision `9d6c6756-75ed-4ab4-aaef-c0944f30c0f5` was answered `restore_blob` by a verified human. The exact 4,339-byte candidate for `q057622e37ec0310f43e73da759cd37c3-saba5e3ba867f380d144` was published with its reviewed SHA-256; its source temp remains and no target was replaced. Follow-up notification `950de5b0-85d3-4277-98d1-1b2981906da5` was sent. Snapshot listing then advanced to another missing blob `q24decdf22b7ca74f9f48f99ca06937d0-sf40366ce03934d79144`; a fresh `cache-probe` is still running to inspect a candidate/overlay. Destination usage continues to fail on an unrelated repository `.tmp` EIO. The plan remains disabled.

**Third candidate and pending decision, 2026-11-04 02:04Z:** The completed isolated-cache probe after the second repair surfaced missing blob `q94792c43d91d65761874f12697342f99-sb8e29a2fa4be3332144` (the immediately preceding run's first error was q24...). Its sole adjacent temp candidate is 4,298 bytes, SHA-256 `7cd3be7cb18d6491c10d57708286902eac859b30ff8bf28f6b3cebdfffa7c885`; adding it only in the disposable overlay makes snapshot listing pass. Exact write decision `8bd6a969-dad4-4533-8db8-c23e82a30733` is pending. Continue keeping the plan disabled. This nondeterministic discovery of different missing manifest blobs plus independent EIO shows the repository has more than one missing-object symptom; do not claim the first or second repair fixed the repository.

**Fourth repair outcome, 2026-11-04 02:09Z:** Notification Hub decision `f708e982-9515-4354-96c1-83102e5d5373` was answered `restore_blob` by a verified human. The exact 4,298-byte candidate for `q06ceaedaf81c135d865086ea1abd7f43-s02fff88589daa810144` was published with its reviewed hash, preserving the temp and replacing nothing. Follow-up notification `b335ea69-dae6-498b-93c9-739b0c3f5a52` was sent. To disambiguate the persistent direct error `q24decdf22b7ca74f9f48f99ca06937d0-sf40366ce03934d79144`, a new `cache-probe --blob-id <exact-q-id>` diagnostic now tests that exact ID in its disposable overlay even if another concurrent manifest lookup reports a different first error. Its q24 probe is still running. No new repair is authorized yet.

**Targeted persistent-blob probe, 2026-11-04 02:10Z:** The optional `cache-probe --blob-id q24decdf22b7ca74f9f48f99ca06937d0-sf40366ce03934d79144` found its unique adjacent temp (4,298 bytes, SHA-256 `96aab4dd1e9462f9aa76e10e360637438b05b5d717e7db183f7d78572c1efd1d`) and passed snapshot listing in the disposable overlay. Its exact repair ask `ff3850cf-985e-4a32-9b60-4d39c1f83872` is pending. The baseline concurrent read surfaced a different missing blob, `q3d011f23e6f9b0bc1353db1c0a56ef35-s4654612484971716144`, further confirming multiple manifests are involved. No q24 write has run.

**Fifth and sixth repair outcomes, 2026-11-04 02:13Z:** Verified-human answers approved both exact asks: q24 decision `ff3850cf-985e-4a32-9b60-4d39c1f83872` and q3d decision `3b348a30-4f10-4eb0-8d04-41ef5ef6d477`. Both hash-matched candidates were published only at absent paths, with temp sources preserved and no replacement. Notifications `83432d22-3ad5-4d8c-a367-152806856e8f` and `1d028494-5d69-4631-b75b-664d88b65861` report the outcomes. Fresh reads then surfaced `qe0901a11462c74b3ab3c92efedcfde21-s77cc3c6509f55fdd144`; its one adjacent candidate is 4,298 bytes with SHA-256 `0a319af19058b144a60c610884c166a125de3c106f3634dc63100da3e8ed20f7`. Its targeted disposable overlay passes snapshot listing. Decision `e2466a33-63b2-4b92-994d-57d5949462ec` is pending. The separate repository EIO remains.

**Seventh repair outcome and eighth ask, 2026-11-04 02:14Z:** Verified-human answer to `e2466a33-63b2-4b92-994d-57d5949462ec` approved the qe090 blob; its exact 4,298-byte candidate with SHA-256 `0a319af19058b144a60c610884c166a125de3c106f3634dc63100da3e8ed20f7` was published, source preserved, no replacement. Notification `d43c2a91-d481-4f79-9753-797527f18537` reports completion; follow-up `c454f95c-f21e-4164-8856-54d3343f8d6e` reports verification. Snapshot and usage reads advance to qcae `qcae9b48442746ccef0ccbd13bff045b4-saa974d19571ef3b8144`, whose one 61,888-byte adjacent candidate (SHA-256 `3da36b6bde9be380027c05f3d02553deaa7ff49f814e8e54f93f16c53f5daafe`) passes targeted overlay enumeration. Exact qcae ask `a1417609-c979-443c-8783-58e73a3023f4` is pending. Keep the plan disabled; separate EIO remains.

**Eighth repair outcome and ninth ask, 2026-11-04 02:16Z:** Verified-human answer to `a1417609-c979-443c-8783-58e73a3023f4` approved qcae; the 61,888-byte exact candidate was published by its reviewed hash, preserving the source temp and replacing nothing. Notification `f549220e-b74b-454d-8f7e-7dba2359186a` was sent. Snapshot/usage reads then surfaced qf020 `qf0203aff1f3577a39a6a274eab1f68e8-s6b70871c747d5cad144`; its 4,339-byte unique temp (SHA-256 `3d8ae3f9542906f48ef0a7399a25318ccf45a972a51a124fb15b2e089bfd751b`) passes targeted overlay enumeration. Ask `ec30d056-0f45-4bc2-a137-4d8cbc4d7ab0` is pending. Separate EIO remains.

**Standing authorization and current boundary, 2026-11-04 02:22Z:** Verified-human Notification Hub answer `Other` on qf020 ask `ec30d056-0f45-4bc2-a137-4d8cbc4d7ab0` included: “I approve of this blob restore and all future ones. Stop asking!” This explicitly authorizes repeated restores only for the same narrowly defined orphan-q class: one unique adjacent temp; candidate hash checked; final path absent; disposable overlay proves snapshot enumeration; atomically link without replacing anything and preserve the temp. No more per-q decisions are needed for that class. Nine q blobs have now been restored under the exact evidence gates, and Kopia snapshot listing succeeds (192 total, 189 DBM-tagged).

The control plane then performed a safe offline filesystem check by unmounting the exact drive (`fuser` showed no users), got `consistent=yes`, `changed=false`, and remounted it read/write. Mounted NTFS readiness still says UNKNOWN. The separate repository temp path `s/f43/64b7506cef44e4fbb3fc66ba7f2d9-sf40366ce03934d79144.f.tmp.8c9d63ca28444838` remains the hard blocker: `stat` returns EIO, although its parent shard is readable and its directory listing shows only the temp name. Kopia repo stats and DBM destination usage still fail on that path. Do not delete/move it or enable backup, restore, or prune yet. Next safe evidence is control-plane/host-owner SMART and kernel transport diagnostics or a read-only open/list from another authorized computer. The plan remains disabled with 14 targets, retention three, and 13 old failed/overdue results; the independent off-host restore remains unproven.

**Unpaired content-temp EIO, 2026-11-04 02:30Z:** After the nine q-manifest restores, snapshot enumeration passes but `resource-kopia repo stats` and DBM destination usage still fail on `s/f43/64b7506cef44e4fbb3fc66ba7f2d9-sf40366ce03934d79144.f.tmp.8c9d63ca28444838` with `EIO`. The shard directory itself is readable and lists only that `.tmp` name; the corresponding final `.f` is absent. `stat` returns EIO for the temp. A resource-kopia owner action was added to move one exact unpaired `s` temp into a unique sibling quarantine without reading/deleting its bytes; it built and installed, then the exact move failed with EIO. The source remained listed and no quarantine directory remained. Offline control-plane NTFS check returned `consistent=yes`, `changed=false`, with the drive remounted read/write; mounted readiness remains UNKNOWN. DBM status is still degraded on 13 failed Elements target postures. Do not retry the move, delete the temp, enable the plan, or run backup/prune/restore until SMART/kernel transport evidence or another authorized read-only host clarifies this file-level EIO.

**Recovery monitor activation, 2026-11-04 02:35Z:** DBM was restarted via
`vrooli scenario restart data-backup-manager`, activating the volume monitor and
signed decision callback in the running process. Vrooli Events reports the
enabled owner subscription (`id=16`) to `/api/v1/events/notification-hub`.
The monitor observes Elements mounted at its recorded root and does not run
filesystem repair for a mounted device. Its startup observation is healthy;
the remaining EIO is repository-object I/O rather than a mount failure. Direct
read-only usage still fails on the same exact `.tmp`; passwordless SMART is
unavailable, UDisks exposes no health attributes, and readable kernel logs
show no matching device errors. Preserve the temp and keep backup writes off.

**Preflight trusted repository stats, 2026-11-04 02:52Z:** The live Elements
plan preflight returned `ready=true` while `destinations usage` and Kopia repo
stats failed on an EIO path. Preflight checked repo status and snapshot
metadata only; it did not verify that usage could be measured, even though
capacity and retention decisions rely on that probe. A run could therefore
start capture while usage/capacity evidence was missing. Preflight now requires
`RepoStats` to succeed after status and manifest listing. The rebuilt, restarted
API now reports a grouped `repository_invalid` incident across all 14 plan
targets before source capture. This closes the unsafe write path but does not
repair the repository or prove filesystem readiness. Keep the Elements plan
disabled and preserve the EIO temp.

**Capacity estimate remains a lower bound, 2026-11-04 02:54Z:** A metadata-only
size scan of the 14 registered Elements-plan source locators summed to
328,472,976,155 bytes. With three retained copies, the destination free-space
check passes against 1,621,584,723,968 bytes available. The scan reported
permission-denied files under `.vrooli/state`, so this is not a complete upper
bound and must not be treated as final capacity qualification. Repo usage still
fails with EIO and preflight blocks before capture.

## 2026-11-04 — source enumeration and transport diagnostics extended

DBM preflight previously checked only `stat` on the root of filesystem and SQLite sources. The live `elements-daily-runtime` plan includes `~/.vrooli/state`, whose nested Agent Manager receipt path `.../planner-live-same-id-20261019/retry-preflight` is mode `0600` and cannot be traversed. A root-only check incorrectly allowed this target to proceed into capture. Preflight now uses the same filesystem walk as capture to estimate local sources, marks an incompletely enumerated target as blocked, and passes a complete logical source estimate plus `keep_latest` to the shared destination capacity check. Live read-only preflight reports that target separately from the destination repository EIO. Capacity remains unknown until the source tree is fully readable.

The operator answered Notification Hub ask `d085cf69-1c61-4e4c-ac8e-d78b746dd32a` with “hold repair; obtain SMART/transport or alternate-host diagnostics first.” No drive mutation was performed. The direct ask was not correlated to a DBM recovery journal and therefore could not dispatch a repair callback; treat the human answer as the operator's direction, not an executable recovery receipt. `system-monitor` identified `/dev/sda` as the Elements USB/SCSI disk but SMART is unavailable: the device is `root:disk` mode `0660`, and the monitor user was denied access. Read-only SCSI counters showed `state=running`, `device_blocked=0`, `iotmo_cnt=0`, while `ioerr_cnt` increased from `0x1787` to `0x1789` as request count advanced from `0x80606` to `0x807ad`. This is evidence of SCSI command errors but does not by itself attribute them to the exact Kopia object or establish SMART health. Kernel log filtering found no matching storage/USB error message. Keep the EIO object intact, the plan disabled, and do not run backup, restore, prune, or broad filesystem repair.

**Transport counter attribution check, 2026-11-04 03:17Z:** One paired direct `stat` on the exact EIO temp and one DBM destination-usage request both returned the same EIO, but neither changed the SCSI request/error counters (`0x807ad` / `0x1789` before and after). The earlier two-error counter increase is real but cannot be attributed to this exact Kopia path from current evidence. SMART remains unavailable due `/dev/sda` permissions; obtain a permitted SMART report or alternate-host check before repair.

**Post-repair repository check, 2026-11-04 05:44Z:** Verified-human approval
completed one identity-bound NTFS repair/remount through DBM's source-bound
callback (decision `1cef04aa-0df3-458f-825f-7441bb3f0d44`, recovery journal
`739086a9fc6e63cd5ea2c97b`). A DBM observer defect initially misclassified the
mounted device because stable `lsblk` identity lookup omitted live mount facts;
the inspector now joins identity back to a fresh mounted-volume scan, and the
journal completed with the drive verified read/write. Read-only isolated-cache
Kopia snapshot enumeration passes, but repo stats still hit EIO on the exact
unpaired `s/f43/64b7506cef44e4fbb3fc66ba7f2d9-sf40366ce03934d79144.f.tmp.8c9d63ca28444838`.
After the approved filesystem repair, the resource-kopia exact quarantine
command retried its same-volume move; `rename` returned EIO again. The source
temp remains the sole matching shard entry and no quarantine directory remains.
No Elements repository bytes were deleted or replaced. Preflight still blocks
all 14 targets. The filesystem repair did not resolve this object-level I/O
failure; a separate authorized-host diagnosis is still needed before another
repository mutation or backup write.

## 2026-11-18 — snapshot identity, capacity, and degraded readability (code fixed; live proof pending)

**Symptom (was):** `destinations snapshots` returned “unidentifiable or
mismatched snapshot” for targets that had snapshots; runs preflight reported
`repository_invalid` for a 218-target local plan; a 218-target plan failed every
target on capacity while an incremental run added only ~1.85 GB; and a
mounted-but-unknown Elements volume could not even be read for snapshots or
usage.

**Root causes:** (1) a 128-manifest cap in `KopiaCLI.SnapshotList` rejected
valid repositories with 269 snapshots; (2) Kopia writes tags as a
`tag:`-prefixed JSON object, but the engine decoded only arrays, so every
tagged snapshot looked unattributed; (3) `retentionCopies` summed every GFS tier
and multiplied the full source estimate; (4) `WithResolvedDestination` required
write-ready filesystem evidence for read-only operations.

**Resolution:** byte-bounded snapshot inventory with normalized tags; tag-first
identity with a legacy source-suffix fallback; `internal/capacity` admission
model; `WithResolvedDestinationForRead`; explicit `WriteVerdict` plus a bounded
write probe. See `DECISIONS.md` 2026-11-18 and `INVARIANTS.md`.

**Live proof:** run `1f3fc427-3d46-49c9-b10f-aa048af744bd` completed with
snapshot `55e9da76f6efc57ff7ad33c653e0f65c`; drill
`f5a31874-0aac-4943-bc10-b82712ffba2d` restored it to scratch and reached
`DRILL_STATUS_VERIFIED` (restore `fb6643cf-d17b-435a-9c68-4c0f77ae4d22`).
External snapshot listing and usage return live values in ~6s. The 218-target
local plan preflights with only the known `adguard-home` permission denial.

**Resolved since:** the 218-target local run completed 213/218 (4 real
permission/path failures, 1 blocked); Elements usage reads 397.6 GB; a
degraded write to the Elements drive succeeded for all 17
`free-disk-root-migration` targets. The UI mounts the library design tokens, so
the sidebar no longer collapses.

Test Genie `20261118-195708-2653aa5f` passed unit, contracts, and storage
(warnings only) after the storage declarations and CLI binding fixes.

**Still open:** a restore/drill from a *new* Elements snapshot (the verified
drill used the existing 12 Nov private-data snapshot); credential-bundle
completeness; the host-level EIO objects that make Kopia's own `blob stats`
fail on `elements-local` (DBM's exception-backed direct scan covers usage); and
the broader "Am I safe?" protection-console redesign. The control-plane lease
should also make its holder die with its parent, and `vrooli scenario restart`
still hits the build-capacity residency-fence timeout. The 6 private-data
targets other than `operator/private` still have no independent copy, so their
Elements writes remain correctly blocked until a second device or off-host
destination exists.

**Deployment friction (control plane, not DBM):** on 2026-11-18 a normal
`vrooli scenario restart` and the setup phase of `vrooli scenario start` failed
at `release component api build capacity lease: read claim residency transition
fence: context deadline exceeded`. The artifact was built and fresh; a
stop/start with `--best-effort` after setup freshness succeeded. The owner is
the control-plane capacity/lifecycle lease path, not this scenario.

### 2026-11-18 — host-volume lease can orphan its holder and block every write

**Symptom:** an Elements write run froze in `capturing` with a stopped
heartbeat and no operator-visible failure. `lslocks` showed
`~/.vrooli/state/locks/artifact-host-volume-operation-*.lock` held forever by a
`vrooli host volume lease` process whose parent was gone. Every later lease
waited on that flock, so no DBM write could start.

**Root cause:** DBM runs the `vrooli` entry point, which may spawn the workspace
build binary as a child. DBM killed only the process it started (the shim), so
the child that actually held the exclusive flock survived. The lease handshake
was also unbounded, so a hung control plane froze the run indefinitely instead
of failing it.

**Resolution (DBM side):** the lease command runs in its own process group and
release/cancel kills the whole group (`lease_process_unix.go`); the readiness
handshake is bounded at 30s (`DefaultLeaseReadyTimeout`). A lease failure now
surfaces its bounded cause instead of "destination readiness changed".

**Still open (control plane):** the holder should also die with its parent on an
abnormal DBM exit (for example `PR_SET_PDEATHSIG` in the lease action) and the
lease action should not fork a child that outlives the process DBM supervises.
Until then, a killed or crashed DBM can leave one orphan holding the flock;
`lslocks` plus killing that PID clears it.

### 2026-11-18 — nested destination bundles were reported as a collision

**Symptom:** every write to `free-disk-root` and `protected-private-data` failed
with `repository_invalid` even though reads and repository probes succeeded.
The bundles live under the `elements-local` bundle root, so the locator-collision
check treated the ancestor relationship as a conflict.

**Root cause:** the collision check compared *bundle roots* with containment
(`diskPathsOverlap`) instead of comparing repository paths and exact bundle
identity.

**Resolution:** compare repository paths for overlap and bundle roots for
equality only; nested bundles with distinct repositories are legitimate. A
regression test covers both the nested case and the shared-bundle case.

### 2026-11-18 — stale backup exception blocked all Elements usage

**Symptom:** `destinations usage` for `elements-local` always failed with
`1 unreadable repository entry is not covered by an approved exception (first:
s/084/...)`, even though the exception file listed that exact path.

**Root cause:** the approved temp-file path had since become a readable empty
directory. `repohealth.Evaluate` classified any shape change as `Blocking`, so a
stale approval gated every usage read forever.

**Resolution:** a shape change to a readable directory is reported as `Stale`
(not a blocker) because the scan traverses and measures it; a link or special
file still blocks. Elements usage now returns 397,583,865,403 bytes. Revoking a
stale exception still has no CLI; the approval remains in the store harmlessly.

### 2026-11-18 — protection wording, drive setup, and self-heal policy (round 3)

**Protection.** Coverage now counts distinct storage devices: two or more
devices with a successful copy is protected, one is a single-copy warning, and
none is the issue. The live report now shows 21 targets protected by two
devices (the baseline-safety + free-disk-root pairs) and 197 single-copy
warnings. A second local drive counts; off-host placement remains a stronger,
separate level.

**Drive behaviour.** `packages/driveprofile` is the one table for check/repair
tool, state-signal availability, suitability, and the operator remedy. The
control plane's `internal/volumeremediation` and DBM readiness both read it.
Creation accepts a profiled filesystem whose driver exposes no state (NTFS via
ntfs-3g, exFAT); the write gate still decides admission. `destinations create`
defaults to filesystem, so a drive is:
`destinations create --name second-drive --location /media/<drive>/vrooli-backups`.

**Coexistence safety.** DBM may share a drive folder with unrelated data and
never deletes foreign files. It refuses only a managed name occupied by the
wrong type. The anomalous `s/084/...f.tmp...` path on the Elements repository
contains a real 214-byte blob, so it was left untouched; it is classified
`Stale` and no longer blocks usage.

**Self-heal.** The failure worker records a policy exception for EIO
`leftover_temp`/session-marker entries and retries without asking
(`DBM_AUTO_ACCEPT_LEFTOVER_TEMP`, on by default in production, off in tests).
Snapshot data, manifests, and ambiguous entries still reach the verified
decision flow. Policy-accepted leftovers do not generate daily notices.

### 2026-11-18 — write admission by evidence, bulk activation, repair ceiling (round 4)

**Over-strict write gate removed.** Preflight blocked a destination whose
native filesystem state was unobservable, even when it was mounted read/write
and a bounded write/read/fsync probe passed. That is why six single-copy
private-data targets kept failing with `destination filesystem readiness is
unknown` while the drive was demonstrably writable. Copy count is a drive-loss
risk, not a writability fact, and blocking a write leaves the data with fewer
copies. The verdict is now: dirty/needs-check, read-only, write-protected, or a
failed probe blocks; unobservable state with a passing probe is a warning.
`AllowDegraded`, `DegradedDestinations`, the runs-side device-key rule, and the
write-gate boolean were deleted.

**One-step drive activation.** `plans add-destination --destination <id>`
targets every enabled plan when `--plans` is omitted (preview by default,
`--apply` to write, idempotent). `destinations create` prints that exact
command. A new drive is therefore two short commands, not an enumeration of
plan ids.

**Protection on the wire.** Coverage exposes `protection_level` and
`protection_device_count` per target plus protected/single-copy/unprotected/
identity-unknown counts. The CLI and UI render those instead of re-deriving
protection from destination counts, which used to mis-bucket a target whose
second destination had failed but whose first copy was good.
`physical_independence_proven` is deprecated and means off-host only.

**Honest repair ceiling.** A diagnosis carries the drive profile's
`RepairLimitation` and `RepairRemedy`. When the platform's repair cannot resolve
the diagnosed cause (for example `ntfsfix` against damaged NTFS metadata), the
question no longer recommends repair again; it recommends a fresh backup set and
names the real remedy. A read-only mount still recommends repair, because a
dirty-flag reset does fix that.

**Per-platform profile.** `packages/driveprofile` now carries per-platform
suitability and notes: NTFS is recommended on Windows (chkdsk), APFS on macOS,
ext4 is Linux-only. Readiness reads them for `runtime.GOOS`.

**Deliberately not implemented:**
- **Standing grant for destructive repair.** It would add a new unattended
  destructive authority, and on this host the only NTFS repair is a dirty-flag
  reset that cannot fix the actual fault. The correct behavior is the honest
  ceiling above, not more automation of an ineffective repair. Revisit only if a
  drive whose repair is a true metadata repair is in use and the operator asks
  for unattended repair.
- **Inferred off-host level.** Backend names and URLs are not endpoint identity.
  Claiming off-host without a real endpoint-identity source would be false, so
  the field stays false and the docs say why. Add it only with that source.

**Live proof (2026-11-18):**
- `protected-private-data` (6 targets, Elements NTFS) run
  `07bcfa91-41af-44e1-bbbc-4b299d4ec134` completed 6/6 with zero incidents and
  zero warnings. Those targets previously failed at preflight with
  `destination filesystem readiness is unknown`.
- Drill `25188dc3-3ace-4f51-8b7c-0f8ea182c34e` reached `DRILL_STATUS_VERIFIED`
  from snapshot `5321e797f5330cb8fe20de7536f9f4b2`, written by that run — a new
  Elements snapshot is recoverable, not just listable.
- `free-disk-root-migration` run `6c7dfd6d-4561-4341-aa1e-a15d63ac7132`
  completed 17/17 with retention applied and no warnings. The previous run
  (`31b6d199`) had warned `destination readiness or identity changed before
  retention; existing snapshots were kept` for all 17 outcomes — the
  write-accepted/retention-refused split is gone.
- Health posture dropped from 13 to 7 issues; the remaining ones are unrelated
  stale last-run failures that clear as their plans next run.
- `plans add-destination --destination <id>` preview selected 16 of 17 enabled
  plans without mutating anything, and every plan reports
  `off-host-placement=unproven` (correct for a single host).

### 2026-11-18 — Elements root cause: one interrupted write burst, never repaired

**Evidence.** The thirteen leftover temp entries were written in one ~10-second
window: `2026-11-10 05:04:41`–`05:04:50`. Seven are readable (sizes ~21 MB, one
214 B, one 4.7 KB); six return `EIO` on `lstat`. All are `s/...f.tmp.<random>`
session markers, i.e. unfinished uploads, not committed blobs. The repository
still lists 191 snapshots and a drill from a new snapshot verifies. SMART is
clean, so this is metadata, not media.

**Mechanism.** An interrupted write burst left inconsistent NTFS MFT records.
Linux's only NTFS "repair" (`ntfsfix -d`) resets the journal and schedules a
Windows chkdsk that never runs; it cannot repair an MFT record. DBM therefore ran
a non-repair and reported the drive as repaired, while the records stayed broken.
This is a product defect, not a recurring hardware fault: the damage happened
once and was never fixed.

**What changed.** Repair is no longer offered when the platform's tool cannot
resolve the cause; `applyRepair` refuses the legacy path; creating a new NTFS
destination requires `--allow-unrepairable`; readiness reports a
`filesystem_repair` check; `destinations repair-options` prints the exact
migration; `destinations safe-eject` flushes and unmounts before a copy/unplug.

**Resolution path (this machine, no VM):** `repair-options --staging <second
drive>` → `rsync` the bundle to staging while mounted → `umount` → `mkfs.ext4`
the Elements partition → mount → `rsync` back to the same bundle path →
`bind-recovery-identity` with the new UUID → readiness and a drill. The
destination id and plan bindings survive because the bundle path is unchanged.

**Space requirement (2026-11-19):** the whole-volume safety copy needs
**552.2 GiB free on a different device** at the default 17% headroom plus 5 GiB
(469.0 GiB used). `--headroom-percent 10` lowers that to **520.9 GiB**; the
minimum accepted is 5%. `repair-drive.sh --auto` prefers staging when a target
is given; without one it runs the read-only `--in-place-check` probe and prints
either the exact in-place steps or the blocking reason. The root NVMe has 241.5 GiB free, and the Elements drive
cannot stage itself, so the repair waits for the second drive. A 640 GB drive is
comfortable; a 600 GB drive works at 10%; a 512 GB drive is too tight only if
the copy really is 469 GiB. Apparent-size measurement is blocked by the six EIO
entries, but they total ~126 MB, so `df used` is the copy size. The volume root also holds two WD installer files outside the DBM
bundle, so a bundle-only copy would have lost them; the script copies and
restores the whole mount.

`scenarios/data-backup-manager/.private/scripts/repair-drive.sh` implements that sequence
as one resumable, safety-gated command. It previews by default, requires one
typed confirmation, verifies the copy by path/size and (by default) checksum
before formatting, refuses the system disk and same-disk staging, re-identifies
the source by serial immediately before `mkfs`, preserves the bundle owner for
the API user, and keeps the staging copy afterwards. `--self-test` exercises
copy, verification, restore, corruption detection, the rsync EIO wording, and
the preview text on a synthetic repository.

**Agent readiness (2026-11-19):** the usage role is rewritten and the improve
role is authored; both are declared in `.vrooli/service.json`. The repair workflow
has no governed program yet because it is rare and destructive; the improve board
records the missing `setpoint-read` program, golden corpus, and usage learning
scope as setup gaps rather than passing rows. The global `safe-operator-script`
skill owns the script contract (dry run, confirmation, identity re-read,
verification, resume, self-test).

**Still open:**
- The conversion itself is operator-run: DBM never formats a drive, so the
  one destructive step (`mkfs.ext4`) remains an explicit command.
- The Elements drive's NTFS MFT-record damage needs Windows `chkdsk /f` or a
  migration to a filesystem this host can repair (ext4). Linux cannot fix it.
- The six private-data targets other than `operator/private` still have no
  independent copy until the second drive is added; that is now a coverage
  warning for the copies that exist, not a write block.
- Test Genie's `experience` phase returns `not_evaluated` because its runtime
  render needs BAS, which the provider reports unavailable. The same result
  reproduces on `plan-manager`, so it is an environment/provider condition, not
  a DBM regression; static UI checks still ran (25 warnings, 2 infos).
- Pre-existing UI template-conformance debt remains (19 components outside
  `ui/src/components|layout`; 9 of 24 slot directories missing; a few
  hand-rolled inputs). It is structural, not backup-safety related.

## Work ladder

- Rung: W3
- Evidence: Configured destination readiness reports `destination_present` and
  `mounted_read_write`, while filesystem health is unknown. Destination usage
  refuses the registered drive because identity, writable state, or filesystem
  health is not proven ready. Before repair, plan preflight returned
  `unavailable: unexpected EOF`; API logs showed synchronous `PreflightRun`
  calls taking 33–67 seconds against api-core's 30-second default
  `WriteTimeout`. After the timeout and error-classification fixes, preflight
  completed in 52 seconds and returned `destination_inaccessible` with the
  correct readiness explanation.
- Blocker: A fresh scheduled backup remains blocked until destination
  filesystem readiness is proven. The fix makes the preflight response
  reliable and accurate; it does not bypass that safety gate.
- Measured: 2026-11-16

### 2026-11-19 — in-place probe result and remount fix

`repair-drive.sh --dry-run` ran the read-only `ntfsresize --info --no-action`
probe. It reported:

```
ERROR: Filesystem check failed!
ERROR: 403 clusters are referenced multiple times.
```

The NTFS metadata is cross-linked. `ntfsresize` correctly refuses, so the
in-place conversion is closed for this drive. Linux cannot repair this; a
Windows `chkdsk /f` is the only in-place repair candidate, and the ext4
conversion avoids needing it. The staging path is required.

The probe also exposed a remount bug: because it ran as root, udisks mounted
the volume at `/media/root/Elements` instead of the recorded
`/media/matthalloran8/Elements`. The probe now unmounts any wrong-path mount
and mounts explicitly at the recorded path with the original options;
`repair-drive.sh --remount` recovers a drive already mounted at the wrong path.
The duplicate `staging FAIL` finding was also removed.

### 2026-11-19 — direct-sudo recovery fixes

A direct `sudo repair-drive.sh --remount` failed with
`data-backup-manager is not on PATH`. `sudo` resets `PATH` and `HOME`, so the
scenario CLI and its state disappeared. The script now resolves the invoking
user from `SUDO_USER`/`SUDO_UID`/`SUDO_GID`, prepends that user's
`~/.vrooli/bin` to `PATH`, sets `HOME`, and uses absolute CLI paths. The
`--remount` path also recreates the recorded mountpoint and re-owns its parent
(`/media/<user>`) to the invoking user, because udisks removes the directory on
unmount and a root-run probe can leave the parent `root:root 710`, which the DBM
API cannot traverse.

### 2026-11-19 — the drive was idle because its plans were off, and still cannot take the bulk plan

`elements-local` looked healthy from the cheap probes: `usage` returned 397.6 GB,
`destinations snapshots` listed 11 snapshots, and readiness passed the directory
and mount checks. Re-enabling `declared-targets` (248 targets) exposed the real
state: kopia `blob stats` fails with
`error reading directory: lstat .../elements-local.kopia/s/f43/64b7506cef44e4fbb3fc66ba7f2d9-sf40366ce03934d79144.f.tmp.8c9d63ca28444838: input/output error`
— the same shard from the November incident. The repository cannot be fully
enumerated, so every write is refused as `repository_invalid` and the plan is
off again. Listing snapshots reads manifests and does not touch that directory,
which is why the earlier probes looked clean.

Two code bugs surfaced while proving this:

1. `sysmounts.platformVolumeIdentity` asked `lsblk` for the whole disk without
   `-d`. `lsblk /dev/sda` prints the disk row and then its partition rows, and a
   partition row carries an empty MODEL/SERIAL. The last row overwrote the
   disk's anchors, so `elements-local` (the only destination that recorded a
   serial) failed the write gate with "destination path is not served by the
   verified mounted volume" while `free-disk-root` (serial `-`) worked. Fixed
   with `lsblk -d` and a regression test.
2. `RebindMetadata` required a `relative_path` that bundles written before that
   field existed do not carry, and `replaceManagedFile` treated ntfs-3g's
   `chmod ... operation not permitted` as fatal. A legacy bundle at an unchanged
   path is now stamped instead of refused, and a `chmod` EPERM is skipped
   because `os.CreateTemp` already created the file 0600 — the mode can only
   narrow, never widen.

The plan that refreshes `elements-local` therefore stays off until the volume is
converted to ext4. The two small Elements plans (`protected-private-data`,
`free-disk-root-migration`, 23 targets) preflight ready and keep working.

### 2026-11-19 — the drive was idle because its plans were off, not because it was broken (superseded)

`elements-local` was readable the whole time (397.6 GB used, snapshots listed,
`usage` within policy), but the only plans that write to it —
`declared-targets` (248 targets) and `elements-daily-runtime` (14) — were both
`enabled=false` after the November repository incident. The drive's other two
repositories (`protected-private-data`, `free-disk-root`) stayed current because
their plans were on, which made the drive look healthy while the bulk of the
catalog had no copy newer than 2026-08-19. Coverage still read 21 protected /
197 single-copy / 30 unprotected because it counts the copies that exist, not
whether the plan that refreshes them is on.

`repair-drive.sh --states` (and bare invocation) now prints the drive, free
space, protection counts, and each bound plan's on/off state. A switched-off
plan is marked `OFF` only when no enabled plan on that drive covers its targets,
so a redundant disabled plan does not read as a gap. (The first version of this
entry said `declared-targets` was re-enabled as a fix; the next entry records
why that had to be undone.)

### 2026-11-19 — four targets are root-owned files DBM cannot read

DBM runs as the invoking user, so these never back up and stay in health as
failures:

| target | path | mode |
|---|---|---|
| `adguard-home/work` | `~/.local/share/vrooli/resources/adguard-home/data` | `root:root 0700` |
| `adguard-home/config` | `~/.config/vrooli/resources/adguard-home/AdGuardHome.yaml` | `root:root 0600` |
| `ollama/credentials` | `~/.local/share/vrooli/resources/ollama/id_ed25519` | `root:root 0600` |
| `autoheal_recovery_privileges/sudoers` | `/etc/sudoers.d/vrooli-autoheal` | `root:root 0440` |
| `onboarding_apply_privileges/sudoers` | `/etc/sudoers.d/vrooli-onboarding-apply` | `root:root 0440` |

The first three live inside user-owned trees, so the owning resource wrote them
as root and a `chown` to the invoking user restores readability. The two
`sudoers.d` files cannot be user-owned; backing them up needs a control-plane
privileged read, which does not exist yet. Do not silently exclude them: they
are hand-authored privilege policy, and a quiet exclusion would report complete
coverage for state that has no copy.

### 2026-11-19 — rebuild-in-place: three failures the dry run could not show

1. **The mount was busy.** `umount` failed because Nautilus held a directory
   handle on `/media/matthalloran8/Elements` (`fuser -vm` named it). The script
   now prints the holders and `nautilus -q` when an unmount fails, and the whole
   rebuild is resumable, so recovering is a re-run rather than a restart.
2. **Deleting records before formatting removes the catalog row the next run
   needs to select the drive.** The delete has to happen while the old volume
   identity still matches, because the delete gate re-observes the volume and
   refuses once the UUID changes. The run therefore writes
   `$STATE_ROOT/rebuild-resume.env` plus the captured destination and plan
   lists, reconstructs them from the capture when the env file is absent, and
   tracks `rebuild-removed`/`rebuild-formatted` so a failure after `mkfs` resumes
   at the destination rebuild instead of trying to format again.
3. **`plans update` refuses on incomplete default coverage.** Discovery had found
   `docker/graph_root` and `rasdaemon/event_database`, neither registered, so
   every re-point failed with `failed_precondition: incomplete default coverage`.
   The rebuild passes `--allow-incomplete-coverage`; a plain manual re-point
   needs the same flag.

### 2026-11-19 — a root-run CLI rebuild poisons the user's resource CLIs

Running `data-backup-manager` or `vrooli` under `sudo` let the shared CLI
installer rebuild `resource-kopia` under root's identity. The binary's baked
fingerprint then disagreed with what the user's process computed, so every
repository read failed with:

```
repair destination bundle path metadata: ... error obtaining VCS status: exit status 128
go build failed in /home/matthalloran8/Vrooli/resources/kopia/cli
```

DBM classified those as `repository_invalid`; 77 targets of the 220-target local
plan failed that way while both the local and the Elements repositories were
healthy. Rebuilding `resource-kopia` once as the invoking user cleared it.
`repair-drive.sh` now routes `dbm()` and the scenario start/stop calls through
`as_user` (`runuser -u $SUDO_USER`), so a root-run repair cannot poison the
operator's CLI again. The general rule: a privileged script may touch the
device, but scenario and resource CLIs must run as their owner.

### 2026-11-19 — a root-run DBM is the failure mode behind "repository_invalid"

Running DBM as root has one root cause and four symptoms, and the symptoms all
look like broken backup infrastructure:

1. `~/.vrooli/secrets.enc.json` was rewritten `root:root 0600`, so kopia could
   not read any repository passphrase. Every destination reported
   `repository_invalid` and the local repository looked "unavailable" while its
   167 GB were intact.
2. The shared CLI installer rebuilt `resource-kopia` under root, where git
   refuses the user's work tree ("dubious ownership"), so the rebuild failed
   with `error obtaining VCS status: exit status 128` on every invocation.
3. `dbm destinations create` under root created the new ext4 bundles and the
   kopia registry as root, so the user's process could not traverse them.
4. The lifecycle's own `ui/.vrooli-displaced-dist` was created root-owned, and
   `scenarioBuildIdentity` hashed it into the source identity, so the scenario
   could not start with `open ...: permission denied`.

Fixes: the API refuses to start as root (`refuseRoot`, unit-tested) and the UI
exits when `getuid()===0`; `repair-drive.sh` runs every scenario/resource CLI
through `runuser -u $SUDO_USER` and refuses a rebuild when the running API is
not owned by the invoking user; `scenarioBuildIdentity` skips
`.vrooli-displaced-*` (control plane); ownership was repaired for the drive
bundles, the kopia state/cache, the credential store, and the resource trees.

Operational rule: a privileged script may act on the device, but it must never
start the scenario, its UI, or any resource CLI as root.

### 2026-11-19 — kopia's repo registry is per-user state, not per-repository

Deleting `~/.local/state/vrooli/resources/kopia/registry.json` to escape a
root-owned copy is not recoverable by reconnecting lazily: `destinations usage`
for the local destination calls `repo stats`, which requires a registered name.
Re-register it explicitly:

```
resource-kopia repo connect --name baseline-safety --backend filesystem \
  --path ~/.vrooli/baseline-safety/repositories/baseline-safety.kopia
```

The drive destinations self-heal because their rebind path runs `repo connect`.

### 2026-11-19 — root-only host state is explicit, not excluded

The current unreadable set is six paths, not four: the two `sudoers.d` files,
`/etc/vrooli/local-service-admission.json`, `/var/log/vrooli/log-volume-bounds`,
`/var/lib/rasdaemon/ras-mc_event.db`, and `/var/lib/docker`. Only the last two
are kept out of the plans, because container layers are reacquirable and the
rasdaemon database is a hardware event log owned by another service. The other
four remain registered and in both selector plans on purpose: they are
hand-authored privilege policy and audit evidence, and a silent exclusion would
claim complete coverage for state with no copy. They will keep `/health`
degraded until the control plane gains a supported privileged read for exact
paths. Do not remove them from the plans to make the verdict green.

The per-target cause is now attributed correctly (see the preflight incident
merge fix), so each blocked path names itself instead of naming the first
unreadable target in the run.

### 2026-11-19 — `runs browse` restores the whole snapshot before listing it

**Symptom:** browsing a large snapshot (`research-desk/research-home`, 73 GB)
allocates multi-GB temp trees, runs for minutes, exceeds DBM's 512 KiB/128-entry
response bounds, and can starve the host-volume lease handshake for every other
target. Two browse attempts left about 10 GB in `/tmp/resource-kopia-browse-*`.

**Root cause:** `resource-kopia snapshot browse` runs `kopia snapshot restore`
into a temp directory and then walks it. Kopia has a native `list <object-path>`
that reads the directory object without restoring contents, but the wrapper
does not use it. `--path` does not reduce the restore cost.

**Workaround:** do not browse a large snapshot; use `restores verify` or a
path-scoped audit instead, and remove `/tmp/resource-kopia-browse-*` afterwards.

**Real fix:** replace the restore-backed browse with `kopia list` (parse its
long output, or add a JSON projection in the wrapper), and cap the response by
paging rather than rejecting at 512 KiB.

**Owner:** resource-kopia with Data Backup Manager.

**Refs:** `resources/kopia/cli/internal/snapshot/snapshot.go` `Browse`,
`scenarios/data-backup-manager/api/internal/engine/kopia.go` `BrowseSnapshot`.

### 2026-11-19 — health's first post-restart probe caches cold-start failures

**Symptom:** for up to ten minutes after a DBM restart, `/health` reports
`repository <name> unavailable` and `destination <id> unavailable` for every
destination, while `destinations usage` answers normally.

**Root cause:** the first cached posture probe runs while resource-kopia is
still warming/refreshing. Its failures are stored as the stale-but-servable
result and only replaced when the next background probe finishes.

**Workaround:** wait for the next probe (the log line changes from
`+ N unavailable` to only the real target issues) before acting on the verdict.

**Real fix:** do not publish the first cold-start probe as the cached verdict;
retry failed destination connectivity once within the same probe, or mark the
result "warming" instead of "unavailable".

**Owner:** Data Backup Manager health adapter.

**Refs:** `api/adapters.go` `backupPosture.OverdueOrFailed`,
`api/handlers/health/cached_posture.go`.

### 2026-11-19 — lease-bound destinations need serial target fan-out (fixed)

**Symptom:** a drive plan with four-way target fan-out failed a large share of
targets with `destination_inaccessible`, `acquire host volume operation lease:
did not become ready within 30s; the control plane lease owner did not answer`.

**Root cause:** each snapshot holds the exclusive host-volume lease until the
snapshot ends, so three of four concurrent units were waiters. The 30-second
bound is a handshake bound, not a queue bound, so waiters failed instead of
waiting for the holder to finish.

**Fix:** `runs.targetConcurrencyFor` now forces one target at a time when any
destination has a strong filesystem identity and therefore takes the lease. A
legacy path destination without device identity keeps the configured
parallelism. The handshake bound rose to two minutes so a concurrent verify,
drill, or browse on the same spinning disk cannot starve the control plane's
volume inspection; it remains bounded for a genuinely hung control plane.

**Refs:** `api/internal/runs/service.go`, `api/internal/runs/concurrency_test.go`.

### 2026-11-19 — a 30-minute drill poll bound marks large restores failed (fixed)

**Symptom:** the 73 GB `research-desk/research-home` drill was marked
`DRILL_STATUS_FAILED` with "verified restore did not reach a terminal state
before the drill timeout" while the restore was still running and had reached
about 46 GB. The restore process kept running after the drill record failed.

**Root cause:** `drills.NewService` defaulted `PollTimeout` to 30 minutes. A
large snapshot on this spinning USB drive takes roughly an hour to restore and
hash, so the bound was shorter than the work it was bounding.

**Fix:** the default is now three hours, because the verified restore also does
a 100% byte-verify and a tree checksum after the restore itself. The drill still
fails closed on a genuinely stuck restore. A timed-out drill does not cancel the
linked restore; the restore record remains the authority for whether the bytes
verified.

**Refs:** `api/internal/drills/service.go` `NewService`,
`api/internal/drills/service.go` `execute`.
