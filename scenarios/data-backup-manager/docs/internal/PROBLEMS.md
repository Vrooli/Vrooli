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
discovered durable targets (sensitive credential/token targets require explicit
`--include-sensitive`); `plans create/update` are guarded
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
The server `WriteTimeout: 6h` in `main.go` is **kept** — but its justification
changed: runs no longer need it, but `restores verify/restore` are still
synchronous and long-running (see "Still open" below), and the api-core default
30s `WriteTimeout` severs them mid-operation ("unexpected EOF"). It drops to the
default only once restores also become async. Refs:
`api/internal/runs/{service,executor,repository,sqlite,migrate}.go`,
`api/main.go` (execCtx + Reconcile + Cleanup drain),
`cli/domains/runs/handlers.go`.

**Still open (restores):** `restores.{VerifyTarget,RestoreTarget}` remain
synchronous and long-running. They are single, bounded, operator-initiated
operations, so they keep the unlimited-timeout Connect client on the CLI side
(`cli/domains/restores/handlers.go`) rather than the async treatment. The
background-job seam is general enough that they *could* opt in later, but that is
deliberately out of scope. **Resume** of an interrupted run (vs fail) is also
deferred — v1 fails orphans cleanly.

**Owner:** restores async / run-resume — unassigned. **Refs:**
`api/internal/restores/service.go`, `cli/domains/restores/handlers.go`.

### 2026-06-03 — `next_scheduled_at` not yet on the freshness view (Phase 6 partial)

**Symptom:** `runs status` / `TargetStatus` / `/health` report per-target
`overdue` and `last_success_age_seconds`, but not the *next* scheduled backup
time per target.

**Root cause:** "Next scheduled" for a target = the minimum, over the plans that
include it, of (plan's last fire + its schedule interval). That requires three
inputs the runs status rollup does not have: the target→plans membership, each
plan's schedule, and the **scheduler's in-memory `lastFire`** map
(`internal/scheduler/scheduler.go`), which is not currently exposed through any
seam.

**Workaround:** None needed — overdue + age already answer "is this target
behind?", the primary cadence question. Next-scheduled is additive.

**Real fix:** Add a scheduler→status seam that surfaces per-plan `lastFire` +
schedule, join it through target membership in `ListTargetStatus`, and populate
a `next_scheduled_at` field (additive on `TargetStatus`). Deferred from Phase 6
of the perf/observability plan.

**Owner:** unassigned. **Refs:** `internal/scheduler/scheduler.go` (`lastFire`),
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

**Owner:** unassigned (resource-kopia server-mode is the enabling work). **Refs:**
`api/internal/engine/kopia.go::SnapshotCreate`, `api/internal/runs/service.go`
(fan-out), Phase-4 section of
`data-backup-manager-backup-performance-observability-async-execution`.

### 2026-06-03 — `destinations usage` fails: resource-kopia `repo stats --json`

**Symptom:** `data-backup-manager destinations usage <id>` errors with
`kopia … content stats --json: unknown long flag '--json'`.

**Root cause:** External — `resource-kopia repo stats --json` shells out to
`kopia content stats --json`, but the installed kopia build does not accept
`--json` on `content stats`. Not a DBM defect; DBM only consumes the wrapper.

**Workaround:** Usage reporting is informational and unused by the backup/verify
path; ignore the error. Capacity caps still work (they use a different path).

**Real fix:** Fix `resource-kopia repo stats` to not pass an unsupported flag
(or parse non-JSON output). Filed against the kopia resource via report-bug.

**Owner:** resource-kopia (filed). **Refs:** `resources/kopia` CLI; surfaced at
`api/internal/engine/kopia.go` (RepoStats).

## Architecture Drift

Use this section for deferred findings from `screaming-architecture-audit`.
Do not create a standalone architecture-audit report unless the work is
a migration handoff with a planned retirement path back into
`ARCHITECTURE.md`, `SEAMS.md`, or this file.

| Area | Drift | Maturity Impact | Real Fix |
|---|---|---|---|
| _None yet._ |  |  |  |

## Cross-references

- [`PROGRESS.md`](PROGRESS.md) — lifecycle log (forward-looking)
- [`SEAMS.md`](SEAMS.md) — boundary registry (load-bearing for tests)
- [`TESTING.md`](TESTING.md) — test patterns
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — generic-template issues
