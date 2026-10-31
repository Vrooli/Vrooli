# Data — Scenario Completeness Scoring

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

The template default is embedded SQLite through `modernc.org/sqlite`.
The database path is resolved from the scenario id by `api-core/storage`, and
the API applies schemas on startup through `api-core/database`.

External storage resources should be introduced only when a real
domain needs them. Document those decisions in
[`INTEGRATIONS.md`](INTEGRATIONS.md) before editing
`.vrooli/service.json`.

## Data Ownership

This scenario reads most score inputs from the *target* scenario's
cached artifacts, which other tools own and write:

| Read Path (relative to target scenario) | Owner/Writer | Read By Domain | Notes |
|---|---|---|---|
| `requirements/index.json` + imported `*/module.json` | business-health / agents | signals | Requirement + operational-target pass rates. |
| `coverage/requirements-sync/latest.json` (+ legacy fallbacks) | requirements sync tooling | signals | Preferred source for requirement/target status. |
| `coverage/phase-results/*.json` | test-genie | signals | Findings decode (proto `ArchitectureFinding`) with legacy summary fallback; mapped to maturity-go dimensions. |
| `coverage/runs.index.json` | test-genie (flock-guarded write side) | freshness | Read-only via `packages/freshness-go/runindex`. |
| `.vrooli/service.json` | scenario author | signals | Category for threshold selection. |
| `ui/src/**` sources | scenario author | signals | UI heuristics (template detection, components, routing, API usage, LOC). |

| Data | Owning Domain | Storage | Source Of Truth | Retention | Notes |
|---|---|---|---|---|---|
| Score snapshots | scoring | SQLite | `api/internal/scoring/schema.sql` | Retained until an explicit compaction/retention policy ships. | Digest-deduplicated history written by the background sweeper and explicit page-bounded recomputes; read by `GetScore` deltas, trend, fleet list, and measures. |

## Schema Map

| Table/File/Object | Owner | Defined In | Used By |
|---|---|---|---|
| `score_snapshots` | scoring | `api/internal/scoring/schema.sql` | scoring repository, sweeper, `ScoreService`, measures |
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
| None yet. | n/a | n/a | Add when product requirements include import/export. |

## Retention And Deletion

| Data | Delete Trigger | Retention Rule | Current Gap |
|---|---|---|---|
| `score_snapshots` | None today. | Keep all digest-distinct snapshots for trend and fleet-measure reads. | Define compaction once snapshot volume is measured across a large fleet. |

## Privacy Notes

Generated template data is local development data. If a scenario stores
personal, regulated, customer, financial, or sensitive business data,
update this document and [`../internal/SECURITY.md`](../internal/SECURITY.md)
before implementation expands.

## Cross-References

- [`DOMAINS.md`](DOMAINS.md) — data ownership by domain
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — external resources and scenarios
- [`../reference/configuration.md`](../reference/configuration.md) — runtime configuration
- [`../internal/SECURITY.md`](../internal/SECURITY.md) — privacy/security posture
