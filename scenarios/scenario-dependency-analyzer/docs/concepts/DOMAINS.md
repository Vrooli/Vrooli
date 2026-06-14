# Domains

## Purpose Of This Document

Map SDA bounded contexts and source ownership.

## Domain Inventory

| Domain | Purpose | Primary Archetype | Owns Data | Surfaces | Requirements | Source Paths |
|---|---|---|---|---|---|---|
| interfacegraph | Build actual scenario-to-scenario interface graph from upstream facts. | Fact interpretation / graph read model | No durable source facts; owns response assembly. | API, Connect, CLI, UI | SDA-P0-001, SDA-P0-002, SDA-P0-003 | `api/internal/interfacegraph/`, `api/internal/app/interface_graph_connect.go`, `cli/domains/graph/` |
| drift | Compare actual graph edges with declared scenario dependencies. | Validator / findings producer | Drift findings generated on demand. | API, CLI, test-genie | SDA-P0-004, SDA-P0-005 | `api/internal/interfacegraph/`, `cli/domains/drift/`, `scenarios/test-genie/` integration |
| deployment | Report deployment readiness, metadata gaps, bundle manifests, and DAG exports. | Reporting / planning | SQLite-backed reports and generated bundle metadata. | API, CLI, UI | SDA-P0-006, SDA-P1-001 | `api/internal/deployment/`, `cli/domains/deployment/`, `ui/src/features/deployment/` |
| catalog | List scenarios and summarize stored analysis detail. | Read model / catalog | Stored scenario analysis summaries. | API, CLI, UI | SDA-P0-007 | `api/internal/app/`, `cli/domains/scenarios/`, `ui/src/features/catalog/` |
| presentation | Expose graph, drift, deployment, and catalog capabilities to users and downstream systems. | Adapter / interface | No product data. | REST, Connect, CLI, UI | SDA-P0-008, SDA-P1-002 | `api/`, `cli/`, `ui/`, `packages/proto/schemas/scenario-dependency-analyzer/` |

## Domain Details

The interface graph domain is centered in `api/internal/interfacegraph`. Deployment logic is centered in `api/internal/deployment`.

## Shared Concepts

Scenario slug, evidence source, transport world, stability, drift severity, and declared dependency are shared across API, CLI, and generated proto contracts.

## Deferred Domains

AST extraction remains outside SDA until upstream fact services expose it.

## Non-Domains

SDA does not own proto parsing, language import extraction, or lifecycle process supervision.

## Cross-References

- `ARCHITECTURE.md`
- `FLOWS.md`
- `DATA.md`
- `../internal/SEAMS.md`
