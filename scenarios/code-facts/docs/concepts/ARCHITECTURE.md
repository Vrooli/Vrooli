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

```text
request target
  -> target resolver
  -> surface inventory + parse-unit discovery
  -> analyzer broker
  -> fact normalizer
  -> framework endpoint adapters
  -> proof synthesizer
  -> cache metadata
  -> CodeFactsReport
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

## Extension Rules

- Add new fact families as explicit enum/contract values.
- Add new analyzer providers behind the analyzer seam.
- Add new endpoint framework coverage through Code Facts adapters backed by
  generic graph facts.
- Add new proof families in the proof domain, backed by generic facts.
- Keep provider-specific language details out of shared Code Facts evidence unless normalized.

## Architecture Maturity

| Surface | Level | Evidence | Remaining Drift |
|---|---:|---|---|
| Docs | 2 | Domain map and seams documented in Phase 5 | Later phases need resolver/analyzer details |
| API | 3 | `CodeFactsService` exposes describe, proof, cache, target resolution, surface inventory, parse-unit discovery, analyzer brokering, cache reuse, proto adoption proof, and REST endpoint proof | CLI/UI proof and widget proof families remain later work |
| CLI | 3 | `facts` and `cache` command groups call generated Connect clients, including cache status/inspect/clear | More human summaries land with proof data |
| UI | 2 | `/facts` workbench reads the describe report and displays cache state/hash metadata | Phase 11 deepens inspection/filtering |

## Intentional Deviations

Analyzer-backed generic language evidence is active as of Phase 8 for imports, symbols, references, calls, and provider warnings. Cache reuse is active as of Phase 9 for graph and report payloads with source/config hash evidence. Phase 10 interprets those generic facts into proto adoption and REST endpoint proof evidence while leaving CLI proof and UI widget proof for later phases.

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
