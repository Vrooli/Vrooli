# Domains — Prose Studio

This is the ownership map for the governed prose capability. Product records
are data; sampler and selection *kinds* remain closed algorithms because their
invariants are enforced in code.

## Domain Inventory

| Domain | Responsibility | Purpose | Owns Data | Primary Archetype | Secondary Traits | Source Paths |
|---|---|---|---|---|---|---|
| `styles` | Versioned voices and extends resolution | Define reusable writing style | Styles and conformance spans | service | validation, mutation | `api/internal/prose`, `packages/proto/schemas/prose-studio` |
| `profiles` | Sampler, constraints, policy, budgets, roles, context policy | Resolve generation policy | Profiles and registry kinds | service | validation, mutation | `api/internal/prose`, `packages/proto/schemas/prose-studio` |
| `generation` | Gateway calls, rounds, candidates, provenance, disclosure, cost | Produce auditable candidate sets | Rounds and candidates | orchestration | provider, mutation | `api/internal/prose`, `api/handlers/prose` |
| `measurement` | Deterministic text metrics, pairwise matrix, set basis | Measure outputs without model taste | Metric payloads | validation | aggregation, scoring | `packages/textmetrics`, `api/internal/prose` |
| `selection` | Eligibility gates, rarity/coverage policies, selection events | Keep quality gates separate from choice | Selection events | classification | validation, scoring | `api/internal/prose` |
| `sessions` | Append-only round graph and convergence verbs | Manage pin/reject/reroll/refine/commit/abandon | Sessions and rounds | orchestration | mutation, validation | `api/internal/prose`, `api/handlers/prose` |
| `declarations` | Consumer file scan, hashes, collisions, lifecycle states | Register file-authoritative consumers | Declaration projections | validation | mutation, query | `api/internal/prose`, `.vrooli/prose-studio` |
| `documents` | Outline candidates, section sessions, bounded context, assembly | Compose converged sections into documents | Documents and sections | composition-root | orchestration, aggregation | `api/internal/prose` |

The dependency direction is:

```text
textmetrics -> styles -> profiles -> generation -> selection -> sessions -> documents
                         ^
                         declarations
```

No domain calls a model vendor. Inference crosses the ai-gateway seam only.
No domain renders presentation bytes; assembly returns ordered text and
structure for document-manager's generation spine.

## Purpose Of This Document

This document keeps ownership, source paths, and surface responsibilities
explicit as the governed prose capability grows.

## Domain Details

Styles and profiles are versioned data; generation and measurement produce
auditable candidate records; sessions own convergence; documents own assembly.
Declarations are consumer-owned inputs and never override file authority.

## Shared Concepts

Every generated candidate carries deterministic measurements, provenance, a
machine-generated disclosure, and the effective context and sampling inputs.
The API and CLI expose the same Connect contract.

## Deferred Domains

Axis-space planning, acceptance calibration, and richer outline generation are
tracked as bounded P1/P2 follow-ups rather than implied implementation.

## Non-Domains

Model vendor clients, editorial taste, consumer claims, and host remediation
remain outside Prose Studio ownership.

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system boundaries and contracts
- [`FLOWS.md`](FLOWS.md) — lifecycle and convergence behavior
- [`DATA.md`](DATA.md) — persisted records and authority rules
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — injectable boundaries
