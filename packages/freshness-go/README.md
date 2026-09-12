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

`Compute` is the frozen v1 compatibility adapter: sha256 over the sorted list
of `relpath \x00 sha256(file bytes) \x0a` for every git-tracked or
untracked-not-ignored file under one scenario directory, excluding generated
state. Existing callers keep byte-identical `td:` identities.

New validation lifecycles use `BuildInputManifest`. Its `ci:v1:` identity
composes a primary root, explicitly declared shared dependency roots,
configuration, and toolchain scalars. Absolute checkout paths, commit, branch,
dirty/index state, and mtimes are attribution only. A required input that is
missing, unreadable, a symlink, case-colliding, or modified during capture
returns an error instead of publishing incomplete evidence.

```go
digest, err := treedigest.Compute(scenarioDir)        // "td:<hex>"
gitCtx := treedigest.CollectGitContext(scenarioDir)    // best-effort sha/branch/dirty

request := treedigest.ManifestRequest{
    Primary: treedigest.RootSpec{Name: "scenario", Path: scenarioDir,
        Selections: []treedigest.InputSelection{{Glob: "**", Required: true}}},
    Dependencies: []treedigest.RootSpec{{Name: "shared-proto", Path: protoDir,
        Selections: []treedigest.InputSelection{{Glob: "gen/go/example/**", Required: true}}},
    Configuration: map[string]string{"preset": "comprehensive"},
    Toolchain: map[string]string{"go": "1.25"},
    Attribution: treedigest.ManifestAttribution{Commit: sha, Branch: branch, Dirty: dirty},
}
manifest, err := treedigest.BuildInputManifest(request)
// manifest.Identity is independent of Attribution and checkout location.

// Optional warm builder: it still enumerates and reads every selected file.
// Only SHA-256 results for exact bytes already read are reused.
builder := treedigest.NewManifestBuilder(8 << 20)
manifest, err = builder.Build(request)
```

`ManifestBuilder` deliberately does not cache enumeration, metadata verdicts,
or completed manifests. Newly created, removed, renamed, unreadable, and
mid-read-mutated inputs therefore remain authoritative on every capture.

Git's cached-file enumeration includes paths intentionally deleted from the
working tree. Those already-absent paths contribute absence—not an I/O
failure—to the current identity; an explicit required selection that has no
remaining eligible file still fails closed. A selected file observed before
reading must remain a regular, non-symlink file through the stable read.
`node_modules` is generated dependency state and is excluded at every package
depth, including Git-tracked workspace links such as `ui/node_modules/...`.

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
