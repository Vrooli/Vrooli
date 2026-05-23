# Decisions — TypeScript Code Graph

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
| 2026-05-23 | Wrap `ts-morph` exclusively; no alternative TS parser. | `ts-morph` is the canonical TypeScript compiler-API wrapper with formatting-preserving refactor APIs. A raw `typescript` SDK call would lose the refactor surface. Lighter parsers (tree-sitter-typescript) lose type info and import resolution. | Two consumers (cartographer + react-component-library) get a typed, type-checked graph with leading-comment fidelity. Performance is bounded by `ts-morph`'s Project initialization cost (mitigated by the SLA in OT-P0-012). | Revisit only if `ts-morph` becomes unmaintained or a benchmark proves a Go-native TS parser matches its fidelity (see OT-P2-006). |
| 2026-05-23 | Run `ts-morph` in a Node sidecar process, not in-process. | The API is Go; `ts-morph` is a Node library. Options were: (a) Node sidecar over IPC, (b) embed Node via cgo, (c) write a Go-native TS parser. (b) is fragile and platform-dependent; (c) loses `ts-morph` parity. The sidecar isolates Node's lifecycle from the API and lets the API restart it cleanly on crash. | Adds sidecar lifecycle complexity (REQ-P0-009) and IPC overhead. The sidecar mechanics are hidden behind Connect-RPC so consumers never see Node. Performance includes one IPC round-trip per call. | Revisit when a Go-native TS parser with `ts-morph` parity becomes viable (OT-P2-006). |
| 2026-05-23 | Shared Graph proto envelope at `packages/proto/schemas/common/v1/code_graph.proto`. | Both typescript-code-graph and go-code-graph emit the same Graph shape. Cartographer normalizes both into one internal model. Per-scenario protos would create adapter drift; a dedicated `code-graph/v1/` namespace would add a namespace nothing else uses; `go-code-graph` owning it and TS importing would create weird semantic coupling. | The shared envelope must stay language-agnostic. TS-specific `NodeKind` / `EdgeKind` values live in `typescript-code-graph/v1/` extensions. Coordinating envelope changes requires touching both scenarios. | Revisit if a third sibling scenario finds the envelope too restrictive. |
| 2026-05-23 | Leading-comment metadata is a load-bearing contract from day one. | `react-component-library` currently uses regex to scrape `@vrooliWidget*` and `@vrooliComponent*` JSDoc tags from TS sources. The migration off regex onto a typed graph requires the graph to carry leading comments verbatim. If we ship without comments, rcl can't migrate. | Every declaration node has a `leading_comments: string[]` field. The `bas/fixtures/ts-jsdoc-tags/` fixture pins the contract. Removing comments later would be a breaking change for rcl. | Treat as permanent for v1. Revisit only if a structured tag parser is added as a separate field (in which case raw comments remain for fidelity). |
| 2026-05-23 | Two-step Rewrite (plan + apply); never invoke git or build. | The operator owns git. Cartographer's build-green guardrail invokes `tsc` at its layer; typescript-code-graph is a single-purpose mutator. The two-step shape gives consumers a preview surface. `ts-morph` preserves formatting on save, which solves one of the most common refactor pains. | Mid-apply crash leaves the working tree torn. Operator recovers via `git restore .`. The scenario does not implement rollback. | Revisit only if a consumer demonstrably needs atomic rollback that git can't provide cheaply. |
| 2026-05-23 | No internal extraction cache in v1. | `ts-morph` is fast enough on Vrooli scenarios (≤2000 files <30s, per OT-P0-012). Cartographer caches at its layer. Caching includes invalidation logic, harder to get right than re-parsing. The Node sidecar's Project initialization is a one-time cost per call. | `Extract` re-parses every call. The `graph_hash` field (REQ-P1-005) lets callers detect "graph unchanged" without redoing their own diff. | Revisit if profiling shows the bottleneck for a real consumer is repeated identical extractions on hot paths. |
| 2026-05-23 | Per-path serialization at both Go and sidecar layers. | `ts-morph` Project state is not safe to share across parallel invocations against the same project. The Go-side mutex prevents same-path race; the sidecar-side mutex prevents internal sidecar state corruption if the Go-side mutex is ever bypassed. Two-layer enforcement gives defense in depth. | Two locks per same-path call. Negligible overhead in practice. Different paths still run in parallel. | Revisit only if contention becomes a measured bottleneck. |
| 2026-05-23 | Partial graph + structured warnings on parse failures. | Mid-migration scenarios are first-class inputs (cartographer + rcl both depend on parsing broken projects). Hard-failing on any parse error would make the tool unusable on the scenarios that need it most. | `Extract` returns a graph plus `Warnings[]`. Catastrophic failures (no `tsconfig.json`, unreadable path, sidecar dead) return typed errors. | Revisit if consumers actually want strict mode; add `--strict` flag as an opt-in. |
| 2026-05-23 | Single-project per `Extract` call; reject pnpm workspaces. | Vrooli scenarios are single-project by convention. Multi-project workspaces add resolution complexity that has no current consumer. | Workspace support is deferred to P2 (OT-P2-005). Inputs in a pnpm workspace ancestor are rejected with a typed error pointing at a specific project. | Revisit only if a Vrooli scenario actually adopts a workspace layout. |
| 2026-05-23 | Fixtures owned by typescript-code-graph. | Cartographer's `bas/fixtures/` would need TS source. That source belongs to *extraction* correctness (this scenario), not conflict-detection correctness (cartographer's scope). | typescript-code-graph's `bas/fixtures/` owns TS-source fixtures + `expected-graph.json`, including the load-bearing `ts-jsdoc-tags/` fixture for the leading-comment contract. | Revisit only if a fixture has genuinely cross-scenario use that justifies duplication. |
| 2026-05-23 | UI scope: graph explorer + diagnostics (including sidecar status panel) in P0; fixture validator in P1. | "Scenarios always have UI" (per project memory). The most useful human surface is a debug path (paste path → see graph + comments) and a server-status view, including prominent sidecar health because that's the typical failure mode. Fixture validation directly accelerates the golden-file tests. | UI is intentionally not a workbench in v1. The sidecar status panel is always visible on the diagnostics page. No "edit refactor plan in browser" surface. | Revisit when a consumer scenario asks for a richer UI surface. |

## Superseded Decisions

| Date | Superseded Decision | Replacement | Notes |
|---|---|---|---|
| None yet. | n/a | n/a | Add when a durable decision is replaced. |

## Cross-References

- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system decisions
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — dependency contracts
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved drift and debt
- [`PROGRESS.md`](PROGRESS.md) — completed work history
