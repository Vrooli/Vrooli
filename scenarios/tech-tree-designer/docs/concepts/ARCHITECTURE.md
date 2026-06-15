# Architecture - Tech Tree Designer

Tech Tree Designer is Vrooli's scenario-centric planning surface. It keeps the modern `react-vite` scenario shape while replacing the old Gin/Postgres implementation with Connect, SQLite, generated proto contracts, and domain-owned code.

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
SQLite planning + ontology metadata
```

The graph is scenario-centric: nodes are scenarios, edges are real interface dependencies. The ontology is a separate top-down capability layer joined to scenarios through explicit fulfillment links.

## Interfaces

| Interface | Obligation |
|---|---|
| Programmatic | Connect graph, planning, and ontology RPCs must be stable enough for agents and future scenarios to consume. CLI commands mirror the API for operator and agent use. |
| Direct UI | The interactive graph, planning editor, and ontology coverage views must handle loading, error, empty, and validation-finding states before the scenario is considered production-ready. |

## Ecosystem Fit

Role: meta / interface-enabler. TTD advances the engineering meta-capability by letting agents plan scenarios around proto contracts and actual dependency evidence before code exists.

Compound-value seams:
- `GraphSource` lets the live graph use `proto-health` now and `scenario-dependency-analyzer` later.
- Planning RPCs let future agents create, validate, inspect, and materialize planned proto contracts.
- Ontology coverage/focus RPCs let future agents prioritize gaps and place unmapped scenarios.
- Export/query RPCs let other scenarios ask neighborhood, path, ancestry, and graph-shape questions.

Monetization: internal meta scenario; no paid-feature wiring.

## Domain Module Pattern

Each real domain owns its API internals, handlers, CLI package, UI feature, proto schema, and storage schema. Current implementation contains health, the graph domain's proto-health-backed query/export surface, the planning domain's SQLite-backed planned proto file tree, validator, materializer, planned graph overlay, and the ontology domain's capability tree, fulfillment, coverage, focus, and overlay projection. Graph, planning, and ontology are exposed through Connect RPC, CLI commands, and production UI routes.

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
- `api/internal/ontology/schema.sql` for capabilities, capability edges, fulfillment links, and explicit coverage exclusions.

`materialize` is the one intentional outside-scenario write path: it writes validated planned proto text into `packages/proto/schemas/<slug>/` and runs `make generate`.

## Intentional Deviations

| Date | Deviation | Reason | Revisit Trigger |
|---|---|---|---|
| 2026-06-14 | Old TTD deleted instead of migrated. | No consumed proto/API contract existed; preserving Gin/Postgres code would keep obsolete architecture and dependency heuristics alive. | Never; use git history for old concepts only. |
| 2026-06-14 | Phase 2 shipped graph/planning/roadmap proto contracts before all domains were mounted. | Contract-first planning kept the wire shape reviewable before persistence and UI work landed. | Resolved by Phases 3-6; keep as historical context. |
| 2026-06-14 | Graph API and CLI ran before planned nodes existed. | The live graph was useful independently and exercised the `GraphSource` seam over proto-health. | Resolved by Phase 4 planned-node merge behavior; keep as historical context. |
| 2026-06-14 | `plan tree <slug> [path]` and `plan add` cover tree/show and add/edit semantics. | `cli-health` enforces one CLI binding per RPC; duplicate `tree/show` and `add/edit` command bindings are rejected. | Add dedicated proto RPCs if separate command names become worth the contract cost. |

## Cross-References

- [`DOMAINS.md`](DOMAINS.md) - domain inventory and ownership
- [`INTEGRATIONS.md`](INTEGRATIONS.md) - scenario dependency decisions
- [`../internal/SEAMS.md`](../internal/SEAMS.md) - seam registry
- [`../../PRD.md`](../../PRD.md) - operational targets
