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
source of truth today; `flow-verifier verify check` passes because there are no
`flow.json` files to check.

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

### 2026-05-26 — vault secret-read CLI is a stub; snapshot-browse assumes a kopia verb

**Symptom:** (1) `INTEGRATIONS.md` assumes a `resource-vault secret get` CLI,
but that surface is a stub. (2) `RunsService.BrowseSnapshot` shells out to
`resource-kopia snapshot browse`, a command the kopia resource does not yet
expose.

**Root cause:** Both are substrate gaps surfaced during implementation. Source
credentials are sidestepped today by having each source resource CLI
self-source its own credentials (so no direct vault read is needed). Snapshot
browse has no resource-kopia equivalent yet.

**Workaround:** Source capture works without a raw vault read. Snapshot browse
returns whatever `resource-kopia snapshot browse` yields; the catalog
unit tests use the fake engine, so default tests are unaffected.

**Real fix:** (1) Only add `resource-vault secret get` if a concrete source
genuinely needs a raw secret the manager must pass. (2) Add a
`resource-kopia snapshot browse|ls --json` command (fix-substrate) and confirm
the engine parse in `internal/engine/kopia.go::BrowseSnapshot`.

**Owner:** unassigned. **Refs:** `internal/engine/kopia.go`;
`INTEGRATIONS.md`; the kopia resource.

### 2026-05-26 — ListTargetStatus owner filter not wired

**Symptom:** `RunsService.ListTargetStatus` accepts an `owner` field but ignores
it — runs keys purely on target id and has no target→owner mapping.

**Root cause:** The runs domain does not own targets; resolving owners would
couple it to the targets domain.

**Workaround:** v1 returns all targets seen in run history. The health rollup
applies its own thresholds without owner filtering.

**Real fix:** Resolve owner→target-ids via a targets adapter at the handler
edge and pass the id set to the repository (which already filters by id).

**Owner:** unassigned. **Refs:** `handlers/runs/connect_handler.go::ListTargetStatus`;
`internal/runs/repository.go::TargetStatuses`.

### 2026-05-26 — UI is a deliberate follow-up (DBM-UI-001 not implemented)

**Symptom:** The scenario ships API + CLI; `ui/src/features/*` is still the
template shape. DBM-UI-001 is unmet.

**Root cause:** The implementation pass was scoped to API + CLI. The proto
contracts authored here are exactly what the UI will consume.

**Real fix:** A dedicated UI plan: destinations (usage-vs-cap), plans, run
history, guided restore/verify, against the generated Connect-Web clients.

**Owner:** unassigned. **Refs:** `UI-ARCHITECTURE.md`; requirements module 08.

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
