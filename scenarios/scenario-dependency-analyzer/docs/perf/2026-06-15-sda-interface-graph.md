# SDA `DescribeInterfaceGraph` — fleet interface graph performance

**Date:** 2026-06-15
**Scope:** `scenario-dependency-analyzer` `graph actual` / `DescribeInterfaceGraph` over the full fleet (116 scenario nodes).
**Plan:** `fleet-interface-graph-performance-ttd-sda-wiring-proto-drift-cleanup` (Phase 2 / §5 audit).

## Summary

| Metric | Before | After |
|---|---|---|
| `graph actual` (full fleet), **cold** | **never completed in 3 min** (`deadline_exceeded` against 30s client timeout) | **17.1s** |
| `graph actual` (full fleet), **warm** (cache hit) | n/a (never completed) | **0.21s** |
| Nodes / edges returned | partial / none (timed out) | **116 nodes / 78 edges** |
| Evidence sources on edges | proto only (when it returned at all) | **`proto_import` + `go_import`** |

The graph now returns the complete fleet within deadline cold, and is sub-second warm.

## Method

Host: development machine, 2026-06-15. Running stack: `code-facts`, `proto-health`, `scenario-dependency-analyzer`, `tech-tree-designer`.

```
# cold (first call after restart / cache miss)
time scenario-dependency-analyzer graph actual --json   # real 0m17.114s  -> 116 nodes / 78 edges
# warm (immediate re-run, persistent SQLite cache hit)
time scenario-dependency-analyzer graph actual --json   # real 0m0.210s   -> 116 nodes / 78 edges
```

Edge-source check (both evidence kinds present):

```
jq -r '[.graph.edges[].evidence[]?.source] | unique' /tmp/sda_actual.json
# EVIDENCE_SOURCE_GO_IMPORT, EVIDENCE_SOURCE_PROTO_IMPORT
```

## Root causes fixed (per plan Phase 2)

1. **Sequential upstream calls** → the two upstream sources (proto-health `DescribeScenariosProtos`, code-facts `DescribeFleetImports`) now run in parallel (`interfacegraph/builder.go` WaitGroup).
2. **In-memory cache (warm slower than cold)** → persistent precompute cache wired (`store/interface_graph_cache.go` + `interface_graph_cache` table); warm calls are real hits.
3. **Per-RPC deadline** → graph-appropriate per-command timeout (90s build / 5-min cache TTL) via the `cliapp/connect.go` seam, without raising the global 30s default.
4. **Request bounding** → `max_scenario_hops` (proto field) + neighborhood BFS for scoped queries.

## Tradeoffs / levers

- Cache TTL (5 min), per-RPC build deadline (90s), and `max_scenario_hops` are tunable levers with sane defaults.
- Warm correctness depends on the upstream code-facts cache fingerprint (mtime+size signature with content-hash fallback) — see the code-facts perf note.
