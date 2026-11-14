# Runbook — Nutrition Planner

This document records operator procedures for running, diagnosing,
recovering, and maintaining the scenario.

The working product name is **Nooch** (formerly Daily; decision D-042). Scenario id is `nutrition-planner`.
All state lives in one local SQLite database plus the build artifacts; the
database is **user-owned, private, and non-regenerable**.

## Purpose Of This Document

Use this document to answer:

- How do I start, stop, and inspect the scenario?
- What checks should I run during an incident?
- How do I back up or restore state?
- Where should operational issues be recorded?

## Start / Stop / Status

Use the scenario lifecycle. Do **not** start the API or UI binaries
directly — the lifecycle owns process naming, ports, health checks, logs,
and recovery.

```bash
# From the scenario directory
make setup        # build/install; wraps `vrooli scenario setup`
make start        # start API + UI; wraps `vrooli scenario start`
make status       # lifecycle status and health
make logs         # tail lifecycle-managed API/UI logs
make stop         # stop the scenario
make test         # run the scenario test suite

# Equivalent control-plane surface
vrooli scenario start nutrition-planner
vrooli scenario status nutrition-planner
vrooli scenario logs nutrition-planner
vrooli scenario stop nutrition-planner
```

Health endpoints: API `/health` and UI `/health`, as declared in
`.vrooli/service.json`. Ports come from the lifecycle (read them from
`vrooli scenario status nutrition-planner`); never hard-code them. The CLI
(`nutrition-planner status`) is a typed mirror of the API; it must never be
treated as the authorization boundary. Today the CLI exposes only `status`,
`workspace list`, `recipe list`, and `recipe create`.

**Healthy is not usable (2026-09-22).** `/health` reports healthy while every
workspace RPC returns `401 unauthenticated`, because `.vrooli/service.json`
declares no authentication profile (blocker B1). Check a real call —
`nutrition-planner workspace list` — not only the health endpoint.

## Common Incidents

| Symptom | Checks | Fix | Escalation |
|---|---|---|---|
| Scenario does not start | `make status`, `make logs` | `make restart`, then inspect lifecycle logs | Record recurring failures in [`../internal/PROBLEMS.md`](../internal/PROBLEMS.md). |
| Every page shows an error; the CLI reports an "unauthenticated — verified actor required" error | `nutrition-planner workspace list`; whether `.vrooli/service.json` has an `authentication` block; whether the API process received authentication configuration | Declare the platform authentication profile (`hybrid`, default mode `personal_local`) per repository `docs/concepts/IDENTITY-AND-AUTHENTICATION.md`, then `make restart`. Never add a handler bypass. | Known blocker B1 (D-032); tracked in `docs/internal/REDESIGN_PLAN.md` §3.2. |
| Settings data health shows a parse error | Browser network panel: the request goes to `/diagnostics` and returns HTML | Known defect B6: the client must call `/api/v1/diagnostics` | Fix in `ui/src/api/diagnostics.ts`; do not treat it as a server fault. |
| A meal image is missing or looks wrong | Asset manifest entry (approval, appearance, recipe revision compatibility); the safe fallback diagnostic in the logs | Expected fallback to editorial or minimal is not an incident. For a wrong or rejected curated asset, regenerate it through the asset procedure below; never hot-swap an unreviewed image. | Record the asset id and reason in `docs/internal/REDESIGN_LEDGER.md`. |
| API or database unavailable | `/health` payload, SQLite file exists and is writable, API logs | `make setup`; verify the storage path resolves for this scenario id, then `make restart`. If the file is corrupt, go to Backup / Restore. | A lost or unreadable database is data loss, not an incident to retry. Preserve a quarantine copy before any restore. |
| Migration mismatch (API refuses to start or reports a version error) | API health/log output naming the expected migration version; the database's recorded schema version | Do not delete the database. Restore a pre-migration checkpoint, or run the documented migration path. `SYS-05` requires testing upgrades on a populated database. | If no checkpoint exists, escalate before touching schema — do not guess. |
| Provider or AI unavailable | Job state (`queued`/`running`/`failed`), provider configured-vs-unavailable signal, safe error code | Expected. Use the prominent manual fallback: manual capture, manual prices, deterministic planning. No core flow is blocked. | Only escalate if manual fallback is missing or a job is stuck `running` past its budget. |
| Import or restore failure | Staging report, checkpoint presence, `UNSUPPORTED_IMPORT_VERSION` / `IMPORT_LIMIT_EXCEEDED` / `VALIDATION_FAILED` code | Application is transactional and all-or-nothing; cancel leaves current state untouched. If restore failed after a checkpoint was created, the old usable state must still be available. | If a partial write is observed, stop and escalate — that violates `SYS-02`. |
| Stale save conflict (`REVISION_CONFLICT` / `STALE_INPUTS`) | The conflict payload: entity type, expected vs current revision, suggested action | Keep the local draft; offer reload/compare. Never let a refresh discard the user's work. Re-apply against the current revision. | Recurring conflicts on one entity may indicate a client bug — record it. |
| PDF generation failure | API logs for the render path, requested layout/scope, whether an input snapshot exists | Retry generation from the immutable input snapshot. A failed PDF must not mutate plan data. | If long exports repeatedly fail, check the job budget and disk space; record the case. |
| CLI talks to an old or wrong API | `nutrition-planner status`, configured API base/port | Reinstall via `make setup`; use the scenario-prefixed env vars, not un-prefixed `API_PORT`. | Update [`../reference/cli-commands.md`](../reference/cli-commands.md) if a command changed. |

**The idempotency rule that governs every recovery.** A retry of a purchase,
a consumption event, a prepared-batch creation, an import/restore, or a
plan apply **must never apply the same effect twice** (`DOM-07` #12,
`SYS-02`). If a request committed but its response was lost, resolve by
re-reading the operation result or reusing the same idempotency key — do not
issue a fresh logical mutation. A conflict is not a network error and is
handled by revision compare, not by blind retry.

## Backup / Restore

**The SQLite database is non-regenerable.** It holds the user's recipes,
plans, intakes, batches, prices, purchases, feedback, and settings. None of
it can be recomputed from anything else. Treat backup as a first-class
release and operator task (`SYS-05`), and treat the portable export as a
second, format-independent copy.

Recommended procedure:

1. **Portable export (preferred, always valid).** Use the in-product
   Transfer center to produce a `daily.workspace` envelope at
   `schemaVersion: 2`. Confirm the manifest names its format, schema
   version, creation time, included record kinds, and any omissions (for
   example, source attachments omitted from a text-only export). Keep the
   exported file outside the scenario directory.
2. **File-level backup (operator).** Stop the scenario so no writer is
   active. Resolve the active database path through the storage resolver
   used by `api/main.go`; do not guess a home-directory path. Copy the
   database **and its WAL/SHM companions**, record the UTC timestamp and a
   checksum, and keep more than one dated copy on separate storage. For a
   live copy, use SQLite's backup-capable tooling against the database
   connection rather than copying only the main file.
3. **Verify.** Open the copy read-only and confirm expected row counts
   (recipes/revisions, plan occurrences, intake events) before trusting it.

Restore procedure:

1. Stop the scenario and **preserve the damaged database as a dated
   quarantine copy** — never overwrite the only copy.
2. Create a recoverable checkpoint of the current state before replacing
   domain state (`SYS-05`).
3. Restore the selected verified file backup, or apply the portable export
   through the staged, review-before-write import path. Restore into the
   currently authorized workspace with an explicit identity remapping
   policy; imports and restores must never install credentials, account
   membership, entitlements, or access tokens (`DOM-07` #11).
4. Start through the lifecycle and run the health check, then `make test`.
   A restore is not complete until recipe/plan/intake counts match the
   verification record and unknown values are still unknown, not zero.

| Data | Backup | Restore | Status |
|---|---|---|---|
| SQLite database (all domain state) | Portable `daily.workspace` export plus stopped-file copy with checksum | Staged import or file restore after checkpoint | **Required before real use; no automated backup target is configured yet.** The current workspace export covers only workspace, recipes, and plan and declares eleven omitted record kinds, so the file-level copy is the only complete backup today. |
| Media originals and derivatives (redesign) | Curated assets live in source with their manifest; user photos need a bounded archive with manifest and checksums (R25.2) | Re-link by asset id and content hash | Planned; generated assets are never regenerated on restore unless explicitly requested and budget-authorized. |
| Source attachments (if retained) | Bounded archive or explicitly omitted with a manifest report | Re-link by metadata reference | Deferred; a JSON-only export must state the omission. |
| Provider/AI credentials | Not here — held in the platform secret store | Re-resolved by reference | Deliberate; nothing to back up in this scenario. |
| Export bundles | Store outside the scenario directory | N/A | User-owned; never include secrets or billing identifiers. |

## Maintenance Tasks

| Task | Frequency | Command / Procedure |
|---|---|---|
| Validate tests | before handoff / release | `make test` (or `vrooli scenario test nutrition-planner --phases <relevant>`) |
| Inspect logs | as needed | `make logs` |
| Verify export round-trip | each release | Export, re-import into a clean workspace, compare semantic content (`ACT-029`). |
| Test migration on populated data | each schema change | Upgrade a copy of a populated database before shipping (`SYS-05`). |
| Regenerate endpoints | after API endpoint changes | `make endpoints` |
| Regenerate UI strings | after i18n changes | `cd ui && pnpm strings:gen` |
| Inspect PDFs/CSV | each release | Open the generated recipe/week PDFs and CSV bytes; do not trust HTTP 200 (`ACT-031`). |
| Review data health | weekly while in use | Unresolved ingredients, aged price observations, unknown targets, failed scheduled jobs (`OPS-02`). |
| Load demo data | on demand, development only | **Planned:** a repeat-safe demo seed command that loads the vegan fixture catalog into a separate demo workspace (R27.1). It does not exist yet; never load fixtures into a real workspace by hand. |
| Produce or replace a curated asset | when an asset is missing, rejected, or its recipe revision changed | Generate or edit through image-tools (`image-tools ai generate`, `ai edit`, `ai bg-removal`, then `image-tools ops convert`/`compress` for renditions; check flags with `image-tools ai <op> --help` and job progress with `image-tools jobs help`). Review at real UI sizes against R19.3, record prompt version, reference hashes, model role, reported cost, and reviewer in the asset manifest, then validate the manifest (D-029). Never crop concept mockups into assets. |
| Validate the asset manifest | after any asset change | **Planned:** a manifest validator (ids, hashes, dimensions, required metadata, safe-text rectangles within bounds, approval state) run as part of the unit or structure phase. |
| Review generation usage | while in-app generation is enabled | Settings › Generation & usage shows period usage, reservations, and queued jobs; a reservation that never settles stays pending, not free (R18.3). |

## Escalation

Record known operational issues in
[`../internal/PROBLEMS.md`](../internal/PROBLEMS.md). Append meaningful
completed work to [`../internal/PROGRESS.md`](../internal/PROGRESS.md).
For a suspected data-loss or cross-workspace exposure event, also read
[`../internal/SECURITY.md`](../internal/SECURITY.md) before acting; preserve
evidence rather than repairing in place.

## Cross-References

- [`DEPLOYMENT.md`](DEPLOYMENT.md) — deployment tiers and release checklist
- [`OBSERVABILITY.md`](OBSERVABILITY.md) — logs, metrics, and health signals
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — common fixes
- [`../reference/configuration.md`](../reference/configuration.md) — runtime configuration
- [`../internal/SECURITY.md`](../internal/SECURITY.md) — isolation and secret handling
- [`../internal/REDESIGN_PLAN.md`](../internal/REDESIGN_PLAN.md) — blockers, build order, and the artwork workflow
- [`../reference/product-specification.md`](../reference/product-specification.md) — R19, R22, R25, and Appendix A sections 11, 17, and 18
