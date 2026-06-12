# Domains — Go Code Graph

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
| graph | Build the ground-truth code graph for a target Go module by loading `golang.org/x/tools/go/packages` and normalizing the result into the shared envelope. | Service / parser | No data in v1 (graph is returned, not persisted). | API, CLI, UI | REQ-P0-001, REQ-P0-002, REQ-P0-004, REQ-P0-005, REQ-P0-006 | `api/internal/graph/`, `api/handlers/graph/`, `cli/domains/graph/`, `ui/src/features/graph/`, `packages/proto/schemas/go-code-graph/v1/graph/`, `packages/proto/schemas/common/v1/code_graph.proto` |
| rewrite | Plan and execute mechanical Go refactors (file moves + import rewrites) as a two-step `plan → apply` operation. | Mutator / executor | Optional Operation Log (P1) in SQLite. | API, CLI, UI | REQ-P0-003 | `api/internal/rewrite/`, `api/handlers/rewrite/`, `cli/domains/rewrite/`, `ui/src/features/rewrite/`, `packages/proto/schemas/go-code-graph/v1/rewrite/` |
| explorer | UI surface for human debugging: paste a module path, view extracted nodes/edges/warnings, inspect recent calls. (P1) Fixture validator. | UI / inspection | None. | UI (primary), API (read-only telemetry endpoints) | REQ-P0-009, REQ-P1-001 | `ui/src/features/explorer/`, `ui/src/features/fixtures/`, `api/handlers/explorer/` |
| health | Report runtime readiness and dependency reachability. | Reporting / query | No product data. | API, UI | Starter scaffold health. | `api/handlers/health/`, `ui/src/features/health/`, `packages/proto/schemas/go-code-graph/v1/health/` |

The four domains map one-to-one onto the operational targets in [`../../PRD.md`](../../PRD.md): `graph` owns extraction; `rewrite` owns the two-step refactor surface; `explorer` owns the UI debug path; `health` is template-inherited infrastructure.

## Domain Details

### graph

- Purpose: produce a deterministic, byte-stable graph snapshot of a Go module by delegating all parsing to `golang.org/x/tools/go/packages` and normalizing the result into `common.v1.code_graph.Graph`.
- Primary archetype: service / parser.
- Secondary traits: deterministic serialization, partial-graph error reporting, per-path serialization.
- Owns: the Go parser client (a single load-mode configuration), normalization rules from raw `*packages.Package` to the shared envelope, warning aggregation, the per-path mutex registry, and the graph-hash derivation.
- Does not own: storage (graph is returned, not persisted in v1), the shared envelope itself (lives in `common/v1/`), or rewriting (that is `rewrite`'s job).
- API: `api/internal/graph/`, `api/handlers/graph/`. Connect-RPC service `GoCodeGraphService` with method `Extract`.
- CLI: `cli/domains/graph/` with `go-code-graph extract <path>` (default human output, optional `--json`).
- UI: `ui/src/features/graph/` — surfaces a read-only graph display for the `explorer` feature to consume.
- Storage: none in v1.
- Requirements: REQ-P0-001 (deterministic extraction), REQ-P0-002 (shared envelope adoption), REQ-P0-004 (single-module resolution), REQ-P0-005 (partial-graph warnings), REQ-P0-006 (per-path serialization).
- Tests: unit tests on normalization + warning aggregation + per-path mutex; integration tests against fixture Go modules in `bas/fixtures/`; performance regression tests.
- Related docs: [`FLOWS.md`](FLOWS.md) (Extract flow), [`../internal/SEAMS.md`](../internal/SEAMS.md) (PackagesLoader seam).

### rewrite

- Purpose: execute mechanical Go refactors (file moves and import rewrites) on behalf of consumer scenarios as a deterministic two-step `plan → apply` operation.
- Primary archetype: mutator / executor.
- Secondary traits: dry-run planning, atomic-per-op apply (no rollback), filesystem-only (never git, never build).
- Owns: operation normalization, plan-ID derivation (content hash of normalized ops), apply executor that mutates files in order, the optional Operation Log (P1).
- Does not own: git operations of any kind (never `git mv`, `git commit`, `git diff`), build invocation (never `go build`), or rollback (operator's responsibility via git).
- API: `api/internal/rewrite/`, `api/handlers/rewrite/`. Connect-RPC service `GoCodeGraphService` with methods `RewritePlan` and `RewriteApply`.
- CLI: `cli/domains/rewrite/` with `go-code-graph rewrite plan <ops.json>` and `go-code-graph rewrite apply <plan_id>`.
- UI: `ui/src/features/rewrite/` — read-only plan preview; apply is intentionally CLI-only in v1 to keep the destructive path explicit.
- Storage: optional SQLite Operation Log if/when REQ-P1-002 lands.
- Requirements: REQ-P0-003 (two-step rewrite), REQ-P1-002 (operation log).
- Tests: unit tests on operation normalization, plan-ID derivation, and atomic per-op behavior; integration tests with fixture modules covering FileMove and ImportRewrite; partial-failure tests (crash mid-apply leaves disk torn — verified, not rolled back).
- Related docs: [`FLOWS.md`](FLOWS.md) (Rewrite plan + apply flows), [`../internal/SEAMS.md`](../internal/SEAMS.md) (RewriteExecutor seam).

### explorer

- Purpose: human-facing UI surface for debugging extractions, viewing diagnostics, and (P1) validating fixtures.
- Primary archetype: UI / inspection.
- Secondary traits: read-only by design; never mutates source code; never invokes Rewrite.
- Owns: the graph-explorer page (paste path → see nodes/edges/warnings), the diagnostics page (server health, recent calls), and (P1) the fixture-validator page (drop fixture → diff expected vs. actual).
- Does not own: extraction itself (calls into `graph`), refactor execution (`rewrite` owns that), or any persisted product data.
- API: `api/handlers/explorer/` — small read-only endpoints for recent-calls telemetry. No new business logic; this is a UI-support surface.
- CLI: none (the explorer is UI-only by design; CLI agents call `extract` directly).
- UI: `ui/src/features/explorer/`, `ui/src/features/fixtures/`.
- Storage: none (recent-calls log is in-memory and bounded).
- Requirements: REQ-P0-009 (graph explorer + diagnostics UI), REQ-P1-001 (fixture validator UI).
- Tests: UI feature tests with mocked graph clients; accessibility tests for the graph view + diagnostics view + fixture validator.
- Related docs: [`UI-ARCHITECTURE.md`](UI-ARCHITECTURE.md) (slot layout).

### health

- Purpose: expose API/database readiness and show the UI can read live backend state.
- Primary archetype: reporting / query.
- Secondary traits: operational health, template-inherited.
- Owns: health response construction and dependency status mapping.
- Does not own: product data, business rules.
- API: `api/handlers/health/`.
- CLI: built-in `status` command provided through cli-core.
- UI: `ui/src/features/health/HealthCard.tsx` (embedded in the diagnostics page).
- Storage: none.
- Requirements: starter scaffold health only.
- Tests: handler, module, UI feature, accessibility.

## Shared Concepts

| Concept | Meaning | Owner |
|---|---|---|
| Graph envelope | The `Graph` proto message at `packages/proto/schemas/common/v1/code_graph.proto`. Language-agnostic; both go-code-graph and typescript-code-graph (planned) emit it. | `common/v1/code_graph.proto` |
| NodeKind / EdgeKind | Extensible enums on the shared envelope. Go-specific kinds (`go_type`, `go_func`, `go_var`, `go_const`, `go_interface`, `go_method`) live in the go-code-graph proto extension. | `common/v1/code_graph.proto` + `go-code-graph/v1/` |
| Warning | `{file, kind, message}` structured entry on partial graphs. Kinds are an enum: `parse_error`, `unresolved_import`, `type_check_failure`, `ambiguous_declaration`. | `common/v1/code_graph.proto` |
| Plan ID | Deterministic content hash of a normalized operation list returned by `RewritePlan`. Apply requires the matching plan ID. | `rewrite` |
| Per-path mutex | An in-process lock keyed by absolute `module_path`. Serializes extraction and apply for the same path; parallel across paths. | `graph` (extraction) + `rewrite` (apply) |
| Seam | Test-substitutable boundary wired once in production. | [`../internal/SEAMS.md`](../internal/SEAMS.md) |
| Requirement | Implementation-facing measurement tied back to the PRD. | `requirements/` |

## Deferred Domains

| Candidate Domain | Why Deferred | Revisit Trigger |
|---|---|---|
| `find_references` | LSP-style symbol reference search across the loaded module. P2 in PRD (OT-P2-001). | Promote when a consumer scenario needs cross-symbol reasoning beyond cartographer's current scope. |
| `diff_graphs` | Structured graph-vs-graph comparison. P2 in PRD (OT-P2-002). | Promote when migration before/after verification has a real consumer. |
| `public_api` | Exported-declarations digest for breaking-change detection. P2 in PRD (OT-P2-003). | Promote when a consumer scenario needs Go API drift signals. |
| `parse_diagnostics` | Structured Go compiler errors as first-class data. P2 in PRD (OT-P2-004). | Promote when a consumer wants compiler errors without re-running `go build`. |
| `workspace` | `go.work` multi-module support. P2 in PRD (OT-P2-005). | Promote only when a Vrooli scenario actually adopts a workspace layout (today's convention is one module per scenario). |

## Non-Domains

These are important but should not become product domains:

- `api/internal/server/` — HTTP composition substrate.
- `api/internal/module/` — shared module descriptor type.
- `api/internal/modules/` — thin registry for boot/codegen.
- `api/internal/database/` — cross-cutting database infrastructure (only relevant if/when REQ-P1-002 lands).
- `api/internal/testutil/` — cross-domain test harnesses.
- `ui/src/components/` — shared presentation primitives.
- `ui/src/test-utils/` — cross-feature testing support.

If one of these starts using product vocabulary (e.g. "graph" or "rewrite"), split the product piece into an owning domain instead of growing infrastructure.

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system shape and extension rules
- [`FLOWS.md`](FLOWS.md) — workflows and state transitions
- [`DATA.md`](DATA.md) — data ownership and storage
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — dependency contracts
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — boundary registry
- [`../internal/TESTING.md`](../internal/TESTING.md) — test strategy
