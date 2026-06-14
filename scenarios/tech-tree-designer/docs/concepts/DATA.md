# Data - Tech Tree Designer

## Purpose Of This Document

Record data ownership, storage, retention, and import/export expectations for TTD.

## Storage Overview

TTD uses SQLite through the `react-vite` template's routed database substrate. Phase 1 has no product tables beyond the system schema. Future domains add domain-owned schemas beside their code.

## Data Ownership

| Data | Owning Domain | Storage | Source Of Truth | Retention | Notes |
|---|---|---|---|---|---|
| Health status | health | none | runtime probes | not persisted | Operational only. |
| Graph cache | graph | SQLite optional | `proto-health` / future SDA | rebuildable | Cache must never become graph SSOT. |
| Planned scenarios | planning | SQLite | `planned_scenario` | until deleted/materialized | Slug, sector, tier, target stability. |
| Planned proto files | planning | SQLite | `planned_proto_file.text` | until deleted/materialized | Real `.proto` text is the plan SSOT. |
| Roadmap overlays | roadmap | SQLite | roadmap tables | until deleted | Sector/tier/milestone metadata. |

## Schema Map

| Domain | Planned Schema Location | Status |
|---|---|---|
| health | none | implemented |
| graph | `api/internal/graph/schema.sql` | planned |
| planning | `api/internal/planning/schema.sql` | planned |
| roadmap | `api/internal/roadmap/schema.sql` | planned |

## Migrations And Compatibility

This is a greenfield regeneration. Use declarative domain-owned schemas until real production users exist. Do not add Postgres migrations or compatibility shims for the deleted implementation.

## Import / Export

Graph export will support JSON, DOT, and text. Planning materialization exports validated planned proto files to `packages/proto/schemas/<slug>/` and runs proto generation.

## Retention And Deletion

Planned scenarios and roadmap overlays should be deletable through domain APIs/CLI. Materialized proto files become shared repo artifacts and are governed by proto package policy, not by TTD's SQLite retention.

## Privacy Notes

TTD stores planning metadata and proto text, not end-user private content. Planned proto text may reveal future scenario names and interface intent, so treat it as internal planning data.

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md)
- [`DOMAINS.md`](DOMAINS.md)
- [`../internal/SEAMS.md`](../internal/SEAMS.md)
