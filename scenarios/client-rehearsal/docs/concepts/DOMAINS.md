# Domains — Client Workflow Rehearsal

Client Workflow Rehearsal is a reusable rehearsal capability for freelance automation delivery. Its first bounded context is `rehearsals`: it owns imported workflow revisions, credential substitution, synthetic run evidence, operator review, and handoff state. Execution engines and simulated third-party behavior are collaborators behind typed seams, not domains that own client state.

## Domain Inventory

| Domain | Responsibility | Owns Data | Primary surfaces |
|---|---|---|---|
| `health` | Report API and database readiness. | None. | API, shell readiness card |
| `rehearsals` | Import a workflow, scrub credentials, run it through an engine against synthetic services, capture faults/repair evidence, and collect operator disposition. | Engagements, workflow revisions, credential bindings, simulator profile references, runs, trace events, review decisions, handoff snapshots. | Connect API, CLI, rehearsal workspace UI |

## `rehearsals` ownership

The domain owns the durable record and application policy. Its first import accepts a made-up workflow document with an explicit set of typed synthetic actions; the importer recursively replaces secret-shaped values before persistence. The engine adapter accepts only the normalized, credential-free workflow. Its simulator interface resolves declared operations to deterministic responses and records every request. A workflow cannot direct the simulator to an arbitrary URL.

The first synthetic workflow exercises a CRM contact upsert followed by a spreadsheet append. A seeded validation fault is detected, repaired using the synthetic execution record, and presented for operator review. Review states are `needs_review`, `approved`, and `changes_requested`; execution success never implies approval. The review decision and generated handoff are append-only evidence associated with a run.

| Concern | Owner |
|---|---|
| Workflow parsing, secret replacement, run policy, repair evidence, review transitions | `api/internal/rehearsals/` |
| Proto-backed public contract and generated client | `packages/proto/schemas/client-rehearsal/v1/rehearsals/` |
| Runtime engine and simulated-service collaborators | typed `Engine` and `Simulator` interfaces injected into the domain service |
| SQLite tables and repository | `api/internal/rehearsals/` |
| CLI and browser translation | `cli/domains/rehearsals/`, `ui/src/features/rehearsals/` |

## Deferred domains

- `simulators`: defer until the reusable catalog is independently managed; the rehearsal catalog now supports OpenAPI and recorded-exchange matching; native runtime routing is a separate open integration gate.
- `engagements`: engagement metadata stays in `rehearsals` until a second workflow or consumer proves an independent lifecycle is needed.
- Additional execution engines: implement adapters against the stable engine contract; do not fork rehearsal records or review semantics.
