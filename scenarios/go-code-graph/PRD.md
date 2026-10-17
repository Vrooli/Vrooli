# Product Requirements Document (PRD)

## 🎯 Overview

Purpose: Go Code Graph extracts a deterministic, reproducible graph of files, packages, declarations, import edges, and generic Go usage facts from any Go module and executes mechanical Go refactors (file moves and import rewrites) on behalf of consumer scenarios. It wraps `golang.org/x/tools/go/packages` so that no other Vrooli scenario has to parse Go source itself.

Target users: Vrooli scenario maintainers and migration agents working through architecture-cartographer; future Go-static-analysis scenarios that need a typed graph instead of grep-based heuristics; template authors evaluating screaming-architecture compliance for Go scenarios.

Deployment surfaces: Connect-RPC API (primary contract for programmatic consumers), Go CLI (primary surface for agents and direct invocation), React+Vite UI (graph explorer for debugging, server diagnostics, fixture validation), plus the lifecycle integration provided by the react-vite template.

Value proposition: Replace ad-hoc, grep-based, regex-heavy Go inspection with a single typed graph contract and a single mechanical refactor surface. Consumers stop reimplementing Go parsers and stop reinventing file moves and import rewrites — they call `Extract` and `Rewrite` and trust the result. The graph is deterministic enough to be checked into version control as a baseline; the refactors are scoped enough that operators retain full control via git.

## 🎯 Operational Targets

Operational targets are measurable outcomes; checkboxes may auto-update based on validation.

### 🔴 P0 – Must ship for viability

- [ ] OT-P0-001 | Deterministic Graph Extraction | Implement `Extract(ExtractRequest{module_path}) → Graph` against `golang.org/x/tools/go/packages` using a fixed load mode that returns files, packages, top-level declarations (types, funcs, vars, consts, interfaces, methods), import specs, symbol references, call expressions, type usages, and import edges; byte-stable serialization for identical inputs.
- [ ] OT-P0-002 | Shared Code-Graph Proto Envelope | Define `packages/proto/schemas/common/v1/code_graph.proto` with a language-agnostic Graph envelope plus extensible `NodeKind` / `EdgeKind` enums, and reference it from both `go-code-graph` and (later) `typescript-code-graph`; never duplicate the envelope.
- [ ] OT-P0-003 | Two-Step Rewrite (Plan + Apply) | Implement `Rewrite(plan_id?, operations: []FileMove|ImportRewrite) → RewriteResponse`. First call returns a `plan_id` plus a normalized operation log and no disk change. A second call with the `plan_id` and explicit `--apply` flag mutates the filesystem. The scenario never invokes git and never invokes `go build`.
- [ ] OT-P0-004 | Single-Module Project Resolution | `Extract` and `Rewrite` operate on exactly one `go.mod` per call. Reject ambiguous inputs (no `go.mod` discoverable, multiple `go.mod` files, or `go.work` workspace) with a typed error explaining which file to point at.
- [ ] OT-P0-005 | Partial Graph + Structured Warnings | When source files fail to parse or imports are unresolvable, return a partial graph plus a `Warnings[]` list of `{file, kind, message}` entries. Hard fail only on catastrophic project errors (missing `go.mod`, unreadable path). Mid-migration scenarios are first-class inputs.
- [ ] OT-P0-006 | Per-Path Serialization, Parallel Across Paths | Two concurrent `Extract` calls for the same `module_path` serialize through an in-process queue. Calls for different paths run in parallel. The same rule applies to `Rewrite` apply operations on the same path.
- [ ] OT-P0-007 | Connect-RPC + CLI + UI Parity | Every capability is exposed through proto + Connect-RPC. The CLI (`go-code-graph extract`, `go-code-graph rewrite plan`, `go-code-graph rewrite apply`) is a translation layer over Connect clients; the UI is the same. No REST/JSON workarounds.
- [ ] OT-P0-008 | Golden-Fixture Determinism Gate | Ship at minimum two fixture Go modules under `bas/fixtures/` (`go-cycles/` and `go-mislocated/`) each with hand-curated `expected-graph.json`. CI fails if any fixture's extracted graph diverges byte-for-byte from the expected file. This is the trust anchor for every consumer.
- [ ] OT-P0-009 | Graph Explorer + Diagnostics UI | Ship a UI surface that lets an operator paste a module path, see the extracted nodes/edges/warnings, view server health, and inspect recent extraction calls. The graph explorer is the human debug path when an automated consumer reports unexpected results.
- [ ] OT-P0-010 | Performance SLA | Extraction completes in <5 seconds for any scenario with ≤200 files and <30 seconds for any scenario with ≤2000 files, measured end-to-end including Connect-RPC transport. Performance regression tests live in CI.

### 🟠 P1 – Should have post-launch

- [ ] OT-P1-001 | Fixture Validator UI | Ship a UI surface where an operator can drop a fixture directory (Go source + expected-graph.json) and see the diff between expected and actual extraction. Directly accelerates the determinism tests that consumers like cartographer depend on.
- [ ] OT-P1-002 | Operation Log Persistence | Persist completed Rewrite operations (plan ID, files affected, timestamp, success/failure) in a local SQLite store so operators can audit what the scenario actually changed. Read-only API on the audit trail; no replay capability in v1.
- [ ] OT-P1-003 | Vendored-Deps Awareness | By default exclude `vendor/` directories and module-cache paths from extraction. Provide a `--include-vendor` flag for callers that need full coverage. Document the default in the proto.
- [ ] OT-P1-004 | Extended Method-Set Coverage | Beyond top-level declarations, surface receiver types for methods and embedded fields for structs so cartographer's auto-placement signals can reason about ownership clusters more precisely.
- [ ] OT-P1-005 | Content-Hash Cache Hint | Return a `graph_hash` field on every `Extract` response so consumers can detect "graph unchanged since last call" without redoing their own diff. No internal cache in this scenario.

### 🟢 P2 – Future / expansion

- [ ] OT-P2-001 | FindReferences RPC | `FindReferences(symbol_id) → []Reference` — LSP-style query API over the reference facts emitted by `Extract`.
- [ ] OT-P2-002 | DiffGraphs RPC | `DiffGraphs(graph_a, graph_b) → GraphDiff` — structured comparison for migration before/after verification.
- [ ] OT-P2-003 | PublicAPI RPC | `PublicAPI(module) → APIDigest` — exported declarations only, suitable for breaking-change detection between commits.
- [ ] OT-P2-004 | ParseDiagnostics RPC | `ParseDiagnostics(module_path) → []Diagnostic` — structured Go compiler errors and warnings as first-class data.
- [ ] OT-P2-005 | Workspace (go.work) Support | Optional second mode that follows a `go.work` file and returns a multi-module graph. Out of v1 because Vrooli scenarios are single-module by convention.

## 🧱 Tech Direction Snapshot

Preferred stacks: Go API using Connect-RPC for the graph service, registered through `api/internal/modules/registry.go` per the react-vite template. Go CLI using the `cliapp.ArgSchema` pattern. React+Vite UI using the vrooli-default design kit. All cross-scenario calls go through proto + Connect-RPC — never REST/JSON, including for the cartographer integration. Proto contract lives at `packages/proto/schemas/common/v1/code_graph.proto` (shared envelope) plus `packages/proto/schemas/go-code-graph/v1/` (service definition and Go-specific node-kind extensions). The shared envelope is language-agnostic so future Python/Rust siblings can join without redefining the Graph type.

Parser dependency: `golang.org/x/tools/go/packages` remains the only parser and resolver. The default `full` profile uses `NeedFiles | NeedImports | NeedTypes | NeedSyntax | NeedTypesInfo | NeedName | NeedDeps` with tests enabled for compatibility. Explicit `semantic` (same semantic mode, tests excluded) and `structural` (files, imports, syntax, and names without type checking) profiles let consumers pay only for the facts they need.

Storage: SQLite for the optional Operation Log audit trail (P1). Extraction results are cached as disposable, atomically-written JSON entries under `SCENARIO_DATA_DIR/extraction-cache` (or `GO_CODE_GRAPH_CACHE_DIR`) and invalidated by a source-content fingerprint plus profile, scope, and loader-environment settings. Cache storage is bounded by `GO_CODE_GRAPH_CACHE_MAX_BYTES` (512 MiB default). Cache failures are non-fatal.

Concurrency model: per-path serialization, parallel across paths. An in-process mutex keyed by absolute `module_path` serializes `Extract` calls for the same path; different paths run in parallel goroutines. Apply operations on the same path serialize through the same lock.

Integration strategy: consumers shell out to the Go CLI for one-off interactive use, but production callers (cartographer, react-component-library, future siblings) hold a long-lived Connect-RPC client against the running scenario. No HTTP routing, no REST endpoints, no per-call process spawn for the typical path.

Non-goals: No source-code parsing outside `golang.org/x/tools/go/packages`. No git operations of any kind — never `git mv`, `git commit`, `git status`, or `git diff`. No build invocation — never `go build` or `go test`. No automatic rollback on partial Rewrite failure; the operator owns rollback via git. No endpoint, proto-health, or Vrooli-surface policy in graph facts; consumers interpret generic imports, references, calls, and type usages at their own layer. No support for parsing non-Go languages — that belongs in sibling scenarios.

## 🤝 Dependencies & Launch Plan

Required resources: None. SQLite is bundled if/when OT-P1-002 lands.

Scenario dependencies: None at runtime. The shared Graph proto envelope at `packages/proto/schemas/common/v1/code_graph.proto` is a build-time dependency only.

Consumer dependencies (informational): architecture-cartographer is the first declared consumer; its `graph` and `apply` domains call `Extract` and `Rewrite` via Connect-RPC. Future consumers may include Go-specific scenarios for dead-code detection, public-API drift, or refactor recipes. None of these are runtime dependencies of go-code-graph.

Operational risks: (a) Incomplete graph extraction leading to false confidence in downstream consumers — mitigated by the golden-fixture determinism gate (OT-P0-008) that runs in CI on every change. (b) `go/packages` performance regressions on large modules — mitigated by the explicit performance SLA (OT-P0-010) and CI perf regression tests. (c) Rewrite operations that crash mid-apply leaving the working tree in a torn state — accepted by design; operators recover via git, since the scenario refuses to manage git itself. (d) Concurrent Extract calls thrashing CPU on resource-constrained hosts — mitigated by per-path serialization (OT-P0-006); different paths share CPU but only one extraction runs per path at a time.

Launch sequencing: (1) Ship the shared `common/v1/code_graph.proto` envelope plus `go-code-graph/v1/` service definitions. (2) Implement `Extract` with the full compatibility profile and partial-graph warning model; pass two fixture-validated determinism tests. (3) Implement the two-step `Rewrite` (plan + apply); add fixture coverage for FileMove and ImportRewrite. (4) Wire CLI surfaces (`extract`, `rewrite plan`, `rewrite apply`) and UI graph explorer + diagnostics. (5) Stand up per-path serialization mutex and concurrency tests. (6) Ship performance regression suite, lighter extraction profiles, scoped package patterns, and the bounded content-fingerprint cache.

## 🎨 UX & Branding

User experience: Two distinct audiences. The CLI is the primary surface for agents and operators — concise, scriptable, default human-friendly output with optional `--json`. Commands map one-to-one to RPCs: `go-code-graph extract <path>` prints the graph summary plus warnings; `go-code-graph rewrite plan <ops.json>` returns a plan ID and the normalized operation log; `go-code-graph rewrite apply <plan_id>` executes and prints the result. The UI is the secondary surface for human debugging — paste a path, see the graph, inspect warnings, view server health, and (P1) drop a fixture to diff expected vs. actual. Dense, operational, scan-friendly. Not a marketing surface.

Visual design: Inherits the vrooli-default design kit and AppShell pattern from the react-vite template. Utilitarian density. Default tokens, no custom overrides without review. Graphs paired with node/edge tables for keyboard-navigable access. Warning severity is communicated with both icon and label, never color alone. PWA install surface (`ui/public/site.webmanifest`, `apple-icon-180.png`, `favicon-196.png`, maskable manifest icons) seeded from the template; generic icons replaced when final branding stabilizes.

Voice and messaging: Engineer-direct. State what the scenario did, what it could not do, and what evidence backs the result. Never paper over warnings; always surface them in the response. CLI error messages name the offending file path and the typed error kind ("multiple go.mod files found", "vendor/ directory excluded by default").

Accessibility: WCAG AA target. Every visual graph has a keyboard-navigable text alternative listing nodes, edges, and warnings. Color contrast meets AA ratios for all text and interactive elements. Focus order is logical and visible for keyboard-only navigation across the graph explorer, diagnostics view, and fixture validator.

Branding hooks: Inherits Vrooli branding tokens. No standalone product identity in v1 — go-code-graph is infrastructure, not a marketed product.
