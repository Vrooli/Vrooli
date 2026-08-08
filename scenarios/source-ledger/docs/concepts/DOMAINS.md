# Domains — Source Ledger

Source Ledger is the general append-only corpus capability. The journal is
the authority; derived forest, facet, recall, policy, and federation domains
are consumers or controls around that authority.

## Domain Inventory

| Domain | Responsibility | Owns data | Source Paths | Primary Archetype | Primary surfaces |
|---|---|---|---|---|---|
| journal | Append immutable source entries, provenance, retry state, and high-water marks. | `entries`, `journal_retry_queue`, `journal_high_water_mark`, `marks`, attachments. | `api/internal/journal/`, `packages/proto/schemas/source-ledger/v1/journal/` | mutation | `SL-P0-001`, `SL-P0-005`; append/list CLI and Connect service. |
| forest | Build and query the rebuildable compaction canopy, frontier, summaries, and tree edges. | `summaries`, `tree_edges`, derived embeddings used by compaction. | `api/internal/forest/`, `packages/proto/schemas/source-ledger/v1/forest/` | service | `SL-P0-003`; frontier, compact, and rebuild operations. |
| facets | Define scope vocabularies, retention policies, assignments, rules, pins, reviews, and supersession. | `facet_definitions`, `facet_policies`, `facet_assignments`, `classification_rules`, `pins`, `pin_reviews`, `merge_proposals`. | `api/internal/facets/`, `packages/proto/schemas/source-ledger/v1/facets/` | mutation | `SL-P1-001`; vocabulary and curation operations. |
| recall | Select bounded ambient wake and semantic recall results from journal and derived nodes. | Recall statistics and no corpus authority. | `api/internal/recall/`, `packages/proto/schemas/source-ledger/v1/recall/` | query | `SL-P0-002`, `SL-P1-005`; recall, wake, siblings, and search queries. |
| policy | Resolve scope configuration and enforce engine purity, budgets, and retention decisions. | `scopes` and policy cache state. | `api/internal/policy/`, `packages/proto/schemas/source-ledger/v1/policy/` | service | `SL-P0-002`, `SL-P0-004`; scope registry and request policy. |
| federation | Publish one search-hub provider per scope and route scope-labelled results. | Provider descriptors and registration state; no journal rows. | `api/internal/federation/`, `packages/proto/schemas/source-ledger/v1/federation/` | provider | `SL-P1-002`; provider registration and federated search. |
| health | Report database, maintenance, canopy, and dependency readiness. | No product data. | `api/handlers/health/`, `api/internal/health/` | aggregation | Lifecycle health and operator diagnostics. |

## Ownership Rules

- `journal` owns the sole durable corpus. No other domain edits or deletes a
  journal row.
- `forest` is a cache. Rebuild may discard and recreate derived rows without
  changing journal content.
- `facets` appends corrections and never mutates the original entry.
- `recall` and `federation` read through scope-aware interfaces and cannot
  select a scope implicitly after the request boundary.
- `policy` contains no named facet taxonomy. Vocabularies remain data.

## Shared Concepts

| Concept | Meaning | Owner |
|---|---|---|
| Scope | A named ledger partition with its own vocabulary and budgets. | `policy` registry. |
| Entry | Immutable source material with provenance and a stable identity. | `journal`. |
| Node | A recallable entry or derived summary. | `recall` reads; `forest` derives. |
| Frontier | Unabsorbed, policy-eligible nodes available for compaction. | `forest`. |

## Marketing Crew Scope

`team:marketing-crew` is the first non-agent adoption of Source Ledger. Its
vocabulary is deliberately source-shaped rather than agent-memory-shaped:

| Facet | Source file | Retention | Resident budget |
|---|---|---|---:|
| `handoff` | `handoff-history.jsonl` | expire on resolution | 8 |
| `knowledge` | `knowledge.jsonl` | retain | 8 |
| `audience-finding` | `audience-scans.jsonl` | retain | 8 |
| `campaign` | `campaign-drafts.jsonl` | retain | 8 |
| `decision` | `decisions.jsonl` | pin or review | 16 |
| `publication` | `published-scenario-mentions.jsonl` | retain | 4 |

The scope uses a frontier target of 32, a 256-line wake budget, and a
four-line per-entry wake excerpt. The adapter imports each JSONL line as the
immutable body, attaches `prompt-manager` source provenance and a content
hash, and can be replayed without inference work. `heartbeat-attempts.jsonl`
and empty publish telemetry logs remain outside the ledger because they are
operational telemetry, not durable team knowledge.

## Non-Domains

- `api/internal/server/` and transport modules are composition substrate.
- Harness adapters, prompt blocks, projections, and native capture remain in
  `vrooli-memory`; they are not ledger domains.
- UI components, test utilities, and generated contracts are shared seams.

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — dependency direction and invariants
- [`DATA.md`](DATA.md) — authority and rebuild contract
- [`FLOWS.md`](FLOWS.md) — append, recall, and compaction flows
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — dependency and failure contracts

## Architecture Maturity

The domain map is contract-complete for the extraction boundary. The health
surface is scenario-local; the named ledger domains remain engine-pure and
are not yet implemented in this scaffold.

## Contracts And Data Flow

Each domain owns its data and boundary operations while sharing the explicit
scope contract and immutable journal identity.

## Purpose Of This Document

This document records bounded contexts before the engine packages move.

## Domain Details

Each row above names the data owner, archetype, source boundary, and intended
programmatic surface for a domain.

## Deferred Domains

Harness import, projection, prompt, and capture domains remain owned by
`vrooli-memory` and are deliberately outside this scenario.

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md)
- [`FLOWS.md`](FLOWS.md)
- [`DATA.md`](DATA.md)
- [`../internal/SEAMS.md`](../internal/SEAMS.md)
