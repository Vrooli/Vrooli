# Decisions — Go Code Graph

This document records durable decisions and tradeoffs future agents should not accidentally relitigate.

## Purpose Of This Document

Use this document when a choice:

- affects multiple files or future agents,
- rejects a plausible alternative,
- changes architecture, deployment, data, security, monetization, or testing direction,
- needs a revisit trigger.

Routine implementation notes belong in [`PROGRESS.md`](PROGRESS.md). Known unresolved issues belong in [`PROBLEMS.md`](PROBLEMS.md).

## Decision Log

| Date | Decision | Context | Consequences | Revisit Trigger |
|---|---|---|---|---|
| 2026-05-23 | Adopt the `react-vite` scenario documentation contract. | Scenario scaffold was generated from the template. | Docs start with stubs and maturity metadata in `docs/manifest.json`. | Revisit only if the scenario adopts a different template or doc contract. |
| 2026-05-23 | Wrap `golang.org/x/tools/go/packages` exclusively; no alternative Go parser. | This is the canonical Go AST + import resolver with full type-loading. Vendoring `go/parser` directly or using a lighter library (e.g. tree-sitter-go) would lose import resolution and type info. | Cartographer and other consumers get a typed, type-checked graph. Performance is bounded by `go/packages`'s full load mode (mitigated by the SLA in OT-P0-010). | Revisit only if `go/packages` becomes unmaintained or a benchmark proves a lighter library matches its fidelity. |
| 2026-05-23 | Shared Graph proto envelope at `packages/proto/schemas/common/v1/code_graph.proto`. | Both go-code-graph and (planned) typescript-code-graph emit the same Graph shape. Cartographer normalizes both into one internal model. Per-scenario protos would create adapter drift; a dedicated `code-graph/v1/` namespace would add a namespace nothing else uses; `go-code-graph` owning it and TS importing would create weird semantic coupling. | The shared envelope must stay language-agnostic. Language-specific `NodeKind` / `EdgeKind` values live in each scenario's `v1/` extensions. Coordinating envelope changes requires touching both scenarios. | Revisit if a third sibling scenario (Python, Rust) finds the envelope too restrictive, or if the language-specific extensions outgrow the shared model. |
| 2026-05-23 | Two-step Rewrite (plan + apply); never invoke git or build. | The operator owns git. Cartographer's build-green guardrail invokes `go build` at its layer; go-code-graph is a single-purpose mutator. The two-step shape gives consumers a preview surface and keeps the destructive action explicit. | Mid-apply crash leaves the working tree torn. Operator recovers via `git restore .`. The scenario does not implement rollback. | Revisit only if a consumer demonstrably needs atomic rollback that git can't provide cheaply. |
| 2026-05-23 | No internal extraction cache in v1. | `go/packages` is fast enough on Vrooli scenarios (≤2000 files <30s, per OT-P0-010). Cartographer caches at its layer. Adding an internal SQLite cache means cache-invalidation logic, which is harder to get right than re-parsing. | `Extract` re-parses every call. The `graph_hash` field (REQ-P1-005) lets callers detect "graph unchanged" without redoing their own diff. | Revisit if profiling shows the bottleneck for a real consumer is repeated identical extractions on hot paths. |
| 2026-05-23 | Per-path serialization, parallel across paths. | `go/packages.Load` is CPU-heavy. Two concurrent loads on the same path waste work; concurrent loads on different paths are safe and useful. | A per-path mutex serializes per `filepath.Abs(scenario_path)`. Apply also takes the mutex, so a Rewrite cannot race an Extract on the same path. | Revisit if concurrency contention becomes a measured bottleneck. |
| 2026-05-23 | Partial graph + structured warnings on parse failures. | Mid-migration scenarios are first-class inputs (cartographer's whole job is helping when things are broken). Hard-failing on any parse error would make the tool unusable on the scenarios that need it most. | `Extract` returns a graph plus `Warnings[]` for non-catastrophic failures. Catastrophic failures (no `go.mod`, unreadable path) return typed errors. | Revisit if consumers actually want strict mode; we'd add `--strict` flag as an opt-in, not flip the default. |
| 2026-05-23 | Single-module per `Extract` call; reject `go.work` workspaces. | Vrooli scenarios are single-module by convention. `go.work` adds resolution complexity that has no current consumer. | Workspace support is deferred to P2 (OT-P2-005). Inputs with `go.work` are rejected with a typed error pointing the operator at a specific module. | Revisit only if a Vrooli scenario actually adopts a workspace layout. |
| 2026-05-23 | Fixtures owned by go-code-graph. | Cartographer's `bas/fixtures/go-cycles/` currently holds Go source. That source belongs to *extraction* correctness (this scenario), not conflict-detection correctness (cartographer's scope). | go-code-graph's `bas/fixtures/` owns Go-source fixtures + `expected-graph.json`. Cartographer keeps only conflict-detection fixtures referencing graph outputs. Migration tracked under Task #10 ("Update cartographer references") at scenario init time. | Revisit only if a fixture has genuinely cross-scenario use that justifies duplication. |
| 2026-05-23 | UI scope: graph explorer + diagnostics in P0; fixture validator in P1. | "Scenarios always have UI" (per project memory). The most useful human surface for a parser scenario is a debug path (paste path → see graph) and a server-status view. Fixture validation directly accelerates the golden-file tests we depend on. | UI is intentionally not a workbench in v1. No "edit refactor plan in browser" surface; that path stays CLI-only. | Revisit when a consumer scenario asks for a richer UI surface. |

## Superseded Decisions

| Date | Superseded Decision | Replacement | Notes |
|---|---|---|---|
| None yet. | n/a | n/a | Add when a durable decision is replaced. |

## Cross-References

- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system decisions
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — dependency contracts
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved drift and debt
- [`PROGRESS.md`](PROGRESS.md) — completed work history
