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

## Debt discovery ownership

Code Facts is the shared source and graph substrate. Consumers request bounded
facts or language graphs through its broker and reuse its source identity,
provider admission, in-flight coalescing, and cache. Consumers retain their own
policy and durable findings; they do not discover parse units or call the same
language parser through a parallel client.

| Owner | Computation retained | Consolidation boundary |
| --- | --- | --- |
| code-facts | Target resolution, source inventory, parse-unit discovery, graph brokering, fact projection, freshness and caching | Sole entry point for source facts and language graphs |
| go-code-graph | Go syntax/type analysis and mechanical Go rewrites | Parser provider behind Code Facts; public rewrite operations remain |
| typescript-code-graph | TypeScript compiler analysis, comments, mechanical rewrites and sidecar supervision | Parser provider behind Code Facts; public rewrite operations remain |
| architecture-cartographer | Domain inference, coupling signals, architectural conflicts, immutable audit snapshots | Consume Code Facts graphs; remove private language clients and source scanning |
| tidiness-manager | Maintainability detectors, calibrated reduction estimates, ranked opportunities and cleanup campaigns | One detector registry and recommendation ranking over shared facts |
| scenario-auditor | Standards-rule execution and managed fix operations | Publish rule findings to the detector registry; remove duplicate discovery and ranking |
| flow-verifier | Inferred and declared lifecycle contracts, transition verification and counterexamples | Supply lifecycle detectors and conversion evidence |
| template-manager | Template provenance, inherited drift/debt and governed migrations | Supply inherited-debt detectors and migration evidence |

The first migration exposes Code Facts' cached graph results with their parse
unit, source identity and extraction contract, then switches Cartographer to
that surface. Historical Cartographer snapshots remain its domain evidence.
Whole-scenario retirement is a separate operator decision; removing redundant
internal scanners and clients does not retire a scenario's public capability.

## Cross-References

- [ARCHITECTURE.md](ARCHITECTURE.md)
- [FLOWS.md](FLOWS.md)
- [DATA.md](DATA.md)
- [../internal/SEAMS.md](../internal/SEAMS.md)
