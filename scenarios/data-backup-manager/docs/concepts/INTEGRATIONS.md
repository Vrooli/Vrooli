# Integrations — Data Backup Manager

This document is the canonical dependency contract for resources,
other scenarios, and third-party services used by the scenario.

## Purpose Of This Document

Use this document to answer:

- What does the scenario depend on?
- Which dependencies are required versus optional?
- Which domain uses each dependency?
- What is the failure or degradation behavior?
- Where is the dependency declared or configured?

The guiding principle is **wrap-not-use**: this scenario does not
hand-roll crypto, dedup, an object-storage client, or an external
orchestrator. It wraps the `kopia` resource for all snapshot/restore
work and reads each source through that source's own resource CLI. The
prior n8n + MinIO design that backed up the repo source tree is the
explicit anti-pattern and is not echoed here.

## Dependency Inventory

| Dependency | Type | Required? | Used By | Contract | Failure Behavior |
|---|---|---|---|---|---|
| SQLite | embedded storage | yes | API (all domains) | `SQLITE_PATH` lifecycle env var | API reports unhealthy if unreachable. |
| `kopia` resource | Vrooli resource (backup engine) | yes | destinations, runs, restores | `resource-kopia` CLI (repo/snapshot/restore/verify/stats/policy) | A run/restore against an unreachable engine fails closed and is recorded as a failed run; health flags it. |
| credential authority | Vrooli shared package/store | yes | destinations, kopia boundary | Per-repository identity/field for repository passphrases | Fail closed — never fall back to a default/empty passphrase or run unencrypted. |
| `postgres` resource | Vrooli resource (source kind) | on demand | sources (postgres kind) | `pg_dump` via the postgres resource CLI | Capture of a Postgres target fails for that target; other targets in the run continue. |
| `redis` resource | Vrooli resource (source kind) | on demand | sources (redis kind) | prefix `SCAN` + `DUMP` via the redis resource CLI | Best-effort capture; failure marks that target failed (not point-in-time consistent by design). |
| `qdrant` resource | Vrooli resource (source kind) | on demand | sources (qdrant kind) | Qdrant snapshot API via the qdrant resource CLI | Capture of a Qdrant target fails for that target. |
| `minio` resource | Vrooli resource (source + destination) | on demand | sources (object-storage kind), destinations (s3 backend) | S3-compatible SDK; shared between the object-storage source mirror and the S3 destination backend | Unreachable endpoint fails that target/destination operation. |
| Vrooli lifecycle | local platform | yes | API, UI, CLI | `.vrooli/service.json`, Makefile targets | Scenario should be started through lifecycle commands. |

## Vrooli Resources

Resources are declared in `.vrooli/service.json` under
`dependencies.resources`. `kopia` and the credential authority are core to the design;
the source-kind resources are needed only when a target of that kind is
registered.

| Resource | Status | Reason | Revisit Trigger |
|---|---|---|---|
| kopia | required | The wrapped backup engine; one kopia repository per destination. All snapshot/restore/verify/stats/retention go through `resource-kopia`. | n/a — foundational to the scenario. |
<!-- kopia snapshot metadata: see "Self-identifying snapshot metadata" below. -->
| credential authority | required | Destination repository passphrases are stored under per-repository identities; never config or argv. | n/a — foundational to the encryption-on-by-default rule. |
| postgres | conditional | Needed to `pg_dump` Postgres-kind targets. | Enable when a Postgres target is registered. |
| redis | conditional | Needed to capture Redis-kind targets (best-effort prefix snapshot). | Enable when a Redis target is registered. |
| qdrant | conditional | Needed to capture Qdrant-kind targets via the snapshot API. | Enable when a Qdrant target is registered. |
| minio | conditional | Backs object-storage source mirrors and S3-backed destinations. | Enable when an S3/MinIO source or destination is used. |

### Self-identifying snapshot metadata

Every backup run stamps self-identifying metadata onto its kopia snapshot via
`resource-kopia snapshot create`, so a standalone `kopia snapshot list --json`
can attribute a snapshot without DBM running:

- `--override-source dbm://<owner>/<name>` — a stable logical source, so the
  snapshot is not labeled with the throwaway staging path.
- `--description "Data Backup Manager target <owner>/<name> run <run-id>"`.
- `--tags` (repeatable `key:value`): `dbm:true`, `dbm.target_id:<id>`,
  `dbm.owner:<owner>`, `dbm.name:<name>`, `dbm.kind:<kind>`, `dbm.run_id:<id>`,
  `dbm.destination_id:<id>`.

The target **locator** is deliberately excluded from tags/description/override
(potentially sensitive); none of these fields ever carry a secret value.

## Scenario Dependencies

| Scenario | Status | Reason | Contract |
|---|---|---|---|
| prompt-manager | first customer (planned) | First real registration customer: registers `store/teams/**` as a filesystem target and stops committing it to git once a verified restore is proven (OT-P1-005). | prompt-manager self-registers via this scenario's CLI at its lifecycle; this scenario does not depend on prompt-manager to run. |

This scenario stays agnostic of which scenarios use it — owning
scenarios register themselves. No scenario is required for this one to
start.

## Third-Party Services

| Service | Status | Reason | Contract |
|---|---|---|---|
| S3-compatible object storage | optional (via kopia/minio) | Destinations and object-storage sources can target S3/MinIO. Other kopia backends (B2, GCS, Azure, SFTP) are P2. | Reached only through the `kopia`/`minio` resources; repository and backend credentials use the credential authority. |

## Failure Modes

| Dependency | Failure Signal | Expected Behavior | Tests |
|---|---|---|---|
| SQLite | `PingContext` error | `/health` returns unhealthy dependency status. | health handler tests |
| kopia resource | `resource-kopia` non-zero exit / unreachable | Run or restore fails closed and is recorded as a failed run; health flags overdue/failed backups. | run/restore service tests with a fake kopia seam |
| credential authority | secret unavailable | Fail closed — refuse to run unencrypted or with a default credential; surface an actionable error. | destination service tests with fake authority seam |
| source resource (postgres/redis/qdrant/minio) | capture command failure | That target's outcome is failed; other targets in the run continue; run is partial-failed. | per-source-kind handler tests |

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system boundaries
- [`DATA.md`](DATA.md) — storage ownership
- [`../reference/configuration.md`](../reference/configuration.md) — environment and service manifest
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — deployment readiness
