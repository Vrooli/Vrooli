# SDA Unified Graph Ingest — 2026-06-22

## Scope
Phase 0 measurement gate for the graph interface-engine re-sourcing + freshness-gated
ingest sweeper. Measures the cost of building the unified, evidence-tagged
`graph_edges` store so the sweeper's cadence/concurrency/budget knobs are sized
from data, not guesses.

## Method
- Live fleet (≈100 scenarios) with `proto-health` and `code-facts` reachable.
- Ingest path instrumented with `common.v1.ExecutionMetrics` per stage
  (`analyze`, `interfacegraph` with proto/import/assemble sub-gauges, `merge`,
  `persist`) via `internal/graphingest`.
- Two probes:
  1. Full-fleet `POST /api/v1/graph/rebuild?apply=true` (the manual override:
     `AnalyzeAllScenarios` over every scenario dir + one fleet-wide interface
     graph build + merge + atomic replace).
  2. Single-scenario `POST /api/v1/graph/rebuild?scenario=agent-manager`
     (the sweeper's per-scenario unit: scoped interface-graph build + merge).

## Results
| Probe | Cold | Warm |
|---|---|---|
| Full-fleet rebuild (analyze-all + fleet interfacegraph + replace) | > 5 min | — |
| Per-scenario ingest (scoped interfacegraph + merge) | — | ~0.17 s |

Warm full-fleet rebuild stage breakdown (`build_stats`, 111 scenarios):
`ProtoFetchMs=122`, **`ImportFetchMs≈17,900`** (code-facts fleet imports dominate),
`AssembleMs=20`, 120 nodes / 98 interface edges. `AnalyzeAllScenarios` over the
fleet is the other major cost. Cold is far worse (cold code-facts).

Populated store after a full fleet rebuild: **431 edges** —
`resource=284`, scenario→scenario `147` (`proto_import` + `go_import` + `declared`
+ `vrooli_cli`). The `proto_import`/`go_import` edges are exactly what the legacy
analyze-only centrality store missed fleet-wide. `code-facts` degraded gracefully
on two scenarios (`vrooli-autoheal`, `vrooli-events`: `multiple_go_mod_files`) —
the build still completed and `degraded_sources` reported it.

Centrality re-sourced from the unified store discriminates with real variance,
e.g. `agent-manager`: direct reverse-deps 4, transitive 6, required-weighted 14,
core-seed distance 0, dependents `{app-issue-tracker,
development-toolchain-validator, swarm-manager, knowledge-observatory, …}`.

## Sizing decision (gate output)
The **monolithic full-fleet rebuild is expensive** (> 5 min cold), dominated by
`AnalyzeAllScenarios` (fleet-wide filesystem walk) and the cold fleet-wide
interface-graph build. It is therefore **not** the background mechanism — it is a
manual operator override only.

The **background sweeper uses the per-scenario scoped path** (~0.17 s warm each),
freshness-gated by tree digest so most cycles re-ingest only the handful of
scenarios that changed. Derived conservative defaults
(`SDA_GRAPH_SWEEP_*`):

| Knob | Default | Rationale |
|---|---|---|
| `INTERVAL` | 30m | Far above per-scenario cost; freshness gate makes most cycles near-no-ops. |
| `CONCURRENCY` | 3 | 3 × ~0.2 s warm keeps upstream load light; cold first sweep stays bounded. |
| `CYCLE_BUDGET` | 10m | Hard wall-clock cap; even a cold first sweep of the full fleet fits with margin and degrades gracefully if not. |
| `START_JITTER` | 90s | Avoid a thundering-herd hit on proto-health/code-facts at boot. |
| `BREAKER_THRESHOLD` / `BREAKER_COOLDOWN` | 4 / 5m | Trip after sustained upstream failure; retain last-good edges meanwhile. |

These are the values wired in `internal/graphsweeper/config.go` `Defaults()`.
