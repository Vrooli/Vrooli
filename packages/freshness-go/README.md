# freshness-go

Shared single source of truth for **run freshness**: did a scenario's tests
pass against exactly its current byte-state? Extracted from test-genie by the
scenario-status-layer plan so test-genie's `RunsService.CheckFreshness` RPC
and cached status readers (scenario-completeness-scoring) share one digest
spec and one verdict semantics.

This is a **pure-logic, dependency-free library**: no service calls, stdlib
only. The only filesystem touches are reading the scenario tree (digest) and
the run index file (read-only).

## Packages

### `github.com/vrooli/freshness-go/treedigest`

Deterministic content digest of a scenario's working tree — the freshness
identity for test runs.

**The spec is FROZEN** (requirements-traceability plan, §8 Contract
Decisions): sha256 over the sorted list of `relpath \x00 sha256(file bytes)
\x0a` for every git-tracked or untracked-not-ignored file under the scenario
directory, excluding generated/state directories (`coverage/`, `data/`,
`dist/`, `node_modules/`, …), `"td:"`-prefixed. Any refactor must produce
byte-identical digests for identical trees. Documented v1 limitation: scoped
to the scenario directory only — shared `packages/*` edits do not change a
scenario's digest.

```go
digest, err := treedigest.Compute(scenarioDir)        // "td:<hex>"
gitCtx := treedigest.CollectGitContext(scenarioDir)    // best-effort sha/branch/dirty
```

### `github.com/vrooli/freshness-go/runindex`

The record types stored in test-genie's append-only run index
(`coverage/runs.index.json`) and a read-only, lock-free loader (safe because
the write side replaces the file atomically). test-genie keeps the
write/locking side internal and aliases these types.

```go
records, err := runindex.Load(scenarioDir) // newest-first; missing index => nil, nil
```

### `github.com/vrooli/freshness-go` (package `freshness`)

The pure verdict core: per-phase `fresh` / `stale` / `unknown` against the
current digest, plus the copy-pastable remediation command.

```go
report := freshness.Check(records, digest, freshness.RequiredPhases())
cmd := freshness.SuggestedCommand(scenario, report.Phases, true)
```

Semantics (reused verbatim from test-genie — do not fork):

- A phase is **fresh** iff some run stamped with the current digest passed it.
- Verdicts are **unknown** when no digest-stamped runs exist at all
  (pre-digest history can never prove staleness).
- `RequiredPhases()` mirrors test-genie's quick preset and is deliberately a
  code-level SSOT, NOT per-scenario configurable (operator anti-gaming
  decision). A guard test in test-genie pins the two lists equal.

## Consumers

- `scenarios/test-genie/api` — digest stamping at run start
  (`suite_execution.go`), the CheckFreshness RPC, run-index types.
- `scenarios/scenario-completeness-scoring` — staleness labels on cached
  score output (planned by the scenario-status-layer plan).

## Adoption

Governed Go module, `go_module_replace` adoption:

```
require github.com/vrooli/freshness-go v0.0.0
replace github.com/vrooli/freshness-go => ../../../packages/freshness-go
```
