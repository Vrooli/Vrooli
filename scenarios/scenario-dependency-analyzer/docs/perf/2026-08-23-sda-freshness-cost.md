# SDA Freshness Cost Forensics — 2026-08-23

## Status

Measurement record. Read-only investigation; no repository files were changed to
produce it. This document is the **authoritative evidence source** for the
implementation plan that follows from it. A rendered HTML version of the same
material sits beside this file at
`scenarios/scenario-dependency-analyzer/docs/perf/2026-08-23-sda-freshness-cost.html`
(published copy: `https://claude.ai/code/artifact/6c0fd1a7-b430-45e5-84e2-e8729b4d5bcd`).
Read the markdown; the HTML is the same content with a stylesheet.

## Scope

Why `scenario-dependency-analyzer freshness --touched` costs 71.9 seconds on
this host, what else is wrong with the surrounding commands, and what the
remediation is worth. The command matters because `internal/app/hygiene`
executes it as a provider on every `make hygiene`, which is the repository
precommit path.

## Host and tree

| Fact | Value |
|---|---|
| Date | 2026-08-23 |
| Branch | `agi` |
| Cores / RAM | 32 / 60 GB |
| `go env GOPROXY` | `https://proxy.golang.org,direct` |
| `GOPRIVATE`, `GONOSUMDB` | unset |
| CPU `sha_ni` flag | present |
| In-repo Go modules | 315 total |
| Under `scenarios/` | 247 |
| Under `packages/` | 23 |
| Under `resources/` | 31 |
| Under `templates/` | 12 |
| Root + `cmd/` | 2 |
| `.go` files (scenarios+packages+resources+cmd) | 20,972 (158.9 MB) |
| Working-tree change driving `--touched` | root `go.mod`, `go.sum` |

## Headline

`freshness --touched` spends **68.5 of its 71.9 seconds on failed network
lookups for a module that has no upstream**. Setting `GOPROXY=off` reduces the
command to **3.4 seconds and produces verdict-for-verdict identical output**.

## The measurement table

Every row was run on this host on 2026-08-23. "Verdicts" is the tool's own
`--json` summary as `clean / stale / error`.

| Probe | Wall | User+Sys | Verdicts |
|---|---|---|---|
| `freshness --touched --json` | 71.9 s | 21.2 + 11.7 s | 44 / 4 / 199 |
| Same, immediately again (warm) | 69.9 s | 20.9 + 10.8 s | 44 / 4 / 199 |
| Same, `--concurrency 32` | 70.8 s | 20.0 + 10.8 s | 44 / 4 / 199 |
| Same, with `GOPROXY=off` in env | **3.4 s** | 12.1 + 9.8 s | **44 / 4 / 199** |
| 247 bare `go mod tidy -diff`, offline, `xargs -P8` | 1.5 s | 11.0 + 8.7 s | 44 / 4 / 199 |
| Same, offline, `xargs -P32` | 1.2 s | 10.6 + 8.3 s | 44 / 4 / 199 |
| Same, network on, `xargs -P8` | 71.0 s | 18.9 + 11.5 s | 44 / 4 / 199 |
| `GOPRIVATE=github.com/vrooli/*` (proxy bypassed, `direct` still on) | 60.9 s | 18.3 + 10.8 s | 44 / 4 / 199 |
| `deps reconcile --all --json` | 30.8 s | 0.04 + 0.03 s | aborts, no output |
| Root module alone: `go mod tidy -diff` | 2.4 s | 9.5 + 10.3 s | stale |
| SHA-256 over all 20,972 `.go` files, `-P8` | 0.050 s | 0.14 + 0.19 s | — |
| Same, `-P32` | 0.051 s | 0.14 + 0.21 s | — |
| Repo walk, pruned dirs, warm | 0.245 s | 0.35 + 0.81 s | 312,508 files |

### What each row proves

- **Warm rerun costs the same as cold.** There is no result cache anywhere.
- **Concurrency 8 vs 32 is a 1.1 s difference.** The worker pool is not the
  constraint. The Go module cache serializes the failing lookups. Arithmetic
  confirms it: 198 lookups x ~0.35 s ~= 69 s ~= the whole run.
- **`GOPROXY=off` gives identical verdicts.** The 68.5 s of network contributes
  zero information.
- **`GOPRIVATE` alone is not the fix** (60.9 s). Bypassing the proxy still
  leaves a `git ls-remote` to GitHub per surface.
- **The bare-subprocess row reproduces the cost outside SDA.** This is Go
  toolchain behavior, not an SDA-specific bug — but SDA is what triggers it 247
  times per commit.
- **Cache-key computation is free.** Hashing the entire Go corpus is ~50 ms.

## Root cause

`cli/domains/health/freshness.go`, function `runGo`:

```go
cmd.Env = envkit.WithOverlay(envkit.Env(os.Environ()), envkit.SameScenario,
    envkit.Env{"GOWORK=off", "GOFLAGS="})
```

`GOPROXY` is not overlaid, so it passes through from the ambient environment.
198 of the 247 scenario surfaces lack
`replace github.com/vrooli/envkit-go => ../../../packages/envkit-go`, reach
`envkit-go` transitively through `cli-core` or `api-core`, and therefore attempt
to download a module that exists only in-repo.

Error text on 198 of the 199 error surfaces begins:

```
go: downloading github.com/vrooli/envkit-go v0.0.0
```

## Findings

### P1 — Failed network lookups dominate the runtime

68.5 s of 71.9 s. See root cause above. Fix: overlay `GOPROXY=off` plus a typed
retry tier (see "Confirmed retry signature").

### P2 — The worker pool is inert while P1 stands

Concurrency 8 and 32 produce the same wall clock. The run uses 31 s of CPU
against 70 s of wall on 32 cores — roughly 4.6% utilization. Concurrency only
becomes a meaningful knob after P1 is fixed (1.5 s at 8-way, 1.2 s at 32-way).

### P3 — No result cache exists

`grep -rn "cache\|Cache" cli/ --include=*.go | grep -v _test` returns three
hits, all of which are directory names in walk-skip lists
(`freshness.go:237`, `freshness.go:280`) plus one unrelated `--cached` git flag
(`freshness.go:608`). Every invocation re-walks the repository, re-parses 315
`go.mod` files, and re-spawns 247 subprocesses.

`go mod tidy -diff` is a pure function of a small hashable input set, so this is
cacheable exactly rather than heuristically.

### P4 — One defect produces 198 of the 199 error rows

| Fact | Count |
|---|---|
| Scenario surfaces declaring the `envkit-go` replace | 39 |
| Scenario surfaces missing it | 208 |
| Error surfaces whose first line is the `envkit-go` download | 198 |

`packages/envkit-go/go.mod` exists and declares `module github.com/vrooli/envkit-go`.
The freshness gate is currently reporting one defect 198 times. Surfaces in the
`error` state are never assessed for actual staleness — the check stops first.

This is why speed and accuracy are one piece of work: removing the noise also
removes the cost.

### P5 — Double repo walk and unmemoized closures

- `discoverGoSurfaces(root)` walks `scenarios/`.
- `discoverGoModules(root)` then walks the whole repository root, which contains
  `scenarios/`. Every scenario `go.mod` is discovered and hand-parsed twice.
- `impactedSurfaces` calls `transitiveRequirements(surface.module, modules)`
  once per surface with no memoization, re-deriving identical closures from the
  same 315-module map.
- That call happens even when `changedModules` is empty, where the result cannot
  affect the outcome.

Measured cost of SDA's own overhead (total 3.4 s minus the 1.5 s subprocess
floor) is ~1.9 s. A single warm walk is ~0.245 s.

### P6 — Coverage gap in both directions

**Never checked (68 modules):** surfaces are discovered under `scenarios/` only.
That excludes `packages/` (23, including `envkit-go`, `api-core`, `cli-core`),
`resources/` (31), `templates/` (12), the root module, and `cmd/`. A root
`go.mod` edit is precisely what triggers the fleet-wide fan-out, and the root
module is the one module the check will not look at.

**Checked but should not be (5 modules):** deliberately synthetic fixtures.

```
scenarios/go-code-graph/bas/fixtures/go-cycles/go.mod
scenarios/go-code-graph/bas/fixtures/go-usage-facts/go.mod
scenarios/go-code-graph/bas/fixtures/go-tests/go.mod
scenarios/go-code-graph/bas/fixtures/go-mislocated/go.mod
scenarios/browser-automation-studio/bas/seeds/go.mod
```

Exclusion is currently an inline `switch d.Name()` list duplicated at
`freshness.go:237` and `freshness.go:280`.

### P7 — `deps reconcile --all` is serial, fail-fast, and re-walks per scenario

Observed: 30.8 s wall, 0.07 s of CLI CPU (all time is server-side), then abort
with no output for any scenario.

```
reconcile go.mod replaces: internal: exit status 1: go: -require={{SCENARIO_ID}}@v0.0.0:
  invalid path: malformed import path "{{SCENARIO_ID}}": invalid char '{'
```

Four distinct defects:

1. **CLI fan-out is a serial loop with fail-fast.**
   `cli/domains/depsapproved/reconcile.go` issues one `PreviewFix`/`ApplyFix`
   RPC per scenario and does `return cliapp.WrapAPIError(...)` on the first
   error.
2. **Server aborts the whole request on one bad surface.**
   `api/internal/dependencyhealth/gomod_fix.go` returns
   `connect.NewError(connect.CodeInternal, cerr)` inside the per-surface loop.
3. **Topology is rebuilt per RPC.** `gomodreconcile.LoadTopology(repoRoot)` walks
   all 315 modules, and it runs on every one of the ~121 requests.
4. **Linear scan per import.** `matchingInRepoModule` scans all 315 module paths
   for every import of every `.go` file, via `importedInRepoModules`, which
   `parser.ParseFile(..., parser.ImportsOnly)` walks across the scenario. Order
   of 10^8 prefix comparisons fleet-wide where a longest-prefix map is one
   lookup.

### P8 — Two hand-rolled `go.mod` parsers with opposite trade-offs

- `cli/domains/health/freshness.go:parseGoModFile` is a line scanner. It ignores
  `replace` entirely and does `strings.Fields(rest)[0]` with no length check
  (panics on a bare `module ` line).
- `api/internal/gomodreconcile/reconcile.go:parseGoMod` shells out to
  `go mod edit -json` per file, with a package doc that explicitly rejects
  `golang.org/x/mod` on the grounds that "the go binary is authoritative".

`golang.org/x/mod/modfile` is the parser the go binary itself uses. It is
present in the module cache transitively (`golang.org/x/mod@v0.10.0` through
`v0.37.0`) but is **not** a direct require of any in-repo `go.mod` and is **not**
in `.vrooli/dependencies/approved-dependencies.json`. Adoption must go through
`scenario-dependency-analyzer deps` per `docs/package-governance.md`.

### P9 — `--touched` cannot see new files; the run is unbounded

- `changedPaths` runs `git diff --name-only HEAD` and
  `git diff --cached --name-only`, with a `git status --porcelain=v1
  --untracked-files=no` fallback. A new, unstaged `go.mod` — exactly what a new
  scenario adds — is invisible.
- `runGo` caps each subprocess at 120 s. Nothing caps the run, so the worst case
  is 247 x 120 s with no ceiling and no partial result.

## What is NOT the problem

### The fan-out is honest

`impactedSurfaces` short-circuits: when root `go.mod`/`go.sum` changes it marks
every surface impacted without consulting the graph. That shortcut is real, and
it is very nearly correct.

Computing the true transitive require closure for all 247 scenario surfaces:

| Result | Count |
|---|---|
| Transitively require `github.com/vrooli/vrooli` | **238** |
| Do not | 9 |

Of the 9, four are the `bas/fixtures` modules from P6 and one is
`browser-automation-studio/bas/seeds`. Almost all of the 238 reach the root
module through `packages/cli-core`, which requires it directly.

Note the distinction that makes this counter-intuitive: 236 scenario `go.mod`
files carry `replace github.com/vrooli/vrooli => ../../..`, but a `replace`
directive creates no dependency edge. Only 27 `require` it directly. The
transitive closure through `cli-core` is what reaches 238.

**Consequence for planning:** making impact analysis exact removes 9 surfaces
(3.6%). The lever is not "compute impact more precisely so less work happens" —
it is "make each unit of work cost 6 ms instead of 350 ms, then cache it." An
earlier portability analysis proposed reporting impact and deferring
verification on the theory that the consumer list would be short. The list is
cheap to compute and it is not short.

### It is not a fork storm and not a memory problem

247 subprocesses are bounded by the worker pool. Peak memory never approached
the host's 60 GB. The ~9 s of `sys` time is ordinary spawn cost for 247 `go`
invocations, and the offline sweep completes in 1.5 s.

## Confirmed retry signature

Verified against a scratch module requiring a nonexistent version:

```
$ GOWORK=off GOFLAGS= GOPROXY=off go mod tidy -diff
go: downloading github.com/nats-io/nats.go v1.99.99
go: probe imports
	github.com/nats-io/nats.go: module lookup disabled by GOPROXY=off
```

The string `module lookup disabled by GOPROXY=off` is the stable, unambiguous
marker for "this surface needs the network for a legitimate reason". Surfaces
emitting it are retried once with the network. Surfaces failing for any other
reason — including all 198 `envkit-go` cases — never touch the network.

## Cache design

`go mod tidy -diff` depends on more than `go.mod`: it resolves the module's
actual import graph, so a source file that adds or removes an import changes the
answer with no manifest edit. An mtime-only or manifest-only key goes stale in
exactly the case the check exists to catch.

Key:

```
sha256(
  go.mod bytes
  go.sum bytes
  sorted import set of every .go file under the module   <- the non-obvious one
  go toolchain version   (go env GOVERSION)
  tool schema version    (bump to invalidate all entries on a logic change)
)
```

Constraints:

- Store the full `freshnessSurfaceReport` (status, diff paths, error text) so a
  cached stale verdict is as actionable as a fresh one.
- Write under the scenario's own data directory, never `/tmp`. Scratch
  directories on this host do not survive days.
- Bound it. The scenario already carries an unbounded `phase-cache/` at 56 MB /
  362 files with no owner; a second unbounded cache repeats that.
- Never cache a `needs_download` result. That status describes the host's module
  cache, not the surface, and it changes with no file change.

Cost is not a concern: hashing all 20,972 `.go` files takes ~50 ms (SHA-NI).

## Projected end state

Warm page cache, 32 cores, `GOPROXY=off`.

| Case | Now | Projected | Confidence |
|---|---|---|---|
| Commit touching one scenario | 71.9 s | ~0.35 s | high |
| Root `go.mod` touched, cache warm (today's state on `agi`) | 71.9 s | ~0.4 s | high |
| Cold cache, all 315 modules with P6 coverage | n/a | ~3 s | medium |
| `deps reconcile --all` | 30.8 s, aborts | 1-3 s, completes | low-medium |
| `make hygiene` end to end | 2 m 05 s | ~40 s | medium |

Component budget:

| Component | Today | After |
|---|---|---|
| Repo walk + manifest parse | ~0.5 s (two walks, 630 parses) | ~0.25 s (one walk) |
| Impact closure | 247 unmemoized walks | ~0.01 s memoized |
| Cache keys, 315 modules / 21k files | n/a | ~0.05-0.10 s |
| Subprocess sweep | 1.5 s, all 247 | misses only |
| Network | **68.5 s** | 0 |

### Caveats on the projection

1. **The root module is a 2.4 s floor.** Measured directly. Once it enters
   coverage, any run that genuinely invalidates it pays that, and concurrency
   cannot help because it is one item. This is what sets the ~3 s cold figure,
   and it is why the cache is required rather than optional.
2. **The reconcile projection is inference, not measurement.** Its symptom is
   measured; its internals were not profiled. Re-measure after the fix rather
   than assuming.
3. **Cold CI is a different question.** All figures assume a warm `GOMODCACHE`.
   On a fresh runner the `needs_download` tier fires for real and the run pays
   actual downloads once.

## Adjacent control-plane facts

These are outside SDA but bound the same lane.

- `Makefile:8` defines `VROOLI := go run ./cmd/vrooli --no-stale-check`, so
  `make hygiene` recompiles the CLI every invocation (~20 s). The `setup` target
  already has a `command -v vrooli` fast path; `hygiene` does not.
- `internal/app/hygiene/providers.go:Registry.Run` iterates providers serially
  with no time budget. Only `sdaFreshnessProvider` is registered today.
- `internal/app/hygiene/sda_freshness_provider.go:79` execs
  `scenario-dependency-analyzer freshness --touched --json --repo-root <root>`
  and degrades gracefully when the scenario is unreachable.

## Reproduction commands

```bash
# baseline
time scenario-dependency-analyzer freshness --touched --json | jq -c .summary

# the isolation that identifies the cause
time GOPROXY=off scenario-dependency-analyzer freshness --touched --json | jq -c .summary

# error-class census
scenario-dependency-analyzer freshness --touched --json \
  | jq -r '.surfaces[]|select(.status=="error")|.error' \
  | grep -oE 'go: downloading [^ ]+ v[0-9.]+' | sort | uniq -c | sort -rn

# true transitive impact of a root go.mod edit (should print 238)
# parse every go.mod, walk the require graph, count surfaces reaching
# github.com/vrooli/vrooli
```
