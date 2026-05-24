# Problems — Go Code Graph

Persistent register of known issues, tech debt, and deferred work specific to **this** scenario. Future agents read this file to avoid re-discovering the same constraint.

This file ships empty in newly generated scenarios. Append entries as they appear.

## What belongs here

- **Known bugs** that are real but not yet worth fixing
- **Tech debt** — workarounds that need a real fix later
- **Deferred work** — features descoped from a phase, with the reason
- **Architecture drift** — code/docs/tests that no longer line up with the intended capability map or boundary model
- **Constraints discovered the hard way** that aren't visible from the code (e.g., "this resource needs warm-up before the first call; see commit X")

## What does NOT belong here

- **Generic template issues** — those go in [`../guides/troubleshooting.md`](../guides/troubleshooting.md)
- **Open feature requests** — track those in PRD operational targets
- **Code comments** — if the constraint is local to one file, a comment there is more discoverable
- **Test failures** — fix them, don't document them

## Entry template

Use this shape so entries are scannable. Append newest at the bottom.

```markdown
### YYYY-MM-DD — short title

**Symptom:** What goes wrong, observable from outside the system.

**Root cause:** What actually causes it (or "unknown" if not yet diagnosed).

**Workaround:** What to do today to keep moving.

**Real fix:** What needs to happen for this entry to be deleted.

**Owner:** Who should drive the fix (or "unassigned").

**Refs:** Code paths, related issues, prior commits.
```

## Entries

## Architecture Drift

Use this section for deferred findings from `screaming-architecture-audit`. Do not create a standalone architecture-audit report unless the work is a migration handoff with a planned retirement path back into `ARCHITECTURE.md`, `SEAMS.md`, or this file.

| Area | Drift | Maturity Impact | Real Fix |
|---|---|---|---|
| (none open) | — | — | — |

### 2026-05-23 — `Extract` proto envelope under-emits cartographer-relevant attributes

**Symptom:** Cartographer's `RawGraph` shape (in `scenarios/architecture-cartographer/api/internal/graph/types.go`) carries fields that go-code-graph does not yet populate over the wire, so the cartographer-side translator (`scenarios/architecture-cartographer/api/internal/graph/gocodegraph/client.go`, `protoToRawGraph`) leaves them at Go zero values. Concretely:

- `FileNode.Lines` — go-code-graph emits no `lines` attribute on `NODE_KIND_FILE` nodes. Cartographer leaves `0`.
- `FileNode.IsTest` — go-code-graph emits no `is_test` attribute (today `Normalize()` doesn't split `GoFiles` vs `TestGoFiles`; it only reads `p.GoFiles`). Cartographer leaves `false`. Test-file coverage is silently lost.
- `PackageNode.Internal` — go-code-graph emits no `internal` attribute on package nodes. Cartographer leaves `false`. Used by cartographer's domain-classification heuristics, so the loss is observable.
- `ImportEdge.TestOnly` — go-code-graph emits no `test_only` attribute on `EDGE_KIND_IMPORT` edges (only `_test.go`-only imports should set this). Cartographer leaves `false`.
- `ImportEdge.SymbolIDs` — no analogue in the proto envelope. Cartographer cannot reconstruct symbol-level import provenance from `Extract` output.

**Root cause:** `internal/graph/normalize.go` only writes the attributes it needs for go-code-graph's own determinism gate (`language`, `import_path`, `package_id`, `file_id`, `exported`, plus the `kind` tag added in `handlers/graph/adapter.go`). The producer side was designed for go-code-graph's own consumers first; the cartographer-side requirements (richer FileNode/PackageNode/ImportEdge attributes) were not yet visible when the proto contract was first wired.

**Workaround:** Cartographer's translator preserves all proto fields it does see (the `protoToRawGraph` mapping in `client.go`) and leaves zero values on missing fields rather than silently dropping the node. Determinism is unaffected because the missing fields are non-discriminating; only the downstream domain-classification accuracy is degraded.

**Real fix:** Extend `Normalize()` (and the package loader feeding it) to emit:

1. `file:` nodes with `lines` (counted from the file's `*ast.File`'s `Fset.Position(...).Line` of the EOF token) and `is_test` (true when the basename ends in `_test.go` or the file is in `p.TestGoFiles`/`p.XTestGoFiles`).
2. `package:` nodes with `internal` (true when the import path matches `*/internal/*` or `internal/*`).
3. `import:` edges with `test_only` (true when the edge originates only from a `*_test.go` file). This requires walking `p.TestImports` separately and merging.

Then update `cloneAttributes` in `handlers/graph/adapter.go` to whitelist the new attributes (it already pass-throughs the full map, so this is "no-op" for emission but worth a doc note). Cartographer-side: extend `protoToRawGraph` to read the new attributes. Determinism gate (Phase 4 fixtures) will catch regressions.

**Owner:** Next go-code-graph implementation pass (or paired with cartographer's domain-classification improvements; whichever surfaces first).

**Refs:** `internal/graph/normalize.go`, `handlers/graph/adapter.go`, `scenarios/architecture-cartographer/api/internal/graph/gocodegraph/client.go` (`protoToRawGraph` data-loss notes), `scenarios/architecture-cartographer/api/internal/graph/types.go`.

### 2026-05-23 — Per-path mutex memory leak

**Symptom:** `graph.PathMutex` registers a `sync.Mutex` per absolute scenario path it has ever seen and never evicts it. In a long-running scenario in CI (or a multi-tenant operator setup) the internal `sync.Map` grows monotonically. Memory growth is small per entry but unbounded over time.

**Root cause:** v1 implementation prioritises correctness (serialization invariant per path) over reclamation. There is no LRU eviction or refcount-based cleanup.

**Workaround:** None needed for v1's expected workload (single-operator, dozens of distinct scenario paths). Restart the scenario if a process accumulates thousands of stale entries.

**Real fix:** Switch the registry to a bounded LRU (capacity ~10,000) with refcount-based eviction so the entry only goes away once no goroutine holds the mutex. Recorded as deferred in plan §11 of `~/.vrooli/plans/go-code-graph-proto-api-cli-implementation.md`.

**Owner:** Future infra-pass on `internal/graph/mutex.go`.

**Refs:** `internal/graph/mutex.go`, plan §11 risk table.

### 2026-05-23 — In-memory `PlanStore` evaporates on restart

**Symptom:** `RewritePlan` returns a `plan_id`; if the scenario restarts before the operator calls `RewriteApply`, the plan is gone and `RewriteApply` returns `InvalidArgument` ("plan_id unknown"). The operator must re-plan.

**Root cause:** `internal/rewrite/store_memory.go::MemoryStore` is an in-process `sync.Map`. No persistence backend in v1.

**Workaround:** Always run `plan` and `apply` back-to-back in the same scenario uptime. The cartographer consumer re-plans before every apply, so this does not block the integration today.

**Real fix:** Land REQ-P1-002 — a SQLite-backed `PlanStore` implementation (`internal/rewrite/store_sqlite.go`) wired through the existing `PlanStore` seam. The interface is already in place; only the implementation and its `mocks/` peer change. Persist `{plan_id, scenario_path, normalized_operations, created_at}` and optionally a TTL.

**Owner:** Next rewrite-domain implementation pass.

**Refs:** `internal/rewrite/store.go`, `internal/rewrite/store_memory.go`, `requirements/02-two-step-rewrite/module.json` (REQ-P1-002).

### 2026-05-23 — SQLite Operation Log not implemented

**Symptom:** The PRD calls for a durable Operation Log (REQ-P1-002) so operators can audit historical rewrite plans and applies after a scenario restart. Today no such log exists; `MemoryStore` records plans only, no apply results.

**Root cause:** Deferred from the v1 implementation plan (plan §4 Out of Scope, plan §11). Only the in-memory `PlanStore` shipped.

**Workaround:** Operators that need an audit trail should diff against git after each `apply` (the scenario never invokes git itself, but the operator owns the working tree). Per-op `OperationResult` records flow back through the API response synchronously, so the immediate-consumer view is intact.

**Real fix:** REQ-P1-002 — add a SQLite-backed log of `(plan_id, op_index, kind, status, message, applied_at)` rows. Likely lives alongside the persistent `PlanStore` (same SQLite file, two tables).

**Owner:** Same pass as the persistent `PlanStore`.

**Refs:** `requirements/02-two-step-rewrite/module.json` (REQ-P1-002).

### 2026-05-23 — Fixture Validator UI not implemented

**Symptom:** REQ-P1-001 calls for a Fixture Validator UI surface that lets an operator pick a fixture under `bas/fixtures/`, run `Extract`, and visually diff against `expected-graph.json`. The byte-stable determinism gate runs in tests today, but there is no human-facing diff surface.

**Root cause:** UI work is out of scope for the v1 implementation plan (plan §4 Out of Scope). Only API and CLI shipped.

**Workaround:** `UPDATE_FIXTURES=1 go test ./internal/graph/...` regenerates `expected-graph.json` files; the operator reviews the git diff manually.

**Real fix:** REQ-P1-001 — separate UI plan. Wire a `ui/src/features/fixture-validator/` feature on top of the existing `Extract` Connect client, rendering nodes/edges side-by-side with a JSON diff component.

**Owner:** UI plan (not yet authored).

**Refs:** `requirements/01-deterministic-graph-extraction/module.json` (REQ-P1-001), `bas/fixtures/`.

### 2026-05-23 — `include_vendor` flag is wired through but not honored

**Symptom:** `ExtractRequest.include_vendor` flows from CLI flag (`--include-vendor`) to the proto field to the service input, and is set on the underlying `packages.Config`. But the loader does not filter vendor packages out of the result when `include_vendor=false`; both values currently return the same graph. The flag is silently a no-op for filtering purposes.

**Root cause:** REQ-P1-003 deep-handling was deferred. The proto contract reservation and CLI flag landed in v1 so the wire shape is stable; the actual vendor-filtering pass on the loaded `[]*packages.Package` was descoped.

**Workaround:** Document the behaviour and warn operators. Today the vendor directory contents simply appear as additional `package:` nodes in every extraction; downstream consumers (cartographer) tolerate this.

**Real fix:** REQ-P1-003 — in `internal/graph/normalize.go`, filter packages whose `PkgPath` contains `/vendor/` (and their incident edges) when `LoadOptions.IncludeVendor == false`. Update the determinism fixtures or add a `go-vendored` fixture so the test surface exercises both branches.

**Owner:** Next graph-domain implementation pass.

**Refs:** `internal/graph/normalize.go`, `internal/graph/loader.go::LoadOptions`, `requirements/01-deterministic-graph-extraction/module.json` (REQ-P1-003).

### 2026-05-23 — Extended method-set coverage not implemented

**Symptom:** REQ-P1-004 calls for extended Go method-set coverage on `GO_METHOD` nodes — capturing interface satisfaction, embedded-type promotion, and pointer-vs-value receivers. Today `Normalize()` emits `GO_METHOD` nodes with name/path/exported attributes but no method-set metadata.

**Root cause:** Deferred from v1; the `go/types` extraction needed is substantially more involved than the v1 normalization pass.

**Workaround:** Consumers that need method-set semantics (cartographer's interface-conformance heuristics) compute them themselves from the raw nodes + edges.

**Real fix:** REQ-P1-004 — extend `Normalize()` to walk `types.Package.Scope()` for each loaded package, identify method sets via `types.NewMethodSet`, and emit `attributes["method_set"]` (or a richer set of attributes) plus interface-conformance edges.

**Owner:** Next graph-domain implementation pass.

**Refs:** `internal/graph/normalize.go`, `requirements/01-deterministic-graph-extraction/module.json` (REQ-P1-004).

## Cross-references

- [`PROGRESS.md`](PROGRESS.md) — lifecycle log (forward-looking)
- [`SEAMS.md`](SEAMS.md) — boundary registry (load-bearing for tests)
- [`TESTING.md`](TESTING.md) — test patterns
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — generic-template issues
