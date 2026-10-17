# Product Requirements Document (PRD)

## 🎯 Overview

Purpose: TypeScript Code Graph extracts a deterministic, reproducible graph of files, modules, declarations, imports, references, calls, JSX usage, exports, and import bindings from any TypeScript project and executes mechanical TypeScript refactors (file moves and import rewrites) on behalf of consumer scenarios. It wraps `ts-morph` so that no other Vrooli scenario has to parse TypeScript source itself. Each declaration node also carries its verbatim leading JSDoc/comment block as structured metadata, so consumers that previously relied on regex tag scraping (most notably `react-component-library`) can migrate onto a typed contract without losing fidelity.

Target users: Vrooli scenario maintainers and migration agents working through architecture-cartographer; `react-component-library` as the second declared consumer (planned migration off regex parsing); future TS-static-analysis scenarios that need a typed graph instead of grep-based heuristics; template authors evaluating screaming-architecture compliance for TS scenarios.

Deployment surfaces: Connect-RPC API (primary contract for programmatic consumers), Go CLI (primary surface for agents and direct invocation), React+Vite UI (graph explorer for debugging, server diagnostics, fixture validation), plus the lifecycle integration provided by the react-vite template. The TS parser library (`ts-morph`) runs as a Node sidecar process spawned by the Go API; see Tech Direction for the rationale.

Value proposition: Replace ad-hoc regex tag scraping and brittle handwritten TS inspection with a single typed graph contract and a single mechanical refactor surface. Two consumers exist on day one (cartographer + react-component-library), which justifies a standalone scenario instead of inlined parsing. The graph is deterministic enough to be checked into version control as a baseline; the refactors are scoped enough that operators retain full control via git.

## 🎯 Operational Targets

Operational targets are measurable outcomes; checkboxes may auto-update based on validation.

### 🔴 P0 – Must ship for viability

- [ ] OT-P0-001 | Deterministic Graph Extraction | Implement `Extract(ExtractRequest{project_path}) → Graph` against `ts-morph` using a fixed Project configuration that returns files, modules, declarations (components, hooks, classes, interfaces, types, functions, vars, consts), import edges, import bindings, references, calls, JSX usage, and export summary facts; byte-stable serialization for identical inputs. `project_path` may point at a project root directory or an explicit `tsconfig.json`.
- [ ] OT-P0-002 | Shared Code-Graph Proto Envelope | Consume `packages/proto/schemas/common/v1/code_graph.proto` as the Graph envelope; extend `NodeKind` / `EdgeKind` only with TypeScript-specific kinds (component, hook, re_export). Do not duplicate the envelope. The envelope is co-owned with `go-code-graph`.
- [ ] OT-P0-003 | Leading-Comment Metadata On Declarations | Every declaration node carries a `leading_comments: string[]` field containing the verbatim leading JSDoc/comment blocks as they appear in source. This is the load-bearing contract that unblocks `react-component-library`'s migration off `@vrooliWidget*` / `@vrooliComponent*` regex scraping.
- [ ] OT-P0-004 | Two-Step Rewrite (Plan + Apply) | Implement `Rewrite(plan_id?, operations: []FileMove|ImportRewrite) → RewriteResponse`. First call returns a `plan_id` plus a normalized operation log and no disk change. A second call with the `plan_id` and explicit `--apply` flag mutates the filesystem via `ts-morph` (which preserves formatting). The scenario never invokes git and never invokes `tsc`.
- [ ] OT-P0-005 | Single-Project Resolution | `Extract` and `Rewrite` operate on exactly one `tsconfig.json` per call. Reject ambiguous inputs (no `tsconfig.json` discoverable, multiple at the same level, pnpm workspace ambiguity) with a typed error explaining which file to point at.
- [ ] OT-P0-006 | Partial Graph + Structured Warnings | When source files fail to parse or imports are unresolvable, return a partial graph plus a `Warnings[]` list of `{file, kind, message}` entries. Hard fail only on catastrophic project errors (missing `tsconfig.json`, unreadable path, broken parser process). Mid-migration scenarios are first-class inputs.
- [ ] OT-P0-007 | Per-Path Serialization, Parallel Across Paths | Two concurrent `Extract` calls for the same `project_path` serialize through an in-process queue (additionally enforced at the Node sidecar layer because `ts-morph` Project state is not safe to share across parallel invocations). Calls for different paths run in parallel.
- [ ] OT-P0-008 | Connect-RPC + CLI + UI Parity | Every capability is exposed through proto + Connect-RPC. The CLI (`typescript-code-graph graph extract`, `typescript-code-graph rewrite plan`, `typescript-code-graph rewrite apply`) is a translation layer over Connect clients; the UI is the same. No REST/JSON workarounds.
- [ ] OT-P0-009 | Node Sidecar With Lifecycle Supervision | Spawn a Node child process that hosts `ts-morph` and communicates with the Go API via a stable IPC channel (JSON over stdio or a local Unix socket). Lifecycle-supervised: crash → restart with backoff; health surfaced via `/health`. Hide all sidecar mechanics behind the Connect-RPC interface.
- [ ] OT-P0-010 | Golden-Fixture Determinism Gate | Ship fixture TS projects under `bas/fixtures/` (`ts-junk-drawer/`, `ts-jsdoc-tags/`, and `ts-usage-facts/`) each with hand-curated `expected-graph.json`. CI fails if any fixture's extracted graph diverges byte-for-byte from the expected file. The `ts-jsdoc-tags/` fixture pins the leading-comment contract that `react-component-library` depends on; `ts-usage-facts/` pins generic import/call/reference/JSX/export facts for future `code-facts` consumers.
- [ ] OT-P0-011 | Graph Explorer + Diagnostics UI | Ship a UI surface that lets an operator paste a TS project root or tsconfig path, see the extracted nodes/edges/leading comments/warnings, view server health (including Node sidecar status), and inspect recent extraction calls.
- [ ] OT-P0-012 | Performance SLA | Extraction completes in <5 seconds for any scenario with ≤200 files and <30 seconds for any scenario with ≤2000 files, measured end-to-end including Connect-RPC transport and Node sidecar round-trip. Performance regression tests live in CI.

### 🟠 P1 – Should have post-launch

- [ ] OT-P1-001 | Fixture Validator UI | Ship a UI surface where an operator can drop a fixture directory (TS source + expected-graph.json) and see the diff between expected and actual extraction. Mirrors `go-code-graph`'s P1 capability.
- [ ] OT-P1-002 | Operation Log Persistence | Persist completed Rewrite operations in a local SQLite store so operators can audit what the scenario actually changed. Read-only API on the audit trail; no replay capability in v1.
- [ ] OT-P1-003 | Barrel-File and Re-Export Coverage | First-class support for re-export edges (`export * from './x'`, `export { Y } from './y'`) as a distinct `EdgeKind`; expose `module → declaration` mappings even when nothing is imported directly.
- [ ] OT-P1-004 | JSX/TSX Declaration Awareness | Surface React components (named function / arrow / class) and hooks (`use*` convention or `@vrooliWidget` tag) as their own `NodeKind` rather than collapsing them into "function".
- [ ] OT-P1-005 | Content-Hash Cache Hint | Return a `graph_hash` field on every `Extract` response so consumers can detect "graph unchanged since last call" without redoing their own diff. No internal cache in this scenario.
- [ ] OT-P1-006 | react-component-library Migration Cutover | With this scenario shipped and OT-P0-003 stable, coordinate the `react-component-library` upgrade off its current regex parsing onto a typescript-code-graph client. (Work executed in the rcl scenario; this OT tracks the cross-scenario coordination commitment.)

### 🟢 P2 – Future / expansion

- [ ] OT-P2-001 | FindReferences RPC | `FindReferences(symbol_id) → []Reference` — LSP-style symbol reference search across the loaded project.
- [ ] OT-P2-002 | DiffGraphs RPC | `DiffGraphs(graph_a, graph_b) → GraphDiff` — structured comparison for migration before/after verification.
- [ ] OT-P2-003 | PublicAPI RPC | `PublicAPI(project) → APIDigest` — exported declarations only, suitable for breaking-change detection between commits.
- [ ] OT-P2-004 | ParseDiagnostics RPC | `ParseDiagnostics(project_path) → []Diagnostic` — structured TypeScript compiler errors and warnings as first-class data.
- [ ] OT-P2-005 | pnpm Workspace Support | Optional second mode that follows a pnpm workspace and returns a multi-project graph. Out of v1 because Vrooli scenarios are single-project by convention.
- [ ] OT-P2-006 | In-Process Parser (No Sidecar) | Evaluate replacing the Node sidecar with a Go-native TS parser (e.g. tree-sitter-typescript) when accuracy and ts-morph parity allow. Performance-only optimization; not a scope expansion.

## 🧱 Tech Direction Snapshot

Preferred stacks: Go API using Connect-RPC for the graph service, registered through `api/internal/modules/registry.go` per the react-vite template. Go CLI using the `cliapp.ArgSchema` pattern. React+Vite UI using the vrooli-default design kit. All cross-scenario calls go through proto + Connect-RPC — never REST/JSON. Proto contract references the shared envelope at `packages/proto/schemas/common/v1/code_graph.proto` plus `packages/proto/schemas/typescript-code-graph/v1/` for the service definition and TS-specific node-kind extensions.

Parser dependency: `ts-morph` (canonical TypeScript compiler-API wrapper) loaded against the target `tsconfig.json`. Because `ts-morph` is a Node library and the API is Go, the scenario spawns and supervises a small Node sidecar process. The sidecar exposes a JSON-over-stdio (or local Unix socket) protocol; the Go API holds a single sidecar client and serializes per-path. The Connect-RPC interface intentionally hides the sidecar — consumers never see Node, never see `ts-morph`, and never see IPC mechanics. P2 evaluates replacing the sidecar with an in-process Go TS parser.

Storage: SQLite for the optional Operation Log audit trail (P1). Nothing else is persisted. v1 ships without any cache layer — `Extract` re-parses on every call. Consumers (cartographer, rcl) cache snapshots at their own layer.

Concurrency model: per-path serialization, parallel across paths. An in-process Go-side mutex keyed by absolute `project_path` serializes `Extract` calls for the same path. The Node sidecar also serializes per-Project internally (Project state is not safe across parallel invocations). Apply operations on the same path serialize through the same Go lock.

Integration strategy: consumers shell out to the Go CLI for one-off interactive use, but production callers (cartographer, react-component-library) hold a long-lived Connect-RPC client against the running scenario. No HTTP routing, no REST endpoints, no per-call process spawn for the typical path.

Non-goals: No source-code parsing outside `ts-morph`. No git operations of any kind — never `git mv`, `git commit`, `git status`, or `git diff`. No build invocation — never `tsc`, never `pnpm`. No automatic rollback on partial Rewrite failure; the operator owns rollback via git. No internal extraction cache in v1. No endpoint/proto-health/widget policy inference in the graph provider; this scenario emits generic TypeScript evidence only. No support for parsing non-TS languages — that belongs in sibling scenarios.

## 🤝 Dependencies & Launch Plan

Required resources: None. SQLite is bundled if/when OT-P1-002 lands. Node runtime (≥20.x) for the sidecar — bundled by the react-vite template's lifecycle.

Scenario dependencies: None at runtime. The shared Graph proto envelope at `packages/proto/schemas/common/v1/code_graph.proto` is a build-time dependency only (co-owned with `go-code-graph`).

Consumer dependencies (informational): architecture-cartographer's `graph` and `apply` domains, and react-component-library's planned migration off regex parsing (OT-P1-006). Neither is a runtime dependency of typescript-code-graph.

Operational risks: (a) Incomplete graph extraction leading to false confidence in downstream consumers — mitigated by the golden-fixture determinism gate (OT-P0-010), including a dedicated `ts-jsdoc-tags/` fixture that pins the leading-comment contract. (b) Node sidecar crash leaving the API in a degraded state — mitigated by lifecycle supervision (OT-P0-009) with health surfaced through `/health`. (c) `ts-morph` performance regressions on large projects — mitigated by the explicit performance SLA (OT-P0-012) and CI perf regression tests. (d) Rewrite operations that crash mid-apply leaving the working tree in a torn state — accepted by design; operators recover via git. (e) IPC stalls between Go API and Node sidecar — mitigated by a request-level timeout and a heartbeat probe on the sidecar; stalled requests return a typed timeout error.

Launch sequencing: (1) Confirm the shared `common/v1/code_graph.proto` envelope (owned jointly with `go-code-graph`) plus author `typescript-code-graph/v1/` service definition with TS node-kind extensions. (2) Stand up the Node sidecar with `ts-morph` and the IPC contract. (3) Implement `Extract` against the sidecar including leading-comment metadata; pass fixture-validated determinism tests (`ts-junk-drawer/`, `ts-jsdoc-tags/`, `ts-usage-facts/`). (4) Implement the two-step `Rewrite` via `ts-morph` Project APIs; add fixture coverage for FileMove and ImportRewrite. (5) Wire CLI surfaces and UI graph explorer + diagnostics (including sidecar status panel). (6) Stand up per-path serialization mutex and concurrency tests. (7) Ship generic usage facts for imports, references, calls, JSX usage, and exports as graph nodes. (8) Ship performance regression suite. (9) P1: Fixture Validator UI, Operation Log persistence, barrel/re-export coverage, content-hash cache hint. (10) Coordinate react-component-library cutover (OT-P1-006) once OT-P0-003 is stable.

## 🎨 UX & Branding

User experience: Two distinct audiences. The CLI is the primary surface for agents and operators — concise, scriptable, default human-friendly output with optional `--json`. Commands map one-to-one to RPCs: `typescript-code-graph graph extract <path>` prints the graph summary plus warnings; `typescript-code-graph rewrite plan <ops.json>` returns a plan ID and the normalized operation log; `typescript-code-graph rewrite apply <plan_id>` executes and prints the result. The UI is the secondary surface for human debugging — paste a path, see the graph, inspect leading comments and warnings, view server health (API + Node sidecar), and (P1) drop a fixture to diff expected vs. actual. Dense, operational, scan-friendly. Not a marketing surface.

Visual design: Inherits the vrooli-default design kit and AppShell pattern from the react-vite template. Utilitarian density. Default tokens, no custom overrides without review. Graphs paired with node/edge tables for keyboard-navigable access. Leading-comment metadata surfaced inline on declaration nodes (collapsed by default, expandable). Sidecar status panel exposed prominently on the diagnostics view because a dead sidecar is the most common failure mode. Warning severity is communicated with both icon and label, never color alone. PWA install surface seeded from the template; generic icons replaced when final branding stabilizes.

Voice and messaging: Engineer-direct. State what the scenario did, what it could not do, and what evidence backs the result. Never paper over warnings; always surface them in the response. CLI error messages name the offending file path and the typed error kind ("no tsconfig.json found at path", "sidecar timeout after 30s", "ts-morph emitted N parse errors").

Accessibility: WCAG AA target. Every visual graph has a keyboard-navigable text alternative listing nodes, edges, leading comments, and warnings. Color contrast meets AA ratios for all text and interactive elements. Focus order is logical and visible for keyboard-only navigation across the graph explorer, diagnostics view, sidecar status panel, and fixture validator.

Branding hooks: Inherits Vrooli branding tokens. No standalone product identity in v1 — typescript-code-graph is infrastructure, not a marketed product.
