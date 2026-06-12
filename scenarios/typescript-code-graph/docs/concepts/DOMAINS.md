# Domains — TypeScript Code Graph

This document is the canonical map of product capabilities, bounded contexts, and ownership for this scenario. Keep it current whenever a domain is added, renamed, split, merged, or removed.

`notes` is a worked example from the template, not product scope. It will be removed once the real domains are green per the Gate 7 step in [`../START-HERE.md`](../START-HERE.md).

## Purpose Of This Document

Use this document to answer:

- What product capabilities does this scenario expose?
- Which domain owns each concept, table, proto, endpoint, UI feature, CLI command, and test surface?
- Which concepts are shared, deferred, or deliberately not domains?

System-level architecture belongs in [`ARCHITECTURE.md`](ARCHITECTURE.md). Workflow details belong in [`FLOWS.md`](FLOWS.md). Storage details belong in [`DATA.md`](DATA.md).

## Domain Inventory

| Domain | Purpose | Primary Archetype | Owns Data | Surfaces | Requirements | Source Paths |
|---|---|---|---|---|---|---|
| graph | Build the ground-truth code graph for a target TS project by orchestrating the Node sidecar against `ts-morph`, normalizing the result into the shared envelope (with declarations, imports, references, calls, JSX usage, exports, import bindings, and leading-comment metadata). | Service / parser | No data in v1 (graph is returned, not persisted). | API, CLI, UI | REQ-P0-001, REQ-P0-002, REQ-P0-003, REQ-P0-005, REQ-P0-006, REQ-P0-007 | `api/internal/graph/`, `api/handlers/graph/`, `cli/domains/graph/`, `ui/src/features/graph/`, `packages/proto/schemas/typescript-code-graph/v1/graph/`, `packages/proto/schemas/common/v1/code_graph.proto`, `api/internal/sidecar/` (delegate) |
| rewrite | Plan and execute mechanical TS refactors (file moves + import rewrites) via `ts-morph`'s formatting-preserving Project APIs as a two-step `plan → apply` operation. | Mutator / executor | Optional Operation Log (P1) in SQLite. | API, CLI, UI | REQ-P0-004 | `api/internal/rewrite/`, `api/handlers/rewrite/`, `cli/domains/rewrite/`, `ui/src/features/rewrite/`, `packages/proto/schemas/typescript-code-graph/v1/rewrite/`, `api/internal/sidecar/` (delegate) |
| sidecar | Lifecycle-supervise the Node child process that hosts `ts-morph`. Spawn, health-probe, restart-with-backoff, route IPC. | Process supervisor | None (sidecar metrics are in-memory). | API (internal), UI (status panel on diagnostics) | REQ-P0-009 | `api/internal/sidecar/`, `sidecar/` (Node code), `ui/src/features/sidecar/` |
| explorer | UI surface for human debugging: paste a TS project root or tsconfig path, view extracted nodes/edges/warnings/leading comments, inspect server health (API + sidecar). (P1) Fixture validator. | UI / inspection | None. | UI (primary), API (read-only telemetry endpoints) | REQ-P0-011, REQ-P1-001 | `ui/src/features/explorer/`, `ui/src/features/fixtures/`, `api/handlers/explorer/` |
| health | Report runtime readiness and dependency reachability (API health + sidecar reachability). | Reporting / query | No product data. | API, UI | Starter scaffold health. | `api/handlers/health/`, `ui/src/features/health/`, `packages/proto/schemas/typescript-code-graph/v1/health/` |

The five domains map onto the operational targets in [`../../PRD.md`](../../PRD.md): `graph` and `rewrite` mirror their go-code-graph counterparts with TS specifics; `sidecar` is unique to this scenario because `ts-morph` is a Node library; `explorer` owns the UI debug path; `health` is template-inherited.

## Domain Details

### graph

- Purpose: produce a deterministic, byte-stable graph snapshot of a TS project by delegating all parsing to `ts-morph` (via the Node sidecar) and normalizing the result into `common.v1.code_graph.Graph` with TS-specific node kinds, source-ranged usage facts, and leading-comment metadata.
- Primary archetype: service / parser.
- Secondary traits: deterministic serialization, partial-graph error reporting, per-path serialization (enforced at both Go and sidecar layers), leading-comment fidelity for `react-component-library`.
- Owns: the IPC contract with the Node sidecar, normalization rules from raw `ts-morph` output to the shared envelope, generic TypeScript usage fact extraction (import bindings, references, calls, JSX usage, exports), warning aggregation, the per-path mutex registry, and the graph-hash derivation. The graph domain does **not** own sidecar lifecycle or Vrooli policy inference — those belong to `sidecar` and downstream consumers such as `code-facts`.
- Does not own: storage (graph is returned, not persisted in v1), the shared envelope itself (lives in `common/v1/`), sidecar process management (delegated to `sidecar`), or rewriting (that is `rewrite`'s job).
- API: `api/internal/graph/`, `api/handlers/graph/`. Connect-RPC service `TypeScriptCodeGraphService` with method `Extract`.
- CLI: `cli/domains/graph/` with `typescript-code-graph graph extract <path>` (default human output, optional `--json`).
- UI: `ui/src/features/graph/` — surfaces a read-only graph display for the `explorer` feature to consume. Renders leading-comment metadata inline (collapsed by default).
- Storage: none in v1.
- Requirements: REQ-P0-001 (deterministic extraction), REQ-P0-002 (shared envelope adoption), REQ-P0-003 (leading-comment metadata), REQ-P0-005 (single-project resolution), REQ-P0-006 (partial-graph warnings), REQ-P0-007 (per-path serialization).
- Tests: unit tests on normalization + warning aggregation + per-path mutex + IPC framing; integration tests against fixture TS projects in `bas/fixtures/`; performance regression tests; specific `ts-jsdoc-tags/` fixture for the leading-comment contract.
- Related docs: [`FLOWS.md`](FLOWS.md) (Extract flow), [`../internal/SEAMS.md`](../internal/SEAMS.md) (SidecarClient seam, PathMutex seam).

### rewrite

- Purpose: execute mechanical TS refactors (file moves and import rewrites) on behalf of consumer scenarios as a deterministic two-step `plan → apply` operation, leveraging `ts-morph`'s formatting-preserving Project APIs via the Node sidecar.
- Primary archetype: mutator / executor.
- Secondary traits: dry-run planning, formatting-preserving file mutations, atomic-per-op apply (no rollback), filesystem-only (never git, never build).
- Owns: operation normalization, plan-ID derivation, apply executor that delegates to the sidecar's `ts-morph` Project API, the optional Operation Log (P1).
- Does not own: git operations (never `git mv`, `git commit`, `git diff`), build invocation (never `tsc`, never `pnpm`), or rollback (operator's responsibility via git).
- API: `api/internal/rewrite/`, `api/handlers/rewrite/`. Connect-RPC service `TypeScriptCodeGraphService` with methods `RewritePlan` and `RewriteApply`.
- CLI: `cli/domains/rewrite/` with `typescript-code-graph rewrite plan <ops.json>` and `typescript-code-graph rewrite apply <plan_id>`.
- UI: `ui/src/features/rewrite/` — read-only plan preview; apply is intentionally CLI-only in v1 to keep the destructive path explicit.
- Storage: optional SQLite Operation Log if/when REQ-P1-002 lands.
- Requirements: REQ-P0-004 (two-step rewrite), REQ-P1-002 (operation log).
- Tests: unit tests on operation normalization, plan-ID derivation; integration tests with fixture projects covering FileMove and ImportRewrite via `ts-morph`; partial-failure tests (crash mid-apply leaves disk torn — verified, not rolled back); formatting-preservation tests (Prettier output unchanged after rewrite).
- Related docs: [`FLOWS.md`](FLOWS.md) (Rewrite plan + apply flows), [`../internal/SEAMS.md`](../internal/SEAMS.md) (SidecarClient seam).

### sidecar

- Purpose: lifecycle-supervise the Node child process that hosts `ts-morph`, hiding all process-management mechanics behind a stable Go-side client. Both `graph` and `rewrite` route their parsing/mutation through this domain.
- Primary archetype: process supervisor.
- Secondary traits: spawn-with-backoff, health probe, IPC over JSON-over-stdio (or local Unix socket), request-level timeouts, restart on crash, structured error mapping.
- Owns: the sidecar binary location lookup, child-process lifecycle (`exec.Command`, stdin/stdout pipes), the IPC protocol (request envelope, response envelope, error envelope, heartbeat), the supervisor goroutine that monitors the process and restarts on exit, the in-memory sidecar status the `health` and `explorer` features consume.
- Does not own: the parsing logic itself (that's `ts-morph` inside the Node code in `sidecar/`), normalization (that's `graph`), or refactor execution (that's `rewrite`).
- API: `api/internal/sidecar/` — internal client surface. No public Connect-RPC endpoint in v1 except the sidecar-status read on the health handler.
- CLI: none directly. (`typescript-code-graph status` shows sidecar health via the existing `status` command.)
- UI: `ui/src/features/sidecar/SidecarStatusPanel.tsx` — embedded in the explorer's diagnostics page. Always visible because a dead sidecar is the most common failure mode.
- Storage: none.
- Requirements: REQ-P0-009 (lifecycle supervision).
- Tests: unit tests on IPC framing, request timeout, error decoding; integration tests that kill the sidecar mid-call to verify restart-with-backoff; chaos tests that inject latency to verify timeout behavior.
- Related docs: [`FLOWS.md`](FLOWS.md) (Sidecar lifecycle flow), [`../internal/SEAMS.md`](../internal/SEAMS.md) (SidecarClient seam).

### explorer

- Purpose: human-facing UI surface for debugging extractions, viewing diagnostics, and (P1) validating fixtures.
- Primary archetype: UI / inspection.
- Secondary traits: read-only by design; never mutates source code; never invokes Rewrite. Prominent sidecar-status panel because a dead sidecar is the typical failure mode.
- Owns: the graph-explorer page (paste path → see nodes/edges/leading comments/warnings), the diagnostics page (API health + sidecar status + recent calls), and (P1) the fixture-validator page (drop fixture → diff expected vs. actual).
- Does not own: extraction itself (calls into `graph`), refactor execution (`rewrite` owns that), sidecar process management (`sidecar` owns that), or any persisted product data.
- API: `api/handlers/explorer/` — small read-only endpoints for recent-calls telemetry and sidecar status. No new business logic.
- CLI: none (the explorer is UI-only by design; CLI agents call `extract` directly).
- UI: `ui/src/features/explorer/`, `ui/src/features/fixtures/`.
- Storage: none (recent-calls log is in-memory and bounded).
- Requirements: REQ-P0-011 (graph explorer + diagnostics UI), REQ-P1-001 (fixture validator UI).
- Tests: UI feature tests with mocked graph clients; accessibility tests for the graph view + diagnostics view + sidecar status panel + fixture validator.
- Related docs: [`UI-ARCHITECTURE.md`](UI-ARCHITECTURE.md) (slot layout).

### health

- Purpose: expose API readiness, database reachability (if P1 SQLite Operation Log is enabled), and sidecar reachability.
- Primary archetype: reporting / query.
- Secondary traits: operational health, template-inherited but extended with sidecar status.
- Owns: health response construction including sidecar status.
- Does not own: product data, business rules.
- API: `api/handlers/health/`.
- CLI: built-in `status` command provided through cli-core.
- UI: `ui/src/features/health/HealthCard.tsx` (embedded in the diagnostics page next to the sidecar status panel).
- Storage: none.
- Requirements: starter scaffold health only (extended with sidecar status check).
- Tests: handler, module, UI feature, accessibility.

## Shared Concepts

| Concept | Meaning | Owner |
|---|---|---|
| Graph envelope | The `Graph` proto message at `packages/proto/schemas/common/v1/code_graph.proto`. Language-agnostic; both go-code-graph and typescript-code-graph emit it. | `common/v1/code_graph.proto` |
| NodeKind / EdgeKind | Extensible enums on the shared envelope. TS-specific kinds (`ts_module`, `ts_component`, `ts_hook`, `ts_class`, `ts_interface`, `ts_type`, `ts_function`, `ts_var`, `ts_const`, `ts_re_export`, `ts_import_binding`, `ts_reference`, `ts_call`, `ts_jsx_usage`, `ts_export`) live in the typescript-code-graph proto extension. | `common/v1/code_graph.proto` + `typescript-code-graph/v1/` |
| Usage facts | Generic source-ranged TypeScript evidence for imports, references, calls, JSX tags/components, and exports. Policy-specific proof remains out of scope for this scenario. | `graph` |
| Leading comments | `leading_comments: string[]` field on every declaration node, carrying verbatim JSDoc/comment blocks. Load-bearing contract for `react-component-library`'s migration off regex parsing. | `graph` (extracts), shared envelope (defines field) |
| Warning | `{file, kind, message}` structured entry on partial graphs. Kinds: `parse_error`, `unresolved_import`, `type_check_failure`, `ambiguous_declaration`. | `common/v1/code_graph.proto` |
| Plan ID | Deterministic content hash of a normalized operation list returned by `RewritePlan`. Apply requires the matching plan ID. | `rewrite` |
| Per-path mutex | An in-process lock keyed by absolute `project_path`. Serializes extraction and apply for the same path; parallel across paths. **Additionally enforced inside the sidecar** because `ts-morph` Project state is not safe to share. | `graph` (extraction) + `rewrite` (apply) + sidecar |
| Sidecar | The Node child process hosting `ts-morph`. Communicates with the Go API via a stable IPC protocol. Lifecycle-supervised. | `sidecar` |
| Seam | Test-substitutable boundary wired once in production. | [`../internal/SEAMS.md`](../internal/SEAMS.md) |
| Requirement | Implementation-facing measurement tied back to the PRD. | `requirements/` |

## Deferred Domains

| Candidate Domain | Why Deferred | Revisit Trigger |
|---|---|---|
| `find_references` | LSP-style symbol reference search across the loaded project. P2 in PRD (OT-P2-001). | Promote when a consumer scenario needs cross-symbol reasoning. |
| `diff_graphs` | Structured graph-vs-graph comparison. P2 in PRD (OT-P2-002). | Promote when migration before/after verification has a real consumer. |
| `public_api` | Exported-declarations digest for breaking-change detection. P2 in PRD (OT-P2-003). | Promote when a consumer scenario needs TS API drift signals. |
| `parse_diagnostics` | Structured TS compiler errors as first-class data. P2 in PRD (OT-P2-004). | Promote when a consumer wants compiler errors without re-running `tsc`. |
| `workspace` | pnpm workspace multi-project support. P2 in PRD (OT-P2-005). | Promote only when a Vrooli scenario actually adopts a multi-project layout. |
| `in_process_parser` | Replace the Node sidecar with a Go-native TS parser. P2 in PRD (OT-P2-006). | Promote only when a Go TS parser reaches `ts-morph` parity and the sidecar's overhead becomes a measured bottleneck. |

## Non-Domains

These are important but should not become product domains:

- `api/internal/server/` — HTTP composition substrate.
- `api/internal/module/` — shared module descriptor type.
- `api/internal/modules/` — thin registry for boot/codegen.
- `api/internal/database/` — cross-cutting database infrastructure (only relevant if/when REQ-P1-002 lands).
- `api/internal/testutil/` — cross-domain test harnesses.
- `ui/src/components/` — shared presentation primitives.
- `ui/src/test-utils/` — cross-feature testing support.

If one of these starts using product vocabulary (e.g. "graph", "rewrite", or "sidecar"), split the product piece into an owning domain instead of growing infrastructure.

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system shape and extension rules
- [`FLOWS.md`](FLOWS.md) — workflows and state transitions
- [`DATA.md`](DATA.md) — data ownership and storage
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — dependency contracts
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — boundary registry
- [`../internal/TESTING.md`](../internal/TESTING.md) — test strategy
