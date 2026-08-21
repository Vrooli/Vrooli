# Architecture

## Purpose Of This Document

This document defines the durable Code Facts architecture before implementation. Code Facts is a broker and evidence synthesizer; it is not a language parser.

## Scenario Shape

Code Facts has three surfaces:

- API: Connect-RPC service for target resolution, describe queries, proof queries, and cache diagnostics.
- CLI: thin agent/operator wrapper over the same Connect-RPC operations.
- UI: operator workbench for target entry, surface/fact inspection, warnings, evidence, and cache status.

## System Boundaries

In bounds:

- Resolve bounded targets.
- Read shallow Vrooli metadata such as `.vrooli/service.json`, `.vrooli/endpoints.json`, `cli/manifest.json`, and `.vrooli/testing.json`.
- Call `go-code-graph` and `typescript-code-graph` through provider seams.
- Normalize provider output into Code Facts fact families.
- Synthesize higher-level evidence such as proto adoption and endpoint proof.

Out of bounds:

- Parsing Go or TypeScript source directly.
- Owning proto-health policy.
- Mutating target source code.
- Running unbounded monorepo analysis.

## Contracts And Data Flow

```mermaid
flowchart TB
    H[handlers] --> T[targets]
    H --> R[retrieval]
    H --> IC[indexcontrol]
    H --> P[proof]
    C[composition root] --> T
    C --> CA[catalog]
    C --> A[analysis]
    C --> R
    C --> IC
    C --> P
    CA --> DB[(SQLite catalog and FTS)]
    R --> DB
    R --> VS[vector-store seam]
    A --> GP[indexed graph projections]
    P --> GP
```

Fact-family filtering is applied at the Code Facts service boundary. Provider-level extraction may still be full graph extraction until providers add selective extraction.

Endpoint proof is intentionally two-stage. Graph providers emit generic route,
import, reference, call, and type facts. Code Facts framework adapters then
interpret only the generic attributes they support and produce normalized
endpoint implementation evidence. The final proof step compares that evidence
to `.vrooli/endpoints.json`; downstream consumers such as proto-health consume
only the resulting proof statuses and evidence.

## Shared Infrastructure

| Area | Purpose | Expected Shape |
|---|---|---|
| Provider clients | Hide graph-provider transport behind fakes | API internal analyzer seams |
| Target filesystem | Resolve paths and metadata deterministically | API internal target seams |
| Cache store | Store graph/report entries, hash evidence, and diagnostics | API internal cache seams with SQLite production repository and in-memory tests |
| Rendering | Keep CLI output thin and proto-shaped for JSON | CLI domain renderers |

## Zone Map

| Directory | Zone | May import | Enforcement |
|---|---|---|---|
| `api/handlers/facts/` | Transport edge | generated proto, Connect, domain contracts, module/httpx substrate | architecture tests and endpoint coverage |
| `api/handlers/health/` | Transport edge | health contract and substrate | architecture tests and endpoint coverage |
| `api/internal/targets/` | Domain core | standard library only | transport, sibling-domain, and ambient-dependency gate |
| `api/internal/catalog/` | Domain core and persistence contract | standard library and `database/sql` only in adapter files | transport, sibling-domain, and ambient-dependency gate |
| `api/internal/analysis/` | Domain core | standard library only | transport, sibling-domain, and ambient-dependency gate |
| `api/internal/retrieval/` | Domain core | standard library only | transport, sibling-domain, and ambient-dependency gate |
| `api/internal/proof/` | Domain core | standard library only | transport, sibling-domain, and ambient-dependency gate |
| `api/internal/indexcontrol/` | Domain core | standard library only | transport, sibling-domain, and ambient-dependency gate |
| `api/internal/cache/` | Domain core and persistence contract | standard library and `database/sql` only in adapter files | transport, sibling-domain, and ambient-dependency gate |
| `api/internal/facts/` | Transitional orchestration | legacy providers and cache implementation | retirement is gated by vertical replacement tests in Phases 4-9 |
| `api/internal/database/` | Cross-cutting substrate | database drivers and standard library | package tests |
| `api/internal/logging/` | Cross-cutting substrate | standard library only | compile-time interface use |
| `api/internal/httpc/` | Cross-cutting substrate | `net/http` | package tests |
| `api/internal/httpx/` | Cross-cutting substrate | `net/http` | package tests |
| `api/internal/middleware/` | Cross-cutting substrate | HTTP and logging | package tests |
| `api/internal/module/` | Transport substrate | Gorilla mux and endpoint descriptors | module tests |
| `api/internal/modules/` | Composition registry | handler modules | registry tests |
| `api/internal/registration/` | Search Hub integration adapter | HTTP, Connect, generated registry client | integration tests |
| `api/internal/server/` | Composition root | all mounted modules and transport substrate | server tests |
| `api/internal/testutil/` | Test-only substrate | domain contracts and deterministic fakes | no-production-import gate |
| `api/main.go` | Composition root | all concrete adapters | lifecycle and server tests |
| `api/cmd/` | Operator/build tooling | domain and substrate packages | command tests |
| `cli/` | Transport edge | generated clients and renderers | manifest/service coverage tests |
| `ui/src/api/` | UI transport edge | generated web clients | UI unit tests |
| `ui/src/features/` | UI capability domains | UI types and shared components | UI unit and selector tests |

`api/internal/facts/` is not a destination domain. It is the legacy
orchestration package whose vertical slices are removed as `targets`, `catalog`,
`analysis`, `retrieval`, `proof`, `indexcontrol`, and `cache` gain production
implementations. New storage, retrieval, or control behavior must land in the
owning domain.

## Boundary Maturity

| Zone | Level | Evidence | Remaining drift |
|---|---:|---|---|
| New API domains | 5 | `TestDomainPackagesAreTransportFreeAndIndependent` and seam reconciliation | Production adapters arrive with their owning phases |
| API transport/substrate | 4 | Existing handler, module, server, and no-production-test-import tests | `facts` still hosts legacy orchestration and provider transports |
| CLI | 3 | Manifest-driven generated Connect client | Index-control commands are not implemented yet |
| UI | 3 | Feature folders and shared API client | Evidence workspace domains and journeys are not implemented yet |

## Extension Rules

- Add new fact families as explicit enum/contract values.
- Add new analyzer providers behind the analyzer seam.
- Add new endpoint framework coverage through Code Facts adapters backed by
  generic graph facts.
- Add new proof families in the proof domain, backed by generic facts.
- Keep provider-specific language details out of shared Code Facts evidence unless normalized.
- Keep opinion-bearing architecture policy out of Code Facts. The `file_domain`
  fact family is exposed here as a cache/query contract, but Architecture
  Cartographer produces the verdicts because it owns domain-map authority,
  signal aggregation, and confidence thresholds.

## Architecture Maturity

| Surface | Level | Evidence | Remaining Drift |
|---|---:|---|---|
| Docs | 2 | Domain map and seams documented in Phase 5 | Later phases need resolver/analyzer details |
| API | 3 | `CodeFactsService` exposes describe, proof, cache, target resolution, surface inventory, parse-unit discovery, analyzer brokering, cache reuse, proto adoption proof, and REST endpoint proof | CLI/UI proof and widget proof families remain later work |
| CLI | 3 | `facts` and `cache` command groups call generated Connect clients, including cache status/inspect/clear | More human summaries land with proof data |
| UI | 2 | `/facts` workbench reads the describe report and displays cache state/hash metadata | Phase 11 deepens inspection/filtering |

## Intentional Deviations

Analyzer-backed generic language evidence is active as of Phase 8 for imports, symbols, references, calls, and provider warnings. Cache reuse is active as of Phase 9 for graph and report payloads with source/config hash evidence. Phase 10 interprets those generic facts into proto adoption and REST endpoint proof evidence while leaving CLI proof and UI widget proof for later phases. Architecture Cartographer's Phase 6 `file_domain` family is query-backed through cartographer: Code Facts delegates verdict production to cartographer, normalizes the returned verdicts into `GenericFact`s, and emits typed unsupported evidence when no cartographer provider is configured.

## Documentation Architecture

- [DOMAINS.md](DOMAINS.md) owns bounded contexts.
- [FLOWS.md](FLOWS.md) will own lifecycle/state flows once cache and describe jobs exist.
- [DATA.md](DATA.md) owns fact/cache data shape.
- [INTEGRATIONS.md](INTEGRATIONS.md) owns provider and consumer relationships.
- [../internal/SEAMS.md](../internal/SEAMS.md) owns test seams.
- [../internal/TESTING.md](../internal/TESTING.md) owns validation strategy.

## Cross-References

- [Fact Families](../reference/fact-families.md)
- [Evidence Status](../reference/evidence-status.md)
- [Cache](../reference/cache.md)
