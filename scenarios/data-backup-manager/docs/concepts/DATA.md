# Data — Data Backup Manager

This document is the canonical data ownership and storage map for the
scenario. Update it when domains add tables, files, blobs, external
records, retention rules, migrations, imports, or exports.

## Purpose Of This Document

Use this document to answer:

- What data does the scenario persist?
- Which domain owns each data shape?
- Where is the source of truth?
- What is the retention/deletion story?
- How are schema changes handled?

## Storage Overview

There are two distinct stores, and the distinction is load-bearing.

1. **The manager's own catalog** — embedded SQLite through
   `modernc.org/sqlite`, per-domain schema (greenfield, no migrations
   folder). The lifecycle sets `SQLITE_PATH` through
   `.vrooli/service.json`, and the API applies each domain's schema on
   startup through `api-core/database`. This catalog holds targets,
   destinations, plans, run history, and restore records.

2. **Backup artifacts** — these live in **kopia repositories**, one per
   destination, owned by the `kopia` resource. They are never stored
   under the scenario source tree, and the manager never hand-rolls
   their encryption, dedup, or compression. Artifact bytes are reached
   only through `resource-kopia`.

The catalog is intentionally a **cache and run-history anchor, not the
single source of truth** for which targets exist. Scenarios re-register
their targets idempotently on boot, so a lost catalog can be rebuilt
from re-registration; what is lost is run/restore history, not the
registration model. Source data read during capture (Postgres dumps,
Redis dumps, Qdrant snapshots, etc.) is transient and streamed into
kopia, not persisted in the catalog.

Source secrets and S3/backend credentials live in the `vault` resource;
repository passphrases live in the credential authority. They are
referenced — never copied — by the catalog.
Document resource decisions in [`INTEGRATIONS.md`](INTEGRATIONS.md)
before editing `.vrooli/service.json`.

## Data Ownership

| Data | Owning Domain | Storage | Source Of Truth | Retention | Notes |
|---|---|---|---|---|---|
| Targets | targets | SQLite | `api/internal/targets/schema.sql` | Until deregistered by the owning scenario | Keyed by `owner + name`; reconstructable from re-registration on boot. |
| Destinations | destinations | SQLite | `api/internal/destinations/schema.sql` | Until removed by an operator | Holds backend kind, storage cap, vault references, and the bundle-root (`location`) vs repository-path (`repository_location`) split — never the secrets themselves. |
| Destination bundle files | destinations | filesystem (bundle root) | The bundle root on the backend drive | Lives with the destination | `README.txt`, `RECOVERY.txt`, `vrooli-backup-destination.json` — non-secret operator-facing files written by the `BundleWriter` seam. Never contain a passphrase, only a credential-authority identity/field reference. |
| Plan bindings | plans | SQLite | `api/internal/plans/schema.sql` | Until the plan is deleted | Many-to-many plan↔target and plan↔destination membership plus schedule and retention. |
| Run history | runs | SQLite | `api/internal/runs/schema.sql` | Bounded retention (see Retention And Deletion) | Per-run outcomes plus redacted grouped preflight incidents; carries stable root-cause codes, next actions, last-success-per-target, and snapshot references. |
| Restore records | restores | SQLite | `api/internal/restores/schema.sql` | Bounded retention | Includes verify outcomes and last-verified-per-target (the git-removal gate). |
| Audit records | audits | SQLite | `api/internal/audits/schema.sql` | Bounded retention | Generic snapshot-audit proofs: status, restorability, live + snapshot inventories and the comparison, stored as JSON blobs. **Only relative paths, counts, and hashes — never file contents or secrets.** |
| Backup artifacts (snapshots) | destinations / kopia | kopia repository (filesystem or S3/MinIO) | The kopia repository per destination | Per-plan retention policy, applied by kopia; **alert+block, never silent eviction** | Encrypted by default; reached only via `resource-kopia`; never under the scenario source tree. |
| Repository passphrases | (referenced by) destinations | credential authority | `vrooli/kopia/<repository>` | Lives with the destination | Catalog and bundles store only the identity/field reference; the `kopia` resource resolves the value at call time. |
| S3/backend access keys | (referenced by) destinations | `vault` resource | `vault` | Lives with the destination | Catalog stores only secret references; the `kopia` resource resolves values at call time. |

## Schema Map

## Protection Tiers

Plans declare one independent protection role: `full_primary`,
`critical_primary`, or `critical_secondary`. Tier membership is explicit in
the plan and is persisted independently of destination readiness. Critical
plans should contain only the encrypted key inventory and the minimum state
needed to unlock and restore the full repository; plaintext credential stores
are never copied. A secondary tier must use a separately approved destination
and is not inferred from a full-primary result.
Every plan read also includes a derived topology assessment. The service sets
`destinations_physically_independent` only when selected roots do not overlap
and available volume identity does not reveal a shared device;
`shared_risk_warnings` reports missing identity, unsuitable media, shared
volumes, provider-domain uncertainty, and other evidence that prevents a
strong independence claim. Critical plan creation still rejects known
overlapping roots and source overlap.

| Table/File/Object | Owner | Defined In | Used By |
|---|---|---|---|
| targets tables | targets | `api/internal/targets/schema.sql` | targets repository/service/handlers |
| destinations tables | destinations | `api/internal/destinations/schema.sql` | destinations repository/service/handlers |
| plans tables (+ membership) | plans | `api/internal/plans/schema.sql` | plans repository/service/handlers |
| runs tables (+ per-target outcomes + grouped incidents) | runs | `api/internal/runs/schema.sql` | runs repository/service/handlers, health rollup |
| restores tables | restores | `api/internal/restores/schema.sql` | restores repository/service/handlers |
| audits table | audits | `api/internal/audits/schema.sql` | audits repository/service/handlers |
| kopia repositories (per destination) | kopia resource | created via `resource-kopia repo create` | runs (snapshot create), restores (snapshot restore/verify), audits (snapshot restore-to-scratch), destinations (stats/usage) |
| system schema | infrastructure | `api/internal/database/system.sql` | API boot and cross-cutting DB setup |

## Migrations And Compatibility

The generated template uses idempotent schema bootstrap. Domain schema
files should use `CREATE TABLE IF NOT EXISTS` and live beside the code
that interprets them.

For production data migrations that need column drops, renames, or data
backfills, add a scenario-specific migration plan here and update
[`../internal/DECISIONS.md`](../internal/DECISIONS.md) with the tradeoff.

## Import / Export

| Path | Format | Owner | Status |
|---|---|---|---|
| Snapshot restore | kopia repository → filesystem | restores | Intended: `restore` exports a target's snapshot to a location; `verify` exports to scratch and checksums. |
| Standalone disaster recovery | kopia repository → filesystem (no Vrooli) | kopia resource | Intended: artifacts are vanilla kopia repositories, restorable with the plain `kopia` binary + passphrase even with Vrooli down (see the kopia resource RECOVERY runbook). |

## Retention And Deletion

| Data | Delete Trigger | Retention Rule | Current Gap |
|---|---|---|---|
| Targets | Owning scenario deregisters (or owner is decommissioned) | Live while registered; re-registration re-creates | Deregistration semantics for orphaned owners to be defined in requirements. |
| Destinations | Operator removes the destination | Removal must not delete the underlying repository implicitly | Destination removal vs repository deletion separation to be defined in requirements. |
| Backup artifacts (snapshots) | Per-plan retention policy applied by kopia (e.g. GFS) | **Alert + block by default at the storage cap** — never silent eviction; eviction happens only via explicit retention | Cap-breach behavior and retention policy editing are the locked default; exact thresholds are configuration. |
| Run / restore history | Bounded history retention in the catalog | Keep enough to show last-success and last-verified per target | Exact history-trim policy to be defined in requirements. |

The single most important deletion rule: the manager never deletes a
backup to stay under a storage cap. Hitting a cap alerts and refuses
new writes; reclaiming space requires explicit retention on a plan.

### Filesystem destination bundle layout

A filesystem destination is a self-describing bundle, not a bare repository:

```text
<location>/                                  ← bundle root (operator-facing)
  README.txt                                 ← what this is; do not edit the repo by hand
  RECOVERY.txt                               ← standalone recovery steps (plain kopia + passphrase)
  vrooli-backup-destination.json             ← non-secret manifest (schema version, ids, repo path, secret REF)
  repositories/
    <slug>.kopia/                            ← the vanilla, encrypted kopia repository
```

- `location` is the operator-facing **bundle root**; `repository_location` is the
  concrete kopia repository path (`<location>/repositories/<name>.kopia`). The
  name is slug-safe (lowercase, digits, hyphens) because it doubles as the kopia
  repository name.
- The repository remains a **vanilla kopia repository** — no proprietary layer.
  It is restorable with the plain `kopia` binary plus the passphrase from the
  credential authority.
- Bundle files never contain a secret value, only a credential-authority
  identity/field reference (and backend references where applicable).
- S3 destinations have no filesystem bundle files; `location` and
  `repository_location` are the bucket/prefix.

### Deletion retention rules (destinations)

| Command | Removes | Leaves intact |
|---|---|---|
| `destinations delete` (default) | Catalog row only | Local kopia metadata, credential-authority passphrase and S3 secret refs, encrypted repository bytes, bundle files |
| `destinations delete --delete-repository` | Catalog row + local resource-kopia metadata/config/cache + credential-authority passphrase and S3 secret refs | **Encrypted repository bytes on the backend** and bundle files — never removed by DBM |

To destroy the backups themselves, an operator removes the bundle folder
manually. DBM never deletes backend repository bytes.

## Privacy Notes

This scenario backs up **other scenarios' runtime state**, which may
contain personal, regulated, customer, financial, or sensitive
business data. Two controls bound the exposure: every destination is
encrypted by default, and all passphrases and access keys come from the
`vault` resource — never config files, env files, or process argv. The
manager handles source bytes only in transit (capture → kopia) and does
not retain them in its own catalog. The generic snapshot audit upholds
the same bound: it walks restored and live bytes in scratch only, and the
persisted audit record carries relative paths, counts, and hashes — never
file contents, table rows, or secret values. Update this document and
[`../internal/SECURITY.md`](../internal/SECURITY.md) when source kinds,
destination backends, or secret handling change.

## Cross-References

- [`DOMAINS.md`](DOMAINS.md) — data ownership by domain
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — external resources and scenarios
- [`../reference/configuration.md`](../reference/configuration.md) — runtime configuration
- [`../internal/SECURITY.md`](../internal/SECURITY.md) — privacy/security posture
