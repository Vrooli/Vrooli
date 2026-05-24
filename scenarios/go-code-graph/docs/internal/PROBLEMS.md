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

### 2026-05-23 — `Extract` proto envelope under-emits cartographer-relevant attributes (partial fix 2026-05-24)

**Status (2026-05-24):** Partially closed. `lines` (real signal, computed from `*ast.File` end-position) and `internal` (path-based heuristic) now flow through and are consumed by cartographer's `protoToRawGraph`. `is_test` and `test_only` are emitted on the wire but always serialize as `"false"` today because the loader still runs `packages.Config{Tests: false}` — test files do not appear in `p.GoFiles`, so the basename heuristic never fires.

**Remaining gap:** Switch `internal/graph/loader_packages.go` to `Tests: true` and add a variant-merge pass in `Normalize` so test variants of a package (same `PkgPath`, different IDs; `ForTest` non-empty) are folded back into the canonical entry rather than skipped by `seenPkg[pkgID]`. Filter synthetic test-binary packages (`PkgPath` ending in `.test`). Once merged, `p.TestGoFiles` / `p.XTestGoFiles` populate `is_test`, and the diff between prod and test-variant `Imports` populates `test_only` on edges. Add a `go-tests` fixture exercising both branches; the determinism gate enforces stability.

`ImportEdge.SymbolIDs` (symbol-level import provenance) remains unmapped — no analogue exists in the proto envelope today and needs a proto contract turn before any producer work.

**Refs:** `internal/graph/normalize.go` (current `lines` / `internal` emit + placeholder `is_test` / `test_only`), `internal/graph/loader_packages.go` (`Tests: false` switch site), `scenarios/architecture-cartographer/api/internal/graph/gocodegraph/client.go` (`protoToRawGraph` now reads all four new attributes).

### 2026-05-23 — Per-path mutex memory leak — closed 2026-05-24

**Status:** Closed. `PathMutex` is now bounded by an LRU with refcount-based eviction (capacity 10,000 by default). Held entries are never evicted; idle entries are dropped LRU-first when over capacity. See `internal/graph/mutex.go` and the `TestPathMutexLRU*` tests.

### 2026-05-23 — In-memory `PlanStore` + Operation Log gap — closed 2026-05-24

**Status:** Closed (REQ-P1-002 landed). `internal/rewrite/store_sqlite.go` is now the production `PlanStore`, wired in `api/main.go` against the scenario's existing SQLite handle. The same struct satisfies `OperationLog` so non-dry-run `Apply` calls record one row per operation in `rewrite_operation_log`. Schema lives in `internal/rewrite/schema.sql` and registers through `handlers/rewrite/endpoints.go::Schema` → `modules.AllSchemas`. `MemoryStore` is retained for in-process tests that don't need a real DB.

### 2026-05-23 — Fixture Validator UI not implemented

**Symptom:** REQ-P1-001 calls for a Fixture Validator UI surface that lets an operator pick a fixture under `bas/fixtures/`, run `Extract`, and visually diff against `expected-graph.json`. The byte-stable determinism gate runs in tests today, but there is no human-facing diff surface.

**Root cause:** UI work is out of scope for the v1 implementation plan (plan §4 Out of Scope). Only API and CLI shipped.

**Workaround:** `UPDATE_FIXTURES=1 go test ./internal/graph/...` regenerates `expected-graph.json` files; the operator reviews the git diff manually.

**Real fix:** REQ-P1-001 — separate UI plan. Wire a `ui/src/features/fixture-validator/` feature on top of the existing `Extract` Connect client, rendering nodes/edges side-by-side with a JSON diff component.

**Owner:** UI plan (not yet authored).

**Refs:** `requirements/01-deterministic-graph-extraction/module.json` (REQ-P1-001), `bas/fixtures/`.

### 2026-05-23 — `include_vendor` flag is wired through but not honored — closed 2026-05-24

**Status:** Closed (REQ-P1-003 landed). `Service.Extract` now post-filters vendored packages (directory-based: any package whose source directory contains a `vendor/` segment) when `LoadOptions.IncludeVendor=false`. See `internal/graph/service.go::filterVendorPackages` and the synthetic unit test in `internal/graph/vendor_filter_test.go`. A real-Go-module `go-vendored` determinism fixture is left as follow-up; the unit test exercises the filter directly against synthetic `*packages.Package` values.

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
