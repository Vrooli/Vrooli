# Architecture - Tech Tree Designer

Tech Tree Designer is being regenerated as Vrooli's scenario-centric planning surface. It keeps the modern `react-vite` scenario shape while replacing the old Gin/Postgres implementation with Connect, SQLite, generated proto contracts, and domain-owned code.

## Purpose Of This Document

This document records the intended system shape for the regenerated scenario. Product capability boundaries live in [`DOMAINS.md`](DOMAINS.md); detailed data retention will evolve in [`DATA.md`](DATA.md); durable seams live in [`../internal/SEAMS.md`](../internal/SEAMS.md).

## Scenario Shape

```
proto-health DescribeScenariosProtos
        |
        v
  graph.GraphSource seam
        |
        v
Connect API <-> CLI / UI / future agents
        |
        v
SQLite planning + roadmap metadata
```

The graph is scenario-centric: nodes are scenarios, edges are real interface dependencies. Sector, tier, and milestone data are overlays for grouping and progress, not a separate fulfillment model.

## Interfaces

| Interface | Obligation |
|---|---|
| Programmatic | Connect graph and planning RPCs must be stable enough for agents and future scenarios to consume. CLI commands mirror the API for operator and agent use. |
| Direct UI | The D3 graph, planning editor, and roadmap views must handle loading, error, empty, and validation-finding states before the scenario is considered production-ready. |

## Ecosystem Fit

Role: meta / interface-enabler. TTD advances the engineering meta-capability by letting agents plan scenarios around proto contracts and actual dependency evidence before code exists.

Compound-value seams:
- `GraphSource` lets the live graph use `proto-health` now and `scenario-dependency-analyzer` later.
- Planning RPCs let future agents create, validate, inspect, and materialize planned proto contracts.
- Export/query RPCs let other scenarios ask neighborhood, path, ancestry, and graph-shape questions.

Monetization: internal meta scenario; no paid-feature wiring in this phase.

## Domain Module Pattern

Each real domain owns its API internals, handlers, CLI package, UI feature, proto schema, and storage schema. Current implementation contains health plus the graph domain's proto contract, `GraphSource` seam, Connect handlers, query/export service, and CLI commands. Planning storage, roadmap storage, and UI features land in later phases.

## Contract Rules

- Proto schemas under `packages/proto/schemas/tech-tree-designer/` are the source of truth for wire contracts.
- Connect-RPC is the default transport.
- REST exceptions require explicit `rest_exception.proto_payloads` metadata in `.vrooli/endpoints.json`.
- Do not hand-edit generated proto outputs.
- Do not recreate the old service.json declared-dependency heuristic catalog.

## Storage Rules

SQLite is the default store. Domain schemas live beside their domain code:
- `api/internal/graph/schema.sql` for optional graph cache.
- `api/internal/planning/schema.sql` for planned scenarios and planned proto files.
- `api/internal/roadmap/schema.sql` for sectors, tiers, milestones, and overlays.

`materialize` is the one intentional outside-scenario write path: it writes validated planned proto text into `packages/proto/schemas/<slug>/` and runs `make generate`.

## Intentional Deviations

| Date | Deviation | Reason | Revisit Trigger |
|---|---|---|---|
| 2026-06-14 | Old TTD deleted instead of migrated. | No consumed proto/API contract existed; preserving Gin/Postgres code would keep obsolete architecture and dependency heuristics alive. | Never; use git history for old concepts only. |
| 2026-06-14 | Phase 2 shipped graph/planning/roadmap proto contracts before all domains were mounted. | Contract-first planning keeps the wire shape reviewable before persistence and UI work land. | Planning storage in Phase 4, roadmap overlay in Phase 5, UI in Phase 6. |
| 2026-06-14 | Graph API and CLI now run before planned nodes exist. | The live graph is useful independently and exercises the `GraphSource` seam over proto-health. | Add planned-node merge behavior when planning storage lands. |

## Cross-References

- [`DOMAINS.md`](DOMAINS.md) - domain inventory and ownership
- [`INTEGRATIONS.md`](INTEGRATIONS.md) - scenario dependency decisions
- [`../internal/SEAMS.md`](../internal/SEAMS.md) - seam registry
- [`../../PRD.md`](../../PRD.md) - operational targets
