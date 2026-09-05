# Domains

## Purpose Of This Document

Map SDA bounded contexts and source ownership.

## Domain Inventory

| Domain | Purpose | Primary Archetype | Owns Data | Surfaces | Requirements | Source Paths |
|---|---|---|---|---|---|---|
| interfacegraph | Build actual scenario-to-scenario interface graph from upstream facts. | Fact interpretation / graph read model | No durable source facts; owns response assembly. | API, Connect, CLI, UI | SDA-P0-001, SDA-P0-002, SDA-P0-003 | `api/internal/interfacegraph/`, `api/internal/graph/`, `cli/domains/graph/` |
| drift | Compare actual graph edges with declared scenario dependencies. | Validator / findings producer | Drift findings generated on demand. | API, CLI, test-genie | SDA-P0-004, SDA-P0-005 | `api/internal/interfacegraph/`, `cli/domains/drift/`, `scenarios/test-genie/` integration |
| deployment | Report deployment readiness, metadata gaps, bundle manifests, and DAG exports. | Reporting / planning | SQLite-backed reports and generated bundle metadata. | API, CLI, UI | SDA-P0-006, SDA-P1-001 | `api/internal/deployment/`, `cli/domains/deployment/`, `ui/src/features/deployment/` |
| catalog | List scenarios and summarize stored analysis detail. | Read model / catalog | Stored scenario analysis summaries. | API, CLI, UI | SDA-P0-007 | `api/internal/catalog/`, `api/internal/app/`, `cli/domains/scenarios/`, `ui/src/features/catalog/` |
| analysis | Analyze one scenario or the fleet and optionally scan/apply dependency diffs. | Orchestration / compatibility adapter | Persists through backing dependency services. | API, UI, CLI | SDA-P0-001, SDA-P0-005 | `api/internal/analysis/`, `api/internal/app/service_registry.go`, `ui/src/features/catalog/` |
| dependencies | Read stored dependencies and dependency impact reports. | Read model / impact analysis | Stored dependency edges. | API, UI | SDA-P0-001, SDA-P0-005 | `api/internal/dependencies/`, `api/internal/app/dependency_service.go`, `ui/src/features/graph/` |
| graph | Generate legacy dependency graphs, centrality reports, and cycle findings from stored dependency data. | Reporting / graph analytics | No durable data; computes from dependency store reports. | API, UI | SDA-P0-001, SDA-P1-002 | `api/internal/graph/`, `api/internal/app/service_graph.go`, `ui/src/features/graph/` |
| optimization | Generate dependency optimization recommendations. | Recommendation workflow | Stored recommendations when applied by backing services. | API, CLI | SDA-P1-003 | `api/internal/optimization/`, `api/internal/app/optimization/`, `api/internal/app/service_registry.go` |
| proposal | Analyze proposed scenarios before they exist on disk. | Planning / advisory | None. | API | SDA-P1-004 | `api/internal/proposal/`, `api/internal/app/proposal.go` |
| coreset | Report Baseline Modes core/trusted-base dependency set. | Fresh-from-disk report | None. | API | SDA-P1-002 | `api/internal/coreset/` |
| aisearch | Federate SDA's three search leaves into Search Hub on one shared multi-corpus engine: `.dependencies` (governance records, backs `SearchApprovedDependencies` ranking), `.scenarios` (connection graph — depends-on/used-by, `SearchInterfaceGraph`), `.resources` (resource→consuming-scenarios from a fleet `service.json` scan, `SearchResourceUsage`). One Reconciler/sync loop/embedder, one Qdrant collection per corpus, plus the token-gated `SearchControlService` control plane (reindex + tuning/corpus write-back). | Federated read model / semantic search | No durable source data; indexes derived facts into Qdrant. | Connect, CLI, Search Hub | SDA-P0-001, SDA-P0-007 | `api/internal/aisearch/`, `api/internal/resourceusage/`, `api/internal/searchcontrol/`, `cli/domains/{graph,resources}/`, `.vrooli/search.json` |
| presentation | Expose graph, drift, deployment, catalog, analysis, dependency, proposal, optimization, and core-set capabilities to users and downstream systems. | Adapter / interface | No product data. | REST, Connect, CLI, UI | SDA-P0-008, SDA-P1-002 | `api/`, `cli/`, `ui/`, `packages/proto/schemas/scenario-dependency-analyzer/` |

## Domain Details

The interface graph domain is centered in `api/internal/interfacegraph`, with REST compatibility and Connect presentation for graph/drift routes in `api/internal/graph`. Legacy dependency graph construction, centrality analytics, and cycle detection are also graph-domain-owned. Deployment logic and REST presentation are centered in `api/internal/deployment`. Catalog REST presentation is owned by `api/internal/catalog`; analysis/scan, dependencies/impact, proposal, optimization, and core-set REST presentation are now owned by matching domain packages. Several backing service implementations still live in focused `api/internal/app/service_*.go` files until their workspace/store/runtime dependencies can move cleanly into domain packages.

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
