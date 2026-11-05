# Domains

## Purpose Of This Document

This document names the real Code Facts bounded contexts before implementation.

## Domain Inventory

| Domain | Responsibility | Surface(s) | Primary Archetype | Secondary Traits | Source Paths | Notes |
|---|---|---|---|---|---|---|
| targets | Turns caller input into a bounded code target. | API, CLI, UI | Policy/rules | Filesystem metadata | `api/internal/targets` | Resolves path/scenario/module/project/package inputs. |
| catalog | Owns governed source identity, roles, scopes, hashes, and generations. | API, CLI, UI | Persistence/workflow | Corpus discovery | `api/internal/catalog` | SQLite is the authority for every derived retrieval leg. |
| analysis | Coordinates graph providers and normalized projections. | API | Integration/client | Graceful degradation | `api/internal/analysis` | Brokers analyzers; never parses supported languages itself. |
| retrieval | Plans exact, semantic, relationship, and contract queries. | API, CLI, UI | Reporting/query | Fusion and ranking | `api/internal/retrieval` | Owns FTS, vector seams, fusion, freshness fences, and explanations. |
| proof | Synthesizes explicit contract and relationship evidence. | API, CLI, UI | Policy/rules | Evidence synthesis | `api/internal/proof` | Keeps proof status independent from retrieval relevance. |
| indexcontrol | Owns reconciliation jobs and generation lifecycle. | API, CLI, UI | Workflow | Authorization and cancellation | `api/internal/indexcontrol` | Reconcile, reindex, cancel, promote, rollback, and cleanup. |
| cache | Provides bounded derived-result lifecycle and diagnostics. | API, CLI, UI | Configuration/settings | Reporting/query | `api/internal/cache` | Owns TTL, quotas, orphan collection, and no-write-on-hit policy. |

## Domain Details

### targets

Owns target kind parsing, repo-root detection, canonical path resolution, scenario-context detection, and bounded-target errors.

### catalog

Owns repository-aware discovery, source roles, stable identities, active and shadow generations, and the normalized persistence model used by retrieval and proof projections.

### analysis

Owns provider routing and failure mapping. It calls graph providers but does not interpret proto-health or endpoint policy.

### retrieval

Owns query regimes, scoped lexical and semantic legs, deterministic fusion, reranking, freshness fences, and result explanations.

### proof

Owns evidence synthesis for proto adoption and endpoint proofs. It consumes normalized facts and emits explicit evidence statuses.

### indexcontrol

Owns durable jobs, progress, cancellation, shadow promotion, rollback, and cleanup. Watcher events are latency hints; catalog reconciliation is the correctness authority.

### cache

Owns deterministic key construction, hit/miss/stale diagnostics, storage seam, and operator cache commands.

## Shared Concepts

- Code target: caller-supplied bounded input.
- Parse unit: concrete Go module or TypeScript project sent to a graph provider.
- Fact family: requested subset of facts/proofs.
- Evidence: status plus source/provenance.

## Evidence Orchestration Package

`api/internal/facts` owns analyzer-provider orchestration and the typed Describe
surface. Public Search is only a boundary here: production composition injects
the active-generation backend from `handlers/facts/production_index.go`. The
former streaming repository scan, resident project index, and comparison
benchmark have been removed.

## Non-Domains

- `notes` is generated template residue.
- Graph parsing belongs to provider scenarios.
- Proto validation policy belongs to `proto-health`.

## Cross-References

- [ARCHITECTURE.md](ARCHITECTURE.md)
- [FLOWS.md](FLOWS.md)
- [DATA.md](DATA.md)
- [../internal/SEAMS.md](../internal/SEAMS.md)
