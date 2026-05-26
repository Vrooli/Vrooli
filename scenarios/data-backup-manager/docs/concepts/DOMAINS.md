# Domains — Data Backup Manager

This document is the canonical map of product capabilities, bounded
contexts, and ownership for this scenario. Keep it current whenever a
domain is added, renamed, split, merged, or removed.

This scenario delivers dependable, engine-backed backup and verified
restore for Vrooli runtime state. Owning scenarios self-register the
state they own (databases, filesystem trees, vector and cache stores);
this manager snapshots that state on a schedule to one or more
encrypted destinations and proves it can restore. The product
vocabulary is fixed: **Target**, **Destination**, **Plan**, **Run**,
**Restore**, and **Source kind**. Every domain below speaks that
vocabulary.

## Purpose Of This Document

Use this document to answer:

- What product capabilities does this scenario expose?
- Which domain owns each concept, table, proto, endpoint, UI feature,
  CLI command, and test surface?
- Which concepts are shared, deferred, or deliberately not domains?

System-level architecture belongs in [`ARCHITECTURE.md`](ARCHITECTURE.md).
Workflow details belong in [`FLOWS.md`](FLOWS.md). Storage details
belong in [`DATA.md`](DATA.md).

## Domain Inventory

These are the intended product domains for the locked design. Nothing
in the list is built yet; this inventory is the target boundary map
that requirements and code will fill in. The `health` domain is the
template-provided readiness surface and is retained.

| Domain | Purpose | Primary Archetype | Owns Data | Surfaces | Requirements | Source Paths |
|---|---|---|---|---|---|---|
| targets | Self-registered backup sources owned by other scenarios (owner+name keyed). | Registration / entity | Target records, source kind, locator, optional quiesce-hook references. | API, CLI, UI | OT-P0-001 | `api/internal/targets/`, `api/handlers/targets/`, `cli/domains/targets/`, `ui/src/features/targets/`, `packages/proto/schemas/data-backup-manager/v1/targets/` |
| destinations | Backup destinations, each a kopia repository (local filesystem or S3/MinIO). | Configuration / entity | Destination records, backend kind, cap, secret references. | API, CLI, UI | OT-P0-003, OT-P0-007, OT-P0-008 | `api/internal/destinations/`, `api/handlers/destinations/`, `cli/domains/destinations/`, `ui/src/features/destinations/`, `packages/proto/schemas/data-backup-manager/v1/destinations/` |
| plans | Many-to-many bindings of targets to destinations with schedule and retention. | Orchestration / entity | Plan records, plan↔target and plan↔destination membership, schedule, retention policy. | API, CLI, UI | OT-P0-004, OT-P0-005, OT-P1-002 | `api/internal/plans/`, `api/handlers/plans/`, `cli/domains/plans/`, `ui/src/features/plans/`, `packages/proto/schemas/data-backup-manager/v1/plans/` |
| runs | Executions of plans; run history and last-success-per-target. | Workflow / job | Run records, per-target run outcomes, snapshot references. | API, CLI, UI | OT-P0-005, OT-P0-009, OT-P0-010 | `api/internal/runs/`, `api/handlers/runs/`, `cli/domains/runs/`, `ui/src/features/runs/`, `packages/proto/schemas/data-backup-manager/v1/runs/` |
| restores | Restore a target to a location; verify mode test-restores to scratch and checksums. | Workflow / job | Restore records, verify outcomes, last-verified-per-target. | API, CLI, UI | OT-P0-006, OT-P1-004 | `api/internal/restores/`, `api/handlers/restores/`, `cli/domains/restores/`, `ui/src/features/restores/`, `packages/proto/schemas/data-backup-manager/v1/restores/` |
| sources | Source-kind handlers that turn a registered source into snapshottable bytes. | Strategy / adapter | No durable product data; owns the per-kind capture/restore behavior. | API (internal), CLI (diagnostics) | OT-P0-002, OT-P1-001 | `api/internal/sources/`, `packages/proto/schemas/data-backup-manager/v1/sources/` |
| health | Report runtime readiness, dependency reachability, and overdue/failed backup posture. | Reporting / query | No product data. | API, UI | OT-P0-010 | `api/handlers/health/`, `ui/src/features/health/`, `packages/proto/schemas/data-backup-manager/v1/health/` |

## Domain Details

### targets

- Purpose: hold the catalog of backup sources, each registered
  idempotently by the scenario that owns the underlying state. The key
  is `owner + name`; re-registration on boot is an upsert, mirroring
  agent-manager's `EnsureProfile` pattern.
- Primary archetype: registration / entity.
- Secondary traits: idempotent upsert, owner-scoped uniqueness.
- Owns: target records, the source kind, the locator (e.g. a
  storage-root-relative path, a database name, a Redis key prefix, a
  Qdrant collection, an object-storage bucket/prefix), and optional
  references to pre/post quiesce hooks (P1).
- Does not own: where artifacts land (destinations), when capture runs
  (plans/runs), or how bytes are produced (sources). Does not own
  source secrets — those live in vault.
- Catalog property: the targets table is a cache and a run-history
  anchor, not the single source of truth. Because scenarios
  re-register on boot, the catalog is reconstructable; a lost SQLite
  cache costs run history, not the registration model.
- API: `api/internal/targets/`, `api/handlers/targets/`.
- CLI: `cli/domains/targets/` — used by scenarios to self-register at
  lifecycle and by operators to inspect the catalog.
- UI: `ui/src/features/targets/` (catalog view; registration is
  primarily a programmatic/CLI surface).
- Storage: domain-owned SQLite schema in
  `api/internal/targets/schema.sql`.
- Requirements: OT-P0-001 (self-registration), with OT-P1-001
  (quiesce hooks) extending it.
- Related docs: [`FLOWS.md`](FLOWS.md), [`DATA.md`](DATA.md),
  [`../internal/SEAMS.md`](../internal/SEAMS.md).

### destinations

- Purpose: model each backup destination as a kopia repository and
  enforce the safety rules around it.
- Primary archetype: configuration / entity.
- Secondary traits: encryption-by-default, per-destination storage cap.
- Owns: destination records, the backend kind (`filesystem` or
  `s3`/MinIO), the configurable storage cap, and references to the
  vault-held passphrase and access keys. Tracks usage versus cap from
  kopia repository stats.
- Does not own: the secrets themselves (the `vault` resource holds
  them; destinations hold only references), repository internals (the
  `kopia` resource owns dedup/encryption/compression), or scheduling.
- Invariants: every destination is encrypted by default; a destination
  MUST NOT point under the storage root it would protect
  (separate-root rule); storage limits default to **alert + block**
  (alert and refuse new writes — never silent eviction; eviction
  happens only through explicit retention on a plan).
- API: `api/internal/destinations/`, `api/handlers/destinations/`.
- CLI: `cli/domains/destinations/`.
- UI: `ui/src/features/destinations/` — usage-versus-cap is the visual
  centerpiece.
- Storage: domain-owned SQLite schema in
  `api/internal/destinations/schema.sql`; backup artifacts live in the
  kopia repository, never under the scenario source tree.
- Requirements: OT-P0-003 (multiple destinations), OT-P0-007
  (encryption on by default), OT-P0-008 (storage limits); OT-P1-003
  (destination dry-run) extends it.
- Related docs: [`DATA.md`](DATA.md),
  [`INTEGRATIONS.md`](INTEGRATIONS.md).

### plans

- Purpose: bind targets to destinations and decide when and how long
  snapshots are kept.
- Primary archetype: orchestration / entity.
- Secondary traits: many-to-many membership, schedule, retention.
- Owns: plan records, the plan↔target and plan↔destination membership
  tables (a target may belong to multiple plans — e.g. daily-to-local
  and weekly-to-offsite), the schedule, and the per-plan retention
  policy.
- Does not own: the act of running (that is a run), or the engine-side
  retention mechanics (kopia policy, set through the resource).
- API: `api/internal/plans/`, `api/handlers/plans/`.
- CLI: `cli/domains/plans/`.
- UI: `ui/src/features/plans/`.
- Storage: domain-owned SQLite schema in
  `api/internal/plans/schema.sql`.
- Requirements: OT-P0-004 (backup plans), OT-P0-005 (scheduled
  execution); OT-P1-002 (GFS retention) extends it.
- Related docs: [`FLOWS.md`](FLOWS.md).

### runs

- Purpose: record each execution of a plan and the per-target outcome,
  and answer "when did this target last succeed?".
- Primary archetype: workflow / job.
- Secondary traits: in-process scheduler trigger and on-demand trigger,
  event emission for platform monitoring.
- Owns: run records (plan reference, trigger source, start/finish,
  status), per-target run outcomes, and references to the kopia
  snapshots produced. Surfaces last-success-per-target for the catalog
  and health views.
- Does not own: snapshot bytes (kopia), restore verification (restores
  domain), or the source-capture mechanics (sources domain).
- API: `api/internal/runs/`, `api/handlers/runs/`.
- CLI: `cli/domains/runs/`.
- UI: `ui/src/features/runs/` — run history and per-target status.
- Storage: domain-owned SQLite schema in
  `api/internal/runs/schema.sql`.
- Requirements: OT-P0-005 (scheduled + on-demand), OT-P0-009 (catalog
  & run history), OT-P0-010 (health & observability).
- Related docs: [`FLOWS.md`](FLOWS.md),
  [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md).

### restores

- Purpose: restore a target to a chosen location, and prove a backup
  is restorable before any committed runtime data is removed from git.
- Primary archetype: workflow / job.
- Secondary traits: verify mode (test-restore to scratch + checksum),
  last-verified tracking.
- Owns: restore records, verify outcomes, and last-verified-per-target.
  Verify mode is the gate: a target is only safe to drop from git once
  a verified restore has succeeded.
- Does not own: the snapshot store (kopia) or the bytes themselves.
- API: `api/internal/restores/`, `api/handlers/restores/`.
- CLI: `cli/domains/restores/` — guided restore and verify.
- UI: `ui/src/features/restores/` — guided restore/verify flow.
- Storage: domain-owned SQLite schema in
  `api/internal/restores/schema.sql`.
- Requirements: OT-P0-006 (verified restore); OT-P1-004 (restore
  granularity) and OT-P2-004 (automated restore drills) extend it.
- Related docs: [`FLOWS.md`](FLOWS.md).

### sources

- Purpose: turn a registered target's source into a consistent,
  snapshottable artifact (and back, on restore), one handler per source
  kind.
- Primary archetype: strategy / adapter.
- Secondary traits: best-effort versus point-in-time semantics differ
  per kind.
- Owns: the six v1 source-kind handlers and their capture/restore
  behavior:
  - **filesystem** — tar/glob of a storage-root-relative path (relative
    for portability).
  - **sqlite** — `VACUUM INTO` a consistent copy.
  - **postgres** — `pg_dump` via the `postgres` resource CLI.
  - **redis** — prefix `SCAN` + `DUMP`; explicitly **best-effort, not a
    transactional point-in-time** snapshot.
  - **qdrant** — the Qdrant snapshot API.
  - **object-storage** — S3/MinIO mirror, sharing the SDK with the
    object-storage destination backend.
- Does not own: durable product records (it is stateless strategy
  code), secrets (read from vault at capture time), or the snapshot
  engine.
- API: `api/internal/sources/` (internal — invoked by runs/restores).
- CLI: diagnostic/inspection only; not a primary operator surface.
- UI: none directly; surfaced through target/run views.
- Storage: none of its own.
- Requirements: OT-P0-002 (six source kinds); OT-P1-001 (quiesce
  hooks) extends the capture path.
- Related docs: [`FLOWS.md`](FLOWS.md),
  [`INTEGRATIONS.md`](INTEGRATIONS.md).

### health

- Purpose: expose API/database readiness, show the UI can read live
  backend state, and flag overdue or failed backups for platform
  monitoring.
- Primary archetype: reporting / query.
- Secondary traits: operational health, backup-posture summary.
- Owns: health response construction, dependency status mapping, and
  the overdue/failed-backup rollup derived from runs.
- Does not own: product data, business rules, or scenario-specific
  domain behavior beyond the readiness rollup.
- API: `api/handlers/health/`.
- CLI: built-in `status` command is provided through cli-core.
- UI: `ui/src/features/health/HealthCard.tsx`.
- Storage: none; probes configured database reachability and reads
  run-derived posture.
- Requirements: OT-P0-010 (health & observability).
- Related docs: [`../reference/api-endpoints.md`](../reference/api-endpoints.md),
  [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md).

## Shared Concepts

| Concept | Meaning | Owner |
|---|---|---|
| Domain | Product capability boundary that should be easy to find, test, and delete. | `DOMAINS.md` defines the map; code owns implementation. |
| Surface | API, UI, CLI, or contract layer exposing the same product capability. | `ARCHITECTURE.md`. |
| Seam | Test-substitutable boundary wired once in production. | `../internal/SEAMS.md`. |
| Requirement | Implementation-facing measurement tied back to the PRD. | `requirements/`. |

## Deferred Domains

Add future or intentionally deferred capabilities here only when they
are real enough to affect architecture or requirements.

| Candidate Domain | Why Deferred | Revisit Trigger |
|---|---|---|
| replication | Cross-destination replication for offsite tiers (OT-P2-002). | Not needed for v1; a destination handles one repository. | When an offsite-tier requirement lands and a plan must copy snapshots between destinations. |
| analytics | Usage and growth trends per destination/target (OT-P2-003). | Storage usage is surfaced today via destination stats; trend history is future scope. | When operators need historical growth charts beyond current usage-vs-cap. |
| notifications | Stale/failed-backup alerting through platform monitoring (OT-P1-006). | Health flags posture in v1; routed alerting is post-launch. | When alerting integration becomes a requirement; likely composes a platform notification surface rather than a local domain. |

## Non-Domains

These are important but should not become product domains:

- `api/internal/server/` — HTTP composition substrate.
- `api/internal/module/` — shared module descriptor type.
- `api/internal/modules/` — thin registry for boot/codegen.
- `api/internal/database/` — cross-cutting database infrastructure.
- `api/internal/testutil/` — cross-domain test harnesses.
- `ui/src/components/` — shared presentation primitives.
- `ui/src/test-utils/` — cross-feature testing support.

If one of these starts using product vocabulary, split the product
piece into an owning domain instead of growing infrastructure.

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system shape and extension rules
- [`FLOWS.md`](FLOWS.md) — workflows and state transitions
- [`DATA.md`](DATA.md) — data ownership and storage
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — dependency contracts
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — boundary registry
- [`../internal/TESTING.md`](../internal/TESTING.md) — test strategy
