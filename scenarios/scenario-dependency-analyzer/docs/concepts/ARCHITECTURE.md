# Scenario Dependency Analyzer Architecture

SDA is evolving from a declared-dependency reporter into Vrooli's actual cross-scenario interface graph authority.

## Role

- **Ecosystem role**: Meta / interface-enabler.
- **Primary consumers**: Operators through the UI/CLI; `test-genie`, `tech-tree-designer`, and future deployment planners through the programmatic surface.
- **Stable seam**: `DescribeInterfaceGraph`, a Connect RPC that returns evidence-tagged scenario edges.

## Boundary Of Responsibility

| Capability | Owner | SDA dependency |
|---|---|---|
| Protobuf surface extraction | `proto-health` | Batch proto surface RPC |
| Language-level import extraction | `code-facts` | Batch imports fact RPC |
| Import/proto path attribution to scenario slugs | SDA | Fleet naming conventions |
| Declared-vs-actual drift interpretation | SDA | `service.json` + actual edge set |
| Runtime URL and CLI shell-out facts | Future AST facts | Follow-up plan |

SDA must not scan source files for actual graph evidence. Its responsibility is interpretation, not extraction.

## Actual Graph Model

The actual graph is assembled from upstream facts and served from a derived cache when fresh:

1. Fetch batch proto surfaces from `proto-health`.
2. Fetch batch import facts from `code-facts`.
3. Attribute import paths to scenario slugs.
4. Merge evidence into source-to-target scenario edges.
5. Persist the derived graph by request signature for fast repeat reads.
6. Compare actual edges with declared scenario dependencies.

Edges carry:

- `evidence[]`: `proto_import`, `go_import`, or future AST fact kinds.
- `transport_world`: forwarded from proto surface facts where available.
- `stability`: forwarded from proto surface facts where available.

The derived graph cache is persisted in SQLite for repeat-read performance. It is rebuildable from upstream facts and must not become the source of truth for proto surfaces or language imports.

## Drift Semantics

| Drift kind | Meaning | Severity |
|---|---|---|
| `undeclared_but_used` | Actual import evidence targets a scenario not declared in `service.json` | `WARNING` |
| `declared_without_import_evidence` | `service.json` declares a scenario but no import evidence exists | `INFO` |

The second case is intentionally advisory because runtime discovery calls and CLI shell-outs are deferred to AST fact providers.

## Storage

SDA uses SQLite for local, history-bearing data. Schemas are domain-owned (`api/internal/<domain>/schema.sql` + `schema.go`) and registered at boot through the shared database schema helper. The actual interface graph is recomputed from upstream facts instead of stored.

## API Shape

Existing Gin REST endpoints remain during the hybrid migration for UI/operator compatibility. New scenario-to-scenario consumers use the Connect surface.

The CLI must expose machine-readable access to the same graph and drift data, with `graph actual --json` and `drift --json` as the intended operator surface.
