# Decisions — Data Backup Manager

This document records durable decisions and tradeoffs future agents
should not accidentally relitigate.

## Purpose Of This Document

Use this document when a choice:

- affects multiple files or future agents,
- rejects a plausible alternative,
- changes architecture, deployment, data, security, monetization, or
  testing direction,
- needs a revisit trigger.

Routine implementation notes belong in [`PROGRESS.md`](PROGRESS.md).
Known unresolved issues belong in [`PROBLEMS.md`](PROBLEMS.md).

## Decision Log

| Date | Decision | Context | Consequences | Revisit Trigger |
|---|---|---|---|---|
| 2026-05-26 | Use the generated `react-vite` scenario documentation contract. | Scenario scaffold was generated from the template. | Docs start with stubs and maturity metadata in `docs/manifest.json`. | Revisit when scenario adopts a different template or doc contract. |
| 2026-05-26 | Wrap the new `kopia` resource as the backup engine rather than hand-rolling backup mechanics. | Backup needs dedup, encryption, compression, incrementals, multi-destination, retention, integrity checks, stats, and standalone restore. Building these is years of work that an established engine already solves. Follows the wrap-not-use principle. | The manager orchestrates kopia rather than reimplementing crypto/dedup. Snapshots remain restorable with the plain `kopia` CLI even if Vrooli is down — a deliberate disaster-recovery property. Companion plan: `docs/plans/kopia-resource-plan.md` (repo root). | Revisit only if kopia is abandoned upstream or cannot meet a hard requirement (e.g., a backend it will never support). |
| 2026-05-26 | Decouple Sources from Destinations with an explicit five-noun model. | A backup target (what to protect) and a backup destination (where it lands) change independently, and the same target often needs more than one destination. | The domain is TARGET (a registered source) × DESTINATION (a kopia repository) × PLAN (a many-to-many binding carrying schedule + retention) → RUN → RESTORE. Plans, not targets or destinations, own scheduling and retention. | Revisit if a simpler model proves sufficient, or if a target/destination distinction collapses in practice. |
| 2026-05-26 | Support six source kinds in v1: filesystem, SQLite, Postgres, Redis, Qdrant, object-storage. | These cover the runtime state Vrooli scenarios actually own today. | Each kind captures into a consistent artifact: SQLite via `VACUUM INTO`, Postgres via `pg_dump`, Redis via prefix `SCAN`+`DUMP` (best-effort, non-transactional — accepted limitation), Qdrant via its snapshot API, object-storage via S3/MinIO mirror, filesystem directly. Redis's lack of point-in-time consistency is documented and accepted. | Add kinds as new runtime stores appear; revisit Redis semantics if a transactional snapshot path becomes available. |
| 2026-05-26 | Self-registration with a reconstructable catalog. | Scenarios own their runtime state and know best what to back up; the manager should not hard-code which scenarios exist. | Scenarios idempotently register/deregister targets (owner+name keyed, like agent-manager's `EnsureProfile`) and re-register on boot. The manager's SQLite DB is a cache + run history, not the source of truth — it can be rebuilt from re-registration. | Revisit if registration churn or boot ordering makes the cache-not-truth model painful. |
| 2026-05-26 | Encryption ON by default for every destination; repository passphrases use the credential authority and backend/source secrets use `vault`. | Backups frequently contain secrets-bearing source data, and a leaked unencrypted repository is a full breach. | Every destination is an encrypted kopia repository. Repository passphrases are stored under per-repository credential-authority identities; backend access keys and source credentials come from the `vault` resource at runtime. Neither is read from config files or process arguments. See `SECURITY.md`. | Revisit only with a vetted reason to allow an unencrypted destination (none expected). |
| 2026-08-05 | Route all DBM and resource-kopia repository/backend credentials through the credential authority; remove Vault from this stack. | The authority already provides native OS keyring or encrypted portable storage plus operator-controlled recovery bundles. Vault's sealed bootstrap is not needed for the filesystem-first backup product and creates an avoidable lifecycle dependency. | Repository passphrases and future S3 credentials use per-repository authority identities; DBM and Kopia have no Vault dependency. Source resources retain their own credential contracts until separately migrated. Existing encrypted repositories remain preserved but require their original passphrases. | Revisit only if a concrete source resource requires an explicitly governed compatibility adapter. |
| 2026-05-26 | Storage limits default to alert+block, never silent eviction. | Safety over tidiness: a backup tool that quietly deletes backups to stay under a cap can destroy the only recovery copy. | Per-destination caps are configurable and default to alerting and blocking new writes when exceeded. Old snapshots are removed only by an explicit retention policy, never to free space. | Revisit only if an operator-controlled, clearly-labeled eviction mode is explicitly requested. |
| 2026-05-26 | Verified restore is first-class and gates removal of committed runtime data from git. | "We commit it so we don't lose it" is only safe to retire once restore is proven, not assumed. | A verify mode test-restores to scratch and checksums the result. No committed runtime data is removed from git until its target has a passing verified restore. | Revisit if the verify gate proves too slow to run at the needed cadence (optimize, don't drop the gate). |
| 2026-05-26 | Separate-root rule: a destination must not live under the storage root it protects. | A destination inside the protected tree is destroyed by the same incident that destroys the source. | Destinations are validated to be outside the protected root; an offsite destination is preferred for at least one tier. | Revisit when adding cross-destination replication (PRD OT-P2-002) — offsite tiering interacts with this rule. |
| 2026-05-26 | No n8n / no external orchestrator; in-process scheduler and greenfield per-domain SQLite. | The prior data-backup-manager leaned on n8n and was never used; an external orchestrator adds a moving part for a capability that must be maximally dependable. | Scheduling runs in-process. Catalog/run-history storage is greenfield per-domain SQLite (`modernc.org/sqlite`), no migrations folder. | Revisit if scheduling needs outgrow in-process (e.g., distributed multi-node execution). |
| 2026-05-26 | Replace the stale pre-2025 data-backup-manager. | The earlier scenario (n8n + MinIO) backed up the repo *source tree*, solved the wrong problem, and was never used. | This is a clean rebuild around runtime state, not source. The old scenario is preserved at `/tmp/data-backup-manager-old` for reference only and is not carried forward. | Recorded under Superseded Decisions; no revisit expected. |

## Superseded Decisions

| Date | Superseded Decision | Replacement | Notes |
|---|---|---|---|
| 2026-05-26 | Pre-2025 data-backup-manager: n8n + MinIO backing up the repository source tree. | This rebuild: kopia-backed backup of registered *runtime state*, in-process scheduler. | The old scenario was never used and solved the wrong problem (source, not runtime state). Preserved at `/tmp/data-backup-manager-old` for reference only. |

## Cross-References

- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system decisions
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved drift and debt
- [`PROGRESS.md`](PROGRESS.md) — completed work history
