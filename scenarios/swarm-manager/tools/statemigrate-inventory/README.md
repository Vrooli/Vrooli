# statemigrate-inventory (TEMPORARY)

Read-only inventory of all swarm-manager persisted state, built to make the
Phase 8 declarative-operations **state migration** measurable and reversible.

> **This tool is temporary.** It exists only to support the migration. It is
> scheduled for **deletion in Phase 9** of the "Declarative agent operations for
> backlog items and initiatives" plan, per the `storage-steer` one-shot
> migration policy (migration scaffolding does not become permanent surface).
> Do not build durable features on it.

## What it does

Walks the swarm-manager storage roots (`data`, `state`, `cache` classes, plus the
repo `config/settings.json`), hashes every file, attributes each to an object
class, parses the primary documents, and emits:

- **`inventory-phase1.json`** — a deterministic, byte-stable machine inventory:
  per-class counts, status/kind distributions, per-object stable identities +
  sizes + sha256, plan-ref usage (managed vs unmanaged), run-owner ownership +
  ambiguity, referential-integrity findings (dangling deps/members/targets,
  membership divergence, orphaned runs), corrupt/invalid-state anomalies,
  expected-but-absent state, and a master content hash.
- **`inventory-phase1-summary.md`** — a human-readable rollup of the same data.

Outputs live in `../../docs/operations/migration/inventory/`.

### Determinism / byte-stability

Two back-to-back runs over unchanged state produce **byte-identical** output.
There are no timestamps in the payload; every slice is sorted; maps encode with
sorted keys; absolute roots are home-redacted. The `totals.content_hash` (sha256
over every file's content) is the pre/post-migration reconciliation anchor — see
`../../docs/operations/migration/RUNBOOK.md`.

### Never silently skips

Unreadable files, corrupt JSON, and invalid enum values are emitted as explicit
`anomalies` records — never dropped. Files matching no known storage pattern are
emitted as `unclassified_artifact` findings so unknown state surfaces loudly.

## Design constraints

- **Standalone, stdlib-only.** Its own Go module, zero external dependencies, not
  part of the `swarm-manager` api module. It reads raw JSON off disk rather than
  importing `internal/…` packages, so it neither disturbs nor is coupled to the
  api build. (Go also forbids importing another module's `internal/` packages.)
- **Read-only.** It only stats, reads, and hashes. It never opens the live
  `events.db` (treated as an opaque hashed artifact) and never writes under any
  runtime root.

## Usage

```sh
# Default: resolve live roots (~/.vrooli/{data,state,cache}/vrooli/swarm-manager)
SCENARIO_ROOT=$(pwd)/../.. go run . --out-dir ../../docs/operations/migration/inventory

# Explicit roots (e.g. a restored backup or a *_shadow namespace)
go run . --data-root <dir> --state-root <dir> --cache-root <dir> --out-dir <dir>

# JSON to stdout (no files)
go run .
```

Roots resolve like `internal/runtimepaths`: `VROOLI_STORAGE_ROOT` (or `~/.vrooli`)
→ `<root>/<class>/vrooli/swarm-manager`. A sibling `*_shadow` namespace, if
present, is reported under `roots.shadow_namespaces_present`.

## Tests

`go test ./...` builds a synthetic fixture tree containing malformed JSON,
dangling references, membership divergence, ambiguous ownership, unmanaged
plan-refs, an orphaned run, and an unclassified artifact, and asserts each is
**reported** (not dropped) and that two runs are byte-identical.
