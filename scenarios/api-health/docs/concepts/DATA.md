# Data — API Health

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

API Health starts with embedded SQLite through the generated template. The
path is resolved by `api-core/storage` from the scenario id, and the API applies
system schemas on startup.

Initial provider validation is compute-on-read over target scenario files and
optional live probe responses. Persisted product data is intentionally minimal.

## Data Ownership

| Data | Owning Domain | Storage | Source Of Truth | Retention | Notes |
|---|---|---|---|---|---|
| Provider system tables | infrastructure | SQLite | `api/internal/database/system.sql` | Until scenario data directory is removed. | Template bootstrap only. |
| Probe evidence | probe | response-local initially; optional SQLite later | Native validation detail | Not persisted in v1. | If P2 fleet/probe history lands, add retention before storing. |
| Fix sessions | remediation | response-local initially; optional SQLite later | Fix RPC response | Not persisted in v1. | Apply writes target scenario files only on explicit request. |
| Migration ledger | migration | repository docs/tests | `docs/reference/scenario-auditor-api-migration.md` and fixtures | Version-controlled until migration retired. | Planned document, not database state. |

## Schema Map

| Table/File/Object | Owner | Defined In | Used By |
|---|---|---|---|
| system schema | infrastructure | `api/internal/database/system.sql` | API boot and health |
| `.vrooli/maturity.json` | validation | `.vrooli/maturity.json` | assessment mapping, provider conformance |
| target scenario source files | target scenario | target repo tree | validation and probe domains, read-only |

## Migrations And Compatibility

The current persisted schema is generated infrastructure only. Any future probe
history, fix session, or fleet-readiness table must ship with:

- a domain-owned schema file,
- migration tests,
- retention policy in this document,
- privacy/security review in `docs/internal/SECURITY.md`.

## Import / Export

| Path | Format | Owner | Status |
|---|---|---|---|
| Validation report JSON | proto JSON | validation | Planned CLI/API output, not persisted. |
| Fix preview JSON | proto JSON | remediation | Planned CLI/API output, not persisted. |

## Retention And Deletion

| Data | Delete Trigger | Retention Rule | Current Gap |
|---|---|---|---|
| Provider SQLite system data | Scenario data cleanup. | Keep while scenario is installed. | None. |
| Live probe evidence | Not persisted in v1. | Response-local only. | Define before adding history. |
| Fix preview/apply evidence | Not persisted in v1. | Response-local only. | Define before adding history. |

## Privacy Notes

API Health reads source files and may capture live `/health` payloads from local
scenarios. Health payloads can include dependency names, database names, and
error strings. Reports should avoid storing probe bodies by default and should
redact secrets if future persistence is introduced.

## Cross-References

- [`DOMAINS.md`](DOMAINS.md) — data ownership by domain
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — external resources and scenarios
- [`../reference/configuration.md`](../reference/configuration.md) — runtime configuration
- [`../internal/SECURITY.md`](../internal/SECURITY.md) — privacy/security posture
