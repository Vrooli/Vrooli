# Domains

## Purpose Of This Document

This document names the real Code Facts bounded contexts before implementation.

## Domain Inventory

| Domain | Responsibility | Surface(s) | Primary Archetype | Secondary Traits | Source Paths | Notes |
|---|---|---|---|---|---|---|
| target | Turns caller input into a bounded code target. | API, CLI, UI | Policy/rules | Filesystem metadata | `api/internal/target`, `cli/domains/target`, `ui/src/features/target` | Resolves path/scenario/module/project inputs. |
| surface | Discovers the analyzable surfaces and parse units of a target. | API, CLI, UI | Reporting/query | Scenario metadata | `api/internal/surface`, `ui/src/features/surfaces` | Inventories scenario surfaces and parse units. |
| analyzer | Coordinates graph-provider calls and their failure semantics. | API | Integration/client | Graceful degradation | `api/internal/analyzer` | Brokers graph providers through seams. |
| facts | Normalizes graph output into requested fact families. | API, CLI, UI | Reporting/query | Normalization | `api/internal/facts`, `ui/src/features/facts` | Normalizes graph output and filters fact families. |
| proof | Synthesizes explicit adoption and endpoint evidence. | API, CLI, UI | Policy/rules | Evidence synthesis | `api/internal/proof`, `cli/domains/proof` | Produces proto adoption and endpoint proof evidence. |
| cache | Provides deterministic fact-cache lifecycle and diagnostics. | API, CLI, UI | Configuration/settings | Reporting/query | `api/internal/cache`, `cli/domains/cache`, `ui/src/features/cache` | Owns cache keys, invalidation, and diagnostics. |

## Domain Details

### target

Owns target kind parsing, repo-root detection, canonical path resolution, scenario-context detection, and bounded-target errors.

### surface

Owns scenario metadata reads, API/CLI/UI/sidecar surface inventory, endpoint and CLI metadata references, and parse-unit discovery.

### analyzer

Owns provider routing and failure mapping. It calls graph providers but does not interpret proto-health or endpoint policy.

### facts

Owns normalized fact families: surfaces, parse units, imports, symbols, references, calls, type usages, warnings, and provider provenance.

### proof

Owns evidence synthesis for proto adoption and endpoint proofs. It consumes normalized facts and emits explicit evidence statuses.

### cache

Owns deterministic key construction, hit/miss/stale diagnostics, storage seam, and operator cache commands.

## Shared Concepts

- Code target: caller-supplied bounded input.
- Parse unit: concrete Go module or TypeScript project sent to a graph provider.
- Fact family: requested subset of facts/proofs.
- Evidence: status plus source/provenance.

## Deferred Domains

- CLI proof and UI widget proof are P1 until consumers are ready.
- Cross-provider snapshot diffing is P1.
- Additional language providers are P2.

## Non-Domains

- `notes` is generated template residue.
- Graph parsing belongs to provider scenarios.
- Proto validation policy belongs to `proto-health`.

## Cross-References

- [ARCHITECTURE.md](ARCHITECTURE.md)
- [FLOWS.md](FLOWS.md)
- [DATA.md](DATA.md)
- [../internal/SEAMS.md](../internal/SEAMS.md)
